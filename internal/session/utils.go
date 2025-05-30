package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// GenerateSessionID generates a cryptographically secure session ID
func GenerateSessionID() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to time-based ID if crypto/rand fails
		return fmt.Sprintf("session_%d_%d", time.Now().UnixNano(), time.Now().Unix())
	}
	return hex.EncodeToString(bytes)
}

// SerializeSessionData serializes session data to bytes (legacy function)
func SerializeSessionData(data map[string]interface{}) ([]byte, error) {
	store := NewDefaultStore()
	return store.Serialize(nil, data)
}

// DeserializeSessionData deserializes session data from bytes (legacy function)
func DeserializeSessionData(data []byte) (map[string]interface{}, error) {
	store := NewDefaultStore()
	return store.Deserialize(nil, data)
}

// CreateMiddleware creates session middleware for HTTP handlers
func CreateMiddleware(manager Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			
			// Start or get session
			session, err := manager.StartSession(ctx, w, r)
			if err != nil {
				// Log error and continue without session
				// You might want to inject a logger here
				next.ServeHTTP(w, r)
				return
			}
			
			// Add session to request context
			ctx = context.WithValue(ctx, ContextKeySession, session)
			r = r.WithContext(ctx)
			
			// Create response wrapper to capture response
			wrapper := &responseWriter{
				ResponseWriter: w,
				session:        session,
			}
			
			// Call next handler
			next.ServeHTTP(wrapper, r)
			
			// Save session after request
			if session != nil && session.IsStarted() {
				session.Save()
			}
		})
	}
}

// responseWriter wraps http.ResponseWriter to save session data
type responseWriter struct {
	http.ResponseWriter
	session Session
}

// Write implements http.ResponseWriter
func (rw *responseWriter) Write(data []byte) (int, error) {
	return rw.ResponseWriter.Write(data)
}

// WriteHeader implements http.ResponseWriter
func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.ResponseWriter.WriteHeader(statusCode)
}

// ContextWithSession creates a new context with session
func ContextWithSession(ctx context.Context, session Session) context.Context {
	return context.WithValue(ctx, ContextKeySession, session)
}

// GetSessionFromContext retrieves session from context
func GetSessionFromContext(ctx context.Context) (Session, bool) {
	session, ok := ctx.Value(ContextKeySession).(Session)
	return session, ok
}

// GetSessionFromRequest retrieves session from HTTP request context
func GetSessionFromRequest(r *http.Request) (Session, bool) {
	return GetSessionFromContext(r.Context())
}

// ValidateSessionID validates a session ID format
func ValidateSessionID(sessionID string) bool {
	// Check minimum length
	if len(sessionID) < 16 {
		return false
	}
	
	// Check maximum length
	if len(sessionID) > 128 {
		return false
	}
	
	// Check for valid hex characters (if using hex encoding)
	if len(sessionID) == 64 { // 32 bytes * 2 hex chars
		if _, err := hex.DecodeString(sessionID); err != nil {
			return false
		}
	}
	
	return true
}

// IsSessionExpired checks if a session timestamp indicates expiration
func IsSessionExpired(lastAccess time.Time, lifetime time.Duration) bool {
	return time.Since(lastAccess) > lifetime
}

// CalculateSessionExpiry calculates when a session will expire
func CalculateSessionExpiry(lastAccess time.Time, lifetime time.Duration) time.Time {
	return lastAccess.Add(lifetime)
}

// CreateCookieConfig creates a cookie configuration from session config
func CreateCookieConfig(config Config) *http.Cookie {
	return &http.Cookie{
		Name:     config.CookieName,
		Path:     config.CookiePath,
		Domain:   config.CookieDomain,
		MaxAge:   int(config.Lifetime.Seconds()),
		Secure:   config.Secure,
		HttpOnly: config.HTTPOnly,
		SameSite: config.SameSite,
	}
}

// FormatSessionData formats session data for debugging
func FormatSessionData(data map[string]interface{}) string {
	if len(data) == 0 {
		return "{}"
	}
	
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error formatting session data: %v", err)
	}
	
	return string(jsonData)
}

// GetSessionSize calculates the approximate size of session data
func GetSessionSize(data map[string]interface{}) int {
	store := NewDefaultStore()
	
	serialized, err := store.Serialize(nil, data)
	if err != nil {
		return 0
	}
	
	return len(serialized)
}

// CleanSessionData removes expired or invalid entries from session data
func CleanSessionData(data map[string]interface{}) map[string]interface{} {
	cleaned := make(map[string]interface{})
	
	for key, value := range data {
		// Skip internal/system keys that might be corrupted
		if len(key) > 0 && key[0] == '_' {
			// Validate internal keys
			if key == "_created_at" || key == "_last_access" {
				if _, ok := value.(time.Time); ok {
					cleaned[key] = value
				}
			} else if key == "_lifetime" {
				if _, ok := value.(time.Duration); ok {
					cleaned[key] = value
				}
			} else if len(key) > 7 && key[:7] == "_flash." {
				// Flash data
				cleaned[key] = value
			}
		} else {
			// Regular session data
			cleaned[key] = value
		}
	}
	
	return cleaned
}

// SecureCompare performs a constant-time comparison of session IDs
func SecureCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	
	var result byte
	for i := 0; i < len(a); i++ {
		result |= a[i] ^ b[i]
	}
	
	return result == 0
}

// GenerateCSRFToken generates a CSRF token for the session
func GenerateCSRFToken() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("csrf_%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

// ValidateCSRFToken validates a CSRF token
func ValidateCSRFToken(session Session, token string) bool {
	sessionToken := session.Get("_csrf_token")
	if sessionToken == nil {
		return false
	}
	
	sessionTokenStr, ok := sessionToken.(string)
	if !ok {
		return false
	}
	
	return SecureCompare(sessionTokenStr, token)
}

// SetCSRFToken sets a CSRF token in the session
func SetCSRFToken(session Session) string {
	token := GenerateCSRFToken()
	session.Put("_csrf_token", token)
	return token
}

// GetOrCreateCSRFToken gets existing CSRF token or creates a new one
func GetOrCreateCSRFToken(session Session) string {
	if token := session.Get("_csrf_token"); token != nil {
		if tokenStr, ok := token.(string); ok && len(tokenStr) > 0 {
			return tokenStr
		}
	}
	
	return SetCSRFToken(session)
}