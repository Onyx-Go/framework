package events

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestBaseEvent_Creation(t *testing.T) {
	name := "test.event"
	payload := "test payload"

	event := NewBaseEvent(name, payload)

	if event.GetName() != name {
		t.Errorf("Expected name %s, got %s", name, event.GetName())
	}

	if event.GetPayload() != payload {
		t.Errorf("Expected payload %v, got %v", payload, event.GetPayload())
	}

	if event.GetContext() == nil {
		t.Error("Expected context to be set")
	}

	if event.GetPriority() != PriorityNormal {
		t.Errorf("Expected priority %v, got %v", PriorityNormal, event.GetPriority())
	}
}

func TestBaseEvent_WithContext(t *testing.T) {
	event := NewBaseEvent("test", nil)
	ctx := context.WithValue(context.Background(), "key", "value")

	newEvent := event.WithContext(ctx)

	if newEvent.GetContext() != ctx {
		t.Error("Context not set correctly")
	}

	// Original event should be unchanged
	if event.GetContext() == ctx {
		t.Error("Original event context was modified")
	}
}

func TestBaseEvent_Metadata(t *testing.T) {
	event := NewBaseEvent("test", nil)

	// Test setting metadata
	event.SetMetadata("key1", "value1")
	event.SetMetadata("key2", 42)

	metadata := event.GetMetadata()
	if len(metadata) != 2 {
		t.Errorf("Expected 2 metadata items, got %d", len(metadata))
	}

	if metadata["key1"] != "value1" {
		t.Errorf("Expected value1, got %v", metadata["key1"])
	}

	if metadata["key2"] != 42 {
		t.Errorf("Expected 42, got %v", metadata["key2"])
	}

	// Test getting specific metadata value
	value, exists := event.GetMetadataValue("key1")
	if !exists {
		t.Error("Expected key1 to exist")
	}
	if value != "value1" {
		t.Errorf("Expected value1, got %v", value)
	}

	// Test non-existent key
	_, exists = event.GetMetadataValue("nonexistent")
	if exists {
		t.Error("Expected nonexistent key to not exist")
	}
}

func TestBaseEvent_Priority(t *testing.T) {
	event := NewBaseEvent("test", nil)

	// Test default priority
	if event.GetPriority() != PriorityNormal {
		t.Errorf("Expected default priority %v, got %v", PriorityNormal, event.GetPriority())
	}

	// Test setting priority
	event.SetPriority(PriorityHigh)
	if event.GetPriority() != PriorityHigh {
		t.Errorf("Expected priority %v, got %v", PriorityHigh, event.GetPriority())
	}
}

func TestBaseEvent_Clone(t *testing.T) {
	event := NewBaseEvent("test", "payload")
	event.SetMetadata("key", "value")
	event.SetPriority(PriorityHigh)

	cloned := event.Clone()

	if cloned.GetName() != event.GetName() {
		t.Error("Cloned event name mismatch")
	}

	if cloned.GetPayload() != event.GetPayload() {
		t.Error("Cloned event payload mismatch")
	}

	if cloned.(*BaseEvent).GetPriority() != event.GetPriority() {
		t.Error("Cloned event priority mismatch")
	}

	clonedMetadata := cloned.GetMetadata()
	originalMetadata := event.GetMetadata()

	if len(clonedMetadata) != len(originalMetadata) {
		t.Error("Cloned event metadata length mismatch")
	}

	for key, value := range originalMetadata {
		if clonedMetadata[key] != value {
			t.Errorf("Cloned metadata mismatch for key %s", key)
		}
	}

	// Ensure they are separate instances
	cloned.(*BaseEvent).SetMetadata("new_key", "new_value")
	if _, exists := event.GetMetadataValue("new_key"); exists {
		t.Error("Original event was modified when cloned event was changed")
	}
}

func TestUserEvent(t *testing.T) {
	userID := "user123"
	userData := map[string]interface{}{"name": "John Doe"}

	event := NewUserEvent("user.login", userID, userData)

	if event.GetName() != "user.login" {
		t.Errorf("Expected name user.login, got %s", event.GetName())
	}

	if event.UserID != userID {
		t.Errorf("Expected UserID %s, got %s", userID, event.UserID)
	}

	// Compare payload as interface{} since we can't directly compare maps
	if fmt.Sprintf("%v", event.GetPayload()) != fmt.Sprintf("%v", userData) {
		t.Error("Payload mismatch")
	}

	// Check metadata
	userIDMeta, exists := event.GetMetadataValue("user_id")
	if !exists {
		t.Error("Expected user_id in metadata")
	}
	if userIDMeta != userID {
		t.Errorf("Expected user_id %s in metadata, got %v", userID, userIDMeta)
	}
}

func TestRequestEvent(t *testing.T) {
	event := NewRequestEvent("request.received", "GET", "/api/users")

	if event.Method != "GET" {
		t.Errorf("Expected method GET, got %s", event.Method)
	}

	if event.URL != "/api/users" {
		t.Errorf("Expected URL /api/users, got %s", event.URL)
	}

	if event.Headers == nil {
		t.Error("Expected headers to be initialized")
	}
}

func TestErrorEvent(t *testing.T) {
	err := fmt.Errorf("test error")
	event := NewErrorEvent("error.occurred", err)

	if event.Error != err {
		t.Error("Error mismatch")
	}

	if event.Message != err.Error() {
		t.Errorf("Expected message %s, got %s", err.Error(), event.Message)
	}

	if event.GetPayload() != err {
		t.Error("Payload should be the error")
	}
}

func TestSystemEvent(t *testing.T) {
	event := NewSystemEvent("system.started", "database", "connect")

	if event.Component != "database" {
		t.Errorf("Expected component database, got %s", event.Component)
	}

	if event.Action != "connect" {
		t.Errorf("Expected action connect, got %s", event.Action)
	}

	if event.Details == nil {
		t.Error("Expected details to be initialized")
	}
}

func TestDatabaseEvent(t *testing.T) {
	event := NewDatabaseEvent("database.query", "users", "SELECT")

	if event.Table != "users" {
		t.Errorf("Expected table users, got %s", event.Table)
	}

	if event.Operation != "SELECT" {
		t.Errorf("Expected operation SELECT, got %s", event.Operation)
	}

	if event.Data == nil {
		t.Error("Expected data to be initialized")
	}
}

func TestCacheEvent(t *testing.T) {
	event := NewCacheEvent("cache.hit", "memory", "user:123", true)

	if event.Store != "memory" {
		t.Errorf("Expected store memory, got %s", event.Store)
	}

	if event.Key != "user:123" {
		t.Errorf("Expected key user:123, got %s", event.Key)
	}

	if event.Hit != true {
		t.Error("Expected hit to be true")
	}
}

func TestEventBuilder(t *testing.T) {
	ctx := context.Background()
	payload := "test payload"

	event := NewEventBuilder("test.event").
		WithPayload(payload).
		WithContext(ctx).
		WithMetadata("key", "value").
		WithPriority(PriorityHigh).
		Build()

	if event.GetName() != "test.event" {
		t.Error("Event name mismatch")
	}

	if event.GetPayload() != payload {
		t.Error("Event payload mismatch")
	}

	if event.GetContext() != ctx {
		t.Error("Event context mismatch")
	}

	if baseEvent, ok := event.(*BaseEvent); ok {
		if baseEvent.GetPriority() != PriorityHigh {
			t.Error("Event priority mismatch")
		}

		value, exists := baseEvent.GetMetadataValue("key")
		if !exists || value != "value" {
			t.Error("Event metadata mismatch")
		}
	} else {
		t.Error("Expected BaseEvent type")
	}
}

func TestPriorityString(t *testing.T) {
	tests := []struct {
		priority Priority
		expected string
	}{
		{PriorityLow, "low"},
		{PriorityNormal, "normal"},
		{PriorityHigh, "high"},
		{PriorityCritical, "critical"},
		{Priority(999), "unknown"},
	}

	for _, test := range tests {
		if test.priority.String() != test.expected {
			t.Errorf("Expected %s for priority %d, got %s", test.expected, test.priority, test.priority.String())
		}
	}
}

func TestNewEventWithContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), "test", "value")
	event := NewEventWithContext(ctx, "test.event", "payload")

	if event.GetContext() != ctx {
		t.Error("Context not set correctly")
	}

	if event.GetName() != "test.event" {
		t.Error("Name not set correctly")
	}

	if event.GetPayload() != "payload" {
		t.Error("Payload not set correctly")
	}
}

func TestEventTimestamp(t *testing.T) {
	beforeCreation := time.Now()
	event := NewBaseEvent("test", nil)
	afterCreation := time.Now()

	timestamp := event.GetTimestamp()

	if timestamp.Before(beforeCreation) || timestamp.After(afterCreation) {
		t.Error("Event timestamp not within expected range")
	}
}

func TestEventConcurrency(t *testing.T) {
	event := NewBaseEvent("test", nil)

	// Test concurrent metadata access
	done := make(chan bool, 2)

	go func() {
		for i := 0; i < 100; i++ {
			event.SetMetadata("key", i)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			event.GetMetadata()
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// Should not panic or race
}