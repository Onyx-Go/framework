package session

import (
	"context"
	"sync"
	"time"
)

// MemoryHandler implements the Handler interface using in-memory storage
type MemoryHandler struct {
	sessions map[string]*sessionData
	mutex    sync.RWMutex
	gcTimer  *time.Timer
	gcStop   chan struct{}
}

// sessionData holds session information in memory
type sessionData struct {
	data      []byte
	createdAt time.Time
	lastAccess time.Time
}

// NewMemoryHandler creates a new memory-based session handler
func NewMemoryHandler() *MemoryHandler {
	return &MemoryHandler{
		sessions: make(map[string]*sessionData),
		gcStop:   make(chan struct{}),
	}
}

// Open initializes the memory handler
func (mh *MemoryHandler) Open(ctx context.Context) error {
	// Start garbage collection timer
	mh.startGC(ctx)
	return nil
}

// Close shuts down the memory handler
func (mh *MemoryHandler) Close(ctx context.Context) error {
	mh.mutex.Lock()
	defer mh.mutex.Unlock()
	
	// Stop garbage collection
	if mh.gcStop != nil {
		close(mh.gcStop)
	}
	
	if mh.gcTimer != nil {
		mh.gcTimer.Stop()
	}
	
	// Clear all sessions
	mh.sessions = make(map[string]*sessionData)
	
	return nil
}

// Read retrieves session data by ID
func (mh *MemoryHandler) Read(ctx context.Context, sessionID string) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	
	mh.mutex.RLock()
	defer mh.mutex.RUnlock()
	
	if data, exists := mh.sessions[sessionID]; exists {
		// Update last access time
		data.lastAccess = time.Now()
		
		// Return copy of data
		result := make([]byte, len(data.data))
		copy(result, data.data)
		return result, nil
	}
	
	return []byte{}, nil
}

// Write stores session data by ID
func (mh *MemoryHandler) Write(ctx context.Context, sessionID string, data []byte) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	
	mh.mutex.Lock()
	defer mh.mutex.Unlock()
	
	now := time.Now()
	
	if existing, exists := mh.sessions[sessionID]; exists {
		// Update existing session
		existing.data = make([]byte, len(data))
		copy(existing.data, data)
		existing.lastAccess = now
	} else {
		// Create new session data
		sessionData := &sessionData{
			data:       make([]byte, len(data)),
			createdAt:  now,
			lastAccess: now,
		}
		copy(sessionData.data, data)
		mh.sessions[sessionID] = sessionData
	}
	
	return nil
}

// Destroy removes session data by ID
func (mh *MemoryHandler) Destroy(ctx context.Context, sessionID string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	
	mh.mutex.Lock()
	defer mh.mutex.Unlock()
	
	delete(mh.sessions, sessionID)
	return nil
}

// Exists checks if a session exists
func (mh *MemoryHandler) Exists(ctx context.Context, sessionID string) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}
	
	mh.mutex.RLock()
	defer mh.mutex.RUnlock()
	
	_, exists := mh.sessions[sessionID]
	return exists, nil
}

// GC performs garbage collection of expired sessions
func (mh *MemoryHandler) GC(ctx context.Context, maxLifetime int64) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	
	mh.mutex.Lock()
	defer mh.mutex.Unlock()
	
	cutoff := time.Now().Add(-time.Duration(maxLifetime) * time.Second)
	
	for sessionID, data := range mh.sessions {
		if data.lastAccess.Before(cutoff) {
			delete(mh.sessions, sessionID)
		}
	}
	
	return nil
}

// Count returns the number of active sessions
func (mh *MemoryHandler) Count(ctx context.Context) (int, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}
	
	mh.mutex.RLock()
	defer mh.mutex.RUnlock()
	
	return len(mh.sessions), nil
}

// ReadMultiple retrieves multiple sessions at once
func (mh *MemoryHandler) ReadMultiple(ctx context.Context, sessionIDs []string) (map[string][]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	
	mh.mutex.RLock()
	defer mh.mutex.RUnlock()
	
	result := make(map[string][]byte)
	now := time.Now()
	
	for _, sessionID := range sessionIDs {
		if data, exists := mh.sessions[sessionID]; exists {
			// Update last access time
			data.lastAccess = now
			
			// Copy data
			dataCopy := make([]byte, len(data.data))
			copy(dataCopy, data.data)
			result[sessionID] = dataCopy
		}
	}
	
	return result, nil
}

// WriteMultiple stores multiple sessions at once
func (mh *MemoryHandler) WriteMultiple(ctx context.Context, sessions map[string][]byte) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	
	mh.mutex.Lock()
	defer mh.mutex.Unlock()
	
	now := time.Now()
	
	for sessionID, data := range sessions {
		if existing, exists := mh.sessions[sessionID]; exists {
			// Update existing session
			existing.data = make([]byte, len(data))
			copy(existing.data, data)
			existing.lastAccess = now
		} else {
			// Create new session data
			sessionData := &sessionData{
				data:       make([]byte, len(data)),
				createdAt:  now,
				lastAccess: now,
			}
			copy(sessionData.data, data)
			mh.sessions[sessionID] = sessionData
		}
	}
	
	return nil
}

// DestroyMultiple removes multiple sessions at once
func (mh *MemoryHandler) DestroyMultiple(ctx context.Context, sessionIDs []string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	
	mh.mutex.Lock()
	defer mh.mutex.Unlock()
	
	for _, sessionID := range sessionIDs {
		delete(mh.sessions, sessionID)
	}
	
	return nil
}

// GetStatistics returns memory handler statistics
func (mh *MemoryHandler) GetStatistics(ctx context.Context) (*Statistics, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	
	mh.mutex.RLock()
	defer mh.mutex.RUnlock()
	
	stats := &Statistics{
		ActiveSessions: len(mh.sessions),
		TotalSessions:  len(mh.sessions), // For memory handler, total = active
	}
	
	// Calculate average lifetime
	if len(mh.sessions) > 0 {
		var totalLifetime time.Duration
		now := time.Now()
		
		for _, data := range mh.sessions {
			lifetime := now.Sub(data.createdAt)
			totalLifetime += lifetime
		}
		
		stats.AverageLifetime = totalLifetime / time.Duration(len(mh.sessions))
	}
	
	return stats, nil
}

// startGC starts the garbage collection timer
func (mh *MemoryHandler) startGC(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(10 * time.Minute) // GC every 10 minutes
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				// Perform garbage collection with 1 hour max lifetime
				mh.GC(ctx, 3600)
			case <-mh.gcStop:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

// GetAllSessions returns information about all sessions (for debugging)
func (mh *MemoryHandler) GetAllSessions(ctx context.Context) map[string]time.Time {
	select {
	case <-ctx.Done():
		return nil
	default:
	}
	
	mh.mutex.RLock()
	defer mh.mutex.RUnlock()
	
	result := make(map[string]time.Time)
	for sessionID, data := range mh.sessions {
		result[sessionID] = data.lastAccess
	}
	
	return result
}

// ClearExpiredSessions manually removes expired sessions
func (mh *MemoryHandler) ClearExpiredSessions(ctx context.Context, maxLifetime time.Duration) (int, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}
	
	mh.mutex.Lock()
	defer mh.mutex.Unlock()
	
	cutoff := time.Now().Add(-maxLifetime)
	removed := 0
	
	for sessionID, data := range mh.sessions {
		if data.lastAccess.Before(cutoff) {
			delete(mh.sessions, sessionID)
			removed++
		}
	}
	
	return removed, nil
}