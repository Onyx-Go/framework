package onyx

import (
	"compress/gzip"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDefaultCompressionConfig(t *testing.T) {
	config := DefaultCompressionConfig()
	
	if config.Level != gzip.DefaultCompression {
		t.Errorf("Expected default compression level %d, got %d", gzip.DefaultCompression, config.Level)
	}
	
	if config.MinLength != 1024 {
		t.Errorf("Expected min length 1024, got %d", config.MinLength)
	}
	
	expectedTypes := []string{
		"text/plain",
		"text/html", 
		"application/json",
	}
	
	for _, expectedType := range expectedTypes {
		found := false
		for _, configType := range config.Types {
			if configType == expectedType {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected type %s not found in default config", expectedType)
		}
	}
}

func TestCompressionMiddleware_NoAcceptEncoding(t *testing.T) {
	app := New()
	
	// Add compression middleware
	app.UseMiddleware(CompressionMiddleware())
	
	// Add test route
	app.GetHandler("/test", func(c Context) error {
		return c.String(200, strings.Repeat("Hello World! ", 100)) // > 1KB
	})
	
	// Create request without Accept-Encoding
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	
	app.Router().ServeHTTP(w, req)
	
	// Should not be compressed
	if w.Header().Get("Content-Encoding") == "gzip" {
		t.Error("Response should not be compressed without Accept-Encoding")
	}
	
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestCompressionMiddleware_WithGzipAcceptEncoding(t *testing.T) {
	app := New()
	
	// Add compression middleware using new-style API
	app.Use(NewStyleCompressionMiddleware())
	
	// Large response that should be compressed
	largeContent := strings.Repeat("Hello World! This is a test of compression. ", 50) // > 1KB
	
	app.GetHandler("/test", func(c Context) error {
		return c.String(200, largeContent)
	})
	
	// Create request with gzip Accept-Encoding
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	w := httptest.NewRecorder()
	
	app.Router().ServeHTTP(w, req)
	
	// Debug: Print all headers
	t.Logf("All response headers: %v", w.Header())
	t.Logf("Content-Encoding: '%s'", w.Header().Get("Content-Encoding"))
	t.Logf("Vary: '%s'", w.Header().Get("Vary"))
	
	// Should be compressed
	if w.Header().Get("Content-Encoding") != "gzip" {
		t.Error("Response should be compressed with gzip")
	}
	
	if w.Header().Get("Vary") != "Accept-Encoding" {
		t.Error("Vary header should be set to Accept-Encoding")
	}
	
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	// Decompress and verify content
	reader, err := gzip.NewReader(w.Body)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer reader.Close()
	
	decompressed, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to decompress response: %v", err)
	}
	
	if string(decompressed) != largeContent {
		t.Error("Decompressed content doesn't match original")
	}
}

func TestCompressionMiddleware_SmallResponse(t *testing.T) {
	app := New()
	
	// Add compression middleware with very high minimum length to test size threshold
	config := CompressionConfig{
		Level:     gzip.DefaultCompression,
		MinLength: 10000, // Very high threshold
		Types:     []string{"text/plain"},
		Exclude:   []string{},
	}
	app.UseMiddleware(CompressionMiddleware(config))
	
	// Small response that should not be compressed due to high threshold
	smallContent := "Hello World!"
	
	app.GetHandler("/test", func(c Context) error {
		return c.String(200, smallContent)
	})
	
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	
	app.Router().ServeHTTP(w, req)
	
	// Small response should NOT be compressed due to high threshold
	if w.Header().Get("Content-Encoding") == "gzip" {
		t.Error("Small response should not be compressed with high threshold")
	}
}

func TestCompressionMiddleware_CustomConfig(t *testing.T) {
	app := New()
	
	// Custom config with very small minimum length
	config := CompressionConfig{
		Level:     gzip.BestSpeed,
		MinLength: 10, // Very small threshold
		Types:     []string{"text/plain"},
		Exclude:   []string{"/health"},
	}
	
	app.UseMiddleware(CompressionMiddleware(config))
	
	// Small response that should now be compressed due to low threshold
	content := "Hello World! This is a test."
	
	app.GetHandler("/test", func(c Context) error {
		return c.String(200, content)
	})
	
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	
	app.Router().ServeHTTP(w, req)
	
	// Should be compressed with custom config
	if w.Header().Get("Content-Encoding") != "gzip" {
		t.Error("Response should be compressed with custom config")
	}
}

func TestCompressionMiddleware_ExcludedPath(t *testing.T) {
	app := New()
	
	config := CompressionConfig{
		Level:     gzip.DefaultCompression,
		MinLength: 10,
		Types:     []string{"text/plain"},
		Exclude:   []string{"/health", "/metrics"},
	}
	
	app.UseMiddleware(CompressionMiddleware(config))
	
	// Large content on excluded path
	content := strings.Repeat("Health check data ", 100)
	
	app.GetHandler("/health", func(c Context) error {
		return c.String(200, content)
	})
	
	req := httptest.NewRequest("GET", "/health", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	
	app.Router().ServeHTTP(w, req)
	
	// Should not be compressed due to exclusion
	if w.Header().Get("Content-Encoding") == "gzip" {
		t.Error("Excluded path should not be compressed")
	}
}

func TestCompressionMiddleware_UnsupportedContentType(t *testing.T) {
	app := New()
	
	config := CompressionConfig{
		Level:     gzip.DefaultCompression,
		MinLength: 10,
		Types:     []string{"text/html"}, // Only HTML
		Exclude:   []string{},
	}
	
	app.UseMiddleware(CompressionMiddleware(config))
	
	// Large plain text content (not in supported types)
	content := strings.Repeat("Plain text data ", 100)
	
	app.GetHandler("/test", func(c Context) error {
		return c.String(200, content) // Content-Type: text/plain
	})
	
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	
	app.Router().ServeHTTP(w, req)
	
	// Should not be compressed due to unsupported content type
	if w.Header().Get("Content-Encoding") == "gzip" {
		t.Error("Unsupported content type should not be compressed")
	}
}

func TestCompressionMiddleware_JSONResponse(t *testing.T) {
	app := New()
	
	app.UseMiddleware(CompressionMiddleware())
	
	// Large JSON response
	data := map[string]interface{}{
		"users": make([]map[string]string, 100),
	}
	
	// Fill with test data to make it large
	for i := 0; i < 100; i++ {
		data["users"].([]map[string]string)[i] = map[string]string{
			"name":  "User " + strings.Repeat("Test", 10),
			"email": "user" + strings.Repeat("test", 10) + "@example.com",
		}
	}
	
	app.GetHandler("/api/users", func(c Context) error {
		return c.JSON(200, data)
	})
	
	req := httptest.NewRequest("GET", "/api/users", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	
	app.Router().ServeHTTP(w, req)
	
	// Should be compressed
	if w.Header().Get("Content-Encoding") != "gzip" {
		t.Error("JSON response should be compressed")
	}
	
	if w.Header().Get("Content-Type") != "application/json" {
		t.Error("Content-Type should remain application/json")
	}
}

func TestCompressionMiddleware_ErrorResponse(t *testing.T) {
	app := New()
	
	app.UseMiddleware(CompressionMiddleware())
	
	app.GetHandler("/error", func(c Context) error {
		return c.String(500, strings.Repeat("Error message ", 100))
	})
	
	req := httptest.NewRequest("GET", "/error", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	
	app.Router().ServeHTTP(w, req)
	
	// Error responses should not be compressed
	if w.Header().Get("Content-Encoding") == "gzip" {
		t.Error("Error responses should not be compressed")
	}
	
	if w.Code != 500 {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestCompressionMiddleware_RedirectResponse(t *testing.T) {
	app := New()
	
	app.UseMiddleware(CompressionMiddleware())
	
	app.GetHandler("/redirect", func(c Context) error {
		return c.Redirect(302, "/somewhere")
	})
	
	req := httptest.NewRequest("GET", "/redirect", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	
	app.Router().ServeHTTP(w, req)
	
	// Redirect responses should not be compressed
	if w.Header().Get("Content-Encoding") == "gzip" {
		t.Error("Redirect responses should not be compressed")
	}
	
	if w.Code != 302 {
		t.Errorf("Expected status 302, got %d", w.Code)
	}
}

func TestGzipMiddleware_ConvenienceFunction(t *testing.T) {
	app := New()
	
	// Test convenience function
	app.UseMiddleware(GzipMiddleware())
	
	content := strings.Repeat("Test content ", 100)
	
	app.GetHandler("/test", func(c Context) error {
		return c.String(200, content)
	})
	
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	
	app.Router().ServeHTTP(w, req)
	
	// Should be compressed
	if w.Header().Get("Content-Encoding") != "gzip" {
		t.Error("Response should be compressed with GzipMiddleware")
	}
}

func TestCustomCompressionMiddleware(t *testing.T) {
	app := New()
	
	// Test custom compression function
	app.UseMiddleware(CustomCompressionMiddleware(
		gzip.BestSpeed,
		100, // Very low threshold
		[]string{"text/plain", "application/json"},
	))
	
	content := strings.Repeat("Custom test ", 20) // Should exceed 100 bytes
	
	app.GetHandler("/test", func(c Context) error {
		return c.String(200, content)
	})
	
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	
	app.Router().ServeHTTP(w, req)
	
	// Should be compressed
	if w.Header().Get("Content-Encoding") != "gzip" {
		t.Error("Response should be compressed with custom middleware")
	}
}

func TestCompressionRatio(t *testing.T) {
	app := New()
	
	app.UseMiddleware(CompressionMiddleware())
	
	// Highly compressible content
	content := strings.Repeat("This is highly repetitive content that should compress very well. ", 50)
	
	app.GetHandler("/test", func(c Context) error {
		return c.String(200, content)
	})
	
	// Test uncompressed
	req1 := httptest.NewRequest("GET", "/test", nil)
	w1 := httptest.NewRecorder()
	app.Router().ServeHTTP(w1, req1)
	uncompressedSize := len(w1.Body.Bytes())
	
	// Test compressed
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.Header.Set("Accept-Encoding", "gzip")
	w2 := httptest.NewRecorder()
	app.Router().ServeHTTP(w2, req2)
	compressedSize := len(w2.Body.Bytes())
	
	// Compressed should be significantly smaller
	compressionRatio := float64(compressedSize) / float64(uncompressedSize)
	if compressionRatio > 0.5 { // Should compress to less than 50%
		t.Errorf("Compression ratio too low: %.2f (compressed: %d, uncompressed: %d)", 
			compressionRatio, compressedSize, uncompressedSize)
	}
	
	t.Logf("Compression ratio: %.2f%% (compressed: %d bytes, uncompressed: %d bytes)", 
		compressionRatio*100, compressedSize, uncompressedSize)
}