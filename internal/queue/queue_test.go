package queue

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestBaseJob_Creation(t *testing.T) {
	job := NewBaseJob()

	if job.GetQueue() != "default" {
		t.Errorf("Expected default queue, got %s", job.GetQueue())
	}

	if job.GetMaxTries() != 3 {
		t.Errorf("Expected 3 max tries, got %d", job.GetMaxTries())
	}

	if job.GetTimeout() != 60*time.Second {
		t.Errorf("Expected 60s timeout, got %v", job.GetTimeout())
	}

	if job.GetPriority() != PriorityNormal {
		t.Errorf("Expected normal priority, got %v", job.GetPriority())
	}
}

func TestBaseJob_FluentInterface(t *testing.T) {
	job := NewBaseJob().
		OnQueue("test-queue").
		WithPriority(PriorityHigh).
		WithTimeout(30 * time.Second).
		WithMaxTries(5).
		Delay(2 * time.Second).
		WithMetadata("test", "value")

	if job.GetQueue() != "test-queue" {
		t.Errorf("Expected test-queue, got %s", job.GetQueue())
	}

	if job.GetPriority() != PriorityHigh {
		t.Errorf("Expected high priority, got %v", job.GetPriority())
	}

	if job.GetTimeout() != 30*time.Second {
		t.Errorf("Expected 30s timeout, got %v", job.GetTimeout())
	}

	if job.GetMaxTries() != 5 {
		t.Errorf("Expected 5 max tries, got %d", job.GetMaxTries())
	}

	if job.GetDelay() != 2*time.Second {
		t.Errorf("Expected 2s delay, got %v", job.GetDelay())
	}

	metadata := job.GetMetadata()
	if metadata["test"] != "value" {
		t.Errorf("Expected metadata test=value, got %v", metadata["test"])
	}
}

func TestBaseJob_PayloadOperations(t *testing.T) {
	job := NewBaseJob()

	// Test setting payload values
	job.SetPayloadValue("key1", "value1")
	job.SetPayloadValue("key2", 42)

	payload := job.GetPayload()
	if payload["key1"] != "value1" {
		t.Errorf("Expected value1, got %v", payload["key1"])
	}

	if payload["key2"] != 42 {
		t.Errorf("Expected 42, got %v", payload["key2"])
	}

	// Test getting specific payload value
	value, exists := job.GetPayloadValue("key1")
	if !exists {
		t.Error("Expected key1 to exist")
	}
	if value != "value1" {
		t.Errorf("Expected value1, got %v", value)
	}

	// Test non-existent key
	_, exists = job.GetPayloadValue("nonexistent")
	if exists {
		t.Error("Expected nonexistent key to not exist")
	}
}

func TestBaseJob_Clone(t *testing.T) {
	original := NewBaseJob()
	original.OnQueue("test")
	original.WithPriority(PriorityHigh)
	original.SetPayloadValue("key", "value")
	original.WithMetadata("meta", "data")

	clone := original.Clone()

	if clone.GetQueue() != original.GetQueue() {
		t.Error("Cloned job queue mismatch")
	}

	if clone.GetPriority() != original.GetPriority() {
		t.Error("Cloned job priority mismatch")
	}

	// Ensure they are separate instances
	clone.SetPayloadValue("new_key", "new_value")
	if _, exists := original.GetPayloadValue("new_key"); exists {
		t.Error("Original job was modified when clone was changed")
	}
}

func TestCallbackJob(t *testing.T) {
	called := false
	callback := func(ctx context.Context) error {
		called = true
		return nil
	}

	job := NewCallbackJob("test-callback", callback)

	if job.String() != "CallbackJob[test-callback]" {
		t.Errorf("Expected CallbackJob[test-callback], got %s", job.String())
	}

	err := job.Handle(context.Background())
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !called {
		t.Error("Expected callback to be called")
	}
}

func TestCallbackJob_NoCallback(t *testing.T) {
	job := NewCallbackJob("test", nil)

	err := job.Handle(context.Background())
	if err == nil {
		t.Error("Expected error for nil callback")
	}
}

func TestCommandJob(t *testing.T) {
	job := NewCommandJob("echo", "hello", "world")

	if job.Command != "echo" {
		t.Errorf("Expected echo, got %s", job.Command)
	}

	if len(job.Args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(job.Args))
	}

	if job.String() != "CommandJob[echo [hello world]]" {
		t.Errorf("Expected CommandJob[echo [hello world]], got %s", job.String())
	}

	// Check payload
	payload := job.GetPayload()
	if payload["command"] != "echo" {
		t.Errorf("Expected command in payload, got %v", payload["command"])
	}
}

func TestQueueJob(t *testing.T) {
	baseJob := NewBaseJob()
	queueJob := NewQueueJob(baseJob, "test-id")

	if queueJob.GetID() != "test-id" {
		t.Errorf("Expected test-id, got %s", queueJob.GetID())
	}

	if queueJob.GetAttempts() != 0 {
		t.Errorf("Expected 0 attempts, got %d", queueJob.GetAttempts())
	}

	if queueJob.GetStatus() != JobStatusPending {
		t.Errorf("Expected pending status, got %v", queueJob.GetStatus())
	}

	// Test status changes
	queueJob.MarkAsStarted()
	if queueJob.GetStatus() != JobStatusProcessing {
		t.Errorf("Expected processing status, got %v", queueJob.GetStatus())
	}

	queueJob.MarkAsCompleted()
	if queueJob.GetStatus() != JobStatusCompleted {
		t.Errorf("Expected completed status, got %v", queueJob.GetStatus())
	}

	// Test failure
	err := fmt.Errorf("test error")
	queueJob.MarkAsFailed(err)
	if queueJob.GetStatus() != JobStatusFailed {
		t.Errorf("Expected failed status, got %v", queueJob.GetStatus())
	}

	if queueJob.GetLastError() != "test error" {
		t.Errorf("Expected test error, got %s", queueJob.GetLastError())
	}

	if queueJob.GetAttempts() != 1 {
		t.Errorf("Expected 1 attempt, got %d", queueJob.GetAttempts())
	}
}

func TestQueueJob_ShouldRetry(t *testing.T) {
	baseJob := NewBaseJob()
	baseJob.WithMaxTries(3)
	queueJob := NewQueueJob(baseJob, "test-id")

	// Should retry when attempts < max tries
	if !queueJob.ShouldRetry() {
		t.Error("Expected job to be retryable")
	}

	// Mark as failed to increment attempts
	queueJob.MarkAsFailed(fmt.Errorf("error"))
	queueJob.MarkAsFailed(fmt.Errorf("error"))
	queueJob.MarkAsFailed(fmt.Errorf("error"))

	// Should not retry when attempts >= max tries
	if queueJob.ShouldRetry() {
		t.Error("Expected job to not be retryable")
	}
}

func TestQueueJob_ToPayload(t *testing.T) {
	baseJob := NewBaseJob()
	baseJob.OnQueue("test-queue")
	baseJob.WithPriority(PriorityHigh)
	baseJob.SetPayloadValue("key", "value")

	queueJob := NewQueueJob(baseJob, "test-id")
	queueJob.MarkAsStarted()

	payload := queueJob.ToPayload()

	if payload.ID != "test-id" {
		t.Errorf("Expected test-id, got %s", payload.ID)
	}

	if payload.Queue != "test-queue" {
		t.Errorf("Expected test-queue, got %s", payload.Queue)
	}

	if payload.Priority != PriorityHigh {
		t.Errorf("Expected high priority, got %v", payload.Priority)
	}

	if payload.Data["key"] != "value" {
		t.Errorf("Expected payload data, got %v", payload.Data)
	}
}

func TestJobBuilder(t *testing.T) {
	job := NewJobBuilder().
		OnQueue("test").
		WithDelay(5 * time.Second).
		WithPriority(PriorityHigh).
		WithTimeout(30 * time.Second).
		WithMaxTries(5).
		WithPayload(map[string]interface{}{"key": "value"}).
		WithMetadata("meta", "data").
		Build()

	if job.GetQueue() != "test" {
		t.Errorf("Expected test queue, got %s", job.GetQueue())
	}

	if job.GetDelay() != 5*time.Second {
		t.Errorf("Expected 5s delay, got %v", job.GetDelay())
	}

	if job.GetPriority() != PriorityHigh {
		t.Errorf("Expected high priority, got %v", job.GetPriority())
	}

	payload := job.GetPayload()
	if payload["key"] != "value" {
		t.Errorf("Expected payload key=value, got %v", payload["key"])
	}

	metadata := job.GetMetadata()
	if metadata["meta"] != "data" {
		t.Errorf("Expected metadata meta=data, got %v", metadata["meta"])
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

func TestGenerateJobID(t *testing.T) {
	id1 := GenerateJobID()
	id2 := GenerateJobID()

	if id1 == id2 {
		t.Error("Expected unique job IDs")
	}

	if !containsString(id1, "job_") {
		t.Errorf("Expected job ID to contain 'job_', got %s", id1)
	}
}

func TestJobConcurrency(t *testing.T) {
	job := NewBaseJob()

	// Test concurrent access to payload
	done := make(chan bool, 2)

	go func() {
		for i := 0; i < 100; i++ {
			job.SetPayloadValue("key", i)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			job.GetPayload()
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// Should not panic or race
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr || 
		   (len(s) > len(substr) && s[len(s)-len(substr):] == substr) ||
		   (len(s) > len(substr) && contains(s, substr))
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}