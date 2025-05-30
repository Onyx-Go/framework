package security

import (
	httpInternal "github.com/onyx-go/framework/internal/http"
)

// SecurityMiddlewareGroup returns a group of security middleware for common use
func (f *MiddlewareFactory) SecurityMiddlewareGroup() MiddlewareGroup {
	return MiddlewareGroup{
		f.SecurityLoggerMiddleware(),
		f.TrimStringsMiddleware(),
		f.ConvertEmptyStringsToNullMiddleware(),
		f.InputSanitizationMiddleware(),
		f.SQLInjectionProtectionMiddleware(),
		f.XSSProtectionMiddleware(),
		f.SecurityHeadersMiddleware(),
		f.CORSMiddleware(),
		f.RateLimitingSecurityMiddleware(),
	}
}

// WebSecurityMiddlewareGroup returns security middleware for web routes (includes CSRF)
func (f *MiddlewareFactory) WebSecurityMiddlewareGroup() MiddlewareGroup {
	middleware := f.SecurityMiddlewareGroup()
	
	// Insert CSRF protection after input sanitization (index 3)
	result := make(MiddlewareGroup, 0, len(middleware)+1)
	result = append(result, middleware[:4]...)
	result = append(result, f.CSRFMiddleware())
	result = append(result, middleware[4:]...)
	
	return result
}

// APISecurityMiddlewareGroup returns security middleware for API routes (excludes CSRF)
func (f *MiddlewareFactory) APISecurityMiddlewareGroup() MiddlewareGroup {
	return f.SecurityMiddlewareGroup() // Same as base group, no CSRF for APIs
}

// MinimalSecurityMiddlewareGroup returns basic security middleware for lightweight routes
func (f *MiddlewareFactory) MinimalSecurityMiddlewareGroup() MiddlewareGroup {
	return MiddlewareGroup{
		f.SecurityLoggerMiddleware(),
		f.XSSProtectionMiddleware(),
		f.SecurityHeadersMiddleware(),
		f.RateLimitingSecurityMiddleware(),
	}
}

// HighSecurityMiddlewareGroup returns comprehensive security middleware for sensitive routes
func (f *MiddlewareFactory) HighSecurityMiddlewareGroup() MiddlewareGroup {
	middleware := f.WebSecurityMiddlewareGroup()
	
	// Add additional security measures
	result := make(MiddlewareGroup, 0, len(middleware)+1)
	result = append(result, middleware...)
	result = append(result, f.XSSProtectionValidator()) // Additional XSS validation
	
	return result
}

// Convert MiddlewareGroup to slice for backward compatibility
func (mg MiddlewareGroup) ToSlice() []httpInternal.MiddlewareFunc {
	return []httpInternal.MiddlewareFunc(mg)
}

// Convenience functions for creating middleware groups with default dependencies

// SecurityMiddlewareGroup returns a group of security middleware for common use
func SecurityMiddlewareGroup(deps *Dependencies) []httpInternal.MiddlewareFunc {
	factory := NewMiddlewareFactory(deps)
	return factory.SecurityMiddlewareGroup().ToSlice()
}

// WebSecurityMiddlewareGroup returns security middleware for web routes (includes CSRF)
func WebSecurityMiddlewareGroup(deps *Dependencies) []httpInternal.MiddlewareFunc {
	factory := NewMiddlewareFactory(deps)
	return factory.WebSecurityMiddlewareGroup().ToSlice()
}

// APISecurityMiddlewareGroup returns security middleware for API routes (excludes CSRF)
func APISecurityMiddlewareGroup(deps *Dependencies) []httpInternal.MiddlewareFunc {
	factory := NewMiddlewareFactory(deps)
	return factory.APISecurityMiddlewareGroup().ToSlice()
}

// MinimalSecurityMiddlewareGroup returns basic security middleware for lightweight routes
func MinimalSecurityMiddlewareGroup(deps *Dependencies) []httpInternal.MiddlewareFunc {
	factory := NewMiddlewareFactory(deps)
	return factory.MinimalSecurityMiddlewareGroup().ToSlice()
}

// HighSecurityMiddlewareGroup returns comprehensive security middleware for sensitive routes
func HighSecurityMiddlewareGroup(deps *Dependencies) []httpInternal.MiddlewareFunc {
	factory := NewMiddlewareFactory(deps)
	return factory.HighSecurityMiddlewareGroup().ToSlice()
}