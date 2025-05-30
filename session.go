package onyx

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type Session interface {
	ID() string
	Get(key string) interface{}
	Put(key string, value interface{})
	Remove(key string)
	Flush()
	Flash(key string, value interface{})
	GetFlash(key string) interface{}
	Has(key string) bool
	All() map[string]interface{}
	Regenerate() error
	InvalidateSession() error
	Migrate(destroy bool) error
	IsStarted() bool
	Save() error
}

type SessionHandler interface {
	Open() error
	Close() error
	Read(sessionID string) ([]byte, error)
	Write(sessionID string, data []byte) error
	Destroy(sessionID string) error
	GC(maxLifetime int64) error
}

type MemorySession struct {
	id        string
	data      map[string]interface{}
	flash     map[string]interface{}
	started   bool
	mutex     sync.RWMutex
	handler   SessionHandler
	lifetime  time.Duration
	lastAccess time.Time
}

func NewMemorySession(id string, handler SessionHandler) *MemorySession {
	return &MemorySession{
		id:        id,
		data:      make(map[string]interface{}),
		flash:     make(map[string]interface{}),
		handler:   handler,
		lifetime:  time.Hour * 2,
		lastAccess: time.Now(),
	}
}

func (ms *MemorySession) ID() string {
	return ms.id
}

func (ms *MemorySession) Get(key string) interface{} {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()
	
	ms.lastAccess = time.Now()
	return ms.data[key]
}

func (ms *MemorySession) Put(key string, value interface{}) {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	
	ms.data[key] = value
	ms.lastAccess = time.Now()
}

func (ms *MemorySession) Remove(key string) {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	
	delete(ms.data, key)
	ms.lastAccess = time.Now()
}

func (ms *MemorySession) Flush() {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	
	ms.data = make(map[string]interface{})
	ms.flash = make(map[string]interface{})
	ms.lastAccess = time.Now()
}

func (ms *MemorySession) Flash(key string, value interface{}) {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	
	ms.flash[key] = value
	ms.lastAccess = time.Now()
}

func (ms *MemorySession) GetFlash(key string) interface{} {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	
	value := ms.flash[key]
	delete(ms.flash, key)
	ms.lastAccess = time.Now()
	return value
}

func (ms *MemorySession) Has(key string) bool {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()
	
	_, exists := ms.data[key]
	return exists
}

func (ms *MemorySession) All() map[string]interface{} {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()
	
	result := make(map[string]interface{})
	for k, v := range ms.data {
		result[k] = v
	}
	return result
}

func (ms *MemorySession) Regenerate() error {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	
	newID := generateSessionID()
	oldID := ms.id
	
	if ms.handler != nil {
		if err := ms.handler.Destroy(oldID); err != nil {
			return err
		}
	}
	
	ms.id = newID
	ms.lastAccess = time.Now()
	return nil
}

func (ms *MemorySession) InvalidateSession() error {
	ms.Flush()
	return ms.Regenerate()
}

func (ms *MemorySession) Migrate(destroy bool) error {
	if destroy {
		return ms.InvalidateSession()
	}
	return ms.Regenerate()
}

func (ms *MemorySession) IsStarted() bool {
	return ms.started
}

func (ms *MemorySession) Save() error {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()
	
	if ms.handler != nil {
		data := make(map[string]interface{})
		for k, v := range ms.data {
			data[k] = v
		}
		for k, v := range ms.flash {
			data["_flash."+k] = v
		}
		
		serialized, err := serializeSessionData(data)
		if err != nil {
			return err
		}
		
		return ms.handler.Write(ms.id, serialized)
	}
	
	return nil
}

type MemorySessionHandler struct {
	sessions map[string][]byte
	mutex    sync.RWMutex
}

func NewMemorySessionHandler() *MemorySessionHandler {
	return &MemorySessionHandler{
		sessions: make(map[string][]byte),
	}
}

func (msh *MemorySessionHandler) Open() error {
	return nil
}

func (msh *MemorySessionHandler) Close() error {
	msh.mutex.Lock()
	defer msh.mutex.Unlock()
	msh.sessions = make(map[string][]byte)
	return nil
}

func (msh *MemorySessionHandler) Read(sessionID string) ([]byte, error) {
	msh.mutex.RLock()
	defer msh.mutex.RUnlock()
	
	if data, exists := msh.sessions[sessionID]; exists {
		return data, nil
	}
	
	return []byte{}, nil
}

func (msh *MemorySessionHandler) Write(sessionID string, data []byte) error {
	msh.mutex.Lock()
	defer msh.mutex.Unlock()
	
	msh.sessions[sessionID] = data
	return nil
}

func (msh *MemorySessionHandler) Destroy(sessionID string) error {
	msh.mutex.Lock()
	defer msh.mutex.Unlock()
	
	delete(msh.sessions, sessionID)
	return nil
}

func (msh *MemorySessionHandler) GC(maxLifetime int64) error {
	return nil
}

type SessionManager struct {
	handler     SessionHandler
	cookieName  string
	cookiePath  string
	cookieDomain string
	secure      bool
	httpOnly    bool
	lifetime    time.Duration
}

func NewSessionManager(handler SessionHandler) *SessionManager {
	return &SessionManager{
		handler:     handler,
		cookieName:  "onyx_session",
		cookiePath:  "/",
		secure:      false,
		httpOnly:    true,
		lifetime:    time.Hour * 2,
	}
}

func (sm *SessionManager) StartSession(c *Context) (Session, error) {
	sessionID := sm.getSessionID(c)
	if sessionID == "" {
		sessionID = generateSessionID()
	}
	
	session := NewMemorySession(sessionID, sm.handler)
	
	if data, err := sm.handler.Read(sessionID); err == nil && len(data) > 0 {
		sessionData, err := deserializeSessionData(data)
		if err == nil {
			for k, v := range sessionData {
				if len(k) > 7 && k[:7] == "_flash." {
					session.flash[k[7:]] = v
				} else {
					session.data[k] = v
				}
			}
		}
	}
	
	session.started = true
	
	cookie := &http.Cookie{
		Name:     sm.cookieName,
		Value:    sessionID,
		Path:     sm.cookiePath,
		Domain:   sm.cookieDomain,
		MaxAge:   int(sm.lifetime.Seconds()),
		Secure:   sm.secure,
		HttpOnly: sm.httpOnly,
		SameSite: http.SameSiteLaxMode,
	}
	
	c.SetCookie(cookie)
	
	return session, nil
}

func (sm *SessionManager) getSessionID(c *Context) string {
	cookie, err := c.Cookie(sm.cookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func generateSessionID() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func serializeSessionData(data map[string]interface{}) ([]byte, error) {
	return []byte(fmt.Sprintf("%+v", data)), nil
}

func deserializeSessionData(data []byte) (map[string]interface{}, error) {
	return make(map[string]interface{}), nil
}

func SessionMiddleware(sm *SessionManager) MiddlewareFunc {
	return func(c *Context) error {
		session, err := sm.StartSession(c)
		if err != nil {
			return err
		}
		
		c.Set("session", session)
		
		err = c.Next()
		
		if session != nil {
			session.Save()
		}
		
		return err
	}
}

func (c *Context) Session() Session {
	if session := c.Get("session"); session != nil {
		if s, ok := session.(Session); ok {
			return s
		}
	}
	return nil
}