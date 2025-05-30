package events

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestDefaultDispatcher_Listen(t *testing.T) {
	dispatcher := NewDefaultDispatcher(NewNoopMetrics())

	listener := ListenerFunc(func(ctx context.Context, event Event) error {
		return nil
	})

	dispatcher.Listen("test.event", listener)

	if !dispatcher.HasListeners("test.event") {
		t.Error("Expected dispatcher to have listeners for test.event")
	}

	listeners := dispatcher.GetListeners("test.event")
	if len(listeners) != 1 {
		t.Errorf("Expected 1 listener, got %d", len(listeners))
	}
}

func TestDefaultDispatcher_Dispatch(t *testing.T) {
	dispatcher := NewDefaultDispatcher(NewNoopMetrics())
	var receivedEvent Event
	called := false

	listener := ListenerFunc(func(ctx context.Context, event Event) error {
		called = true
		receivedEvent = event
		return nil
	})

	dispatcher.Listen("test.event", listener)

	event := NewBaseEvent("test.event", "test payload")
	err := dispatcher.Dispatch(context.Background(), event)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !called {
		t.Error("Expected listener to be called")
	}

	if receivedEvent != event {
		t.Error("Listener received wrong event")
	}
}

func TestDefaultDispatcher_DispatchError(t *testing.T) {
	dispatcher := NewDefaultDispatcher(NewNoopMetrics())
	expectedError := fmt.Errorf("listener error")

	listener := ListenerFunc(func(ctx context.Context, event Event) error {
		return expectedError
	})

	dispatcher.Listen("test.event", listener)

	event := NewBaseEvent("test.event", nil)
	err := dispatcher.Dispatch(context.Background(), event)

	if err == nil {
		t.Error("Expected error from dispatch")
	}

	if err.Error() != fmt.Sprintf("listener error for event test.event: %v", expectedError) {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestDefaultDispatcher_WildcardListeners(t *testing.T) {
	dispatcher := NewDefaultDispatcher(NewNoopMetrics())
	callCount := 0

	wildcardListener := ListenerFunc(func(ctx context.Context, event Event) error {
		callCount++
		return nil
	})

	dispatcher.Listen("user.*", wildcardListener)

	// Test matching events
	events := []string{"user.created", "user.updated", "user.deleted"}
	for _, eventName := range events {
		event := NewBaseEvent(eventName, nil)
		err := dispatcher.Dispatch(context.Background(), event)
		if err != nil {
			t.Errorf("Unexpected error for event %s: %v", eventName, err)
		}
	}

	if callCount != 3 {
		t.Errorf("Expected wildcard listener to be called 3 times, got %d", callCount)
	}

	// Test non-matching event
	event := NewBaseEvent("order.created", nil)
	err := dispatcher.Dispatch(context.Background(), event)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if callCount != 3 {
		t.Errorf("Expected call count to remain 3, got %d", callCount)
	}
}

func TestDefaultDispatcher_Push(t *testing.T) {
	dispatcher := NewDefaultDispatcher(NewNoopMetrics())
	called := false

	listener := ListenerFunc(func(ctx context.Context, event Event) error {
		called = true
		return nil
	})

	dispatcher.Listen("test.event", listener)

	// Push an event
	dispatcher.Push("test.event", "test payload")

	// Event shouldn't be dispatched yet
	if called {
		t.Error("Expected listener not to be called before flush")
	}

	// Flush the event
	err := dispatcher.Flush(context.Background(), "test.event")
	if err != nil {
		t.Errorf("Unexpected error during flush: %v", err)
	}

	if !called {
		t.Error("Expected listener to be called after flush")
	}
}

func TestDefaultDispatcher_Forget(t *testing.T) {
	dispatcher := NewDefaultDispatcher(NewNoopMetrics())

	listener := ListenerFunc(func(ctx context.Context, event Event) error {
		return nil
	})

	dispatcher.Listen("test.event", listener)

	if !dispatcher.HasListeners("test.event") {
		t.Error("Expected dispatcher to have listeners")
	}

	dispatcher.Forget("test.event")

	if dispatcher.HasListeners("test.event") {
		t.Error("Expected dispatcher to have no listeners after forget")
	}
}

func TestDefaultDispatcher_ForgetPushed(t *testing.T) {
	dispatcher := NewDefaultDispatcher(NewNoopMetrics())

	dispatcher.Push("test.event", "payload1")
	dispatcher.Push("test.event", "payload2")

	// Verify events are pushed
	if len(dispatcher.pushed["test.event"]) != 2 {
		t.Errorf("Expected 2 pushed events, got %d", len(dispatcher.pushed["test.event"]))
	}

	dispatcher.ForgetPushed("test.event")

	// Verify events are cleared
	if len(dispatcher.pushed["test.event"]) != 0 {
		t.Errorf("Expected 0 pushed events after forget, got %d", len(dispatcher.pushed["test.event"]))
	}
}

func TestDefaultDispatcher_DispatchUntil(t *testing.T) {
	dispatcher := NewDefaultDispatcher(NewNoopMetrics())
	callCount := 0

	listener1 := ListenerFunc(func(ctx context.Context, event Event) error {
		callCount++
		return nil
	})

	listener2 := ListenerFunc(func(ctx context.Context, event Event) error {
		callCount++
		return fmt.Errorf("stop here")
	})

	listener3 := ListenerFunc(func(ctx context.Context, event Event) error {
		callCount++
		return nil
	})

	dispatcher.Listen("test.event", listener1)
	dispatcher.Listen("test.event", listener2)
	dispatcher.Listen("test.event", listener3)

	event := NewBaseEvent("test.event", nil)
	result, err := dispatcher.DispatchUntil(context.Background(), event, func(result interface{}) bool {
		// Halt on any error
		return result != nil
	})

	if result == nil {
		t.Error("Expected result from DispatchUntil")
	}

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Should stop after listener2 returns error
	if callCount != 2 {
		t.Errorf("Expected 2 calls, got %d", callCount)
	}
}

func TestDefaultDispatcher_Subscribe(t *testing.T) {
	dispatcher := NewDefaultDispatcher(NewNoopMetrics())
	subscribed := false

	subscriber := &testSubscriber{
		subscribeFunc: func(d Dispatcher) {
			subscribed = true
			d.Listen("test.event", ListenerFunc(func(ctx context.Context, event Event) error {
				return nil
			}))
		},
	}

	dispatcher.Subscribe(subscriber)

	if !subscribed {
		t.Error("Expected subscriber to be called")
	}

	if !dispatcher.HasListeners("test.event") {
		t.Error("Expected dispatcher to have listeners after subscription")
	}
}

func TestDefaultDispatcher_Concurrency(t *testing.T) {
	dispatcher := NewDefaultDispatcher(NewNoopMetrics())
	var wg sync.WaitGroup
	callCount := int64(0)

	listener := ListenerFunc(func(ctx context.Context, event Event) error {
		callCount++
		return nil
	})

	dispatcher.Listen("test.event", listener)

	// Dispatch events concurrently
	numGoroutines := 10
	eventsPerGoroutine := 10

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				event := NewBaseEvent("test.event", nil)
				dispatcher.Dispatch(context.Background(), event)
			}
		}()
	}

	wg.Wait()

	expectedCalls := int64(numGoroutines * eventsPerGoroutine)
	if callCount != expectedCalls {
		t.Errorf("Expected %d calls, got %d", expectedCalls, callCount)
	}
}

func TestAsyncDispatcher(t *testing.T) {
	dispatcher := NewAsyncDispatcher(NewNoopMetrics(), 2)
	defer dispatcher.Close()

	var mu sync.Mutex
	var receivedEvents []Event

	listener := ListenerFunc(func(ctx context.Context, event Event) error {
		mu.Lock()
		receivedEvents = append(receivedEvents, event)
		mu.Unlock()
		return nil
	})

	dispatcher.Listen("test.event", listener)

	// Dispatch multiple events
	numEvents := 5
	for i := 0; i < numEvents; i++ {
		event := NewBaseEvent("test.event", fmt.Sprintf("payload-%d", i))
		err := dispatcher.Dispatch(context.Background(), event)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	}

	// Wait a bit for async processing
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	if len(receivedEvents) != numEvents {
		t.Errorf("Expected %d events, got %d", numEvents, len(receivedEvents))
	}
	mu.Unlock()
}

func TestAsyncDispatcher_DispatchAsync(t *testing.T) {
	dispatcher := NewAsyncDispatcher(NewNoopMetrics(), 2)
	defer dispatcher.Close()

	var mu sync.Mutex
	callCount := 0

	listener := ListenerFunc(func(ctx context.Context, event Event) error {
		mu.Lock()
		callCount++
		mu.Unlock()
		return nil
	})

	dispatcher.Listen("test.event", listener)

	// Dispatch async (fire and forget)
	event := NewBaseEvent("test.event", nil)
	err := dispatcher.DispatchAsync(context.Background(), event)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	if callCount != 1 {
		t.Errorf("Expected 1 call, got %d", callCount)
	}
	mu.Unlock()
}

func TestPipelineDispatcher(t *testing.T) {
	pipeline := NewPipeline()
	transformerCalled := false

	transformer := TransformerFunc(func(ctx context.Context, event Event) (Event, error) {
		transformerCalled = true
		return event, nil
	})

	pipeline.AddTransformer(transformer)

	dispatcher := NewPipelineDispatcher(NewNoopMetrics(), pipeline)
	listenerCalled := false

	listener := ListenerFunc(func(ctx context.Context, event Event) error {
		listenerCalled = true
		return nil
	})

	dispatcher.Listen("test.event", listener)

	event := NewBaseEvent("test.event", nil)
	err := dispatcher.Dispatch(context.Background(), event)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !transformerCalled {
		t.Error("Expected transformer to be called")
	}

	if !listenerCalled {
		t.Error("Expected listener to be called")
	}
}

// Test helper types

type testSubscriber struct {
	subscribeFunc func(Dispatcher)
}

func (ts *testSubscriber) Subscribe(dispatcher Dispatcher) {
	if ts.subscribeFunc != nil {
		ts.subscribeFunc(dispatcher)
	}
}

func TestListenerFunc(t *testing.T) {
	called := false
	var receivedEvent Event

	lf := ListenerFunc(func(ctx context.Context, event Event) error {
		called = true
		receivedEvent = event
		return nil
	})

	event := NewBaseEvent("test", nil)
	err := lf.Handle(context.Background(), event)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !called {
		t.Error("Expected function to be called")
	}

	if receivedEvent != event {
		t.Error("Received wrong event")
	}
}

func TestSyncDispatcher(t *testing.T) {
	dispatcher := NewSyncDispatcher(NewNoopMetrics())
	called := false

	listener := ListenerFunc(func(ctx context.Context, event Event) error {
		called = true
		return nil
	})

	dispatcher.Listen("test.event", listener)

	event := NewBaseEvent("test.event", nil)
	err := dispatcher.Dispatch(context.Background(), event)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !called {
		t.Error("Expected listener to be called")
	}
}

func TestDispatcher_SupportsWildcards(t *testing.T) {
	dispatcher := NewDefaultDispatcher(NewNoopMetrics())

	if !dispatcher.SupportsWildcards() {
		t.Error("Expected dispatcher to support wildcards")
	}
}