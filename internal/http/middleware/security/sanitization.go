package security

import (
	"strings"

	httpInternal "github.com/onyx-go/framework/internal/http"
)

// TrimStringsMiddleware trims whitespace from all string inputs
func (f *MiddlewareFactory) TrimStringsMiddleware() httpInternal.MiddlewareFunc {
	return func(c httpInternal.Context) error {
		if !f.deps.Config.TrimStrings() {
			return c.Next()
		}

		// Parse form data if available
		req := c.Request()
		if req.Method == "POST" || req.Method == "PUT" || req.Method == "PATCH" {
			req.ParseForm()
			
			// Trim form values
			for key, values := range req.PostForm {
				for i, value := range values {
					req.PostForm[key][i] = strings.TrimSpace(value)
				}
			}
		}

		return c.Next()
	}
}

// ConvertEmptyStringsToNullMiddleware converts empty strings to null values
func (f *MiddlewareFactory) ConvertEmptyStringsToNullMiddleware() httpInternal.MiddlewareFunc {
	return func(c httpInternal.Context) error {
		if !f.deps.Config.ConvertEmptyToNull() {
			return c.Next()
		}

		req := c.Request()
		if req.Method == "POST" || req.Method == "PUT" || req.Method == "PATCH" {
			req.ParseForm()
			
			// Convert empty strings to nil (represented by removing the key)
			for key, values := range req.PostForm {
				var newValues []string
				for _, value := range values {
					if strings.TrimSpace(value) != "" {
						newValues = append(newValues, value)
					}
				}
				if len(newValues) > 0 {
					req.PostForm[key] = newValues
				} else {
					delete(req.PostForm, key)
				}
			}
		}

		return c.Next()
	}
}

// InputSanitizationMiddleware sanitizes all string inputs
func (f *MiddlewareFactory) InputSanitizationMiddleware() httpInternal.MiddlewareFunc {
	return func(c httpInternal.Context) error {
		req := c.Request()
		if req.Method == "POST" || req.Method == "PUT" || req.Method == "PATCH" {
			req.ParseForm()
			
			// Sanitize form values
			for key, values := range req.PostForm {
				for i, value := range values {
					// Basic sanitization
					sanitized := f.deps.Sanitizer.SanitizeString(value)
					
					// Strip tags if configured
					if f.deps.Config.StripTags() {
						sanitized = f.deps.Sanitizer.StripTags(sanitized)
					}
					
					// Check max length
					maxLength := f.deps.Config.MaxInputLength()
					if maxLength > 0 && len(sanitized) > maxLength {
						sanitized = sanitized[:maxLength]
					}
					
					req.PostForm[key][i] = sanitized
				}
			}
		}

		return c.Next()
	}
}

// Convenience functions for backward compatibility

// TrimStringsMiddleware creates a trim strings middleware with default dependencies
func TrimStringsMiddleware(deps *Dependencies) httpInternal.MiddlewareFunc {
	factory := NewMiddlewareFactory(deps)
	return factory.TrimStringsMiddleware()
}

// ConvertEmptyStringsToNullMiddleware creates a convert empty strings middleware with default dependencies
func ConvertEmptyStringsToNullMiddleware(deps *Dependencies) httpInternal.MiddlewareFunc {
	factory := NewMiddlewareFactory(deps)
	return factory.ConvertEmptyStringsToNullMiddleware()
}

// InputSanitizationMiddleware creates an input sanitization middleware with default dependencies
func InputSanitizationMiddleware(deps *Dependencies) httpInternal.MiddlewareFunc {
	factory := NewMiddlewareFactory(deps)
	return factory.InputSanitizationMiddleware()
}