package queue

import (
	"context"
	"testing"
	"time"
)

func TestMemoryQueue_PushPop(t *testing.T) {
	queue := NewMemoryQueue()
	defer queue.Close()

	ctx := context.Background()
	job := NewCallbackJob("test", func(ctx context.Context) error {
		return nil
	})

	// Test Push
	err := queue.Push(ctx, job)
	if err != nil {
		t.Fatalf("Failed to push job: %v", err)
	}

	// Test size
	size, err := queue.Size(ctx)
	if err != nil {
		t.Fatalf("Failed to get size: %v", err)
	}
	if size != 1 {
		t.Errorf("Expected size 1, got %d", size)
	}

	// Test Pop
	poppedJob, err := queue.Pop(ctx)
	if err != nil {
		t.Fatalf("Failed to pop job: %v", err)
	}

	if poppedJob == nil {
		t.Error("Expected job, got nil")
	}

	// Queue should be empty now
	size, _ = queue.Size(ctx)
	if size != 0 {
		t.Errorf("Expected size 0 after pop, got %d", size)
	}
}

func TestMemoryQueue_PushOn(t *testing.T) {
	queue := NewMemoryQueue()
	defer queue.Close()

	ctx := context.Background()
	job := NewCallbackJob("test", func(ctx context.Context) error {
		return nil
	})

	// Push to specific queue
	err := queue.PushOn(ctx, "custom-queue", job)
	if err != nil {
		t.Fatalf("Failed to push job: %v", err)
	}

	// Default queue should be empty
	size, _ := queue.Size(ctx)
	if size != 0 {
		t.Errorf("Expected default queue size 0, got %d", size)
	}

	// Custom queue should have 1 job
	size, _ = queue.Size(ctx, "custom-queue")
	if size != 1 {
		t.Errorf("Expected custom queue size 1, got %d", size)
	}

	// Pop from custom queue
	poppedJob, err := queue.Pop(ctx, "custom-queue")
	if err != nil {
		t.Fatalf("Failed to pop from custom queue: %v", err)
	}

	if poppedJob == nil {
		t.Error("Expected job, got nil")
	}
}

func TestMemoryQueue_Later(t *testing.T) {
	queue := NewMemoryQueue()
	defer queue.Close()

	ctx := context.Background()
	job := NewCallbackJob("test", func(ctx context.Context) error {
		return nil
	})

	// Push with delay
	delay := 100 * time.Millisecond
	err := queue.Later(ctx, delay, job)
	if err != nil {
		t.Fatalf("Failed to push delayed job: %v", err)
	}

	// Job should not be available immediately
	_, err = queue.Pop(ctx)
	if err == nil {
		t.Error("Expected error when popping delayed job immediately")
	}

	// Wait for delay
	time.Sleep(delay + 10*time.Millisecond)

	// Job should be available now
	poppedJob, err := queue.Pop(ctx)
	if err != nil {
		t.Fatalf("Failed to pop delayed job: %v", err)
	}

	if poppedJob == nil {
		t.Error("Expected job, got nil")
	}
}

func TestMemoryQueue_Peek(t *testing.T) {
	queue := NewMemoryQueue()
	defer queue.Close()

	ctx := context.Background()
	job := NewCallbackJob("test", func(ctx context.Context) error {
		return nil
	})

	// Push job
	err := queue.Push(ctx, job)
	if err != nil {
		t.Fatalf("Failed to push job: %v", err)
	}

	// Peek should return job without removing it
	peekedJob, err := queue.Peek(ctx)
	if err != nil {
		t.Fatalf("Failed to peek job: %v", err)
	}

	if peekedJob == nil {
		t.Error("Expected job, got nil")
	}

	// Job should still be in queue
	size, _ := queue.Size(ctx)
	if size != 1 {
		t.Errorf("Expected size 1 after peek, got %d", size)
	}

	// Pop should still work
	_, err = queue.Pop(ctx)
	if err != nil {
		t.Fatalf("Failed to pop after peek: %v", err)
	}
}

func TestMemoryQueue_Clear(t *testing.T) {
	queue := NewMemoryQueue()
	defer queue.Close()

	ctx := context.Background()

	// Push multiple jobs
	for i := 0; i < 5; i++ {
		job := NewCallbackJob("test", func(ctx context.Context) error {
			return nil
		})
		queue.Push(ctx, job)
	}

	// Verify jobs are there
	size, _ := queue.Size(ctx)
	if size != 5 {
		t.Errorf("Expected size 5, got %d", size)
	}

	// Clear queue
	err := queue.Clear(ctx, "default")
	if err != nil {
		t.Fatalf("Failed to clear queue: %v", err)
	}

	// Queue should be empty
	size, _ = queue.Size(ctx)
	if size != 0 {
		t.Errorf("Expected size 0 after clear, got %d", size)
	}
}

func TestMemoryQueue_Stats(t *testing.T) {
	queue := NewMemoryQueue()
	defer queue.Close()

	ctx := context.Background()

	// Get stats for empty queue
	stats, err := queue.Stats(ctx)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats.Name != "default" {
		t.Errorf("Expected default queue name, got %s", stats.Name)
	}

	if stats.Size != 0 {
		t.Errorf("Expected size 0, got %d", stats.Size)
	}

	// Push a job
	job := NewCallbackJob("test", func(ctx context.Context) error {
		return nil
	})
	queue.Push(ctx, job)

	// Get updated stats
	stats, _ = queue.Stats(ctx)
	if stats.Size != 1 {
		t.Errorf("Expected size 1, got %d", stats.Size)
	}

	if stats.TotalJobs != 1 {
		t.Errorf("Expected total jobs 1, got %d", stats.TotalJobs)
	}
}

func TestMemoryQueue_Priority(t *testing.T) {
	queue := NewMemoryQueue()
	defer queue.Close()

	ctx := context.Background()

	// Push jobs with different priorities
	lowJob := NewCallbackJob("low", func(ctx context.Context) error { return nil })
	lowJob.WithPriority(PriorityLow)

	highJob := NewCallbackJob("high", func(ctx context.Context) error { return nil })
	highJob.WithPriority(PriorityHigh)

	normalJob := NewCallbackJob("normal", func(ctx context.Context) error { return nil })
	normalJob.WithPriority(PriorityNormal)

	// Push in random order
	queue.Push(ctx, lowJob)
	queue.Push(ctx, highJob)
	queue.Push(ctx, normalJob)

	// Pop should return high priority first
	poppedJob, err := queue.Pop(ctx)
	if err != nil {
		t.Fatalf("Failed to pop job: %v", err)
	}

	if queueJob, ok := poppedJob.(*QueueJob); ok {
		if name, exists := queueJob.GetPayloadValue("name"); !exists || name != "high" {
			t.Error("Expected high priority job to be popped first")
		}
	}
}

func TestMemoryQueue_Concurrency(t *testing.T) {
	queue := NewMemoryQueue()
	defer queue.Close()

	ctx := context.Background()
	numWorkers := 10
	jobsPerWorker := 10

	// Push jobs concurrently
	done := make(chan bool, numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			defer func() { done <- true }()
			for j := 0; j < jobsPerWorker; j++ {
				job := NewCallbackJob("test", func(ctx context.Context) error {
					return nil
				})
				queue.Push(ctx, job)
			}
		}(i)
	}

	// Wait for all workers to finish
	for i := 0; i < numWorkers; i++ {
		<-done
	}

	// Check total jobs
	size, err := queue.Size(ctx)
	if err != nil {
		t.Fatalf("Failed to get size: %v", err)
	}

	expectedSize := numWorkers * jobsPerWorker
	if size != expectedSize {
		t.Errorf("Expected size %d, got %d", expectedSize, size)
	}

	// Pop all jobs
	popped := 0
	for {
		_, err := queue.Pop(ctx)
		if err != nil {
			break
		}
		popped++
	}

	if popped != expectedSize {
		t.Errorf("Expected to pop %d jobs, got %d", expectedSize, popped)
	}
}

func TestMemoryQueue_ContextCancellation(t *testing.T) {
	queue := NewMemoryQueue()
	defer queue.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	job := NewCallbackJob("test", func(ctx context.Context) error {
		return nil
	})

	// Operations should respect context cancellation
	err := queue.Push(ctx, job)
	if err != context.Canceled {
		t.Errorf("Expected context canceled error, got %v", err)
	}

	_, err = queue.Pop(ctx)
	if err != context.Canceled {
		t.Errorf("Expected context canceled error, got %v", err)
	}
}

func TestMemoryQueue_Close(t *testing.T) {
	queue := NewMemoryQueue()
	ctx := context.Background()

	// Push a job
	job := NewCallbackJob("test", func(ctx context.Context) error {
		return nil
	})
	queue.Push(ctx, job)

	// Close queue
	err := queue.Close()
	if err != nil {
		t.Fatalf("Failed to close queue: %v", err)
	}

	// Operations should fail after close
	err = queue.Push(ctx, job)
	if err == nil {
		t.Error("Expected error when pushing to closed queue")
	}

	_, err = queue.Pop(ctx)
	if err == nil {
		t.Error("Expected error when popping from closed queue")
	}
}

func TestPriorityQueue(t *testing.T) {
	queue := NewPriorityQueue()
	defer queue.Close()

	ctx := context.Background()

	// Create jobs with different priorities and times
	jobs := []struct {
		name     string
		priority Priority
	}{
		{"low1", PriorityLow},
		{"high1", PriorityHigh},
		{"normal1", PriorityNormal},
		{"critical1", PriorityCritical},
		{"high2", PriorityHigh},
		{"normal2", PriorityNormal},
	}

	// Push all jobs
	for _, jobInfo := range jobs {
		job := NewCallbackJob(jobInfo.name, func(ctx context.Context) error { return nil })
		job.WithPriority(jobInfo.priority)
		time.Sleep(1 * time.Millisecond) // Ensure different creation times
		queue.Push(ctx, job)
	}

	// Pop all jobs and verify order
	expectedOrder := []Priority{
		PriorityCritical, // critical1
		PriorityHigh,     // high1 (older)
		PriorityHigh,     // high2 (newer)
		PriorityNormal,   // normal1 (older)
		PriorityNormal,   // normal2 (newer)
		PriorityLow,      // low1
	}

	for i, expectedPriority := range expectedOrder {
		poppedJob, err := queue.Pop(ctx)
		if err != nil {
			t.Fatalf("Failed to pop job %d: %v", i, err)
		}

		if poppedJob.GetPriority() != expectedPriority {
			t.Errorf("Job %d: expected priority %v, got %v", i, expectedPriority, poppedJob.GetPriority())
		}
	}
}

func TestMemoryQueue_MultipleQueues(t *testing.T) {
	queue := NewMemoryQueue()
	defer queue.Close()

	ctx := context.Background()

	// Push to multiple queues
	queues := []string{"queue1", "queue2", "queue3"}
	for _, queueName := range queues {
		for i := 0; i < 3; i++ {
			job := NewCallbackJob("test", func(ctx context.Context) error { return nil })
			queue.PushOn(ctx, queueName, job)
		}
	}

	// Verify each queue has correct size
	for _, queueName := range queues {
		size, err := queue.Size(ctx, queueName)
		if err != nil {
			t.Fatalf("Failed to get size for %s: %v", queueName, err)
		}
		if size != 3 {
			t.Errorf("Expected size 3 for %s, got %d", queueName, size)
		}
	}

	// Get all queue names
	allQueues := queue.GetAllQueues()
	if len(allQueues) != 3 {
		t.Errorf("Expected 3 queues, got %d", len(allQueues))
	}

	// Clear one queue
	queue.Clear(ctx, "queue1")
	size, _ := queue.Size(ctx, "queue1")
	if size != 0 {
		t.Errorf("Expected queue1 to be empty after clear, got size %d", size)
	}

	// Other queues should be unaffected
	size, _ = queue.Size(ctx, "queue2")
	if size != 3 {
		t.Errorf("Expected queue2 size 3 after clearing queue1, got %d", size)
	}
}