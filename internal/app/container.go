package app

import (
	"fmt"
	"reflect"
	"sync"
)

// Container provides dependency injection functionality
type Container struct {
	bindings  map[string]interface{}
	instances map[string]interface{}
	mutex     sync.RWMutex
}

// ServiceProvider interface for service registration and bootstrapping
type ServiceProvider interface {
	Register(*Container)
	Boot(*Container)
}

// Binding represents different types of service bindings
type Binding interface {
	Resolve(*Container) (interface{}, error)
}

// singletonBinding represents a singleton service binding
type singletonBinding struct {
	factory interface{}
}

func (sb *singletonBinding) Resolve(c *Container) (interface{}, error) {
	return c.resolve(sb.factory)
}

// transientBinding represents a transient service binding
type transientBinding struct {
	factory interface{}
}

func (tb *transientBinding) Resolve(c *Container) (interface{}, error) {
	return c.resolve(tb.factory)
}

// NewContainer creates a new dependency injection container
func NewContainer() *Container {
	return &Container{
		bindings:  make(map[string]interface{}),
		instances: make(map[string]interface{}),
	}
}

// Bind registers a transient service binding
func (c *Container) Bind(name string, factory interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.bindings[name] = &transientBinding{factory: factory}
}

// Singleton registers a singleton service binding
func (c *Container) Singleton(name string, factory interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.bindings[name] = &singletonBinding{factory: factory}
}

// Instance registers a pre-created instance
func (c *Container) Instance(name string, instance interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.instances[name] = instance
}

// Make resolves a service from the container
func (c *Container) Make(name string) (interface{}, error) {
	c.mutex.RLock()
	
	// Check for existing instance first
	if instance, exists := c.instances[name]; exists {
		c.mutex.RUnlock()
		return instance, nil
	}
	
	// Check for binding
	binding, exists := c.bindings[name]
	if !exists {
		c.mutex.RUnlock()
		return nil, fmt.Errorf("binding not found: %s", name)
	}
	
	c.mutex.RUnlock()
	
	// Handle singleton bindings
	if singleton, ok := binding.(*singletonBinding); ok {
		return c.resolveSingleton(name, singleton)
	}
	
	// Handle transient bindings
	if transient, ok := binding.(*transientBinding); ok {
		return transient.Resolve(c)
	}
	
	// Fallback for legacy bindings (direct factory functions)
	return c.resolve(binding)
}

// resolveSingleton handles singleton resolution with proper locking
func (c *Container) resolveSingleton(name string, singleton *singletonBinding) (interface{}, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	// Double-check pattern - instance might have been created while waiting for lock
	if instance, exists := c.instances[name]; exists {
		return instance, nil
	}
	
	// Resolve the singleton
	instance, err := c.resolve(singleton.factory)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve singleton %s: %w", name, err)
	}
	
	// Store the instance
	c.instances[name] = instance
	return instance, nil
}

// resolve calls a factory function with dependency injection
func (c *Container) resolve(factory interface{}) (interface{}, error) {
	factoryValue := reflect.ValueOf(factory)
	factoryType := factoryValue.Type()
	
	// If it's not a function, return as-is
	if factoryType.Kind() != reflect.Func {
		return factory, nil
	}
	
	// Prepare arguments for the factory function
	numIn := factoryType.NumIn()
	args := make([]reflect.Value, numIn)
	
	for i := 0; i < numIn; i++ {
		argType := factoryType.In(i)
		
		// Special case: inject the container itself
		if argType == reflect.TypeOf((*Container)(nil)) {
			args[i] = reflect.ValueOf(c)
			continue
		}
		
		// Resolve dependency by type name
		typeName := argType.String()
		dependency, err := c.Make(typeName)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve dependency %s: %w", typeName, err)
		}
		
		args[i] = reflect.ValueOf(dependency)
	}
	
	// Call the factory function
	results := factoryValue.Call(args)
	
	if len(results) == 0 {
		return nil, fmt.Errorf("factory function must return at least one value")
	}
	
	// Handle error return value
	if len(results) == 2 && !results[1].IsNil() {
		if err, ok := results[1].Interface().(error); ok {
			return nil, err
		}
	}
	
	return results[0].Interface(), nil
}

// Has checks if a service is registered in the container
func (c *Container) Has(name string) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	_, hasInstance := c.instances[name]
	_, hasBinding := c.bindings[name]
	
	return hasInstance || hasBinding
}

// RegisterProvider registers and boots a service provider
func (c *Container) RegisterProvider(provider ServiceProvider) {
	provider.Register(c)
	provider.Boot(c)
}

// Clear removes all bindings and instances (useful for testing)
func (c *Container) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	c.bindings = make(map[string]interface{})
	c.instances = make(map[string]interface{})
}

// GetBindings returns a copy of all registered bindings (for debugging)
func (c *Container) GetBindings() map[string]interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	bindings := make(map[string]interface{})
	for k, v := range c.bindings {
		bindings[k] = v
	}
	return bindings
}

// GetInstances returns a copy of all resolved instances (for debugging)
func (c *Container) GetInstances() map[string]interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	instances := make(map[string]interface{})
	for k, v := range c.instances {
		instances[k] = v
	}
	return instances
}