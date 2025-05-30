package cache

import (
	"fmt"
	"sync"
)

// repository implements the Repository interface
type repository struct {
	stores      map[string]Store
	caches      map[string]Cache
	defaultName string
	mutex       sync.RWMutex
	metrics     Metrics
}

// NewRepository creates a new cache repository
func NewRepository(metrics Metrics) Repository {
	return &repository{
		stores:      make(map[string]Store),
		caches:      make(map[string]Cache),
		defaultName: "memory",
		metrics:     metrics,
	}
}

// Store returns a cache instance for the specified store
func (r *repository) Store(name ...string) Cache {
	storeName := r.defaultName
	if len(name) > 0 && name[0] != "" {
		storeName = name[0]
	}
	
	r.mutex.RLock()
	if cache, exists := r.caches[storeName]; exists {
		r.mutex.RUnlock()
		return cache
	}
	r.mutex.RUnlock()
	
	// Create cache if it doesn't exist
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	// Double-check after acquiring write lock
	if cache, exists := r.caches[storeName]; exists {
		return cache
	}
	
	store, exists := r.stores[storeName]
	if !exists {
		// Create a default store if not registered
		store = r.createDefaultStore(storeName)
		r.stores[storeName] = store
	}
	
	cache := NewCache(store, r.metrics)
	r.caches[storeName] = cache
	return cache
}

// RegisterStore registers a store with the repository
func (r *repository) RegisterStore(name string, store Store) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	r.stores[name] = store
	// Invalidate cached instance if it exists
	delete(r.caches, name)
}

// SetDefaultStore sets the default store name
func (r *repository) SetDefaultStore(name string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.defaultName = name
}

// GetDefaultStore returns the default store name
func (r *repository) GetDefaultStore() string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.defaultName
}

// CreateStore creates a new store with the given configuration
func (r *repository) CreateStore(name string, config Config) (Store, error) {
	var store Store
	var err error
	
	switch config.Driver {
	case "memory":
		store, err = NewMemoryStore(config.Memory)
	case "file":
		store, err = NewFileStore(config.File)
	default:
		return nil, fmt.Errorf("unsupported cache driver: %s", config.Driver)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to create %s store: %w", config.Driver, err)
	}
	
	r.RegisterStore(name, store)
	return store, nil
}

// Close closes all stores and cleans up resources
func (r *repository) Close() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	var errors []string
	
	// Close all stores that implement closer interface
	for name, store := range r.stores {
		if closer, ok := store.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				errors = append(errors, fmt.Sprintf("store %s: %v", name, err))
			}
		}
	}
	
	// Clear maps
	r.stores = make(map[string]Store)
	r.caches = make(map[string]Cache)
	
	if len(errors) > 0 {
		return fmt.Errorf("errors closing stores: %v", errors)
	}
	
	return nil
}

// createDefaultStore creates a default store for a given name
func (r *repository) createDefaultStore(name string) Store {
	switch name {
	case "file":
		config := FileConfig{
			Path:        "storage/cache",
			Permissions: 0755,
			MaxFileSize: 10 * 1024 * 1024, // 10MB
			Compress:    false,
		}
		store, _ := NewFileStore(config)
		return store
	default:
		// Default to memory cache
		config := MemoryConfig{
			Size:            1000,
			CleanupInterval: 60,
			EvictionPolicy:  "LRU",
		}
		store, _ := NewMemoryStore(config)
		return store
	}
}

// Global repository instance
var globalRepository Repository

// SetupCache initializes the global cache repository
func SetupCache(metrics Metrics) {
	globalRepository = NewRepository(metrics)
}

// GetRepository returns the global cache repository
func GetRepository() Repository {
	if globalRepository == nil {
		// Create default repository if none exists
		globalRepository = NewRepository(NewSimpleMetrics())
	}
	return globalRepository
}

// Global cache functions

// DefaultStore returns a cache instance from the global repository
func DefaultStore(name ...string) Cache {
	return GetRepository().Store(name...)
}

// RegisterStore registers a store with the global repository
func RegisterStore(name string, store Store) {
	GetRepository().RegisterStore(name, store)
}

// SetDefaultStore sets the default store in the global repository
func SetDefaultStore(name string) {
	GetRepository().SetDefaultStore(name)
}

// CreateStore creates a new store with the global repository
func CreateStore(name string, config Config) (Store, error) {
	return GetRepository().CreateStore(name, config)
}