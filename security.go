package onyx

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"
	"unicode"
)

// Security configuration and interfaces

// SecurityConfig contains security-related configuration
type SecurityConfig struct {
	// Input sanitization
	TrimStrings            bool `json:"trim_strings"`
	ConvertEmptyToNull     bool `json:"convert_empty_to_null"`
	StripTags              bool `json:"strip_tags"`
	MaxInputLength         int  `json:"max_input_length"`
	
	// XSS Protection
	XSSProtection          bool `json:"xss_protection"`
	ContentTypeOptions     bool `json:"content_type_options"`
	FrameOptions           string `json:"frame_options"`
	
	// CSRF Protection
	CSRFProtection         bool   `json:"csrf_protection"`
	CSRFTokenName          string `json:"csrf_token_name"`
	CSRFSessionKey         string `json:"csrf_session_key"`
	CSRFTokenExpiry        time.Duration `json:"csrf_token_expiry"`
	
	// CORS Configuration
	CORSEnabled            bool     `json:"cors_enabled"`
	CORSAllowedOrigins     []string `json:"cors_allowed_origins"`
	CORSAllowedMethods     []string `json:"cors_allowed_methods"`
	CORSAllowedHeaders     []string `json:"cors_allowed_headers"`
	CORSExposedHeaders     []string `json:"cors_exposed_headers"`
	CORSMaxAge             int      `json:"cors_max_age"`
	CORSAllowCredentials   bool     `json:"cors_allow_credentials"`
	
	// Security Headers
	HSTS                   HSTSConfig `json:"hsts"`
	CSP                    CSPConfig  `json:"csp"`
	ReferrerPolicy         string     `json:"referrer_policy"`
	
	// Rate limiting for security
	LoginRateLimit         int `json:"login_rate_limit"`
	FailedLoginWindow      time.Duration `json:"failed_login_window"`
}

// HSTSConfig for HTTP Strict Transport Security
type HSTSConfig struct {
	Enabled           bool `json:"enabled"`
	MaxAge            int  `json:"max_age"`
	IncludeSubDomains bool `json:"include_subdomains"`
	Preload           bool `json:"preload"`
}

// CSPConfig for Content Security Policy
type CSPConfig struct {
	Enabled      bool     `json:"enabled"`
	DefaultSrc   []string `json:"default_src"`
	ScriptSrc    []string `json:"script_src"`
	StyleSrc     []string `json:"style_src"`
	ImgSrc       []string `json:"img_src"`
	ConnectSrc   []string `json:"connect_src"`
	FontSrc      []string `json:"font_src"`
	ObjectSrc    []string `json:"object_src"`
	MediaSrc     []string `json:"media_src"`
	FrameSrc     []string `json:"frame_src"`
	ReportURI    string   `json:"report_uri"`
}

// Security-specific validation rules that extend the existing validation system

// SecurityValidationRule interface for security-focused validation rules
type SecurityValidationRule interface {
	Validate(value interface{}) error
	Message() string
}

// SecurityValidatorFunc is a function type for security validation
type SecurityValidatorFunc func(value interface{}) error

// Security-specific validation rules (extend existing validation system)

// NoScriptRule validates that input doesn't contain script tags
type NoScriptRule struct{}

func (r *NoScriptRule) Validate(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return nil // Only validate strings
	}
	
	scriptRegex := regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`)
	if scriptRegex.MatchString(str) {
		return fmt.Errorf("script tags are not allowed")
	}
	
	return nil
}

func (r *NoScriptRule) Message() string {
	return "Script tags are not allowed"
}

// NoSQLInjectionRule validates that input doesn't contain SQL injection patterns
type NoSQLInjectionRule struct{}

func (r *NoSQLInjectionRule) Validate(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return nil // Only validate strings
	}
	
	sqlPatterns := []string{
		`(?i)(union\s+select)`,
		`(?i)(insert\s+into)`,
		`(?i)(delete\s+from)`,
		`(?i)(drop\s+table)`,
		`(?i)(\'\s*;\s*drop)`,
		`(?i)(--\s*$)`,
	}
	
	for _, pattern := range sqlPatterns {
		if matched, _ := regexp.MatchString(pattern, str); matched {
			return fmt.Errorf("potentially harmful SQL detected")
		}
	}
	
	return nil
}

func (r *NoSQLInjectionRule) Message() string {
	return "Input contains potentially harmful SQL patterns"
}

// Input sanitization functions

// SanitizeString removes potentially dangerous characters from input
func SanitizeString(input string) string {
	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")
	
	// Trim whitespace
	input = strings.TrimSpace(input)
	
	return input
}

// StripTags removes HTML and PHP tags from string
func StripTags(input string) string {
	// Remove HTML tags
	htmlRegex := regexp.MustCompile(`<[^>]*>`)
	input = htmlRegex.ReplaceAllString(input, "")
	
	// Remove PHP tags
	phpRegex := regexp.MustCompile(`<\?(?:php)?[\s\S]*?\?>`)
	input = phpRegex.ReplaceAllString(input, "")
	
	return input
}

// EscapeHTML escapes HTML special characters
func EscapeHTML(input string) string {
	return html.EscapeString(input)
}

// EscapeJS escapes JavaScript special characters
func EscapeJS(input string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"'", "\\'",
		"\"", "\\\"",
		"\n", "\\n",
		"\r", "\\r",
		"\t", "\\t",
		"<", "\\u003c",
		">", "\\u003e",
		"&", "\\u0026",
	)
	return replacer.Replace(input)
}

// EscapeCSS escapes CSS special characters
func EscapeCSS(input string) string {
	var result strings.Builder
	for _, r := range input {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			result.WriteRune(r)
		} else {
			result.WriteString(fmt.Sprintf("\\%06x", r))
		}
	}
	return result.String()
}

// EscapeURL encodes URL components
func EscapeURL(input string) string {
	return strings.ReplaceAll(strings.ReplaceAll(input, " ", "%20"), "&", "%26")
}

// CSRF Protection

// CSRFManager handles CSRF token generation and validation
type CSRFManager struct {
	config *SecurityConfig
}

// NewCSRFManager creates a new CSRF manager
func NewCSRFManager(config *SecurityConfig) *CSRFManager {
	return &CSRFManager{config: config}
}

// GenerateToken generates a new CSRF token
func (cm *CSRFManager) GenerateToken() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// ValidateToken validates a CSRF token
func (cm *CSRFManager) ValidateToken(token string, sessionToken string) bool {
	if token == "" || sessionToken == "" {
		return false
	}
	return token == sessionToken
}

// SetCSRFToken sets CSRF token in context
func SetCSRFToken(c Context) error {
	manager := NewCSRFManager(GetSecurityConfig())
	token, err := manager.GenerateToken()
	if err != nil {
		return err
	}
	
	// Use context storage instead of session for now
	// TODO: Implement session support with interface-based system
	c.Set("csrf_token", token)
	return nil
}

// GetCSRFTokenFromSecurity gets CSRF token from context (security module)
func GetCSRFTokenFromSecurity(c Context) string {
	if token, exists := c.Get("csrf_token"); exists {
		if str, ok := token.(string); ok {
			return str
		}
	}
	
	// TODO: Add session support when available
	return ""
}

// ValidateCSRF validates CSRF token from request
func ValidateCSRF(c Context) bool {
	manager := NewCSRFManager(GetSecurityConfig())
	
	// Get token from header for now (form parsing needs additional work)
	// TODO: Implement form parsing with interface-based system
	token := c.Header("X-CSRF-Token")
	if token == "" {
		token = c.Query("_token")
	}
	
	sessionToken := GetCSRFTokenFromSecurity(c)
	return manager.ValidateToken(token, sessionToken)
}

// Security configuration management

var globalSecurityConfig *SecurityConfig

// GetSecurityConfig returns the global security configuration
func GetSecurityConfig() *SecurityConfig {
	if globalSecurityConfig == nil {
		globalSecurityConfig = DefaultSecurityConfig()
	}
	return globalSecurityConfig
}

// SetSecurityConfig sets the global security configuration
func SetSecurityConfig(config *SecurityConfig) {
	globalSecurityConfig = config
}

// DefaultSecurityConfig returns default security configuration
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		TrimStrings:        true,
		ConvertEmptyToNull: true,
		StripTags:          false,
		MaxInputLength:     1024 * 1024, // 1MB
		
		XSSProtection:      true,
		ContentTypeOptions: true,
		FrameOptions:       "DENY",
		
		CSRFProtection:     true,
		CSRFTokenName:      "_token",
		CSRFSessionKey:     "csrf_token",
		CSRFTokenExpiry:    time.Hour * 2,
		
		CORSEnabled:        false,
		CORSAllowedOrigins: []string{"*"},
		CORSAllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		CORSAllowedHeaders: []string{"Content-Type", "Authorization", "X-Requested-With"},
		CORSExposedHeaders: []string{},
		CORSMaxAge:         86400, // 24 hours
		CORSAllowCredentials: false,
		
		HSTS: HSTSConfig{
			Enabled:           true,
			MaxAge:            31536000, // 1 year
			IncludeSubDomains: true,
			Preload:           false,
		},
		
		CSP: CSPConfig{
			Enabled:    true,
			DefaultSrc: []string{"'self'"},
			ScriptSrc:  []string{"'self'", "'unsafe-inline'"},
			StyleSrc:   []string{"'self'", "'unsafe-inline'"},
			ImgSrc:     []string{"'self'", "data:"},
			ConnectSrc: []string{"'self'"},
			FontSrc:    []string{"'self'"},
			ObjectSrc:  []string{"'none'"},
			MediaSrc:   []string{"'self'"},
			FrameSrc:   []string{"'none'"},
		},
		
		ReferrerPolicy:    "strict-origin-when-cross-origin",
		LoginRateLimit:    5,
		FailedLoginWindow: time.Minute * 15,
	}
}

// Security helper functions for validation

// ValidateNoScript checks if input contains script tags
func ValidateNoScript(input string) bool {
	rule := &NoScriptRule{}
	return rule.Validate(input) == nil
}

// ValidateNoSQLInjection checks if input contains SQL injection patterns
func ValidateNoSQLInjection(input string) bool {
	rule := &NoSQLInjectionRule{}
	return rule.Validate(input) == nil
}