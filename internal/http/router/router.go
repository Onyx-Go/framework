package router

import (
	"net/http"
	"regexp"
	"strings"

	httpInternal "github.com/onyx-go/framework/internal/http"
	"github.com/onyx-go/framework/internal/http/context"
)

// route represents a single HTTP route
type route struct {
	method      string
	pattern     string
	handler     httpInternal.HandlerFunc
	regex       *regexp.Regexp
	paramNames  []string
	middleware  []httpInternal.MiddlewareFunc
}

// Router implements the HTTP router with pattern matching and middleware support
type Router struct {
	routes     []*route
	middleware []httpInternal.MiddlewareFunc
	notFound   httpInternal.HandlerFunc
	app        httpInternal.Application
}

// NewRouter creates a new HTTP router
func NewRouter() *Router {
	return &Router{
		routes:     make([]*route, 0),
		middleware: make([]httpInternal.MiddlewareFunc, 0),
		notFound: func(c httpInternal.Context) error {
			return c.String(404, "Not Found")
		},
	}
}

// SetApplication sets the application instance for the router
func (r *Router) SetApplication(app httpInternal.Application) {
	r.app = app
}

// Use adds middleware to the router
func (r *Router) Use(middleware ...httpInternal.MiddlewareFunc) {
	r.middleware = append(r.middleware, middleware...)
}

// addRoute adds a route with the specified method, pattern, and handler
func (r *Router) addRoute(method, pattern string, handler httpInternal.HandlerFunc, middleware ...httpInternal.MiddlewareFunc) {
	route := &route{
		method:     strings.ToUpper(method),
		pattern:    pattern,
		handler:    handler,
		middleware: middleware,
	}
	
	route.regex, route.paramNames = r.compilePattern(pattern)
	r.routes = append(r.routes, route)
}

// GET registers a GET route
func (r *Router) GET(pattern string, handler httpInternal.HandlerFunc, middleware ...httpInternal.MiddlewareFunc) {
	r.addRoute("GET", pattern, handler, middleware...)
}

// POST registers a POST route
func (r *Router) POST(pattern string, handler httpInternal.HandlerFunc, middleware ...httpInternal.MiddlewareFunc) {
	r.addRoute("POST", pattern, handler, middleware...)
}

// PUT registers a PUT route
func (r *Router) PUT(pattern string, handler httpInternal.HandlerFunc, middleware ...httpInternal.MiddlewareFunc) {
	r.addRoute("PUT", pattern, handler, middleware...)
}

// DELETE registers a DELETE route
func (r *Router) DELETE(pattern string, handler httpInternal.HandlerFunc, middleware ...httpInternal.MiddlewareFunc) {
	r.addRoute("DELETE", pattern, handler, middleware...)
}

// PATCH registers a PATCH route
func (r *Router) PATCH(pattern string, handler httpInternal.HandlerFunc, middleware ...httpInternal.MiddlewareFunc) {
	r.addRoute("PATCH", pattern, handler, middleware...)
}

// OPTIONS registers an OPTIONS route
func (r *Router) OPTIONS(pattern string, handler httpInternal.HandlerFunc, middleware ...httpInternal.MiddlewareFunc) {
	r.addRoute("OPTIONS", pattern, handler, middleware...)
}

// HEAD registers a HEAD route
func (r *Router) HEAD(pattern string, handler httpInternal.HandlerFunc, middleware ...httpInternal.MiddlewareFunc) {
	r.addRoute("HEAD", pattern, handler, middleware...)
}

// ANY registers a route for all HTTP methods
func (r *Router) ANY(pattern string, handler httpInternal.HandlerFunc, middleware ...httpInternal.MiddlewareFunc) {
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"}
	for _, method := range methods {
		r.addRoute(method, pattern, handler, middleware...)
	}
}

// Group creates a new route group with the specified prefix and middleware
func (r *Router) Group(prefix string, middleware ...httpInternal.MiddlewareFunc) httpInternal.RouteGroup {
	return &RouteGroup{
		router:     r,
		Prefix_:    strings.TrimSuffix(prefix, "/"),
		middleware: middleware,
	}
}

// SetNotFound sets the handler for 404 Not Found responses
func (r *Router) SetNotFound(handler httpInternal.HandlerFunc) {
	r.notFound = handler
}

// compilePattern compiles a route pattern into a regex and extracts parameter names
func (r *Router) compilePattern(pattern string) (*regexp.Regexp, []string) {
	var paramNames []string
	
	regexPattern := "^"
	parts := strings.Split(pattern, "/")
	
	for _, part := range parts {
		if part == "" {
			continue
		}
		
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			paramName := part[1 : len(part)-1]
			
			if strings.Contains(paramName, ":") {
				parts := strings.Split(paramName, ":")
				paramName = parts[0]
				constraint := parts[1]
				
				switch constraint {
				case "int", "number":
					regexPattern += "/([0-9]+)"
				case "alpha":
					regexPattern += "/([a-zA-Z]+)"
				case "alphanum":
					regexPattern += "/([a-zA-Z0-9]+)"
				default:
					regexPattern += "/([^/]+)"
				}
			} else {
				regexPattern += "/([^/]+)"
			}
			
			paramNames = append(paramNames, paramName)
		} else {
			regexPattern += "/" + regexp.QuoteMeta(part)
		}
	}
	
	regexPattern += "$"
	regex := regexp.MustCompile(regexPattern)
	
	return regex, paramNames
}

// match finds a matching route for the given method and path
func (r *Router) match(method, path string) (*route, map[string]string) {
	for _, route := range r.routes {
		if route.method != method {
			continue
		}
		
		matches := route.regex.FindStringSubmatch(path)
		if matches == nil {
			continue
		}
		
		params := make(map[string]string)
		for i, name := range route.paramNames {
			if i+1 < len(matches) {
				params[name] = matches[i+1]
			}
		}
		
		return route, params
	}
	
	return nil, nil
}

// ServeHTTP implements the http.Handler interface
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	method := req.Method
	
	route, params := r.match(method, path)
	
	// Create context from the context package
	ctx := context.NewContext(w, req, r.app)
	
	// Set route parameters
	for key, value := range params {
		ctx.SetParam(key, value)
	}
	
	if route == nil {
		// No matching route found
		ctx.AddMiddleware(r.middleware...)
		ctx.AddMiddleware(func(c httpInternal.Context) error {
			return r.notFound(c)
		})
	} else {
		// Route found, add middleware and handler
		ctx.AddMiddleware(r.middleware...)
		ctx.AddMiddleware(route.middleware...)
		ctx.AddMiddleware(func(c httpInternal.Context) error {
			return route.handler(c)
		})
	}
	
	// Execute middleware chain
	if err := ctx.Next(); err != nil {
		// Handle error through application error handler
		if !ctx.IsAborted() && r.app != nil {
			r.app.ErrorHandler().Handle(ctx, err)
		}
	}
}

// GetRoutes returns all registered routes (for debugging/introspection)
func (r *Router) GetRoutes() []RouteInfo {
	routes := make([]RouteInfo, len(r.routes))
	for i, route := range r.routes {
		routes[i] = RouteInfo{
			Method:     route.method,
			Pattern:    route.pattern,
			ParamNames: route.paramNames,
		}
	}
	return routes
}

// RouteInfo provides information about a registered route
type RouteInfo struct {
	Method     string
	Pattern    string
	ParamNames []string
}