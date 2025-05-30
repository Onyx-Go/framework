package cache

import (
	"context"
	"time"
)

// Cache interface defines the contract for cache operations with context support
type Cache interface {
	// Context-aware cache operations
	GetContext(ctx context.Context, key string) (interface{}, error)
	PutContext(ctx context.Context, key string, value interface{}, duration time.Duration) error
	ForeverContext(ctx context.Context, key string, value interface{}) error
	ForgetContext(ctx context.Context, key string) error
	FlushContext(ctx context.Context) error
	
	// Legacy methods (for backward compatibility)
	Get(key string) (interface{}, error)
	Put(key string, value interface{}, duration time.Duration) error
	Forever(key string, value interface{}) error
	Forget(key string) error
	Flush() error
	
	// Advanced operations
	Remember(key string, duration time.Duration, callback func() interface{}) (interface{}, error)
	RememberForever(key string, callback func() interface{}) (interface{}, error)
	RememberContext(ctx context.Context, key string, duration time.Duration, callback func(ctx context.Context) interface{}) (interface{}, error)
	
	// Utility methods
	Has(key string) bool
	Missing(key string) bool
	Increment(key string, value ...int) (int, error)
	Decrement(key string, value ...int) (int, error)
	Pull(key string) (interface{}, error)
	
	// Batch operations
	Many(keys []string) (map[string]interface{}, error)
	PutMany(items map[string]interface{}, duration time.Duration) error
	
	// Tagged caching
	Tags(tags []string) TaggedCache
}

// TaggedCache extends Cache with tag-based invalidation
type TaggedCache interface {
	Cache
	FlushTags() error
	FlushTagsContext(ctx context.Context) error
}

// Store interface for cache store implementations
type Store interface {
	// Basic operations
	Get(ctx context.Context, key string) (*Item, error)
	Put(ctx context.Context, key string, item *Item) error
	Delete(ctx context.Context, key string) error
	Clear(ctx context.Context) error
	
	// Utility operations
	Exists(ctx context.Context, key string) bool
	Touch(ctx context.Context, key string, duration time.Duration) error
	
	// Batch operations
	GetMultiple(ctx context.Context, keys []string) (map[string]*Item, error)
	PutMultiple(ctx context.Context, items map[string]*Item) error
	DeleteMultiple(ctx context.Context, keys []string) error
	
	// Increment/Decrement operations (if supported)
	Increment(ctx context.Context, key string, value int) (int, error)
	Decrement(ctx context.Context, key string, value int) (int, error)
	
	// Store information
	GetInfo() StoreInfo
}

// Repository interface for managing cache stores
type Repository interface {
	// Store management
	Store(name ...string) Cache
	RegisterStore(name string, store Store)
	SetDefaultStore(name string)
	GetDefaultStore() string
	
	// Store creation
	CreateStore(name string, config Config) (Store, error)
	
	// Lifecycle
	Close() error
}

// Serializer interface for cache value serialization
type Serializer interface {
	Serialize(value interface{}) ([]byte, error)
	Unserialize(data []byte, target interface{}) error
}

// Item represents a cached item with metadata
type Item struct {
	Key       string                 `json:"key"`
	Value     interface{}            `json:"value"`
	ExpiresAt time.Time              `json:"expires_at"`
	Tags      []string               `json:"tags,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// IsExpired checks if the cache item has expired
func (i *Item) IsExpired() bool {
	return !i.ExpiresAt.IsZero() && time.Now().After(i.ExpiresAt)
}

// TTL returns the time to live for the cache item
func (i *Item) TTL() time.Duration {
	if i.ExpiresAt.IsZero() {
		return 0 // No expiration
	}
	
	remaining := time.Until(i.ExpiresAt)
	if remaining < 0 {
		return 0 // Expired
	}
	
	return remaining
}

// StoreInfo provides information about a cache store
type StoreInfo struct {
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	Size         int64                  `json:"size"`
	Count        int64                  `json:"count"`
	HitRate      float64                `json:"hit_rate"`
	MissRate     float64                `json:"miss_rate"`
	Capabilities []string               `json:"capabilities"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// Config represents cache configuration
type Config struct {
	Driver     string                 `json:"driver"`
	Prefix     string                 `json:"prefix"`
	TTL        time.Duration          `json:"ttl"`
	Serializer string                 `json:"serializer"`
	Options    map[string]interface{} `json:"options"`
	
	// Store-specific configuration
	Memory MemoryConfig `json:"memory"`
	File   FileConfig   `json:"file"`
	Redis  RedisConfig  `json:"redis"`
}

// MemoryConfig represents memory cache configuration
type MemoryConfig struct {
	Size            int           `json:"size"`              // Maximum number of items
	CleanupInterval time.Duration `json:"cleanup_interval"`  // Cleanup interval for expired items
	EvictionPolicy  string        `json:"eviction_policy"`   // LRU, LFU, FIFO
}

// FileConfig represents file cache configuration
type FileConfig struct {
	Path        string        `json:"path"`
	Permissions int           `json:"permissions"`
	MaxFileSize int64         `json:"max_file_size"`
	Compress    bool          `json:"compress"`
	Sync        time.Duration `json:"sync"` // Sync interval for writing to disk
}

// RedisConfig represents Redis cache configuration
type RedisConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Password string `json:"password"`
	Database int    `json:"database"`
	PoolSize int    `json:"pool_size"`
}

// Callback functions for cache operations
type RememberCallback func() interface{}
type RememberContextCallback func(ctx context.Context) interface{}

// Lock interface for distributed locking
type Lock interface {
	Acquire(ctx context.Context, key string, ttl time.Duration) (bool, error)
	Release(ctx context.Context, key string) error
	Extend(ctx context.Context, key string, ttl time.Duration) error
}

// Event types for cache events
type EventType string

const (
	EventCacheHit    EventType = "cache_hit"
	EventCacheMiss   EventType = "cache_miss"
	EventCacheWrite  EventType = "cache_write"
	EventCacheDelete EventType = "cache_delete"
	EventCacheFlush  EventType = "cache_flush"
)

// Event represents a cache event
type Event struct {
	Type      EventType              `json:"type"`
	Key       string                 `json:"key"`
	Tags      []string               `json:"tags,omitempty"`
	Store     string                 `json:"store"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// EventListener interface for cache event handling
type EventListener interface {
	Handle(event Event)
}

// Metrics interface for cache metrics collection
type Metrics interface {
	RecordHit(store string)
	RecordMiss(store string)
	RecordWrite(store string, key string, size int64)
	RecordDelete(store string, key string)
	GetStats(store string) StoreInfo
}