package onyx

import (
	framework "github.com/onyx-go/framework"
)

// Application represents the main Onyx application
type Application = framework.Application

// Router represents the HTTP router
type Router = framework.Router

// Container represents the dependency injection container
type Container = framework.Container

// NewApplication creates a new Onyx application instance
func NewApplication() *Application {
	return framework.New()
}

// NewRouter creates a new router instance
func NewRouter() *Router {
	return framework.NewRouter()
}

// NewContainer creates a new dependency injection container
func NewContainer() *Container {
	return framework.NewContainer()
}
