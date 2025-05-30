package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"
)

// DefaultBcryptHasher implements bcrypt password hashing
type DefaultBcryptHasher struct {
	rounds int
}

// NewBcryptHasher creates a new bcrypt hasher
func NewBcryptHasher() *DefaultBcryptHasher {
	return &DefaultBcryptHasher{
		rounds: 10,
	}
}

// NewBcryptHasherWithRounds creates a new bcrypt hasher with custom rounds
func NewBcryptHasherWithRounds(rounds int) *DefaultBcryptHasher {
	return &DefaultBcryptHasher{
		rounds: rounds,
	}
}

// Make creates a hash from a plain text password
func (bh *DefaultBcryptHasher) Make(ctx context.Context, value string) (string, error) {
	// Simple implementation for demo - in production, use proper bcrypt
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	
	// Create hash with salt
	hasher := sha256.New()
	hasher.Write([]byte(value))
	hasher.Write(salt)
	hash := hasher.Sum(nil)
	
	// Encode with salt
	encoded := fmt.Sprintf("$simple$%s$%s",
		base64.StdEncoding.EncodeToString(salt),
		base64.StdEncoding.EncodeToString(hash))
	
	return encoded, nil
}

// Check verifies a password against a hash
func (bh *DefaultBcryptHasher) Check(ctx context.Context, value, hashed string) bool {
	parts := strings.Split(hashed, "$")
	if len(parts) != 4 || parts[1] != "simple" {
		return false
	}
	
	salt, err := base64.StdEncoding.DecodeString(parts[2])
	if err != nil {
		return false
	}
	
	expectedHash, err := base64.StdEncoding.DecodeString(parts[3])
	if err != nil {
		return false
	}
	
	// Recreate hash
	hasher := sha256.New()
	hasher.Write([]byte(value))
	hasher.Write(salt)
	actualHash := hasher.Sum(nil)
	
	return subtle.ConstantTimeCompare(expectedHash, actualHash) == 1
}

// NeedsRehash checks if a hash needs to be rehashed
func (bh *DefaultBcryptHasher) NeedsRehash(ctx context.Context, hashed string) bool {
	// For simple implementation, never needs rehash
	return false
}

// SetRounds sets the number of rounds
func (bh *DefaultBcryptHasher) SetRounds(rounds int) {
	bh.rounds = rounds
}

// GetRounds returns the number of rounds
func (bh *DefaultBcryptHasher) GetRounds() int {
	return bh.rounds
}

// PlaceholderArgon2iHasher provides a placeholder for Argon2i (use proper implementation in production)
type PlaceholderArgon2iHasher struct {
	*DefaultBcryptHasher
}

// NewArgon2iHasher creates a placeholder Argon2i hasher
func NewArgon2iHasher() *PlaceholderArgon2iHasher {
	return &PlaceholderArgon2iHasher{
		DefaultBcryptHasher: NewBcryptHasher(),
	}
}

// PlaceholderScryptHasher provides a placeholder for Scrypt (use proper implementation in production)
type PlaceholderScryptHasher struct {
	*DefaultBcryptHasher
}

// NewScryptHasher creates a placeholder Scrypt hasher
func NewScryptHasher() *PlaceholderScryptHasher {
	return &PlaceholderScryptHasher{
		DefaultBcryptHasher: NewBcryptHasher(),
	}
}

// HasherFactory creates hashers based on configuration
type HasherFactory struct{}

// NewHasherFactory creates a new hasher factory
func NewHasherFactory() *HasherFactory {
	return &HasherFactory{}
}

// CreateHasher creates a hasher based on type and configuration
func (hf *HasherFactory) CreateHasher(hasherType HasherType, config HasherConfig) (Hasher, error) {
	switch hasherType {
	case HasherTypeBcrypt:
		rounds := config.Rounds
		if rounds == 0 {
			rounds = 10
		}
		return NewBcryptHasherWithRounds(rounds), nil

	case HasherTypeArgon2i:
		return NewArgon2iHasher(), nil

	case HasherTypeScrypt:
		return NewScryptHasher(), nil

	default:
		return nil, fmt.Errorf("unsupported hasher type: %s", hasherType)
	}
}

// HasherManager manages multiple hashers
type HasherManager struct {
	hashers map[string]Hasher
	factory *HasherFactory
	default_ string
}

// NewHasherManager creates a new hasher manager
func NewHasherManager(factory *HasherFactory) *HasherManager {
	return &HasherManager{
		hashers: make(map[string]Hasher),
		factory: factory,
		default_: "bcrypt",
	}
}

// Register registers a hasher
func (hm *HasherManager) Register(name string, hasher Hasher) {
	hm.hashers[name] = hasher
}

// Get retrieves a hasher
func (hm *HasherManager) Get(name string) (Hasher, bool) {
	hasher, exists := hm.hashers[name]
	return hasher, exists
}

// GetDefault returns the default hasher
func (hm *HasherManager) GetDefault() Hasher {
	if hasher, exists := hm.hashers[hm.default_]; exists {
		return hasher
	}
	return NewBcryptHasher()
}

// SetDefault sets the default hasher
func (hm *HasherManager) SetDefault(name string) {
	hm.default_ = name
}

// Create creates and registers a new hasher
func (hm *HasherManager) Create(hasherType HasherType, name string, config HasherConfig) (Hasher, error) {
	hasher, err := hm.factory.CreateHasher(hasherType, config)
	if err != nil {
		return nil, err
	}

	hm.Register(name, hasher)
	return hasher, nil
}

// Remove removes a hasher
func (hm *HasherManager) Remove(name string) {
	delete(hm.hashers, name)
}

// List returns all registered hasher names
func (hm *HasherManager) List() []string {
	names := make([]string, 0, len(hm.hashers))
	for name := range hm.hashers {
		names = append(names, name)
	}
	return names
}

// Count returns the number of registered hashers
func (hm *HasherManager) Count() int {
	return len(hm.hashers)
}

// Make creates a hash using the default hasher
func (hm *HasherManager) Make(ctx context.Context, value string) (string, error) {
	return hm.GetDefault().Make(ctx, value)
}

// Check verifies a password using the appropriate hasher
func (hm *HasherManager) Check(ctx context.Context, value, hashed string) bool {
	// Try to determine the hasher type from the hash format
	if strings.HasPrefix(hashed, "$simple") {
		// Our simple hash format
		if hasher, exists := hm.Get("bcrypt"); exists {
			return hasher.Check(ctx, value, hashed)
		}
	}

	// Fallback to default hasher
	return hm.GetDefault().Check(ctx, value, hashed)
}

// NeedsRehash checks if a hash needs to be rehashed
func (hm *HasherManager) NeedsRehash(ctx context.Context, hashed string) bool {
	// Determine the hasher type and check if rehash is needed
	if strings.HasPrefix(hashed, "$simple") {
		if hasher, exists := hm.Get("bcrypt"); exists {
			return hasher.NeedsRehash(ctx, hashed)
		}
	}

	// If we can't determine the type, assume it needs rehashing
	return true
}

// Utility functions

// DetermineHasherType attempts to determine the hasher type from a hash
func DetermineHasherType(hashed string) HasherType {
	if strings.HasPrefix(hashed, "$simple") {
		return HasherTypeBcrypt
	}
	return HasherTypeBcrypt // default
}

// ValidateHasherConfig validates hasher configuration
func ValidateHasherConfig(hasherType HasherType, config HasherConfig) error {
	switch hasherType {
	case HasherTypeBcrypt:
		if config.Rounds < 4 || config.Rounds > 31 {
			return fmt.Errorf("bcrypt rounds must be between 4 and 31")
		}
	case HasherTypeArgon2i:
		if config.Time == 0 {
			return fmt.Errorf("argon2i time parameter must be greater than 0")
		}
		if config.Memory == 0 {
			return fmt.Errorf("argon2i memory parameter must be greater than 0")
		}
		if config.Threads == 0 {
			return fmt.Errorf("argon2i threads parameter must be greater than 0")
		}
	case HasherTypeScrypt:
		if config.Memory <= 0 {
			return fmt.Errorf("scrypt memory parameter must be greater than 0")
		}
	}
	return nil
}