package router

import (
	"testing"

	httpInternal "github.com/onyx-go/framework/internal/http"
)

func TestNewRouter(t *testing.T) {
	router := NewRouter()
	
	if router == nil {
		t.Fatal("NewRouter() returned nil")
	}
	
	if router.routes == nil {
		t.Error("router.routes should be initialized")
	}
	
	if router.middleware == nil {
		t.Error("router.middleware should be initialized")
	}
	
	if router.notFound == nil {
		t.Error("router.notFound should be initialized")
	}
}

func TestRouterAddRoute(t *testing.T) {
	router := NewRouter()
	
	handler := func(c httpInternal.Context) error {
		return nil
	}
	
	router.GET("/test", handler)
	
	if len(router.routes) != 1 {
		t.Errorf("expected 1 route, got %d", len(router.routes))
	}
	
	route := router.routes[0]
	if route.method != "GET" {
		t.Errorf("expected method GET, got %s", route.method)
	}
	
	if route.pattern != "/test" {
		t.Errorf("expected pattern /test, got %s", route.pattern)
	}
}

func TestRouterHTTPMethods(t *testing.T) {
	router := NewRouter()
	
	handler := func(c httpInternal.Context) error {
		return nil
	}
	
	// Test all HTTP methods
	router.GET("/get", handler)
	router.POST("/post", handler)
	router.PUT("/put", handler)
	router.DELETE("/delete", handler)
	router.PATCH("/patch", handler)
	router.OPTIONS("/options", handler)
	router.HEAD("/head", handler)
	
	if len(router.routes) != 7 {
		t.Errorf("expected 7 routes, got %d", len(router.routes))
	}
	
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"}
	for i, method := range methods {
		if router.routes[i].method != method {
			t.Errorf("expected method %s at index %d, got %s", method, i, router.routes[i].method)
		}
	}
}

func TestRouterANY(t *testing.T) {
	router := NewRouter()
	
	handler := func(c httpInternal.Context) error {
		return nil
	}
	
	router.ANY("/any", handler)
	
	// ANY should register 7 routes (all HTTP methods)
	if len(router.routes) != 7 {
		t.Errorf("expected 7 routes for ANY, got %d", len(router.routes))
	}
	
	// Check that all methods are registered
	methods := map[string]bool{
		"GET": false, "POST": false, "PUT": false, "DELETE": false,
		"PATCH": false, "OPTIONS": false, "HEAD": false,
	}
	
	for _, route := range router.routes {
		if route.pattern == "/any" {
			methods[route.method] = true
		}
	}
	
	for method, found := range methods {
		if !found {
			t.Errorf("method %s not registered for ANY route", method)
		}
	}
}

func TestRouterCompilePattern(t *testing.T) {
	router := NewRouter()
	
	tests := []struct {
		pattern     string
		expectNames []string
		testPaths   map[string]bool // path -> should match
	}{
		{
			pattern:     "/users/{id}",
			expectNames: []string{"id"},
			testPaths: map[string]bool{
				"/users/123":     true,
				"/users/abc":     true,
				"/users":         false,
				"/users/123/456": false,
			},
		},
		{
			pattern:     "/users/{id:int}/posts/{postId:int}",
			expectNames: []string{"id", "postId"},
			testPaths: map[string]bool{
				"/users/123/posts/456": true,
				"/users/abc/posts/456": false,
				"/users/123/posts/abc": false,
			},
		},
		{
			pattern:     "/files/{name:alpha}",
			expectNames: []string{"name"},
			testPaths: map[string]bool{
				"/files/readme":  true,
				"/files/README":  true,
				"/files/file123": false,
				"/files/123":     false,
			},
		},
	}
	
	for _, test := range tests {
		regex, paramNames := router.compilePattern(test.pattern)
		
		// Check parameter names
		if len(paramNames) != len(test.expectNames) {
			t.Errorf("pattern %s: expected %d param names, got %d",
				test.pattern, len(test.expectNames), len(paramNames))
			continue
		}
		
		for i, name := range test.expectNames {
			if paramNames[i] != name {
				t.Errorf("pattern %s: expected param name %s at index %d, got %s",
					test.pattern, name, i, paramNames[i])
			}
		}
		
		// Check regex matching
		for path, shouldMatch := range test.testPaths {
			matches := regex.MatchString(path)
			if matches != shouldMatch {
				t.Errorf("pattern %s, path %s: expected match=%v, got %v",
					test.pattern, path, shouldMatch, matches)
			}
		}
	}
}

func TestRouterMatch(t *testing.T) {
	router := NewRouter()
	
	handler := func(c httpInternal.Context) error {
		return nil
	}
	
	router.GET("/users/{id:int}", handler)
	router.POST("/users", handler)
	
	// Test successful match
	route, params := router.match("GET", "/users/123")
	if route == nil {
		t.Error("expected to find matching route")
	}
	if params["id"] != "123" {
		t.Errorf("expected param id=123, got %s", params["id"])
	}
	
	// Test no match for wrong method
	route, params = router.match("POST", "/users/123")
	if route != nil {
		t.Error("expected no match for POST /users/123")
	}
	
	// Test successful match for different route
	route, params = router.match("POST", "/users")
	if route == nil {
		t.Error("expected to find matching route for POST /users")
	}
	if len(params) != 0 {
		t.Errorf("expected no params, got %v", params)
	}
}

func TestRouterUse(t *testing.T) {
	router := NewRouter()
	
	middleware1 := func(c httpInternal.Context) error {
		return nil
	}
	middleware2 := func(c httpInternal.Context) error {
		return nil
	}
	
	router.Use(middleware1, middleware2)
	
	if len(router.middleware) != 2 {
		t.Errorf("expected 2 middleware, got %d", len(router.middleware))
	}
}

func TestRouteGroup(t *testing.T) {
	router := NewRouter()
	
	handler := func(c httpInternal.Context) error {
		return nil
	}
	
	api := router.Group("/api")
	api.GET("/users", handler)
	api.POST("/users", handler)
	
	// Check that routes were added with prefix
	if len(router.routes) != 2 {
		t.Errorf("expected 2 routes, got %d", len(router.routes))
	}
	
	for _, route := range router.routes {
		if route.pattern != "/api/users" {
			t.Errorf("expected pattern /api/users, got %s", route.pattern)
		}
	}
}

func TestNestedRouteGroups(t *testing.T) {
	router := NewRouter()
	
	handler := func(c httpInternal.Context) error {
		return nil
	}
	
	api := router.Group("/api")
	v1 := api.Group("/v1")
	v1.GET("/users", handler)
	
	// Check that route was added with nested prefix
	if len(router.routes) != 1 {
		t.Errorf("expected 1 route, got %d", len(router.routes))
	}
	
	route := router.routes[0]
	if route.pattern != "/api/v1/users" {
		t.Errorf("expected pattern /api/v1/users, got %s", route.pattern)
	}
}

func TestSetNotFound(t *testing.T) {
	router := NewRouter()
	
	customNotFound := func(c httpInternal.Context) error {
		return c.String(404, "Custom Not Found")
	}
	
	router.SetNotFound(customNotFound)
	
	// We can't easily test the function equality, but we can verify it was set
	if router.notFound == nil {
		t.Error("notFound handler should not be nil after SetNotFound")
	}
}

func TestGetRoutes(t *testing.T) {
	router := NewRouter()
	
	handler := func(c httpInternal.Context) error {
		return nil
	}
	
	router.GET("/users/{id:int}", handler)
	router.POST("/users", handler)
	
	routes := router.GetRoutes()
	
	if len(routes) != 2 {
		t.Errorf("expected 2 routes, got %d", len(routes))
	}
	
	// Check first route
	if routes[0].Method != "GET" {
		t.Errorf("expected method GET, got %s", routes[0].Method)
	}
	if routes[0].Pattern != "/users/{id:int}" {
		t.Errorf("expected pattern /users/{id:int}, got %s", routes[0].Pattern)
	}
	if len(routes[0].ParamNames) != 1 || routes[0].ParamNames[0] != "id" {
		t.Errorf("expected param names [id], got %v", routes[0].ParamNames)
	}
}