package security

import (
	"strings"

	httpInternal "github.com/onyx-go/framework/internal/http"
)

// SecurityLoggerMiddleware logs security-related events and suspicious requests
func (f *MiddlewareFactory) SecurityLoggerMiddleware() httpInternal.MiddlewareFunc {
	return func(c httpInternal.Context) error {
		// Check for suspicious requests before processing
		if f.isSuspiciousRequest(c) {
			f.logSuspiciousRequest(c)
		}

		// Continue with request processing
		err := c.Next()

		// Log security events after processing if needed
		if err != nil {
			f.logSecurityError(c, err)
		}

		return err
	}
}

// isSuspiciousRequest checks if a request looks suspicious
func (f *MiddlewareFactory) isSuspiciousRequest(c httpInternal.Context) bool {
	userAgent := c.Header("User-Agent")
	url := c.URL()

	// Check for common attack patterns in user agent
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
		"curl", // Can be suspicious in certain contexts
		"wget",
		"python-requests",
		"bot",
		"spider",
		"crawler",
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
		"../",           // Directory traversal
		"..\\",          // Directory traversal (Windows)
		"/etc/passwd",   // Unix system file access
		"/proc/",        // Unix process information
		"cmd=",          // Command injection
		"exec=",         // Command execution
		"<script",       // XSS attempt
		"javascript:",   // XSS attempt
		"vbscript:",     // XSS attempt
		"onload=",       // XSS attempt
		"onerror=",      // XSS attempt
		"union+select",  // SQL injection
		"drop+table",    // SQL injection
		"select+from",   // SQL injection
		"insert+into",   // SQL injection
		"/admin",        // Admin panel probing
		"/wp-admin",     // WordPress admin probing
		"/phpmyadmin",   // Database admin probing
		"/.env",         // Environment file access
		"/.git",         // Git repository access
		"/config.php",   // Config file access
		"/config.ini",   // Config file access
	}

	for _, pattern := range suspiciousURLPatterns {
		if strings.Contains(urlLower, pattern) {
			return true
		}
	}

	// Check for empty or very short user agents (often bots)
	if len(strings.TrimSpace(userAgent)) < 10 {
		return true
	}

	// Check for unusual HTTP methods
	method := c.Method()
	if method != "GET" && method != "POST" && method != "PUT" && 
	   method != "DELETE" && method != "PATCH" && method != "HEAD" && 
	   method != "OPTIONS" {
		return true
	}

	return false
}

// logSuspiciousRequest logs a suspicious request
func (f *MiddlewareFactory) logSuspiciousRequest(c httpInternal.Context) {
	context := map[string]interface{}{
		"ip":         c.RemoteIP(),
		"method":     c.Method(),
		"url":        c.URL(),
		"user_agent": c.Header("User-Agent"),
		"referer":    c.Header("Referer"),
		"event":      "suspicious_request",
	}
	
	f.deps.Logger.Warn("Suspicious request detected", context)
}

// logSecurityError logs security-related errors
func (f *MiddlewareFactory) logSecurityError(c httpInternal.Context, err error) {
	context := map[string]interface{}{
		"ip":         c.RemoteIP(),
		"method":     c.Method(),
		"url":        c.URL(),
		"user_agent": c.Header("User-Agent"),
		"error":      err.Error(),
		"event":      "security_error",
	}
	
	f.deps.Logger.Error("Security middleware error", context)
}

// SecurityLoggerMiddleware creates a security logger middleware with default dependencies
func SecurityLoggerMiddleware(deps *Dependencies) httpInternal.MiddlewareFunc {
	factory := NewMiddlewareFactory(deps)
	return factory.SecurityLoggerMiddleware()
}