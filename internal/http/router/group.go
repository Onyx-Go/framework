package router

import (
	"strings"

	httpInternal "github.com/onyx-go/framework/internal/http"
)

// RouteGroup represents a group of routes with a common prefix and middleware
type RouteGroup struct {
	router     *Router
	prefix     string
	middleware []httpInternal.MiddlewareFunc
}

// GET registers a GET route in the group
func (g *RouteGroup) GET(pattern string, handler httpInternal.HandlerFunc, middleware ...httpInternal.MiddlewareFunc) {
	fullPattern := g.prefix + pattern
	allMiddleware := append(g.middleware, middleware...)
	g.router.GET(fullPattern, handler, allMiddleware...)
}

// POST registers a POST route in the group
func (g *RouteGroup) POST(pattern string, handler httpInternal.HandlerFunc, middleware ...httpInternal.MiddlewareFunc) {
	fullPattern := g.prefix + pattern
	allMiddleware := append(g.middleware, middleware...)
	g.router.POST(fullPattern, handler, allMiddleware...)
}

// PUT registers a PUT route in the group
func (g *RouteGroup) PUT(pattern string, handler httpInternal.HandlerFunc, middleware ...httpInternal.MiddlewareFunc) {
	fullPattern := g.prefix + pattern
	allMiddleware := append(g.middleware, middleware...)
	g.router.PUT(fullPattern, handler, allMiddleware...)
}

// DELETE registers a DELETE route in the group
func (g *RouteGroup) DELETE(pattern string, handler httpInternal.HandlerFunc, middleware ...httpInternal.MiddlewareFunc) {
	fullPattern := g.prefix + pattern
	allMiddleware := append(g.middleware, middleware...)
	g.router.DELETE(fullPattern, handler, allMiddleware...)
}

// PATCH registers a PATCH route in the group
func (g *RouteGroup) PATCH(pattern string, handler httpInternal.HandlerFunc, middleware ...httpInternal.MiddlewareFunc) {
	fullPattern := g.prefix + pattern
	allMiddleware := append(g.middleware, middleware...)
	g.router.PATCH(fullPattern, handler, allMiddleware...)
}

// OPTIONS registers an OPTIONS route in the group
func (g *RouteGroup) OPTIONS(pattern string, handler httpInternal.HandlerFunc, middleware ...httpInternal.MiddlewareFunc) {
	fullPattern := g.prefix + pattern
	allMiddleware := append(g.middleware, middleware...)
	g.router.OPTIONS(fullPattern, handler, allMiddleware...)
}

// HEAD registers a HEAD route in the group
func (g *RouteGroup) HEAD(pattern string, handler httpInternal.HandlerFunc, middleware ...httpInternal.MiddlewareFunc) {
	fullPattern := g.prefix + pattern
	allMiddleware := append(g.middleware, middleware...)
	g.router.HEAD(fullPattern, handler, allMiddleware...)
}

// ANY registers a route for all HTTP methods in the group
func (g *RouteGroup) ANY(pattern string, handler httpInternal.HandlerFunc, middleware ...httpInternal.MiddlewareFunc) {
	fullPattern := g.prefix + pattern
	allMiddleware := append(g.middleware, middleware...)
	g.router.ANY(fullPattern, handler, allMiddleware...)
}

// Group creates a nested route group with additional prefix and middleware
func (g *RouteGroup) Group(prefix string, middleware ...httpInternal.MiddlewareFunc) httpInternal.RouteGroup {
	fullPrefix := g.prefix + strings.TrimSuffix(prefix, "/")
	allMiddleware := append(g.middleware, middleware...)
	
	return &RouteGroup{
		router:     g.router,
		prefix:     fullPrefix,
		middleware: allMiddleware,
	}
}

// Use adds middleware to the route group
func (g *RouteGroup) Use(middleware ...httpInternal.MiddlewareFunc) {
	g.middleware = append(g.middleware, middleware...)
}

// Prefix returns the current prefix of the route group
func (g *RouteGroup) Prefix() string {
	return g.prefix
}

// Middleware returns the middleware stack of the route group
func (g *RouteGroup) Middleware() []httpInternal.MiddlewareFunc {
	return g.middleware
}

// Resource creates RESTful routes for a resource
func (g *RouteGroup) Resource(name string, controller ResourceController) {
	// Implement RESTful routing convention
	g.GET("/"+name, controller.Index)           // GET /resource
	g.GET("/"+name+"/{id}", controller.Show)    // GET /resource/{id}
	g.POST("/"+name, controller.Store)          // POST /resource
	g.PUT("/"+name+"/{id}", controller.Update)  // PUT /resource/{id}
	g.DELETE("/"+name+"/{id}", controller.Destroy) // DELETE /resource/{id}
}

// ResourceController interface for RESTful controllers
type ResourceController interface {
	Index(httpInternal.Context) error   // List all resources
	Show(httpInternal.Context) error    // Show a specific resource
	Store(httpInternal.Context) error   // Create a new resource
	Update(httpInternal.Context) error  // Update an existing resource
	Destroy(httpInternal.Context) error // Delete a resource
}