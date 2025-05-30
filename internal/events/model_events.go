package events

import (
	"context"
	"fmt"
	"reflect"
	"sync"
)

// ModelEvent represents the type of event that occurred on a model
type ModelEvent string

const (
	// Before events - can prevent the operation
	EventModelCreating ModelEvent = "creating"
	EventModelUpdating ModelEvent = "updating"
	EventModelSaving   ModelEvent = "saving"   // Before create or update
	EventModelDeleting ModelEvent = "deleting"

	// After events - read-only, operation already completed
	EventModelCreated ModelEvent = "created"
	EventModelUpdated ModelEvent = "updated"
	EventModelSaved   ModelEvent = "saved"   // After create or update
	EventModelDeleted ModelEvent = "deleted"
)

// String returns the string representation of ModelEvent
func (me ModelEvent) String() string {
	return string(me)
}

// ModelEventHandler defines the interface for handling model events with context
type ModelEventHandler func(ctx context.Context, model interface{}) error

// ModelEventContext provides context for model events
type ModelEventContext struct {
	Model     interface{}
	Event     ModelEvent
	ModelName string
	Fields    map[string]interface{} // Changed/dirty fields
	Original  map[string]interface{} // Original values (for updates)
	Context   context.Context
	Metadata  map[string]interface{}
}

// EventableModel interface for models that support events
type EventableModel interface {
	GetModelName() string
	GetEventContext() *ModelEventContext
}

// ModelLifecycleObserver interface for implementing observer pattern on models
type ModelLifecycleObserver interface {
	Creating(ctx context.Context, model interface{}) error
	Created(ctx context.Context, model interface{}) error
	Updating(ctx context.Context, model interface{}) error
	Updated(ctx context.Context, model interface{}) error
	Saving(ctx context.Context, model interface{}) error
	Saved(ctx context.Context, model interface{}) error
	Deleting(ctx context.Context, model interface{}) error
	Deleted(ctx context.Context, model interface{}) error
}

// BaseModelLifecycleObserver provides default implementations for ModelLifecycleObserver
type BaseModelLifecycleObserver struct{}

func (bo *BaseModelLifecycleObserver) Creating(ctx context.Context, model interface{}) error { return nil }
func (bo *BaseModelLifecycleObserver) Created(ctx context.Context, model interface{}) error  { return nil }
func (bo *BaseModelLifecycleObserver) Updating(ctx context.Context, model interface{}) error { return nil }
func (bo *BaseModelLifecycleObserver) Updated(ctx context.Context, model interface{}) error  { return nil }
func (bo *BaseModelLifecycleObserver) Saving(ctx context.Context, model interface{}) error   { return nil }
func (bo *BaseModelLifecycleObserver) Saved(ctx context.Context, model interface{}) error    { return nil }
func (bo *BaseModelLifecycleObserver) Deleting(ctx context.Context, model interface{}) error { return nil }
func (bo *BaseModelLifecycleObserver) Deleted(ctx context.Context, model interface{}) error  { return nil }

// ModelEventDispatcher manages model events and observers with context support
type ModelEventDispatcher struct {
	observers map[string][]ModelLifecycleObserver          // Model name -> observers
	handlers  map[string]map[ModelEvent][]ModelEventHandler // Model name -> event -> handlers
	mutex     sync.RWMutex
	metrics   Metrics
}

// NewModelEventDispatcher creates a new model event dispatcher
func NewModelEventDispatcher(metrics Metrics) *ModelEventDispatcher {
	return &ModelEventDispatcher{
		observers: make(map[string][]ModelLifecycleObserver),
		handlers:  make(map[string]map[ModelEvent][]ModelEventHandler),
		metrics:   metrics,
	}
}

// ClearObservers clears all registered observers (useful for testing)
func (ed *ModelEventDispatcher) ClearObservers() {
	ed.mutex.Lock()
	defer ed.mutex.Unlock()
	ed.observers = make(map[string][]ModelLifecycleObserver)
	ed.handlers = make(map[string]map[ModelEvent][]ModelEventHandler)
}

// RegisterObserver registers an observer for a specific model type
func (ed *ModelEventDispatcher) RegisterObserver(modelName string, observer ModelLifecycleObserver) {
	ed.mutex.Lock()
	defer ed.mutex.Unlock()

	if ed.observers[modelName] == nil {
		ed.observers[modelName] = make([]ModelLifecycleObserver, 0)
	}
	ed.observers[modelName] = append(ed.observers[modelName], observer)
}

// RegisterHandler registers an event handler for a specific model and event
func (ed *ModelEventDispatcher) RegisterHandler(modelName string, event ModelEvent, handler ModelEventHandler) {
	ed.mutex.Lock()
	defer ed.mutex.Unlock()

	if ed.handlers[modelName] == nil {
		ed.handlers[modelName] = make(map[ModelEvent][]ModelEventHandler)
	}
	if ed.handlers[modelName][event] == nil {
		ed.handlers[modelName][event] = make([]ModelEventHandler, 0)
	}
	ed.handlers[modelName][event] = append(ed.handlers[modelName][event], handler)
}

// DispatchEvent dispatches a model event to all registered observers and handlers
func (ed *ModelEventDispatcher) DispatchEvent(ctx context.Context, event ModelEvent, model interface{}) error {
	modelName := extractModelName(model)
	eventName := fmt.Sprintf("model.%s.%s", modelName, event)

	if ed.metrics != nil {
		defer func(start int64) {
			ed.metrics.RecordEvent(eventName, start)
		}(timeNow())
	}

	ed.mutex.RLock()
	defer ed.mutex.RUnlock()

	// Call observers first
	if observers, exists := ed.observers[modelName]; exists {
		for _, observer := range observers {
			if err := ed.callObserverMethod(ctx, observer, event, model); err != nil {
				if ed.metrics != nil {
					ed.metrics.RecordError(eventName, err)
				}
				return &ModelEventError{
					Event:     event,
					ModelName: modelName,
					Err:       err,
				}
			}
		}
	}

	// Call registered handlers
	if modelHandlers, exists := ed.handlers[modelName]; exists {
		if handlers, exists := modelHandlers[event]; exists {
			for _, handler := range handlers {
				if err := handler(ctx, model); err != nil {
					if ed.metrics != nil {
						ed.metrics.RecordError(eventName, err)
					}
					return &ModelEventError{
						Event:     event,
						ModelName: modelName,
						Err:       err,
					}
				}
			}
		}
	}

	return nil
}

// callObserverMethod calls the appropriate observer method based on the event
func (ed *ModelEventDispatcher) callObserverMethod(ctx context.Context, observer ModelLifecycleObserver, event ModelEvent, model interface{}) error {
	switch event {
	case EventModelCreating:
		return observer.Creating(ctx, model)
	case EventModelCreated:
		return observer.Created(ctx, model)
	case EventModelUpdating:
		return observer.Updating(ctx, model)
	case EventModelUpdated:
		return observer.Updated(ctx, model)
	case EventModelSaving:
		return observer.Saving(ctx, model)
	case EventModelSaved:
		return observer.Saved(ctx, model)
	case EventModelDeleting:
		return observer.Deleting(ctx, model)
	case EventModelDeleted:
		return observer.Deleted(ctx, model)
	default:
		return fmt.Errorf("unknown model event: %s", event)
	}
}

// extractModelName extracts the model name from a model instance
func extractModelName(model interface{}) string {
	if eventable, ok := model.(EventableModel); ok {
		return eventable.GetModelName()
	}

	// Fallback to reflection
	modelType := reflect.TypeOf(model)
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}
	return modelType.Name()
}

// GetObservers returns all observers for a model
func (ed *ModelEventDispatcher) GetObservers(modelName string) []ModelLifecycleObserver {
	ed.mutex.RLock()
	defer ed.mutex.RUnlock()

	observers, exists := ed.observers[modelName]
	if !exists {
		return nil
	}

	// Return a copy to prevent race conditions
	result := make([]ModelLifecycleObserver, len(observers))
	copy(result, observers)
	return result
}

// GetHandlers returns all handlers for a model and event
func (ed *ModelEventDispatcher) GetHandlers(modelName string, event ModelEvent) []ModelEventHandler {
	ed.mutex.RLock()
	defer ed.mutex.RUnlock()

	modelHandlers, exists := ed.handlers[modelName]
	if !exists {
		return nil
	}

	handlers, exists := modelHandlers[event]
	if !exists {
		return nil
	}

	// Return a copy to prevent race conditions
	result := make([]ModelEventHandler, len(handlers))
	copy(result, handlers)
	return result
}

// HasObservers checks if there are observers for a model
func (ed *ModelEventDispatcher) HasObservers(modelName string) bool {
	ed.mutex.RLock()
	defer ed.mutex.RUnlock()

	observers, exists := ed.observers[modelName]
	return exists && len(observers) > 0
}

// HasHandlers checks if there are handlers for a model and event
func (ed *ModelEventDispatcher) HasHandlers(modelName string, event ModelEvent) bool {
	ed.mutex.RLock()
	defer ed.mutex.RUnlock()

	modelHandlers, exists := ed.handlers[modelName]
	if !exists {
		return false
	}

	handlers, exists := modelHandlers[event]
	return exists && len(handlers) > 0
}

// ModelEventError represents an error that occurred during model event handling
type ModelEventError struct {
	Event     ModelEvent
	ModelName string
	Err       error
}

func (e *ModelEventError) Error() string {
	return fmt.Sprintf("model event error: %s on %s: %v", e.Event, e.ModelName, e.Err)
}

func (e *ModelEventError) Unwrap() error {
	return e.Err
}

// IsModelEventError checks if an error is a model event error
func IsModelEventError(err error) bool {
	_, ok := err.(*ModelEventError)
	return ok
}

// Global model event dispatcher instance
var globalModelEventDispatcher *ModelEventDispatcher

// SetupModelEvents initializes the global model event dispatcher
func SetupModelEvents(metrics Metrics) {
	globalModelEventDispatcher = NewModelEventDispatcher(metrics)
}

// GetModelEventDispatcher returns the global model event dispatcher
func GetModelEventDispatcher() *ModelEventDispatcher {
	if globalModelEventDispatcher == nil {
		globalModelEventDispatcher = NewModelEventDispatcher(nil)
	}
	return globalModelEventDispatcher
}

// Helper functions for easy event registration

// ObserveModel registers an observer for a specific model type
func ObserveModel(modelName string, observer ModelLifecycleObserver) {
	GetModelEventDispatcher().RegisterObserver(modelName, observer)
}

// OnModelEvent registers a handler for a specific model event
func OnModelEvent(modelName string, event ModelEvent, handler ModelEventHandler) {
	GetModelEventDispatcher().RegisterHandler(modelName, event, handler)
}

// Convenience functions for specific events

// OnCreating registers a handler for the creating event
func OnCreating(modelName string, handler ModelEventHandler) {
	OnModelEvent(modelName, EventModelCreating, handler)
}

// OnCreated registers a handler for the created event
func OnCreated(modelName string, handler ModelEventHandler) {
	OnModelEvent(modelName, EventModelCreated, handler)
}

// OnUpdating registers a handler for the updating event
func OnUpdating(modelName string, handler ModelEventHandler) {
	OnModelEvent(modelName, EventModelUpdating, handler)
}

// OnUpdated registers a handler for the updated event
func OnUpdated(modelName string, handler ModelEventHandler) {
	OnModelEvent(modelName, EventModelUpdated, handler)
}

// OnSaving registers a handler for the saving event (before create or update)
func OnSaving(modelName string, handler ModelEventHandler) {
	OnModelEvent(modelName, EventModelSaving, handler)
}

// OnSaved registers a handler for the saved event (after create or update)
func OnSaved(modelName string, handler ModelEventHandler) {
	OnModelEvent(modelName, EventModelSaved, handler)
}

// OnDeleting registers a handler for the deleting event
func OnDeleting(modelName string, handler ModelEventHandler) {
	OnModelEvent(modelName, EventModelDeleting, handler)
}

// OnDeleted registers a handler for the deleted event
func OnDeleted(modelName string, handler ModelEventHandler) {
	OnModelEvent(modelName, EventModelDeleted, handler)
}

// DispatchModelEvent dispatches a model event using the global dispatcher
func DispatchModelEvent(ctx context.Context, event ModelEvent, model interface{}) error {
	return GetModelEventDispatcher().DispatchEvent(ctx, event, model)
}