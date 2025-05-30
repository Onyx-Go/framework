package app

import (
	"errors"
	"testing"
)

func TestNewContainer(t *testing.T) {
	container := NewContainer()
	
	if container == nil {
		t.Fatal("NewContainer() returned nil")
	}
	
	if container.bindings == nil {
		t.Error("container.bindings should be initialized")
	}
	
	if container.instances == nil {
		t.Error("container.instances should be initialized")
	}
}

func TestContainerBind(t *testing.T) {
	container := NewContainer()
	
	factory := func() string {
		return "test"
	}
	
	container.Bind("test", factory)
	
	if !container.Has("test") {
		t.Error("container should have 'test' binding")
	}
}

func TestContainerSingleton(t *testing.T) {
	container := NewContainer()
	counter := 0
	
	factory := func() int {
		counter++
		return counter
	}
	
	container.Singleton("counter", factory)
	
	// First call
	result1, err := container.Make("counter")
	if err != nil {
		t.Fatalf("Make() returned error: %v", err)
	}
	
	// Second call
	result2, err := container.Make("counter")
	if err != nil {
		t.Fatalf("Make() returned error: %v", err)
	}
	
	if result1 != result2 {
		t.Error("singleton should return the same instance")
	}
	
	if result1.(int) != 1 {
		t.Errorf("expected 1, got %v", result1)
	}
}

func TestContainerTransient(t *testing.T) {
	container := NewContainer()
	counter := 0
	
	factory := func() int {
		counter++
		return counter
	}
	
	container.Bind("counter", factory)
	
	// First call
	result1, err := container.Make("counter")
	if err != nil {
		t.Fatalf("Make() returned error: %v", err)
	}
	
	// Second call
	result2, err := container.Make("counter")
	if err != nil {
		t.Fatalf("Make() returned error: %v", err)
	}
	
	if result1 == result2 {
		t.Error("transient binding should return different instances")
	}
	
	if result1.(int) != 1 || result2.(int) != 2 {
		t.Errorf("expected 1 and 2, got %v and %v", result1, result2)
	}
}

func TestContainerInstance(t *testing.T) {
	container := NewContainer()
	instance := "test instance"
	
	container.Instance("test", instance)
	
	result, err := container.Make("test")
	if err != nil {
		t.Fatalf("Make() returned error: %v", err)
	}
	
	if result != instance {
		t.Errorf("expected %v, got %v", instance, result)
	}
}

func TestContainerMakeNotFound(t *testing.T) {
	container := NewContainer()
	
	_, err := container.Make("nonexistent")
	if err == nil {
		t.Error("Make() should return error for nonexistent binding")
	}
}

func TestContainerDependencyInjection(t *testing.T) {
	container := NewContainer()
	
	// Register a dependency by type name that Go would use
	container.Instance("string", "test-config")
	
	// Register a service that depends on config
	container.Bind("service", func(config string) string {
		return "service with " + config
	})
	
	result, err := container.Make("service")
	if err != nil {
		t.Fatalf("Make() returned error: %v", err)
	}
	
	expected := "service with test-config"
	if result != expected {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestContainerFactoryError(t *testing.T) {
	container := NewContainer()
	
	factory := func() (string, error) {
		return "", errors.New("factory error")
	}
	
	container.Bind("failing", factory)
	
	_, err := container.Make("failing")
	if err == nil {
		t.Error("Make() should return error when factory fails")
	}
}

func TestContainerClear(t *testing.T) {
	container := NewContainer()
	
	container.Bind("test", "value")
	container.Instance("instance", "value")
	
	if !container.Has("test") || !container.Has("instance") {
		t.Error("container should have bindings before clear")
	}
	
	container.Clear()
	
	if container.Has("test") || container.Has("instance") {
		t.Error("container should not have bindings after clear")
	}
}

func TestContainerConcurrency(t *testing.T) {
	container := NewContainer()
	counter := 0
	
	factory := func() int {
		counter++
		return counter
	}
	
	container.Singleton("counter", factory)
	
	// Simulate concurrent access
	done := make(chan bool, 10)
	results := make(chan interface{}, 10)
	
	for i := 0; i < 10; i++ {
		go func() {
			result, err := container.Make("counter")
			if err != nil {
				t.Errorf("Make() returned error: %v", err)
			}
			results <- result
			done <- true
		}()
	}
	
	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
	close(results)
	
	// All results should be the same (singleton)
	firstResult := <-results
	for result := range results {
		if result != firstResult {
			t.Error("singleton should return the same instance across goroutines")
		}
	}
}

type testServiceProvider struct {
	registered bool
	booted     bool
}

func (p *testServiceProvider) Register(c *Container) {
	p.registered = true
	c.Bind("provider-service", "test")
}

func (p *testServiceProvider) Boot(c *Container) {
	p.booted = true
}

func TestContainerRegisterProvider(t *testing.T) {
	container := NewContainer()
	provider := &testServiceProvider{}
	
	container.RegisterProvider(provider)
	
	if !provider.registered {
		t.Error("provider should be registered")
	}
	
	if !provider.booted {
		t.Error("provider should be booted")
	}
	
	if !container.Has("provider-service") {
		t.Error("container should have provider service")
	}
}