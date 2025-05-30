package events

import (
	"context"
	"sync"
	"time"
)

// BaseEvent provides a basic implementation of the Event interface
type BaseEvent struct {
	name      string
	payload   interface{}
	metadata  map[string]interface{}
	context   context.Context
	timestamp time.Time
	priority  Priority
	mutex     sync.RWMutex
}

// NewBaseEvent creates a new BaseEvent
func NewBaseEvent(name string, payload interface{}) *BaseEvent {
	return &BaseEvent{
		name:      name,
		payload:   payload,
		metadata:  make(map[string]interface{}),
		context:   context.Background(),
		timestamp: time.Now(),
		priority:  PriorityNormal,
	}
}

// NewEventWithContext creates a new BaseEvent with context
func NewEventWithContext(ctx context.Context, name string, payload interface{}) *BaseEvent {
	event := NewBaseEvent(name, payload)
	event.context = ctx
	return event
}

// GetName returns the event name
func (e *BaseEvent) GetName() string {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	return e.name
}

// GetPayload returns the event payload
func (e *BaseEvent) GetPayload() interface{} {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	return e.payload
}

// GetMetadata returns the event metadata
func (e *BaseEvent) GetMetadata() map[string]interface{} {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	
	// Return a copy to prevent race conditions
	result := make(map[string]interface{})
	for k, v := range e.metadata {
		result[k] = v
	}
	return result
}

// SetMetadata sets a metadata value
func (e *BaseEvent) SetMetadata(key string, value interface{}) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.metadata[key] = value
}

// GetMetadataValue gets a specific metadata value
func (e *BaseEvent) GetMetadataValue(key string) (interface{}, bool) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	value, exists := e.metadata[key]
	return value, exists
}

// WithContext returns a new event with the given context
func (e *BaseEvent) WithContext(ctx context.Context) Event {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	
	newEvent := &BaseEvent{
		name:      e.name,
		payload:   e.payload,
		metadata:  make(map[string]interface{}),
		context:   ctx,
		timestamp: e.timestamp,
		priority:  e.priority,
	}
	
	// Copy metadata
	for k, v := range e.metadata {
		newEvent.metadata[k] = v
	}
	
	return newEvent
}

// GetContext returns the event context
func (e *BaseEvent) GetContext() context.Context {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	return e.context
}

// GetTimestamp returns the event timestamp
func (e *BaseEvent) GetTimestamp() time.Time {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	return e.timestamp
}

// GetPriority returns the event priority
func (e *BaseEvent) GetPriority() Priority {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	return e.priority
}

// SetPriority sets the event priority
func (e *BaseEvent) SetPriority(priority Priority) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.priority = priority
}

// Clone creates a copy of the event
func (e *BaseEvent) Clone() Event {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	
	newEvent := &BaseEvent{
		name:      e.name,
		payload:   e.payload,
		metadata:  make(map[string]interface{}),
		context:   e.context,
		timestamp: e.timestamp,
		priority:  e.priority,
	}
	
	// Copy metadata
	for k, v := range e.metadata {
		newEvent.metadata[k] = v
	}
	
	return newEvent
}

// String returns a string representation of the event
func (e *BaseEvent) String() string {
	return e.GetName()
}

// Common event types

// UserEvent represents user-related events
type UserEvent struct {
	*BaseEvent
	UserID   string      `json:"user_id"`
	UserData interface{} `json:"user_data"`
}

// NewUserEvent creates a new user event
func NewUserEvent(name string, userID string, userData interface{}) *UserEvent {
	event := &UserEvent{
		BaseEvent: NewBaseEvent(name, userData),
		UserID:    userID,
		UserData:  userData,
	}
	event.SetMetadata("user_id", userID)
	return event
}

// RequestEvent represents HTTP request events
type RequestEvent struct {
	*BaseEvent
	Method     string            `json:"method"`
	URL        string            `json:"url"`
	Headers    map[string]string `json:"headers"`
	RemoteAddr string            `json:"remote_addr"`
	UserAgent  string            `json:"user_agent"`
}

// NewRequestEvent creates a new request event
func NewRequestEvent(name string, method, url string) *RequestEvent {
	return &RequestEvent{
		BaseEvent: NewBaseEvent(name, nil),
		Method:    method,
		URL:       url,
		Headers:   make(map[string]string),
	}
}

// ErrorEvent represents error events
type ErrorEvent struct {
	*BaseEvent
	Error   error       `json:"error"`
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Stack   string      `json:"stack,omitempty"`
	Context interface{} `json:"context,omitempty"`
}

// NewErrorEvent creates a new error event
func NewErrorEvent(name string, err error) *ErrorEvent {
	return &ErrorEvent{
		BaseEvent: NewBaseEvent(name, err),
		Error:     err,
		Message:   err.Error(),
	}
}

// SystemEvent represents system-level events
type SystemEvent struct {
	*BaseEvent
	Component string                 `json:"component"`
	Action    string                 `json:"action"`
	Details   map[string]interface{} `json:"details"`
}

// NewSystemEvent creates a new system event
func NewSystemEvent(name, component, action string) *SystemEvent {
	return &SystemEvent{
		BaseEvent: NewBaseEvent(name, nil),
		Component: component,
		Action:    action,
		Details:   make(map[string]interface{}),
	}
}

// DatabaseEvent represents database operation events
type DatabaseEvent struct {
	*BaseEvent
	Table     string                 `json:"table"`
	Operation string                 `json:"operation"`
	Query     string                 `json:"query,omitempty"`
	Duration  time.Duration          `json:"duration"`
	Affected  int64                  `json:"affected"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// NewDatabaseEvent creates a new database event
func NewDatabaseEvent(name, table, operation string) *DatabaseEvent {
	return &DatabaseEvent{
		BaseEvent: NewBaseEvent(name, nil),
		Table:     table,
		Operation: operation,
		Data:      make(map[string]interface{}),
	}
}

// CacheEvent represents cache operation events
type CacheEvent struct {
	*BaseEvent
	Store    string        `json:"store"`
	Key      string        `json:"key"`
	Hit      bool          `json:"hit"`
	Duration time.Duration `json:"duration"`
	Size     int64         `json:"size,omitempty"`
}

// NewCacheEvent creates a new cache event
func NewCacheEvent(name, store, key string, hit bool) *CacheEvent {
	return &CacheEvent{
		BaseEvent: NewBaseEvent(name, nil),
		Store:     store,
		Key:       key,
		Hit:       hit,
	}
}

// EventBuilder provides a fluent interface for building events
type EventBuilder struct {
	event *BaseEvent
}

// NewEventBuilder creates a new event builder
func NewEventBuilder(name string) *EventBuilder {
	return &EventBuilder{
		event: NewBaseEvent(name, nil),
	}
}

// WithPayload sets the event payload
func (eb *EventBuilder) WithPayload(payload interface{}) *EventBuilder {
	eb.event.payload = payload
	return eb
}

// WithContext sets the event context
func (eb *EventBuilder) WithContext(ctx context.Context) *EventBuilder {
	eb.event.context = ctx
	return eb
}

// WithMetadata sets event metadata
func (eb *EventBuilder) WithMetadata(key string, value interface{}) *EventBuilder {
	eb.event.SetMetadata(key, value)
	return eb
}

// WithPriority sets the event priority
func (eb *EventBuilder) WithPriority(priority Priority) *EventBuilder {
	eb.event.SetPriority(priority)
	return eb
}

// Build returns the constructed event
func (eb *EventBuilder) Build() Event {
	return eb.event
}

// Event name constants for common events
const (
	// Application events
	EventApplicationStarted  = "app.started"
	EventApplicationStopped  = "app.stopped"
	EventApplicationError    = "app.error"
	
	// Request events
	EventRequestReceived   = "request.received"
	EventRequestCompleted  = "request.completed"
	EventRequestFailed     = "request.failed"
	
	// User events
	EventUserRegistered   = "user.registered"
	EventUserLoggedIn     = "user.logged_in"
	EventUserLoggedOut    = "user.logged_out"
	EventUserUpdated      = "user.updated"
	EventUserDeleted      = "user.deleted"
	
	// Database events
	EventDatabaseConnected    = "database.connected"
	EventDatabaseDisconnected = "database.disconnected"
	EventDatabaseQuery        = "database.query"
	EventDatabaseError        = "database.error"
	
	// Cache events
	EventCacheHit    = "cache.hit"
	EventCacheMiss   = "cache.miss"
	EventCacheWrite  = "cache.write"
	EventCacheDelete = "cache.delete"
	EventCacheFlush  = "cache.flush"
	
	// Queue events
	EventJobQueued    = "job.queued"
	EventJobStarted   = "job.started"
	EventJobCompleted = "job.completed"
	EventJobFailed    = "job.failed"
)