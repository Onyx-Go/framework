package onyx

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Security Middleware Functions

// TrimStringsMiddleware trims whitespace from all string inputs
func TrimStringsMiddleware() MiddlewareFunc {
	return func(c *Context) error {
		config := GetSecurityConfig()
		if !config.TrimStrings {
			return c.Next()
		}

		// Parse form data if available
		if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
			c.Request.ParseForm()
			
			// Trim form values
			for key, values := range c.Request.PostForm {
				for i, value := range values {
					c.Request.PostForm[key][i] = strings.TrimSpace(value)
				}
			}
		}

		return c.Next()
	}
}

// ConvertEmptyStringsToNullMiddleware converts empty strings to null values
func ConvertEmptyStringsToNullMiddleware() MiddlewareFunc {
	return func(c *Context) error {
		config := GetSecurityConfig()
		if !config.ConvertEmptyToNull {
			return c.Next()
		}

		if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
			c.Request.ParseForm()
			
			// Convert empty strings to nil (represented by removing the key)
			for key, values := range c.Request.PostForm {
				var newValues []string
				for _, value := range values {
					if strings.TrimSpace(value) != "" {
						newValues = append(newValues, value)
					}
				}
				if len(newValues) > 0 {
					c.Request.PostForm[key] = newValues
				} else {
					delete(c.Request.PostForm, key)
				}
			}
		}

		return c.Next()
	}
}

// InputSanitizationMiddleware sanitizes all string inputs
func InputSanitizationMiddleware() MiddlewareFunc {
	return func(c *Context) error {
		config := GetSecurityConfig()
		
		if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
			c.Request.ParseForm()
			
			// Sanitize form values
			for key, values := range c.Request.PostForm {
				for i, value := range values {
					// Basic sanitization
					sanitized := SanitizeString(value)
					
					// Strip tags if configured
					if config.StripTags {
						sanitized = StripTags(sanitized)
					}
					
					// Check max length
					if config.MaxInputLength > 0 && len(sanitized) > config.MaxInputLength {
						sanitized = sanitized[:config.MaxInputLength]
					}
					
					c.Request.PostForm[key][i] = sanitized
				}
			}
		}

		return c.Next()
	}
}

// CSRFProtectionMiddleware provides CSRF protection
func CSRFProtectionMiddleware() MiddlewareFunc {
	return func(c *Context) error {
		config := GetSecurityConfig()
		if !config.CSRFProtection {
			return c.Next()
		}

		// Skip CSRF protection for safe methods
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			// Set CSRF token for safe methods
			if err := c.SetCSRFToken(); err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{
					"error": "Failed to generate CSRF token",
				})
			}
			return c.Next()
		}

		// Validate CSRF token for unsafe methods
		if !c.ValidateCSRF() {
			return c.JSON(http.StatusForbidden, map[string]string{
				"error":   "CSRF token mismatch",
				"message": "The request could not be authenticated",
			})
		}

		return c.Next()
	}
}

// XSSProtectionMiddleware adds XSS protection headers
func XSSProtectionMiddleware() MiddlewareFunc {
	return func(c *Context) error {
		config := GetSecurityConfig()
		
		if config.XSSProtection {
			c.Header("X-XSS-Protection", "1; mode=block")
		}
		
		if config.ContentTypeOptions {
			c.Header("X-Content-Type-Options", "nosniff")
		}
		
		if config.FrameOptions != "" {
			c.Header("X-Frame-Options", config.FrameOptions)
		}

		return c.Next()
	}
}

// SecurityHeadersMiddleware adds comprehensive security headers
func SecurityHeadersMiddleware() MiddlewareFunc {
	return func(c *Context) error {
		config := GetSecurityConfig()

		// HSTS Header
		if config.HSTS.Enabled {
			hstsValue := fmt.Sprintf("max-age=%d", config.HSTS.MaxAge)
			if config.HSTS.IncludeSubDomains {
				hstsValue += "; includeSubDomains"
			}
			if config.HSTS.Preload {
				hstsValue += "; preload"
			}
			c.Header("Strict-Transport-Security", hstsValue)
		}

		// Content Security Policy
		if config.CSP.Enabled {
			cspValue := buildCSPHeader(&config.CSP)
			c.Header("Content-Security-Policy", cspValue)
		}

		// Referrer Policy
		if config.ReferrerPolicy != "" {
			c.Header("Referrer-Policy", config.ReferrerPolicy)
		}

		// Additional security headers
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", config.FrameOptions)
		c.Header("X-XSS-Protection", "1; mode=block")

		return c.Next()
	}
}

// EnhancedCORSMiddleware provides CORS support with comprehensive configuration
func EnhancedCORSMiddleware() MiddlewareFunc {
	return func(c *Context) error {
		config := GetSecurityConfig()
		if !config.CORSEnabled {
			return c.Next()
		}

		origin := c.GetHeader("Origin")
		
		// Check if origin is allowed
		allowedOrigin := ""
		for _, allowed := range config.CORSAllowedOrigins {
			if allowed == "*" || allowed == origin {
				allowedOrigin = allowed
				break
			}
		}
		
		if allowedOrigin != "" {
			if allowedOrigin == "*" {
				c.Header("Access-Control-Allow-Origin", "*")
			} else {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Vary", "Origin")
			}
		}

		// Set other CORS headers
		if len(config.CORSAllowedMethods) > 0 {
			c.Header("Access-Control-Allow-Methods", strings.Join(config.CORSAllowedMethods, ", "))
		}

		if len(config.CORSAllowedHeaders) > 0 {
			c.Header("Access-Control-Allow-Headers", strings.Join(config.CORSAllowedHeaders, ", "))
		}

		if len(config.CORSExposedHeaders) > 0 {
			c.Header("Access-Control-Expose-Headers", strings.Join(config.CORSExposedHeaders, ", "))
		}

		if config.CORSMaxAge > 0 {
			c.Header("Access-Control-Max-Age", strconv.Itoa(config.CORSMaxAge))
		}

		if config.CORSAllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.Status(http.StatusNoContent)
			c.Abort()
			return nil
		}

		return c.Next()
	}
}

// SQLInjectionProtectionMiddleware provides additional SQL injection protection
func SQLInjectionProtectionMiddleware() MiddlewareFunc {
	return func(c *Context) error {
		// SQL injection patterns to detect
		sqlPatterns := []string{
			`(?i)(union\s+select)`,
			`(?i)(insert\s+into)`,
			`(?i)(delete\s+from)`,
			`(?i)(drop\s+table)`,
			`(?i)(truncate\s+table)`,
			`(?i)(update\s+\w+\s+set)`,
			`(?i)(exec\s*\()`,
			`(?i)(sp_executesql)`,
			`(?i)(xp_cmdshell)`,
			`(?i)(\'\s*;\s*drop)`,
			`(?i)(\'\s*;\s*delete)`,
			`(?i)(\'\s*;\s*insert)`,
			`(?i)(\'\s*;\s*update)`,
			`(?i)(--\s*$)`,
			`(?i)(/\*.*\*/)`,
		}

		if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
			c.Request.ParseForm()

			for _, values := range c.Request.PostForm {
				for _, value := range values {
					for _, pattern := range sqlPatterns {
						if matched, _ := regexp.MatchString(pattern, value); matched {
							Warn("Potential SQL injection attempt detected", map[string]interface{}{
								"ip":      c.RemoteIP(),
								"url":     c.Request.URL.String(),
								"method":  c.Request.Method,
								"pattern": pattern,
								"value":   value,
							})
							
							return c.JSON(http.StatusBadRequest, map[string]string{
								"error":   "Invalid input detected",
								"message": "The request contains potentially harmful content",
							})
						}
					}
				}
			}
		}

		return c.Next()
	}
}

// RateLimitingSecurityMiddleware provides security-focused rate limiting
func RateLimitingSecurityMiddleware() MiddlewareFunc {
	return func(c *Context) error {
		config := GetSecurityConfig()
		
		// Create a security-focused rate limiter for login attempts
		if strings.Contains(c.Request.URL.Path, "/login") || 
		   strings.Contains(c.Request.URL.Path, "/auth") {
			
			// Use IP-based rate limiting for authentication endpoints
			key := fmt.Sprintf("auth_attempts:%s", c.RemoteIP())
			limiter := NewMemoryRateLimiter("token_bucket")
			
			result, err := limiter.Allow(c.Request.Context(), key, config.LoginRateLimit, config.FailedLoginWindow)
			if err != nil {
				Error("Rate limiting error", map[string]interface{}{
					"error": err.Error(),
					"ip":    c.RemoteIP(),
				})
				return c.Next() // Continue on error
			}
			
			if !result.Allowed {
				Warn("Rate limit exceeded for authentication", map[string]interface{}{
					"ip":        c.RemoteIP(),
					"remaining": result.Remaining,
					"reset":     result.ResetTime,
				})
				
				c.Header("Retry-After", strconv.Itoa(int(result.RetryAfter.Seconds())))
				return c.JSON(http.StatusTooManyRequests, map[string]interface{}{
					"error":       "Too many authentication attempts",
					"message":     "Please wait before trying again",
					"retry_after": int(result.RetryAfter.Seconds()),
				})
			}
		}

		return c.Next()
	}
}

// SecurityLoggerMiddleware logs security-related events
func SecurityLoggerMiddleware() MiddlewareFunc {
	return func(c *Context) error {
		start := time.Now()
		
		// Log potentially suspicious requests
		if isSuspiciousRequest(c) {
			Warn("Suspicious request detected", map[string]interface{}{
				"ip":         c.RemoteIP(),
				"user_agent": c.UserAgent(),
				"method":     c.Request.Method,
				"url":        c.Request.URL.String(),
				"headers":    c.Request.Header,
			})
		}

		err := c.Next()
		
		// Note: Status tracking would need to be implemented in Context
		// For now, we'll log all requests and let the error handler log failures
		_ = time.Since(start) // Track duration but don't use it yet

		return err
	}
}

// Helper functions

// buildCSPHeader builds Content Security Policy header value
func buildCSPHeader(csp *CSPConfig) string {
	var parts []string

	if len(csp.DefaultSrc) > 0 {
		parts = append(parts, fmt.Sprintf("default-src %s", strings.Join(csp.DefaultSrc, " ")))
	}
	if len(csp.ScriptSrc) > 0 {
		parts = append(parts, fmt.Sprintf("script-src %s", strings.Join(csp.ScriptSrc, " ")))
	}
	if len(csp.StyleSrc) > 0 {
		parts = append(parts, fmt.Sprintf("style-src %s", strings.Join(csp.StyleSrc, " ")))
	}
	if len(csp.ImgSrc) > 0 {
		parts = append(parts, fmt.Sprintf("img-src %s", strings.Join(csp.ImgSrc, " ")))
	}
	if len(csp.ConnectSrc) > 0 {
		parts = append(parts, fmt.Sprintf("connect-src %s", strings.Join(csp.ConnectSrc, " ")))
	}
	if len(csp.FontSrc) > 0 {
		parts = append(parts, fmt.Sprintf("font-src %s", strings.Join(csp.FontSrc, " ")))
	}
	if len(csp.ObjectSrc) > 0 {
		parts = append(parts, fmt.Sprintf("object-src %s", strings.Join(csp.ObjectSrc, " ")))
	}
	if len(csp.MediaSrc) > 0 {
		parts = append(parts, fmt.Sprintf("media-src %s", strings.Join(csp.MediaSrc, " ")))
	}
	if len(csp.FrameSrc) > 0 {
		parts = append(parts, fmt.Sprintf("frame-src %s", strings.Join(csp.FrameSrc, " ")))
	}
	if csp.ReportURI != "" {
		parts = append(parts, fmt.Sprintf("report-uri %s", csp.ReportURI))
	}

	return strings.Join(parts, "; ")
}

// isSuspiciousRequest checks if a request looks suspicious
func isSuspiciousRequest(c *Context) bool {
	userAgent := c.UserAgent()
	url := c.Request.URL.String()

	// Check for common attack patterns
	suspiciousPatterns := []string{
		"sqlmap",
		"nikto",
		"nmap",
		"masscan",
		"acunetix",
		"nessus",
		"openvas",
		"w3af",
		"burp",
		"zap",
		"dirbuster",
		"gobuster",
		"wfuzz",
	}

	userAgentLower := strings.ToLower(userAgent)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(userAgentLower, pattern) {
			return true
		}
	}

	// Check for suspicious URL patterns
	urlLower := strings.ToLower(url)
	suspiciousURLPatterns := []string{
		"../",
		"..\\",
		"/etc/passwd",
		"/proc/",
		"cmd=",
		"exec=",
		"<script",
		"javascript:",
		"vbscript:",
		"onload=",
		"onerror=",
	}

	for _, pattern := range suspiciousURLPatterns {
		if strings.Contains(urlLower, pattern) {
			return true
		}
	}

	return false
}

// SecurityMiddlewareGroup returns a group of security middleware for common use
func SecurityMiddlewareGroup() []MiddlewareFunc {
	return []MiddlewareFunc{
		SecurityLoggerMiddleware(),
		TrimStringsMiddleware(),
		ConvertEmptyStringsToNullMiddleware(),
		InputSanitizationMiddleware(),
		SQLInjectionProtectionMiddleware(),
		XSSProtectionMiddleware(),
		SecurityHeadersMiddleware(),
		EnhancedCORSMiddleware(),
		RateLimitingSecurityMiddleware(),
	}
}

// WebSecurityMiddlewareGroup returns security middleware for web routes (includes CSRF)
func WebSecurityMiddlewareGroup() []MiddlewareFunc {
	middleware := SecurityMiddlewareGroup()
	// Insert CSRF protection after input sanitization
	return append(middleware[:4], append([]MiddlewareFunc{CSRFProtectionMiddleware()}, middleware[4:]...)...)
}

// APISecurityMiddlewareGroup returns security middleware for API routes (excludes CSRF)
func APISecurityMiddlewareGroup() []MiddlewareFunc {
	return SecurityMiddlewareGroup()
}