package security

import (
	"net/http"
	"strconv"
	"strings"

	httpInternal "github.com/onyx-go/framework/internal/http"
)

// CORSMiddleware provides CORS support with comprehensive configuration
func (f *MiddlewareFactory) CORSMiddleware() httpInternal.MiddlewareFunc {
	return func(c httpInternal.Context) error {
		if !f.deps.Config.CORSEnabled() {
			return c.Next()
		}

		origin := c.Header("Origin")
		
		// Check if origin is allowed
		if f.isOriginAllowed(origin) {
			c.SetHeader("Access-Control-Allow-Origin", origin)
		} else if len(f.deps.Config.CORSOrigins()) == 1 && f.deps.Config.CORSOrigins()[0] == "*" {
			c.SetHeader("Access-Control-Allow-Origin", "*")
		}

		// Set allowed methods
		methods := f.deps.Config.CORSMethods()
		if len(methods) > 0 {
			c.SetHeader("Access-Control-Allow-Methods", strings.Join(methods, ", "))
		}

		// Set allowed headers
		headers := f.deps.Config.CORSHeaders()
		if len(headers) > 0 {
			c.SetHeader("Access-Control-Allow-Headers", strings.Join(headers, ", "))
		}

		// Set credentials
		if f.deps.Config.CORSCredentials() {
			c.SetHeader("Access-Control-Allow-Credentials", "true")
		}

		// Set max age
		maxAge := f.deps.Config.CORSMaxAge()
		if maxAge > 0 {
			c.SetHeader("Access-Control-Max-Age", strconv.Itoa(maxAge))
		}

		// Handle preflight requests
		if c.Method() == "OPTIONS" {
			c.Status(http.StatusOK)
			return nil
		}

		return c.Next()
	}
}

// isOriginAllowed checks if an origin is in the allowed list
func (f *MiddlewareFactory) isOriginAllowed(origin string) bool {
	allowedOrigins := f.deps.Config.CORSOrigins()
	
	for _, allowed := range allowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
		
		// Support wildcard subdomain matching
		if strings.HasPrefix(allowed, "*.") {
			domain := allowed[2:] // Remove "*."
			if strings.HasSuffix(origin, domain) {
				return true
			}
		}
	}
	
	return false
}

// EnhancedCORSMiddleware creates a CORS middleware with default dependencies
func EnhancedCORSMiddleware(deps *Dependencies) httpInternal.MiddlewareFunc {
	factory := NewMiddlewareFactory(deps)
	return factory.CORSMiddleware()
}