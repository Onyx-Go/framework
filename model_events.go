package onyx

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
	EventCreating ModelEvent = "creating"
	EventUpdating ModelEvent = "updating" 
	EventSaving   ModelEvent = "saving"   // Before create or update
	EventDeleting ModelEvent = "deleting"
	
	// After events - read-only, operation already completed
	EventCreated ModelEvent = "created"
	EventUpdated ModelEvent = "updated"
	EventSaved   ModelEvent = "saved"   // After create or update
	EventDeleted ModelEvent = "deleted"
)

// ModelEventHandler defines the interface for handling model events
type ModelEventHandler func(ctx context.Context, model interface{}) error

// ModelEventContext provides context for model events
type ModelEventContext struct {
	Model     interface{}
	Event     ModelEvent
	ModelName string
	Fields    map[string]interface{} // Changed/dirty fields
	Original  map[string]interface{} // Original values (for updates)
}

// EventableModel interface for models that support events
type EventableModel interface {
	Model
	GetModelName() string
	GetEventContext() *ModelEventContext
}

// ModelLifecycleObserver interface for implementing observer pattern
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

// ModelEventDispatcher manages model events and observers
type ModelEventDispatcher struct {
	observers map[string][]ModelLifecycleObserver          // Model name -> observers
	handlers  map[string]map[ModelEvent][]ModelEventHandler // Model name -> event -> handlers
	mutex     sync.RWMutex
}

// Global model event dispatcher instance
var globalModelEventDispatcher = NewModelEventDispatcher()

// NewModelEventDispatcher creates a new model event dispatcher
func NewModelEventDispatcher() *ModelEventDispatcher {
	return &ModelEventDispatcher{
		observers: make(map[string][]ModelLifecycleObserver),
		handlers:  make(map[string]map[ModelEvent][]ModelEventHandler),
	}
}

// GetModelEventDispatcher returns the global model event dispatcher
func GetModelEventDispatcher() *ModelEventDispatcher {
	return globalModelEventDispatcher
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

// DispatchEvent dispatches an event to all registered observers and handlers
func (ed *ModelEventDispatcher) DispatchEvent(ctx context.Context, event ModelEvent, model interface{}) error {
	modelName := extractModelName(model)
	
	ed.mutex.RLock()
	defer ed.mutex.RUnlock()
	
	// Call observers first
	if observers, exists := ed.observers[modelName]; exists {
		for _, observer := range observers {
			if err := ed.callObserverMethod(ctx, observer, event, model); err != nil {
				return fmt.Errorf("observer error for %s.%s: %w", modelName, event, err)
			}
		}
	}
	
	// Call registered handlers
	if modelHandlers, exists := ed.handlers[modelName]; exists {
		if handlers, exists := modelHandlers[event]; exists {
			for _, handler := range handlers {
				if err := handler(ctx, model); err != nil {
					return fmt.Errorf("handler error for %s.%s: %w", modelName, event, err)
				}
			}
		}
	}
	
	return nil
}

// callObserverMethod calls the appropriate observer method based on the event
func (ed *ModelEventDispatcher) callObserverMethod(ctx context.Context, observer ModelLifecycleObserver, event ModelEvent, model interface{}) error {
	switch event {
	case EventCreating:
		return observer.Creating(ctx, model)
	case EventCreated:
		return observer.Created(ctx, model)
	case EventUpdating:
		return observer.Updating(ctx, model)
	case EventUpdated:
		return observer.Updated(ctx, model)
	case EventSaving:
		return observer.Saving(ctx, model)
	case EventSaved:
		return observer.Saved(ctx, model)
	case EventDeleting:
		return observer.Deleting(ctx, model)
	case EventDeleted:
		return observer.Deleted(ctx, model)
	default:
		return fmt.Errorf("unknown event: %s", event)
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

// Helper functions for easy event registration

// ObserveModel registers an observer for a specific model type
func ObserveModel(modelName string, observer ModelLifecycleObserver) {
	globalModelEventDispatcher.RegisterObserver(modelName, observer)
}

// OnModelEvent registers a handler for a specific model event
func OnModelEvent(modelName string, event ModelEvent, handler ModelEventHandler) {
	globalModelEventDispatcher.RegisterHandler(modelName, event, handler)
}

// Convenience functions for specific events

// OnCreating registers a handler for the creating event
func OnCreating(modelName string, handler ModelEventHandler) {
	OnModelEvent(modelName, EventCreating, handler)
}

// OnCreated registers a handler for the created event
func OnCreated(modelName string, handler ModelEventHandler) {
	OnModelEvent(modelName, EventCreated, handler)
}

// OnUpdating registers a handler for the updating event
func OnUpdating(modelName string, handler ModelEventHandler) {
	OnModelEvent(modelName, EventUpdating, handler)
}

// OnUpdated registers a handler for the updated event
func OnUpdated(modelName string, handler ModelEventHandler) {
	OnModelEvent(modelName, EventUpdated, handler)
}

// OnSaving registers a handler for the saving event (before create or update)
func OnSaving(modelName string, handler ModelEventHandler) {
	OnModelEvent(modelName, EventSaving, handler)
}

// OnSaved registers a handler for the saved event (after create or update)
func OnSaved(modelName string, handler ModelEventHandler) {
	OnModelEvent(modelName, EventSaved, handler)
}

// OnDeleting registers a handler for the deleting event
func OnDeleting(modelName string, handler ModelEventHandler) {
	OnModelEvent(modelName, EventDeleting, handler)
}

// OnDeleted registers a handler for the deleted event
func OnDeleted(modelName string, handler ModelEventHandler) {
	OnModelEvent(modelName, EventDeleted, handler)
}

// ModelEventError represents an error that occurred during event handling
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