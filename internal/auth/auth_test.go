package auth

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestDefaultUser(t *testing.T) {
	user := &DefaultUser{
		ID:       1,
		Email:    "test@example.com",
		Username: "testuser",
		Password: "hashed_password",
	}

	if user.GetID() != 1 {
		t.Errorf("Expected ID 1, got %v", user.GetID())
	}

	if user.GetAuthIdentifier() != "1" {
		t.Errorf("Expected auth identifier '1', got %s", user.GetAuthIdentifier())
	}

	if user.GetAuthPassword() != "hashed_password" {
		t.Errorf("Expected password 'hashed_password', got %s", user.GetAuthPassword())
	}

	token := "test_token"
	user.SetRememberToken(token)
	if user.GetRememberToken() != token {
		t.Errorf("Expected remember token '%s', got %s", token, user.GetRememberToken())
	}

	if user.GetAuthIdentifierName() != "id" {
		t.Errorf("Expected auth identifier name 'id', got %s", user.GetAuthIdentifierName())
	}

	if user.GetAuthPasswordName() != "password" {
		t.Errorf("Expected auth password name 'password', got %s", user.GetAuthPasswordName())
	}

	if user.GetRememberTokenName() != "remember_token" {
		t.Errorf("Expected remember token name 'remember_token', got %s", user.GetRememberTokenName())
	}
}

func TestDefaultBcryptHasher(t *testing.T) {
	hasher := NewBcryptHasher()
	ctx := context.Background()

	password := "test_password"
	
	// Test Make
	hash, err := hasher.Make(ctx, password)
	if err != nil {
		t.Fatalf("Make failed: %v", err)
	}

	if hash == "" {
		t.Error("Hash should not be empty")
	}

	// Test Check with correct password
	if !hasher.Check(ctx, password, hash) {
		t.Error("Check should return true for correct password")
	}

	// Test Check with incorrect password
	if hasher.Check(ctx, "wrong_password", hash) {
		t.Error("Check should return false for incorrect password")
	}

	// Test NeedsRehash
	if hasher.NeedsRehash(ctx, hash) {
		t.Error("Hash should not need rehashing")
	}

	// Test rounds
	hasher.SetRounds(12)
	if hasher.GetRounds() != 12 {
		t.Errorf("Expected rounds 12, got %d", hasher.GetRounds())
	}
}

func TestHasherFactory(t *testing.T) {
	factory := NewHasherFactory()

	// Test bcrypt hasher creation
	config := HasherConfig{
		Driver: string(HasherTypeBcrypt),
		Rounds: 10,
	}

	hasher, err := factory.CreateHasher(HasherTypeBcrypt, config)
	if err != nil {
		t.Fatalf("CreateHasher failed: %v", err)
	}

	if hasher == nil {
		t.Error("Hasher should not be nil")
	}

	// Test unsupported hasher type
	_, err = factory.CreateHasher("unsupported", config)
	if err == nil {
		t.Error("Expected error for unsupported hasher type")
	}
}

func TestHasherManager(t *testing.T) {
	factory := NewHasherFactory()
	manager := NewHasherManager(factory)

	// Test default hasher
	defaultHasher := manager.GetDefault()
	if defaultHasher == nil {
		t.Error("Default hasher should not be nil")
	}

	// Test register and get
	bcryptHasher := NewBcryptHasher()
	manager.Register("test", bcryptHasher)

	retrieved, exists := manager.Get("test")
	if !exists {
		t.Error("Hasher should exist")
	}
	if retrieved != bcryptHasher {
		t.Error("Retrieved hasher should match registered hasher")
	}

	// Test list
	names := manager.List()
	found := false
	for _, name := range names {
		if name == "test" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Hasher name should be in list")
	}

	// Test remove
	manager.Remove("test")
	_, exists = manager.Get("test")
	if exists {
		t.Error("Hasher should not exist after removal")
	}
}

func TestDefaultGate(t *testing.T) {
	container := &MockContainer{}
	userResolver := func(ctx context.Context) User {
		return &DefaultUser{ID: 1, Email: "test@example.com"}
	}

	gate := NewDefaultGate(container, userResolver)
	ctx := context.Background()

	// Test ability definition
	gate.Define("test-ability", func(ctx context.Context, user User, args ...interface{}) bool {
		return user.GetID() == 1
	})

	if !gate.HasAbility("test-ability") {
		t.Error("Gate should have test-ability")
	}

	// Test check
	if !gate.Check(ctx, "test-ability") {
		t.Error("Check should return true for defined ability")
	}

	// Test allows and denies
	if !gate.Allows(ctx, "test-ability") {
		t.Error("Allows should return true")
	}

	if gate.Denies(ctx, "test-ability") {
		t.Error("Denies should return false")
	}

	// Test authorize
	err := gate.Authorize(ctx, "test-ability")
	if err != nil {
		t.Errorf("Authorize should not return error: %v", err)
	}

	// Test undefined ability
	if gate.Check(ctx, "undefined-ability") {
		t.Error("Check should return false for undefined ability")
	}

	err = gate.Authorize(ctx, "undefined-ability")
	if err == nil {
		t.Error("Authorize should return error for undefined ability")
	}
}

func TestGuardFactory(t *testing.T) {
	container := &MockContainer{}
	factory := NewGuardFactory(container)

	provider := &MockUserProvider{}
	config := GuardConfig{
		Type:     string(GuardTypeSession),
		Provider: "test",
	}

	guard, err := factory.CreateGuard(GuardTypeSession, "test", provider, config)
	if err != nil {
		t.Fatalf("CreateGuard failed: %v", err)
	}

	if guard == nil {
		t.Error("Guard should not be nil")
	}

	// Test unsupported guard type
	_, err = factory.CreateGuard("unsupported", "test", provider, config)
	if err == nil {
		t.Error("Expected error for unsupported guard type")
	}
}

func TestDefaultManager(t *testing.T) {
	container := &MockContainer{}
	config := DefaultAuthConfig()
	manager := NewDefaultManager(container, config)
	ctx := context.Background()

	// Test default guard and provider names
	if manager.GetDefaultGuard() != "web" {
		t.Errorf("Expected default guard 'web', got %s", manager.GetDefaultGuard())
	}

	if manager.GetDefaultProvider() != "users" {
		t.Errorf("Expected default provider 'users', got %s", manager.GetDefaultProvider())
	}

	// Test setting defaults
	manager.SetDefaultGuard("api")
	if manager.GetDefaultGuard() != "api" {
		t.Errorf("Expected default guard 'api', got %s", manager.GetDefaultGuard())
	}

	manager.SetDefaultProvider("custom")
	if manager.GetDefaultProvider() != "custom" {
		t.Errorf("Expected default provider 'custom', got %s", manager.GetDefaultProvider())
	}

	// Test guard registration
	mockGuard := &MockGuard{}
	manager.RegisterGuard("mock", mockGuard)

	guard := manager.Guard(ctx, "mock")
	if guard != mockGuard {
		t.Error("Retrieved guard should match registered guard")
	}

	// Test provider registration
	mockProvider := &MockUserProvider{}
	manager.RegisterProvider("mock", mockProvider)

	provider := manager.Provider(ctx, "mock")
	if provider != mockProvider {
		t.Error("Retrieved provider should match registered provider")
	}
}

func TestAuthenticationEvents(t *testing.T) {
	user := &DefaultUser{ID: 1, Email: "test@example.com"}
	ctx := context.Background()

	// Test login event
	loginEvent := &LoginEvent{
		BaseEvent: BaseEvent{
			EventType: string(EventLogin),
			EventUser: user,
			EventCtx:  ctx,
			EventData: map[string]interface{}{
				"ip": "127.0.0.1",
			},
			EventTime: time.Now(),
		},
	}

	if loginEvent.Type() != string(EventLogin) {
		t.Errorf("Expected event type '%s', got %s", EventLogin, loginEvent.Type())
	}

	if loginEvent.User() != user {
		t.Error("Event user should match")
	}

	if loginEvent.Context() != ctx {
		t.Error("Event context should match")
	}

	// Test logout event
	logoutEvent := &LogoutEvent{
		BaseEvent: BaseEvent{
			EventType: string(EventLogout),
			EventUser: user,
			EventCtx:  ctx,
			EventTime: time.Now(),
		},
	}

	if logoutEvent.Type() != string(EventLogout) {
		t.Errorf("Expected event type '%s', got %s", EventLogout, logoutEvent.Type())
	}
}

func TestCredentials(t *testing.T) {
	creds := Credentials{
		"email":    "test@example.com",
		"password": "secret",
		"remember": true,
	}

	// Test Get
	if creds.Get("email") != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got %v", creds.Get("email"))
	}

	// Test GetString
	if creds.GetString("email") != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got %s", creds.GetString("email"))
	}

	if creds.GetString("nonexistent") != "" {
		t.Error("GetString should return empty string for nonexistent key")
	}

	// Test Has
	if !creds.Has("email") {
		t.Error("Credentials should have email")
	}

	if creds.Has("nonexistent") {
		t.Error("Credentials should not have nonexistent key")
	}

	// Test Set
	creds.Set("new_field", "new_value")
	if creds.GetString("new_field") != "new_value" {
		t.Error("Set should add new field")
	}
}

func TestUserData(t *testing.T) {
	data := UserData{
		"name":   "John Doe",
		"age":    30,
		"active": true,
	}

	// Test GetString
	if data.GetString("name") != "John Doe" {
		t.Errorf("Expected name 'John Doe', got %s", data.GetString("name"))
	}

	// Test GetInt
	if data.GetInt("age") != 30 {
		t.Errorf("Expected age 30, got %d", data.GetInt("age"))
	}

	// Test GetBool
	if !data.GetBool("active") {
		t.Error("Expected active to be true")
	}

	// Test Has
	if !data.Has("name") {
		t.Error("Data should have name")
	}

	// Test Set
	data.Set("email", "john@example.com")
	if data.GetString("email") != "john@example.com" {
		t.Error("Set should add new field")
	}
}

func TestResolveAbilityMethod(t *testing.T) {
	tests := []struct {
		ability  string
		expected string
	}{
		{"view", "View"},
		{"create", "Create"},
		{"view-posts", "ViewPosts"},
		{"create_user", "CreateUser"},
		{"delete-user-posts", "DeleteUserPosts"},
		{"", ""},
	}

	for _, test := range tests {
		result := ResolveAbilityMethod(test.ability)
		if result != test.expected {
			t.Errorf("ResolveAbilityMethod(%s): expected %s, got %s", test.ability, test.expected, result)
		}
	}
}

func TestGetModelName(t *testing.T) {
	user := &DefaultUser{}
	name := GetModelName(user)
	if name != "DefaultUser" {
		t.Errorf("Expected model name 'DefaultUser', got %s", name)
	}

	// Test with non-pointer
	user2 := DefaultUser{}
	name2 := GetModelName(user2)
	if name2 != "DefaultUser" {
		t.Errorf("Expected model name 'DefaultUser', got %s", name2)
	}
}

func TestIsModel(t *testing.T) {
	user := &DefaultUser{}
	if !IsModel(user) {
		t.Error("DefaultUser should be recognized as a model")
	}

	if IsModel(nil) {
		t.Error("nil should not be recognized as a model")
	}

	if IsModel("string") {
		t.Error("string should not be recognized as a model")
	}

	if IsModel(123) {
		t.Error("int should not be recognized as a model")
	}
}

// Mock implementations for testing

type MockContainer struct {
	services map[string]interface{}
}

func (c *MockContainer) Make(abstract string) (interface{}, error) {
	if c.services == nil {
		c.services = make(map[string]interface{})
	}
	
	if service, exists := c.services[abstract]; exists {
		return service, nil
	}
	
	return nil, fmt.Errorf("service not found: %s", abstract)
}

func (c *MockContainer) Bind(abstract string, concrete interface{}) {
	if c.services == nil {
		c.services = make(map[string]interface{})
	}
	c.services[abstract] = concrete
}

func (c *MockContainer) Singleton(abstract string, concrete interface{}) {
	c.Bind(abstract, concrete)
}

func (c *MockContainer) Has(abstract string) bool {
	_, exists := c.services[abstract]
	return exists
}

type MockUserProvider struct{}

func (p *MockUserProvider) RetrieveById(ctx context.Context, identifier interface{}) (User, error) {
	return &DefaultUser{ID: identifier, Email: "test@example.com"}, nil
}

func (p *MockUserProvider) RetrieveByToken(ctx context.Context, identifier interface{}, token string) (User, error) {
	return &DefaultUser{ID: 1, Email: "test@example.com", RememberToken: token}, nil
}

func (p *MockUserProvider) UpdateRememberToken(ctx context.Context, user User, token string) error {
	user.SetRememberToken(token)
	return nil
}

func (p *MockUserProvider) RetrieveByCredentials(ctx context.Context, credentials map[string]interface{}) (User, error) {
	return &DefaultUser{ID: 1, Email: "test@example.com"}, nil
}

func (p *MockUserProvider) ValidateCredentials(ctx context.Context, user User, credentials map[string]interface{}) bool {
	return true
}

type MockGuard struct {
	user User
}

func (g *MockGuard) Check(ctx context.Context) bool {
	return g.user != nil
}

func (g *MockGuard) Guest(ctx context.Context) bool {
	return g.user == nil
}

func (g *MockGuard) User(ctx context.Context) User {
	return g.user
}

func (g *MockGuard) ID(ctx context.Context) interface{} {
	if g.user != nil {
		return g.user.GetID()
	}
	return nil
}

func (g *MockGuard) Validate(ctx context.Context, credentials map[string]interface{}) bool {
	return true
}

func (g *MockGuard) Attempt(ctx context.Context, credentials map[string]interface{}, remember bool) bool {
	g.user = &DefaultUser{ID: 1, Email: "test@example.com"}
	return true
}

func (g *MockGuard) Login(ctx context.Context, user User, remember bool) error {
	g.user = user
	return nil
}

func (g *MockGuard) LoginUsingID(ctx context.Context, id interface{}, remember bool) error {
	g.user = &DefaultUser{ID: id, Email: "test@example.com"}
	return nil
}

func (g *MockGuard) Once(ctx context.Context, credentials map[string]interface{}) bool {
	g.user = &DefaultUser{ID: 1, Email: "test@example.com"}
	return true
}

func (g *MockGuard) Logout(ctx context.Context) error {
	g.user = nil
	return nil
}

func (g *MockGuard) SetUser(user User) {
	g.user = user
}

func (g *MockGuard) SetSession(session SessionManager) {
	// Mock implementation
}

func (g *MockGuard) SetRequest(request RequestContext) {
	// Mock implementation
}

type MockSessionManager struct {
	data map[string]interface{}
}

func (s *MockSessionManager) Get(key string) interface{} {
	if s.data == nil {
		return nil
	}
	return s.data[key]
}

func (s *MockSessionManager) Put(key string, value interface{}) {
	if s.data == nil {
		s.data = make(map[string]interface{})
	}
	s.data[key] = value
}

func (s *MockSessionManager) Remove(key string) {
	if s.data != nil {
		delete(s.data, key)
	}
}

func (s *MockSessionManager) Regenerate() error {
	return nil
}

func (s *MockSessionManager) Flash(key string, value interface{}) {
	s.Put("flash_"+key, value)
}

func (s *MockSessionManager) GetFlash(key string) interface{} {
	value := s.Get("flash_" + key)
	s.Remove("flash_" + key)
	return value
}

func (s *MockSessionManager) Has(key string) bool {
	return s.Get(key) != nil
}

func (s *MockSessionManager) All() map[string]interface{} {
	if s.data == nil {
		return make(map[string]interface{})
	}
	result := make(map[string]interface{})
	for k, v := range s.data {
		result[k] = v
	}
	return result
}

func (s *MockSessionManager) Invalidate() error {
	s.data = make(map[string]interface{})
	return nil
}

func (s *MockSessionManager) GetID() string {
	return "mock_session_id"
}