package auth

import (
	"context"
	"time"
)

// User represents an authenticated user
type User interface {
	GetID() interface{}
	GetAuthIdentifier() string
	GetAuthPassword() string
	GetRememberToken() string
	SetRememberToken(string)
}

// Authenticatable extends User with additional auth methods
type Authenticatable interface {
	User
	GetAuthIdentifierName() string
	GetAuthPasswordName() string
	GetRememberTokenName() string
}

// UserProvider handles user retrieval and validation
type UserProvider interface {
	RetrieveById(ctx context.Context, identifier interface{}) (User, error)
	RetrieveByToken(ctx context.Context, identifier interface{}, token string) (User, error)
	UpdateRememberToken(ctx context.Context, user User, token string) error
	RetrieveByCredentials(ctx context.Context, credentials map[string]interface{}) (User, error)
	ValidateCredentials(ctx context.Context, user User, credentials map[string]interface{}) bool
}

// Guard manages user authentication state
type Guard interface {
	Check(ctx context.Context) bool
	Guest(ctx context.Context) bool
	User(ctx context.Context) User
	ID(ctx context.Context) interface{}
	Validate(ctx context.Context, credentials map[string]interface{}) bool
	Attempt(ctx context.Context, credentials map[string]interface{}, remember bool) bool
	Login(ctx context.Context, user User, remember bool) error
	LoginUsingID(ctx context.Context, id interface{}, remember bool) error
	Once(ctx context.Context, credentials map[string]interface{}) bool
	Logout(ctx context.Context) error
	SetUser(user User)
	SetSession(session SessionManager)
	SetRequest(request RequestContext)
}

// Manager manages multiple guards and providers
type Manager interface {
	Guard(ctx context.Context, name ...string) Guard
	Provider(ctx context.Context, name ...string) UserProvider
	RegisterGuard(name string, guard Guard)
	RegisterProvider(name string, provider UserProvider)
	SetDefaultGuard(name string)
	SetDefaultProvider(name string)
	GetDefaultGuard() string
	GetDefaultProvider() string
	CreateGuard(ctx context.Context, name string, config GuardConfig) (Guard, error)
	CreateProvider(ctx context.Context, name string, config ProviderConfig) (UserProvider, error)
}

// Hasher handles password hashing
type Hasher interface {
	Make(ctx context.Context, value string) (string, error)
	Check(ctx context.Context, value, hashed string) bool
	NeedsRehash(ctx context.Context, hashed string) bool
	SetRounds(rounds int)
	GetRounds() int
}

// SessionManager handles session operations for authentication
type SessionManager interface {
	Get(key string) interface{}
	Put(key string, value interface{})
	Remove(key string)
	Regenerate() error
	Flash(key string, value interface{})
	GetFlash(key string) interface{}
	Has(key string) bool
	All() map[string]interface{}
	Invalidate() error
	GetID() string
}

// RequestContext provides request-specific context
type RequestContext interface {
	Session() SessionManager
	IP() string
	UserAgent() string
	Header(key string) string
	Cookie(name string) (string, error)
	SetCookie(name, value string, maxAge int, path, domain string, secure, httpOnly bool)
}

// TokenGenerator generates secure tokens
type TokenGenerator interface {
	Generate(ctx context.Context, length int) (string, error)
	GenerateRememberToken(ctx context.Context) (string, error)
	GeneratePasswordResetToken(ctx context.Context) (string, error)
	ValidateToken(ctx context.Context, token string) bool
}

// EventDispatcher handles authentication events
type EventDispatcher interface {
	Dispatch(ctx context.Context, event Event) error
	Listen(eventType string, listener EventListener)
	Subscribe(subscriber EventSubscriber)
}

// EventListener handles specific events
type EventListener func(ctx context.Context, event Event) error

// EventSubscriber subscribes to multiple events
type EventSubscriber interface {
	Subscribe(dispatcher EventDispatcher)
}

// RateLimiter limits authentication attempts
type RateLimiter interface {
	Hit(ctx context.Context, key string, decayMinutes int) error
	TooManyAttempts(ctx context.Context, key string, maxAttempts int) bool
	AvailableIn(ctx context.Context, key string) int
	RetriesLeft(ctx context.Context, key string, maxAttempts int) int
	Clear(ctx context.Context, key string) error
}

// Cache provides caching for authentication data
type Cache interface {
	Get(ctx context.Context, key string) (interface{}, bool)
	Set(ctx context.Context, key string, value interface{}, duration time.Duration) error
	Delete(ctx context.Context, key string) error
	Clear(ctx context.Context) error
	Has(ctx context.Context, key string) bool
}

// Database provides database operations for authentication
type Database interface {
	Query(ctx context.Context, query string, args ...interface{}) (Rows, error)
	QueryRow(ctx context.Context, query string, args ...interface{}) Row
	Exec(ctx context.Context, query string, args ...interface{}) (Result, error)
	Begin(ctx context.Context) (Transaction, error)
	Table(name string) QueryBuilder
}

// QueryBuilder provides query building capabilities
type QueryBuilder interface {
	Where(column, operator string, value interface{}) QueryBuilder
	WhereIn(column string, values []interface{}) QueryBuilder
	WhereNotIn(column string, values []interface{}) QueryBuilder
	WhereNull(column string) QueryBuilder
	WhereNotNull(column string) QueryBuilder
	OrderBy(column, direction string) QueryBuilder
	Limit(limit int) QueryBuilder
	Offset(offset int) QueryBuilder
	First(dest interface{}) error
	Find(dest interface{}) error
	Count() (int64, error)
	Update(values map[string]interface{}) (Result, error)
	Delete() (Result, error)
	Insert(values map[string]interface{}) (Result, error)
}

// Rows represents query result rows
type Rows interface {
	Next() bool
	Scan(dest ...interface{}) error
	Close() error
	Err() error
}

// Row represents a single query result row
type Row interface {
	Scan(dest ...interface{}) error
}

// Result represents query execution result
type Result interface {
	LastInsertId() (int64, error)
	RowsAffected() (int64, error)
}

// Transaction represents a database transaction
type Transaction interface {
	Commit() error
	Rollback() error
	QueryBuilder
}

// Configuration types

// GuardConfig configures a guard
type GuardConfig struct {
	Type     string                 `json:"type"`
	Provider string                 `json:"provider"`
	Options  map[string]interface{} `json:"options"`
}

// ProviderConfig configures a user provider
type ProviderConfig struct {
	Type    string                 `json:"type"`
	Model   string                 `json:"model,omitempty"`
	Table   string                 `json:"table,omitempty"`
	Options map[string]interface{} `json:"options"`
}

// AuthConfig contains authentication configuration
type AuthConfig struct {
	DefaultGuard    string                     `json:"default_guard"`
	DefaultProvider string                     `json:"default_provider"`
	Guards          map[string]GuardConfig     `json:"guards"`
	Providers       map[string]ProviderConfig  `json:"providers"`
	Passwords       PasswordConfig             `json:"passwords"`
	Session         SessionConfig              `json:"session"`
	RateLimit       RateLimitConfig            `json:"rate_limit"`
	Security        SecurityConfig             `json:"security"`
}

// PasswordConfig configures password handling
type PasswordConfig struct {
	Default string                 `json:"default"`
	Hashers map[string]HasherConfig `json:"hashers"`
}

// HasherConfig configures password hashing
type HasherConfig struct {
	Driver  string `json:"driver"`
	Rounds  int    `json:"rounds,omitempty"`
	Memory  int    `json:"memory,omitempty"`
	Time    int    `json:"time,omitempty"`
	Threads int    `json:"threads,omitempty"`
}

// SessionConfig configures session handling
type SessionConfig struct {
	Driver     string        `json:"driver"`
	Lifetime   time.Duration `json:"lifetime"`
	ExpireOnClose bool       `json:"expire_on_close"`
	Encrypt    bool          `json:"encrypt"`
	Files      string        `json:"files,omitempty"`
	Connection string        `json:"connection,omitempty"`
	Table      string        `json:"table,omitempty"`
	Store      string        `json:"store,omitempty"`
}

// RateLimitConfig configures rate limiting
type RateLimitConfig struct {
	MaxAttempts   int           `json:"max_attempts"`
	DecayMinutes  int           `json:"decay_minutes"`
	LockoutTime   time.Duration `json:"lockout_time"`
	ThrottleKey   string        `json:"throttle_key"`
}

// SecurityConfig configures security settings
type SecurityConfig struct {
	RememberTokenLength int           `json:"remember_token_length"`
	SessionTimeout      time.Duration `json:"session_timeout"`
	MaxSessions         int           `json:"max_sessions"`
	RequireHttps        bool          `json:"require_https"`
	SameSiteCookies     string        `json:"same_site_cookies"`
	SecureCookies       bool          `json:"secure_cookies"`
}

// Event types and structures

// Event represents an authentication event
type Event interface {
	Type() string
	User() User
	Context() context.Context
	Data() map[string]interface{}
	Timestamp() time.Time
}

// BaseEvent provides common event functionality
type BaseEvent struct {
	EventType string
	EventUser User
	EventCtx  context.Context
	EventData map[string]interface{}
	EventTime time.Time
}

func (e *BaseEvent) Type() string { return e.EventType }
func (e *BaseEvent) User() User { return e.EventUser }
func (e *BaseEvent) Context() context.Context { return e.EventCtx }
func (e *BaseEvent) Data() map[string]interface{} { return e.EventData }
func (e *BaseEvent) Timestamp() time.Time { return e.EventTime }

// Specific event types
type LoginEvent struct{ BaseEvent }
type LogoutEvent struct{ BaseEvent }
type LoginAttemptEvent struct{ BaseEvent }
type FailedLoginEvent struct{ BaseEvent }
type PasswordResetEvent struct{ BaseEvent }
type UserRegisteredEvent struct{ BaseEvent }
type UserVerifiedEvent struct{ BaseEvent }

// Helper types and constants

// AuthEventType represents event type constants
type AuthEventType string

const (
	EventLogin        AuthEventType = "auth.login"
	EventLogout       AuthEventType = "auth.logout"
	EventLoginAttempt AuthEventType = "auth.login_attempt"
	EventLoginFailed  AuthEventType = "auth.login_failed"
	EventPasswordReset AuthEventType = "auth.password_reset"
	EventUserRegistered AuthEventType = "auth.user_registered"
	EventUserVerified AuthEventType = "auth.user_verified"
)

// GuardType represents guard type constants
type GuardType string

const (
	GuardTypeSession GuardType = "session"
	GuardTypeToken   GuardType = "token"
	GuardTypeJWT     GuardType = "jwt"
	GuardTypeAPI     GuardType = "api"
)

// ProviderType represents provider type constants
type ProviderType string

const (
	ProviderTypeDatabase ProviderType = "database"
	ProviderTypeEloquent ProviderType = "eloquent"
	ProviderTypeLDAP     ProviderType = "ldap"
	ProviderTypeCustom   ProviderType = "custom"
)

// HasherType represents hasher type constants
type HasherType string

const (
	HasherTypeBcrypt  HasherType = "bcrypt"
	HasherTypeArgon2i HasherType = "argon2i"
	HasherTypeArgon2d HasherType = "argon2id"
	HasherTypeScrypt  HasherType = "scrypt"
)

// Middleware types and helpers

// MiddlewareFunc represents authentication middleware
type MiddlewareFunc func(next Handler) Handler

// Handler represents a request handler
type Handler interface {
	Handle(ctx context.Context, req RequestContext) error
}

// HandlerFunc adapter
type HandlerFunc func(ctx context.Context, req RequestContext) error

func (f HandlerFunc) Handle(ctx context.Context, req RequestContext) error {
	return f(ctx, req)
}

// AuthMiddlewareConfig configures authentication middleware
type AuthMiddlewareConfig struct {
	Guards     []string `json:"guards"`
	Optional   bool     `json:"optional"`
	Redirect   string   `json:"redirect,omitempty"`
	JsonError  bool     `json:"json_error"`
	StatusCode int      `json:"status_code"`
}

// Utility functions for working with authentication data

// Credentials represents user credentials
type Credentials map[string]interface{}

func (c Credentials) Get(key string) interface{} {
	return c[key]
}

func (c Credentials) GetString(key string) string {
	if v, ok := c[key].(string); ok {
		return v
	}
	return ""
}

func (c Credentials) Has(key string) bool {
	_, exists := c[key]
	return exists
}

func (c Credentials) Set(key string, value interface{}) {
	c[key] = value
}

// UserData represents additional user data
type UserData map[string]interface{}

func (ud UserData) Get(key string) interface{} {
	return ud[key]
}

func (ud UserData) GetString(key string) string {
	if v, ok := ud[key].(string); ok {
		return v
	}
	return ""
}

func (ud UserData) GetInt(key string) int {
	if v, ok := ud[key].(int); ok {
		return v
	}
	return 0
}

func (ud UserData) GetBool(key string) bool {
	if v, ok := ud[key].(bool); ok {
		return v
	}
	return false
}

func (ud UserData) Has(key string) bool {
	_, exists := ud[key]
	return exists
}

func (ud UserData) Set(key string, value interface{}) {
	ud[key] = value
}