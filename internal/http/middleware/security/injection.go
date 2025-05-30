package security

import (
	"net/http"
	"regexp"

	httpInternal "github.com/onyx-go/framework/internal/http"
)

// SQLInjectionProtectionMiddleware detects and blocks potential SQL injection attempts
func (f *MiddlewareFactory) SQLInjectionProtectionMiddleware() httpInternal.MiddlewareFunc {
	// Common SQL injection patterns
	sqlPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(union\s+select)`),
		regexp.MustCompile(`(?i)(select\s+.*\s+from)`),
		regexp.MustCompile(`(?i)(insert\s+into)`),
		regexp.MustCompile(`(?i)(update\s+.*\s+set)`),
		regexp.MustCompile(`(?i)(delete\s+from)`),
		regexp.MustCompile(`(?i)(drop\s+table)`),
		regexp.MustCompile(`(?i)(drop\s+database)`),
		regexp.MustCompile(`(?i)(create\s+table)`),
		regexp.MustCompile(`(?i)(alter\s+table)`),
		regexp.MustCompile(`(?i)(\'\s*or\s*\')`),
		regexp.MustCompile(`(?i)(\'\s*;\s*--)`),
		regexp.MustCompile(`(?i)(--\s*$)`),
		regexp.MustCompile(`(?i)(/\*.*\*/)`),
		regexp.MustCompile(`(?i)(exec\s*\()`),
		regexp.MustCompile(`(?i)(sp_executesql)`),
		regexp.MustCompile(`(?i)(xp_cmdshell)`),
	}

	return func(c httpInternal.Context) error {
		req := c.Request()
		
		// Check query parameters
		for key, values := range req.URL.Query() {
			for _, value := range values {
				if f.containsSQLInjection(value, sqlPatterns) {
					f.logSQLInjectionAttempt(c, key, value, "query")
					return c.JSON(http.StatusBadRequest, map[string]string{
						"error": "Invalid request parameters",
					})
				}
			}
		}

		// Check form data for POST/PUT/PATCH requests
		if req.Method == "POST" || req.Method == "PUT" || req.Method == "PATCH" {
			req.ParseForm()
			
			for key, values := range req.PostForm {
				for _, value := range values {
					if f.containsSQLInjection(value, sqlPatterns) {
						f.logSQLInjectionAttempt(c, key, value, "form")
						return c.JSON(http.StatusBadRequest, map[string]string{
							"error": "Invalid form data",
						})
					}
				}
			}
		}

		return c.Next()
	}
}

// containsSQLInjection checks if a value contains SQL injection patterns
func (f *MiddlewareFactory) containsSQLInjection(value string, patterns []*regexp.Regexp) bool {
	for _, pattern := range patterns {
		if pattern.MatchString(value) {
			return true
		}
	}
	return false
}

// logSQLInjectionAttempt logs a potential SQL injection attempt
func (f *MiddlewareFactory) logSQLInjectionAttempt(c httpInternal.Context, field, value, source string) {
	context := map[string]interface{}{
		"ip":      c.RemoteIP(),
		"method":  c.Method(),
		"url":     c.URL(),
		"field":   field,
		"value":   value,
		"source":  source,
		"pattern": "SQL injection attempt",
	}
	
	f.deps.Logger.Warn("Potential SQL injection attempt detected", context)
}

// XSSProtectionValidator provides XSS detection and blocking
func (f *MiddlewareFactory) XSSProtectionValidator() httpInternal.MiddlewareFunc {
	// Common XSS patterns
	xssPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`),
		regexp.MustCompile(`(?i)<iframe[^>]*>.*?</iframe>`),
		regexp.MustCompile(`(?i)<object[^>]*>.*?</object>`),
		regexp.MustCompile(`(?i)<embed[^>]*>`),
		regexp.MustCompile(`(?i)<link[^>]*>`),
		regexp.MustCompile(`(?i)javascript:`),
		regexp.MustCompile(`(?i)vbscript:`),
		regexp.MustCompile(`(?i)onload\s*=`),
		regexp.MustCompile(`(?i)onerror\s*=`),
		regexp.MustCompile(`(?i)onclick\s*=`),
		regexp.MustCompile(`(?i)onmouseover\s*=`),
		regexp.MustCompile(`(?i)onfocus\s*=`),
		regexp.MustCompile(`(?i)onblur\s*=`),
		regexp.MustCompile(`(?i)alert\s*\(`),
		regexp.MustCompile(`(?i)confirm\s*\(`),
		regexp.MustCompile(`(?i)prompt\s*\(`),
	}

	return func(c httpInternal.Context) error {
		req := c.Request()
		
		// Check query parameters
		for key, values := range req.URL.Query() {
			for _, value := range values {
				if f.containsXSS(value, xssPatterns) {
					f.logXSSAttempt(c, key, value, "query")
					return c.JSON(http.StatusBadRequest, map[string]string{
						"error": "Invalid request parameters",
					})
				}
			}
		}

		// Check form data for POST/PUT/PATCH requests
		if req.Method == "POST" || req.Method == "PUT" || req.Method == "PATCH" {
			req.ParseForm()
			
			for key, values := range req.PostForm {
				for _, value := range values {
					if f.containsXSS(value, xssPatterns) {
						f.logXSSAttempt(c, key, value, "form")
						return c.JSON(http.StatusBadRequest, map[string]string{
							"error": "Invalid form data",
						})
					}
				}
			}
		}

		return c.Next()
	}
}

// containsXSS checks if a value contains XSS patterns
func (f *MiddlewareFactory) containsXSS(value string, patterns []*regexp.Regexp) bool {
	for _, pattern := range patterns {
		if pattern.MatchString(value) {
			return true
		}
	}
	return false
}

// logXSSAttempt logs a potential XSS attempt
func (f *MiddlewareFactory) logXSSAttempt(c httpInternal.Context, field, value, source string) {
	context := map[string]interface{}{
		"ip":      c.RemoteIP(),
		"method":  c.Method(),
		"url":     c.URL(),
		"field":   field,
		"value":   value,
		"source":  source,
		"pattern": "XSS attempt",
	}
	
	f.deps.Logger.Warn("Potential XSS attempt detected", context)
}

// Convenience functions for backward compatibility

// SQLInjectionProtectionMiddleware creates a SQL injection protection middleware with default dependencies
func SQLInjectionProtectionMiddleware(deps *Dependencies) httpInternal.MiddlewareFunc {
	factory := NewMiddlewareFactory(deps)
	return factory.SQLInjectionProtectionMiddleware()
}

// XSSValidationMiddleware creates an XSS validation middleware with default dependencies
func XSSValidationMiddleware(deps *Dependencies) httpInternal.MiddlewareFunc {
	factory := NewMiddlewareFactory(deps)
	return factory.XSSProtectionValidator()
}