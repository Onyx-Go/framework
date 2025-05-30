package security

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// Mock implementations for testing

type mockSecurityConfig struct {
	trimStrings         bool
	convertEmptyToNull  bool
	stripTags           bool
	maxInputLength      int
	csrfProtection      bool
	xssProtection       bool
	contentTypeOptions  bool
	frameOptions        string
	hstsConfig          *mockHSTSConfig
	cspConfig           *mockCSPConfig
	corsEnabled         bool
	corsOrigins         []string
	corsMethods         []string
	corsHeaders         []string
	corsCredentials     bool
	corsMaxAge          int
	rateLimitEnabled    bool
	referrerPolicy      string
}

func (m *mockSecurityConfig) TrimStrings() bool         { return m.trimStrings }
func (m *mockSecurityConfig) ConvertEmptyToNull() bool { return m.convertEmptyToNull }
func (m *mockSecurityConfig) StripTags() bool          { return m.stripTags }
func (m *mockSecurityConfig) MaxInputLength() int      { return m.maxInputLength }
func (m *mockSecurityConfig) CSRFProtection() bool     { return m.csrfProtection }
func (m *mockSecurityConfig) XSSProtection() bool      { return m.xssProtection }
func (m *mockSecurityConfig) ContentTypeOptions() bool { return m.contentTypeOptions }
func (m *mockSecurityConfig) FrameOptions() string     { return m.frameOptions }
func (m *mockSecurityConfig) HSTS() HSTSConfig         { return m.hstsConfig }
func (m *mockSecurityConfig) CSP() CSPConfig           { return m.cspConfig }
func (m *mockSecurityConfig) CORSEnabled() bool        { return m.corsEnabled }
func (m *mockSecurityConfig) CORSOrigins() []string    { return m.corsOrigins }
func (m *mockSecurityConfig) CORSMethods() []string    { return m.corsMethods }
func (m *mockSecurityConfig) CORSHeaders() []string    { return m.corsHeaders }
func (m *mockSecurityConfig) CORSCredentials() bool    { return m.corsCredentials }
func (m *mockSecurityConfig) CORSMaxAge() int          { return m.corsMaxAge }
func (m *mockSecurityConfig) RateLimitEnabled() bool   { return m.rateLimitEnabled }
func (m *mockSecurityConfig) ReferrerPolicy() string   { return m.referrerPolicy }

type mockHSTSConfig struct {
	enabled           bool
	maxAge            int
	includeSubDomains bool
	preload           bool
}

func (m *mockHSTSConfig) Enabled() bool           { return m.enabled }
func (m *mockHSTSConfig) MaxAge() int             { return m.maxAge }
func (m *mockHSTSConfig) IncludeSubDomains() bool { return m.includeSubDomains }
func (m *mockHSTSConfig) Preload() bool           { return m.preload }

type mockCSPConfig struct {
	enabled    bool
	defaultSrc []string
	scriptSrc  []string
	styleSrc   []string
	imgSrc     []string
	connectSrc []string
	fontSrc    []string
	objectSrc  []string
	mediaSrc   []string
	frameSrc   []string
	reportURI  string
}

func (m *mockCSPConfig) Enabled() bool        { return m.enabled }
func (m *mockCSPConfig) DefaultSrc() []string { return m.defaultSrc }
func (m *mockCSPConfig) ScriptSrc() []string  { return m.scriptSrc }
func (m *mockCSPConfig) StyleSrc() []string   { return m.styleSrc }
func (m *mockCSPConfig) ImgSrc() []string     { return m.imgSrc }
func (m *mockCSPConfig) ConnectSrc() []string { return m.connectSrc }
func (m *mockCSPConfig) FontSrc() []string    { return m.fontSrc }
func (m *mockCSPConfig) ObjectSrc() []string  { return m.objectSrc }
func (m *mockCSPConfig) MediaSrc() []string   { return m.mediaSrc }
func (m *mockCSPConfig) FrameSrc() []string   { return m.frameSrc }
func (m *mockCSPConfig) ReportURI() string    { return m.reportURI }

type mockSecurityLogger struct {
	warnings []logEntry
	errors   []logEntry
}

type logEntry struct {
	message string
	context map[string]interface{}
}

func (m *mockSecurityLogger) Warn(message string, context map[string]interface{}) {
	m.warnings = append(m.warnings, logEntry{message, context})
}

func (m *mockSecurityLogger) Error(message string, context map[string]interface{}) {
	m.errors = append(m.errors, logEntry{message, context})
}

type mockRateLimiter struct {
	allowed map[string]bool
}

func (m *mockRateLimiter) Allow(key string) bool {
	if m.allowed == nil {
		return true
	}
	return m.allowed[key]
}

type mockSanitizer struct{}

func (m *mockSanitizer) SanitizeString(input string) string {
	return strings.ReplaceAll(input, "<", "&lt;")
}

func (m *mockSanitizer) StripTags(input string) string {
	return strings.ReplaceAll(input, "<script>", "")
}

type mockContext struct {
	request        *http.Request
	responseWriter http.ResponseWriter
	headers        map[string]string
	statusCode     int
	jsonResponse   interface{}
	data           map[string]interface{}
	nextCalled     bool
}

func newMockContext(req *http.Request, w http.ResponseWriter) *mockContext {
	return &mockContext{
		request:        req,
		responseWriter: w,
		headers:        make(map[string]string),
		data:           make(map[string]interface{}),
	}
}

func (m *mockContext) Method() string           { return m.request.Method }
func (m *mockContext) URL() string              { return m.request.URL.String() }
func (m *mockContext) Path() string             { return m.request.URL.Path }
func (m *mockContext) Query(key string) string  { return m.request.URL.Query().Get(key) }
func (m *mockContext) QueryDefault(key, defaultValue string) string {
	if value := m.Query(key); value != "" {
		return value
	}
	return defaultValue
}
func (m *mockContext) Param(key string) string  { return "" }
func (m *mockContext) Header(key string) string { return m.request.Header.Get(key) }

func (m *mockContext) Status(code int)                { m.statusCode = code }
func (m *mockContext) SetHeader(key, value string)    { m.headers[key] = value }
func (m *mockContext) String(code int, text string) error {
	m.statusCode = code
	return nil
}
func (m *mockContext) JSON(code int, data interface{}) error {
	m.statusCode = code
	m.jsonResponse = data
	return nil
}
func (m *mockContext) HTML(code int, html string) error {
	m.statusCode = code
	return nil
}

func (m *mockContext) Next() error {
	m.nextCalled = true
	return nil
}
func (m *mockContext) Abort()                         {}
func (m *mockContext) IsAborted() bool                { return false }
func (m *mockContext) Set(key string, value interface{}) { m.data[key] = value }
func (m *mockContext) Get(key string) (interface{}, bool) {
	value, exists := m.data[key]
	return value, exists
}

func (m *mockContext) Bind(obj interface{}) error     { return nil }
func (m *mockContext) Body() ([]byte, error)          { return nil, nil }
func (m *mockContext) ResponseWriter() http.ResponseWriter { return m.responseWriter }
func (m *mockContext) Request() *http.Request         { return m.request }

func (m *mockContext) RemoteIP() string {
	return "192.168.1.1"
}

func (m *mockContext) UserAgent() string {
	return m.request.UserAgent()
}

func (m *mockContext) SetCSRFToken() error {
	return nil
}

func (m *mockContext) ValidateCSRF() bool {
	return true
}

func (m *mockContext) SetCookie(cookie *http.Cookie) {
	// Mock implementation
}

func (m *mockContext) Cookie(name string) (*http.Cookie, error) {
	return m.request.Cookie(name)
}

// Test functions

func TestTrimStringsMiddleware(t *testing.T) {
	config := &mockSecurityConfig{trimStrings: true}
	deps := &Dependencies{
		Config:    config,
		Logger:    &mockSecurityLogger{},
		Limiter:   &mockRateLimiter{},
		Sanitizer: &mockSanitizer{},
	}

	middleware := TrimStringsMiddleware(deps)

	// Create a form request with whitespace
	form := url.Values{}
	form.Add("name", "  John Doe  ")
	form.Add("email", "  john@example.com  ")

	req := httptest.NewRequest("POST", "/test", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	ctx := newMockContext(req, w)

	err := middleware(ctx)
	if err != nil {
		t.Errorf("Middleware returned error: %v", err)
	}

	if !ctx.nextCalled {
		t.Error("Next() was not called")
	}

	// Check that form values were trimmed
	req.ParseForm()
	if req.PostForm.Get("name") != "John Doe" {
		t.Errorf("Expected trimmed name 'John Doe', got '%s'", req.PostForm.Get("name"))
	}
}

func TestSQLInjectionProtectionMiddleware(t *testing.T) {
	config := &mockSecurityConfig{}
	logger := &mockSecurityLogger{}
	deps := &Dependencies{
		Config:    config,
		Logger:    logger,
		Limiter:   &mockRateLimiter{},
		Sanitizer: &mockSanitizer{},
	}

	middleware := SQLInjectionProtectionMiddleware(deps)

	// Test malicious SQL injection attempt
	req := httptest.NewRequest("GET", "/test?id=1%27%3B+DROP+TABLE+users%3B+--", nil)
	w := httptest.NewRecorder()
	ctx := newMockContext(req, w)

	err := middleware(ctx)
	if err != nil {
		t.Errorf("Middleware returned error: %v", err)
	}

	// Should have blocked the request
	if ctx.statusCode != 400 {
		t.Errorf("Expected status 400, got %d", ctx.statusCode)
	}

	// Should have logged the attempt
	if len(logger.warnings) == 0 {
		t.Error("Expected warning to be logged")
	}
}

func TestXSSProtectionMiddleware(t *testing.T) {
	config := &mockSecurityConfig{
		xssProtection:      true,
		contentTypeOptions: true,
		frameOptions:       "DENY",
	}
	deps := &Dependencies{
		Config:    config,
		Logger:    &mockSecurityLogger{},
		Limiter:   &mockRateLimiter{},
		Sanitizer: &mockSanitizer{},
	}

	middleware := XSSProtectionMiddleware(deps)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	ctx := newMockContext(req, w)

	err := middleware(ctx)
	if err != nil {
		t.Errorf("Middleware returned error: %v", err)
	}

	// Check security headers were set
	if ctx.headers["X-XSS-Protection"] != "1; mode=block" {
		t.Errorf("Expected X-XSS-Protection header, got '%s'", ctx.headers["X-XSS-Protection"])
	}

	if ctx.headers["X-Content-Type-Options"] != "nosniff" {
		t.Errorf("Expected X-Content-Type-Options header, got '%s'", ctx.headers["X-Content-Type-Options"])
	}

	if ctx.headers["X-Frame-Options"] != "DENY" {
		t.Errorf("Expected X-Frame-Options header, got '%s'", ctx.headers["X-Frame-Options"])
	}
}

func TestCORSMiddleware(t *testing.T) {
	config := &mockSecurityConfig{
		corsEnabled:     true,
		corsOrigins:     []string{"https://example.com"},
		corsMethods:     []string{"GET", "POST"},
		corsHeaders:     []string{"Content-Type", "Authorization"},
		corsCredentials: true,
		corsMaxAge:      3600,
	}
	deps := &Dependencies{
		Config:    config,
		Logger:    &mockSecurityLogger{},
		Limiter:   &mockRateLimiter{},
		Sanitizer: &mockSanitizer{},
	}

	middleware := EnhancedCORSMiddleware(deps)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	ctx := newMockContext(req, w)

	err := middleware(ctx)
	if err != nil {
		t.Errorf("Middleware returned error: %v", err)
	}

	// Check CORS headers were set
	if ctx.headers["Access-Control-Allow-Origin"] != "https://example.com" {
		t.Errorf("Expected Access-Control-Allow-Origin header, got '%s'", ctx.headers["Access-Control-Allow-Origin"])
	}

	if ctx.headers["Access-Control-Allow-Methods"] != "GET, POST" {
		t.Errorf("Expected Access-Control-Allow-Methods header, got '%s'", ctx.headers["Access-Control-Allow-Methods"])
	}
}

func TestSecurityMiddlewareGroup(t *testing.T) {
	config := &mockSecurityConfig{
		trimStrings:      true,
		xssProtection:    true,
		rateLimitEnabled: true,
		hstsConfig:       &mockHSTSConfig{enabled: false},
		cspConfig:        &mockCSPConfig{enabled: false},
		corsEnabled:      false,
	}
	deps := &Dependencies{
		Config:    config,
		Logger:    &mockSecurityLogger{},
		Limiter:   &mockRateLimiter{},
		Sanitizer: &mockSanitizer{},
	}

	middlewareGroup := SecurityMiddlewareGroup(deps)

	// Should have multiple middleware functions
	if len(middlewareGroup) == 0 {
		t.Error("Security middleware group is empty")
	}

	// Test that all middleware in the group can be called
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	ctx := newMockContext(req, w)

	for _, middleware := range middlewareGroup {
		// Reset next called flag
		ctx.nextCalled = false
		
		err := middleware(ctx)
		if err != nil {
			t.Errorf("Middleware in group returned error: %v", err)
		}
	}
}

func TestRateLimitingSecurityMiddleware(t *testing.T) {
	config := &mockSecurityConfig{rateLimitEnabled: true}
	logger := &mockSecurityLogger{}
	limiter := &mockRateLimiter{
		allowed: map[string]bool{
			"security_rate_limit:192.168.1.1": false, // Deny this IP
		},
	}
	deps := &Dependencies{
		Config:    config,
		Logger:    logger,
		Limiter:   limiter,
		Sanitizer: &mockSanitizer{},
	}

	middleware := RateLimitingSecurityMiddleware(deps)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	ctx := newMockContext(req, w)

	err := middleware(ctx)
	if err != nil {
		t.Errorf("Middleware returned error: %v", err)
	}

	// Should have blocked the request
	if ctx.statusCode != 429 {
		t.Errorf("Expected status 429, got %d", ctx.statusCode)
	}

	// Should have logged the rate limit exceeded
	if len(logger.warnings) == 0 {
		t.Error("Expected warning to be logged for rate limit exceeded")
	}
}

func TestSecurityLoggerMiddleware(t *testing.T) {
	config := &mockSecurityConfig{}
	logger := &mockSecurityLogger{}
	deps := &Dependencies{
		Config:    config,
		Logger:    logger,
		Limiter:   &mockRateLimiter{},
		Sanitizer: &mockSanitizer{},
	}

	middleware := SecurityLoggerMiddleware(deps)

	// Test with suspicious user agent
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "sqlmap/1.0")
	w := httptest.NewRecorder()
	ctx := newMockContext(req, w)

	err := middleware(ctx)
	if err != nil {
		t.Errorf("Middleware returned error: %v", err)
	}

	// Should have logged suspicious request
	if len(logger.warnings) == 0 {
		t.Error("Expected warning to be logged for suspicious request")
	}

	if !strings.Contains(logger.warnings[0].message, "Suspicious request") {
		t.Errorf("Expected suspicious request warning, got '%s'", logger.warnings[0].message)
	}
}