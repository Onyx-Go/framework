package security

import (
	"net/http"

	httpInternal "github.com/onyx-go/framework/internal/http"
)

// CSRFMiddleware provides CSRF protection
func (f *MiddlewareFactory) CSRFMiddleware() httpInternal.MiddlewareFunc {
	return func(c httpInternal.Context) error {
		if !f.deps.Config.CSRFProtection() {
			return c.Next()
		}

		req := c.Request()
		
		// Skip CSRF protection for safe methods
		if req.Method == "GET" || req.Method == "HEAD" || req.Method == "OPTIONS" {
			// Set CSRF token for safe methods
			if err := f.setCSRFToken(c); err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{
					"error": "Failed to generate CSRF token",
				})
			}
			return c.Next()
		}

		// Validate CSRF token for unsafe methods
		if !f.validateCSRF(c) {
			return c.JSON(http.StatusForbidden, map[string]string{
				"error":   "CSRF token mismatch",
				"message": "The request could not be authenticated",
			})
		}

		return c.Next()
	}
}

// setCSRFToken sets a CSRF token in the context
// This is a simplified implementation - in a real implementation,
// this would generate and store a proper CSRF token
func (f *MiddlewareFactory) setCSRFToken(c httpInternal.Context) error {
	// For now, this is a placeholder implementation
	// In the actual framework, this would integrate with the session system
	c.Set("csrf_token", "placeholder_token")
	return nil
}

// validateCSRF validates a CSRF token from the request
// This is a simplified implementation - in a real implementation,
// this would validate against stored tokens
func (f *MiddlewareFactory) validateCSRF(c httpInternal.Context) bool {
	// For now, this is a placeholder implementation
	// In the actual framework, this would validate against session tokens
	token := c.Header("X-CSRF-Token")
	if token == "" {
		// Check form data
		req := c.Request()
		req.ParseForm()
		token = req.FormValue("_token")
	}
	
	// Placeholder validation - in real implementation would validate against session
	return token == "placeholder_token"
}

// CSRFProtectionMiddleware creates a CSRF protection middleware with default dependencies
func CSRFProtectionMiddleware(deps *Dependencies) httpInternal.MiddlewareFunc {
	factory := NewMiddlewareFactory(deps)
	return factory.CSRFMiddleware()
}