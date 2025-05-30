package session

import (
	"context"
	"sync"
	"time"
)

// DefaultSession implements the Session interface
type DefaultSession struct {
	id         string
	data       map[string]interface{}
	flash      map[string]interface{}
	started    bool
	createdAt  time.Time
	lastAccess time.Time
	lifetime   time.Duration
	handler    Handler
	mutex      sync.RWMutex
	dirty      bool // Track if session has been modified
}

// NewSession creates a new session instance
func NewSession(id string, handler Handler, lifetime time.Duration) *DefaultSession {
	now := time.Now()
	return &DefaultSession{
		id:         id,
		data:       make(map[string]interface{}),
		flash:      make(map[string]interface{}),
		started:    false,
		createdAt:  now,
		lastAccess: now,
		lifetime:   lifetime,
		handler:    handler,
		dirty:      false,
	}
}

// ID returns the session ID
func (s *DefaultSession) ID() string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.id
}

// Get retrieves a value from the session
func (s *DefaultSession) Get(key string) interface{} {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	s.touch()
	return s.data[key]
}

// Put stores a value in the session
func (s *DefaultSession) Put(key string, value interface{}) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	s.data[key] = value
	s.touch()
	s.dirty = true
}

// Remove deletes a key from the session
func (s *DefaultSession) Remove(key string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	delete(s.data, key)
	s.touch()
	s.dirty = true
}

// Flush clears all session data
func (s *DefaultSession) Flush() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	s.data = make(map[string]interface{})
	s.flash = make(map[string]interface{})
	s.touch()
	s.dirty = true
}

// Has checks if a key exists in the session
func (s *DefaultSession) Has(key string) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	_, exists := s.data[key]
	return exists
}

// All returns all session data
func (s *DefaultSession) All() map[string]interface{} {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	result := make(map[string]interface{})
	for k, v := range s.data {
		result[k] = v
	}
	return result
}

// Flash stores a flash message
func (s *DefaultSession) Flash(key string, value interface{}) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	s.flash[key] = value
	s.touch()
	s.dirty = true
}

// GetFlash retrieves and removes a flash message
func (s *DefaultSession) GetFlash(key string) interface{} {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	value := s.flash[key]
	delete(s.flash, key)
	s.touch()
	if value != nil {
		s.dirty = true
	}
	return value
}

// GetAllFlash returns all flash messages without removing them
func (s *DefaultSession) GetAllFlash() map[string]interface{} {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	result := make(map[string]interface{})
	for k, v := range s.flash {
		result[k] = v
	}
	return result
}

// ClearFlash removes all flash messages
func (s *DefaultSession) ClearFlash() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	if len(s.flash) > 0 {
		s.flash = make(map[string]interface{})
		s.touch()
		s.dirty = true
	}
}

// Regenerate creates a new session ID
func (s *DefaultSession) Regenerate() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	oldID := s.id
	newID := GenerateSessionID()
	
	// Destroy old session if handler is available
	if s.handler != nil {
		ctx := context.Background()
		if err := s.handler.Destroy(ctx, oldID); err != nil {
			return err
		}
	}
	
	s.id = newID
	s.touch()
	s.dirty = true
	
	return nil
}

// InvalidateSession destroys all session data and regenerates ID
func (s *DefaultSession) InvalidateSession() error {
	s.Flush()
	return s.Regenerate()
}

// Migrate regenerates the session ID, optionally destroying old data
func (s *DefaultSession) Migrate(destroy bool) error {
	if destroy {
		return s.InvalidateSession()
	}
	return s.Regenerate()
}

// IsStarted returns whether the session has been started
func (s *DefaultSession) IsStarted() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.started
}

// Save persists the session data
func (s *DefaultSession) Save() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	if !s.dirty || s.handler == nil {
		return nil
	}
	
	// Prepare data for serialization
	data := make(map[string]interface{})
	
	// Add regular data
	for k, v := range s.data {
		data[k] = v
	}
	
	// Add flash data with prefix
	for k, v := range s.flash {
		data["_flash."+k] = v
	}
	
	// Add metadata
	data["_created_at"] = s.createdAt
	data["_last_access"] = s.lastAccess
	data["_lifetime"] = s.lifetime
	
	// Serialize and save
	serialized, err := SerializeSessionData(data)
	if err != nil {
		return err
	}
	
	ctx := context.Background()
	if err := s.handler.Write(ctx, s.id, serialized); err != nil {
		return err
	}
	
	s.dirty = false
	return nil
}

// GetCreatedAt returns the session creation time
func (s *DefaultSession) GetCreatedAt() time.Time {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.createdAt
}

// GetLastAccess returns the last access time
func (s *DefaultSession) GetLastAccess() time.Time {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.lastAccess
}

// GetLifetime returns the session lifetime
func (s *DefaultSession) GetLifetime() time.Duration {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.lifetime
}

// SetLifetime sets the session lifetime
func (s *DefaultSession) SetLifetime(duration time.Duration) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	s.lifetime = duration
	s.touch()
	s.dirty = true
}

// IsExpired checks if the session has expired
func (s *DefaultSession) IsExpired() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	return time.Since(s.lastAccess) > s.lifetime
}

// GetWithContext retrieves a value with context support
func (s *DefaultSession) GetWithContext(ctx context.Context, key string) (interface{}, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return s.Get(key), nil
	}
}

// PutWithContext stores a value with context support
func (s *DefaultSession) PutWithContext(ctx context.Context, key string, value interface{}) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		s.Put(key, value)
		return nil
	}
}

// RemoveWithContext removes a key with context support
func (s *DefaultSession) RemoveWithContext(ctx context.Context, key string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		s.Remove(key)
		return nil
	}
}

// LoadFromData populates session from deserialized data
func (s *DefaultSession) LoadFromData(data map[string]interface{}) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	// Load regular data and flash data
	for k, v := range data {
		if len(k) > 7 && k[:7] == "_flash." {
			s.flash[k[7:]] = v
		} else if k == "_created_at" {
			if createdAt, ok := v.(time.Time); ok {
				s.createdAt = createdAt
			}
		} else if k == "_last_access" {
			if lastAccess, ok := v.(time.Time); ok {
				s.lastAccess = lastAccess
			}
		} else if k == "_lifetime" {
			if lifetime, ok := v.(time.Duration); ok {
				s.lifetime = lifetime
			}
		} else if k[0] != '_' { // Skip other metadata
			s.data[k] = v
		}
	}
	
	s.started = true
}

// MarkAsStarted marks the session as started
func (s *DefaultSession) MarkAsStarted() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.started = true
}

// touch updates the last access time (internal method)
func (s *DefaultSession) touch() {
	s.lastAccess = time.Now()
}

// IsDirty returns whether the session has been modified
func (s *DefaultSession) IsDirty() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.dirty
}

// MarkClean marks the session as clean (not modified)
func (s *DefaultSession) MarkClean() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.dirty = false
}