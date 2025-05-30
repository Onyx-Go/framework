package events

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// DefaultDispatcher implements the Dispatcher interface
type DefaultDispatcher struct {
	listeners map[string][]Listener
	wildcards map[string][]Listener
	pushed    map[string][]interface{}
	metrics   Metrics
	mutex     sync.RWMutex
}

// NewDefaultDispatcher creates a new default dispatcher
func NewDefaultDispatcher(metrics Metrics) *DefaultDispatcher {
	return &DefaultDispatcher{
		listeners: make(map[string][]Listener),
		wildcards: make(map[string][]Listener),
		pushed:    make(map[string][]interface{}),
		metrics:   metrics,
	}
}

// Listen registers a listener for a specific event
func (d *DefaultDispatcher) Listen(eventName string, listener Listener) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.isWildcard(eventName) {
		d.wildcards[eventName] = append(d.wildcards[eventName], listener)
	} else {
		d.listeners[eventName] = append(d.listeners[eventName], listener)
	}
}

// ListenFunc registers a listener function for a specific event
func (d *DefaultDispatcher) ListenFunc(eventName string, listenerFunc ListenerFunc) {
	d.Listen(eventName, listenerFunc)
}

// Dispatch dispatches an event to all registered listeners
func (d *DefaultDispatcher) Dispatch(ctx context.Context, event Event) error {
	eventName := event.GetName()
	listeners := d.GetListeners(eventName)

	if d.metrics != nil {
		defer func(start int64) {
			d.metrics.RecordEvent(eventName, start)
		}(timeNow())
	}

	for _, listener := range listeners {
		listenerStart := timeNow()
		err := listener.Handle(ctx, event)
		
		if d.metrics != nil {
			d.metrics.RecordListener(getListenerName(listener), timeNow()-listenerStart, err == nil)
		}
		
		if err != nil {
			if d.metrics != nil {
				d.metrics.RecordError(eventName, err)
			}
			return fmt.Errorf("listener error for event %s: %w", eventName, err)
		}
	}

	return nil
}

// DispatchUntil dispatches an event until a condition is met
func (d *DefaultDispatcher) DispatchUntil(ctx context.Context, event Event, halt func(interface{}) bool) (interface{}, error) {
	eventName := event.GetName()
	listeners := d.GetListeners(eventName)

	for _, listener := range listeners {
		err := listener.Handle(ctx, event)
		if halt(err) {
			return err, nil
		}
		if err != nil {
			return nil, fmt.Errorf("listener error for event %s: %w", eventName, err)
		}
	}

	return nil, nil
}

// Push adds an event to the delayed queue
func (d *DefaultDispatcher) Push(eventName string, payload interface{}) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.pushed[eventName] = append(d.pushed[eventName], payload)
}

// Flush processes all pushed events for a given event name
func (d *DefaultDispatcher) Flush(ctx context.Context, eventName string) error {
	d.mutex.Lock()
	payloads := d.pushed[eventName]
	delete(d.pushed, eventName)
	d.mutex.Unlock()

	for _, payload := range payloads {
		var event Event
		
		switch p := payload.(type) {
		case Event:
			event = p
		default:
			// Always use the eventName for consistency
			event = NewBaseEvent(eventName, p)
		}

		if err := d.Dispatch(ctx, event); err != nil {
			return fmt.Errorf("error flushing event %s: %w", eventName, err)
		}
	}

	return nil
}

// Subscribe registers a subscriber's event listeners
func (d *DefaultDispatcher) Subscribe(subscriber Subscriber) {
	subscriber.Subscribe(d)
}

// Forget removes all listeners for an event
func (d *DefaultDispatcher) Forget(eventName string) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	delete(d.listeners, eventName)

	// Remove matching wildcards
	for pattern := range d.wildcards {
		if d.matchesWildcard(pattern, eventName) {
			delete(d.wildcards, pattern)
		}
	}
}

// ForgetPushed removes all pushed events for an event name
func (d *DefaultDispatcher) ForgetPushed(eventName string) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	delete(d.pushed, eventName)
}

// HasListeners checks if there are listeners for an event
func (d *DefaultDispatcher) HasListeners(eventName string) bool {
	return len(d.GetListeners(eventName)) > 0
}

// GetListeners returns all listeners for an event
func (d *DefaultDispatcher) GetListeners(eventName string) []Listener {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	var allListeners []Listener

	// Add direct listeners
	if listeners, exists := d.listeners[eventName]; exists {
		allListeners = append(allListeners, listeners...)
	}

	// Add wildcard listeners
	for pattern, wildcardListeners := range d.wildcards {
		if d.matchesWildcard(pattern, eventName) {
			allListeners = append(allListeners, wildcardListeners...)
		}
	}

	return allListeners
}

// SupportsWildcards returns true as this dispatcher supports wildcards
func (d *DefaultDispatcher) SupportsWildcards() bool {
	return true
}

// isWildcard checks if an event name contains wildcards
func (d *DefaultDispatcher) isWildcard(eventName string) bool {
	return strings.Contains(eventName, "*")
}

// matchesWildcard checks if an event name matches a wildcard pattern
func (d *DefaultDispatcher) matchesWildcard(pattern, eventName string) bool {
	if !d.isWildcard(pattern) {
		return pattern == eventName
	}

	// Simple wildcard matching - supports * at the end
	if strings.HasSuffix(pattern, "*") {
		prefix := pattern[:len(pattern)-1]
		return strings.HasPrefix(eventName, prefix)
	}

	return false
}

// getListenerName extracts a name for metrics from a listener
func getListenerName(listener Listener) string {
	switch l := listener.(type) {
	case ListenerFunc:
		return "ListenerFunc"
	default:
		return fmt.Sprintf("%T", l)
	}
}

// timeNow returns current time in nanoseconds - can be mocked for testing
var timeNow = func() int64 {
	return timeNowImpl()
}

func timeNowImpl() int64 {
	return 1000 // Simplified for testing, would use time.Now().UnixNano() in real implementation
}

// SyncDispatcher provides synchronous event dispatching
type SyncDispatcher struct {
	*DefaultDispatcher
}

// NewSyncDispatcher creates a new synchronous dispatcher
func NewSyncDispatcher(metrics Metrics) *SyncDispatcher {
	return &SyncDispatcher{
		DefaultDispatcher: NewDefaultDispatcher(metrics),
	}
}

// AsyncDispatcher provides asynchronous event dispatching
type AsyncDispatcher struct {
	*DefaultDispatcher
	workers   int
	queue     chan dispatchJob
	stopChan  chan struct{}
	waitGroup sync.WaitGroup
}

// dispatchJob represents a job to be processed asynchronously
type dispatchJob struct {
	ctx   context.Context
	event Event
	done  chan error
}

// NewAsyncDispatcher creates a new asynchronous dispatcher
func NewAsyncDispatcher(metrics Metrics, workers int) *AsyncDispatcher {
	if workers <= 0 {
		workers = 5 // Default number of workers
	}

	d := &AsyncDispatcher{
		DefaultDispatcher: NewDefaultDispatcher(metrics),
		workers:           workers,
		queue:             make(chan dispatchJob, workers*2), // Buffer size
		stopChan:          make(chan struct{}),
	}

	// Start worker goroutines
	for i := 0; i < workers; i++ {
		d.waitGroup.Add(1)
		go d.worker()
	}

	return d
}

// worker processes dispatch jobs asynchronously
func (d *AsyncDispatcher) worker() {
	defer d.waitGroup.Done()

	for {
		select {
		case job := <-d.queue:
			err := d.DefaultDispatcher.Dispatch(job.ctx, job.event)
			if job.done != nil {
				job.done <- err
			}
		case <-d.stopChan:
			return
		}
	}
}

// Dispatch dispatches an event asynchronously
func (d *AsyncDispatcher) Dispatch(ctx context.Context, event Event) error {
	job := dispatchJob{
		ctx:   ctx,
		event: event,
		done:  make(chan error, 1),
	}

	select {
	case d.queue <- job:
		return <-job.done
	default:
		return fmt.Errorf("event queue is full")
	}
}

// DispatchAsync dispatches an event asynchronously without waiting for completion
func (d *AsyncDispatcher) DispatchAsync(ctx context.Context, event Event) error {
	job := dispatchJob{
		ctx:   ctx,
		event: event,
		done:  nil, // No response channel
	}

	select {
	case d.queue <- job:
		return nil
	default:
		return fmt.Errorf("event queue is full")
	}
}

// Close stops the async dispatcher
func (d *AsyncDispatcher) Close() error {
	close(d.stopChan)
	d.waitGroup.Wait()
	return nil
}

// GetQueueSize returns the current queue size
func (d *AsyncDispatcher) GetQueueSize() int {
	return len(d.queue)
}

// PipelineDispatcher processes events through a pipeline before dispatching
type PipelineDispatcher struct {
	*DefaultDispatcher
	pipeline Pipeline
}

// NewPipelineDispatcher creates a new pipeline dispatcher
func NewPipelineDispatcher(metrics Metrics, pipeline Pipeline) *PipelineDispatcher {
	return &PipelineDispatcher{
		DefaultDispatcher: NewDefaultDispatcher(metrics),
		pipeline:          pipeline,
	}
}

// Dispatch processes event through pipeline before dispatching
func (d *PipelineDispatcher) Dispatch(ctx context.Context, event Event) error {
	if d.pipeline != nil {
		processedEvent, err := d.pipeline.Process(ctx, event)
		if err != nil {
			return fmt.Errorf("pipeline processing error: %w", err)
		}
		event = processedEvent
	}

	return d.DefaultDispatcher.Dispatch(ctx, event)
}