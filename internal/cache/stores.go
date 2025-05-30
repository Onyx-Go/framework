package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// MemoryStore implements an in-memory cache store
type MemoryStore struct {
	items           map[string]*Item
	mutex           sync.RWMutex
	config          MemoryConfig
	stopCleanup     chan struct{}
	cleanupDone     chan struct{}
	stats           storeStats
	evictionPolicy  EvictionPolicy
}

// storeStats tracks store statistics
type storeStats struct {
	hits   int64
	misses int64
	writes int64
	deletes int64
	size   int64
	count  int64
}

// EvictionPolicy interface for different eviction strategies
type EvictionPolicy interface {
	OnAccess(key string)
	OnAdd(key string)
	OnRemove(key string)
	ShouldEvict(currentSize int, maxSize int) (string, bool)
}

// LRUEviction implements Least Recently Used eviction
type LRUEviction struct {
	accessOrder []string
	mutex       sync.Mutex
}

// NewLRUEviction creates a new LRU eviction policy
func NewLRUEviction() *LRUEviction {
	return &LRUEviction{
		accessOrder: make([]string, 0),
	}
}

func (lru *LRUEviction) OnAccess(key string) {
	lru.mutex.Lock()
	defer lru.mutex.Unlock()
	
	// Remove key from current position
	for i, k := range lru.accessOrder {
		if k == key {
			lru.accessOrder = append(lru.accessOrder[:i], lru.accessOrder[i+1:]...)
			break
		}
	}
	// Add to end (most recent)
	lru.accessOrder = append(lru.accessOrder, key)
}

func (lru *LRUEviction) OnAdd(key string) {
	lru.OnAccess(key)
}

func (lru *LRUEviction) OnRemove(key string) {
	lru.mutex.Lock()
	defer lru.mutex.Unlock()
	
	for i, k := range lru.accessOrder {
		if k == key {
			lru.accessOrder = append(lru.accessOrder[:i], lru.accessOrder[i+1:]...)
			break
		}
	}
}

func (lru *LRUEviction) ShouldEvict(currentSize int, maxSize int) (string, bool) {
	lru.mutex.Lock()
	defer lru.mutex.Unlock()
	
	if currentSize >= maxSize && len(lru.accessOrder) > 0 {
		return lru.accessOrder[0], true // Return least recently used
	}
	return "", false
}

// NewMemoryStore creates a new memory store
func NewMemoryStore(config MemoryConfig) (Store, error) {
	if config.Size <= 0 {
		config.Size = 1000
	}
	if config.CleanupInterval <= 0 {
		config.CleanupInterval = 60 * time.Second
	}
	if config.EvictionPolicy == "" {
		config.EvictionPolicy = "LRU"
	}

	var evictionPolicy EvictionPolicy
	switch config.EvictionPolicy {
	case "LRU":
		evictionPolicy = NewLRUEviction()
	default:
		evictionPolicy = NewLRUEviction()
	}

	store := &MemoryStore{
		items:          make(map[string]*Item),
		config:         config,
		stopCleanup:    make(chan struct{}),
		cleanupDone:    make(chan struct{}),
		evictionPolicy: evictionPolicy,
	}

	// Start cleanup goroutine
	go store.cleanup()

	return store, nil
}

// Get retrieves an item from memory
func (ms *MemoryStore) Get(ctx context.Context, key string) (*Item, error) {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()

	item, exists := ms.items[key]
	if !exists {
		ms.stats.misses++
		return nil, fmt.Errorf("key not found")
	}

	if item.IsExpired() {
		ms.stats.misses++
		// Remove expired item (will be done in cleanup, but we can do it now)
		delete(ms.items, key)
		ms.evictionPolicy.OnRemove(key)
		ms.stats.count--
		return nil, fmt.Errorf("key expired")
	}

	ms.stats.hits++
	ms.evictionPolicy.OnAccess(key)
	return item, nil
}

// Put stores an item in memory
func (ms *MemoryStore) Put(ctx context.Context, key string, item *Item) error {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	// Check if we need to evict items
	if len(ms.items) >= ms.config.Size {
		if evictKey, shouldEvict := ms.evictionPolicy.ShouldEvict(len(ms.items), ms.config.Size); shouldEvict {
			delete(ms.items, evictKey)
			ms.evictionPolicy.OnRemove(evictKey)
			ms.stats.count--
		}
	}

	// Add or update item
	if _, exists := ms.items[key]; !exists {
		ms.stats.count++
	}
	
	ms.items[key] = item
	ms.evictionPolicy.OnAdd(key)
	ms.stats.writes++

	return nil
}

// Delete removes an item from memory
func (ms *MemoryStore) Delete(ctx context.Context, key string) error {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	if _, exists := ms.items[key]; exists {
		delete(ms.items, key)
		ms.evictionPolicy.OnRemove(key)
		ms.stats.deletes++
		ms.stats.count--
	}

	return nil
}

// Clear removes all items from memory
func (ms *MemoryStore) Clear(ctx context.Context) error {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	ms.items = make(map[string]*Item)
	ms.stats.count = 0
	// Reset eviction policy
	switch ms.config.EvictionPolicy {
	case "LRU":
		ms.evictionPolicy = NewLRUEviction()
	default:
		ms.evictionPolicy = NewLRUEviction()
	}

	return nil
}

// Exists checks if a key exists in memory
func (ms *MemoryStore) Exists(ctx context.Context, key string) bool {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()

	item, exists := ms.items[key]
	if !exists {
		return false
	}

	return !item.IsExpired()
}

// Touch updates the expiration time of an item
func (ms *MemoryStore) Touch(ctx context.Context, key string, duration time.Duration) error {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	item, exists := ms.items[key]
	if !exists {
		return fmt.Errorf("key not found")
	}

	if duration > 0 {
		item.ExpiresAt = time.Now().Add(duration)
	} else {
		item.ExpiresAt = time.Time{}
	}

	ms.evictionPolicy.OnAccess(key)
	return nil
}

// GetMultiple retrieves multiple items from memory
func (ms *MemoryStore) GetMultiple(ctx context.Context, keys []string) (map[string]*Item, error) {
	result := make(map[string]*Item)

	for _, key := range keys {
		if item, err := ms.Get(ctx, key); err == nil {
			result[key] = item
		}
	}

	return result, nil
}

// PutMultiple stores multiple items in memory
func (ms *MemoryStore) PutMultiple(ctx context.Context, items map[string]*Item) error {
	for key, item := range items {
		if err := ms.Put(ctx, key, item); err != nil {
			return err
		}
	}
	return nil
}

// DeleteMultiple removes multiple items from memory
func (ms *MemoryStore) DeleteMultiple(ctx context.Context, keys []string) error {
	for _, key := range keys {
		ms.Delete(ctx, key)
	}
	return nil
}

// Increment increments a numeric value
func (ms *MemoryStore) Increment(ctx context.Context, key string, value int) (int, error) {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	item, exists := ms.items[key]
	if !exists || item.IsExpired() {
		// Create new item with initial value
		newItem := &Item{
			Key:       key,
			Value:     value,
			ExpiresAt: time.Time{},
		}
		ms.items[key] = newItem
		if !exists {
			ms.stats.count++
		}
		ms.evictionPolicy.OnAdd(key)
		ms.stats.writes++
		return value, nil
	}

	// Increment existing value
	if currentValue, ok := item.Value.(int); ok {
		newValue := currentValue + value
		item.Value = newValue
		ms.evictionPolicy.OnAccess(key)
		ms.stats.writes++
		return newValue, nil
	}

	return 0, fmt.Errorf("value is not an integer")
}

// Decrement decrements a numeric value
func (ms *MemoryStore) Decrement(ctx context.Context, key string, value int) (int, error) {
	return ms.Increment(ctx, key, -value)
}

// GetInfo returns store information
func (ms *MemoryStore) GetInfo() StoreInfo {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()

	totalHits := ms.stats.hits + ms.stats.misses
	hitRate := float64(0)
	if totalHits > 0 {
		hitRate = float64(ms.stats.hits) / float64(totalHits)
	}

	return StoreInfo{
		Name:     "memory",
		Type:     "memory",
		Size:     ms.stats.size,
		Count:    ms.stats.count,
		HitRate:  hitRate,
		MissRate: 1.0 - hitRate,
		Capabilities: []string{
			"increment",
			"decrement",
			"batch",
			"eviction",
		},
		Metadata: map[string]interface{}{
			"max_size":         ms.config.Size,
			"cleanup_interval": ms.config.CleanupInterval,
			"eviction_policy":  ms.config.EvictionPolicy,
		},
	}
}

// cleanup removes expired items periodically
func (ms *MemoryStore) cleanup() {
	ticker := time.NewTicker(ms.config.CleanupInterval)
	defer ticker.Stop()
	defer close(ms.cleanupDone)

	for {
		select {
		case <-ticker.C:
			ms.mutex.Lock()
			for key, item := range ms.items {
				if item.IsExpired() {
					delete(ms.items, key)
					ms.evictionPolicy.OnRemove(key)
					ms.stats.count--
				}
			}
			ms.mutex.Unlock()
		case <-ms.stopCleanup:
			return
		}
	}
}

// Close closes the memory store
func (ms *MemoryStore) Close() error {
	close(ms.stopCleanup)
	<-ms.cleanupDone
	return nil
}

// FileStore implements a file-based cache store
type FileStore struct {
	config      FileConfig
	mutex       sync.RWMutex
	stats       storeStats
	serializer  Serializer
}

// NewFileStore creates a new file store
func NewFileStore(config FileConfig) (Store, error) {
	if config.Path == "" {
		config.Path = "storage/cache"
	}
	if config.Permissions == 0 {
		config.Permissions = 0755
	}
	if config.MaxFileSize <= 0 {
		config.MaxFileSize = 10 * 1024 * 1024 // 10MB
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(config.Path, os.FileMode(config.Permissions)); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %v", err)
	}

	store := &FileStore{
		config:     config,
		serializer: NewJSONSerializer(),
	}

	return store, nil
}

// Get retrieves an item from file
func (fs *FileStore) Get(ctx context.Context, key string) (*Item, error) {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	filename := fs.keyToFilename(key)
	data, err := os.ReadFile(filename)
	if err != nil {
		fs.stats.misses++
		return nil, fmt.Errorf("key not found")
	}

	var item Item
	if err := fs.serializer.Unserialize(data, &item); err != nil {
		fs.stats.misses++
		return nil, fmt.Errorf("failed to deserialize item: %v", err)
	}

	if item.IsExpired() {
		fs.stats.misses++
		os.Remove(filename) // Clean up expired file
		return nil, fmt.Errorf("key expired")
	}

	fs.stats.hits++
	return &item, nil
}

// Put stores an item to file
func (fs *FileStore) Put(ctx context.Context, key string, item *Item) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	data, err := fs.serializer.Serialize(item)
	if err != nil {
		return fmt.Errorf("failed to serialize item: %v", err)
	}

	filename := fs.keyToFilename(key)
	
	// Check file size limits
	if fs.config.MaxFileSize > 0 && int64(len(data)) > fs.config.MaxFileSize {
		return fmt.Errorf("item size exceeds maximum file size")
	}

	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write cache file: %v", err)
	}

	fs.stats.writes++
	fs.stats.size += int64(len(data))
	return nil
}

// Delete removes an item file
func (fs *FileStore) Delete(ctx context.Context, key string) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	filename := fs.keyToFilename(key)
	if err := os.Remove(filename); err != nil && !os.IsNotExist(err) {
		return err
	}

	fs.stats.deletes++
	return nil
}

// Clear removes all cache files
func (fs *FileStore) Clear(ctx context.Context) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	pattern := filepath.Join(fs.config.Path, "*.cache")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}

	for _, filename := range matches {
		os.Remove(filename)
	}

	fs.stats.count = 0
	fs.stats.size = 0
	return nil
}

// Exists checks if a key file exists
func (fs *FileStore) Exists(ctx context.Context, key string) bool {
	filename := fs.keyToFilename(key)
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return false
	}

	// Check if expired
	if item, err := fs.Get(ctx, key); err != nil || item.IsExpired() {
		return false
	}

	return true
}

// Touch updates the expiration time of an item
func (fs *FileStore) Touch(ctx context.Context, key string, duration time.Duration) error {
	item, err := fs.Get(ctx, key)
	if err != nil {
		return err
	}

	if duration > 0 {
		item.ExpiresAt = time.Now().Add(duration)
	} else {
		item.ExpiresAt = time.Time{}
	}

	return fs.Put(ctx, key, item)
}

// GetMultiple retrieves multiple items from files
func (fs *FileStore) GetMultiple(ctx context.Context, keys []string) (map[string]*Item, error) {
	result := make(map[string]*Item)

	for _, key := range keys {
		if item, err := fs.Get(ctx, key); err == nil {
			result[key] = item
		}
	}

	return result, nil
}

// PutMultiple stores multiple items to files
func (fs *FileStore) PutMultiple(ctx context.Context, items map[string]*Item) error {
	for key, item := range items {
		if err := fs.Put(ctx, key, item); err != nil {
			return err
		}
	}
	return nil
}

// DeleteMultiple removes multiple item files
func (fs *FileStore) DeleteMultiple(ctx context.Context, keys []string) error {
	for _, key := range keys {
		fs.Delete(ctx, key)
	}
	return nil
}

// Increment increments a numeric value (not atomic for file store)
func (fs *FileStore) Increment(ctx context.Context, key string, value int) (int, error) {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	item, err := fs.Get(ctx, key)
	if err != nil {
		// Create new item with initial value
		newItem := &Item{
			Key:       key,
			Value:     value,
			ExpiresAt: time.Time{},
		}
		if err := fs.Put(ctx, key, newItem); err != nil {
			return 0, err
		}
		return value, nil
	}

	// Increment existing value
	if currentValue, ok := item.Value.(int); ok {
		newValue := currentValue + value
		item.Value = newValue
		if err := fs.Put(ctx, key, item); err != nil {
			return 0, err
		}
		return newValue, nil
	}

	return 0, fmt.Errorf("value is not an integer")
}

// Decrement decrements a numeric value
func (fs *FileStore) Decrement(ctx context.Context, key string, value int) (int, error) {
	return fs.Increment(ctx, key, -value)
}

// GetInfo returns store information
func (fs *FileStore) GetInfo() StoreInfo {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	totalHits := fs.stats.hits + fs.stats.misses
	hitRate := float64(0)
	if totalHits > 0 {
		hitRate = float64(fs.stats.hits) / float64(totalHits)
	}

	return StoreInfo{
		Name:     "file",
		Type:     "file",
		Size:     fs.stats.size,
		Count:    fs.stats.count,
		HitRate:  hitRate,
		MissRate: 1.0 - hitRate,
		Capabilities: []string{
			"increment",
			"decrement",
			"batch",
			"persistent",
		},
		Metadata: map[string]interface{}{
			"path":          fs.config.Path,
			"max_file_size": fs.config.MaxFileSize,
			"compress":      fs.config.Compress,
		},
	}
}

// keyToFilename converts a cache key to a filename
func (fs *FileStore) keyToFilename(key string) string {
	return filepath.Join(fs.config.Path, fmt.Sprintf("%x.cache", []byte(key)))
}

// JSONSerializer implements JSON serialization
type JSONSerializer struct{}

// NewJSONSerializer creates a new JSON serializer
func NewJSONSerializer() *JSONSerializer {
	return &JSONSerializer{}
}

// Serialize serializes a value to JSON
func (js *JSONSerializer) Serialize(value interface{}) ([]byte, error) {
	return json.Marshal(value)
}

// Unserialize deserializes JSON data to a target
func (js *JSONSerializer) Unserialize(data []byte, target interface{}) error {
	return json.Unmarshal(data, target)
}