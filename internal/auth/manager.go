package auth

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DefaultManager implements the Manager interface
type DefaultManager struct {
	guards          map[string]Guard
	providers       map[string]UserProvider
	defaultGuard    string
	defaultProvider string
	config          *AuthConfig
	container       Container
	mutex           sync.RWMutex
}

// NewDefaultManager creates a new authentication manager
func NewDefaultManager(container Container, config *AuthConfig) *DefaultManager {
	if config == nil {
		config = DefaultAuthConfig()
	}

	return &DefaultManager{
		guards:          make(map[string]Guard),
		providers:       make(map[string]UserProvider),
		defaultGuard:    config.DefaultGuard,
		defaultProvider: config.DefaultProvider,
		config:          config,
		container:       container,
	}
}

// Guard returns a guard instance
func (m *DefaultManager) Guard(ctx context.Context, name ...string) Guard {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	guardName := m.defaultGuard
	if len(name) > 0 {
		guardName = name[0]
	}

	if guard, exists := m.guards[guardName]; exists {
		return guard
	}

	// Create guard if it doesn't exist
	guard, err := m.createGuard(ctx, guardName)
	if err != nil {
		return nil
	}

	m.guards[guardName] = guard
	return guard
}

// Provider returns a user provider instance
func (m *DefaultManager) Provider(ctx context.Context, name ...string) UserProvider {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	providerName := m.defaultProvider
	if len(name) > 0 {
		providerName = name[0]
	}

	if provider, exists := m.providers[providerName]; exists {
		return provider
	}

	// Create provider if it doesn't exist
	provider, err := m.createProvider(ctx, providerName)
	if err != nil {
		return nil
	}

	m.providers[providerName] = provider
	return provider
}

// RegisterGuard registers a guard instance
func (m *DefaultManager) RegisterGuard(name string, guard Guard) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.guards[name] = guard
}

// RegisterProvider registers a user provider instance
func (m *DefaultManager) RegisterProvider(name string, provider UserProvider) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.providers[name] = provider
}

// SetDefaultGuard sets the default guard name
func (m *DefaultManager) SetDefaultGuard(name string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.defaultGuard = name
}

// SetDefaultProvider sets the default provider name
func (m *DefaultManager) SetDefaultProvider(name string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.defaultProvider = name
}

// GetDefaultGuard returns the default guard name
func (m *DefaultManager) GetDefaultGuard() string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.defaultGuard
}

// GetDefaultProvider returns the default provider name
func (m *DefaultManager) GetDefaultProvider() string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.defaultProvider
}

// CreateGuard creates a new guard instance
func (m *DefaultManager) CreateGuard(ctx context.Context, name string, config GuardConfig) (Guard, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	guard, err := m.createGuardFromConfig(ctx, name, config)
	if err != nil {
		return nil, err
	}

	m.guards[name] = guard
	return guard, nil
}

// CreateProvider creates a new provider instance
func (m *DefaultManager) CreateProvider(ctx context.Context, name string, config ProviderConfig) (UserProvider, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	provider, err := m.createProviderFromConfig(ctx, name, config)
	if err != nil {
		return nil, err
	}

	m.providers[name] = provider
	return provider, nil
}

// Private methods

func (m *DefaultManager) createGuard(ctx context.Context, name string) (Guard, error) {
	config, exists := m.config.Guards[name]
	if !exists {
		return nil, fmt.Errorf("guard configuration not found: %s", name)
	}

	return m.createGuardFromConfig(ctx, name, config)
}

func (m *DefaultManager) createProvider(ctx context.Context, name string) (UserProvider, error) {
	config, exists := m.config.Providers[name]
	if !exists {
		return nil, fmt.Errorf("provider configuration not found: %s", name)
	}

	return m.createProviderFromConfig(ctx, name, config)
}

func (m *DefaultManager) createGuardFromConfig(ctx context.Context, name string, config GuardConfig) (Guard, error) {
	switch GuardType(config.Type) {
	case GuardTypeSession:
		provider := m.Provider(ctx, config.Provider)
		if provider == nil {
			return nil, fmt.Errorf("provider not found for guard %s: %s", name, config.Provider)
		}
		return NewSessionGuard(name, provider, m.container), nil

	case GuardTypeToken:
		provider := m.Provider(ctx, config.Provider)
		if provider == nil {
			return nil, fmt.Errorf("provider not found for guard %s: %s", name, config.Provider)
		}
		return NewTokenGuard(name, provider, m.container), nil

	default:
		return nil, fmt.Errorf("unsupported guard type: %s", config.Type)
	}
}

func (m *DefaultManager) createProviderFromConfig(ctx context.Context, name string, config ProviderConfig) (UserProvider, error) {
	switch ProviderType(config.Type) {
	case ProviderTypeDatabase:
		db, err := m.container.Make("database")
		if err != nil {
			return nil, fmt.Errorf("database not available for provider %s: %v", name, err)
		}

		database, ok := db.(Database)
		if !ok {
			return nil, fmt.Errorf("invalid database type for provider %s", name)
		}

		hasher := NewBcryptHasher()
		return NewDatabaseUserProvider(database, config.Table, hasher), nil

	default:
		return nil, fmt.Errorf("unsupported provider type: %s", config.Type)
	}
}

// DefaultAuthConfig returns default authentication configuration
func DefaultAuthConfig() *AuthConfig {
	return &AuthConfig{
		DefaultGuard:    "web",
		DefaultProvider: "users",
		Guards: map[string]GuardConfig{
			"web": {
				Type:     string(GuardTypeSession),
				Provider: "users",
				Options:  make(map[string]interface{}),
			},
			"api": {
				Type:     string(GuardTypeToken),
				Provider: "users",
				Options:  make(map[string]interface{}),
			},
		},
		Providers: map[string]ProviderConfig{
			"users": {
				Type:    string(ProviderTypeDatabase),
				Table:   "users",
				Options: make(map[string]interface{}),
			},
		},
		Passwords: PasswordConfig{
			Default: "bcrypt",
			Hashers: map[string]HasherConfig{
				"bcrypt": {
					Driver: string(HasherTypeBcrypt),
					Rounds: 10,
				},
			},
		},
		Session: SessionConfig{
			Driver:        "file",
			Lifetime:      120 * 60, // 2 hours in seconds
			ExpireOnClose: false,
			Encrypt:       false,
		},
		RateLimit: RateLimitConfig{
			MaxAttempts:  5,
			DecayMinutes: 1,
		},
		Security: SecurityConfig{
			RememberTokenLength: 60,
			RequireHttps:        false,
			SameSiteCookies:     "lax",
			SecureCookies:       false,
		},
	}
}

// AuthenticationState manages authentication state for a request
type AuthenticationState struct {
	guards    map[string]Guard
	providers map[string]UserProvider
	user      User
	manager   *DefaultManager
	mutex     sync.RWMutex
}

// NewAuthenticationState creates a new authentication state
func NewAuthenticationState(manager *DefaultManager) *AuthenticationState {
	return &AuthenticationState{
		guards:    make(map[string]Guard),
		providers: make(map[string]UserProvider),
		manager:   manager,
	}
}

// SetUser sets the authenticated user
func (as *AuthenticationState) SetUser(user User) {
	as.mutex.Lock()
	defer as.mutex.Unlock()

	as.user = user
}

// GetUser returns the authenticated user
func (as *AuthenticationState) GetUser() User {
	as.mutex.RLock()
	defer as.mutex.RUnlock()

	return as.user
}

// IsAuthenticated checks if a user is authenticated
func (as *AuthenticationState) IsAuthenticated() bool {
	return as.GetUser() != nil
}

// Guard returns a guard for this state
func (as *AuthenticationState) Guard(ctx context.Context, name ...string) Guard {
	return as.manager.Guard(ctx, name...)
}

// Provider returns a provider for this state
func (as *AuthenticationState) Provider(ctx context.Context, name ...string) UserProvider {
	return as.manager.Provider(ctx, name...)
}

// AuthenticationResult represents the result of an authentication attempt
type AuthenticationResult struct {
	Success bool
	User    User
	Error   error
	Message string
}

// NewAuthenticationResult creates a new authentication result
func NewAuthenticationResult(success bool, user User, err error, message string) *AuthenticationResult {
	return &AuthenticationResult{
		Success: success,
		User:    user,
		Error:   err,
		Message: message,
	}
}

// LoginAttempt represents a login attempt
type LoginAttempt struct {
	Credentials Credentials
	Remember    bool
	Guard       string
	Provider    string
	IP          string
	UserAgent   string
	Timestamp   int64
}

// NewLoginAttempt creates a new login attempt
func NewLoginAttempt(credentials Credentials, remember bool, guard, provider, ip, userAgent string) *LoginAttempt {
	return &LoginAttempt{
		Credentials: credentials,
		Remember:    remember,
		Guard:       guard,
		Provider:    provider,
		IP:          ip,
		UserAgent:   userAgent,
		Timestamp:   getCurrentTimestamp(),
	}
}

// AuthenticationAttempt manages authentication attempts
type AuthenticationAttempt struct {
	manager     *DefaultManager
	guard       Guard
	provider    UserProvider
	rateLimiter RateLimiter
	dispatcher  EventDispatcher
}

// NewAuthenticationAttempt creates a new authentication attempt manager
func NewAuthenticationAttempt(manager *DefaultManager, guard Guard, provider UserProvider) *AuthenticationAttempt {
	return &AuthenticationAttempt{
		manager:  manager,
		guard:    guard,
		provider: provider,
	}
}

// SetRateLimiter sets the rate limiter
func (aa *AuthenticationAttempt) SetRateLimiter(rateLimiter RateLimiter) {
	aa.rateLimiter = rateLimiter
}

// SetEventDispatcher sets the event dispatcher
func (aa *AuthenticationAttempt) SetEventDispatcher(dispatcher EventDispatcher) {
	aa.dispatcher = dispatcher
}

// Attempt performs an authentication attempt
func (aa *AuthenticationAttempt) Attempt(ctx context.Context, attempt *LoginAttempt) *AuthenticationResult {
	// Check rate limiting
	if aa.rateLimiter != nil {
		key := aa.buildRateLimitKey(attempt)
		if aa.rateLimiter.TooManyAttempts(ctx, key, 5) {
			return NewAuthenticationResult(false, nil, nil, "Too many login attempts")
		}
	}

	// Dispatch attempt event
	if aa.dispatcher != nil {
		event := &LoginAttemptEvent{
			BaseEvent: BaseEvent{
				EventType: string(EventLoginAttempt),
				EventCtx:  ctx,
				EventData: map[string]interface{}{
					"credentials": attempt.Credentials,
					"ip":          attempt.IP,
					"user_agent":  attempt.UserAgent,
				},
				EventTime: getCurrentTime(),
			},
		}
		aa.dispatcher.Dispatch(ctx, event)
	}

	// Attempt authentication
	success := aa.guard.Attempt(ctx, attempt.Credentials, attempt.Remember)
	if success {
		user := aa.guard.User(ctx)
		
		// Dispatch success event
		if aa.dispatcher != nil {
			event := &LoginEvent{
				BaseEvent: BaseEvent{
					EventType: string(EventLogin),
					EventUser: user,
					EventCtx:  ctx,
					EventData: map[string]interface{}{
						"ip":         attempt.IP,
						"user_agent": attempt.UserAgent,
						"remember":   attempt.Remember,
					},
					EventTime: getCurrentTime(),
				},
			}
			aa.dispatcher.Dispatch(ctx, event)
		}

		return NewAuthenticationResult(true, user, nil, "Login successful")
	}

	// Handle failed attempt
	if aa.rateLimiter != nil {
		key := aa.buildRateLimitKey(attempt)
		aa.rateLimiter.Hit(ctx, key, 1)
	}

	// Dispatch failed event
	if aa.dispatcher != nil {
		event := &FailedLoginEvent{
			BaseEvent: BaseEvent{
				EventType: string(EventLoginFailed),
				EventCtx:  ctx,
				EventData: map[string]interface{}{
					"credentials": attempt.Credentials,
					"ip":          attempt.IP,
					"user_agent":  attempt.UserAgent,
				},
				EventTime: getCurrentTime(),
			},
		}
		aa.dispatcher.Dispatch(ctx, event)
	}

	return NewAuthenticationResult(false, nil, nil, "Invalid credentials")
}

func (aa *AuthenticationAttempt) buildRateLimitKey(attempt *LoginAttempt) string {
	identifier := attempt.Credentials.GetString("email")
	if identifier == "" {
		identifier = attempt.IP
	}
	return fmt.Sprintf("login_attempts:%s", identifier)
}

// Helper functions

func getCurrentTimestamp() int64 {
	return getCurrentTime().Unix()
}

func getCurrentTime() time.Time {
	return time.Now()
}

