package onyx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Cache interface {
	Get(key string) (interface{}, error)
	Put(key string, value interface{}, duration time.Duration) error
	Forever(key string, value interface{}) error
	Forget(key string) error
	Flush() error
	Remember(key string, duration time.Duration, callback func() interface{}) (interface{}, error)
	RememberForever(key string, callback func() interface{}) (interface{}, error)
	Has(key string) bool
	Missing(key string) bool
	Increment(key string, value ...int) (int, error)
	Decrement(key string, value ...int) (int, error)
	Pull(key string) (interface{}, error)
	Many(keys []string) (map[string]interface{}, error)
	PutMany(items map[string]interface{}, duration time.Duration) error
	Tags(tags []string) TaggedCache
}

type TaggedCache interface {
	Cache
	FlushTags() error
}

type CacheItem struct {
	Value     interface{} `json:"value"`
	ExpiresAt time.Time   `json:"expires_at"`
	Tags      []string    `json:"tags,omitempty"`
}

func (ci *CacheItem) IsExpired() bool {
	return !ci.ExpiresAt.IsZero() && time.Now().After(ci.ExpiresAt)
}

type MemoryCache struct {
	items map[string]*CacheItem
	mutex sync.RWMutex
	tags  []string
}

func NewMemoryCache() *MemoryCache {
	cache := &MemoryCache{
		items: make(map[string]*CacheItem),
	}
	
	go cache.cleanup()
	return cache
}

func (mc *MemoryCache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		mc.mutex.Lock()
		for key, item := range mc.items {
			if item.IsExpired() {
				delete(mc.items, key)
			}
		}
		mc.mutex.Unlock()
	}
}

func (mc *MemoryCache) Get(key string) (interface{}, error) {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()
	
	item, exists := mc.items[key]
	if !exists {
		return nil, fmt.Errorf("cache miss")
	}
	
	if item.IsExpired() {
		delete(mc.items, key)
		return nil, fmt.Errorf("cache miss")
	}
	
	return item.Value, nil
}

func (mc *MemoryCache) Put(key string, value interface{}, duration time.Duration) error {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	
	var expiresAt time.Time
	if duration > 0 {
		expiresAt = time.Now().Add(duration)
	}
	
	mc.items[key] = &CacheItem{
		Value:     value,
		ExpiresAt: expiresAt,
		Tags:      mc.tags,
	}
	
	return nil
}

func (mc *MemoryCache) Forever(key string, value interface{}) error {
	return mc.Put(key, value, 0)
}

func (mc *MemoryCache) Forget(key string) error {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	
	delete(mc.items, key)
	return nil
}

func (mc *MemoryCache) Flush() error {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	
	mc.items = make(map[string]*CacheItem)
	return nil
}

func (mc *MemoryCache) Remember(key string, duration time.Duration, callback func() interface{}) (interface{}, error) {
	if value, err := mc.Get(key); err == nil {
		return value, nil
	}
	
	value := callback()
	mc.Put(key, value, duration)
	return value, nil
}

func (mc *MemoryCache) RememberForever(key string, callback func() interface{}) (interface{}, error) {
	return mc.Remember(key, 0, callback)
}

func (mc *MemoryCache) Has(key string) bool {
	_, err := mc.Get(key)
	return err == nil
}

func (mc *MemoryCache) Missing(key string) bool {
	return !mc.Has(key)
}

func (mc *MemoryCache) Increment(key string, value ...int) (int, error) {
	increment := 1
	if len(value) > 0 {
		increment = value[0]
	}
	
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	
	item, exists := mc.items[key]
	if !exists || item.IsExpired() {
		mc.items[key] = &CacheItem{
			Value:     increment,
			ExpiresAt: time.Time{},
			Tags:      mc.tags,
		}
		return increment, nil
	}
	
	if currentValue, ok := item.Value.(int); ok {
		newValue := currentValue + increment
		item.Value = newValue
		return newValue, nil
	}
	
	return 0, fmt.Errorf("value is not an integer")
}

func (mc *MemoryCache) Decrement(key string, value ...int) (int, error) {
	decrement := 1
	if len(value) > 0 {
		decrement = value[0]
	}
	
	return mc.Increment(key, -decrement)
}

func (mc *MemoryCache) Pull(key string) (interface{}, error) {
	value, err := mc.Get(key)
	if err != nil {
		return nil, err
	}
	
	mc.Forget(key)
	return value, nil
}

func (mc *MemoryCache) Many(keys []string) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	
	for _, key := range keys {
		if value, err := mc.Get(key); err == nil {
			result[key] = value
		}
	}
	
	return result, nil
}

func (mc *MemoryCache) PutMany(items map[string]interface{}, duration time.Duration) error {
	for key, value := range items {
		if err := mc.Put(key, value, duration); err != nil {
			return err
		}
	}
	return nil
}

func (mc *MemoryCache) Tags(tags []string) TaggedCache {
	return &MemoryCache{
		items: mc.items,
		mutex: mc.mutex,
		tags:  tags,
	}
}

func (mc *MemoryCache) FlushTags() error {
	if len(mc.tags) == 0 {
		return nil
	}
	
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	
	for key, item := range mc.items {
		for _, tag := range mc.tags {
			for _, itemTag := range item.Tags {
				if tag == itemTag {
					delete(mc.items, key)
					break
				}
			}
		}
	}
	
	return nil
}

type FileCache struct {
	path string
	*MemoryCache
}

func NewFileCache(path string) *FileCache {
	if err := os.MkdirAll(path, 0755); err != nil {
		panic(fmt.Sprintf("Failed to create cache directory: %v", err))
	}
	
	return &FileCache{
		path:        path,
		MemoryCache: NewMemoryCache(),
	}
}

func (fc *FileCache) Put(key string, value interface{}, duration time.Duration) error {
	if err := fc.MemoryCache.Put(key, value, duration); err != nil {
		return err
	}
	
	return fc.persistToFile(key, value, duration)
}

func (fc *FileCache) Forever(key string, value interface{}) error {
	return fc.Put(key, value, 0)
}

func (fc *FileCache) persistToFile(key string, value interface{}, duration time.Duration) error {
	var expiresAt time.Time
	if duration > 0 {
		expiresAt = time.Now().Add(duration)
	}
	
	item := &CacheItem{
		Value:     value,
		ExpiresAt: expiresAt,
	}
	
	data, err := json.Marshal(item)
	if err != nil {
		return err
	}
	
	filename := filepath.Join(fc.path, fc.keyToFilename(key))
	return os.WriteFile(filename, data, 0644)
}

func (fc *FileCache) Get(key string) (interface{}, error) {
	if value, err := fc.MemoryCache.Get(key); err == nil {
		return value, nil
	}
	
	filename := filepath.Join(fc.path, fc.keyToFilename(key))
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("cache miss")
	}
	
	var item CacheItem
	if err := json.Unmarshal(data, &item); err != nil {
		return nil, err
	}
	
	if item.IsExpired() {
		os.Remove(filename)
		return nil, fmt.Errorf("cache miss")
	}
	
	fc.MemoryCache.Put(key, item.Value, 0)
	return item.Value, nil
}

func (fc *FileCache) Forget(key string) error {
	fc.MemoryCache.Forget(key)
	filename := filepath.Join(fc.path, fc.keyToFilename(key))
	return os.Remove(filename)
}

func (fc *FileCache) Flush() error {
	fc.MemoryCache.Flush()
	return os.RemoveAll(fc.path)
}

func (fc *FileCache) keyToFilename(key string) string {
	return fmt.Sprintf("%x.cache", []byte(key))
}

type CacheManager struct {
	stores map[string]Cache
	default_ string
}

func NewCacheManager() *CacheManager {
	return &CacheManager{
		stores:  make(map[string]Cache),
		default_: "memory",
	}
}

func (cm *CacheManager) Store(name ...string) Cache {
	storeName := cm.default_
	if len(name) > 0 {
		storeName = name[0]
	}
	
	if store, exists := cm.stores[storeName]; exists {
		return store
	}
	
	return cm.createStore(storeName)
}

func (cm *CacheManager) createStore(name string) Cache {
	var store Cache
	
	switch name {
	case "memory":
		store = NewMemoryCache()
	case "file":
		store = NewFileCache("storage/cache")
	default:
		store = NewMemoryCache()
	}
	
	cm.stores[name] = store
	return store
}

func (cm *CacheManager) SetDefaultStore(name string) {
	cm.default_ = name
}

func (cm *CacheManager) RegisterStore(name string, store Cache) {
	cm.stores[name] = store
}

func CacheMiddleware(cache Cache, duration time.Duration) MiddlewareFunc {
	return func(c Context) error {
		key := fmt.Sprintf("route_cache_%s_%s", c.Method(), c.URL())
		
		// Check if we have a cached response
		if cached, err := cache.Get(key); err == nil {
			if response, ok := cached.(string); ok {
				return c.HTML(200, response)
			}
		}
		
		// For now, just pass through - full response caching would need
		// a different approach with the interface-based system
		// TODO: Implement proper response caching with interface compatibility
		// For now, just pass through - response caching will be implemented later
		return c.Next()
	}
}

type responseRecorder struct {
	http.ResponseWriter
	body   *bytes.Buffer
	status int
}

func (rr *responseRecorder) Write(data []byte) (int, error) {
	rr.body.Write(data)
	return rr.ResponseWriter.Write(data)
}

func (rr *responseRecorder) WriteHeader(statusCode int) {
	rr.status = statusCode
	rr.ResponseWriter.WriteHeader(statusCode)
}

// Cache helper function to get cache from application container
func GetCache(app *Application) Cache {
	cache, _ := app.Container().Make("cache")
	if c, ok := cache.(Cache); ok {
		return c
	}
	return NewMemoryCache()
}