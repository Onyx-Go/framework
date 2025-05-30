package queue

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DefaultWorker implements the Worker interface
type DefaultWorker struct {
	id          string
	queue       Queue
	options     WorkerOptions
	running     bool
	stopCh      chan struct{}
	doneCh      chan struct{}
	stats       *WorkerStats
	mutex       sync.RWMutex
	middleware  []Middleware
}

// NewWorker creates a new worker
func NewWorker(id string) *DefaultWorker {
	return &DefaultWorker{
		id:     id,
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
		stats: &WorkerStats{
			ID:        id,
			StartedAt: time.Now(),
		},
	}
}

// Start begins processing jobs
func (w *DefaultWorker) Start(ctx context.Context, queue Queue, options WorkerOptions) error {
	w.mutex.Lock()
	if w.running {
		w.mutex.Unlock()
		return fmt.Errorf("worker %s is already running", w.id)
	}

	w.queue = queue
	w.options = options
	w.running = true
	w.stats.Queue = options.Queue
	w.stats.IsRunning = true
	w.middleware = options.Middleware
	w.mutex.Unlock()

	go w.runWorker(ctx)
	return nil
}

// Stop gracefully stops the worker
func (w *DefaultWorker) Stop(ctx context.Context) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if !w.running {
		return nil
	}

	w.running = false
	close(w.stopCh)

	// Wait for worker to finish current job
	select {
	case <-w.doneCh:
		w.stats.IsRunning = false
		return nil
	case <-ctx.Done():
		w.stats.IsRunning = false
		return ctx.Err()
	}
}

// IsRunning returns true if the worker is active
func (w *DefaultWorker) IsRunning() bool {
	w.mutex.RLock()
	defer w.mutex.RUnlock()
	return w.running
}

// Stats returns worker statistics
func (w *DefaultWorker) Stats() *WorkerStats {
	w.mutex.RLock()
	defer w.mutex.RUnlock()
	
	// Return a copy
	statsCopy := *w.stats
	return &statsCopy
}

// runWorker is the main worker loop
func (w *DefaultWorker) runWorker(ctx context.Context) {
	defer close(w.doneCh)

	ticker := time.NewTicker(w.options.Sleep)
	defer ticker.Stop()

	processedJobs := 0
	startTime := time.Now()

	for {
		select {
		case <-w.stopCh:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			if w.shouldStop(processedJobs, startTime) {
				w.Stop(context.Background())
				return
			}

			if w.processNextJob(ctx) {
				processedJobs++
			}
		}
	}
}

// processNextJob processes the next available job
func (w *DefaultWorker) processNextJob(ctx context.Context) bool {
	job, err := w.queue.Pop(ctx, w.options.Queue)
	if err != nil {
		if w.options.StopWhenEmpty {
			w.Stop(context.Background())
		}
		return false
	}

	return w.processJob(ctx, job)
}

// processJob processes a single job
func (w *DefaultWorker) processJob(ctx context.Context, job Job) bool {
	startTime := time.Now()
	
	// Update current job in stats
	w.mutex.Lock()
	if queueJob, ok := job.(*QueueJob); ok {
		w.stats.CurrentJob = queueJob.ToPayload()
	}
	w.mutex.Unlock()

	defer func() {
		duration := time.Since(startTime)
		
		w.mutex.Lock()
		w.stats.CurrentJob = nil
		w.stats.TotalTime += duration
		
		// Update average time
		if w.stats.ProcessedJobs+w.stats.FailedJobs > 0 {
			totalJobs := w.stats.ProcessedJobs + w.stats.FailedJobs + 1
			w.stats.AverageTime = time.Duration(int64(w.stats.TotalTime) / int64(totalJobs))
		}
		
		now := time.Now()
		w.stats.LastJobAt = &now
		w.mutex.Unlock()

		// Update queue stats if it's a memory queue
		if memQueue, ok := w.queue.(*MemoryQueue); ok {
			memQueue.MarkJobCompleted(job.GetQueue(), duration)
		}
	}()

	// Handle panics
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("job panicked: %v", r)
			w.handleJobFailure(ctx, job, err)
			
			if w.options.Logger != nil {
				w.options.Logger("Worker %s: Job panicked: %v", w.id, r)
			}
		}
	}()

	// Create context with timeout
	timeout := job.GetTimeout()
	if timeout == 0 {
		timeout = w.options.Timeout
	}

	jobCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Process job through middleware chain
	err := w.executeJobWithMiddleware(jobCtx, job)
	
	if err != nil {
		w.handleJobFailure(ctx, job, err)
		return false
	}

	// Mark job as successful
	w.mutex.Lock()
	w.stats.ProcessedJobs++
	w.mutex.Unlock()

	return true
}

// executeJobWithMiddleware processes the job through the middleware chain
func (w *DefaultWorker) executeJobWithMiddleware(ctx context.Context, job Job) error {
	if len(w.middleware) == 0 {
		return job.Handle(ctx)
	}

	// Build middleware chain
	handler := func(ctx context.Context, job Job) error {
		return job.Handle(ctx)
	}

	// Apply middleware in reverse order
	for i := len(w.middleware) - 1; i >= 0; i-- {
		middleware := w.middleware[i]
		currentHandler := handler
		handler = func(ctx context.Context, job Job) error {
			return middleware.Handle(ctx, job, currentHandler)
		}
	}

	return handler(ctx, job)
}

// handleJobFailure handles a failed job
func (w *DefaultWorker) handleJobFailure(ctx context.Context, job Job, err error) {
	w.mutex.Lock()
	w.stats.FailedJobs++
	w.mutex.Unlock()

	// Update queue stats if it's a memory queue
	if memQueue, ok := w.queue.(*MemoryQueue); ok {
		memQueue.MarkJobFailed(job.GetQueue())
	}

	// Call job's failure handler
	if failErr := job.Failed(ctx, err); failErr != nil && w.options.Logger != nil {
		w.options.Logger("Worker %s: Job failure handler error: %v", w.id, failErr)
	}

	if w.options.Logger != nil {
		w.options.Logger("Worker %s: Job failed: %v", w.id, err)
	}
}

// shouldStop determines if the worker should stop based on options
func (w *DefaultWorker) shouldStop(processedJobs int, startTime time.Time) bool {
	if w.options.MaxJobs > 0 && processedJobs >= w.options.MaxJobs {
		return true
	}

	if w.options.MaxTime > 0 && time.Since(startTime) >= w.options.MaxTime {
		return true
	}

	return false
}

// WorkerPool manages multiple workers
type WorkerPool struct {
	workers map[string]*DefaultWorker
	mutex   sync.RWMutex
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool() *WorkerPool {
	return &WorkerPool{
		workers: make(map[string]*DefaultWorker),
	}
}

// StartWorker starts a new worker
func (wp *WorkerPool) StartWorker(ctx context.Context, id string, queue Queue, options WorkerOptions) error {
	wp.mutex.Lock()
	defer wp.mutex.Unlock()

	if _, exists := wp.workers[id]; exists {
		return fmt.Errorf("worker %s already exists", id)
	}

	worker := NewWorker(id)
	if err := worker.Start(ctx, queue, options); err != nil {
		return err
	}

	wp.workers[id] = worker
	return nil
}

// StopWorker stops a specific worker
func (wp *WorkerPool) StopWorker(ctx context.Context, id string) error {
	wp.mutex.Lock()
	worker, exists := wp.workers[id]
	if !exists {
		wp.mutex.Unlock()
		return fmt.Errorf("worker %s not found", id)
	}
	delete(wp.workers, id)
	wp.mutex.Unlock()

	return worker.Stop(ctx)
}

// StopAll stops all workers
func (wp *WorkerPool) StopAll(ctx context.Context) error {
	wp.mutex.Lock()
	workers := make([]*DefaultWorker, 0, len(wp.workers))
	for _, worker := range wp.workers {
		workers = append(workers, worker)
	}
	wp.workers = make(map[string]*DefaultWorker)
	wp.mutex.Unlock()

	var lastErr error
	for _, worker := range workers {
		if err := worker.Stop(ctx); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// GetWorker returns a specific worker
func (wp *WorkerPool) GetWorker(id string) (*DefaultWorker, bool) {
	wp.mutex.RLock()
	defer wp.mutex.RUnlock()
	worker, exists := wp.workers[id]
	return worker, exists
}

// GetAllWorkers returns all workers
func (wp *WorkerPool) GetAllWorkers() map[string]*DefaultWorker {
	wp.mutex.RLock()
	defer wp.mutex.RUnlock()

	result := make(map[string]*DefaultWorker)
	for id, worker := range wp.workers {
		result[id] = worker
	}
	return result
}

// GetStats returns statistics for all workers
func (wp *WorkerPool) GetStats() []*WorkerStats {
	wp.mutex.RLock()
	defer wp.mutex.RUnlock()

	stats := make([]*WorkerStats, 0, len(wp.workers))
	for _, worker := range wp.workers {
		stats = append(stats, worker.Stats())
	}
	return stats
}

// Common middleware implementations

// LoggingMiddleware logs job execution
type LoggingMiddleware struct {
	logger func(string, ...interface{})
}

// NewLoggingMiddleware creates a new logging middleware
func NewLoggingMiddleware(logger func(string, ...interface{})) *LoggingMiddleware {
	return &LoggingMiddleware{logger: logger}
}

// Handle implements the Middleware interface
func (lm *LoggingMiddleware) Handle(ctx context.Context, job Job, next func(ctx context.Context, job Job) error) error {
	start := time.Now()
	lm.logger("Starting job: %T (Queue: %s, ID: %s)", job, job.GetQueue(), getJobID(job))
	
	err := next(ctx, job)
	
	duration := time.Since(start)
	if err != nil {
		lm.logger("Job failed after %v: %T (Error: %v)", duration, job, err)
	} else {
		lm.logger("Job completed in %v: %T", duration, job)
	}
	
	return err
}

// RetryMiddleware handles job retries
type RetryMiddleware struct {
	policy RetryPolicy
}

// NewRetryMiddleware creates a new retry middleware
func NewRetryMiddleware(policy RetryPolicy) *RetryMiddleware {
	return &RetryMiddleware{policy: policy}
}

// Handle implements the Middleware interface
func (rm *RetryMiddleware) Handle(ctx context.Context, job Job, next func(ctx context.Context, job Job) error) error {
	err := next(ctx, job)
	
	if err != nil && rm.policy != nil {
		if queueJob, ok := job.(*QueueJob); ok {
			if rm.policy.ShouldRetry(queueJob.GetAttempts(), job.GetMaxTries(), err) {
				queueJob.MarkAsRetrying()
				// In a real implementation, this would re-queue the job with a delay
				return nil // Don't treat as failure if we're retrying
			}
		}
	}
	
	return err
}

// getJobID extracts job ID if available
func getJobID(job Job) string {
	if queueJob, ok := job.(*QueueJob); ok {
		return queueJob.GetID()
	}
	return "unknown"
}

// DefaultRetryPolicy implements a simple exponential backoff retry policy
type DefaultRetryPolicy struct {
	BaseDelay time.Duration
	MaxDelay  time.Duration
}

// NewDefaultRetryPolicy creates a new default retry policy
func NewDefaultRetryPolicy() *DefaultRetryPolicy {
	return &DefaultRetryPolicy{
		BaseDelay: 1 * time.Second,
		MaxDelay:  5 * time.Minute,
	}
}

// ShouldRetry determines if a job should be retried
func (drp *DefaultRetryPolicy) ShouldRetry(attempts int, maxTries int, err error) bool {
	return attempts < maxTries
}

// GetDelay calculates the delay before retry
func (drp *DefaultRetryPolicy) GetDelay(attempts int) time.Duration {
	delay := time.Duration(attempts) * drp.BaseDelay
	if delay > drp.MaxDelay {
		delay = drp.MaxDelay
	}
	return delay
}