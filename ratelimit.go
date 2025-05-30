package onyx

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Rate limiting interfaces and types

// RateLimiter interface defines the contract for rate limiting implementations
type RateLimiter interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (*RateLimitResult, error)
	Reset(ctx context.Context, key string) error
	GetRemaining(ctx context.Context, key string, limit int, window time.Duration) (int, error)
}

// RateLimitResult contains information about a rate limit check
type RateLimitResult struct {
	Allowed       bool          `json:"allowed"`
	Limit         int           `json:"limit"`
	Remaining     int           `json:"remaining"`
	RetryAfter    time.Duration `json:"retry_after"`
	ResetTime     time.Time     `json:"reset_time"`
	TotalHits     int           `json:"total_hits"`
	WindowStart   time.Time     `json:"window_start"`
}

// RateLimitConfig defines configuration for rate limiting
type RateLimitConfig struct {
	Algorithm    string            `json:"algorithm"`     // "token_bucket", "sliding_window", "fixed_window"
	Limit        int               `json:"limit"`         // Number of requests allowed
	Window       time.Duration     `json:"window"`        // Time window for the limit
	Burst        int               `json:"burst"`         // Burst capacity for token bucket
	Backend      string            `json:"backend"`       // "memory", "redis"
	RedisConfig  *RedisConfig      `json:"redis_config"`  // Redis configuration
	KeyGenerator KeyGeneratorFunc  `json:"-"`             // Function to generate rate limit keys
	Headers      bool              `json:"headers"`       // Whether to include rate limit headers
	Options      map[string]string `json:"options"`       // Additional options
}

// RedisConfig contains Redis connection configuration
type RedisConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Password string `json:"password"`
	Database int    `json:"database"`
	PoolSize int    `json:"pool_size"`
}

// KeyGeneratorFunc generates rate limit keys based on the request
type KeyGeneratorFunc func(c Context) string

// RateLimitMiddlewareConfig contains middleware-specific configuration
type RateLimitMiddlewareConfig struct {
	Name         string              `json:"name"`
	Config       *RateLimitConfig    `json:"config"`
	OnExceeded   RateLimitHandler    `json:"-"`
	SkipFunc     func(c Context) bool `json:"-"`
	ErrorHandler func(c Context, err error) `json:"-"`
}

// RateLimitHandler handles rate limit exceeded scenarios
type RateLimitHandler func(c Context, result *RateLimitResult)

// Default key generators

// IPKeyGenerator generates keys based on client IP
func IPKeyGenerator(c Context) string {
	ip := c.Request().RemoteAddr
	if colonIndex := strings.LastIndex(ip, ":"); colonIndex != -1 {
		ip = ip[:colonIndex]
	}
	return fmt.Sprintf("rate_limit:ip:%s", ip)
}

// UserKeyGenerator generates keys based on authenticated user ID
func UserKeyGenerator(c Context) string {
	userID := "anonymous"
	// In a real implementation, you'd extract user ID from context/auth
	if authUser, exists := c.Get("user_id"); exists && authUser != nil {
		userID = fmt.Sprintf("%v", authUser)
	}
	return fmt.Sprintf("rate_limit:user:%s", userID)
}

// RouteKeyGenerator generates keys based on route pattern
func RouteKeyGenerator(c Context) string {
	return fmt.Sprintf("rate_limit:route:%s:%s", c.Request().Method, c.Request().URL.Path)
}

// CompositeKeyGenerator combines multiple key generators
func CompositeKeyGenerator(generators ...KeyGeneratorFunc) KeyGeneratorFunc {
	return func(c Context) string {
		var parts []string
		for _, gen := range generators {
			parts = append(parts, gen(c))
		}
		return strings.Join(parts, ":")
	}
}

// In-Memory Rate Limiter Implementation

// MemoryRateLimiter implements rate limiting using in-memory storage
type MemoryRateLimiter struct {
	algorithm string
	buckets   map[string]*tokenBucket
	windows   map[string]*slidingWindow
	mutex     sync.RWMutex
	cleaner   *time.Ticker
	done      chan struct{}
}

// tokenBucket represents a token bucket for rate limiting
type tokenBucket struct {
	capacity    int
	tokens      int
	refillRate  time.Duration
	lastRefill  time.Time
	mutex       sync.Mutex
}

// slidingWindow represents a sliding window for rate limiting
type slidingWindow struct {
	limit       int
	window      time.Duration
	requests    []time.Time
	mutex       sync.Mutex
}

// NewMemoryRateLimiter creates a new in-memory rate limiter
func NewMemoryRateLimiter(algorithm string) *MemoryRateLimiter {
	limiter := &MemoryRateLimiter{
		algorithm: algorithm,
		buckets:   make(map[string]*tokenBucket),
		windows:   make(map[string]*slidingWindow),
		cleaner:   time.NewTicker(5 * time.Minute),
		done:      make(chan struct{}),
	}
	
	// Start cleanup goroutine
	go limiter.cleanup()
	
	return limiter
}

func (m *MemoryRateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (*RateLimitResult, error) {
	switch m.algorithm {
	case "token_bucket":
		return m.allowTokenBucket(key, limit, window)
	case "sliding_window":
		return m.allowSlidingWindow(key, limit, window)
	case "fixed_window":
		return m.allowFixedWindow(key, limit, window)
	default:
		return nil, fmt.Errorf("unsupported algorithm: %s", m.algorithm)
	}
}

func (m *MemoryRateLimiter) allowTokenBucket(key string, limit int, window time.Duration) (*RateLimitResult, error) {
	m.mutex.Lock()
	bucket, exists := m.buckets[key]
	if !exists {
		bucket = &tokenBucket{
			capacity:   limit,
			tokens:     limit,
			refillRate: window / time.Duration(limit),
			lastRefill: time.Now(),
		}
		m.buckets[key] = bucket
	}
	m.mutex.Unlock()
	
	bucket.mutex.Lock()
	defer bucket.mutex.Unlock()
	
	now := time.Now()
	
	// Refill tokens based on elapsed time
	elapsed := now.Sub(bucket.lastRefill)
	tokensToAdd := int(elapsed / bucket.refillRate)
	if tokensToAdd > 0 {
		bucket.tokens = min(bucket.capacity, bucket.tokens+tokensToAdd)
		bucket.lastRefill = now
	}
	
	result := &RateLimitResult{
		Limit:     limit,
		Remaining: bucket.tokens,
		ResetTime: now.Add(bucket.refillRate),
	}
	
	if bucket.tokens > 0 {
		bucket.tokens--
		result.Allowed = true
		result.Remaining = bucket.tokens
	} else {
		result.Allowed = false
		result.RetryAfter = bucket.refillRate
	}
	
	return result, nil
}

func (m *MemoryRateLimiter) allowSlidingWindow(key string, limit int, window time.Duration) (*RateLimitResult, error) {
	m.mutex.Lock()
	sw, exists := m.windows[key]
	if !exists {
		sw = &slidingWindow{
			limit:    limit,
			window:   window,
			requests: make([]time.Time, 0),
		}
		m.windows[key] = sw
	}
	m.mutex.Unlock()
	
	sw.mutex.Lock()
	defer sw.mutex.Unlock()
	
	now := time.Now()
	cutoff := now.Add(-window)
	
	// Remove expired requests
	var validRequests []time.Time
	for _, req := range sw.requests {
		if req.After(cutoff) {
			validRequests = append(validRequests, req)
		}
	}
	sw.requests = validRequests
	
	result := &RateLimitResult{
		Limit:       limit,
		TotalHits:   len(sw.requests),
		Remaining:   max(0, limit-len(sw.requests)),
		WindowStart: cutoff,
	}
	
	if len(sw.requests) < limit {
		sw.requests = append(sw.requests, now)
		result.Allowed = true
		result.Remaining = limit - len(sw.requests)
	} else {
		result.Allowed = false
		if len(sw.requests) > 0 {
			oldestRequest := sw.requests[0]
			result.RetryAfter = window - now.Sub(oldestRequest)
			result.ResetTime = oldestRequest.Add(window)
		}
	}
	
	return result, nil
}

func (m *MemoryRateLimiter) allowFixedWindow(key string, limit int, window time.Duration) (*RateLimitResult, error) {
	// Fixed window is a simplified version of sliding window
	// where the window resets at fixed intervals
	windowKey := fmt.Sprintf("%s:%d", key, time.Now().Truncate(window).Unix())
	return m.allowSlidingWindow(windowKey, limit, window)
}

func (m *MemoryRateLimiter) Reset(ctx context.Context, key string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	delete(m.buckets, key)
	delete(m.windows, key)
	return nil
}

func (m *MemoryRateLimiter) GetRemaining(ctx context.Context, key string, limit int, window time.Duration) (int, error) {
	result, err := m.Allow(ctx, key, limit, window)
	if err != nil {
		return 0, err
	}
	return result.Remaining, nil
}

func (m *MemoryRateLimiter) cleanup() {
	for {
		select {
		case <-m.cleaner.C:
			m.cleanupExpired()
		case <-m.done:
			m.cleaner.Stop()
			return
		}
	}
}

func (m *MemoryRateLimiter) cleanupExpired() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	now := time.Now()
	
	// Clean up expired sliding windows
	for key, sw := range m.windows {
		sw.mutex.Lock()
		cutoff := now.Add(-sw.window * 2) // Keep some buffer
		var validRequests []time.Time
		for _, req := range sw.requests {
			if req.After(cutoff) {
				validRequests = append(validRequests, req)
			}
		}
		if len(validRequests) == 0 {
			delete(m.windows, key)
		} else {
			sw.requests = validRequests
		}
		sw.mutex.Unlock()
	}
}

func (m *MemoryRateLimiter) Close() error {
	close(m.done)
	return nil
}

// Rate Limiting Middleware

// RateLimitManager manages multiple rate limiters and configurations
type RateLimitManager struct {
	limiters map[string]RateLimiter
	configs  map[string]*RateLimitMiddlewareConfig
	mutex    sync.RWMutex
}

// NewRateLimitManager creates a new rate limit manager
func NewRateLimitManager() *RateLimitManager {
	return &RateLimitManager{
		limiters: make(map[string]RateLimiter),
		configs:  make(map[string]*RateLimitMiddlewareConfig),
	}
}

// RegisterLimiter registers a rate limiter with a name
func (rlm *RateLimitManager) RegisterLimiter(name string, limiter RateLimiter) {
	rlm.mutex.Lock()
	defer rlm.mutex.Unlock()
	rlm.limiters[name] = limiter
}

// RegisterConfig registers a rate limiting configuration
func (rlm *RateLimitManager) RegisterConfig(name string, config *RateLimitMiddlewareConfig) {
	rlm.mutex.Lock()
	defer rlm.mutex.Unlock()
	rlm.configs[name] = config
}

// GetLimiter gets a rate limiter by name
func (rlm *RateLimitManager) GetLimiter(name string) (RateLimiter, bool) {
	rlm.mutex.RLock()
	defer rlm.mutex.RUnlock()
	limiter, exists := rlm.limiters[name]
	return limiter, exists
}

// GetConfig gets a rate limiting configuration by name
func (rlm *RateLimitManager) GetConfig(name string) (*RateLimitMiddlewareConfig, bool) {
	rlm.mutex.RLock()
	defer rlm.mutex.RUnlock()
	config, exists := rlm.configs[name]
	return config, exists
}

// CreateMiddleware creates rate limiting middleware
func (rlm *RateLimitManager) CreateMiddleware(configName string) MiddlewareFunc {
	// Create the limiter once when the middleware is created
	rlm.mutex.RLock()
	config, exists := rlm.configs[configName]
	rlm.mutex.RUnlock()
	
	if !exists {
		// Return a middleware that always errors
		return func(c Context) error {
			c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Rate limit configuration error",
			})
			return nil
		}
	}
	
	// Create rate limiter once
	var limiter RateLimiter
	switch config.Config.Backend {
	case "memory":
		limiter = NewMemoryRateLimiter(config.Config.Algorithm)
	default:
		limiter = NewMemoryRateLimiter("token_bucket") // Default fallback
	}
	
	return func(c Context) error {
		// Check if request should be skipped
		if config.SkipFunc != nil && config.SkipFunc(c) {
			c.Next()
			return nil
		}
		
		// Generate rate limit key
		var key string
		if config.Config.KeyGenerator != nil {
			key = config.Config.KeyGenerator(c)
		} else {
			key = IPKeyGenerator(c) // Default to IP-based limiting
		}
		
		// Check rate limit
		result, err := limiter.Allow(c.Request().Context(), key, config.Config.Limit, config.Config.Window)
		if err != nil {
			if config.ErrorHandler != nil {
				config.ErrorHandler(c, err)
			} else {
				c.JSON(http.StatusInternalServerError, map[string]string{
					"error": "Rate limiting error",
				})
			}
			return nil
		}
		
		// Add rate limit headers if enabled
		if config.Config.Headers {
			c.SetHeader("X-RateLimit-Limit", strconv.Itoa(result.Limit))
			c.SetHeader("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
			if !result.ResetTime.IsZero() {
				c.SetHeader("X-RateLimit-Reset", strconv.FormatInt(result.ResetTime.Unix(), 10))
			}
		}
		
		// Handle rate limit exceeded
		if !result.Allowed {
			if result.RetryAfter > 0 {
				c.SetHeader("Retry-After", strconv.Itoa(int(result.RetryAfter.Seconds())))
			}
			
			if config.OnExceeded != nil {
				config.OnExceeded(c, result)
			} else {
				c.JSON(http.StatusTooManyRequests, map[string]interface{}{
					"error":     "Too Many Requests",
					"message":   "Rate limit exceeded",
					"limit":     result.Limit,
					"remaining": result.Remaining,
					"retry_after": int(result.RetryAfter.Seconds()),
				})
			}
			return nil
		}
		
		c.Next()
		return nil
	}
}

// Pre-configured rate limiting middleware functions

// RateLimit creates a basic rate limiting middleware
func RateLimit(limit int, window time.Duration) MiddlewareFunc {
	// Create a single limiter instance for this middleware
	limiter := NewMemoryRateLimiter("token_bucket")
	
	return func(c Context) error {
		// Generate rate limit key
		key := IPKeyGenerator(c)
		
		// Check rate limit
		result, err := limiter.Allow(c.Request().Context(), key, limit, window)
		if err != nil {
			c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Rate limiting error",
			})
			return nil
		}
		
		// Add rate limit headers
		c.SetHeader("X-RateLimit-Limit", strconv.Itoa(result.Limit))
		c.SetHeader("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
		if !result.ResetTime.IsZero() {
			c.SetHeader("X-RateLimit-Reset", strconv.FormatInt(result.ResetTime.Unix(), 10))
		}
		
		// Handle rate limit exceeded
		if !result.Allowed {
			if result.RetryAfter > 0 {
				c.SetHeader("Retry-After", strconv.Itoa(int(result.RetryAfter.Seconds())))
			}
			
			c.JSON(http.StatusTooManyRequests, map[string]interface{}{
				"error":       "Too Many Requests",
				"message":     "Rate limit exceeded",
				"limit":       result.Limit,
				"remaining":   result.Remaining,
				"retry_after": int(result.RetryAfter.Seconds()),
			})
			return nil
		}
		
		c.Next()
		return nil
	}
}

// RateLimitPerUser creates user-based rate limiting middleware
func RateLimitPerUser(limit int, window time.Duration) MiddlewareFunc {
	// Create a single limiter instance for this middleware
	limiter := NewMemoryRateLimiter("sliding_window")
	
	return func(c Context) error {
		// Generate rate limit key
		key := UserKeyGenerator(c)
		
		// Check rate limit
		result, err := limiter.Allow(c.Request().Context(), key, limit, window)
		if err != nil {
			c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Rate limiting error",
			})
			return nil
		}
		
		// Add rate limit headers
		c.SetHeader("X-RateLimit-Limit", strconv.Itoa(result.Limit))
		c.SetHeader("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
		if !result.ResetTime.IsZero() {
			c.SetHeader("X-RateLimit-Reset", strconv.FormatInt(result.ResetTime.Unix(), 10))
		}
		
		// Handle rate limit exceeded
		if !result.Allowed {
			if result.RetryAfter > 0 {
				c.SetHeader("Retry-After", strconv.Itoa(int(result.RetryAfter.Seconds())))
			}
			
			c.JSON(http.StatusTooManyRequests, map[string]interface{}{
				"error":       "Too Many Requests",
				"message":     "Rate limit exceeded",
				"limit":       result.Limit,
				"remaining":   result.Remaining,
				"retry_after": int(result.RetryAfter.Seconds()),
			})
			return nil
		}
		
		c.Next()
		return nil
	}
}

// RateLimitPerRoute creates route-based rate limiting middleware
func RateLimitPerRoute(limit int, window time.Duration) MiddlewareFunc {
	config := &RateLimitMiddlewareConfig{
		Name: "per_route",
		Config: &RateLimitConfig{
			Algorithm:    "fixed_window",
			Limit:        limit,
			Window:       window,
			Backend:      "memory",
			KeyGenerator: RouteKeyGenerator,
			Headers:      true,
		},
	}
	
	manager := NewRateLimitManager()
	manager.RegisterConfig("per_route", config)
	
	return manager.CreateMiddleware("per_route")
}

// CustomRateLimit creates custom rate limiting middleware
func CustomRateLimit(config *RateLimitConfig) MiddlewareFunc {
	middlewareConfig := &RateLimitMiddlewareConfig{
		Name:   "custom",
		Config: config,
	}
	
	manager := NewRateLimitManager()
	manager.RegisterConfig("custom", middlewareConfig)
	
	return manager.CreateMiddleware("custom")
}

// Utility functions

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Global rate limit manager
var globalRateLimitManager *RateLimitManager

// SetGlobalRateLimitManager sets the global rate limit manager
func SetGlobalRateLimitManager(manager *RateLimitManager) {
	globalRateLimitManager = manager
}

// RateLimitManager returns the global rate limit manager
func GetRateLimitManager() *RateLimitManager {
	if globalRateLimitManager == nil {
		globalRateLimitManager = NewRateLimitManager()
		
		// Register default memory limiter
		memoryLimiter := NewMemoryRateLimiter("token_bucket")
		globalRateLimitManager.RegisterLimiter("memory", memoryLimiter)
	}
	return globalRateLimitManager
}

// Context helper function for rate limiting
func GetRateLimitFromContext(c Context) *RateLimitManager {
	// Use global rate limit manager since Application interface doesn't expose Container
	// TODO: Extend Application interface to provide access to Container
	return GetRateLimitManager()
}