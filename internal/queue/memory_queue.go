package queue

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// MemoryQueue implements an in-memory queue
type MemoryQueue struct {
	queues map[string][]*JobPayload
	mutex  sync.RWMutex
	stats  map[string]*QueueStats
	closed bool
}

// NewMemoryQueue creates a new memory queue
func NewMemoryQueue() *MemoryQueue {
	return &MemoryQueue{
		queues: make(map[string][]*JobPayload),
		stats:  make(map[string]*QueueStats),
		closed: false,
	}
}

// Push adds a job to the queue
func (mq *MemoryQueue) Push(ctx context.Context, job Job) error {
	return mq.PushOn(ctx, job.GetQueue(), job)
}

// PushOn adds a job to a specific queue
func (mq *MemoryQueue) PushOn(ctx context.Context, queue string, job Job) error {
	return mq.LaterOn(ctx, queue, 0, job)
}

// Later adds a job to be processed after a delay
func (mq *MemoryQueue) Later(ctx context.Context, delay time.Duration, job Job) error {
	return mq.LaterOn(ctx, job.GetQueue(), delay, job)
}

// LaterOn adds a delayed job to a specific queue
func (mq *MemoryQueue) LaterOn(ctx context.Context, queue string, delay time.Duration, job Job) error {
	if mq.closed {
		return fmt.Errorf("queue is closed")
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	payload := &JobPayload{
		ID:          GenerateJobID(),
		DisplayName: fmt.Sprintf("%T", job),
		Job:         fmt.Sprintf("%T", job),
		MaxTries:    job.GetMaxTries(),
		Timeout:     int(job.GetTimeout().Seconds()),
		Priority:    job.GetPriority(),
		Data:        job.GetPayload(),
		Metadata:    job.GetMetadata(),
		Queue:       queue,
		Attempts:    0,
		CreatedAt:   time.Now(),
		AvailableAt: time.Now().Add(delay),
	}

	mq.mutex.Lock()
	defer mq.mutex.Unlock()

	if _, exists := mq.queues[queue]; !exists {
		mq.queues[queue] = make([]*JobPayload, 0)
		mq.stats[queue] = &QueueStats{
			Name:      queue,
			Size:      0,
			TotalJobs: 0,
		}
	}

	// Insert job in priority order
	mq.queues[queue] = mq.insertByPriority(mq.queues[queue], payload)
	mq.stats[queue].Size++
	mq.stats[queue].TotalJobs++

	return nil
}

// Pop retrieves the next job from the queue
func (mq *MemoryQueue) Pop(ctx context.Context, queue ...string) (Job, error) {
	if mq.closed {
		return nil, fmt.Errorf("queue is closed")
	}

	queueName := "default"
	if len(queue) > 0 {
		queueName = queue[0]
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	mq.mutex.Lock()
	defer mq.mutex.Unlock()

	jobs, exists := mq.queues[queueName]
	if !exists || len(jobs) == 0 {
		return nil, fmt.Errorf("no jobs available in queue %s", queueName)
	}

	now := time.Now()
	for i, payload := range jobs {
		if payload.AvailableAt.Before(now) || payload.AvailableAt.Equal(now) {
			// Remove job from queue
			mq.queues[queueName] = append(jobs[:i], jobs[i+1:]...)
			mq.stats[queueName].Size--
			mq.stats[queueName].Processing++
			
			// Update stats
			if mq.stats[queueName].LastJobAt == nil || payload.CreatedAt.After(*mq.stats[queueName].LastJobAt) {
				mq.stats[queueName].LastJobAt = &payload.CreatedAt
			}
			
			return mq.createJobFromPayload(payload), nil
		}
	}

	return nil, fmt.Errorf("no jobs available in queue %s", queueName)
}

// Peek looks at the next job without removing it
func (mq *MemoryQueue) Peek(ctx context.Context, queue ...string) (Job, error) {
	if mq.closed {
		return nil, fmt.Errorf("queue is closed")
	}

	queueName := "default"
	if len(queue) > 0 {
		queueName = queue[0]
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	mq.mutex.RLock()
	defer mq.mutex.RUnlock()

	jobs, exists := mq.queues[queueName]
	if !exists || len(jobs) == 0 {
		return nil, fmt.Errorf("no jobs available in queue %s", queueName)
	}

	now := time.Now()
	for _, payload := range jobs {
		if payload.AvailableAt.Before(now) || payload.AvailableAt.Equal(now) {
			return mq.createJobFromPayload(payload), nil
		}
	}

	return nil, fmt.Errorf("no jobs available in queue %s", queueName)
}

// Size returns the number of jobs in the queue
func (mq *MemoryQueue) Size(ctx context.Context, queue ...string) (int, error) {
	queueName := "default"
	if len(queue) > 0 {
		queueName = queue[0]
	}

	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	mq.mutex.RLock()
	defer mq.mutex.RUnlock()

	if jobs, exists := mq.queues[queueName]; exists {
		return len(jobs), nil
	}
	return 0, nil
}

// Clear removes all jobs from a queue
func (mq *MemoryQueue) Clear(ctx context.Context, queue string) error {
	if mq.closed {
		return fmt.Errorf("queue is closed")
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	mq.mutex.Lock()
	defer mq.mutex.Unlock()

	delete(mq.queues, queue)
	if stats, exists := mq.stats[queue]; exists {
		stats.Size = 0
	}

	return nil
}

// Stats returns queue statistics
func (mq *MemoryQueue) Stats(ctx context.Context, queue ...string) (*QueueStats, error) {
	queueName := "default"
	if len(queue) > 0 {
		queueName = queue[0]
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	mq.mutex.RLock()
	defer mq.mutex.RUnlock()

	if stats, exists := mq.stats[queueName]; exists {
		// Return a copy
		statsCopy := *stats
		return &statsCopy, nil
	}

	return &QueueStats{
		Name: queueName,
		Size: 0,
	}, nil
}

// Close closes the queue
func (mq *MemoryQueue) Close() error {
	mq.mutex.Lock()
	defer mq.mutex.Unlock()

	mq.closed = true
	mq.queues = make(map[string][]*JobPayload)
	mq.stats = make(map[string]*QueueStats)

	return nil
}

// insertByPriority inserts a job payload in the correct position based on priority
func (mq *MemoryQueue) insertByPriority(jobs []*JobPayload, payload *JobPayload) []*JobPayload {
	// Find the correct position to insert based on priority and creation time
	insertIndex := len(jobs)
	
	for i, job := range jobs {
		if payload.Priority > job.Priority {
			insertIndex = i
			break
		} else if payload.Priority == job.Priority && payload.CreatedAt.Before(job.CreatedAt) {
			insertIndex = i
			break
		}
	}

	// Insert at the correct position
	jobs = append(jobs, nil)
	copy(jobs[insertIndex+1:], jobs[insertIndex:])
	jobs[insertIndex] = payload

	return jobs
}

// createJobFromPayload creates a Job from a JobPayload
func (mq *MemoryQueue) createJobFromPayload(payload *JobPayload) Job {
	baseJob := NewBaseJob()
	baseJob.queue = payload.Queue
	baseJob.maxTries = payload.MaxTries
	baseJob.timeout = time.Duration(payload.Timeout) * time.Second
	baseJob.priority = payload.Priority
	baseJob.payload = payload.Data
	baseJob.metadata = payload.Metadata

	queueJob := NewQueueJob(baseJob, payload.ID)
	queueJob.attempts = payload.Attempts
	queueJob.createdAt = payload.CreatedAt
	queueJob.processedAt = payload.ProcessedAt
	queueJob.failedAt = payload.FailedAt
	queueJob.lastError = payload.LastError

	return queueJob
}

// GetAllQueues returns all queue names
func (mq *MemoryQueue) GetAllQueues() []string {
	mq.mutex.RLock()
	defer mq.mutex.RUnlock()

	queues := make([]string, 0, len(mq.queues))
	for queueName := range mq.queues {
		queues = append(queues, queueName)
	}

	sort.Strings(queues)
	return queues
}

// GetAllStats returns statistics for all queues
func (mq *MemoryQueue) GetAllStats() map[string]*QueueStats {
	mq.mutex.RLock()
	defer mq.mutex.RUnlock()

	result := make(map[string]*QueueStats)
	for name, stats := range mq.stats {
		statsCopy := *stats
		result[name] = &statsCopy
	}

	return result
}

// MarkJobCompleted marks a job as completed (for statistics)
func (mq *MemoryQueue) MarkJobCompleted(queueName string, duration time.Duration) {
	mq.mutex.Lock()
	defer mq.mutex.Unlock()

	if stats, exists := mq.stats[queueName]; exists {
		stats.Processing--
		stats.Processed++
		
		// Update average time
		if stats.Processed == 1 {
			stats.AverageTime = duration
		} else {
			// Simple moving average
			stats.AverageTime = time.Duration((int64(stats.AverageTime)*int64(stats.Processed-1) + int64(duration)) / int64(stats.Processed))
		}
	}
}

// MarkJobFailed marks a job as failed (for statistics)
func (mq *MemoryQueue) MarkJobFailed(queueName string) {
	mq.mutex.Lock()
	defer mq.mutex.Unlock()

	if stats, exists := mq.stats[queueName]; exists {
		stats.Processing--
		stats.Failed++
	}
}

// PriorityQueue implements a priority-based memory queue
type PriorityQueue struct {
	*MemoryQueue
}

// NewPriorityQueue creates a new priority queue
func NewPriorityQueue() *PriorityQueue {
	return &PriorityQueue{
		MemoryQueue: NewMemoryQueue(),
	}
}

// Pop retrieves the highest priority job first
func (pq *PriorityQueue) Pop(ctx context.Context, queue ...string) (Job, error) {
	if pq.closed {
		return nil, fmt.Errorf("queue is closed")
	}

	queueName := "default"
	if len(queue) > 0 {
		queueName = queue[0]
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	pq.mutex.Lock()
	defer pq.mutex.Unlock()

	jobs, exists := pq.queues[queueName]
	if !exists || len(jobs) == 0 {
		return nil, fmt.Errorf("no jobs available in queue %s", queueName)
	}

	now := time.Now()
	
	// Sort by priority (highest first) and then by creation time (oldest first)
	sort.Slice(jobs, func(i, j int) bool {
		if jobs[i].Priority != jobs[j].Priority {
			return jobs[i].Priority > jobs[j].Priority
		}
		return jobs[i].CreatedAt.Before(jobs[j].CreatedAt)
	})

	for i, payload := range jobs {
		if payload.AvailableAt.Before(now) || payload.AvailableAt.Equal(now) {
			// Remove job from queue
			pq.queues[queueName] = append(jobs[:i], jobs[i+1:]...)
			pq.stats[queueName].Size--
			pq.stats[queueName].Processing++
			
			// Update stats
			if pq.stats[queueName].LastJobAt == nil || payload.CreatedAt.After(*pq.stats[queueName].LastJobAt) {
				pq.stats[queueName].LastJobAt = &payload.CreatedAt
			}
			
			return pq.createJobFromPayload(payload), nil
		}
	}

	return nil, fmt.Errorf("no jobs available in queue %s", queueName)
}