package cache

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMemoryStore_Basic(t *testing.T) {
	store, err := NewMemoryStore(MemoryConfig{Size: 10})
	if err != nil {
		t.Fatalf("Failed to create memory store: %v", err)
	}
	defer store.(*MemoryStore).Close()

	ctx := context.Background()
	key := "test_key"
	value := "test_value"

	item := &Item{
		Key:       key,
		Value:     value,
		ExpiresAt: time.Now().Add(time.Hour),
	}

	// Test Put
	err = store.Put(ctx, key, item)
	if err != nil {
		t.Fatalf("Failed to put item: %v", err)
	}

	// Test Get
	retrieved, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("Failed to get item: %v", err)
	}

	if retrieved.Value != value {
		t.Errorf("Expected %v, got %v", value, retrieved.Value)
	}

	// Test Exists
	if !store.Exists(ctx, key) {
		t.Error("Key should exist")
	}

	// Test Delete
	err = store.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Failed to delete item: %v", err)
	}

	if store.Exists(ctx, key) {
		t.Error("Key should not exist after delete")
	}
}

func TestMemoryStore_Expiration(t *testing.T) {
	store, _ := NewMemoryStore(MemoryConfig{Size: 10})
	defer store.(*MemoryStore).Close()

	ctx := context.Background()
	key := "expire_key"
	value := "expire_value"

	item := &Item{
		Key:       key,
		Value:     value,
		ExpiresAt: time.Now().Add(50 * time.Millisecond),
	}

	store.Put(ctx, key, item)

	// Should be available immediately
	_, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("Item should be available: %v", err)
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be expired
	_, err = store.Get(ctx, key)
	if err == nil {
		t.Error("Item should have expired")
	}
}

func TestMemoryStore_IncrementDecrement(t *testing.T) {
	store, _ := NewMemoryStore(MemoryConfig{Size: 10})
	defer store.(*MemoryStore).Close()

	ctx := context.Background()
	key := "counter"

	// Test increment (creates new)
	value, err := store.Increment(ctx, key, 1)
	if err != nil {
		t.Fatalf("Failed to increment: %v", err)
	}
	if value != 1 {
		t.Errorf("Expected 1, got %d", value)
	}

	// Test increment existing
	value, err = store.Increment(ctx, key, 5)
	if err != nil {
		t.Fatalf("Failed to increment: %v", err)
	}
	if value != 6 {
		t.Errorf("Expected 6, got %d", value)
	}

	// Test decrement
	value, err = store.Decrement(ctx, key, 2)
	if err != nil {
		t.Fatalf("Failed to decrement: %v", err)
	}
	if value != 4 {
		t.Errorf("Expected 4, got %d", value)
	}
}

func TestMemoryStore_Batch(t *testing.T) {
	store, _ := NewMemoryStore(MemoryConfig{Size: 10})
	defer store.(*MemoryStore).Close()

	ctx := context.Background()

	items := map[string]*Item{
		"key1": {Key: "key1", Value: "value1", ExpiresAt: time.Now().Add(time.Hour)},
		"key2": {Key: "key2", Value: "value2", ExpiresAt: time.Now().Add(time.Hour)},
		"key3": {Key: "key3", Value: "value3", ExpiresAt: time.Now().Add(time.Hour)},
	}

	// Test PutMultiple
	err := store.PutMultiple(ctx, items)
	if err != nil {
		t.Fatalf("Failed to put multiple: %v", err)
	}

	// Test GetMultiple
	keys := []string{"key1", "key2", "key3", "key4"} // key4 doesn't exist
	retrieved, err := store.GetMultiple(ctx, keys)
	if err != nil {
		t.Fatalf("Failed to get multiple: %v", err)
	}

	if len(retrieved) != 3 {
		t.Errorf("Expected 3 items, got %d", len(retrieved))
	}

	for key, item := range items {
		if retrieved[key] == nil {
			t.Errorf("Key %s should exist", key)
			continue
		}
		if retrieved[key].Value != item.Value {
			t.Errorf("Expected %v for key %s, got %v", item.Value, key, retrieved[key].Value)
		}
	}

	// Test DeleteMultiple
	deleteKeys := []string{"key1", "key2"}
	err = store.DeleteMultiple(ctx, deleteKeys)
	if err != nil {
		t.Fatalf("Failed to delete multiple: %v", err)
	}

	if store.Exists(ctx, "key1") || store.Exists(ctx, "key2") {
		t.Error("Keys should have been deleted")
	}

	if !store.Exists(ctx, "key3") {
		t.Error("Key3 should still exist")
	}
}

func TestMemoryStore_Eviction(t *testing.T) {
	store, _ := NewMemoryStore(MemoryConfig{Size: 3, EvictionPolicy: "LRU"})
	defer store.(*MemoryStore).Close()

	ctx := context.Background()

	// Fill the store to capacity
	for i := 1; i <= 3; i++ {
		item := &Item{
			Key:       fmt.Sprintf("key%d", i),
			Value:     fmt.Sprintf("value%d", i),
			ExpiresAt: time.Now().Add(time.Hour),
		}
		store.Put(ctx, item.Key, item)
	}

	// All should exist
	for i := 1; i <= 3; i++ {
		if !store.Exists(ctx, fmt.Sprintf("key%d", i)) {
			t.Errorf("Key%d should exist", i)
		}
	}

	// Add one more - should evict key1 (least recently used)
	item4 := &Item{
		Key:       "key4",
		Value:     "value4",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	store.Put(ctx, "key4", item4)

	// key1 should be evicted
	if store.Exists(ctx, "key1") {
		t.Error("Key1 should have been evicted")
	}

	// Others should still exist
	for i := 2; i <= 4; i++ {
		if !store.Exists(ctx, fmt.Sprintf("key%d", i)) {
			t.Errorf("Key%d should exist", i)
		}
	}
}

func TestFileStore_Basic(t *testing.T) {
	tempDir := t.TempDir()
	config := FileConfig{
		Path:        tempDir,
		Permissions: 0755,
		MaxFileSize: 1024 * 1024,
	}

	store, err := NewFileStore(config)
	if err != nil {
		t.Fatalf("Failed to create file store: %v", err)
	}

	ctx := context.Background()
	key := "test_key"
	value := "test_value"

	item := &Item{
		Key:       key,
		Value:     value,
		ExpiresAt: time.Now().Add(time.Hour),
	}

	// Test Put
	err = store.Put(ctx, key, item)
	if err != nil {
		t.Fatalf("Failed to put item: %v", err)
	}

	// Test Get
	retrieved, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("Failed to get item: %v", err)
	}

	if retrieved.Value != value {
		t.Errorf("Expected %v, got %v", value, retrieved.Value)
	}

	// Test Exists
	if !store.Exists(ctx, key) {
		t.Error("Key should exist")
	}

	// Test Delete
	err = store.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Failed to delete item: %v", err)
	}

	if store.Exists(ctx, key) {
		t.Error("Key should not exist after delete")
	}
}

func TestFileStore_Persistence(t *testing.T) {
	tempDir := t.TempDir()
	config := FileConfig{
		Path:        tempDir,
		Permissions: 0755,
		MaxFileSize: 1024 * 1024,
	}

	store1, err := NewFileStore(config)
	if err != nil {
		t.Fatalf("Failed to create file store: %v", err)
	}

	ctx := context.Background()
	key := "persist_key"
	value := "persist_value"

	item := &Item{
		Key:       key,
		Value:     value,
		ExpiresAt: time.Now().Add(time.Hour),
	}

	// Put with first store
	err = store1.Put(ctx, key, item)
	if err != nil {
		t.Fatalf("Failed to put item: %v", err)
	}

	// Create second store (simulating restart)
	store2, err := NewFileStore(config)
	if err != nil {
		t.Fatalf("Failed to create second file store: %v", err)
	}

	// Should be able to retrieve from second store
	retrieved, err := store2.Get(ctx, key)
	if err != nil {
		t.Fatalf("Failed to get item from second store: %v", err)
	}

	if retrieved.Value != value {
		t.Errorf("Expected %v, got %v", value, retrieved.Value)
	}
}

func TestFileStore_Expiration(t *testing.T) {
	tempDir := t.TempDir()
	config := FileConfig{
		Path:        tempDir,
		Permissions: 0755,
		MaxFileSize: 1024 * 1024,
	}

	store, err := NewFileStore(config)
	if err != nil {
		t.Fatalf("Failed to create file store: %v", err)
	}

	ctx := context.Background()
	key := "expire_key"
	value := "expire_value"

	item := &Item{
		Key:       key,
		Value:     value,
		ExpiresAt: time.Now().Add(50 * time.Millisecond),
	}

	store.Put(ctx, key, item)

	// Should be available immediately
	_, err = store.Get(ctx, key)
	if err != nil {
		t.Fatalf("Item should be available: %v", err)
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be expired and file should be cleaned up
	_, err = store.Get(ctx, key)
	if err == nil {
		t.Error("Item should have expired")
	}

	// File should be removed
	filename := filepath.Join(tempDir, fmt.Sprintf("%x.cache", []byte(key)))
	if _, err := os.Stat(filename); !os.IsNotExist(err) {
		t.Error("Expired file should have been removed")
	}
}

func TestFileStore_Clear(t *testing.T) {
	tempDir := t.TempDir()
	config := FileConfig{
		Path:        tempDir,
		Permissions: 0755,
		MaxFileSize: 1024 * 1024,
	}

	store, err := NewFileStore(config)
	if err != nil {
		t.Fatalf("Failed to create file store: %v", err)
	}

	ctx := context.Background()

	// Put multiple items
	for i := 1; i <= 3; i++ {
		item := &Item{
			Key:       fmt.Sprintf("key%d", i),
			Value:     fmt.Sprintf("value%d", i),
			ExpiresAt: time.Now().Add(time.Hour),
		}
		store.Put(ctx, item.Key, item)
	}

	// Verify files exist
	for i := 1; i <= 3; i++ {
		if !store.Exists(ctx, fmt.Sprintf("key%d", i)) {
			t.Errorf("Key%d should exist", i)
		}
	}

	// Clear all
	err = store.Clear(ctx)
	if err != nil {
		t.Fatalf("Failed to clear store: %v", err)
	}

	// All should be gone
	for i := 1; i <= 3; i++ {
		if store.Exists(ctx, fmt.Sprintf("key%d", i)) {
			t.Errorf("Key%d should not exist after clear", i)
		}
	}
}

func TestJSONSerializer(t *testing.T) {
	serializer := NewJSONSerializer()

	original := &Item{
		Key:       "test_key",
		Value:     "test_value",
		ExpiresAt: time.Now().Add(time.Hour),
		Tags:      []string{"tag1", "tag2"},
		Metadata:  map[string]interface{}{"foo": "bar"},
	}

	// Test Serialize
	data, err := serializer.Serialize(original)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	// Test Unserialize
	var restored Item
	err = serializer.Unserialize(data, &restored)
	if err != nil {
		t.Fatalf("Failed to unserialize: %v", err)
	}

	// Compare
	if restored.Key != original.Key {
		t.Errorf("Key mismatch: expected %v, got %v", original.Key, restored.Key)
	}

	if restored.Value != original.Value {
		t.Errorf("Value mismatch: expected %v, got %v", original.Value, restored.Value)
	}

	if len(restored.Tags) != len(original.Tags) {
		t.Errorf("Tags length mismatch: expected %d, got %d", len(original.Tags), len(restored.Tags))
	}

	if restored.Metadata["foo"] != original.Metadata["foo"] {
		t.Errorf("Metadata mismatch: expected %v, got %v", original.Metadata["foo"], restored.Metadata["foo"])
	}
}