package events

import (
	"context"
	"fmt"
	"sync"
)

// DefaultRepository implements the Repository interface
type DefaultRepository struct {
	dispatchers map[string]Dispatcher
	defaultName string
	mutex       sync.RWMutex
	metrics     Metrics
}

// NewRepository creates a new event repository
func NewRepository(metrics Metrics) Repository {
	return &DefaultRepository{
		dispatchers: make(map[string]Dispatcher),
		defaultName: "default",
		metrics:     metrics,
	}
}

// GetDispatcher returns a dispatcher instance
func (r *DefaultRepository) GetDispatcher(name ...string) Dispatcher {
	dispatcherName := r.defaultName
	if len(name) > 0 && name[0] != "" {
		dispatcherName = name[0]
	}

	r.mutex.RLock()
	if dispatcher, exists := r.dispatchers[dispatcherName]; exists {
		r.mutex.RUnlock()
		return dispatcher
	}
	r.mutex.RUnlock()

	// Create dispatcher if it doesn't exist
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Double-check after acquiring write lock
	if dispatcher, exists := r.dispatchers[dispatcherName]; exists {
		return dispatcher
	}

	// Create default dispatcher
	dispatcher := r.createDefaultDispatcher(dispatcherName)
	r.dispatchers[dispatcherName] = dispatcher
	return dispatcher
}

// RegisterDispatcher registers a dispatcher with the repository
func (r *DefaultRepository) RegisterDispatcher(name string, dispatcher Dispatcher) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.dispatchers[name] = dispatcher
}

// SetDefaultDispatcher sets the default dispatcher name
func (r *DefaultRepository) SetDefaultDispatcher(name string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.defaultName = name
}

// GetDefaultDispatcher returns the default dispatcher name
func (r *DefaultRepository) GetDefaultDispatcher() string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.defaultName
}

// Dispatch dispatches an event using the default dispatcher
func (r *DefaultRepository) Dispatch(ctx context.Context, event Event) error {
	dispatcher := r.GetDispatcher()
	return dispatcher.Dispatch(ctx, event)
}

// Listen registers a listener with the default dispatcher
func (r *DefaultRepository) Listen(eventName string, listener Listener) {
	dispatcher := r.GetDispatcher()
	dispatcher.Listen(eventName, listener)
}

// ListenFunc registers a listener function with the default dispatcher
func (r *DefaultRepository) ListenFunc(eventName string, listenerFunc ListenerFunc) {
	dispatcher := r.GetDispatcher()
	dispatcher.ListenFunc(eventName, listenerFunc)
}

// Close closes all dispatchers and cleans up resources
func (r *DefaultRepository) Close() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var errors []string

	// Close all dispatchers that implement closer interface
	for name, dispatcher := range r.dispatchers {
		if closer, ok := dispatcher.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				errors = append(errors, fmt.Sprintf("dispatcher %s: %v", name, err))
			}
		}
	}

	// Clear maps
	r.dispatchers = make(map[string]Dispatcher)

	if len(errors) > 0 {
		return fmt.Errorf("errors closing dispatchers: %v", errors)
	}

	return nil
}

// createDefaultDispatcher creates a default dispatcher for a given name
func (r *DefaultRepository) createDefaultDispatcher(name string) Dispatcher {
	switch name {
	case "async":
		return NewAsyncDispatcher(r.metrics, 5) // 5 workers
	case "sync":
		return NewSyncDispatcher(r.metrics)
	default:
		return NewDefaultDispatcher(r.metrics)
	}
}

// GetAvailableDispatchers returns a list of available dispatcher names
func (r *DefaultRepository) GetAvailableDispatchers() []string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	dispatchers := make([]string, 0, len(r.dispatchers))
	for name := range r.dispatchers {
		dispatchers = append(dispatchers, name)
	}
	return dispatchers
}

// GetDispatcherStats returns statistics for all dispatchers
func (r *DefaultRepository) GetDispatcherStats() map[string]interface{} {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	stats := make(map[string]interface{})
	for name, dispatcher := range r.dispatchers {
		dispatcherStats := map[string]interface{}{
			"type":              fmt.Sprintf("%T", dispatcher),
			"supports_wildcards": dispatcher.SupportsWildcards(),
		}

		// Add async-specific stats
		if asyncDispatcher, ok := dispatcher.(*AsyncDispatcher); ok {
			dispatcherStats["queue_size"] = asyncDispatcher.GetQueueSize()
			dispatcherStats["workers"] = asyncDispatcher.workers
		}

		stats[name] = dispatcherStats
	}
	return stats
}

// Global repository instance
var globalRepository Repository

// SetupEvents initializes the global event repository
func SetupEvents(metrics Metrics) {
	globalRepository = NewRepository(metrics)
}

// GetRepository returns the global event repository
func GetRepository() Repository {
	if globalRepository == nil {
		globalRepository = NewRepository(NewSimpleMetrics())
	}
	return globalRepository
}

// Global event functions

// GetDispatcher returns a dispatcher from the global repository
func GetDispatcher(name ...string) Dispatcher {
	return GetRepository().GetDispatcher(name...)
}

// RegisterDispatcher registers a dispatcher with the global repository
func RegisterDispatcher(name string, dispatcher Dispatcher) {
	GetRepository().RegisterDispatcher(name, dispatcher)
}

// SetDefaultDispatcher sets the default dispatcher in the global repository
func SetDefaultDispatcher(name string) {
	GetRepository().SetDefaultDispatcher(name)
}

// Dispatch dispatches an event using the global repository
func Dispatch(ctx context.Context, event Event) error {
	return GetRepository().Dispatch(ctx, event)
}

// Listen registers a listener with the global repository
func Listen(eventName string, listener Listener) {
	GetRepository().Listen(eventName, listener)
}

// ListenFunc registers a listener function with the global repository
func ListenFunc(eventName string, listenerFunc ListenerFunc) {
	GetRepository().ListenFunc(eventName, listenerFunc)
}

// Fire is a convenience function for dispatching events
func Fire(ctx context.Context, eventName string, payload interface{}) error {
	event := NewEventWithContext(ctx, eventName, payload)
	return Dispatch(ctx, event)
}

// FireSync fires an event synchronously
func FireSync(ctx context.Context, eventName string, payload interface{}) error {
	dispatcher := GetDispatcher("sync")
	event := NewEventWithContext(ctx, eventName, payload)
	return dispatcher.Dispatch(ctx, event)
}

// FireAsync fires an event asynchronously
func FireAsync(ctx context.Context, eventName string, payload interface{}) error {
	dispatcher := GetDispatcher("async")
	event := NewEventWithContext(ctx, eventName, payload)
	
	if asyncDispatcher, ok := dispatcher.(interface {
		DispatchAsync(ctx context.Context, event Event) error
	}); ok {
		return asyncDispatcher.DispatchAsync(ctx, event)
	}
	
	// Fallback to regular dispatch
	return dispatcher.Dispatch(ctx, event)
}