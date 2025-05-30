package config

import (
	"sync"
	"time"
)

// Cache handles configuration caching
type Cache struct {
	enabled   bool
	ttl       time.Duration
	cached    map[string]cachedValue
	mutex     sync.RWMutex
	lastLoad  time.Time
}

type cachedValue struct {
	value     interface{}
	expiresAt time.Time
}

// NewCache creates a new configuration cache
func NewCache() *Cache {
	return &Cache{
		enabled: true,
		ttl:     5 * time.Minute,
		cached:  make(map[string]cachedValue),
	}
}

// Get retrieves a cached value
func (c *Cache) Get(key string) (interface{}, bool) {
	if !c.enabled {
		return nil, false
	}
	
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	if cached, exists := c.cached[key]; exists {
		if time.Now().Before(cached.expiresAt) {
			return cached.value, true
		}
		// Expired, remove it
		delete(c.cached, key)
	}
	
	return nil, false
}

// Set stores a value in cache
func (c *Cache) Set(key string, value interface{}) {
	if !c.enabled {
		return
	}
	
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	c.cached[key] = cachedValue{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// Delete removes a value from cache
func (c *Cache) Delete(key string) {
	if !c.enabled {
		return
	}
	
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	delete(c.cached, key)
}

// Clear removes all cached values
func (c *Cache) Clear() {
	if !c.enabled {
		return
	}
	
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	c.cached = make(map[string]cachedValue)
}

// Enable enables caching with the specified TTL
func (c *Cache) Enable(ttl time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	c.enabled = true
	c.ttl = ttl
}

// Disable disables caching
func (c *Cache) Disable() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	c.enabled = false
}

// IsEnabled returns whether caching is enabled
func (c *Cache) IsEnabled() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	return c.enabled
}

// SetLastLoad sets the last load time
func (c *Cache) SetLastLoad(t time.Time) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	c.lastLoad = t
}

// GetLastLoad returns the last load time
func (c *Cache) GetLastLoad() time.Time {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	return c.lastLoad
}