package security

import (
	httpInternal "github.com/onyx-go/framework/internal/http"
)

// SecurityConfig defines the configuration for security middleware
type SecurityConfig interface {
	// Input sanitization settings
	TrimStrings() bool
	ConvertEmptyToNull() bool
	StripTags() bool
	MaxInputLength() int
	
	// CSRF protection
	CSRFProtection() bool
	
	// XSS protection
	XSSProtection() bool
	ContentTypeOptions() bool
	FrameOptions() string
	
	// HSTS settings
	HSTS() HSTSConfig
	
	// CSP settings  
	CSP() CSPConfig
	
	// CORS settings
	CORSEnabled() bool
	CORSOrigins() []string
	CORSMethods() []string
	CORSHeaders() []string
	CORSCredentials() bool
	CORSMaxAge() int
	
	// Rate limiting
	RateLimitEnabled() bool
	
	// General settings
	ReferrerPolicy() string
}

// HSTSConfig defines HSTS configuration
type HSTSConfig interface {
	Enabled() bool
	MaxAge() int
	IncludeSubDomains() bool
	Preload() bool
}

// CSPConfig defines Content Security Policy configuration
type CSPConfig interface {
	Enabled() bool
	DefaultSrc() []string
	ScriptSrc() []string
	StyleSrc() []string
	ImgSrc() []string
	ConnectSrc() []string
	FontSrc() []string
	ObjectSrc() []string
	MediaSrc() []string
	FrameSrc() []string
	ReportURI() string
}

// SecurityLogger defines the interface for security-related logging
type SecurityLogger interface {
	Warn(message string, context map[string]interface{})
	Error(message string, context map[string]interface{})
}

// RateLimiter defines the interface for rate limiting
type RateLimiter interface {
	Allow(key string) bool
}

// Sanitizer defines the interface for input sanitization
type Sanitizer interface {
	SanitizeString(input string) string
	StripTags(input string) string
}

// Dependencies holds all external dependencies for security middleware
type Dependencies struct {
	Config     SecurityConfig
	Logger     SecurityLogger
	Limiter    RateLimiter
	Sanitizer  Sanitizer
}

// MiddlewareFactory creates security middleware with injected dependencies
type MiddlewareFactory struct {
	deps *Dependencies
}

// NewMiddlewareFactory creates a new middleware factory with dependencies
func NewMiddlewareFactory(deps *Dependencies) *MiddlewareFactory {
	return &MiddlewareFactory{deps: deps}
}

// MiddlewareGroup represents a collection of security middleware
type MiddlewareGroup []httpInternal.MiddlewareFunc