package security

import (
	"net/http"
	"strings"

	httpInternal "github.com/onyx-go/framework/internal/http"
)

// RateLimitingSecurityMiddleware provides security-focused rate limiting
func (f *MiddlewareFactory) RateLimitingSecurityMiddleware() httpInternal.MiddlewareFunc {
	return func(c httpInternal.Context) error {
		if !f.deps.Config.RateLimitEnabled() {
			return c.Next()
		}

		// Get client identifier (IP address)
		clientIP := c.RemoteIP()
		
		// Create rate limit key
		key := "security_rate_limit:" + clientIP
		
		// Check if request is allowed
		if !f.deps.Limiter.Allow(key) {
			f.logRateLimitExceeded(c, clientIP)
			return c.JSON(http.StatusTooManyRequests, map[string]string{
				"error":   "Rate limit exceeded",
				"message": "Too many requests from this IP address",
			})
		}

		// Special handling for authentication endpoints
		if f.isAuthEndpoint(c.Path()) {
			authKey := "auth_rate_limit:" + clientIP
			if !f.deps.Limiter.Allow(authKey) {
				f.logAuthRateLimitExceeded(c, clientIP)
				return c.JSON(http.StatusTooManyRequests, map[string]string{
					"error":   "Authentication rate limit exceeded",
					"message": "Too many authentication attempts",
				})
			}
		}

		return c.Next()
	}
}

// isAuthEndpoint checks if the current path is an authentication endpoint
func (f *MiddlewareFactory) isAuthEndpoint(path string) bool {
	authPaths := []string{
		"/login",
		"/register",
		"/password/reset",
		"/password/forgot",
		"/auth/",
		"/api/auth/",
		"/oauth/",
	}

	pathLower := strings.ToLower(path)
	for _, authPath := range authPaths {
		if strings.Contains(pathLower, authPath) {
			return true
		}
	}

	return false
}

// logRateLimitExceeded logs when rate limit is exceeded
func (f *MiddlewareFactory) logRateLimitExceeded(c httpInternal.Context, clientIP string) {
	context := map[string]interface{}{
		"ip":         clientIP,
		"method":     c.Method(),
		"url":        c.URL(),
		"user_agent": c.Header("User-Agent"),
		"event":      "rate_limit_exceeded",
	}
	
	f.deps.Logger.Warn("Rate limit exceeded", context)
}

// logAuthRateLimitExceeded logs when authentication rate limit is exceeded
func (f *MiddlewareFactory) logAuthRateLimitExceeded(c httpInternal.Context, clientIP string) {
	context := map[string]interface{}{
		"ip":         clientIP,
		"method":     c.Method(),
		"url":        c.URL(),
		"user_agent": c.Header("User-Agent"),
		"event":      "auth_rate_limit_exceeded",
	}
	
	f.deps.Logger.Error("Authentication rate limit exceeded", context)
}

// RateLimitingSecurityMiddleware creates a rate limiting security middleware with default dependencies
func RateLimitingSecurityMiddleware(deps *Dependencies) httpInternal.MiddlewareFunc {
	factory := NewMiddlewareFactory(deps)
	return factory.RateLimitingSecurityMiddleware()
}