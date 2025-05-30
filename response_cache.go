package onyx

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ResponseCacheConfig configures response caching
type ResponseCacheConfig struct {
	Enabled        bool          `json:"enabled"`
	DefaultTTL     time.Duration `json:"default_ttl"`
	MaxCacheSize   int           `json:"max_cache_size"`
	IncludedPaths  []string      `json:"included_paths"`
	ExcludedPaths  []string      `json:"excluded_paths"`
	IncludedMethods []string      `json:"included_methods"`
	VaryHeaders    []string      `json:"vary_headers"`
}

// DefaultResponseCacheConfig returns sensible defaults
func DefaultResponseCacheConfig() ResponseCacheConfig {
	return ResponseCacheConfig{
		Enabled:         true,
		DefaultTTL:      5 * time.Minute,
		MaxCacheSize:    1000,
		IncludedPaths:   []string{}, // Empty means cache all paths except excluded
		ExcludedPaths:   []string{"/admin/", "/auth/"},
		IncludedMethods: []string{"GET", "HEAD"},
		VaryHeaders:     []string{"Accept", "Accept-Encoding"},
	}
}

// CachedResponse represents a cached HTTP response
type CachedResponse struct {
	StatusCode int                 `json:"status_code"`
	Headers    map[string][]string `json:"headers"`
	Body       []byte              `json:"body"`
	ExpiresAt  time.Time           `json:"expires_at"`
	CreatedAt  time.Time           `json:"created_at"`
}

// ResponseCache manages HTTP response caching
type ResponseCache struct {
	config  ResponseCacheConfig
	cache   map[string]*CachedResponse
	mutex   sync.RWMutex
	metrics ResponseCacheMetrics
}

// ResponseCacheMetrics tracks cache performance
type ResponseCacheMetrics struct {
	Hits         int64     `json:"hits"`
	Misses       int64     `json:"misses"`
	Stores       int64     `json:"stores"`
	Evictions    int64     `json:"evictions"`
	TotalSize    int       `json:"total_size"`
	LastActivity time.Time `json:"last_activity"`
}

// NewResponseCache creates a new response cache
func NewResponseCache(config ResponseCacheConfig) *ResponseCache {
	return &ResponseCache{
		config: config,
		cache:  make(map[string]*CachedResponse),
	}
}

// Global cache instance for middleware
var globalResponseCache *ResponseCache
var globalCacheConfig ResponseCacheConfig

// ResponseCacheMiddleware returns middleware for response caching
func ResponseCacheMiddleware(config ...ResponseCacheConfig) MiddlewareFunc {
	var cfg ResponseCacheConfig
	if len(config) > 0 {
		cfg = config[0]
	} else {
		cfg = DefaultResponseCacheConfig()
	}
	
	if !cfg.Enabled {
		return func(c Context) error {
			return c.Next()
		}
	}
	
	// Use global cache or create if config changed
	if globalResponseCache == nil {
		globalResponseCache = NewResponseCache(cfg)
		globalCacheConfig = cfg
	}
	cache := globalResponseCache
	
	return func(c Context) error {
		// Check if request should be cached
		if !cache.shouldCache(c.Request()) {
			return c.Next()
		}
		
		// Generate cache key
		cacheKey := cache.generateKey(c.Request())
		
		// Try to get from cache
		if cached := cache.get(cacheKey); cached != nil {
			cache.serveCached(c, cached)
			c.Abort()
			return nil
		}
		
		// TODO: Implement response capture with interface-based system
		// For now, just pass through without caching response bodies
		// Response caching would need a different approach with the interface system
		return c.Next()
	}
}

// ResponseRecorder captures response data for caching
type ResponseRecorder struct {
	http.ResponseWriter
	body       []byte
	statusCode int
	headers    http.Header
}

// WriteHeader captures the status code
func (rr *ResponseRecorder) WriteHeader(statusCode int) {
	rr.statusCode = statusCode
	if rr.headers == nil {
		rr.headers = make(http.Header)
		for k, v := range rr.ResponseWriter.Header() {
			rr.headers[k] = v
		}
	}
	rr.ResponseWriter.WriteHeader(statusCode)
}

// Write captures the response body
func (rr *ResponseRecorder) Write(data []byte) (int, error) {
	rr.body = append(rr.body, data...)
	return rr.ResponseWriter.Write(data)
}

// shouldCache determines if a request should be cached
func (rc *ResponseCache) shouldCache(req *http.Request) bool {
	// Check method
	methodAllowed := false
	for _, method := range rc.config.IncludedMethods {
		if req.Method == method {
			methodAllowed = true
			break
		}
	}
	if !methodAllowed {
		return false
	}
	
	path := req.URL.Path
	
	// Check excluded paths
	for _, excluded := range rc.config.ExcludedPaths {
		if strings.HasPrefix(path, excluded) {
			return false
		}
	}
	
	// Check included paths (if any specified)
	if len(rc.config.IncludedPaths) > 0 {
		pathIncluded := false
		for _, included := range rc.config.IncludedPaths {
			if strings.HasPrefix(path, included) {
				pathIncluded = true
				break
			}
		}
		if !pathIncluded {
			return false
		}
	}
	
	return true
}

// generateKey creates a cache key for the request
func (rc *ResponseCache) generateKey(req *http.Request) string {
	key := fmt.Sprintf("%s:%s", req.Method, req.URL.Path)
	
	// Include query parameters
	if req.URL.RawQuery != "" {
		key += "?" + req.URL.RawQuery
	}
	
	// Include vary headers
	for _, header := range rc.config.VaryHeaders {
		if value := req.Header.Get(header); value != "" {
			key += ":" + header + "=" + value
		}
	}
	
	// Hash the key to keep it manageable
	hash := md5.Sum([]byte(key))
	return fmt.Sprintf("%x", hash)
}

// get retrieves a cached response
func (rc *ResponseCache) get(key string) *CachedResponse {
	rc.mutex.Lock()
	defer rc.mutex.Unlock()
	
	cached, exists := rc.cache[key]
	if !exists {
		rc.metrics.Misses++
		rc.metrics.LastActivity = time.Now()
		return nil
	}
	
	// Check expiration
	if time.Now().After(cached.ExpiresAt) {
		delete(rc.cache, key)
		rc.metrics.Misses++
		rc.metrics.LastActivity = time.Now()
		return nil
	}
	
	rc.metrics.Hits++
	rc.metrics.LastActivity = time.Now()
	return cached
}

// store caches a response
func (rc *ResponseCache) store(key string, recorder *ResponseRecorder, ttl time.Duration) {
	rc.mutex.Lock()
	defer rc.mutex.Unlock()
	
	// Check cache size limit
	if len(rc.cache) >= rc.config.MaxCacheSize {
		rc.evictOldest()
	}
	
	cached := &CachedResponse{
		StatusCode: recorder.statusCode,
		Headers:    make(map[string][]string),
		Body:       make([]byte, len(recorder.body)),
		ExpiresAt:  time.Now().Add(ttl),
		CreatedAt:  time.Now(),
	}
	
	// Copy headers
	if recorder.headers != nil {
		for k, v := range recorder.headers {
			cached.Headers[k] = make([]string, len(v))
			copy(cached.Headers[k], v)
		}
	} else {
		for k, v := range recorder.ResponseWriter.Header() {
			cached.Headers[k] = make([]string, len(v))
			copy(cached.Headers[k], v)
		}
	}
	
	// Copy body
	copy(cached.Body, recorder.body)
	
	rc.cache[key] = cached
	rc.metrics.Stores++
	rc.metrics.TotalSize = len(rc.cache)
	rc.metrics.LastActivity = time.Now()
}

// evictOldest removes the oldest cache entry
func (rc *ResponseCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time
	first := true
	
	for key, cached := range rc.cache {
		if first || cached.CreatedAt.Before(oldestTime) {
			oldestTime = cached.CreatedAt
			oldestKey = key
			first = false
		}
	}
	
	if oldestKey != "" {
		delete(rc.cache, oldestKey)
		rc.metrics.Evictions++
	}
}

// serveCached serves a cached response
func (rc *ResponseCache) serveCached(c Context, cached *CachedResponse) {
	// Set headers
	for k, v := range cached.Headers {
		for _, value := range v {
			c.ResponseWriter().Header().Add(k, value)
		}
	}
	
	// Add cache headers
	c.ResponseWriter().Header().Set("X-Cache", "HIT")
	c.ResponseWriter().Header().Set("X-Cache-Expires", cached.ExpiresAt.Format(time.RFC1123))
	
	// Write status and body
	c.ResponseWriter().WriteHeader(cached.StatusCode)
	c.ResponseWriter().Write(cached.Body)
}

// ClearCache clears all cached responses
func (rc *ResponseCache) ClearCache() {
	rc.mutex.Lock()
	defer rc.mutex.Unlock()
	
	rc.cache = make(map[string]*CachedResponse)
	rc.metrics.TotalSize = 0
}

// GetMetrics returns cache performance metrics
func (rc *ResponseCache) GetMetrics() ResponseCacheMetrics {
	rc.mutex.RLock()
	defer rc.mutex.RUnlock()
	
	metrics := rc.metrics
	metrics.TotalSize = len(rc.cache)
	
	return metrics
}

// GetCacheInfo returns detailed cache information
func (rc *ResponseCache) GetCacheInfo() map[string]interface{} {
	rc.mutex.RLock()
	defer rc.mutex.RUnlock()
	
	metrics := rc.metrics
	totalRequests := metrics.Hits + metrics.Misses
	hitRate := float64(0)
	if totalRequests > 0 {
		hitRate = float64(metrics.Hits) / float64(totalRequests)
	}
	
	return map[string]interface{}{
		"enabled":        rc.config.Enabled,
		"total_entries":  len(rc.cache),
		"max_size":       rc.config.MaxCacheSize,
		"hits":           metrics.Hits,
		"misses":         metrics.Misses,
		"hit_rate":       hitRate,
		"stores":         metrics.Stores,
		"evictions":      metrics.Evictions,
		"last_activity":  metrics.LastActivity,
		"default_ttl":    rc.config.DefaultTTL,
	}
}