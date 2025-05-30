package cache

import (
	"sync"
	"time"
)

// SimpleMetrics provides basic metrics collection for cache operations
type SimpleMetrics struct {
	stores map[string]*StoreMetrics
	mutex  sync.RWMutex
}

// StoreMetrics tracks metrics for a specific store
type StoreMetrics struct {
	Hits      int64     `json:"hits"`
	Misses    int64     `json:"misses"`
	Writes    int64     `json:"writes"`
	Deletes   int64     `json:"deletes"`
	Size      int64     `json:"size"`
	Count     int64     `json:"count"`
	LastHit   time.Time `json:"last_hit"`
	LastMiss  time.Time `json:"last_miss"`
	LastWrite time.Time `json:"last_write"`
}

// NewSimpleMetrics creates a new simple metrics instance
func NewSimpleMetrics() *SimpleMetrics {
	return &SimpleMetrics{
		stores: make(map[string]*StoreMetrics),
	}
}

// RecordHit records a cache hit
func (sm *SimpleMetrics) RecordHit(store string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	metrics := sm.getOrCreateStore(store)
	metrics.Hits++
	metrics.LastHit = time.Now()
}

// RecordMiss records a cache miss
func (sm *SimpleMetrics) RecordMiss(store string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	metrics := sm.getOrCreateStore(store)
	metrics.Misses++
	metrics.LastMiss = time.Now()
}

// RecordWrite records a cache write operation
func (sm *SimpleMetrics) RecordWrite(store string, key string, size int64) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	metrics := sm.getOrCreateStore(store)
	metrics.Writes++
	metrics.Size += size
	metrics.Count++
	metrics.LastWrite = time.Now()
}

// RecordDelete records a cache delete operation
func (sm *SimpleMetrics) RecordDelete(store string, key string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	metrics := sm.getOrCreateStore(store)
	metrics.Deletes++
	if metrics.Count > 0 {
		metrics.Count--
	}
}

// GetStats returns statistics for a store
func (sm *SimpleMetrics) GetStats(store string) StoreInfo {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	metrics := sm.getOrCreateStore(store)
	
	totalRequests := metrics.Hits + metrics.Misses
	hitRate := float64(0)
	if totalRequests > 0 {
		hitRate = float64(metrics.Hits) / float64(totalRequests)
	}

	return StoreInfo{
		Name:     store,
		Type:     "unknown",
		Size:     metrics.Size,
		Count:    metrics.Count,
		HitRate:  hitRate,
		MissRate: 1.0 - hitRate,
		Capabilities: []string{},
		Metadata: map[string]interface{}{
			"total_hits":   metrics.Hits,
			"total_misses": metrics.Misses,
			"total_writes": metrics.Writes,
			"total_deletes": metrics.Deletes,
			"last_hit":     metrics.LastHit,
			"last_miss":    metrics.LastMiss,
			"last_write":   metrics.LastWrite,
		},
	}
}

// getAllStats returns statistics for all stores
func (sm *SimpleMetrics) GetAllStats() map[string]StoreInfo {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	result := make(map[string]StoreInfo)
	for storeName := range sm.stores {
		result[storeName] = sm.GetStats(storeName)
	}
	return result
}

// Reset resets metrics for a store
func (sm *SimpleMetrics) Reset(store string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if _, exists := sm.stores[store]; exists {
		sm.stores[store] = &StoreMetrics{}
	}
}

// ResetAll resets metrics for all stores
func (sm *SimpleMetrics) ResetAll() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	for storeName := range sm.stores {
		sm.stores[storeName] = &StoreMetrics{}
	}
}

// getOrCreateStore gets or creates store metrics
func (sm *SimpleMetrics) getOrCreateStore(store string) *StoreMetrics {
	if metrics, exists := sm.stores[store]; exists {
		return metrics
	}

	metrics := &StoreMetrics{}
	sm.stores[store] = metrics
	return metrics
}

// NoopMetrics provides a no-operation metrics implementation
type NoopMetrics struct{}

// NewNoopMetrics creates a new no-op metrics instance
func NewNoopMetrics() *NoopMetrics {
	return &NoopMetrics{}
}

// RecordHit does nothing
func (nm *NoopMetrics) RecordHit(store string) {}

// RecordMiss does nothing
func (nm *NoopMetrics) RecordMiss(store string) {}

// RecordWrite does nothing
func (nm *NoopMetrics) RecordWrite(store string, key string, size int64) {}

// RecordDelete does nothing
func (nm *NoopMetrics) RecordDelete(store string, key string) {}

// GetStats returns empty stats
func (nm *NoopMetrics) GetStats(store string) StoreInfo {
	return StoreInfo{
		Name:         store,
		Type:         "noop",
		Size:         0,
		Count:        0,
		HitRate:      0,
		MissRate:     0,
		Capabilities: []string{},
		Metadata:     map[string]interface{}{},
	}
}