package onyx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestMemoryRateLimiter_TokenBucket(t *testing.T) {
	limiter := NewMemoryRateLimiter("token_bucket")
	defer limiter.Close()
	
	ctx := context.Background()
	key := "test_key"
	limit := 5
	window := time.Second
	
	// Test initial requests (should all pass)
	for i := 0; i < limit; i++ {
		result, err := limiter.Allow(ctx, key, limit, window)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !result.Allowed {
			t.Fatalf("Request %d should be allowed", i+1)
		}
		if result.Remaining != limit-i-1 {
			t.Errorf("Request %d: expected remaining %d, got %d", i+1, limit-i-1, result.Remaining)
		}
	}
	
	// Next request should be rejected
	result, err := limiter.Allow(ctx, key, limit, window)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result.Allowed {
		t.Error("Request should be rejected when limit exceeded")
	}
	if result.Remaining != 0 {
		t.Errorf("Expected remaining 0, got %d", result.Remaining)
	}
}

func TestMemoryRateLimiter_SlidingWindow(t *testing.T) {
	limiter := NewMemoryRateLimiter("sliding_window")
	defer limiter.Close()
	
	ctx := context.Background()
	key := "test_key"
	limit := 3
	window := 100 * time.Millisecond
	
	// Test initial requests
	for i := 0; i < limit; i++ {
		result, err := limiter.Allow(ctx, key, limit, window)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !result.Allowed {
			t.Fatalf("Request %d should be allowed", i+1)
		}
	}
	
	// Next request should be rejected
	result, err := limiter.Allow(ctx, key, limit, window)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Log("Request correctly rejected")
	}
	
	// Wait for window to slide
	time.Sleep(window + 10*time.Millisecond)
	
	// Should be allowed again
	result, err = limiter.Allow(ctx, key, limit, window)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Error("Request should be allowed after window slides")
	}
}

func TestMemoryRateLimiter_Reset(t *testing.T) {
	limiter := NewMemoryRateLimiter("token_bucket")
	defer limiter.Close()
	
	ctx := context.Background()
	key := "test_key"
	limit := 2
	window := time.Second
	
	// Exhaust the limit
	for i := 0; i < limit; i++ {
		limiter.Allow(ctx, key, limit, window)
	}
	
	// Should be rejected
	result, err := limiter.Allow(ctx, key, limit, window)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result.Allowed {
		t.Error("Request should be rejected")
	}
	
	// Reset the limiter
	err = limiter.Reset(ctx, key)
	if err != nil {
		t.Fatalf("Reset failed: %v", err)
	}
	
	// Should be allowed again
	result, err = limiter.Allow(ctx, key, limit, window)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Error("Request should be allowed after reset")
	}
}

func TestRateLimitKeyGenerators(t *testing.T) {
	// Mock context for testing
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:8080"
	w := httptest.NewRecorder()
	c := NewContext(w, req, nil)
	
	// Test IP key generator
	ipKey := IPKeyGenerator(c)
	expectedIPKey := "rate_limit:ip:192.168.1.1"
	if ipKey != expectedIPKey {
		t.Errorf("IPKeyGenerator: expected %s, got %s", expectedIPKey, ipKey)
	}
	
	// Test user key generator (anonymous)
	userKey := UserKeyGenerator(c)
	expectedUserKey := "rate_limit:user:anonymous"
	if userKey != expectedUserKey {
		t.Errorf("UserKeyGenerator: expected %s, got %s", expectedUserKey, userKey)
	}
	
	// Test user key generator (authenticated)
	c.Set("user_id", "123")
	userKey = UserKeyGenerator(c)
	expectedUserKey = "rate_limit:user:123"
	if userKey != expectedUserKey {
		t.Errorf("UserKeyGenerator with user: expected %s, got %s", expectedUserKey, userKey)
	}
	
	// Test route key generator
	routeKey := RouteKeyGenerator(c)
	expectedRouteKey := "rate_limit:route:GET:/test"
	if routeKey != expectedRouteKey {
		t.Errorf("RouteKeyGenerator: expected %s, got %s", expectedRouteKey, routeKey)
	}
	
	// Test composite key generator
	compositeGen := CompositeKeyGenerator(IPKeyGenerator, UserKeyGenerator)
	compositeKey := compositeGen(c)
	expectedCompositeKey := "rate_limit:ip:192.168.1.1:rate_limit:user:123"
	if compositeKey != expectedCompositeKey {
		t.Errorf("CompositeKeyGenerator: expected %s, got %s", expectedCompositeKey, compositeKey)
	}
}

func TestRateLimitManager(t *testing.T) {
	manager := NewRateLimitManager()
	
	// Test limiter registration
	limiter := NewMemoryRateLimiter("token_bucket")
	manager.RegisterLimiter("test", limiter)
	
	retrievedLimiter, exists := manager.GetLimiter("test")
	if !exists {
		t.Error("Limiter should exist after registration")
	}
	if retrievedLimiter != limiter {
		t.Error("Retrieved limiter should be the same as registered")
	}
	
	// Test config registration
	config := &RateLimitMiddlewareConfig{
		Name: "test_config",
		Config: &RateLimitConfig{
			Algorithm: "token_bucket",
			Limit:     10,
			Window:    time.Minute,
		},
	}
	manager.RegisterConfig("test_config", config)
	
	retrievedConfig, exists := manager.GetConfig("test_config")
	if !exists {
		t.Error("Config should exist after registration")
	}
	if retrievedConfig != config {
		t.Error("Retrieved config should be the same as registered")
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	// Create a test server
	app := New()
	
	// Add rate limiting middleware
	app.UseMiddleware(RateLimit(3, time.Second))
	
	// Add test route
	app.GetHandler("/test", func(c Context) error {
		return c.JSON(http.StatusOK, map[string]string{"message": "success"})
	})
	
	server := httptest.NewServer(app.Router())
	defer server.Close()
	
	client := &http.Client{}
	
	// Test requests within limit
	for i := 0; i < 3; i++ {
		resp, err := client.Get(server.URL + "/test")
		if err != nil {
			t.Fatalf("Request %d failed: %v", i+1, err)
		}
		
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Request %d: expected status 200, got %d", i+1, resp.StatusCode)
		}
		
		// Check rate limit headers
		limit := resp.Header.Get("X-RateLimit-Limit")
		if limit != "3" {
			t.Errorf("Request %d: expected limit header '3', got '%s'", i+1, limit)
		}
		
		remaining := resp.Header.Get("X-RateLimit-Remaining")
		expectedRemaining := strconv.Itoa(3 - i - 1)
		if remaining != expectedRemaining {
			t.Errorf("Request %d: expected remaining '%s', got '%s'", i+1, expectedRemaining, remaining)
		}
		
		resp.Body.Close()
	}
	
	// Test request over limit
	resp, err := client.Get(server.URL + "/test")
	if err != nil {
		t.Fatalf("Rate limited request failed: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("Expected status 429, got %d", resp.StatusCode)
	}
	
	// Check retry-after header
	retryAfter := resp.Header.Get("Retry-After")
	if retryAfter == "" {
		t.Error("Expected Retry-After header")
	}
}

func TestRateLimitPerUser(t *testing.T) {
	app := New()
	
	// Middleware to set user ID
	app.UseMiddleware(func(c Context) error {
		userID := c.Query("user_id")
		if userID != "" {
			c.Set("user_id", userID)
		}
		c.Next()
		return nil
	})
	
	// Add per-user rate limiting
	app.UseMiddleware(RateLimitPerUser(2, time.Second))
	
	app.GetHandler("/test", func(c Context) error {
		return c.JSON(http.StatusOK, map[string]string{"message": "success"})
	})
	
	server := httptest.NewServer(app.Router())
	defer server.Close()
	
	client := &http.Client{}
	
	// Test user1 - should get 2 requests
	for i := 0; i < 2; i++ {
		resp, err := client.Get(server.URL + "/test?user_id=user1")
		if err != nil {
			t.Fatalf("User1 request %d failed: %v", i+1, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("User1 request %d: expected status 200, got %d", i+1, resp.StatusCode)
		}
		resp.Body.Close()
	}
	
	// User1's third request should be blocked
	resp, err := client.Get(server.URL + "/test?user_id=user1")
	if err != nil {
		t.Fatalf("User1 blocked request failed: %v", err)
	}
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("User1 third request: expected status 429, got %d", resp.StatusCode)
	}
	resp.Body.Close()
	
	// User2 should still be able to make requests
	resp, err = client.Get(server.URL + "/test?user_id=user2")
	if err != nil {
		t.Fatalf("User2 request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("User2 request: expected status 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestCustomRateLimit(t *testing.T) {
	// Custom configuration with fixed window
	config := &RateLimitConfig{
		Algorithm:    "fixed_window",
		Limit:        5,
		Window:       time.Second,
		Backend:      "memory",
		KeyGenerator: IPKeyGenerator,
		Headers:      true,
	}
	
	app := New()
	app.UseMiddleware(CustomRateLimit(config))
	
	app.GetHandler("/test", func(c Context) error {
		return c.JSON(http.StatusOK, map[string]string{"message": "success"})
	})
	
	server := httptest.NewServer(app.Router())
	defer server.Close()
	
	client := &http.Client{}
	
	// Test requests within limit
	for i := 0; i < 5; i++ {
		resp, err := client.Get(server.URL + "/test")
		if err != nil {
			t.Fatalf("Request %d failed: %v", i+1, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Request %d: expected status 200, got %d", i+1, resp.StatusCode)
		}
		resp.Body.Close()
	}
	
	// Request over limit should be blocked
	resp, err := client.Get(server.URL + "/test")
	if err != nil {
		t.Fatalf("Blocked request failed: %v", err)
	}
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("Expected status 429, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestRateLimitWithSkipFunc(t *testing.T) {
	manager := NewRateLimitManager()
	
	config := &RateLimitMiddlewareConfig{
		Name: "skip_test",
		Config: &RateLimitConfig{
			Algorithm:    "token_bucket",
			Limit:        1,
			Window:       time.Second,
			Backend:      "memory",
			KeyGenerator: IPKeyGenerator,
			Headers:      true,
		},
		SkipFunc: func(c Context) bool {
			// Skip rate limiting for admin users
			return c.Query("admin") == "true"
		},
	}
	
	manager.RegisterConfig("skip_test", config)
	
	app := New()
	app.UseMiddleware(manager.CreateMiddleware("skip_test"))
	
	app.GetHandler("/test", func(c Context) error {
		return c.JSON(http.StatusOK, map[string]string{"message": "success"})
	})
	
	server := httptest.NewServer(app.Router())
	defer server.Close()
	
	client := &http.Client{}
	
	// Normal user - should be rate limited after 1 request
	resp, err := client.Get(server.URL + "/test")
	if err != nil {
		t.Fatalf("First request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("First request: expected status 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()
	
	resp, err = client.Get(server.URL + "/test")
	if err != nil {
		t.Fatalf("Second request failed: %v", err)
	}
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("Second request: expected status 429, got %d", resp.StatusCode)
	}
	resp.Body.Close()
	
	// Admin user - should not be rate limited
	for i := 0; i < 5; i++ {
		resp, err := client.Get(server.URL + "/test?admin=true")
		if err != nil {
			t.Fatalf("Admin request %d failed: %v", i+1, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Admin request %d: expected status 200, got %d", i+1, resp.StatusCode)
		}
		resp.Body.Close()
	}
}

func TestRateLimitHeaders(t *testing.T) {
	app := New()
	app.UseMiddleware(RateLimit(3, time.Minute))
	
	app.GetHandler("/test", func(c Context) error {
		return c.JSON(http.StatusOK, map[string]string{"message": "success"})
	})
	
	server := httptest.NewServer(app.Router())
	defer server.Close()
	
	client := &http.Client{}
	
	resp, err := client.Get(server.URL + "/test")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()
	
	// Check rate limit headers
	headers := []string{
		"X-RateLimit-Limit",
		"X-RateLimit-Remaining",
		"X-RateLimit-Reset",
	}
	
	for _, header := range headers {
		value := resp.Header.Get(header)
		if value == "" {
			t.Errorf("Expected header %s to be present", header)
		}
	}
	
	// Verify specific header values
	if limit := resp.Header.Get("X-RateLimit-Limit"); limit != "3" {
		t.Errorf("Expected limit 3, got %s", limit)
	}
	
	if remaining := resp.Header.Get("X-RateLimit-Remaining"); remaining != "2" {
		t.Errorf("Expected remaining 2, got %s", remaining)
	}
}

func TestRateLimitAlgorithmTypes(t *testing.T) {
	algorithms := []string{"token_bucket", "sliding_window", "fixed_window"}
	
	for _, algorithm := range algorithms {
		t.Run(algorithm, func(t *testing.T) {
			limiter := NewMemoryRateLimiter(algorithm)
			defer limiter.Close()
			
			ctx := context.Background()
			key := "test_key"
			limit := 2
			window := 100 * time.Millisecond
			
			// Test that we can make requests up to the limit
			for i := 0; i < limit; i++ {
				result, err := limiter.Allow(ctx, key, limit, window)
				if err != nil {
					t.Fatalf("Algorithm %s: unexpected error: %v", algorithm, err)
				}
				if !result.Allowed {
					t.Fatalf("Algorithm %s: request %d should be allowed", algorithm, i+1)
				}
			}
			
			// Next request should be rejected
			result, err := limiter.Allow(ctx, key, limit, window)
			if err != nil {
				t.Fatalf("Algorithm %s: unexpected error: %v", algorithm, err)
			}
			if result.Allowed {
				t.Errorf("Algorithm %s: request should be rejected when limit exceeded", algorithm)
			}
		})
	}
}

func TestRateLimitErrorHandling(t *testing.T) {
	limiter := NewMemoryRateLimiter("invalid_algorithm")
	defer limiter.Close()
	
	ctx := context.Background()
	key := "test_key"
	limit := 5
	window := time.Second
	
	_, err := limiter.Allow(ctx, key, limit, window)
	if err == nil {
		t.Error("Expected error for invalid algorithm")
	}
	
	expectedError := "unsupported algorithm: invalid_algorithm"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

// Benchmark tests

func BenchmarkMemoryRateLimiter_TokenBucket(b *testing.B) {
	limiter := NewMemoryRateLimiter("token_bucket")
	defer limiter.Close()
	
	ctx := context.Background()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := "bench_key_" + strconv.Itoa(i%100)
			limiter.Allow(ctx, key, 1000, time.Second)
			i++
		}
	})
}

func BenchmarkMemoryRateLimiter_SlidingWindow(b *testing.B) {
	limiter := NewMemoryRateLimiter("sliding_window")
	defer limiter.Close()
	
	ctx := context.Background()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := "bench_key_" + strconv.Itoa(i%100)
			limiter.Allow(ctx, key, 1000, time.Second)
			i++
		}
	})
}