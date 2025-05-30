package session

import (
	"context"
	"net/http"
	"time"
)

// Session represents a user session with comprehensive session management capabilities
type Session interface {
	// Basic session operations
	ID() string
	Get(key string) interface{}
	Put(key string, value interface{})
	Remove(key string)
	Flush()
	Has(key string) bool
	All() map[string]interface{}
	
	// Flash messaging
	Flash(key string, value interface{})
	GetFlash(key string) interface{}
	GetAllFlash() map[string]interface{}
	ClearFlash()
	
	// Session lifecycle
	Regenerate() error
	InvalidateSession() error
	Migrate(destroy bool) error
	IsStarted() bool
	Save() error
	
	// Session metadata
	GetCreatedAt() time.Time
	GetLastAccess() time.Time
	GetLifetime() time.Duration
	SetLifetime(duration time.Duration)
	IsExpired() bool
	
	// Context-aware operations
	GetWithContext(ctx context.Context, key string) (interface{}, error)
	PutWithContext(ctx context.Context, key string, value interface{}) error
	RemoveWithContext(ctx context.Context, key string) error
}

// Handler defines the interface for session storage backends
type Handler interface {
	// Connection management
	Open(ctx context.Context) error
	Close(ctx context.Context) error
	
	// Session data operations
	Read(ctx context.Context, sessionID string) ([]byte, error)
	Write(ctx context.Context, sessionID string, data []byte) error
	Destroy(ctx context.Context, sessionID string) error
	Exists(ctx context.Context, sessionID string) (bool, error)
	
	// Maintenance operations
	GC(ctx context.Context, maxLifetime int64) error
	Count(ctx context.Context) (int, error)
	
	// Batch operations
	ReadMultiple(ctx context.Context, sessionIDs []string) (map[string][]byte, error)
	WriteMultiple(ctx context.Context, sessions map[string][]byte) error
	DestroyMultiple(ctx context.Context, sessionIDs []string) error
}

// Manager handles session lifecycle and configuration
type Manager interface {
	// Session lifecycle
	StartSession(ctx context.Context, w http.ResponseWriter, r *http.Request) (Session, error)
	GetSession(ctx context.Context, r *http.Request) (Session, error)
	DestroySession(ctx context.Context, w http.ResponseWriter, r *http.Request) error
	
	// Configuration
	SetCookieName(name string)
	GetCookieName() string
	SetCookiePath(path string)
	GetCookiePath() string
	SetCookieDomain(domain string)
	GetCookieDomain() string
	SetSecure(secure bool)
	IsSecure() bool
	SetHTTPOnly(httpOnly bool)
	IsHTTPOnly() bool
	SetSameSite(sameSite http.SameSite)
	GetSameSite() http.SameSite
	SetLifetime(lifetime time.Duration)
	GetLifetime() time.Duration
	
	// Session management
	RegenerateID(ctx context.Context, session Session) error
	TouchSession(ctx context.Context, session Session) error
	ValidateSession(ctx context.Context, session Session) error
	
	// Cleanup and maintenance
	GarbageCollect(ctx context.Context) error
	GetActiveSessionCount(ctx context.Context) (int, error)
	
	// Handler management
	SetHandler(handler Handler)
	GetHandler() Handler
}

// Store defines the interface for session data serialization
type Store interface {
	// Serialization
	Serialize(ctx context.Context, data map[string]interface{}) ([]byte, error)
	Deserialize(ctx context.Context, data []byte) (map[string]interface{}, error)
	
	// Validation
	Validate(ctx context.Context, data map[string]interface{}) error
	
	// Data transformation
	Transform(ctx context.Context, data map[string]interface{}) (map[string]interface{}, error)
}

// Config holds session configuration options
type Config struct {
	// Cookie configuration
	CookieName   string
	CookiePath   string
	CookieDomain string
	Secure       bool
	HTTPOnly     bool
	SameSite     http.SameSite
	
	// Session configuration
	Lifetime    time.Duration
	IdleTimeout time.Duration
	
	// Handler configuration
	Handler       string
	HandlerConfig map[string]interface{}
	
	// Security configuration
	RegenerateOnLogin  bool
	RegenerateInterval time.Duration
	
	// Cleanup configuration
	GCProbability float64
	GCInterval    time.Duration
}

// DefaultConfig returns a default session configuration
func DefaultConfig() Config {
	return Config{
		CookieName:         "session",
		CookiePath:         "/",
		CookieDomain:       "",
		Secure:             false,
		HTTPOnly:           true,
		SameSite:           http.SameSiteLaxMode,
		Lifetime:           2 * time.Hour,
		IdleTimeout:        30 * time.Minute,
		Handler:            "memory",
		HandlerConfig:      make(map[string]interface{}),
		RegenerateOnLogin:  true,
		RegenerateInterval: 15 * time.Minute,
		GCProbability:      0.1,
		GCInterval:         10 * time.Minute,
	}
}

// MiddlewareFunc defines the session middleware function signature
type MiddlewareFunc func(next http.Handler) http.Handler

// Events defines session event types
type Events struct {
	SessionCreated     string
	SessionDestroyed   string
	SessionRegenerated string
	SessionExpired     string
	SessionSaved       string
}

// DefaultEvents returns default session event names
func DefaultEvents() Events {
	return Events{
		SessionCreated:     "session.created",
		SessionDestroyed:   "session.destroyed",
		SessionRegenerated: "session.regenerated",
		SessionExpired:     "session.expired",
		SessionSaved:       "session.saved",
	}
}

// Statistics holds session statistics
type Statistics struct {
	TotalSessions    int
	ActiveSessions   int
	ExpiredSessions  int
	AverageLifetime  time.Duration
	LastCleanup      time.Time
	CleanupCount     int
}

// Context keys for session data
type ContextKey string

const (
	ContextKeySession     ContextKey = "session"
	ContextKeySessionID   ContextKey = "session_id"
	ContextKeySessionData ContextKey = "session_data"
)