package onyx

import (
	"fmt"
	"reflect"
	"sync"
)

type Container struct {
	bindings  map[string]interface{}
	instances map[string]interface{}
	mutex     sync.RWMutex
}

type ServiceProvider interface {
	Register(*Container)
	Boot(*Container)
}

func NewContainer() *Container {
	return &Container{
		bindings:  make(map[string]interface{}),
		instances: make(map[string]interface{}),
	}
}

func (c *Container) Bind(name string, factory interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.bindings[name] = factory
}

func (c *Container) Singleton(name string, factory interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.bindings[name] = &singletonBinding{factory: factory}
}

func (c *Container) Instance(name string, instance interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.instances[name] = instance
}

func (c *Container) Make(name string) (interface{}, error) {
	c.mutex.RLock()
	
	if instance, exists := c.instances[name]; exists {
		c.mutex.RUnlock()
		return instance, nil
	}
	
	binding, exists := c.bindings[name]
	if !exists {
		c.mutex.RUnlock()
		return nil, fmt.Errorf("binding not found: %s", name)
	}
	
	c.mutex.RUnlock()
	
	if singleton, ok := binding.(*singletonBinding); ok {
		c.mutex.Lock()
		defer c.mutex.Unlock()
		
		if instance, exists := c.instances[name]; exists {
			return instance, nil
		}
		
		instance, err := c.resolve(singleton.factory)
		if err != nil {
			return nil, err
		}
		
		c.instances[name] = instance
		return instance, nil
	}
	
	return c.resolve(binding)
}

func (c *Container) resolve(factory interface{}) (interface{}, error) {
	factoryValue := reflect.ValueOf(factory)
	factoryType := factoryValue.Type()
	
	if factoryType.Kind() != reflect.Func {
		return factory, nil
	}
	
	numIn := factoryType.NumIn()
	args := make([]reflect.Value, numIn)
	
	for i := 0; i < numIn; i++ {
		argType := factoryType.In(i)
		
		if argType == reflect.TypeOf((*Container)(nil)) {
			args[i] = reflect.ValueOf(c)
			continue
		}
		
		typeName := argType.String()
		dependency, err := c.Make(typeName)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve dependency %s: %v", typeName, err)
		}
		
		args[i] = reflect.ValueOf(dependency)
	}
	
	results := factoryValue.Call(args)
	
	if len(results) == 0 {
		return nil, fmt.Errorf("factory function must return at least one value")
	}
	
	if len(results) == 2 && !results[1].IsNil() {
		err := results[1].Interface().(error)
		return nil, err
	}
	
	return results[0].Interface(), nil
}

func (c *Container) Has(name string) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	_, hasInstance := c.instances[name]
	_, hasBinding := c.bindings[name]
	
	return hasInstance || hasBinding
}

type singletonBinding struct {
	factory interface{}
}

func (c *Container) RegisterProvider(provider ServiceProvider) {
	provider.Register(c)
	provider.Boot(c)
}