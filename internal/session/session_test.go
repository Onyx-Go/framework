package session

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestNewSession(t *testing.T) {
	handler := NewMemoryHandler()
	lifetime := 2 * time.Hour
	session := NewSession("test-id", handler, lifetime)
	
	if session.ID() != "test-id" {
		t.Errorf("Expected session ID 'test-id', got %s", session.ID())
	}
	
	if session.GetLifetime() != lifetime {
		t.Errorf("Expected lifetime %v, got %v", lifetime, session.GetLifetime())
	}
	
	if session.IsStarted() {
		t.Error("New session should not be started")
	}
}

func TestSessionBasicOperations(t *testing.T) {
	handler := NewMemoryHandler()
	session := NewSession("test-id", handler, time.Hour)
	
	// Test Put and Get
	session.Put("key1", "value1")
	session.Put("key2", 42)
	
	if value := session.Get("key1"); value != "value1" {
		t.Errorf("Expected 'value1', got %v", value)
	}
	
	if value := session.Get("key2"); value != 42 {
		t.Errorf("Expected 42, got %v", value)
	}
	
	// Test Has
	if !session.Has("key1") {
		t.Error("Expected session to have key1")
	}
	
	if session.Has("nonexistent") {
		t.Error("Expected session not to have nonexistent key")
	}
	
	// Test Remove
	session.Remove("key1")
	if session.Has("key1") {
		t.Error("Expected key1 to be removed")
	}
	
	// Test All
	all := session.All()
	if len(all) != 1 {
		t.Errorf("Expected 1 key in session, got %d", len(all))
	}
	
	if all["key2"] != 42 {
		t.Errorf("Expected key2 value 42, got %v", all["key2"])
	}
}

func TestSessionFlashMessages(t *testing.T) {
	handler := NewMemoryHandler()
	session := NewSession("test-id", handler, time.Hour)
	
	// Test Flash and GetFlash
	session.Flash("message", "Hello World")
	session.Flash("error", "Something went wrong")
	
	// Get all flash messages
	allFlash := session.GetAllFlash()
	if len(allFlash) != 2 {
		t.Errorf("Expected 2 flash messages, got %d", len(allFlash))
	}
	
	// Get and remove flash message
	message := session.GetFlash("message")
	if message != "Hello World" {
		t.Errorf("Expected 'Hello World', got %v", message)
	}
	
	// Message should be removed after getting
	if session.GetFlash("message") != nil {
		t.Error("Flash message should be removed after getting")
	}
	
	// Other flash message should still exist
	if session.GetFlash("error") != "Something went wrong" {
		t.Error("Other flash messages should remain")
	}
	
	// Test ClearFlash
	session.Flash("test", "test")
	session.ClearFlash()
	if len(session.GetAllFlash()) != 0 {
		t.Error("All flash messages should be cleared")
	}
}

func TestSessionLifecycle(t *testing.T) {
	handler := NewMemoryHandler()
	session := NewSession("test-id", handler, time.Hour)
	
	// Test MarkAsStarted
	session.MarkAsStarted()
	if !session.IsStarted() {
		t.Error("Session should be started")
	}
	
	// Test Regenerate
	originalID := session.ID()
	if err := session.Regenerate(); err != nil {
		t.Fatalf("Failed to regenerate session: %v", err)
	}
	
	if session.ID() == originalID {
		t.Error("Session ID should change after regeneration")
	}
	
	// Test Flush
	session.Put("key", "value")
	session.Flash("flash", "message")
	session.Flush()
	
	if len(session.All()) != 0 {
		t.Error("Session data should be empty after flush")
	}
	
	if len(session.GetAllFlash()) != 0 {
		t.Error("Flash data should be empty after flush")
	}
}

func TestSessionExpiration(t *testing.T) {
	handler := NewMemoryHandler()
	
	// Create session with very short lifetime
	session := NewSession("test-id", handler, 1*time.Millisecond)
	
	// Wait for expiration
	time.Sleep(2 * time.Millisecond)
	
	if !session.IsExpired() {
		t.Error("Session should be expired")
	}
}

func TestSessionContextOperations(t *testing.T) {
	handler := NewMemoryHandler()
	session := NewSession("test-id", handler, time.Hour)
	
	ctx := context.Background()
	
	// Test PutWithContext
	err := session.PutWithContext(ctx, "key", "value")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	// Test GetWithContext
	value, err := session.GetWithContext(ctx, "key")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	if value != "value" {
		t.Errorf("Expected 'value', got %v", value)
	}
	
	// Test RemoveWithContext
	err = session.RemoveWithContext(ctx, "key")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	if session.Has("key") {
		t.Error("Key should be removed")
	}
	
	// Test with cancelled context
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	
	err = session.PutWithContext(cancelledCtx, "key", "value")
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

func TestSessionLoadFromData(t *testing.T) {
	handler := NewMemoryHandler()
	session := NewSession("test-id", handler, time.Hour)
	
	now := time.Now()
	lifetime := 2 * time.Hour
	
	data := map[string]interface{}{
		"regular_key":     "regular_value",
		"number_key":      42,
		"_flash.message":  "flash_message",
		"_flash.error":    "flash_error",
		"_created_at":     now,
		"_last_access":    now,
		"_lifetime":       lifetime,
		"_internal_skip":  "should_be_skipped",
	}
	
	session.LoadFromData(data)
	
	// Check regular data
	if session.Get("regular_key") != "regular_value" {
		t.Error("Regular data not loaded correctly")
	}
	
	if session.Get("number_key") != 42 {
		t.Error("Number data not loaded correctly")
	}
	
	// Check flash data
	if session.GetFlash("message") != "flash_message" {
		t.Error("Flash data not loaded correctly")
	}
	
	// Check metadata
	if !session.GetCreatedAt().Equal(now) {
		t.Error("Created at time not loaded correctly")
	}
	
	if session.GetLifetime() != lifetime {
		t.Error("Lifetime not loaded correctly")
	}
	
	// Check that session is marked as started
	if !session.IsStarted() {
		t.Error("Session should be marked as started after loading data")
	}
	
	// Check that internal data is skipped
	if session.Get("_internal_skip") != nil {
		t.Error("Internal data should be skipped")
	}
}

func TestSessionConcurrency(t *testing.T) {
	handler := NewMemoryHandler()
	session := NewSession("test-id", handler, time.Hour)
	
	// Test concurrent access
	done := make(chan bool, 10)
	
	// Start multiple goroutines writing to session
	for i := 0; i < 5; i++ {
		go func(id int) {
			defer func() { done <- true }()
			for j := 0; j < 100; j++ {
				session.Put(fmt.Sprintf("key_%d_%d", id, j), fmt.Sprintf("value_%d_%d", id, j))
			}
		}(i)
	}
	
	// Start multiple goroutines reading from session
	for i := 0; i < 5; i++ {
		go func(id int) {
			defer func() { done <- true }()
			for j := 0; j < 100; j++ {
				session.Get(fmt.Sprintf("key_%d_%d", id, j))
			}
		}(i)
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// Should not panic or race
}

func TestSessionTimestamps(t *testing.T) {
	handler := NewMemoryHandler()
	session := NewSession("test-id", handler, time.Hour)
	
	createdAt := session.GetCreatedAt()
	lastAccess := session.GetLastAccess()
	
	// Should be approximately the same time
	if lastAccess.Sub(createdAt) > time.Second {
		t.Error("Created at and last access should be approximately the same for new session")
	}
	
	// Access session to update last access time
	time.Sleep(1 * time.Millisecond)
	session.Get("nonexistent")
	
	newLastAccess := session.GetLastAccess()
	if !newLastAccess.After(lastAccess) {
		t.Error("Last access time should be updated when accessing session")
	}
}