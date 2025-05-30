package events

import (
	"context"
	"sync"
)

// Event represents a generic event with context support
type Event interface {
	GetName() string
	GetPayload() interface{}
	GetMetadata() map[string]interface{}
	WithContext(ctx context.Context) Event
	GetContext() context.Context
}

// Listener handles events with context support
type Listener interface {
	Handle(ctx context.Context, event Event) error
}

// ListenerFunc is a function that implements the Listener interface
type ListenerFunc func(ctx context.Context, event Event) error

// Handle implements the Listener interface
func (lf ListenerFunc) Handle(ctx context.Context, event Event) error {
	return lf(ctx, event)
}

// Dispatcher manages event dispatching with context support
type Dispatcher interface {
	// Event registration
	Listen(eventName string, listener Listener)
	ListenFunc(eventName string, listenerFunc ListenerFunc)
	
	// Event dispatching
	Dispatch(ctx context.Context, event Event) error
	DispatchUntil(ctx context.Context, event Event, halt func(interface{}) bool) (interface{}, error)
	
	// Delayed event handling
	Push(eventName string, payload interface{})
	Flush(ctx context.Context, eventName string) error
	
	// Subscriber registration
	Subscribe(subscriber Subscriber)
	
	// Management
	Forget(eventName string)
	ForgetPushed(eventName string)
	HasListeners(eventName string) bool
	GetListeners(eventName string) []Listener
	
	// Wildcard support
	SupportsWildcards() bool
}

// Subscriber can register multiple event listeners
type Subscriber interface {
	Subscribe(dispatcher Dispatcher)
}

// Repository manages multiple event dispatchers
type Repository interface {
	// Dispatcher management
	GetDispatcher(name ...string) Dispatcher
	RegisterDispatcher(name string, dispatcher Dispatcher)
	SetDefaultDispatcher(name string)
	GetDefaultDispatcher() string
	
	// Global event operations
	Dispatch(ctx context.Context, event Event) error
	Listen(eventName string, listener Listener)
	ListenFunc(eventName string, listenerFunc ListenerFunc)
	
	// Lifecycle
	Close() error
}

// Config represents event system configuration
type Config struct {
	DefaultDispatcher string                 `json:"default_dispatcher"`
	Dispatchers       map[string]interface{} `json:"dispatchers"`
	EnableMetrics     bool                   `json:"enable_metrics"`
	QueueConfig       QueueConfig            `json:"queue"`
}

// QueueConfig represents queue configuration for delayed events
type QueueConfig struct {
	Enabled    bool   `json:"enabled"`
	Connection string `json:"connection"`
	Queue      string `json:"queue"`
	MaxRetries int    `json:"max_retries"`
}

// Observer interface for the observer pattern
type Observer interface {
	Update(ctx context.Context, subject interface{}, event string, data interface{}) error
}

// Observable manages observers
type Observable interface {
	Attach(observer Observer)
	Detach(observer Observer)
	Notify(ctx context.Context, subject interface{}, event string, data interface{}) error
}

// Metrics interface for event metrics collection
type Metrics interface {
	RecordEvent(eventName string, duration int64)
	RecordListener(listenerName string, duration int64, success bool)
	RecordError(eventName string, err error)
	GetStats() map[string]interface{}
}

// Manager provides high-level event management
type Manager interface {
	// Configuration
	Configure(config Config) error
	
	// Dispatcher access
	GetRepository() Repository
	
	// Convenience methods
	Fire(ctx context.Context, eventName string, payload interface{}) error
	Listen(eventName string, listener Listener)
	Subscribe(subscriber Subscriber)
	
	// Lifecycle
	Start() error
	Stop() error
}

// Middleware for HTTP requests
type Middleware interface {
	Handle(ctx context.Context, event Event, next func(ctx context.Context, event Event) error) error
}

// EventContext provides additional context for events
type EventContext struct {
	RequestID  string                 `json:"request_id,omitempty"`
	UserID     string                 `json:"user_id,omitempty"`
	SessionID  string                 `json:"session_id,omitempty"`
	Timestamp  int64                  `json:"timestamp"`
	Source     string                 `json:"source,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Context    context.Context        `json:"-"`
	Mutex      sync.RWMutex           `json:"-"`
}

// Priority represents event priority levels
type Priority int

const (
	PriorityLow Priority = iota
	PriorityNormal
	PriorityHigh
	PriorityCritical
)

// String returns the string representation of priority
func (p Priority) String() string {
	switch p {
	case PriorityLow:
		return "low"
	case PriorityNormal:
		return "normal"
	case PriorityHigh:
		return "high"
	case PriorityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// PriorityQueue manages events by priority
type PriorityQueue interface {
	Push(priority Priority, event Event)
	Pop() (Event, bool)
	Len() int
	Clear()
}

// EventFilterFunc defines event filtering function
type EventFilterFunc func(ctx context.Context, event Event) bool

// Filter allows filtering events before dispatch
type Filter interface {
	Filter(ctx context.Context, event Event) bool
}

// Transformer allows modifying events before dispatch
type Transformer interface {
	Transform(ctx context.Context, event Event) (Event, error)
}

// Pipeline processes events through a series of transformers and filters
type Pipeline interface {
	AddFilter(filter Filter)
	AddTransformer(transformer Transformer)
	Process(ctx context.Context, event Event) (Event, error)
}

// EventStore persists events for replay or audit
type EventStore interface {
	Store(ctx context.Context, event Event) error
	Retrieve(ctx context.Context, criteria map[string]interface{}) ([]Event, error)
	Count(ctx context.Context, criteria map[string]interface{}) (int64, error)
	Clear(ctx context.Context) error
}

// Serializer handles event serialization
type Serializer interface {
	Serialize(event Event) ([]byte, error)
	Deserialize(data []byte) (Event, error)
	GetContentType() string
}

// Async handles asynchronous event processing
type Async interface {
	DispatchAsync(ctx context.Context, event Event) error
	ProcessQueue(ctx context.Context) error
	GetQueueSize() int
}

// Validator validates events before processing
type Validator interface {
	Validate(ctx context.Context, event Event) error
}

// ErrorHandler handles event processing errors
type ErrorHandler interface {
	Handle(ctx context.Context, event Event, err error) error
}