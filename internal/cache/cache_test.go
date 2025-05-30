package cache

import (
	"context"
	"testing"
	"time"
)

func TestCache_GetSet(t *testing.T) {
	store, _ := NewMemoryStore(MemoryConfig{Size: 100})
	metrics := NewSimpleMetrics()
	cache := NewCache(store, metrics)

	ctx := context.Background()
	key := "test_key"
	value := "test_value"

	// Test Put and Get
	err := cache.PutContext(ctx, key, value, time.Hour)
	if err != nil {
		t.Fatalf("Failed to put value: %v", err)
	}

	retrieved, err := cache.GetContext(ctx, key)
	if err != nil {
		t.Fatalf("Failed to get value: %v", err)
	}

	if retrieved != value {
		t.Errorf("Expected %v, got %v", value, retrieved)
	}
}

func TestCache_Expiration(t *testing.T) {
	store, _ := NewMemoryStore(MemoryConfig{Size: 100})
	metrics := NewSimpleMetrics()
	cache := NewCache(store, metrics)

	ctx := context.Background()
	key := "expire_key"
	value := "expire_value"

	// Put with short expiration
	err := cache.PutContext(ctx, key, value, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to put value: %v", err)
	}

	// Should be available immediately
	_, err = cache.GetContext(ctx, key)
	if err != nil {
		t.Fatalf("Value should be available: %v", err)
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be expired now
	_, err = cache.GetContext(ctx, key)
	if err == nil {
		t.Error("Value should have expired")
	}
}

func TestCache_Forever(t *testing.T) {
	store, _ := NewMemoryStore(MemoryConfig{Size: 100})
	metrics := NewSimpleMetrics()
	cache := NewCache(store, metrics)

	ctx := context.Background()
	key := "forever_key"
	value := "forever_value"

	err := cache.ForeverContext(ctx, key, value)
	if err != nil {
		t.Fatalf("Failed to put forever value: %v", err)
	}

	// Should be available
	retrieved, err := cache.GetContext(ctx, key)
	if err != nil {
		t.Fatalf("Failed to get forever value: %v", err)
	}

	if retrieved != value {
		t.Errorf("Expected %v, got %v", value, retrieved)
	}
}

func TestCache_Forget(t *testing.T) {
	store, _ := NewMemoryStore(MemoryConfig{Size: 100})
	metrics := NewSimpleMetrics()
	cache := NewCache(store, metrics)

	ctx := context.Background()
	key := "forget_key"
	value := "forget_value"

	// Put value
	cache.PutContext(ctx, key, value, time.Hour)

	// Verify it exists
	_, err := cache.GetContext(ctx, key)
	if err != nil {
		t.Fatalf("Value should exist: %v", err)
	}

	// Forget it
	err = cache.ForgetContext(ctx, key)
	if err != nil {
		t.Fatalf("Failed to forget value: %v", err)
	}

	// Should not exist anymore
	_, err = cache.GetContext(ctx, key)
	if err == nil {
		t.Error("Value should have been forgotten")
	}
}

func TestCache_Flush(t *testing.T) {
	store, _ := NewMemoryStore(MemoryConfig{Size: 100})
	metrics := NewSimpleMetrics()
	cache := NewCache(store, metrics)

	ctx := context.Background()

	// Put multiple values
	cache.PutContext(ctx, "key1", "value1", time.Hour)
	cache.PutContext(ctx, "key2", "value2", time.Hour)
	cache.PutContext(ctx, "key3", "value3", time.Hour)

	// Verify they exist
	if !cache.Has("key1") || !cache.Has("key2") || !cache.Has("key3") {
		t.Error("Values should exist before flush")
	}

	// Flush all
	err := cache.FlushContext(ctx)
	if err != nil {
		t.Fatalf("Failed to flush cache: %v", err)
	}

	// Should not exist anymore
	if cache.Has("key1") || cache.Has("key2") || cache.Has("key3") {
		t.Error("Values should not exist after flush")
	}
}

func TestCache_Remember(t *testing.T) {
	store, _ := NewMemoryStore(MemoryConfig{Size: 100})
	metrics := NewSimpleMetrics()
	cache := NewCache(store, metrics)

	ctx := context.Background()
	key := "remember_key"
	expectedValue := "computed_value"
	callCount := 0

	callback := func(ctx context.Context) interface{} {
		callCount++
		return expectedValue
	}

	// First call should execute callback
	value, err := cache.RememberContext(ctx, key, time.Hour, callback)
	if err != nil {
		t.Fatalf("Failed to remember value: %v", err)
	}

	if value != expectedValue {
		t.Errorf("Expected %v, got %v", expectedValue, value)
	}

	if callCount != 1 {
		t.Errorf("Expected callback to be called once, called %d times", callCount)
	}

	// Second call should use cached value
	value, err = cache.RememberContext(ctx, key, time.Hour, callback)
	if err != nil {
		t.Fatalf("Failed to remember cached value: %v", err)
	}

	if value != expectedValue {
		t.Errorf("Expected %v, got %v", expectedValue, value)
	}

	if callCount != 1 {
		t.Errorf("Expected callback to be called once, called %d times", callCount)
	}
}

func TestCache_IncrementDecrement(t *testing.T) {
	store, _ := NewMemoryStore(MemoryConfig{Size: 100})
	metrics := NewSimpleMetrics()
	cache := NewCache(store, metrics)

	key := "counter_key"

	// Test increment
	value, err := cache.Increment(key)
	if err != nil {
		t.Fatalf("Failed to increment: %v", err)
	}
	if value != 1 {
		t.Errorf("Expected 1, got %d", value)
	}

	// Test increment with value
	value, err = cache.Increment(key, 5)
	if err != nil {
		t.Fatalf("Failed to increment by 5: %v", err)
	}
	if value != 6 {
		t.Errorf("Expected 6, got %d", value)
	}

	// Test decrement
	value, err = cache.Decrement(key)
	if err != nil {
		t.Fatalf("Failed to decrement: %v", err)
	}
	if value != 5 {
		t.Errorf("Expected 5, got %d", value)
	}

	// Test decrement with value
	value, err = cache.Decrement(key, 3)
	if err != nil {
		t.Fatalf("Failed to decrement by 3: %v", err)
	}
	if value != 2 {
		t.Errorf("Expected 2, got %d", value)
	}
}

func TestCache_Pull(t *testing.T) {
	store, _ := NewMemoryStore(MemoryConfig{Size: 100})
	metrics := NewSimpleMetrics()
	cache := NewCache(store, metrics)

	key := "pull_key"
	value := "pull_value"

	// Put value
	cache.Put(key, value, time.Hour)

	// Verify it exists
	if !cache.Has(key) {
		t.Error("Value should exist before pull")
	}

	// Pull value
	pulled, err := cache.Pull(key)
	if err != nil {
		t.Fatalf("Failed to pull value: %v", err)
	}

	if pulled != value {
		t.Errorf("Expected %v, got %v", value, pulled)
	}

	// Should not exist anymore
	if cache.Has(key) {
		t.Error("Value should not exist after pull")
	}
}

func TestCache_Many(t *testing.T) {
	store, _ := NewMemoryStore(MemoryConfig{Size: 100})
	metrics := NewSimpleMetrics()
	cache := NewCache(store, metrics)

	// Put multiple values
	cache.Put("key1", "value1", time.Hour)
	cache.Put("key2", "value2", time.Hour)
	cache.Put("key3", "value3", time.Hour)

	// Get many
	keys := []string{"key1", "key2", "key3", "key4"} // key4 doesn't exist
	values, err := cache.Many(keys)
	if err != nil {
		t.Fatalf("Failed to get many: %v", err)
	}

	expected := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	if len(values) != 3 {
		t.Errorf("Expected 3 values, got %d", len(values))
	}

	for key, expectedValue := range expected {
		if values[key] != expectedValue {
			t.Errorf("Expected %v for key %s, got %v", expectedValue, key, values[key])
		}
	}
}

func TestCache_PutMany(t *testing.T) {
	store, _ := NewMemoryStore(MemoryConfig{Size: 100})
	metrics := NewSimpleMetrics()
	cache := NewCache(store, metrics)

	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	// Put many
	err := cache.PutMany(items, time.Hour)
	if err != nil {
		t.Fatalf("Failed to put many: %v", err)
	}

	// Verify all exist
	for key, expectedValue := range items {
		value, err := cache.Get(key)
		if err != nil {
			t.Errorf("Failed to get key %s: %v", key, err)
			continue
		}
		if value != expectedValue {
			t.Errorf("Expected %v for key %s, got %v", expectedValue, key, value)
		}
	}
}

func TestCache_Tags(t *testing.T) {
	store, _ := NewMemoryStore(MemoryConfig{Size: 100})
	metrics := NewSimpleMetrics()
	cache := NewCache(store, metrics)

	tags := []string{"tag1", "tag2"}
	taggedCache := cache.Tags(tags)

	// Put value with tags
	err := taggedCache.Put("tagged_key", "tagged_value", time.Hour)
	if err != nil {
		t.Fatalf("Failed to put tagged value: %v", err)
	}

	// Should be retrievable
	value, err := taggedCache.Get("tagged_key")
	if err != nil {
		t.Fatalf("Failed to get tagged value: %v", err)
	}

	if value != "tagged_value" {
		t.Errorf("Expected 'tagged_value', got %v", value)
	}

	// Should also be retrievable from main cache
	value, err = cache.Get("tagged_key")
	if err != nil {
		t.Fatalf("Failed to get value from main cache: %v", err)
	}

	if value != "tagged_value" {
		t.Errorf("Expected 'tagged_value', got %v", value)
	}
}

func TestCache_LegacyMethods(t *testing.T) {
	store, _ := NewMemoryStore(MemoryConfig{Size: 100})
	metrics := NewSimpleMetrics()
	cache := NewCache(store, metrics)

	// Test legacy methods delegate to context methods
	key := "legacy_key"
	value := "legacy_value"

	err := cache.Put(key, value, time.Hour)
	if err != nil {
		t.Fatalf("Failed to put with legacy method: %v", err)
	}

	retrieved, err := cache.Get(key)
	if err != nil {
		t.Fatalf("Failed to get with legacy method: %v", err)
	}

	if retrieved != value {
		t.Errorf("Expected %v, got %v", value, retrieved)
	}

	err = cache.Forget(key)
	if err != nil {
		t.Fatalf("Failed to forget with legacy method: %v", err)
	}

	_, err = cache.Get(key)
	if err == nil {
		t.Error("Value should have been forgotten")
	}
}