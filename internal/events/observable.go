package events

import (
	"context"
	"fmt"
	"reflect"
	"sync"
)

// DefaultObservable implements the Observable interface
type DefaultObservable struct {
	observers []Observer
	mutex     sync.RWMutex
	metrics   Metrics
}

// NewObservable creates a new observable
func NewObservable(metrics Metrics) *DefaultObservable {
	return &DefaultObservable{
		observers: make([]Observer, 0),
		metrics:   metrics,
	}
}

// Attach adds an observer to the observable
func (o *DefaultObservable) Attach(observer Observer) {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	o.observers = append(o.observers, observer)
}

// Detach removes an observer from the observable
func (o *DefaultObservable) Detach(observer Observer) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	for i, obs := range o.observers {
		if obs == observer {
			o.observers = append(o.observers[:i], o.observers[i+1:]...)
			break
		}
	}
}

// Notify notifies all observers about a change
func (o *DefaultObservable) Notify(ctx context.Context, subject interface{}, event string, data interface{}) error {
	o.mutex.RLock()
	observers := make([]Observer, len(o.observers))
	copy(observers, o.observers)
	o.mutex.RUnlock()

	eventName := "observer." + event
	if o.metrics != nil {
		defer func(start int64) {
			o.metrics.RecordEvent(eventName, start)
		}(timeNow())
	}

	for _, observer := range observers {
		observerStart := timeNow()
		err := observer.Update(ctx, subject, event, data)
		
		if o.metrics != nil {
			observerName := getObserverName(observer)
			o.metrics.RecordListener(observerName, timeNow()-observerStart, err == nil)
			if err != nil {
				o.metrics.RecordError(eventName, err)
			}
		}
		
		if err != nil {
			return err
		}
	}

	return nil
}

// GetObservers returns a copy of all observers
func (o *DefaultObservable) GetObservers() []Observer {
	o.mutex.RLock()
	defer o.mutex.RUnlock()

	observers := make([]Observer, len(o.observers))
	copy(observers, o.observers)
	return observers
}

// HasObservers returns true if there are any observers
func (o *DefaultObservable) HasObservers() bool {
	o.mutex.RLock()
	defer o.mutex.RUnlock()
	return len(o.observers) > 0
}

// ObserverCount returns the number of observers
func (o *DefaultObservable) ObserverCount() int {
	o.mutex.RLock()
	defer o.mutex.RUnlock()
	return len(o.observers)
}

// Clear removes all observers
func (o *DefaultObservable) Clear() {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	o.observers = make([]Observer, 0)
}

// getObserverName extracts a name for metrics from an observer
func getObserverName(observer Observer) string {
	return reflect.TypeOf(observer).String()
}

// ObserverFunc is a function that implements the Observer interface
type ObserverFunc func(ctx context.Context, subject interface{}, event string, data interface{}) error

// Update implements the Observer interface
func (of ObserverFunc) Update(ctx context.Context, subject interface{}, event string, data interface{}) error {
	return of(ctx, subject, event, data)
}

// ModelObserver observes model changes and logs them
type ModelObserver struct {
	name    string
	logger  func(message string)
	metrics Metrics
}

// NewModelObserver creates a new model observer
func NewModelObserver(name string, logger func(message string), metrics Metrics) *ModelObserver {
	return &ModelObserver{
		name:    name,
		logger:  logger,
		metrics: metrics,
	}
}

// Update implements the Observer interface
func (mo *ModelObserver) Update(ctx context.Context, subject interface{}, event string, data interface{}) error {
	subjectType := reflect.TypeOf(subject)
	if subjectType.Kind() == reflect.Ptr {
		subjectType = subjectType.Elem()
	}

	message := fmt.Sprintf("[%s] Model %s: %s with data: %+v", mo.name, subjectType.Name(), event, data)
	if mo.logger != nil {
		mo.logger(message)
	}

	return nil
}

// ConditionalObserver wraps an observer with a condition
type ConditionalObserver struct {
	observer  Observer
	condition func(ctx context.Context, subject interface{}, event string, data interface{}) bool
}

// NewConditionalObserver creates a new conditional observer
func NewConditionalObserver(observer Observer, condition func(ctx context.Context, subject interface{}, event string, data interface{}) bool) *ConditionalObserver {
	return &ConditionalObserver{
		observer:  observer,
		condition: condition,
	}
}

// Update implements the Observer interface
func (co *ConditionalObserver) Update(ctx context.Context, subject interface{}, event string, data interface{}) error {
	if co.condition(ctx, subject, event, data) {
		return co.observer.Update(ctx, subject, event, data)
	}
	return nil
}

// AsyncObserver wraps an observer to run asynchronously
type AsyncObserver struct {
	observer Observer
	workers  int
	queue    chan observerJob
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// observerJob represents a job for async processing
type observerJob struct {
	ctx     context.Context
	subject interface{}
	event   string
	data    interface{}
	done    chan error
}

// NewAsyncObserver creates a new async observer
func NewAsyncObserver(observer Observer, workers int) *AsyncObserver {
	if workers <= 0 {
		workers = 1
	}

	ao := &AsyncObserver{
		observer: observer,
		workers:  workers,
		queue:    make(chan observerJob, workers*2),
		stopChan: make(chan struct{}),
	}

	// Start worker goroutines
	for i := 0; i < workers; i++ {
		ao.wg.Add(1)
		go ao.worker()
	}

	return ao
}

// worker processes observer jobs asynchronously
func (ao *AsyncObserver) worker() {
	defer ao.wg.Done()

	for {
		select {
		case job := <-ao.queue:
			err := ao.observer.Update(job.ctx, job.subject, job.event, job.data)
			if job.done != nil {
				job.done <- err
			}
		case <-ao.stopChan:
			return
		}
	}
}

// Update implements the Observer interface
func (ao *AsyncObserver) Update(ctx context.Context, subject interface{}, event string, data interface{}) error {
	job := observerJob{
		ctx:     ctx,
		subject: subject,
		event:   event,
		data:    data,
		done:    make(chan error, 1),
	}

	select {
	case ao.queue <- job:
		return <-job.done
	default:
		return fmt.Errorf("observer queue is full")
	}
}

// UpdateAsync updates asynchronously without waiting for completion
func (ao *AsyncObserver) UpdateAsync(ctx context.Context, subject interface{}, event string, data interface{}) error {
	job := observerJob{
		ctx:     ctx,
		subject: subject,
		event:   event,
		data:    data,
		done:    nil,
	}

	select {
	case ao.queue <- job:
		return nil
	default:
		return fmt.Errorf("observer queue is full")
	}
}

// Close stops the async observer
func (ao *AsyncObserver) Close() error {
	close(ao.stopChan)
	ao.wg.Wait()
	return nil
}

// ObserverGroup manages a group of observers
type ObserverGroup struct {
	observers map[string]Observer
	mutex     sync.RWMutex
}

// NewObserverGroup creates a new observer group
func NewObserverGroup() *ObserverGroup {
	return &ObserverGroup{
		observers: make(map[string]Observer),
	}
}

// Add adds an observer to the group
func (og *ObserverGroup) Add(name string, observer Observer) {
	og.mutex.Lock()
	defer og.mutex.Unlock()
	og.observers[name] = observer
}

// Remove removes an observer from the group
func (og *ObserverGroup) Remove(name string) {
	og.mutex.Lock()
	defer og.mutex.Unlock()
	delete(og.observers, name)
}

// Update implements the Observer interface for the group
func (og *ObserverGroup) Update(ctx context.Context, subject interface{}, event string, data interface{}) error {
	og.mutex.RLock()
	observers := make(map[string]Observer)
	for name, observer := range og.observers {
		observers[name] = observer
	}
	og.mutex.RUnlock()

	for name, observer := range observers {
		if err := observer.Update(ctx, subject, event, data); err != nil {
			return fmt.Errorf("observer %s error: %w", name, err)
		}
	}

	return nil
}

// GetObserver returns an observer by name
func (og *ObserverGroup) GetObserver(name string) (Observer, bool) {
	og.mutex.RLock()
	defer og.mutex.RUnlock()
	observer, exists := og.observers[name]
	return observer, exists
}

// List returns all observer names
func (og *ObserverGroup) List() []string {
	og.mutex.RLock()
	defer og.mutex.RUnlock()

	names := make([]string, 0, len(og.observers))
	for name := range og.observers {
		names = append(names, name)
	}
	return names
}

// Clear removes all observers
func (og *ObserverGroup) Clear() {
	og.mutex.Lock()
	defer og.mutex.Unlock()
	og.observers = make(map[string]Observer)
}