package onyx

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"
)

type User interface {
	GetID() interface{}
	GetAuthIdentifier() string
	GetAuthPassword() string
	GetRememberToken() string
	SetRememberToken(string)
}

type Authenticatable interface {
	User
	GetAuthIdentifierName() string
	GetAuthPasswordName() string
	GetRememberTokenName() string
}

type UserProvider interface {
	RetrieveById(identifier interface{}) (User, error)
	RetrieveByToken(identifier interface{}, token string) (User, error)
	UpdateRememberToken(user User, token string) error
	RetrieveByCredentials(credentials map[string]interface{}) (User, error)
	ValidateCredentials(user User, credentials map[string]interface{}) bool
}

type Guard interface {
	Check() bool
	Guest() bool
	User() User
	ID() interface{}
	Validate(credentials map[string]interface{}) bool
	Attempt(credentials map[string]interface{}, remember bool) bool
	Login(user User, remember bool) error
	LoginUsingID(id interface{}, remember bool) error
	Once(credentials map[string]interface{}) bool
	Logout() error
	SetUser(user User)
}

type AuthManager struct {
	app             *Application
	guards          map[string]Guard
	userProviders   map[string]UserProvider
	defaultGuard    string
	defaultProvider string
}

func NewAuthManager(app *Application) *AuthManager {
	return &AuthManager{
		app:           app,
		guards:        make(map[string]Guard),
		userProviders: make(map[string]UserProvider),
		defaultGuard:  "web",
	}
}

func (am *AuthManager) Guard(name ...string) Guard {
	guardName := am.defaultGuard
	if len(name) > 0 {
		guardName = name[0]
	}
	
	if guard, exists := am.guards[guardName]; exists {
		return guard
	}
	
	return am.createSessionGuard(guardName)
}

func (am *AuthManager) Provider(name ...string) UserProvider {
	providerName := am.defaultProvider
	if len(name) > 0 {
		providerName = name[0]
	}
	
	if provider, exists := am.userProviders[providerName]; exists {
		return provider
	}
	
	return nil
}

func (am *AuthManager) RegisterGuard(name string, guard Guard) {
	am.guards[name] = guard
}

func (am *AuthManager) RegisterProvider(name string, provider UserProvider) {
	am.userProviders[name] = provider
}

func (am *AuthManager) createSessionGuard(name string) Guard {
	provider := am.Provider()
	if provider == nil {
		panic(fmt.Sprintf("User provider not found for guard: %s", name))
	}
	
	guard := NewSessionGuard(name, provider, am.app)
	am.guards[name] = guard
	return guard
}

type SessionGuard struct {
	name         string
	lastAttempted User
	user         User
	provider     UserProvider
	session      Session
	request      *Context
	loggedOut    bool
}

func NewSessionGuard(name string, provider UserProvider, app *Application) *SessionGuard {
	return &SessionGuard{
		name:     name,
		provider: provider,
	}
}

func (sg *SessionGuard) Check() bool {
	return sg.User() != nil
}

func (sg *SessionGuard) Guest() bool {
	return !sg.Check()
}

func (sg *SessionGuard) User() User {
	if sg.loggedOut {
		return nil
	}
	
	if sg.user != nil {
		return sg.user
	}
	
	if sg.session != nil {
		id := sg.session.Get(sg.getName())
		if id != nil {
			if user, err := sg.provider.RetrieveById(id); err == nil {
				sg.user = user
			}
		}
		
		rememberToken := sg.getRememberToken()
		if sg.user == nil && rememberToken != "" {
			if user, err := sg.provider.RetrieveByToken(sg.getRecallerId(), rememberToken); err == nil {
				sg.user = user
				sg.updateSession(user.GetID())
			}
		}
	}
	
	return sg.user
}

func (sg *SessionGuard) ID() interface{} {
	if sg.loggedOut {
		return nil
	}
	
	if sg.user != nil {
		return sg.user.GetID()
	}
	
	return sg.session.Get(sg.getName())
}

func (sg *SessionGuard) Validate(credentials map[string]interface{}) bool {
	user, err := sg.provider.RetrieveByCredentials(credentials)
	if err != nil {
		return false
	}
	
	return sg.provider.ValidateCredentials(user, credentials)
}

func (sg *SessionGuard) Attempt(credentials map[string]interface{}, remember bool) bool {
	user, err := sg.provider.RetrieveByCredentials(credentials)
	if err != nil {
		return false
	}
	
	sg.lastAttempted = user
	
	if sg.provider.ValidateCredentials(user, credentials) {
		sg.Login(user, remember)
		return true
	}
	
	return false
}

func (sg *SessionGuard) Login(user User, remember bool) error {
	sg.updateSession(user.GetID())
	
	if remember {
		sg.createRememberToken(user)
	}
	
	sg.user = user
	sg.loggedOut = false
	
	return nil
}

func (sg *SessionGuard) LoginUsingID(id interface{}, remember bool) error {
	user, err := sg.provider.RetrieveById(id)
	if err != nil {
		return err
	}
	
	return sg.Login(user, remember)
}

func (sg *SessionGuard) Once(credentials map[string]interface{}) bool {
	if sg.Validate(credentials) {
		sg.user = sg.lastAttempted
		return true
	}
	
	return false
}

func (sg *SessionGuard) Logout() error {
	user := sg.User()
	
	sg.clearUserDataFromStorage()
	
	if user != nil && user.GetRememberToken() != "" {
		sg.cycleRememberToken(user)
	}
	
	sg.user = nil
	sg.loggedOut = true
	
	return nil
}

func (sg *SessionGuard) SetUser(user User) {
	sg.user = user
	sg.loggedOut = false
}

func (sg *SessionGuard) SetSession(session Session) {
	sg.session = session
}

func (sg *SessionGuard) SetRequest(request *Context) {
	sg.request = request
	if request != nil {
		if session := request.Session(); session != nil {
			sg.session = session
		}
	}
}

func (sg *SessionGuard) getName() string {
	return "login_" + sg.name
}

func (sg *SessionGuard) getRecallerName() string {
	return "remember_" + sg.name
}

func (sg *SessionGuard) getRecallerId() interface{} {
	if sg.session != nil {
		return sg.session.Get(sg.getRecallerName())
	}
	return nil
}

func (sg *SessionGuard) getRememberToken() string {
	if sg.session != nil {
		if token := sg.session.Get(sg.getRecallerName()); token != nil {
			if str, ok := token.(string); ok {
				return str
			}
		}
	}
	return ""
}

func (sg *SessionGuard) updateSession(id interface{}) {
	if sg.session != nil {
		sg.session.Put(sg.getName(), id)
		sg.session.Regenerate()
	}
}

func (sg *SessionGuard) createRememberToken(user User) {
	token := sg.generateRememberToken()
	user.SetRememberToken(token)
	sg.provider.UpdateRememberToken(user, token)
	if sg.session != nil {
		sg.session.Put(sg.getRecallerName(), user.GetID())
	}
}

func (sg *SessionGuard) generateRememberToken() string {
	bytes := make([]byte, 60)
	rand.Read(bytes)
	return base64.StdEncoding.EncodeToString(bytes)
}

func (sg *SessionGuard) cycleRememberToken(user User) {
	token := sg.generateRememberToken()
	user.SetRememberToken(token)
	sg.provider.UpdateRememberToken(user, token)
}

func (sg *SessionGuard) clearUserDataFromStorage() {
	if sg.session != nil {
		sg.session.Remove(sg.getName())
		sg.session.Remove(sg.getRecallerName())
	}
}

type DatabaseUserProvider struct {
	db        *DB
	table     string
	hasher    Hasher
}

func NewDatabaseUserProvider(db *DB, table string) *DatabaseUserProvider {
	return &DatabaseUserProvider{
		db:     db,
		table:  table,
		hasher: NewBcryptHasher(),
	}
}

func (dup *DatabaseUserProvider) RetrieveById(identifier interface{}) (User, error) {
	var user GenericUser
	err := dup.db.Table(dup.table).Where("id", "=", identifier).First(&user)
	return &user, err
}

func (dup *DatabaseUserProvider) RetrieveByToken(identifier interface{}, token string) (User, error) {
	var user GenericUser
	err := dup.db.Table(dup.table).
		Where("id", "=", identifier).
		Where("remember_token", "=", token).
		First(&user)
	return &user, err
}

func (dup *DatabaseUserProvider) UpdateRememberToken(user User, token string) error {
	_, err := dup.db.Table(dup.table).
		Where("id", "=", user.GetID()).
		Update(map[string]interface{}{
			"remember_token": token,
		})
	return err
}

func (dup *DatabaseUserProvider) RetrieveByCredentials(credentials map[string]interface{}) (User, error) {
	qb := dup.db.Table(dup.table)
	
	for key, value := range credentials {
		if key != "password" {
			qb = qb.Where(key, "=", value)
		}
	}
	
	var user GenericUser
	err := qb.First(&user)
	return &user, err
}

func (dup *DatabaseUserProvider) ValidateCredentials(user User, credentials map[string]interface{}) bool {
	password, exists := credentials["password"]
	if !exists {
		return false
	}
	
	passwordStr, ok := password.(string)
	if !ok {
		return false
	}
	
	return dup.hasher.Check(passwordStr, user.GetAuthPassword())
}

type GenericUser struct {
	ID           interface{} `db:"id" json:"id"`
	Email        string      `db:"email" json:"email"`
	Password     string      `db:"password" json:"-"`
	RememberToken string     `db:"remember_token" json:"-"`
	CreatedAt    time.Time   `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time   `db:"updated_at" json:"updated_at"`
}

func (gu *GenericUser) GetID() interface{} {
	return gu.ID
}

func (gu *GenericUser) GetAuthIdentifier() string {
	return fmt.Sprintf("%v", gu.ID)
}

func (gu *GenericUser) GetAuthPassword() string {
	return gu.Password
}

func (gu *GenericUser) GetRememberToken() string {
	return gu.RememberToken
}

func (gu *GenericUser) SetRememberToken(token string) {
	gu.RememberToken = token
}

func (gu *GenericUser) GetAuthIdentifierName() string {
	return "id"
}

func (gu *GenericUser) GetAuthPasswordName() string {
	return "password"
}

func (gu *GenericUser) GetRememberTokenName() string {
	return "remember_token"
}

type Hasher interface {
	Make(value string) (string, error)
	Check(value, hashed string) bool
	NeedsRehash(hashed string) bool
}

type BcryptHasher struct {
	rounds int
}

func NewBcryptHasher() *BcryptHasher {
	return &BcryptHasher{rounds: 10}
}

func (bh *BcryptHasher) Make(value string) (string, error) {
	return hashPassword(value), nil
}

func (bh *BcryptHasher) Check(value, hashed string) bool {
	return checkPassword(value, hashed)
}

func (bh *BcryptHasher) NeedsRehash(hashed string) bool {
	return false
}

func hashPassword(password string) string {
	return fmt.Sprintf("hashed_%s", password)
}

func checkPassword(password, hash string) bool {
	return hash == fmt.Sprintf("hashed_%s", password)
}

type AuthSessionManager interface {
	Get(key string) interface{}
	Put(key string, value interface{})
	Remove(key string)
	Regenerate() error
	Flash(key string, value interface{})
	GetFlash(key string) interface{}
}

func AuthMiddleware(guards ...string) MiddlewareFunc {
	return func(c *Context) error {
		auth, _ := c.app.Container().Make("auth")
		if authManager, ok := auth.(*AuthManager); ok {
			var guard Guard
			if len(guards) > 0 {
				guard = authManager.Guard(guards[0])
			} else {
				guard = authManager.Guard()
			}
			
			if sessionGuard, ok := guard.(*SessionGuard); ok {
				sessionGuard.SetRequest(c)
			}
			
			if !guard.Check() {
				return c.JSON(401, map[string]string{
					"error": "Unauthorized",
				})
			}
			
			c.Set("user", guard.User())
		}
		
		return c.Next()
	}
}

func GuestMiddleware(guards ...string) MiddlewareFunc {
	return func(c *Context) error {
		auth, _ := c.app.Container().Make("auth")
		if authManager, ok := auth.(*AuthManager); ok {
			var guard Guard
			if len(guards) > 0 {
				guard = authManager.Guard(guards[0])
			} else {
				guard = authManager.Guard()
			}
			
			if guard.Check() {
				return c.Redirect(302, "/dashboard")
			}
		}
		
		return c.Next()
	}
}

func (c *Context) Auth(guards ...string) Guard {
	auth, _ := c.app.Container().Make("auth")
	if authManager, ok := auth.(*AuthManager); ok {
		var guard Guard
		if len(guards) > 0 {
			guard = authManager.Guard(guards[0])
		} else {
			guard = authManager.Guard()
		}
		
		if sessionGuard, ok := guard.(*SessionGuard); ok {
			sessionGuard.SetRequest(c)
		}
		
		return guard
	}
	return nil
}

func (c *Context) User() User {
	if user := c.Get("user"); user != nil {
		if u, ok := user.(User); ok {
			return u
		}
	}
	
	guard := c.Auth()
	if guard != nil {
		return guard.User()
	}
	
	return nil
}

func (c *Context) Get(key string) interface{} {
	if c.data == nil {
		c.data = make(map[string]interface{})
	}
	return c.data[key]
}

func (c *Context) Set(key string, value interface{}) {
	if c.data == nil {
		c.data = make(map[string]interface{})
	}
	c.data[key] = value
}