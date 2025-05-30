package session

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// DefaultManager implements the Manager interface
type DefaultManager struct {
	handler       Handler
	store         Store
	config        Config
	mutex         sync.RWMutex
	gcTimer       *time.Timer
	gcStop        chan struct{}
	eventHandlers map[string][]func(sessionID string)
}

// NewManager creates a new session manager
func NewManager(handler Handler, config Config) *DefaultManager {
	return &DefaultManager{
		handler:       handler,
		store:         NewDefaultStore(),
		config:        config,
		gcStop:        make(chan struct{}),
		eventHandlers: make(map[string][]func(sessionID string)),
	}
}

// StartSession creates or retrieves a session
func (m *DefaultManager) StartSession(ctx context.Context, w http.ResponseWriter, r *http.Request) (Session, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	
	// Get session ID from cookie
	sessionID := m.getSessionIDFromRequest(r)
	if sessionID == "" {
		sessionID = GenerateSessionID()
	}
	
	// Create session instance
	session := NewSession(sessionID, m.handler, m.config.Lifetime)
	
	// Try to load existing session data
	if sessionID != "" {
		if exists, err := m.handler.Exists(ctx, sessionID); err == nil && exists {
			if data, err := m.handler.Read(ctx, sessionID); err == nil && len(data) > 0 {
				if sessionData, err := m.store.Deserialize(ctx, data); err == nil {
					session.LoadFromData(sessionData)
				}
			}
		}
	}
	
	// Mark session as started
	session.MarkAsStarted()
	
	// Set session cookie
	m.setSessionCookie(w, sessionID)
	
	// Trigger session created event
	m.triggerEvent("session.created", sessionID)
	
	// Start GC if not already running
	m.startGCIfNeeded(ctx)
	
	return session, nil
}

// GetSession retrieves an existing session
func (m *DefaultManager) GetSession(ctx context.Context, r *http.Request) (Session, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	
	sessionID := m.getSessionIDFromRequest(r)
	if sessionID == "" {
		return nil, fmt.Errorf("no session found")
	}
	
	// Check if session exists
	exists, err := m.handler.Exists(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	
	if !exists {
		return nil, fmt.Errorf("session does not exist")
	}
	
	// Create session instance
	session := NewSession(sessionID, m.handler, m.config.Lifetime)
	
	// Load session data
	data, err := m.handler.Read(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	
	if len(data) > 0 {
		sessionData, err := m.store.Deserialize(ctx, data)
		if err != nil {
			return nil, err
		}
		session.LoadFromData(sessionData)
	}
	
	// Check if session is expired
	if session.IsExpired() {
		m.handler.Destroy(ctx, sessionID)
		m.triggerEvent("session.expired", sessionID)
		return nil, fmt.Errorf("session expired")
	}
	
	session.MarkAsStarted()
	return session, nil
}

// DestroySession destroys a session
func (m *DefaultManager) DestroySession(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	
	sessionID := m.getSessionIDFromRequest(r)
	if sessionID == "" {
		return nil // No session to destroy
	}
	
	// Destroy session data
	if err := m.handler.Destroy(ctx, sessionID); err != nil {
		return err
	}
	
	// Clear session cookie
	m.clearSessionCookie(w)
	
	// Trigger session destroyed event
	m.triggerEvent("session.destroyed", sessionID)
	
	return nil
}

// Configuration methods

// SetCookieName sets the session cookie name
func (m *DefaultManager) SetCookieName(name string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.config.CookieName = name
}

// GetCookieName returns the session cookie name
func (m *DefaultManager) GetCookieName() string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.config.CookieName
}

// SetCookiePath sets the session cookie path
func (m *DefaultManager) SetCookiePath(path string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.config.CookiePath = path
}

// GetCookiePath returns the session cookie path
func (m *DefaultManager) GetCookiePath() string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.config.CookiePath
}

// SetCookieDomain sets the session cookie domain
func (m *DefaultManager) SetCookieDomain(domain string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.config.CookieDomain = domain
}

// GetCookieDomain returns the session cookie domain
func (m *DefaultManager) GetCookieDomain() string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.config.CookieDomain
}

// SetSecure sets whether the session cookie should be secure
func (m *DefaultManager) SetSecure(secure bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.config.Secure = secure
}

// IsSecure returns whether the session cookie is secure
func (m *DefaultManager) IsSecure() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.config.Secure
}

// SetHTTPOnly sets whether the session cookie should be HTTP only
func (m *DefaultManager) SetHTTPOnly(httpOnly bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.config.HTTPOnly = httpOnly
}

// IsHTTPOnly returns whether the session cookie is HTTP only
func (m *DefaultManager) IsHTTPOnly() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.config.HTTPOnly
}

// SetSameSite sets the session cookie SameSite attribute
func (m *DefaultManager) SetSameSite(sameSite http.SameSite) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.config.SameSite = sameSite
}

// GetSameSite returns the session cookie SameSite attribute
func (m *DefaultManager) GetSameSite() http.SameSite {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.config.SameSite
}

// SetLifetime sets the session lifetime
func (m *DefaultManager) SetLifetime(lifetime time.Duration) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.config.Lifetime = lifetime
}

// GetLifetime returns the session lifetime
func (m *DefaultManager) GetLifetime() time.Duration {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.config.Lifetime
}

// RegenerateID regenerates the session ID
func (m *DefaultManager) RegenerateID(ctx context.Context, session Session) error {
	if err := session.Regenerate(); err != nil {
		return err
	}
	
	m.triggerEvent("session.regenerated", session.ID())
	return nil
}

// TouchSession updates the session's last access time
func (m *DefaultManager) TouchSession(ctx context.Context, session Session) error {
	// The session automatically updates its last access time
	// This method exists for interface compliance
	return nil
}

// ValidateSession checks if a session is valid
func (m *DefaultManager) ValidateSession(ctx context.Context, session Session) error {
	if session.IsExpired() {
		return fmt.Errorf("session expired")
	}
	
	// Additional validation can be added here
	return nil
}

// GarbageCollect performs garbage collection
func (m *DefaultManager) GarbageCollect(ctx context.Context) error {
	maxLifetime := int64(m.config.Lifetime.Seconds())
	return m.handler.GC(ctx, maxLifetime)
}

// GetActiveSessionCount returns the number of active sessions
func (m *DefaultManager) GetActiveSessionCount(ctx context.Context) (int, error) {
	return m.handler.Count(ctx)
}

// SetHandler sets the session handler
func (m *DefaultManager) SetHandler(handler Handler) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.handler = handler
}

// GetHandler returns the session handler
func (m *DefaultManager) GetHandler() Handler {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.handler
}

// Private helper methods

// getSessionIDFromRequest extracts the session ID from the request
func (m *DefaultManager) getSessionIDFromRequest(r *http.Request) string {
	cookie, err := r.Cookie(m.config.CookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// setSessionCookie sets the session cookie in the response
func (m *DefaultManager) setSessionCookie(w http.ResponseWriter, sessionID string) {
	cookie := &http.Cookie{
		Name:     m.config.CookieName,
		Value:    sessionID,
		Path:     m.config.CookiePath,
		Domain:   m.config.CookieDomain,
		MaxAge:   int(m.config.Lifetime.Seconds()),
		Secure:   m.config.Secure,
		HttpOnly: m.config.HTTPOnly,
		SameSite: m.config.SameSite,
	}
	
	http.SetCookie(w, cookie)
}

// clearSessionCookie clears the session cookie
func (m *DefaultManager) clearSessionCookie(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     m.config.CookieName,
		Value:    "",
		Path:     m.config.CookiePath,
		Domain:   m.config.CookieDomain,
		MaxAge:   -1,
		Secure:   m.config.Secure,
		HttpOnly: m.config.HTTPOnly,
		SameSite: m.config.SameSite,
	}
	
	http.SetCookie(w, cookie)
}

// startGCIfNeeded starts garbage collection if not already running
func (m *DefaultManager) startGCIfNeeded(ctx context.Context) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if m.gcTimer == nil {
		m.startGC(ctx)
	}
}

// startGC starts the garbage collection timer
func (m *DefaultManager) startGC(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(m.config.GCInterval)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				m.GarbageCollect(ctx)
			case <-m.gcStop:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

// Event handling methods

// AddEventHandler adds an event handler
func (m *DefaultManager) AddEventHandler(event string, handler func(sessionID string)) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if m.eventHandlers[event] == nil {
		m.eventHandlers[event] = make([]func(sessionID string), 0)
	}
	
	m.eventHandlers[event] = append(m.eventHandlers[event], handler)
}

// triggerEvent triggers event handlers for a given event
func (m *DefaultManager) triggerEvent(event string, sessionID string) {
	m.mutex.RLock()
	handlers := m.eventHandlers[event]
	m.mutex.RUnlock()
	
	for _, handler := range handlers {
		go handler(sessionID) // Execute handlers asynchronously
	}
}

// Close closes the session manager
func (m *DefaultManager) Close(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	// Stop garbage collection
	if m.gcStop != nil {
		close(m.gcStop)
	}
	
	if m.gcTimer != nil {
		m.gcTimer.Stop()
	}
	
	// Close handler
	if m.handler != nil {
		return m.handler.Close(ctx)
	}
	
	return nil
}