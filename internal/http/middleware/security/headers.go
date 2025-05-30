package security

import (
	"fmt"
	"strings"

	httpInternal "github.com/onyx-go/framework/internal/http"
)

// XSSProtectionMiddleware adds XSS protection headers
func (f *MiddlewareFactory) XSSProtectionMiddleware() httpInternal.MiddlewareFunc {
	return func(c httpInternal.Context) error {
		if f.deps.Config.XSSProtection() {
			c.SetHeader("X-XSS-Protection", "1; mode=block")
		}
		
		if f.deps.Config.ContentTypeOptions() {
			c.SetHeader("X-Content-Type-Options", "nosniff")
		}
		
		frameOptions := f.deps.Config.FrameOptions()
		if frameOptions != "" {
			c.SetHeader("X-Frame-Options", frameOptions)
		}

		return c.Next()
	}
}

// SecurityHeadersMiddleware adds comprehensive security headers
func (f *MiddlewareFactory) SecurityHeadersMiddleware() httpInternal.MiddlewareFunc {
	return func(c httpInternal.Context) error {
		// HSTS Header
		hstsConfig := f.deps.Config.HSTS()
		if hstsConfig.Enabled() {
			hstsValue := fmt.Sprintf("max-age=%d", hstsConfig.MaxAge())
			if hstsConfig.IncludeSubDomains() {
				hstsValue += "; includeSubDomains"
			}
			if hstsConfig.Preload() {
				hstsValue += "; preload"
			}
			c.SetHeader("Strict-Transport-Security", hstsValue)
		}

		// Content Security Policy
		cspConfig := f.deps.Config.CSP()
		if cspConfig.Enabled() {
			cspValue := f.buildCSPHeader(cspConfig)
			c.SetHeader("Content-Security-Policy", cspValue)
		}

		// Referrer Policy
		referrerPolicy := f.deps.Config.ReferrerPolicy()
		if referrerPolicy != "" {
			c.SetHeader("Referrer-Policy", referrerPolicy)
		}

		// Additional security headers
		c.SetHeader("X-Content-Type-Options", "nosniff")
		
		frameOptions := f.deps.Config.FrameOptions()
		if frameOptions != "" {
			c.SetHeader("X-Frame-Options", frameOptions)
		}
		
		if f.deps.Config.XSSProtection() {
			c.SetHeader("X-XSS-Protection", "1; mode=block")
		}

		return c.Next()
	}
}

// buildCSPHeader constructs a Content Security Policy header string
func (f *MiddlewareFactory) buildCSPHeader(csp CSPConfig) string {
	var parts []string

	if len(csp.DefaultSrc()) > 0 {
		parts = append(parts, fmt.Sprintf("default-src %s", strings.Join(csp.DefaultSrc(), " ")))
	}
	if len(csp.ScriptSrc()) > 0 {
		parts = append(parts, fmt.Sprintf("script-src %s", strings.Join(csp.ScriptSrc(), " ")))
	}
	if len(csp.StyleSrc()) > 0 {
		parts = append(parts, fmt.Sprintf("style-src %s", strings.Join(csp.StyleSrc(), " ")))
	}
	if len(csp.ImgSrc()) > 0 {
		parts = append(parts, fmt.Sprintf("img-src %s", strings.Join(csp.ImgSrc(), " ")))
	}
	if len(csp.ConnectSrc()) > 0 {
		parts = append(parts, fmt.Sprintf("connect-src %s", strings.Join(csp.ConnectSrc(), " ")))
	}
	if len(csp.FontSrc()) > 0 {
		parts = append(parts, fmt.Sprintf("font-src %s", strings.Join(csp.FontSrc(), " ")))
	}
	if len(csp.ObjectSrc()) > 0 {
		parts = append(parts, fmt.Sprintf("object-src %s", strings.Join(csp.ObjectSrc(), " ")))
	}
	if len(csp.MediaSrc()) > 0 {
		parts = append(parts, fmt.Sprintf("media-src %s", strings.Join(csp.MediaSrc(), " ")))
	}
	if len(csp.FrameSrc()) > 0 {
		parts = append(parts, fmt.Sprintf("frame-src %s", strings.Join(csp.FrameSrc(), " ")))
	}
	if csp.ReportURI() != "" {
		parts = append(parts, fmt.Sprintf("report-uri %s", csp.ReportURI()))
	}

	return strings.Join(parts, "; ")
}

// Convenience functions for backward compatibility

// XSSProtectionMiddleware creates an XSS protection middleware with default dependencies
func XSSProtectionMiddleware(deps *Dependencies) httpInternal.MiddlewareFunc {
	factory := NewMiddlewareFactory(deps)
	return factory.XSSProtectionMiddleware()
}

// SecurityHeadersMiddleware creates a security headers middleware with default dependencies
func SecurityHeadersMiddleware(deps *Dependencies) httpInternal.MiddlewareFunc {
	factory := NewMiddlewareFactory(deps)
	return factory.SecurityHeadersMiddleware()
}