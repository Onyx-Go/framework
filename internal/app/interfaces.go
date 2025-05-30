package app

// ContainerInterface defines the contract for dependency injection containers
type ContainerInterface interface {
	// Bind registers a transient service binding
	Bind(name string, factory interface{})
	
	// Singleton registers a singleton service binding
	Singleton(name string, factory interface{})
	
	// Instance registers a pre-created instance
	Instance(name string, instance interface{})
	
	// Make resolves a service from the container
	Make(name string) (interface{}, error)
	
	// Has checks if a service is registered
	Has(name string) bool
	
	// RegisterProvider registers and boots a service provider
	RegisterProvider(provider ServiceProvider)
}

// ServiceProviderInterface defines the contract for service providers
type ServiceProviderInterface interface {
	// Register binds services into the container
	Register(container ContainerInterface)
	
	// Boot performs any initialization after all providers are registered
	Boot(container ContainerInterface)
}

// ApplicationInterface defines the core application contract
type ApplicationInterface interface {
	// Container returns the dependency injection container
	Container() ContainerInterface
	
	// RegisterProvider registers a service provider
	RegisterProvider(provider ServiceProvider)
	
	// Boot boots all registered service providers
	Boot() error
	
	// IsBooted returns whether the application has been booted
	IsBooted() bool
}