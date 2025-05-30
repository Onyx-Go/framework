package auth

import (
	"context"
	"fmt"
	"time"
)

// DefaultDatabaseUserProvider implements database-based user provider
type DefaultDatabaseUserProvider struct {
	database Database
	table    string
	hasher   Hasher
}

// NewDatabaseUserProvider creates a new database user provider
func NewDatabaseUserProvider(database Database, table string, hasher Hasher) *DefaultDatabaseUserProvider {
	return &DefaultDatabaseUserProvider{
		database: database,
		table:    table,
		hasher:   hasher,
	}
}

// RetrieveById retrieves a user by ID
func (dup *DefaultDatabaseUserProvider) RetrieveById(ctx context.Context, identifier interface{}) (User, error) {
	var user DefaultUser
	err := dup.database.Table(dup.table).Where("id", "=", identifier).First(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// RetrieveByToken retrieves a user by remember token
func (dup *DefaultDatabaseUserProvider) RetrieveByToken(ctx context.Context, identifier interface{}, token string) (User, error) {
	var user DefaultUser
	query := dup.database.Table(dup.table).Where("remember_token", "=", token)
	
	if identifier != nil {
		query = query.Where("id", "=", identifier)
	}
	
	err := query.First(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateRememberToken updates the user's remember token
func (dup *DefaultDatabaseUserProvider) UpdateRememberToken(ctx context.Context, user User, token string) error {
	_, err := dup.database.Table(dup.table).
		Where("id", "=", user.GetID()).
		Update(map[string]interface{}{
			"remember_token": token,
			"updated_at":     time.Now(),
		})
	return err
}

// RetrieveByCredentials retrieves a user by credentials
func (dup *DefaultDatabaseUserProvider) RetrieveByCredentials(ctx context.Context, credentials map[string]interface{}) (User, error) {
	qb := dup.database.Table(dup.table)

	for key, value := range credentials {
		if key != "password" {
			qb = qb.Where(key, "=", value)
		}
	}

	var user DefaultUser
	err := qb.First(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// ValidateCredentials validates user credentials
func (dup *DefaultDatabaseUserProvider) ValidateCredentials(ctx context.Context, user User, credentials map[string]interface{}) bool {
	password, exists := credentials["password"]
	if !exists {
		return false
	}

	passwordStr, ok := password.(string)
	if !ok {
		return false
	}

	return dup.hasher.Check(ctx, passwordStr, user.GetAuthPassword())
}

// DefaultUser implements the User and Authenticatable interfaces
type DefaultUser struct {
	ID            interface{} `db:"id" json:"id"`
	Email         string      `db:"email" json:"email"`
	Username      string      `db:"username" json:"username,omitempty"`
	Password      string      `db:"password" json:"-"`
	RememberToken string      `db:"remember_token" json:"-"`
	EmailVerifiedAt *time.Time `db:"email_verified_at" json:"email_verified_at,omitempty"`
	CreatedAt     time.Time   `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time   `db:"updated_at" json:"updated_at"`
	DeletedAt     *time.Time  `db:"deleted_at" json:"deleted_at,omitempty"`
}

// GetID returns the user's ID
func (du *DefaultUser) GetID() interface{} {
	return du.ID
}

// GetAuthIdentifier returns the user's authentication identifier
func (du *DefaultUser) GetAuthIdentifier() string {
	return fmt.Sprintf("%v", du.ID)
}

// GetAuthPassword returns the user's hashed password
func (du *DefaultUser) GetAuthPassword() string {
	return du.Password
}

// GetRememberToken returns the user's remember token
func (du *DefaultUser) GetRememberToken() string {
	return du.RememberToken
}

// SetRememberToken sets the user's remember token
func (du *DefaultUser) SetRememberToken(token string) {
	du.RememberToken = token
}

// GetAuthIdentifierName returns the name of the identifier field
func (du *DefaultUser) GetAuthIdentifierName() string {
	return "id"
}

// GetAuthPasswordName returns the name of the password field
func (du *DefaultUser) GetAuthPasswordName() string {
	return "password"
}

// GetRememberTokenName returns the name of the remember token field
func (du *DefaultUser) GetRememberTokenName() string {
	return "remember_token"
}

// GetEmail returns the user's email
func (du *DefaultUser) GetEmail() string {
	return du.Email
}

// GetUsername returns the user's username
func (du *DefaultUser) GetUsername() string {
	return du.Username
}

// IsEmailVerified checks if the user's email is verified
func (du *DefaultUser) IsEmailVerified() bool {
	return du.EmailVerifiedAt != nil
}

// MarkEmailAsVerified marks the user's email as verified
func (du *DefaultUser) MarkEmailAsVerified() {
	now := time.Now()
	du.EmailVerifiedAt = &now
}

// UserProviderFactory creates user providers based on configuration
type UserProviderFactory struct {
	container Container
}

// NewUserProviderFactory creates a new user provider factory
func NewUserProviderFactory(container Container) *UserProviderFactory {
	return &UserProviderFactory{
		container: container,
	}
}

// CreateProvider creates a user provider based on type and configuration
func (upf *UserProviderFactory) CreateProvider(providerType ProviderType, config ProviderConfig) (UserProvider, error) {
	switch providerType {
	case ProviderTypeDatabase:
		return upf.createDatabaseProvider(config)
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}
}

func (upf *UserProviderFactory) createDatabaseProvider(config ProviderConfig) (UserProvider, error) {
	db, err := upf.container.Make("database")
	if err != nil {
		return nil, fmt.Errorf("database not available: %v", err)
	}

	database, ok := db.(Database)
	if !ok {
		return nil, fmt.Errorf("invalid database type")
	}

	hasher := NewBcryptHasher()
	return NewDatabaseUserProvider(database, config.Table, hasher), nil
}

// ProviderManager manages multiple user providers
type ProviderManager struct {
	providers map[string]UserProvider
	factory   *UserProviderFactory
}

// NewProviderManager creates a new provider manager
func NewProviderManager(factory *UserProviderFactory) *ProviderManager {
	return &ProviderManager{
		providers: make(map[string]UserProvider),
		factory:   factory,
	}
}

// Register registers a user provider
func (pm *ProviderManager) Register(name string, provider UserProvider) {
	pm.providers[name] = provider
}

// Get retrieves a user provider
func (pm *ProviderManager) Get(name string) (UserProvider, bool) {
	provider, exists := pm.providers[name]
	return provider, exists
}

// Create creates and registers a new user provider
func (pm *ProviderManager) Create(providerType ProviderType, name string, config ProviderConfig) (UserProvider, error) {
	provider, err := pm.factory.CreateProvider(providerType, config)
	if err != nil {
		return nil, err
	}

	pm.Register(name, provider)
	return provider, nil
}

// Remove removes a user provider
func (pm *ProviderManager) Remove(name string) {
	delete(pm.providers, name)
}

// List returns all registered provider names
func (pm *ProviderManager) List() []string {
	names := make([]string, 0, len(pm.providers))
	for name := range pm.providers {
		names = append(names, name)
	}
	return names
}

// Count returns the number of registered providers
func (pm *ProviderManager) Count() int {
	return len(pm.providers)
}

// UserRepository provides additional user operations
type UserRepository interface {
	Create(ctx context.Context, user User) error
	Update(ctx context.Context, user User) error
	Delete(ctx context.Context, id interface{}) error
	FindByEmail(ctx context.Context, email string) (User, error)
	FindByUsername(ctx context.Context, username string) (User, error)
	Search(ctx context.Context, query string, limit, offset int) ([]User, error)
	Count(ctx context.Context) (int64, error)
	Exists(ctx context.Context, field string, value interface{}) (bool, error)
}

// DefaultUserRepository implements UserRepository
type DefaultUserRepository struct {
	database Database
	table    string
}

// NewDefaultUserRepository creates a new user repository
func NewDefaultUserRepository(database Database, table string) *DefaultUserRepository {
	return &DefaultUserRepository{
		database: database,
		table:    table,
	}
}

// Create creates a new user
func (ur *DefaultUserRepository) Create(ctx context.Context, user User) error {
	defaultUser, ok := user.(*DefaultUser)
	if !ok {
		return fmt.Errorf("invalid user type")
	}

	now := time.Now()
	defaultUser.CreatedAt = now
	defaultUser.UpdatedAt = now

	values := map[string]interface{}{
		"email":      defaultUser.Email,
		"username":   defaultUser.Username,
		"password":   defaultUser.Password,
		"created_at": defaultUser.CreatedAt,
		"updated_at": defaultUser.UpdatedAt,
	}

	result, err := ur.database.Table(ur.table).Insert(values)
	if err != nil {
		return err
	}

	if id, err := result.LastInsertId(); err == nil {
		defaultUser.ID = id
	}

	return nil
}

// Update updates an existing user
func (ur *DefaultUserRepository) Update(ctx context.Context, user User) error {
	defaultUser, ok := user.(*DefaultUser)
	if !ok {
		return fmt.Errorf("invalid user type")
	}

	defaultUser.UpdatedAt = time.Now()

	values := map[string]interface{}{
		"email":          defaultUser.Email,
		"username":       defaultUser.Username,
		"remember_token": defaultUser.RememberToken,
		"updated_at":     defaultUser.UpdatedAt,
	}

	if defaultUser.EmailVerifiedAt != nil {
		values["email_verified_at"] = defaultUser.EmailVerifiedAt
	}

	_, err := ur.database.Table(ur.table).
		Where("id", "=", defaultUser.ID).
		Update(values)

	return err
}

// Delete deletes a user
func (ur *DefaultUserRepository) Delete(ctx context.Context, id interface{}) error {
	_, err := ur.database.Table(ur.table).
		Where("id", "=", id).
		Delete()
	return err
}

// FindByEmail finds a user by email
func (ur *DefaultUserRepository) FindByEmail(ctx context.Context, email string) (User, error) {
	var user DefaultUser
	err := ur.database.Table(ur.table).
		Where("email", "=", email).
		First(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByUsername finds a user by username
func (ur *DefaultUserRepository) FindByUsername(ctx context.Context, username string) (User, error) {
	var user DefaultUser
	err := ur.database.Table(ur.table).
		Where("username", "=", username).
		First(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Search searches for users
func (ur *DefaultUserRepository) Search(ctx context.Context, query string, limit, offset int) ([]User, error) {
	var users []DefaultUser
	err := ur.database.Table(ur.table).
		Where("email", "LIKE", "%"+query+"%").
		Where("username", "LIKE", "%"+query+"%").
		Limit(limit).
		Offset(offset).
		Find(&users)
	
	if err != nil {
		return nil, err
	}

	result := make([]User, len(users))
	for i := range users {
		result[i] = &users[i]
	}

	return result, nil
}

// Count returns the total number of users
func (ur *DefaultUserRepository) Count(ctx context.Context) (int64, error) {
	return ur.database.Table(ur.table).Count()
}

// Exists checks if a user exists with the given field and value
func (ur *DefaultUserRepository) Exists(ctx context.Context, field string, value interface{}) (bool, error) {
	count, err := ur.database.Table(ur.table).
		Where(field, "=", value).
		Count()
	
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// UserValidator validates user data
type UserValidator struct {
	repository UserRepository
}

// NewUserValidator creates a new user validator
func NewUserValidator(repository UserRepository) *UserValidator {
	return &UserValidator{
		repository: repository,
	}
}

// ValidateForCreation validates user data for creation
func (uv *UserValidator) ValidateForCreation(ctx context.Context, data map[string]interface{}) error {
	if email, ok := data["email"].(string); ok {
		if exists, err := uv.repository.Exists(ctx, "email", email); err != nil {
			return err
		} else if exists {
			return fmt.Errorf("email already exists")
		}
	}

	if username, ok := data["username"].(string); ok && username != "" {
		if exists, err := uv.repository.Exists(ctx, "username", username); err != nil {
			return err
		} else if exists {
			return fmt.Errorf("username already exists")
		}
	}

	return nil
}

// ValidateForUpdate validates user data for update
func (uv *UserValidator) ValidateForUpdate(ctx context.Context, userID interface{}, data map[string]interface{}) error {
	if _, ok := data["email"].(string); ok {
		// This would need a more sophisticated query to exclude the current user
		// For now, we'll skip this validation in updates
	}

	if _, ok := data["username"].(string); ok {
		// Similar to email, we'd need to exclude the current user
	}

	return nil
}

// UserMetrics provides user-related metrics
type UserMetrics struct {
	repository UserRepository
}

// NewUserMetrics creates a new user metrics instance
func NewUserMetrics(repository UserRepository) *UserMetrics {
	return &UserMetrics{
		repository: repository,
	}
}

// GetTotalUsers returns the total number of users
func (um *UserMetrics) GetTotalUsers(ctx context.Context) (int64, error) {
	return um.repository.Count(ctx)
}

// GetActiveUsers returns the number of active users (placeholder implementation)
func (um *UserMetrics) GetActiveUsers(ctx context.Context, since time.Time) (int64, error) {
	// This would require tracking user activity
	return 0, fmt.Errorf("not implemented")
}

// GetNewUsers returns the number of new users since a given time
func (um *UserMetrics) GetNewUsers(ctx context.Context, since time.Time) (int64, error) {
	// This would require a more sophisticated query
	return 0, fmt.Errorf("not implemented")
}