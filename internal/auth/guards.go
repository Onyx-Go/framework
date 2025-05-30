package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"
)

// DefaultSessionGuard implements session-based authentication
type DefaultSessionGuard struct {
	name         string
	lastAttempted User
	user         User
	provider     UserProvider
	session      SessionManager
	request      RequestContext
	loggedOut    bool
	container    Container
	mutex        sync.RWMutex
}

// NewSessionGuard creates a new session guard
func NewSessionGuard(name string, provider UserProvider, container Container) *DefaultSessionGuard {
	return &DefaultSessionGuard{
		name:      name,
		provider:  provider,
		container: container,
	}
}

// Check returns true if the user is authenticated
func (sg *DefaultSessionGuard) Check(ctx context.Context) bool {
	return sg.User(ctx) != nil
}

// Guest returns true if the user is not authenticated
func (sg *DefaultSessionGuard) Guest(ctx context.Context) bool {
	return !sg.Check(ctx)
}

// User returns the authenticated user
func (sg *DefaultSessionGuard) User(ctx context.Context) User {
	sg.mutex.Lock()
	defer sg.mutex.Unlock()

	if sg.loggedOut {
		return nil
	}

	if sg.user != nil {
		return sg.user
	}

	if sg.session != nil {
		id := sg.session.Get(sg.getName())
		if id != nil {
			if user, err := sg.provider.RetrieveById(ctx, id); err == nil {
				sg.user = user
			}
		}

		rememberToken := sg.getRememberToken()
		if sg.user == nil && rememberToken != "" {
			if user, err := sg.provider.RetrieveByToken(ctx, sg.getRecallerId(), rememberToken); err == nil {
				sg.user = user
				sg.updateSession(user.GetID())
			}
		}
	}

	return sg.user
}

// ID returns the user ID if authenticated
func (sg *DefaultSessionGuard) ID(ctx context.Context) interface{} {
	sg.mutex.RLock()
	defer sg.mutex.RUnlock()

	if sg.loggedOut {
		return nil
	}

	if sg.user != nil {
		return sg.user.GetID()
	}

	if sg.session != nil {
		return sg.session.Get(sg.getName())
	}

	return nil
}

// Validate validates credentials without logging in
func (sg *DefaultSessionGuard) Validate(ctx context.Context, credentials map[string]interface{}) bool {
	user, err := sg.provider.RetrieveByCredentials(ctx, credentials)
	if err != nil {
		return false
	}

	sg.lastAttempted = user
	return sg.provider.ValidateCredentials(ctx, user, credentials)
}

// Attempt attempts to log in with credentials
func (sg *DefaultSessionGuard) Attempt(ctx context.Context, credentials map[string]interface{}, remember bool) bool {
	user, err := sg.provider.RetrieveByCredentials(ctx, credentials)
	if err != nil {
		return false
	}

	sg.mutex.Lock()
	sg.lastAttempted = user
	sg.mutex.Unlock()

	if sg.provider.ValidateCredentials(ctx, user, credentials) {
		sg.Login(ctx, user, remember)
		return true
	}

	return false
}

// Login logs in a user
func (sg *DefaultSessionGuard) Login(ctx context.Context, user User, remember bool) error {
	sg.mutex.Lock()
	defer sg.mutex.Unlock()

	sg.updateSession(user.GetID())

	if remember {
		sg.createRememberToken(ctx, user)
	}

	sg.user = user
	sg.loggedOut = false

	return nil
}

// LoginUsingID logs in a user by ID
func (sg *DefaultSessionGuard) LoginUsingID(ctx context.Context, id interface{}, remember bool) error {
	user, err := sg.provider.RetrieveById(ctx, id)
	if err != nil {
		return err
	}

	return sg.Login(ctx, user, remember)
}

// Once logs in for a single request
func (sg *DefaultSessionGuard) Once(ctx context.Context, credentials map[string]interface{}) bool {
	if sg.Validate(ctx, credentials) {
		sg.mutex.Lock()
		sg.user = sg.lastAttempted
		sg.mutex.Unlock()
		return true
	}

	return false
}

// Logout logs out the user
func (sg *DefaultSessionGuard) Logout(ctx context.Context) error {
	sg.mutex.Lock()
	defer sg.mutex.Unlock()

	user := sg.user

	sg.clearUserDataFromStorage()

	if user != nil && user.GetRememberToken() != "" {
		sg.cycleRememberToken(ctx, user)
	}

	sg.user = nil
	sg.loggedOut = true

	return nil
}

// SetUser sets the authenticated user
func (sg *DefaultSessionGuard) SetUser(user User) {
	sg.mutex.Lock()
	defer sg.mutex.Unlock()

	sg.user = user
	sg.loggedOut = false
}

// SetSession sets the session manager
func (sg *DefaultSessionGuard) SetSession(session SessionManager) {
	sg.mutex.Lock()
	defer sg.mutex.Unlock()

	sg.session = session
}

// SetRequest sets the request context
func (sg *DefaultSessionGuard) SetRequest(request RequestContext) {
	sg.mutex.Lock()
	defer sg.mutex.Unlock()

	sg.request = request
	if request != nil {
		if session := request.Session(); session != nil {
			sg.session = session
		}
	}
}

// Private methods

func (sg *DefaultSessionGuard) getName() string {
	return "login_" + sg.name
}

func (sg *DefaultSessionGuard) getRecallerName() string {
	return "remember_" + sg.name
}

func (sg *DefaultSessionGuard) getRecallerId() interface{} {
	if sg.session != nil {
		return sg.session.Get(sg.getRecallerName())
	}
	return nil
}

func (sg *DefaultSessionGuard) getRememberToken() string {
	if sg.session != nil {
		if token := sg.session.Get(sg.getRecallerName()); token != nil {
			if str, ok := token.(string); ok {
				return str
			}
		}
	}
	return ""
}

func (sg *DefaultSessionGuard) updateSession(id interface{}) {
	if sg.session != nil {
		sg.session.Put(sg.getName(), id)
		sg.session.Regenerate()
	}
}

func (sg *DefaultSessionGuard) createRememberToken(ctx context.Context, user User) {
	token := sg.generateRememberToken()
	user.SetRememberToken(token)
	sg.provider.UpdateRememberToken(ctx, user, token)
	if sg.session != nil {
		sg.session.Put(sg.getRecallerName(), user.GetID())
	}
}

func (sg *DefaultSessionGuard) generateRememberToken() string {
	bytes := make([]byte, 60)
	rand.Read(bytes)
	return base64.StdEncoding.EncodeToString(bytes)
}

func (sg *DefaultSessionGuard) cycleRememberToken(ctx context.Context, user User) {
	token := sg.generateRememberToken()
	user.SetRememberToken(token)
	sg.provider.UpdateRememberToken(ctx, user, token)
}

func (sg *DefaultSessionGuard) clearUserDataFromStorage() {
	if sg.session != nil {
		sg.session.Remove(sg.getName())
		sg.session.Remove(sg.getRecallerName())
	}
}

// DefaultTokenGuard implements token-based authentication
type DefaultTokenGuard struct {
	name         string
	lastAttempted User
	user         User
	provider     UserProvider
	request      RequestContext
	container    Container
	mutex        sync.RWMutex
}

// NewTokenGuard creates a new token guard
func NewTokenGuard(name string, provider UserProvider, container Container) *DefaultTokenGuard {
	return &DefaultTokenGuard{
		name:      name,
		provider:  provider,
		container: container,
	}
}

// Check returns true if the user is authenticated
func (tg *DefaultTokenGuard) Check(ctx context.Context) bool {
	return tg.User(ctx) != nil
}

// Guest returns true if the user is not authenticated
func (tg *DefaultTokenGuard) Guest(ctx context.Context) bool {
	return !tg.Check(ctx)
}

// User returns the authenticated user
func (tg *DefaultTokenGuard) User(ctx context.Context) User {
	tg.mutex.Lock()
	defer tg.mutex.Unlock()

	if tg.user != nil {
		return tg.user
	}

	// Get token from request (typically from Authorization header)
	token := tg.getTokenFromRequest()
	if token == "" {
		return nil
	}

	// Retrieve user by token
	if user, err := tg.provider.RetrieveByToken(ctx, nil, token); err == nil {
		tg.user = user
		return user
	}

	return nil
}

// ID returns the user ID if authenticated
func (tg *DefaultTokenGuard) ID(ctx context.Context) interface{} {
	user := tg.User(ctx)
	if user != nil {
		return user.GetID()
	}
	return nil
}

// Validate validates credentials without logging in
func (tg *DefaultTokenGuard) Validate(ctx context.Context, credentials map[string]interface{}) bool {
	user, err := tg.provider.RetrieveByCredentials(ctx, credentials)
	if err != nil {
		return false
	}

	tg.mutex.Lock()
	tg.lastAttempted = user
	tg.mutex.Unlock()

	return tg.provider.ValidateCredentials(ctx, user, credentials)
}

// Attempt attempts to log in with credentials
func (tg *DefaultTokenGuard) Attempt(ctx context.Context, credentials map[string]interface{}, remember bool) bool {
	user, err := tg.provider.RetrieveByCredentials(ctx, credentials)
	if err != nil {
		return false
	}

	tg.mutex.Lock()
	tg.lastAttempted = user
	tg.mutex.Unlock()

	if tg.provider.ValidateCredentials(ctx, user, credentials) {
		tg.Login(ctx, user, remember)
		return true
	}

	return false
}

// Login logs in a user (for token guard, this sets the user)
func (tg *DefaultTokenGuard) Login(ctx context.Context, user User, remember bool) error {
	tg.mutex.Lock()
	defer tg.mutex.Unlock()

	tg.user = user
	return nil
}

// LoginUsingID logs in a user by ID
func (tg *DefaultTokenGuard) LoginUsingID(ctx context.Context, id interface{}, remember bool) error {
	user, err := tg.provider.RetrieveById(ctx, id)
	if err != nil {
		return err
	}

	return tg.Login(ctx, user, remember)
}

// Once logs in for a single request
func (tg *DefaultTokenGuard) Once(ctx context.Context, credentials map[string]interface{}) bool {
	if tg.Validate(ctx, credentials) {
		tg.mutex.Lock()
		tg.user = tg.lastAttempted
		tg.mutex.Unlock()
		return true
	}

	return false
}

// Logout logs out the user (for token guard, this clears the user)
func (tg *DefaultTokenGuard) Logout(ctx context.Context) error {
	tg.mutex.Lock()
	defer tg.mutex.Unlock()

	tg.user = nil
	return nil
}

// SetUser sets the authenticated user
func (tg *DefaultTokenGuard) SetUser(user User) {
	tg.mutex.Lock()
	defer tg.mutex.Unlock()

	tg.user = user
}

// SetSession sets the session manager (not used for token guard)
func (tg *DefaultTokenGuard) SetSession(session SessionManager) {
	// Token guards don't use sessions
}

// SetRequest sets the request context
func (tg *DefaultTokenGuard) SetRequest(request RequestContext) {
	tg.mutex.Lock()
	defer tg.mutex.Unlock()

	tg.request = request
}

// Private methods

func (tg *DefaultTokenGuard) getTokenFromRequest() string {
	if tg.request == nil {
		return ""
	}

	// Try Authorization header first
	authHeader := tg.request.Header("Authorization")
	if authHeader != "" {
		// Extract bearer token
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			return authHeader[7:]
		}
	}

	// Try api_token query parameter
	// This would need to be implemented based on the specific request interface

	return ""
}

// GuardFactory creates guards based on configuration
type GuardFactory struct {
	container Container
}

// NewGuardFactory creates a new guard factory
func NewGuardFactory(container Container) *GuardFactory {
	return &GuardFactory{
		container: container,
	}
}

// CreateGuard creates a guard based on type and configuration
func (gf *GuardFactory) CreateGuard(guardType GuardType, name string, provider UserProvider, config GuardConfig) (Guard, error) {
	switch guardType {
	case GuardTypeSession:
		return NewSessionGuard(name, provider, gf.container), nil
	case GuardTypeToken:
		return NewTokenGuard(name, provider, gf.container), nil
	default:
		return nil, fmt.Errorf("unsupported guard type: %s", guardType)
	}
}

// GuardManager manages multiple guards
type GuardManager struct {
	guards   map[string]Guard
	factory  *GuardFactory
	mutex    sync.RWMutex
}

// NewGuardManager creates a new guard manager
func NewGuardManager(factory *GuardFactory) *GuardManager {
	return &GuardManager{
		guards:  make(map[string]Guard),
		factory: factory,
	}
}

// Register registers a guard
func (gm *GuardManager) Register(name string, guard Guard) {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()

	gm.guards[name] = guard
}

// Get retrieves a guard
func (gm *GuardManager) Get(name string) (Guard, bool) {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()

	guard, exists := gm.guards[name]
	return guard, exists
}

// Create creates and registers a new guard
func (gm *GuardManager) Create(guardType GuardType, name string, provider UserProvider, config GuardConfig) (Guard, error) {
	guard, err := gm.factory.CreateGuard(guardType, name, provider, config)
	if err != nil {
		return nil, err
	}

	gm.Register(name, guard)
	return guard, nil
}

// Remove removes a guard
func (gm *GuardManager) Remove(name string) {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()

	delete(gm.guards, name)
}

// List returns all registered guard names
func (gm *GuardManager) List() []string {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()

	names := make([]string, 0, len(gm.guards))
	for name := range gm.guards {
		names = append(names, name)
	}
	return names
}

// Count returns the number of registered guards
func (gm *GuardManager) Count() int {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()

	return len(gm.guards)
}