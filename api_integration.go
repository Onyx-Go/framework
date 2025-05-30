package onyx

import (
	"fmt"
	"strings"
	"time"
)

// APIDocumentationManager manages the complete API documentation system
type APIDocumentationManager struct {
	app              *Application
	docBuilder       *APIDocumentationBuilder
	versionManager   *APIVersionManager
	versionedDocs    *VersionedDocumentation
	middleware       *DocsMiddleware
	config           *APIDocConfig
	middlewareConfig *DocsMiddlewareConfig
}

// NewAPIDocumentationManager creates a new API documentation manager
func NewAPIDocumentationManager(app *Application) *APIDocumentationManager {
	return &APIDocumentationManager{
		app: app,
		config: &APIDocConfig{
			Title:   "API Documentation",
			Version: "1.0.0",
		},
		middlewareConfig: &DocsMiddlewareConfig{
			Enabled:           true,
			AutoDiscovery:     true,
			CacheEnabled:      true,
			CacheDuration:     5 * time.Minute,
			GenerateOnStartup: true,
			ServeSwaggerUI:    true,
			SwaggerUIPath:     "/docs",
		},
	}
}

// Configure configures the API documentation
func (adm *APIDocumentationManager) Configure(config *APIDocConfig) *APIDocumentationManager {
	adm.config = config
	return adm
}

// ConfigureMiddleware configures the documentation middleware
func (adm *APIDocumentationManager) ConfigureMiddleware(config *DocsMiddlewareConfig) *APIDocumentationManager {
	adm.middlewareConfig = config
	return adm
}

// EnableVersioning enables API versioning
func (adm *APIDocumentationManager) EnableVersioning() *APIVersionManager {
	if adm.versionManager == nil {
		adm.versionManager = NewAPIVersionManager()
	}
	return adm.versionManager
}

// AddVersion adds a new API version
func (adm *APIDocumentationManager) AddVersion(version, name, description string) *APIDocumentationManager {
	if adm.versionManager == nil {
		adm.EnableVersioning()
	}
	
	apiVersion := CreateAPIVersionConfig(version, name, description)
	adm.versionManager.RegisterVersion(apiVersion)
	
	return adm
}

// AddDeprecatedVersion adds a deprecated API version
func (adm *APIDocumentationManager) AddDeprecatedVersion(version, name, description string, eolDate time.Time) *APIDocumentationManager {
	if adm.versionManager == nil {
		adm.EnableVersioning()
	}
	
	apiVersion := CreateDeprecatedAPIVersionConfig(version, name, description, eolDate)
	adm.versionManager.RegisterVersion(apiVersion)
	
	return adm
}

// Initialize initializes the API documentation system
func (adm *APIDocumentationManager) Initialize() error {
	// Create documentation builder
	adm.docBuilder = NewAPIDocumentationBuilder(adm.config)
	
	// Set up versioned documentation if versioning is enabled
	if adm.versionManager != nil {
		adm.versionedDocs = NewVersionedDocumentation(adm.versionManager, nil)
		
		// Create builders for each version
		for version := range adm.versionManager.GetAllVersions() {
			adm.versionedDocs.GetBuilder(version)
		}
	}
	
	// Create and configure middleware
	adm.middleware = CreateDocumentationMiddleware(
		adm.app.Router,
		adm.docBuilder,
		adm.versionManager,
		adm.middlewareConfig,
	)
	
	// Add middleware to application
	adm.app.Use(adm.middleware.Middleware())
	
	// Add version middleware if versioning is enabled
	if adm.versionManager != nil {
		adm.app.Use(adm.versionManager.CreateVersionMiddleware())
	}
	
	return nil
}

// GetDocumentationBuilder returns the documentation builder
func (adm *APIDocumentationManager) GetDocumentationBuilder() *APIDocumentationBuilder {
	return adm.docBuilder
}

// GetVersionManager returns the version manager
func (adm *APIDocumentationManager) GetVersionManager() *APIVersionManager {
	return adm.versionManager
}

// GetVersionedDocumentation returns the versioned documentation manager
func (adm *APIDocumentationManager) GetVersionedDocumentation() *VersionedDocumentation {
	return adm.versionedDocs
}

// GetMiddleware returns the documentation middleware
func (adm *APIDocumentationManager) GetMiddleware() *DocsMiddleware {
	return adm.middleware
}

// DocumentRoute adds documentation for a specific route
func (adm *APIDocumentationManager) DocumentRoute(method, pattern string, doc *RouteDocumentation) *APIDocumentationManager {
	if adm.docBuilder != nil {
		adm.docBuilder.DocumentRoute(method, pattern, doc)
	}
	return adm
}

// DocumentVersionedRoute adds documentation for a versioned route
func (adm *APIDocumentationManager) DocumentVersionedRoute(version, method, pattern string, doc *RouteDocumentation) *APIDocumentationManager {
	if adm.versionedDocs != nil {
		builder := adm.versionedDocs.GetBuilder(version)
		if builder != nil {
			builder.DocumentRoute(method, pattern, doc)
		}
	}
	return adm
}

// AddSchema adds a schema to the documentation
func (adm *APIDocumentationManager) AddSchema(name string, schema *OpenAPISchema) *APIDocumentationManager {
	if adm.docBuilder != nil {
		adm.docBuilder.AddSchema(name, schema)
	}
	return adm
}

// AddSchemaFromStruct adds a schema from a Go struct
func (adm *APIDocumentationManager) AddSchemaFromStruct(name string, v interface{}) *APIDocumentationManager {
	if adm.docBuilder != nil {
		adm.docBuilder.AddSchemaFromStruct(name, v)
	}
	return adm
}

// AddTag adds a tag to the documentation
func (adm *APIDocumentationManager) AddTag(name, description string) *APIDocumentationManager {
	tag := &Tag{
		Name:        name,
		Description: description,
	}
	
	if adm.docBuilder != nil {
		adm.docBuilder.AddTag(tag)
	}
	return adm
}

// AddBearerAuth adds JWT Bearer authentication
func (adm *APIDocumentationManager) AddBearerAuth() *APIDocumentationManager {
	if adm.docBuilder != nil {
		adm.docBuilder.AddBearerAuth()
	}
	return adm
}

// AddAPIKeyAuth adds API Key authentication
func (adm *APIDocumentationManager) AddAPIKeyAuth(name, in string) *APIDocumentationManager {
	if adm.docBuilder != nil {
		adm.docBuilder.AddAPIKeyAuth(name, in)
	}
	return adm
}

// GenerateDocumentation generates the complete API documentation
func (adm *APIDocumentationManager) GenerateDocumentation() (*OpenAPISpec, error) {
	if adm.versionedDocs != nil {
		versionedSpec, err := adm.versionedDocs.GetDefaultSpec()
		if err != nil {
			return nil, err
		}
		return versionedSpec.Spec, nil
	}
	
	if adm.docBuilder != nil {
		return adm.docBuilder.GenerateOpenAPISpec()
	}
	
	return nil, fmt.Errorf("no documentation builder available")
}

// Extension methods for Application

// EnableAPIDocumentation enables API documentation for the application
func (app *Application) EnableAPIDocumentation() *APIDocumentationManager {
	manager := NewAPIDocumentationManager(app)
	// Store in container for later access
	app.container.Singleton("api_docs", manager)
	return manager
}

// GetAPIDocumentationManager gets the API documentation manager from container
func (app *Application) GetAPIDocumentationManager() *APIDocumentationManager {
	if manager, err := app.container.Make("api_docs"); err == nil {
		if apiDocs, ok := manager.(*APIDocumentationManager); ok {
			return apiDocs
		}
	}
	return nil
}

// Documentation helper middleware

// CORSForDocs returns CORS middleware configured for documentation
func CORSForDocs() MiddlewareFunc {
	return func(c *Context) error {
		if strings.HasPrefix(c.Request.URL.Path, "/docs") {
			c.Header("Access-Control-Allow-Origin", "*")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, API-Version")
			
			if c.Method() == "OPTIONS" {
				c.Status(204)
				c.Abort()
				return nil
			}
		}
		return c.Next()
	}
}

// APIResponseMiddleware adds consistent API response structure
func APIResponseMiddleware() MiddlewareFunc {
	return func(c *Context) error {
		// Add API version to response headers
		if version := GetVersionFromContext(c); version != "" {
			c.Header("API-Version", version)
		}
		
		// Add request ID for tracing
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = fmt.Sprintf("%d", time.Now().UnixNano())
		}
		c.Header("X-Request-ID", requestID)
		
		return c.Next()
	}
}

// Quick setup functions

// SetupBasicAPIDocumentation sets up basic API documentation
func SetupBasicAPIDocumentation(app *Application, title, version, description string) *APIDocumentationManager {
	manager := app.EnableAPIDocumentation()
	
	config := &APIDocConfig{
		Title:       title,
		Version:     version,
		Description: description,
		Contact: &Contact{
			Name: "API Team",
		},
		License: &License{
			Name: "MIT",
		},
	}
	
	manager.Configure(config)
	manager.AddBearerAuth()
	manager.AddTag("Default", "Default API operations")
	
	// Add basic schemas
	manager.AddSchemaFromStruct("Error", struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
		Message string `json:"message"`
		Code    int    `json:"code"`
	}{})
	
	manager.Initialize()
	return manager
}

// SetupVersionedAPIDocumentation sets up versioned API documentation
func SetupVersionedAPIDocumentation(app *Application, title, description string) *APIDocumentationManager {
	manager := app.EnableAPIDocumentation()
	
	config := &APIDocConfig{
		Title:       title,
		Description: description,
		Contact: &Contact{
			Name: "API Team",
		},
		License: &License{
			Name: "MIT",
		},
	}
	
	manager.Configure(config)
	
	// Enable versioning
	versionManager := manager.EnableVersioning()
	versionManager.SetDefaultVersion("v1")
	
	// Add versions
	manager.AddVersion("v1", "Version 1.0", "Initial API version")
	manager.AddVersion("v2", "Version 2.0", "Enhanced API with new features")
	
	// Add authentication and common schemas
	manager.AddBearerAuth()
	manager.AddAPIKeyAuth("X-API-Key", "header")
	
	// Add common tags
	manager.AddTag("Authentication", "Authentication operations")
	manager.AddTag("Users", "User management operations")
	manager.AddTag("System", "System operations")
	
	// Add common schemas
	manager.AddSchemaFromStruct("Error", struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
		Message string `json:"message"`
		Code    int    `json:"code"`
	}{})
	
	manager.AddSchemaFromStruct("User", struct {
		ID        int       `json:"id"`
		Name      string    `json:"name"`
		Email     string    `json:"email"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}{})
	
	manager.Initialize()
	return manager
}

// Helper functions for route documentation

// CreateDocumentedRoute creates a route with documentation
func CreateDocumentedRoute(
	app *Application,
	method, pattern string,
	handler HandlerFunc,
	doc *RouteDocumentation,
	middleware ...MiddlewareFunc,
) {
	// Add the route
	switch strings.ToUpper(method) {
	case "GET":
		app.Get(pattern, handler, middleware...)
	case "POST":
		app.Post(pattern, handler, middleware...)
	case "PUT":
		app.Put(pattern, handler, middleware...)
	case "DELETE":
		app.Delete(pattern, handler, middleware...)
	case "PATCH":
		app.Patch(pattern, handler, middleware...)
	case "OPTIONS":
		app.Options(pattern, handler, middleware...)
	case "HEAD":
		app.Head(pattern, handler, middleware...)
	}
	
	// Add documentation
	if manager := app.GetAPIDocumentationManager(); manager != nil && doc != nil {
		manager.DocumentRoute(method, pattern, doc)
	}
}

// DocumentedVersionedRoute creates a versioned route with documentation
func DocumentedVersionedRoute(
	app *Application,
	version, method, pattern string,
	handler HandlerFunc,
	doc *RouteDocumentation,
	middleware ...MiddlewareFunc,
) {
	// Create full pattern with version
	fullPattern := fmt.Sprintf("/api/%s%s", version, pattern)
	
	// Add the route
	CreateDocumentedRoute(app, method, fullPattern, handler, doc, middleware...)
	
	// Add versioned documentation
	if manager := app.GetAPIDocumentationManager(); manager != nil && doc != nil {
		manager.DocumentVersionedRoute(version, method, pattern, doc)
	}
}

// QuickCRUDRoutes creates documented CRUD routes for a resource
func QuickCRUDRoutes(
	app *Application,
	resource string,
	controller interface{},
	itemSchema *OpenAPISchema,
) {
	docs := CRUDDocumentation(resource, itemSchema)
	basePath := "/" + resource
	
	// List
	if listHandler := getMethodByName(controller, "Index"); listHandler != nil {
		CreateDocumentedRoute(app, "GET", basePath, listHandler, docs["GET"])
	}
	
	// Create
	if createHandler := getMethodByName(controller, "Create"); createHandler != nil {
		CreateDocumentedRoute(app, "POST", basePath, createHandler, docs["POST"])
	}
	
	// Show
	if showHandler := getMethodByName(controller, "Show"); showHandler != nil {
		CreateDocumentedRoute(app, "GET", basePath+"/{id:int}", showHandler, docs["GET_ID"])
	}
	
	// Update
	if updateHandler := getMethodByName(controller, "Update"); updateHandler != nil {
		CreateDocumentedRoute(app, "PUT", basePath+"/{id:int}", updateHandler, docs["PUT_ID"])
	}
	
	// Delete
	if deleteHandler := getMethodByName(controller, "Delete"); deleteHandler != nil {
		CreateDocumentedRoute(app, "DELETE", basePath+"/{id:int}", deleteHandler, docs["DELETE_ID"])
	}
}

// getMethodByName gets a method from a controller by name
func getMethodByName(controller interface{}, methodName string) HandlerFunc {
	// This would use reflection to get the method
	// For now, return nil as this would need more complex implementation
	return nil
}

// Validation helpers

// ValidateAPIDocumentation validates the API documentation
func ValidateAPIDocumentation(app *Application) []string {
	var errors []string
	
	manager := app.GetAPIDocumentationManager()
	if manager == nil {
		errors = append(errors, "API documentation not configured")
		return errors
	}
	
	if manager.docBuilder != nil {
		builderErrors := manager.docBuilder.ValidateSpec()
		errors = append(errors, builderErrors...)
	}
	
	if manager.versionedDocs != nil {
		versionErrors := manager.versionedDocs.ValidateVersionedSpecs()
		for version, versionErrs := range versionErrors {
			for _, err := range versionErrs {
				errors = append(errors, fmt.Sprintf("Version %s: %s", version, err))
			}
		}
	}
	
	return errors
}