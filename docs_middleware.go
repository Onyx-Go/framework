package onyx

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"
)

// DocsMiddleware provides automatic documentation generation middleware
type DocsMiddleware struct {
	enabled        bool
	docBuilder     *APIDocumentationBuilder
	versionManager *APIVersionManager
	versionedDocs  *VersionedDocumentation
	cache          *DocsCache
	config         *DocsMiddlewareConfig
	discovery      *RouteDiscovery
	mu             sync.RWMutex
}

// DocsMiddlewareConfig configuration for documentation middleware
type DocsMiddlewareConfig struct {
	Enabled              bool          `json:"enabled"`
	AutoDiscovery        bool          `json:"auto_discovery"`
	CacheEnabled         bool          `json:"cache_enabled"`
	CacheDuration        time.Duration `json:"cache_duration"`
	GenerateOnStartup    bool          `json:"generate_on_startup"`
	IncludePrivateRoutes bool          `json:"include_private_routes"`
	OutputPath           string        `json:"output_path"`
	ServeSwaggerUI       bool          `json:"serve_swagger_ui"`
	SwaggerUIPath        string        `json:"swagger_ui_path"`
	AnnotationSources    []string      `json:"annotation_sources"`
	DefaultTags          []string      `json:"default_tags"`
	ExcludePatterns      []string      `json:"exclude_patterns"`
	IncludePatterns      []string      `json:"include_patterns"`
}

// DocsCache handles caching of generated documentation
type DocsCache struct {
	specs       map[string]*CachedSpec
	lastUpdate  time.Time
	duration    time.Duration
	mu          sync.RWMutex
}

// CachedSpec represents a cached OpenAPI specification
type CachedSpec struct {
	Spec      *OpenAPISpec `json:"spec"`
	Version   string       `json:"version"`
	Generated time.Time    `json:"generated"`
	Expires   time.Time    `json:"expires"`
}

// RouteDiscovery handles automatic route discovery and documentation
type RouteDiscovery struct {
	router        *Router
	handlers      map[string]*HandlerInfo
	annotations   map[string]*RouteDocumentation
	sourceCache   map[string]string
	mu            sync.RWMutex
}

// HandlerInfo contains information about a route handler
type HandlerInfo struct {
	Function     reflect.Value
	Name         string
	File         string
	Line         int
	Documentation string
	Parameters   []ParameterInfo
	ReturnType   reflect.Type
}

// ParameterInfo contains information about handler parameters
type ParameterInfo struct {
	Name string
	Type reflect.Type
	Tag  string
}

// NewDocsMiddleware creates a new documentation middleware
func NewDocsMiddleware(config *DocsMiddlewareConfig) *DocsMiddleware {
	if config == nil {
		config = &DocsMiddlewareConfig{
			Enabled:              true,
			AutoDiscovery:        true,
			CacheEnabled:         true,
			CacheDuration:        5 * time.Minute,
			GenerateOnStartup:    true,
			IncludePrivateRoutes: false,
			OutputPath:           "docs/api",
			ServeSwaggerUI:       true,
			SwaggerUIPath:        "/docs",
			AnnotationSources:    []string{"*.go"},
		}
	}

	cache := &DocsCache{
		specs:    make(map[string]*CachedSpec),
		duration: config.CacheDuration,
	}

	discovery := &RouteDiscovery{
		handlers:    make(map[string]*HandlerInfo),
		annotations: make(map[string]*RouteDocumentation),
		sourceCache: make(map[string]string),
	}

	return &DocsMiddleware{
		enabled:   config.Enabled,
		config:    config,
		cache:     cache,
		discovery: discovery,
	}
}

// SetDocumentationBuilder sets the documentation builder
func (dm *DocsMiddleware) SetDocumentationBuilder(builder *APIDocumentationBuilder) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	dm.docBuilder = builder
}

// SetVersionManager sets the version manager
func (dm *DocsMiddleware) SetVersionManager(manager *APIVersionManager) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	dm.versionManager = manager
}

// SetVersionedDocumentation sets the versioned documentation manager
func (dm *DocsMiddleware) SetVersionedDocumentation(versionedDocs *VersionedDocumentation) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	dm.versionedDocs = versionedDocs
}

// Middleware returns the middleware function
func (dm *DocsMiddleware) Middleware() MiddlewareFunc {
	return func(c Context) error {
		if !dm.enabled {
			return c.Next()
		}

		// Handle documentation endpoints
		if dm.handleDocsEndpoints(c) {
			return nil
		}

		// Record route for auto-discovery
		if dm.config.AutoDiscovery {
			dm.recordRoute(c)
		}

		return c.Next()
	}
}

// handleDocsEndpoints handles documentation-specific endpoints
func (dm *DocsMiddleware) handleDocsEndpoints(c *Context) bool {
	path := c.Request.URL.Path

	// Serve OpenAPI JSON
	if path == "/docs/openapi.json" || path == "/docs/swagger.json" {
		return dm.serveOpenAPIJSON(c)
	}

	// Serve specific version OpenAPI JSON
	if strings.HasPrefix(path, "/docs/") && strings.HasSuffix(path, "/openapi.json") {
		version := strings.TrimSuffix(strings.TrimPrefix(path, "/docs/"), "/openapi.json")
		return dm.serveVersionedOpenAPIJSON(c, version)
	}

	// Serve Swagger UI
	if dm.config.ServeSwaggerUI && strings.HasPrefix(path, dm.config.SwaggerUIPath) {
		return dm.serveSwaggerUI(c)
	}

	// API documentation endpoints
	if strings.HasPrefix(path, "/docs/api/") {
		return dm.handleAPIDocsEndpoints(c)
	}

	return false
}

// serveOpenAPIJSON serves the OpenAPI JSON specification
func (dm *DocsMiddleware) serveOpenAPIJSON(c *Context) bool {
	// Get cached spec if available
	if dm.config.CacheEnabled {
		if spec := dm.cache.GetCachedSpec("default"); spec != nil {
			c.Header("Content-Type", "application/json")
			c.Header("Cache-Control", fmt.Sprintf("max-age=%d", int(dm.config.CacheDuration.Seconds())))
			json.NewEncoder(c.ResponseWriter).Encode(spec.Spec)
			return true
		}
	}

	// Generate spec
	var spec *OpenAPISpec
	var err error

	if dm.versionedDocs != nil {
		versionedSpec, verr := dm.versionedDocs.GetDefaultSpec()
		if verr != nil {
			c.String(500, "Failed to generate documentation: %v", verr)
			return true
		}
		spec = versionedSpec.Spec
	} else if dm.docBuilder != nil {
		spec, err = dm.docBuilder.GenerateOpenAPISpec()
		if err != nil {
			c.String(500, "Failed to generate documentation: %v", err)
			return true
		}
	} else {
		c.String(500, "Documentation builder not configured")
		return true
	}

	// Cache the spec
	if dm.config.CacheEnabled {
		dm.cache.CacheSpec("default", spec, "default")
	}

	c.Header("Content-Type", "application/json")
	c.Header("Cache-Control", fmt.Sprintf("max-age=%d", int(dm.config.CacheDuration.Seconds())))
	json.NewEncoder(c.ResponseWriter).Encode(spec)
	return true
}

// serveVersionedOpenAPIJSON serves versioned OpenAPI JSON
func (dm *DocsMiddleware) serveVersionedOpenAPIJSON(c *Context, version string) bool {
	if dm.versionedDocs == nil {
		c.String(404, "Versioned documentation not available")
		return true
	}

	// Get cached spec if available
	cacheKey := fmt.Sprintf("version_%s", version)
	if dm.config.CacheEnabled {
		if spec := dm.cache.GetCachedSpec(cacheKey); spec != nil {
			c.Header("Content-Type", "application/json")
			c.Header("Cache-Control", fmt.Sprintf("max-age=%d", int(dm.config.CacheDuration.Seconds())))
			json.NewEncoder(c.ResponseWriter).Encode(spec.Spec)
			return true
		}
	}

	// Generate versioned spec
	builder := dm.versionedDocs.GetBuilder(version)
	if builder == nil {
		c.String(404, "Version not found: %s", version)
		return true
	}

	spec, err := builder.GenerateOpenAPISpec()
	if err != nil {
		c.String(500, "Failed to generate documentation for version %s: %v", version, err)
		return true
	}

	// Cache the spec
	if dm.config.CacheEnabled {
		dm.cache.CacheSpec(cacheKey, spec, version)
	}

	c.Header("Content-Type", "application/json")
	c.Header("Cache-Control", fmt.Sprintf("max-age=%d", int(dm.config.CacheDuration.Seconds())))
	json.NewEncoder(c.ResponseWriter).Encode(spec)
	return true
}

// serveSwaggerUI serves the Swagger UI interface
func (dm *DocsMiddleware) serveSwaggerUI(c *Context) bool {
	path := c.Request.URL.Path
	
	// Serve index.html for the main path
	if path == dm.config.SwaggerUIPath || path == dm.config.SwaggerUIPath+"/" {
		html := dm.generateSwaggerUIHTML()
		c.Header("Content-Type", "text/html")
		c.ResponseWriter.WriteHeader(200)
		c.ResponseWriter.Write([]byte(html))
		return true
	}

	// For other Swagger UI assets, return a simple message
	c.String(404, "Swagger UI asset not found")
	return true
}

// generateSwaggerUIHTML generates Swagger UI HTML
func (dm *DocsMiddleware) generateSwaggerUIHTML() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>API Documentation - Onyx</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@3.52.5/swagger-ui.css" />
    <style>
        html { box-sizing: border-box; overflow: -moz-scrollbars-vertical; overflow-y: scroll; }
        *, *:before, *:after { box-sizing: inherit; }
        body { margin:0; background: #fafafa; }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@3.52.5/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist@3.52.5/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            const ui = SwaggerUIBundle({
                url: '/docs/openapi.json',
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout",
                tryItOutEnabled: true,
                supportedSubmitMethods: ['get', 'post', 'put', 'delete', 'patch'],
                onComplete: function() {
                    console.log('Swagger UI loaded successfully');
                },
                onFailure: function() {
                    console.error('Failed to load Swagger UI');
                }
            });
        };
    </script>
</body>
</html>`
}

// handleAPIDocsEndpoints handles API documentation endpoints
func (dm *DocsMiddleware) handleAPIDocsEndpoints(c *Context) bool {
	path := strings.TrimPrefix(c.Request.URL.Path, "/docs/api/")
	
	switch {
	case path == "versions":
		return dm.serveVersionsInfo(c)
	case path == "changelog":
		return dm.serveChangelog(c)
	case path == "matrix":
		return dm.serveVersionMatrix(c)
	case strings.HasPrefix(path, "compare/"):
		return dm.serveVersionComparison(c)
	default:
		c.String(404, "Documentation endpoint not found")
		return true
	}
}

// serveVersionsInfo serves version information
func (dm *DocsMiddleware) serveVersionsInfo(c *Context) bool {
	if dm.versionManager == nil {
		c.String(404, "Version management not available")
		return true
	}

	versions := dm.versionManager.GetAllVersions()
	c.Header("Content-Type", "application/json")
	json.NewEncoder(c.ResponseWriter).Encode(versions)
	return true
}

// serveChangelog serves the API changelog
func (dm *DocsMiddleware) serveChangelog(c *Context) bool {
	if dm.versionedDocs == nil {
		c.String(404, "Versioned documentation not available")
		return true
	}

	format := c.Query("format")
	if format == "markdown" {
		changelog, err := dm.versionedDocs.GenerateMarkdownChangelog()
		if err != nil {
			c.String(500, "Failed to generate changelog: %v", err)
			return true
		}
		c.Header("Content-Type", "text/markdown")
		c.ResponseWriter.Write([]byte(changelog))
	} else {
		changelog := dm.versionedDocs.GenerateChangelog()
		c.Header("Content-Type", "application/json")
		json.NewEncoder(c.ResponseWriter).Encode(changelog)
	}
	return true
}

// serveVersionMatrix serves the version compatibility matrix
func (dm *DocsMiddleware) serveVersionMatrix(c *Context) bool {
	if dm.versionedDocs == nil {
		c.String(404, "Versioned documentation not available")
		return true
	}

	matrix := dm.versionedDocs.GenerateVersionMatrix()
	c.Header("Content-Type", "application/json")
	json.NewEncoder(c.ResponseWriter).Encode(matrix)
	return true
}

// serveVersionComparison serves version comparison
func (dm *DocsMiddleware) serveVersionComparison(c *Context) bool {
	if dm.versionedDocs == nil {
		c.String(404, "Versioned documentation not available")
		return true
	}

	// Extract versions from path: /docs/api/compare/v1/v2
	pathParts := strings.Split(strings.TrimPrefix(c.Request.URL.Path, "/docs/api/compare/"), "/")
	if len(pathParts) != 2 {
		c.String(400, "Invalid comparison path. Use /docs/api/compare/version1/version2")
		return true
	}

	version1, version2 := pathParts[0], pathParts[1]
	comparison, err := dm.versionedDocs.CompareVersions(version1, version2)
	if err != nil {
		c.String(500, "Failed to compare versions: %v", err)
		return true
	}

	c.Header("Content-Type", "application/json")
	json.NewEncoder(c.ResponseWriter).Encode(comparison)
	return true
}

// recordRoute records a route for auto-discovery
func (dm *DocsMiddleware) recordRoute(c *Context) {
	if !dm.config.AutoDiscovery {
		return
	}

	// Skip if this is a docs endpoint
	if strings.HasPrefix(c.Request.URL.Path, "/docs") {
		return
	}

	// Extract route information
	method := c.Request.Method
	path := c.Request.URL.Path

	// Check exclude patterns
	for _, pattern := range dm.config.ExcludePatterns {
		if matched, _ := filepath.Match(pattern, path); matched {
			return
		}
	}

	// Check include patterns
	if len(dm.config.IncludePatterns) > 0 {
		included := false
		for _, pattern := range dm.config.IncludePatterns {
			if matched, _ := filepath.Match(pattern, path); matched {
				included = true
				break
			}
		}
		if !included {
			return
		}
	}

	// Record the route
	dm.discovery.RecordRoute(method, path, c)
}

// RecordRoute records a route in the discovery system
func (rd *RouteDiscovery) RecordRoute(method, path string, c *Context) {
	rd.mu.Lock()
	defer rd.mu.Unlock()

	routeKey := fmt.Sprintf("%s %s", method, path)
	
	// Extract handler information if possible
	// This would require access to the current handler, which might need
	// additional context passing in the middleware chain
	
	// For now, create basic documentation
	if _, exists := rd.annotations[routeKey]; !exists {
		rd.annotations[routeKey] = &RouteDocumentation{
			Summary:    fmt.Sprintf("%s %s", method, path),
			Parameters: make([]ParameterDoc, 0),
			Responses:  make(map[string]ResponseDoc),
		}
	}
}

// Cache implementation

// GetCachedSpec gets a cached specification
func (dc *DocsCache) GetCachedSpec(key string) *CachedSpec {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	spec, exists := dc.specs[key]
	if !exists {
		return nil
	}

	if time.Now().After(spec.Expires) {
		delete(dc.specs, key)
		return nil
	}

	return spec
}

// CacheSpec caches a specification
func (dc *DocsCache) CacheSpec(key string, spec *OpenAPISpec, version string) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	now := time.Now()
	dc.specs[key] = &CachedSpec{
		Spec:      spec,
		Version:   version,
		Generated: now,
		Expires:   now.Add(dc.duration),
	}
	dc.lastUpdate = now
}

// ClearCache clears all cached specifications
func (dc *DocsCache) ClearCache() {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	dc.specs = make(map[string]*CachedSpec)
	dc.lastUpdate = time.Now()
}

// GetCacheStats returns cache statistics
func (dc *DocsCache) GetCacheStats() map[string]interface{} {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	stats := map[string]interface{}{
		"total_specs":   len(dc.specs),
		"last_update":   dc.lastUpdate,
		"cache_duration": dc.duration,
	}

	// Count expired specs
	expired := 0
	now := time.Now()
	for _, spec := range dc.specs {
		if now.After(spec.Expires) {
			expired++
		}
	}
	stats["expired_specs"] = expired

	return stats
}

// Auto-discovery helpers

// DiscoverRoutes discovers routes from the router
func (dm *DocsMiddleware) DiscoverRoutes(router *Router) {
	dm.discovery.router = router
	dm.discovery.DiscoverFromRouter()
}

// DiscoverFromRouter discovers routes from the router
func (rd *RouteDiscovery) DiscoverFromRouter() {
	if rd.router == nil {
		return
	}

	rd.mu.Lock()
	defer rd.mu.Unlock()

	for _, route := range rd.router.routes {
		routeKey := fmt.Sprintf("%s %s", route.method, route.pattern)
		
		if _, exists := rd.annotations[routeKey]; !exists {
			// Create basic documentation from route
			doc := &RouteDocumentation{
				Summary:    fmt.Sprintf("%s %s", route.method, route.pattern),
				Parameters: make([]ParameterDoc, 0),
				Responses:  make(map[string]ResponseDoc),
			}

			// Extract path parameters
			for _, paramName := range route.paramNames {
				param := ParameterDoc{
					Name:        paramName,
					In:          "path",
					Type:        "string",
					Required:    true,
					Description: fmt.Sprintf("Path parameter: %s", paramName),
				}
				doc.Parameters = append(doc.Parameters, param)
			}

			// Add default responses
			doc.Responses["200"] = ResponseDoc{
				Description: "Success",
			}

			rd.annotations[routeKey] = doc
		}

		// Extract handler information
		if route.handler != nil {
			rd.extractHandlerInfo(routeKey, route.handler)
		}
	}
}

// extractHandlerInfo extracts information from a handler function
func (rd *RouteDiscovery) extractHandlerInfo(routeKey string, handler HandlerFunc) {
	handlerValue := reflect.ValueOf(handler)
	handlerType := handlerValue.Type()

	// Get function name
	pc := handlerValue.Pointer()
	fn := runtime.FuncForPC(pc)
	
	info := &HandlerInfo{
		Function: handlerValue,
		Name:     fn.Name(),
	}

	// Get file and line
	file, line := fn.FileLine(pc)
	info.File = file
	info.Line = line

	// Analyze function signature
	if handlerType.NumIn() > 0 {
		for i := 0; i < handlerType.NumIn(); i++ {
			paramType := handlerType.In(i)
			paramInfo := ParameterInfo{
				Name: fmt.Sprintf("param%d", i),
				Type: paramType,
			}
			info.Parameters = append(info.Parameters, paramInfo)
		}
	}

	if handlerType.NumOut() > 0 {
		info.ReturnType = handlerType.Out(0)
	}

	rd.handlers[routeKey] = info
}

// ApplyDiscoveredDocumentation applies discovered documentation to builder
func (dm *DocsMiddleware) ApplyDiscoveredDocumentation() {
	if dm.docBuilder == nil {
		return
	}

	dm.discovery.mu.RLock()
	defer dm.discovery.mu.RUnlock()

	for routeKey, doc := range dm.discovery.annotations {
		parts := strings.SplitN(routeKey, " ", 2)
		if len(parts) == 2 {
			dm.docBuilder.DocumentRoute(parts[0], parts[1], doc)
		}
	}
}

// GetDiscoveredRoutes returns discovered route documentation
func (dm *DocsMiddleware) GetDiscoveredRoutes() map[string]*RouteDocumentation {
	dm.discovery.mu.RLock()
	defer dm.discovery.mu.RUnlock()

	result := make(map[string]*RouteDocumentation)
	for k, v := range dm.discovery.annotations {
		result[k] = v
	}
	return result
}

// Middleware factory functions

// CreateDocumentationMiddleware creates documentation middleware with all components
func CreateDocumentationMiddleware(
	router *Router,
	docBuilder *APIDocumentationBuilder,
	versionManager *APIVersionManager,
	config *DocsMiddlewareConfig,
) *DocsMiddleware {
	middleware := NewDocsMiddleware(config)
	middleware.SetDocumentationBuilder(docBuilder)
	middleware.SetVersionManager(versionManager)
	
	if versionManager != nil {
		versionedDocs := NewVersionedDocumentation(versionManager, nil)
		middleware.SetVersionedDocumentation(versionedDocs)
	}

	// Discover routes if auto-discovery is enabled
	if config.AutoDiscovery {
		middleware.DiscoverRoutes(router)
		middleware.ApplyDiscoveredDocumentation()
	}

	return middleware
}