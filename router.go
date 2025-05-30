package onyx

import (
	"net/http"
	"regexp"
	"strings"
)

type HandlerFunc func(*Context) error
type MiddlewareFunc func(*Context) error

type Route struct {
	method      string
	pattern     string
	handler     HandlerFunc
	regex       *regexp.Regexp
	paramNames  []string
	middleware  []MiddlewareFunc
}

type Router struct {
	routes     []*Route
	middleware []MiddlewareFunc
	notFound   HandlerFunc
	app        *Application
}

func NewRouter() *Router {
	return &Router{
		routes: make([]*Route, 0),
		notFound: func(c *Context) error {
			return c.String(404, "Not Found")
		},
	}
}

func (r *Router) Use(middleware ...MiddlewareFunc) {
	r.middleware = append(r.middleware, middleware...)
}

func (r *Router) addRoute(method, pattern string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	route := &Route{
		method:     strings.ToUpper(method),
		pattern:    pattern,
		handler:    handler,
		middleware: middleware,
	}
	
	route.regex, route.paramNames = r.compilePattern(pattern)
	r.routes = append(r.routes, route)
}

func (r *Router) Get(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	r.addRoute("GET", pattern, handler, middleware...)
}

func (r *Router) Post(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	r.addRoute("POST", pattern, handler, middleware...)
}

func (r *Router) Put(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	r.addRoute("PUT", pattern, handler, middleware...)
}

func (r *Router) Delete(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	r.addRoute("DELETE", pattern, handler, middleware...)
}

func (r *Router) Patch(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	r.addRoute("PATCH", pattern, handler, middleware...)
}

func (r *Router) Options(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	r.addRoute("OPTIONS", pattern, handler, middleware...)
}

func (r *Router) Head(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	r.addRoute("HEAD", pattern, handler, middleware...)
}

func (r *Router) Any(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"}
	for _, method := range methods {
		r.addRoute(method, pattern, handler, middleware...)
	}
}

type RouteGroup struct {
	router     *Router
	prefix     string
	middleware []MiddlewareFunc
}

func (r *Router) Group(prefix string, middleware ...MiddlewareFunc) *RouteGroup {
	return &RouteGroup{
		router:     r,
		prefix:     strings.TrimSuffix(prefix, "/"),
		middleware: middleware,
	}
}

func (g *RouteGroup) Get(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	fullPattern := g.prefix + pattern
	allMiddleware := append(g.middleware, middleware...)
	g.router.Get(fullPattern, handler, allMiddleware...)
}

func (g *RouteGroup) Post(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	fullPattern := g.prefix + pattern
	allMiddleware := append(g.middleware, middleware...)
	g.router.Post(fullPattern, handler, allMiddleware...)
}

func (g *RouteGroup) Put(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	fullPattern := g.prefix + pattern
	allMiddleware := append(g.middleware, middleware...)
	g.router.Put(fullPattern, handler, allMiddleware...)
}

func (g *RouteGroup) Delete(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	fullPattern := g.prefix + pattern
	allMiddleware := append(g.middleware, middleware...)
	g.router.Delete(fullPattern, handler, allMiddleware...)
}

func (g *RouteGroup) Patch(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	fullPattern := g.prefix + pattern
	allMiddleware := append(g.middleware, middleware...)
	g.router.Patch(fullPattern, handler, allMiddleware...)
}

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

func (r *Router) match(method, path string) (*Route, map[string]string) {
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

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	method := req.Method
	
	route, params := r.match(method, path)
	
	ctx := NewContext(w, req, r.app)
	
	for key, value := range params {
		ctx.params[key] = value
	}
	
	if route == nil {
		ctx.middleware = append(ctx.middleware, r.middleware...)
		ctx.middleware = append(ctx.middleware, func(c *Context) error {
			return NotFound("Page not found")
		})
	} else {
		ctx.middleware = append(ctx.middleware, r.middleware...)
		ctx.middleware = append(ctx.middleware, route.middleware...)
		ctx.middleware = append(ctx.middleware, func(c *Context) error {
			return route.handler(c)
		})
	}
	
	if err := ctx.Next(); err != nil {
		// Let the error handler middleware handle it
		if !ctx.IsAborted() {
			GetErrorHandler().Handle(ctx, err)
		}
	}
}

func (r *Router) SetNotFound(handler HandlerFunc) {
	r.notFound = handler
}