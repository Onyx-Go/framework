package onyx

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"  hello world  ", "hello world"},
		{"test\x00null", "testnull"},
		{"\t\n  spaced  \r\n", "spaced"},
		{"normal string", "normal string"},
	}

	for _, test := range tests {
		result := SanitizeString(test.input)
		if result != test.expected {
			t.Errorf("SanitizeString(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestStripTags(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"<script>alert('xss')</script>", "alert('xss')"},
		{"<p>Hello <b>world</b></p>", "Hello world"},
		{"<?php echo 'test'; ?>", ""},
		{"No tags here", "No tags here"},
		{"<div><span>nested</span></div>", "nested"},
	}

	for _, test := range tests {
		result := StripTags(test.input)
		if result != test.expected {
			t.Errorf("StripTags(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestEscapeHTML(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"<script>", "&lt;script&gt;"},
		{"Hello & World", "Hello &amp; World"},
		{"\"quotes\"", "&#34;quotes&#34;"},
		{"'single'", "&#39;single&#39;"},
		{"normal text", "normal text"},
	}

	for _, test := range tests {
		result := EscapeHTML(test.input)
		if result != test.expected {
			t.Errorf("EscapeHTML(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestEscapeJS(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"alert('test')", "alert(\\'test\\')"},
		{"line\nbreak", "line\\nbreak"},
		{"tab\there", "tab\\there"},
		{"<script>", "\\u003cscript\\u003e"},
		{"back\\slash", "back\\\\slash"},
	}

	for _, test := range tests {
		result := EscapeJS(test.input)
		if result != test.expected {
			t.Errorf("EscapeJS(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestSecurityValidationRules(t *testing.T) {
	// Test NoScriptRule
	noScriptRule := &NoScriptRule{}
	
	if err := noScriptRule.Validate("<script>alert('xss')</script>"); err == nil {
		t.Error("NoScriptRule should fail for script tags")
	}
	
	if err := noScriptRule.Validate("normal text"); err != nil {
		t.Error("NoScriptRule should pass for normal text")
	}

	// Test NoSQLInjectionRule
	noSQLRule := &NoSQLInjectionRule{}
	
	if err := noSQLRule.Validate("'; DROP TABLE users; --"); err == nil {
		t.Error("NoSQLInjectionRule should fail for SQL injection")
	}
	
	if err := noSQLRule.Validate("normal search term"); err != nil {
		t.Error("NoSQLInjectionRule should pass for normal text")
	}

	// Test ValidateNoScript helper
	if ValidateNoScript("<script>alert('test')</script>") {
		t.Error("ValidateNoScript should return false for script tags")
	}
	
	if !ValidateNoScript("safe text") {
		t.Error("ValidateNoScript should return true for safe text")
	}

	// Test ValidateNoSQLInjection helper
	if ValidateNoSQLInjection("'; DROP TABLE users; --") {
		t.Error("ValidateNoSQLInjection should return false for SQL injection")
	}
	
	if !ValidateNoSQLInjection("safe search") {
		t.Error("ValidateNoSQLInjection should return true for safe text")
	}
}

func TestExistingValidator(t *testing.T) {
	// Test with the existing validator system
	validData := map[string]interface{}{
		"name":  "John Doe",
		"email": "john@example.com",
		"age":   25,
	}

	validRules := map[string][]string{
		"name":  {"required", "min:3"},
		"email": {"required", "email"},
		"age":   {"numeric"},
	}

	validator := NewValidator(validData, validRules)
	
	if !validator.Validate() {
		t.Error("Validator should pass for valid data")
	}

	// Test invalid data
	invalidData := map[string]interface{}{
		"name":  "Jo", // Too short
		"email": "invalid-email",
		"age":   "not a number",
	}

	invalidRules := map[string][]string{
		"name":  {"required", "min:3"},
		"email": {"required", "email"},
		"age":   {"numeric"},
	}

	invalidValidator := NewValidator(invalidData, invalidRules)

	if invalidValidator.Validate() {
		t.Error("Validator should fail for invalid data")
	}

	if !invalidValidator.HasErrors() {
		t.Error("Validator should have errors for invalid data")
	}

	// Check specific errors
	firstNameError := invalidValidator.FirstError("name")
	if firstNameError == "" {
		t.Error("Validator should return first error for name field")
	}
}

func TestCSRFManager(t *testing.T) {
	config := DefaultSecurityConfig()
	manager := NewCSRFManager(config)

	// Test token generation
	token1, err := manager.GenerateToken()
	if err != nil {
		t.Fatalf("Failed to generate CSRF token: %v", err)
	}

	token2, err := manager.GenerateToken()
	if err != nil {
		t.Fatalf("Failed to generate second CSRF token: %v", err)
	}

	// Tokens should be different
	if token1 == token2 {
		t.Error("Generated tokens should be different")
	}

	// Test token validation
	if !manager.ValidateToken(token1, token1) {
		t.Error("Token should validate against itself")
	}

	if manager.ValidateToken(token1, token2) {
		t.Error("Different tokens should not validate")
	}

	if manager.ValidateToken("", token1) {
		t.Error("Empty token should not validate")
	}

	if manager.ValidateToken(token1, "") {
		t.Error("Token should not validate against empty session token")
	}
}

func TestTrimStringsMiddleware(t *testing.T) {
	app := New()
	app.UseMiddleware(TrimStringsMiddleware())

	app.PostHandler("/test", func(c Context) error {
		name := c.PostForm("name")
		return c.JSON(http.StatusOK, map[string]string{"name": name})
	})

	// Create form data with whitespace
	form := url.Values{}
	form.Add("name", "  John Doe  ")

	req := httptest.NewRequest("POST", "/test", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	recorder := httptest.NewRecorder()
	app.Router().ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", recorder.Code)
	}

	// The response should contain trimmed value
	body := recorder.Body.String()
	if !strings.Contains(body, "John Doe") {
		t.Error("Response should contain trimmed name")
	}
}

func TestXSSProtectionMiddleware(t *testing.T) {
	app := New()
	app.UseMiddleware(XSSProtectionMiddleware())

	app.GetHandler("/test", func(c Context) error {
		return c.JSON(http.StatusOK, map[string]string{"message": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	recorder := httptest.NewRecorder()

	app.Router().ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", recorder.Code)
	}

	// Check security headers
	headers := recorder.Header()
	
	if headers.Get("X-XSS-Protection") != "1; mode=block" {
		t.Error("X-XSS-Protection header not set correctly")
	}

	if headers.Get("X-Content-Type-Options") != "nosniff" {
		t.Error("X-Content-Type-Options header not set correctly")
	}

	if headers.Get("X-Frame-Options") != "DENY" {
		t.Error("X-Frame-Options header not set correctly")
	}
}

func TestSecurityHeadersMiddleware(t *testing.T) {
	app := New()
	app.UseMiddleware(SecurityHeadersMiddleware())

	app.GetHandler("/test", func(c Context) error {
		return c.JSON(http.StatusOK, map[string]string{"message": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	recorder := httptest.NewRecorder()

	app.Router().ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", recorder.Code)
	}

	headers := recorder.Header()

	// Check HSTS header
	hsts := headers.Get("Strict-Transport-Security")
	if !strings.Contains(hsts, "max-age=31536000") {
		t.Error("HSTS header not set correctly")
	}

	// Check CSP header
	csp := headers.Get("Content-Security-Policy")
	if !strings.Contains(csp, "default-src 'self'") {
		t.Error("CSP header not set correctly")
	}

	// Check Referrer Policy
	if headers.Get("Referrer-Policy") != "strict-origin-when-cross-origin" {
		t.Error("Referrer-Policy header not set correctly")
	}
}

func TestCORSMiddleware(t *testing.T) {
	// Enable CORS in config
	config := DefaultSecurityConfig()
	config.CORSEnabled = true
	config.CORSAllowedOrigins = []string{"https://example.com", "https://app.example.com"}
	SetSecurityConfig(config)

	app := New()
	app.Use(CORSMiddleware())

	app.GetHandler("/test", func(c Context) error {
		return c.JSON(http.StatusOK, map[string]string{"message": "ok"})
	})

	// Test preflight request
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	recorder := httptest.NewRecorder()

	app.Router().ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Errorf("Expected status 204 for preflight, got %d", recorder.Code)
	}

	headers := recorder.Header()
	if headers.Get("Access-Control-Allow-Origin") != "https://example.com" {
		t.Error("CORS origin header not set correctly")
	}

	// Test actual request
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	recorder = httptest.NewRecorder()

	app.Router().ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", recorder.Code)
	}

	if recorder.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
		t.Error("CORS origin header not set correctly for actual request")
	}
}

func TestSQLInjectionProtectionMiddleware(t *testing.T) {
	app := New()
	app.UseMiddleware(SQLInjectionProtectionMiddleware())

	app.PostHandler("/test", func(c Context) error {
		return c.JSON(http.StatusOK, map[string]string{"message": "ok"})
	})

	// Test malicious SQL injection attempt
	form := url.Values{}
	form.Add("query", "'; DROP TABLE users; --")

	req := httptest.NewRequest("POST", "/test", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	recorder := httptest.NewRecorder()
	app.Router().ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for SQL injection attempt, got %d", recorder.Code)
	}

	// Test legitimate request
	form = url.Values{}
	form.Add("query", "normal search term")

	req = httptest.NewRequest("POST", "/test", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	recorder = httptest.NewRecorder()
	app.Router().ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status 200 for legitimate request, got %d", recorder.Code)
	}
}

func TestInputSanitizationMiddleware(t *testing.T) {
	config := DefaultSecurityConfig()
	config.StripTags = true
	config.MaxInputLength = 10
	SetSecurityConfig(config)

	app := New()
	app.UseMiddleware(InputSanitizationMiddleware())

	app.PostHandler("/test", func(c Context) error {
		content := c.PostForm("content")
		return c.JSON(http.StatusOK, map[string]string{"content": content})
	})

	// Test with HTML tags and long input
	form := url.Values{}
	form.Add("content", "<script>alert('xss')</script>this is a very long input that exceeds the limit")

	req := httptest.NewRequest("POST", "/test", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	recorder := httptest.NewRecorder()
	app.Router().ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", recorder.Code)
	}

	body := recorder.Body.String()
	
	// Should not contain script tags
	if strings.Contains(body, "<script>") {
		t.Error("HTML tags should be stripped")
	}
	
	// Should be truncated to max length
	if strings.Contains(body, "very long input") {
		t.Error("Input should be truncated to max length")
	}
}

func TestSecurityMiddlewareGroup(t *testing.T) {
	app := New()
	
	// Apply security middleware group
	for _, middleware := range SecurityMiddlewareGroup() {
		app.UseMiddleware(middleware)
	}

	app.GetHandler("/test", func(c Context) error {
		return c.JSON(http.StatusOK, map[string]string{"message": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	recorder := httptest.NewRecorder()

	app.Router().ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", recorder.Code)
	}

	headers := recorder.Header()

	// Check that multiple security headers are present
	expectedHeaders := []string{
		"X-XSS-Protection",
		"X-Content-Type-Options",
		"X-Frame-Options",
		"Strict-Transport-Security",
		"Content-Security-Policy",
		"Referrer-Policy",
	}

	for _, header := range expectedHeaders {
		if headers.Get(header) == "" {
			t.Errorf("Security header %s not set", header)
		}
	}
}

func TestContextFormHandling(t *testing.T) {
	app := New()

	app.PostHandler("/test", func(c Context) error {
		name := c.PostForm("name")
		email := c.PostForm("email")
		
		if name == "" || email == "" {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "name and email are required",
			})
		}

		return c.JSON(http.StatusOK, map[string]string{
			"name":  name,
			"email": email,
		})
	})

	// Test with form data
	form := url.Values{}
	form.Add("name", "John Doe")
	form.Add("email", "john@example.com")

	req := httptest.NewRequest("POST", "/test", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	recorder := httptest.NewRecorder()
	app.Router().ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", recorder.Code)
	}

	body := recorder.Body.String()
	if !strings.Contains(body, "John Doe") {
		t.Error("Response should contain form data")
	}
}

func TestSecurityConfiguration(t *testing.T) {
	// Test default configuration
	config := DefaultSecurityConfig()

	if !config.TrimStrings {
		t.Error("TrimStrings should be enabled by default")
	}

	if !config.XSSProtection {
		t.Error("XSSProtection should be enabled by default")
	}

	if !config.CSRFProtection {
		t.Error("CSRFProtection should be enabled by default")
	}

	if !config.HSTS.Enabled {
		t.Error("HSTS should be enabled by default")
	}

	if !config.CSP.Enabled {
		t.Error("CSP should be enabled by default")
	}

	// Test setting custom configuration
	customConfig := &SecurityConfig{
		TrimStrings:   false,
		XSSProtection: false,
	}

	SetSecurityConfig(customConfig)
	retrievedConfig := GetSecurityConfig()

	if retrievedConfig.TrimStrings {
		t.Error("Custom configuration should override defaults")
	}

	if retrievedConfig.XSSProtection {
		t.Error("Custom configuration should override defaults")
	}

	// Reset to default for other tests
	SetSecurityConfig(DefaultSecurityConfig())
}

// Benchmark tests for performance

func BenchmarkSanitizeString(b *testing.B) {
	input := "  test string with \x00 null bytes  "
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		SanitizeString(input)
	}
}

func BenchmarkStripTags(b *testing.B) {
	input := "<div><p>Hello <b>world</b> with <script>alert('xss')</script></p></div>"
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		StripTags(input)
	}
}

func BenchmarkEscapeHTML(b *testing.B) {
	input := "<script>alert('test')</script> & other <stuff>"
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		EscapeHTML(input)
	}
}

func BenchmarkValidator(b *testing.B) {
	data := map[string]interface{}{
		"name":  "John Doe",
		"email": "john@example.com",
		"age":   25,
	}

	rules := map[string][]string{
		"name":  {"required", "min:3"},
		"email": {"required", "email"},
		"age":   {"numeric"},
	}

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		validator := NewValidator(data, rules)
		validator.Validate()
	}
}