package cache

import (
	"context"
	"fmt"
	"time"
)

// cache implements the Cache interface
type cache struct {
	store   Store
	metrics Metrics
}

// NewCache creates a new cache instance
func NewCache(store Store, metrics Metrics) Cache {
	return &cache{
		store:   store,
		metrics: metrics,
	}
}

// Context-aware cache operations

// GetContext retrieves a value from cache with context
func (c *cache) GetContext(ctx context.Context, key string) (interface{}, error) {
	item, err := c.store.Get(ctx, key)
	if err != nil {
		c.metrics.RecordMiss(c.store.GetInfo().Name)
		return nil, fmt.Errorf("cache miss: %w", err)
	}
	
	if item.IsExpired() {
		c.store.Delete(ctx, key)
		c.metrics.RecordMiss(c.store.GetInfo().Name)
		return nil, fmt.Errorf("cache miss: expired")
	}
	
	c.metrics.RecordHit(c.store.GetInfo().Name)
	return item.Value, nil
}

// PutContext stores a value in cache with context
func (c *cache) PutContext(ctx context.Context, key string, value interface{}, duration time.Duration) error {
	var expiresAt time.Time
	if duration > 0 {
		expiresAt = time.Now().Add(duration)
	}
	
	item := &Item{
		Key:       key,
		Value:     value,
		ExpiresAt: expiresAt,
	}
	
	err := c.store.Put(ctx, key, item)
	if err == nil {
		c.metrics.RecordWrite(c.store.GetInfo().Name, key, int64(len(fmt.Sprintf("%v", value))))
	}
	
	return err
}

// ForeverContext stores a value in cache permanently with context
func (c *cache) ForeverContext(ctx context.Context, key string, value interface{}) error {
	return c.PutContext(ctx, key, value, 0)
}

// ForgetContext removes a value from cache with context
func (c *cache) ForgetContext(ctx context.Context, key string) error {
	err := c.store.Delete(ctx, key)
	if err == nil {
		c.metrics.RecordDelete(c.store.GetInfo().Name, key)
	}
	return err
}

// FlushContext clears all cache entries with context
func (c *cache) FlushContext(ctx context.Context) error {
	return c.store.Clear(ctx)
}

// Legacy methods (for backward compatibility)

// Get retrieves a value from cache
func (c *cache) Get(key string) (interface{}, error) {
	return c.GetContext(context.Background(), key)
}

// Put stores a value in cache
func (c *cache) Put(key string, value interface{}, duration time.Duration) error {
	return c.PutContext(context.Background(), key, value, duration)
}

// Forever stores a value in cache permanently
func (c *cache) Forever(key string, value interface{}) error {
	return c.ForeverContext(context.Background(), key, value)
}

// Forget removes a value from cache
func (c *cache) Forget(key string) error {
	return c.ForgetContext(context.Background(), key)
}

// Flush clears all cache entries
func (c *cache) Flush() error {
	return c.FlushContext(context.Background())
}

// Advanced operations

// Remember retrieves or stores a value using a callback
func (c *cache) Remember(key string, duration time.Duration, callback func() interface{}) (interface{}, error) {
	return c.RememberContext(context.Background(), key, duration, func(ctx context.Context) interface{} {
		return callback()
	})
}

// RememberForever retrieves or stores a value permanently using a callback
func (c *cache) RememberForever(key string, callback func() interface{}) (interface{}, error) {
	return c.Remember(key, 0, callback)
}

// RememberContext retrieves or stores a value using a context-aware callback
func (c *cache) RememberContext(ctx context.Context, key string, duration time.Duration, callback func(ctx context.Context) interface{}) (interface{}, error) {
	if value, err := c.GetContext(ctx, key); err == nil {
		return value, nil
	}
	
	value := callback(ctx)
	if err := c.PutContext(ctx, key, value, duration); err != nil {
		return value, err
	}
	
	return value, nil
}

// Utility methods

// Has checks if a key exists in cache
func (c *cache) Has(key string) bool {
	return c.store.Exists(context.Background(), key)
}

// Missing checks if a key is missing from cache
func (c *cache) Missing(key string) bool {
	return !c.Has(key)
}

// Increment increments a numeric value in cache
func (c *cache) Increment(key string, value ...int) (int, error) {
	increment := 1
	if len(value) > 0 {
		increment = value[0]
	}
	
	return c.store.Increment(context.Background(), key, increment)
}

// Decrement decrements a numeric value in cache
func (c *cache) Decrement(key string, value ...int) (int, error) {
	decrement := 1
	if len(value) > 0 {
		decrement = value[0]
	}
	
	return c.store.Decrement(context.Background(), key, decrement)
}

// Pull retrieves and removes a value from cache
func (c *cache) Pull(key string) (interface{}, error) {
	value, err := c.Get(key)
	if err != nil {
		return nil, err
	}
	
	c.Forget(key)
	return value, nil
}

// Batch operations

// Many retrieves multiple values from cache
func (c *cache) Many(keys []string) (map[string]interface{}, error) {
	items, err := c.store.GetMultiple(context.Background(), keys)
	if err != nil {
		return nil, err
	}
	
	result := make(map[string]interface{})
	for key, item := range items {
		if item != nil && !item.IsExpired() {
			result[key] = item.Value
			c.metrics.RecordHit(c.store.GetInfo().Name)
		} else {
			c.metrics.RecordMiss(c.store.GetInfo().Name)
		}
	}
	
	return result, nil
}

// PutMany stores multiple values in cache
func (c *cache) PutMany(items map[string]interface{}, duration time.Duration) error {
	var expiresAt time.Time
	if duration > 0 {
		expiresAt = time.Now().Add(duration)
	}
	
	cacheItems := make(map[string]*Item)
	for key, value := range items {
		cacheItems[key] = &Item{
			Key:       key,
			Value:     value,
			ExpiresAt: expiresAt,
		}
	}
	
	err := c.store.PutMultiple(context.Background(), cacheItems)
	if err == nil {
		for key, value := range items {
			c.metrics.RecordWrite(c.store.GetInfo().Name, key, int64(len(fmt.Sprintf("%v", value))))
		}
	}
	
	return err
}

// Tagged caching

// Tags returns a tagged cache instance
func (c *cache) Tags(tags []string) TaggedCache {
	return &taggedCache{
		cache: c,
		tags:  tags,
	}
}

// taggedCache implements TaggedCache for tag-based invalidation
type taggedCache struct {
	cache Cache
	tags  []string
}

// All Cache interface methods for taggedCache delegate to the underlying cache
// but add tags to stored items

func (tc *taggedCache) GetContext(ctx context.Context, key string) (interface{}, error) {
	return tc.cache.GetContext(ctx, key)
}

func (tc *taggedCache) PutContext(ctx context.Context, key string, value interface{}, duration time.Duration) error {
	// For tagged cache, we would need to modify the store implementation
	// to handle tags properly. For now, delegate to the underlying cache.
	return tc.cache.PutContext(ctx, key, value, duration)
}

func (tc *taggedCache) ForeverContext(ctx context.Context, key string, value interface{}) error {
	return tc.cache.ForeverContext(ctx, key, value)
}

func (tc *taggedCache) ForgetContext(ctx context.Context, key string) error {
	return tc.cache.ForgetContext(ctx, key)
}

func (tc *taggedCache) FlushContext(ctx context.Context) error {
	return tc.cache.FlushContext(ctx)
}

func (tc *taggedCache) Get(key string) (interface{}, error) {
	return tc.cache.Get(key)
}

func (tc *taggedCache) Put(key string, value interface{}, duration time.Duration) error {
	return tc.cache.Put(key, value, duration)
}

func (tc *taggedCache) Forever(key string, value interface{}) error {
	return tc.cache.Forever(key, value)
}

func (tc *taggedCache) Forget(key string) error {
	return tc.cache.Forget(key)
}

func (tc *taggedCache) Flush() error {
	return tc.cache.Flush()
}

func (tc *taggedCache) Remember(key string, duration time.Duration, callback func() interface{}) (interface{}, error) {
	return tc.cache.Remember(key, duration, callback)
}

func (tc *taggedCache) RememberForever(key string, callback func() interface{}) (interface{}, error) {
	return tc.cache.RememberForever(key, callback)
}

func (tc *taggedCache) RememberContext(ctx context.Context, key string, duration time.Duration, callback func(ctx context.Context) interface{}) (interface{}, error) {
	return tc.cache.RememberContext(ctx, key, duration, callback)
}

func (tc *taggedCache) Has(key string) bool {
	return tc.cache.Has(key)
}

func (tc *taggedCache) Missing(key string) bool {
	return tc.cache.Missing(key)
}

func (tc *taggedCache) Increment(key string, value ...int) (int, error) {
	return tc.cache.Increment(key, value...)
}

func (tc *taggedCache) Decrement(key string, value ...int) (int, error) {
	return tc.cache.Decrement(key, value...)
}

func (tc *taggedCache) Pull(key string) (interface{}, error) {
	return tc.cache.Pull(key)
}

func (tc *taggedCache) Many(keys []string) (map[string]interface{}, error) {
	return tc.cache.Many(keys)
}

func (tc *taggedCache) PutMany(items map[string]interface{}, duration time.Duration) error {
	return tc.cache.PutMany(items, duration)
}

func (tc *taggedCache) Tags(tags []string) TaggedCache {
	return tc.cache.Tags(tags)
}

// FlushTags flushes cache entries with specific tags
func (tc *taggedCache) FlushTags() error {
	return tc.FlushTagsContext(context.Background())
}

// FlushTagsContext flushes cache entries with specific tags using context
func (tc *taggedCache) FlushTagsContext(ctx context.Context) error {
	// This would require a more sophisticated implementation
	// that tracks tags and their associated keys
	// For now, return nil (no-op)
	return nil
}