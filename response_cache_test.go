package onyx
import "fmt"

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestDefaultResponseCacheConfig(t *testing.T) {
	config := DefaultResponseCacheConfig()
	
	if !config.Enabled {
		t.Error("Expected response cache to be enabled by default")
	}
	
	if config.DefaultTTL != 5*time.Minute {
		t.Errorf("Expected default TTL of 5 minutes, got %v", config.DefaultTTL)
	}
	
	if config.MaxCacheSize != 1000 {
		t.Errorf("Expected max cache size of 1000, got %d", config.MaxCacheSize)
	}
}

func TestNewResponseCache(t *testing.T) {
	config := DefaultResponseCacheConfig()
	cache := NewResponseCache(config)
	
	if cache == nil {
		t.Fatal("Expected response cache to be created")
	}
	
	if cache.config.Enabled != config.Enabled {
		t.Error("Expected cache config to match input config")
	}
}

func TestResponseCacheMiddleware_CacheHit(t *testing.T) {
	app := New()
	
	app.UseMiddleware(ResponseCacheMiddleware())
	
	callCount := 0
	app.GetHandler("/test", func(c Context) error {
		callCount++
		return c.String(200, fmt.Sprintf("Test %d", callCount))
	})
	
	// First request
	req1 := httptest.NewRequest("GET", "/test", nil)
	w1 := httptest.NewRecorder()
	app.Router().ServeHTTP(w1, req1)
	
	// Second request (should be cached)
	req2 := httptest.NewRequest("GET", "/test", nil)
	w2 := httptest.NewRecorder()
	app.Router().ServeHTTP(w2, req2)
	
	// Handler should be called only once (second request cached)
	if callCount != 1 {
		t.Errorf("Expected handler to be called once, was called %d times", callCount)
	}
	
	// Both responses should be the same
	if w1.Body.String() != w2.Body.String() {
		t.Error("Cached response body should match original")
	}
	
	// Second response should have cache header
	if w2.Header().Get("X-Cache") != "HIT" {
		t.Error("Second response should have X-Cache: HIT header")
	}
}

func TestResponseCacheMiddleware_ExcludedPath(t *testing.T) {
	app := New()
	
	config := DefaultResponseCacheConfig()
	config.ExcludedPaths = []string{"/admin/"}
	app.UseMiddleware(ResponseCacheMiddleware(config))
	
	callCount := 0
	app.GetHandler("/admin/dashboard", func(c Context) error {
		callCount++
		return c.String(200, fmt.Sprintf("Admin %d", callCount))
	})
	
	// First request
	req1 := httptest.NewRequest("GET", "/admin/dashboard", nil)
	w1 := httptest.NewRecorder()
	app.Router().ServeHTTP(w1, req1)
	
	// Second request
	req2 := httptest.NewRequest("GET", "/admin/dashboard", nil)
	w2 := httptest.NewRecorder()
	app.Router().ServeHTTP(w2, req2)
	
	// Handler should be called twice (no caching for excluded paths)
	if callCount != 2 {
		t.Errorf("Expected handler to be called twice for excluded path, was called %d times", callCount)
	}
}

func TestResponseCacheMiddleware_MethodFiltering(t *testing.T) {
	app := New()
	
	config := DefaultResponseCacheConfig()
	config.IncludedMethods = []string{"GET"} // Only cache GET requests
	app.UseMiddleware(ResponseCacheMiddleware(config))
	
	callCount := 0
	app.PostHandler("/data", func(c Context) error {
		callCount++
		return c.String(200, fmt.Sprintf("POST %d", callCount))
	})
	
	// First POST request
	req1 := httptest.NewRequest("POST", "/data", nil)
	w1 := httptest.NewRecorder()
	app.Router().ServeHTTP(w1, req1)
	
	// Second POST request
	req2 := httptest.NewRequest("POST", "/data", nil)
	w2 := httptest.NewRecorder()
	app.Router().ServeHTTP(w2, req2)
	
	// Handler should be called twice (POST not cached)
	if callCount != 2 {
		t.Errorf("Expected handler to be called twice for POST requests, was called %d times", callCount)
	}
}

func TestResponseCacheMiddleware_ErrorResponse(t *testing.T) {
	app := New()
	
	app.UseMiddleware(ResponseCacheMiddleware())
	
	callCount := 0
	app.GetHandler("/error", func(c Context) error {
		callCount++
		return c.String(500, fmt.Sprintf("Error %d", callCount))
	})
	
	// First request (error response)
	req1 := httptest.NewRequest("GET", "/error", nil)
	w1 := httptest.NewRecorder()
	app.Router().ServeHTTP(w1, req1)
	
	// Second request
	req2 := httptest.NewRequest("GET", "/error", nil)
	w2 := httptest.NewRecorder()
	app.Router().ServeHTTP(w2, req2)
	
	// Handler should be called twice (error responses not cached)
	if callCount != 2 {
		t.Errorf("Expected handler to be called twice for error responses, was called %d times", callCount)
	}
}

func TestResponseCacheMiddleware_DisabledCache(t *testing.T) {
	app := New()
	
	config := DefaultResponseCacheConfig()
	config.Enabled = false
	app.UseMiddleware(ResponseCacheMiddleware(config))
	
	callCount := 0
	app.GetHandler("/test", func(c Context) error {
		callCount++
		return c.String(200, fmt.Sprintf("Test %d", callCount))
	})
	
	// First request
	req1 := httptest.NewRequest("GET", "/test", nil)
	w1 := httptest.NewRecorder()
	app.Router().ServeHTTP(w1, req1)
	
	// Second request
	req2 := httptest.NewRequest("GET", "/test", nil)
	w2 := httptest.NewRecorder()
	app.Router().ServeHTTP(w2, req2)
	
	// Handler should be called twice (caching disabled)
	if callCount != 2 {
		t.Errorf("Expected handler to be called twice when caching disabled, was called %d times", callCount)
	}
}

func TestResponseCache_KeyGeneration(t *testing.T) {
	config := DefaultResponseCacheConfig()
	cache := NewResponseCache(config)
	
	// Create test requests
	req1 := httptest.NewRequest("GET", "/test", nil)
	req2 := httptest.NewRequest("GET", "/test", nil)
	req3 := httptest.NewRequest("GET", "/test?param=1", nil)
	
	key1 := cache.generateKey(req1)
	key2 := cache.generateKey(req2)
	key3 := cache.generateKey(req3)
	
	// Same requests should generate same key
	if key1 != key2 {
		t.Error("Same requests should generate same cache key")
	}
	
	// Different requests should generate different keys
	if key1 == key3 {
		t.Error("Different requests should generate different cache keys")
	}
}

func TestResponseCache_Metrics(t *testing.T) {
	config := DefaultResponseCacheConfig()
	cache := NewResponseCache(config)
	
	// Initial metrics
	metrics := cache.GetMetrics()
	if metrics.Hits != 0 {
		t.Error("Initial hits should be 0")
	}
	if metrics.Misses != 0 {
		t.Error("Initial misses should be 0")
	}
	
	// Test cache miss
	req := httptest.NewRequest("GET", "/test", nil)
	cached := cache.get(cache.generateKey(req))
	if cached != nil {
		t.Error("Should be cache miss for new request")
	}
	
	// Test cache store
	recorder := &ResponseRecorder{
		ResponseWriter: httptest.NewRecorder(),
		statusCode:     200,
		body:          []byte("test response"),
	}
	cache.store(cache.generateKey(req), recorder, config.DefaultTTL)
	
	// Test cache hit
	cached = cache.get(cache.generateKey(req))
	if cached == nil {
		t.Error("Should be cache hit after storing")
	}
	
	// Check metrics
	metrics = cache.GetMetrics()
	if metrics.Hits != 1 {
		t.Errorf("Expected 1 hit, got %d", metrics.Hits)
	}
	if metrics.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", metrics.Misses)
	}
	if metrics.Stores != 1 {
		t.Errorf("Expected 1 store, got %d", metrics.Stores)
	}
}

func TestResponseCache_CacheInfo(t *testing.T) {
	config := DefaultResponseCacheConfig()
	cache := NewResponseCache(config)
	
	info := cache.GetCacheInfo()
	
	if !info["enabled"].(bool) {
		t.Error("Cache should be enabled")
	}
	
	if info["total_entries"].(int) != 0 {
		t.Error("Initial cache should be empty")
	}
	
	if info["max_size"].(int) != config.MaxCacheSize {
		t.Error("Max size should match config")
	}
	
	if info["hit_rate"].(float64) != 0 {
		t.Error("Initial hit rate should be 0")
	}
}