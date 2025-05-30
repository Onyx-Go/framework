package session

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestMemoryHandler_BasicOperations(t *testing.T) {
	handler := NewMemoryHandler()
	ctx := context.Background()
	
	// Test Open
	if err := handler.Open(ctx); err != nil {
		t.Fatalf("Failed to open handler: %v", err)
	}
	
	// Test Write and Read
	sessionID := "test-session-id"
	testData := []byte("test session data")
	
	if err := handler.Write(ctx, sessionID, testData); err != nil {
		t.Fatalf("Failed to write session data: %v", err)
	}
	
	readData, err := handler.Read(ctx, sessionID)
	if err != nil {
		t.Fatalf("Failed to read session data: %v", err)
	}
	
	if string(readData) != string(testData) {
		t.Errorf("Expected %s, got %s", string(testData), string(readData))
	}
	
	// Test Exists
	exists, err := handler.Exists(ctx, sessionID)
	if err != nil {
		t.Fatalf("Failed to check if session exists: %v", err)
	}
	
	if !exists {
		t.Error("Session should exist")
	}
	
	// Test non-existent session
	exists, err = handler.Exists(ctx, "non-existent")
	if err != nil {
		t.Fatalf("Failed to check if session exists: %v", err)
	}
	
	if exists {
		t.Error("Non-existent session should not exist")
	}
	
	// Test Count
	count, err := handler.Count(ctx)
	if err != nil {
		t.Fatalf("Failed to get session count: %v", err)
	}
	
	if count != 1 {
		t.Errorf("Expected 1 session, got %d", count)
	}
	
	// Test Destroy
	if err := handler.Destroy(ctx, sessionID); err != nil {
		t.Fatalf("Failed to destroy session: %v", err)
	}
	
	exists, _ = handler.Exists(ctx, sessionID)
	if exists {
		t.Error("Session should not exist after destroy")
	}
	
	// Test Close
	if err := handler.Close(ctx); err != nil {
		t.Fatalf("Failed to close handler: %v", err)
	}
}

func TestMemoryHandler_MultipleOperations(t *testing.T) {
	handler := NewMemoryHandler()
	ctx := context.Background()
	
	handler.Open(ctx)
	defer handler.Close(ctx)
	
	// Test ReadMultiple and WriteMultiple
	sessions := map[string][]byte{
		"session1": []byte("data1"),
		"session2": []byte("data2"),
		"session3": []byte("data3"),
	}
	
	// Write multiple sessions
	if err := handler.WriteMultiple(ctx, sessions); err != nil {
		t.Fatalf("Failed to write multiple sessions: %v", err)
	}
	
	// Read multiple sessions
	sessionIDs := []string{"session1", "session2", "session3", "nonexistent"}
	readSessions, err := handler.ReadMultiple(ctx, sessionIDs)
	if err != nil {
		t.Fatalf("Failed to read multiple sessions: %v", err)
	}
	
	// Verify read data
	for sessionID, expectedData := range sessions {
		if readData, exists := readSessions[sessionID]; exists {
			if string(readData) != string(expectedData) {
				t.Errorf("Session %s: expected %s, got %s", sessionID, string(expectedData), string(readData))
			}
		} else {
			t.Errorf("Session %s should exist in read result", sessionID)
		}
	}
	
	// Verify non-existent session is not in result
	if _, exists := readSessions["nonexistent"]; exists {
		t.Error("Non-existent session should not be in read result")
	}
	
	// Test DestroyMultiple
	destroyIDs := []string{"session1", "session3"}
	if err := handler.DestroyMultiple(ctx, destroyIDs); err != nil {
		t.Fatalf("Failed to destroy multiple sessions: %v", err)
	}
	
	// Verify sessions are destroyed
	for _, sessionID := range destroyIDs {
		if exists, _ := handler.Exists(ctx, sessionID); exists {
			t.Errorf("Session %s should be destroyed", sessionID)
		}
	}
	
	// Verify session2 still exists
	if exists, _ := handler.Exists(ctx, "session2"); !exists {
		t.Error("Session2 should still exist")
	}
}

func TestMemoryHandler_GarbageCollection(t *testing.T) {
	handler := NewMemoryHandler()
	ctx := context.Background()
	
	handler.Open(ctx)
	defer handler.Close(ctx)
	
	// Create some sessions
	handler.Write(ctx, "session1", []byte("data1"))
	handler.Write(ctx, "session2", []byte("data2"))
	handler.Write(ctx, "session3", []byte("data3"))
	
	// Perform garbage collection with very short lifetime (all should be removed)
	if err := handler.GC(ctx, 0); err != nil {
		t.Fatalf("Failed to perform garbage collection: %v", err)
	}
	
	// Check that all sessions are removed
	count, _ := handler.Count(ctx)
	if count != 0 {
		t.Errorf("Expected 0 sessions after GC, got %d", count)
	}
}

func TestMemoryHandler_Statistics(t *testing.T) {
	handler := NewMemoryHandler()
	ctx := context.Background()
	
	handler.Open(ctx)
	defer handler.Close(ctx)
	
	// Create some sessions
	handler.Write(ctx, "session1", []byte("data1"))
	time.Sleep(1 * time.Millisecond) // Ensure different creation times
	handler.Write(ctx, "session2", []byte("data2"))
	
	stats, err := handler.GetStatistics(ctx)
	if err != nil {
		t.Fatalf("Failed to get statistics: %v", err)
	}
	
	if stats.ActiveSessions != 2 {
		t.Errorf("Expected 2 active sessions, got %d", stats.ActiveSessions)
	}
	
	if stats.TotalSessions != 2 {
		t.Errorf("Expected 2 total sessions, got %d", stats.TotalSessions)
	}
	
	if stats.AverageLifetime == 0 {
		t.Error("Average lifetime should be greater than 0")
	}
}

func TestMemoryHandler_ContextCancellation(t *testing.T) {
	handler := NewMemoryHandler()
	
	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	
	// All operations should return context.Canceled error
	if err := handler.Write(ctx, "session1", []byte("data")); err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
	
	if _, err := handler.Read(ctx, "session1"); err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
	
	if err := handler.Destroy(ctx, "session1"); err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
	
	if _, err := handler.Exists(ctx, "session1"); err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
	
	if err := handler.GC(ctx, 3600); err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
	
	if _, err := handler.Count(ctx); err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

func TestMemoryHandler_Concurrency(t *testing.T) {
	handler := NewMemoryHandler()
	ctx := context.Background()
	
	handler.Open(ctx)
	defer handler.Close(ctx)
	
	// Test concurrent access
	done := make(chan bool, 20)
	
	// Start multiple goroutines writing
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()
			for j := 0; j < 100; j++ {
				sessionID := fmt.Sprintf("session_%d_%d", id, j)
				data := []byte(fmt.Sprintf("data_%d_%d", id, j))
				handler.Write(ctx, sessionID, data)
			}
		}(i)
	}
	
	// Start multiple goroutines reading
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()
			for j := 0; j < 100; j++ {
				sessionID := fmt.Sprintf("session_%d_%d", id, j)
				handler.Read(ctx, sessionID)
				handler.Exists(ctx, sessionID)
			}
		}(i)
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < 20; i++ {
		<-done
	}
	
	// Should not panic or race
}

func TestMemoryHandler_DataIntegrity(t *testing.T) {
	handler := NewMemoryHandler()
	ctx := context.Background()
	
	handler.Open(ctx)
	defer handler.Close(ctx)
	
	sessionID := "test-session"
	originalData := []byte("original data")
	
	// Write data
	handler.Write(ctx, sessionID, originalData)
	
	// Read data
	readData, err := handler.Read(ctx, sessionID)
	if err != nil {
		t.Fatalf("Failed to read data: %v", err)
	}
	
	// Modify the returned data
	readData[0] = 'X'
	
	// Read again to ensure original data is unchanged
	readData2, err := handler.Read(ctx, sessionID)
	if err != nil {
		t.Fatalf("Failed to read data again: %v", err)
	}
	
	if string(readData2) != string(originalData) {
		t.Error("Original data should be unchanged when returned data is modified")
	}
}

func TestMemoryHandler_ClearExpiredSessions(t *testing.T) {
	handler := NewMemoryHandler()
	ctx := context.Background()
	
	handler.Open(ctx)
	defer handler.Close(ctx)
	
	// Create some sessions
	handler.Write(ctx, "session1", []byte("data1"))
	handler.Write(ctx, "session2", []byte("data2"))
	
	// Wait a bit
	time.Sleep(1 * time.Millisecond)
	
	// Clear expired sessions with very short lifetime
	removed, err := handler.ClearExpiredSessions(ctx, 0)
	if err != nil {
		t.Fatalf("Failed to clear expired sessions: %v", err)
	}
	
	if removed != 2 {
		t.Errorf("Expected 2 sessions to be removed, got %d", removed)
	}
	
	// Verify sessions are removed
	count, _ := handler.Count(ctx)
	if count != 0 {
		t.Errorf("Expected 0 sessions after clearing expired, got %d", count)
	}
}