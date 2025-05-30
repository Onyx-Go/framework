package http

import (
	"net/http"
)

// HandlerFunc defines the signature for HTTP handlers
type HandlerFunc func(Context) error

// MiddlewareFunc defines the signature for middleware functions
type MiddlewareFunc func(Context) error

// Context interface defines the contract for HTTP request contexts
type Context interface {
	// Request methods
	Method() string
	URL() string
	Path() string
	Query(key string) string
	QueryDefault(key, defaultValue string) string
	Param(key string) string
	Header(key string) string
	RemoteIP() string
	UserAgent() string
	
	// Response methods
	Status(code int)
	SetHeader(key, value string)
	String(code int, text string) error
	JSON(code int, data interface{}) error
	HTML(code int, html string) error
	
	// Utility methods
	Next() error
	Abort()
	IsAborted() bool
	Set(key string, value interface{})
	Get(key string) (interface{}, bool)
	
	// Request body methods
	Bind(obj interface{}) error
	Body() ([]byte, error)
	PostForm(key string) string
	
	// Response writer access
	ResponseWriter() http.ResponseWriter
	Request() *http.Request
	
	// Security methods
	SetCSRFToken() error
	ValidateCSRF() bool
	
	// Cookie methods
	SetCookie(cookie *http.Cookie)
	Cookie(name string) (*http.Cookie, error)
}

// Router interface defines the contract for HTTP routers
type Router interface {
	// HTTP method handlers
	GET(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc)
	POST(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc)
	PUT(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc)
	DELETE(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc)
	PATCH(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc)
	OPTIONS(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc)
	HEAD(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc)
	ANY(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc)
	
	// Middleware
	Use(middleware ...MiddlewareFunc)
	
	// Route groups
	Group(prefix string, middleware ...MiddlewareFunc) RouteGroup
	
	// Configuration
	SetNotFound(handler HandlerFunc)
	
	// Route introspection
	GetRoutes() []Route
	
	// HTTP server integration
	http.Handler
}

// RouteGroup interface defines the contract for grouped routes
type RouteGroup interface {
	GET(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc)
	POST(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc)
	PUT(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc)
	DELETE(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc)
	PATCH(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc)
	
	// Nested groups
	Group(prefix string, middleware ...MiddlewareFunc) RouteGroup
	
	// Middleware
	Use(middleware ...MiddlewareFunc)
}

// Route represents a single HTTP route
type Route interface {
	Method() string
	Pattern() string
	Handler() HandlerFunc
	Middleware() []MiddlewareFunc
	ParamNames() []string
}

// Application interface for accessing application services
type Application interface {
	// Error handling
	ErrorHandler() ErrorHandler
	
	// Template engine
	TemplateEngine() TemplateEngine
}

// ErrorHandler interface for handling HTTP errors
type ErrorHandler interface {
	Handle(Context, error)
}

// TemplateEngine interface for template rendering
type TemplateEngine interface {
	Render(string, interface{}) (string, error)
}