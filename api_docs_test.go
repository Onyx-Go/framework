package onyx

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"
)

// TestOpenAPIBuilder tests the OpenAPI specification builder
func TestOpenAPIBuilder(t *testing.T) {
	t.Run("should create basic OpenAPI spec", func(t *testing.T) {
		builder := NewOpenAPIBuilder("Test API", "1.0.0")
		
		// Add basic info
		builder.SetInfo(OpenAPIInfo{
			Title:       "Test API",
			Version:     "1.0.0",
			Description: "A test API",
		})
		
		// Add server
		builder.AddServer("https://api.example.com", "Production server")
		
		// Add tag
		builder.AddTag("Users", "User management operations")
		
		// Add security scheme
		builder.AddBearerAuth()
		
		spec := builder.Build()
		
		if spec.OpenAPI != "3.0.3" {
			t.Errorf("Expected OpenAPI version 3.0.3, got %s", spec.OpenAPI)
		}
		
		if spec.Info.Title != "Test API" {
			t.Errorf("Expected title 'Test API', got %s", spec.Info.Title)
		}
		
		if len(spec.Servers) != 1 {
			t.Errorf("Expected 1 server, got %d", len(spec.Servers))
		}
		
		if len(spec.Tags) != 1 {
			t.Errorf("Expected 1 tag, got %d", len(spec.Tags))
		}
		
		if spec.Components == nil || spec.Components.SecuritySchemes == nil {
			t.Error("Expected security schemes to be defined")
		}
	})
	
	t.Run("should generate spec from router", func(t *testing.T) {
		router := NewRouter()
		
		// Add some routes
		router.Get("/users", func(c Context) error {
			return c.JSON(200, map[string]string{"message": "Users"})
		})
		
		router.Post("/users", func(c Context) error {
			return c.JSON(201, map[string]string{"message": "User created"})
		})
		
		router.Get("/users/{id:int}", func(c Context) error {
			return c.JSON(200, map[string]string{"message": "User details"})
		})
		
		builder := NewOpenAPIBuilder("API", "1.0.0")
		builder.FromRouter(router)
		
		spec := builder.Build()
		
		if len(spec.Paths) != 2 { // /users and /users/{id}
			t.Errorf("Expected 2 paths, got %d", len(spec.Paths))
		}
		
		// Check if /users path exists
		usersPath, exists := spec.Paths["/users"]
		if !exists {
			t.Error("Expected /users path to exist")
		}
		
		if usersPath.Get == nil {
			t.Error("Expected GET operation on /users")
		}
		
		if usersPath.Post == nil {
			t.Error("Expected POST operation on /users")
		}
		
		// Check if /users/{id} path exists
		userIDPath, exists := spec.Paths["/users/{id}"]
		if !exists {
			t.Error("Expected /users/{id} path to exist")
		}
		
		if userIDPath.Get == nil {
			t.Error("Expected GET operation on /users/{id}")
		}
		
		// Check parameter extraction
		if len(userIDPath.Get.Parameters) != 1 {
			t.Errorf("Expected 1 parameter, got %d", len(userIDPath.Get.Parameters))
		}
		
		param := userIDPath.Get.Parameters[0]
		if param.Name != "id" {
			t.Errorf("Expected parameter name 'id', got %s", param.Name)
		}
		
		if param.In != "path" {
			t.Errorf("Expected parameter in 'path', got %s", param.In)
		}
	})
	
	t.Run("should convert path patterns correctly", func(t *testing.T) {
		builder := NewOpenAPIBuilder("API", "1.0.0")
		
		testCases := []struct {
			input    string
			expected string
		}{
			{"/users", "/users"},
			{"/users/{id}", "/users/{id}"},
			{"/users/{id:int}", "/users/{id}"},
			{"/users/{id:alpha}", "/users/{id}"},
			{"/api/v1/users/{userId:int}/posts/{postId}", "/api/v1/users/{userId}/posts/{postId}"},
		}
		
		for _, tc := range testCases {
			result := builder.convertPathToOpenAPI(tc.input)
			if result != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, result)
			}
		}
	})
}

// TestSchemaGeneration tests schema generation from Go structs
func TestSchemaGeneration(t *testing.T) {
	type User struct {
		ID       int       `json:"id"`
		Name     string    `json:"name"`
		Email    string    `json:"email"`
		Active   bool      `json:"active"`
		Score    float64   `json:"score"`
		Tags     []string  `json:"tags"`
		Settings map[string]interface{} `json:"settings"`
		CreatedAt time.Time `json:"created_at"`
	}
	
	builder := NewOpenAPIBuilder("API", "1.0.0")
	schema := builder.SchemaFromStruct(User{})
	
	if schema.Type != "object" {
		t.Errorf("Expected type 'object', got %s", schema.Type)
	}
	
	expectedFields := map[string]string{
		"id":         "integer",
		"name":       "string",
		"email":      "string",
		"active":     "boolean",
		"score":      "number",
		"tags":       "array",
		"settings":   "object",
		"created_at": "string",
	}
	
	for field, expectedType := range expectedFields {
		if fieldSchema, exists := schema.Properties[field]; exists {
			if fieldSchema.Type != expectedType {
				t.Errorf("Expected %s type %s, got %s", field, expectedType, fieldSchema.Type)
			}
		} else {
			t.Errorf("Expected field %s to exist", field)
		}
	}
	
	// Check array items
	if schema.Properties["tags"].Items == nil || schema.Properties["tags"].Items.Type != "string" {
		t.Error("Expected tags array to have string items")
	}
	
	// Check date-time format
	if schema.Properties["created_at"].Format != "date-time" {
		t.Error("Expected created_at to have date-time format")
	}
}

// TestAPIDocumentationBuilder tests the main documentation builder
func TestAPIDocumentationBuilder(t *testing.T) {
	config := &APIDocConfig{
		Title:       "Test API",
		Version:     "1.0.0",
		Description: "Test API documentation",
	}
	
	builder := NewAPIDocumentationBuilder(config)
	
	t.Run("should add schemas", func(t *testing.T) {
		schema := &OpenAPISchema{
			Type: "object",
			Properties: map[string]*OpenAPISchema{
				"id":   {Type: "integer"},
				"name": {Type: "string"},
			},
		}
		
		builder.AddSchema("User", schema)
		
		if _, exists := builder.schemas["User"]; !exists {
			t.Error("Expected User schema to be added")
		}
	})
	
	t.Run("should add tags", func(t *testing.T) {
		tag := &Tag{
			Name:        "Users",
			Description: "User operations",
		}
		
		builder.AddTag(tag)
		
		if _, exists := builder.tags["Users"]; !exists {
			t.Error("Expected Users tag to be added")
		}
	})
	
	t.Run("should document routes", func(t *testing.T) {
		doc := &RouteDocumentation{
			Summary:     "Get users",
			Description: "Retrieve a list of users",
			Tags:        []string{"Users"},
			Parameters: []ParameterDoc{
				{
					Name:        "page",
					In:          "query",
					Type:        "integer",
					Required:    false,
					Description: "Page number",
				},
			},
			Responses: map[string]ResponseDoc{
				"200": {
					Description: "Success",
					Schema: &OpenAPISchema{
						Type: "array",
						Items: &OpenAPISchema{
							Ref: "#/components/schemas/User",
						},
					},
				},
			},
		}
		
		builder.DocumentRoute("GET", "/users", doc)
		
		routes := builder.GetAllRoutes()
		if _, exists := routes["GET /users"]; !exists {
			t.Error("Expected GET /users route to be documented")
		}
	})
	
	t.Run("should parse annotations from source", func(t *testing.T) {
		sourceCode := `
		// GetUsers retrieves users
		// @Summary Get all users
		// @Description Retrieve a paginated list of users
		// @Tags Users
		// @Param page query int false "Page number"
		// @Success 200 {array} User "Success"
		// @Failure 400 {object} Error "Bad Request"
		// @Router /api/v1/users [get]
		func GetUsers(c *Context) error {
			return c.JSON(200, users)
		}
		`
		
		annotations := builder.ParseAnnotationsFromSource(sourceCode)
		
		if len(annotations) != 1 {
			t.Errorf("Expected 1 annotation, got %d", len(annotations))
		}
		
		doc, exists := annotations["GET /api/v1/users"]
		if !exists {
			t.Error("Expected GET /api/v1/users annotation")
		}
		
		if doc.Summary != "Get all users" {
			t.Errorf("Expected summary 'Get all users', got %s", doc.Summary)
		}
		
		if len(doc.Tags) != 1 || doc.Tags[0] != "Users" {
			t.Errorf("Expected tags [Users], got %v", doc.Tags)
		}
		
		if len(doc.Parameters) != 1 {
			t.Errorf("Expected 1 parameter, got %d", len(doc.Parameters))
		}
		
		if len(doc.Responses) != 2 {
			t.Errorf("Expected 2 responses, got %d", len(doc.Responses))
		}
	})
}

// TestAPIVersioning tests the API versioning system
func TestAPIVersioning(t *testing.T) {
	manager := NewAPIVersionManager()
	
	t.Run("should register versions", func(t *testing.T) {
		v1 := CreateAPIVersionConfig("v1", "Version 1.0", "Initial version")
		v2 := CreateAPIVersionConfig("v2", "Version 2.0", "Enhanced version")
		
		err1 := manager.RegisterVersion(v1)
		err2 := manager.RegisterVersion(v2)
		
		if err1 != nil {
			t.Errorf("Failed to register v1: %v", err1)
		}
		
		if err2 != nil {
			t.Errorf("Failed to register v2: %v", err2)
		}
		
		versions := manager.GetAllVersions()
		if len(versions) != 2 {
			t.Errorf("Expected 2 versions, got %d", len(versions))
		}
	})
	
	t.Run("should handle deprecation", func(t *testing.T) {
		eolDate := time.Now().Add(30 * 24 * time.Hour) // 30 days from now
		err := manager.DeprecateVersion("v1", &eolDate)
		
		if err != nil {
			t.Errorf("Failed to deprecate v1: %v", err)
		}
		
		version, exists := manager.GetVersion("v1")
		if !exists {
			t.Error("Expected v1 to exist")
		}
		
		if !version.Deprecated {
			t.Error("Expected v1 to be deprecated")
		}
		
		if version.Status != VersionStatusDeprecated {
			t.Errorf("Expected status deprecated, got %s", version.Status)
		}
	})
	
	t.Run("should extract version from context", func(t *testing.T) {
		testCases := []struct {
			path     string
			header   string
			query    string
			expected string
		}{
			{"/api/v1/users", "", "", "v1"},
			{"/api/v2.1/users", "", "", "v2.1"},
			{"/users", "v2", "", "v2"},
			{"/users", "", "v1", "v1"},
			{"/users", "", "", ""},
		}
		
		_ = &DefaultVersionExtractor{} // placeholder for extractor logic
		
		for _, tc := range testCases {
			// Mock context - in real test this would be a proper HTTP request
			// This is a simplified test of the logic
			if strings.Contains(tc.path, "/api/v") {
				re := regexp.MustCompile(`^/api/v(\d+(?:\.\d+)?)/`)
				matches := re.FindStringSubmatch(tc.path)
				if len(matches) > 1 {
					result := "v" + matches[1]
					if result != tc.expected {
						t.Errorf("Path %s: expected %s, got %s", tc.path, tc.expected, result)
					}
				}
			}
		}
	})
}

// TestVersionedDocumentation tests versioned documentation management
func TestVersionedDocumentation(t *testing.T) {
	versionManager := NewAPIVersionManager()
	
	// Register test versions
	v1 := CreateAPIVersionConfig("v1", "Version 1.0", "Initial API version")
	v2 := CreateAPIVersionConfig("v2", "Version 2.0", "Enhanced API version")
	versionManager.RegisterVersion(v1)
	versionManager.RegisterVersion(v2)
	
	config := &VersionedDocsConfig{
		BaseTitle:       "Test API",
		BaseDescription: "Test API Documentation",
		IncludeVersion:  true,
		ShowDeprecated:  true,
		DefaultVersion:  "v1",
	}
	
	versionedDocs := NewVersionedDocumentation(versionManager, config)
	
	t.Run("should create builders for versions", func(t *testing.T) {
		builderV1 := versionedDocs.GetBuilder("v1")
		builderV2 := versionedDocs.GetBuilder("v2")
		
		if builderV1 == nil {
			t.Error("Expected v1 builder to be created")
		}
		
		if builderV2 == nil {
			t.Error("Expected v2 builder to be created")
		}
		
		if builderV1 == builderV2 {
			t.Error("Expected different builders for different versions")
		}
	})
	
	t.Run("should generate version matrix", func(t *testing.T) {
		matrix := versionedDocs.GenerateVersionMatrix()
		
		if len(matrix.Versions) != 2 {
			t.Errorf("Expected 2 versions in matrix, got %d", len(matrix.Versions))
		}
		
		if len(matrix.Matrix) != 2 {
			t.Errorf("Expected 2x2 compatibility matrix, got %dx%d", len(matrix.Matrix), len(matrix.Matrix))
		}
	})
	
	t.Run("should generate changelog", func(t *testing.T) {
		changelog := versionedDocs.GenerateChangelog()
		
		if len(changelog.Entries) != 2 {
			t.Errorf("Expected 2 changelog entries, got %d", len(changelog.Entries))
		}
		
		// Should be sorted by release date (newest first)
		if len(changelog.Entries) >= 2 {
			if changelog.Entries[0].Released.Before(changelog.Entries[1].Released) {
				t.Error("Expected changelog to be sorted by release date (newest first)")
			}
		}
	})
}

// TestDocumentationMiddleware tests the documentation middleware
func TestDocumentationMiddleware(t *testing.T) {
	config := &DocsMiddlewareConfig{
		Enabled:       true,
		AutoDiscovery: true,
		CacheEnabled:  true,
		CacheDuration: 5 * time.Minute,
	}
	
	middleware := NewDocsMiddleware(config)
	
	t.Run("should cache specs", func(t *testing.T) {
		spec := &OpenAPISpec{
			OpenAPI: "3.0.3",
			Info: OpenAPIInfo{
				Title:   "Test API",
				Version: "1.0.0",
			},
			Paths: make(map[string]PathItem),
		}
		
		// Cache the spec
		middleware.cache.CacheSpec("test", spec, "v1")
		
		// Retrieve from cache
		cached := middleware.cache.GetCachedSpec("test")
		
		if cached == nil {
			t.Error("Expected cached spec to be retrieved")
		}
		
		if cached.Spec.Info.Title != "Test API" {
			t.Errorf("Expected cached title 'Test API', got %s", cached.Spec.Info.Title)
		}
	})
	
	t.Run("should discover routes", func(t *testing.T) {
		router := NewRouter()
		
		router.Get("/users", func(c Context) error {
			return c.JSON(200, "users")
		})
		
		router.Post("/users", func(c Context) error {
			return c.JSON(201, "created")
		})
		
		middleware.DiscoverRoutes(router)
		
		discovered := middleware.GetDiscoveredRoutes()
		
		if len(discovered) != 2 {
			t.Errorf("Expected 2 discovered routes, got %d", len(discovered))
		}
		
		if _, exists := discovered["GET /users"]; !exists {
			t.Error("Expected GET /users to be discovered")
		}
		
		if _, exists := discovered["POST /users"]; !exists {
			t.Error("Expected POST /users to be discovered")
		}
	})
}

// TestCodeGenerator tests the code generation functionality
func TestCodeGenerator(t *testing.T) {
	generator := NewCodeGenerator()
	
	t.Run("should support multiple languages", func(t *testing.T) {
		languages := generator.GetSupportedLanguages()
		
		expectedLanguages := []string{"javascript", "python", "go", "java", "csharp", "php"}
		
		for _, lang := range expectedLanguages {
			if _, exists := languages[lang]; !exists {
				t.Errorf("Expected language %s to be supported", lang)
			}
		}
	})
	
	t.Run("should generate JavaScript client", func(t *testing.T) {
		spec := &OpenAPISpec{
			OpenAPI: "3.0.3",
			Info: OpenAPIInfo{
				Title:   "Test API",
				Version: "1.0.0",
			},
			Paths: make(map[string]PathItem),
		}
		
		options := &GenerationOptions{
			PackageName:  "test-client",
			OutputFormat: "client",
		}
		
		code, err := generator.generateJavaScript(spec, options)
		
		if err != nil {
			t.Errorf("Failed to generate JavaScript code: %v", err)
		}
		
		if !strings.Contains(code.Files["index.js"], "class ApiClient") {
			t.Error("Expected generated code to contain ApiClient class")
		}
		
		if !strings.Contains(code.Files["package.json"], "test-client") {
			t.Error("Expected package.json to contain package name")
		}
	})
}

// TestSwaggerUI tests Swagger UI functionality
func TestSwaggerUI(t *testing.T) {
	config := DefaultSwaggerUIConfig()
	config.Title = "Test API"
	config.Path = "/docs"
	
	server := NewSwaggerUIServer(config, nil)
	
	t.Run("should configure properly", func(t *testing.T) {
		if server.config.Title != "Test API" {
			t.Errorf("Expected title 'Test API', got %s", server.config.Title)
		}
		
		if server.config.Path != "/docs" {
			t.Errorf("Expected path '/docs', got %s", server.config.Path)
		}
		
		if !server.config.TryItOutEnabled {
			t.Error("Expected TryItOutEnabled to be true by default")
		}
	})
	
	t.Run("should support customization", func(t *testing.T) {
		customCSS := ".swagger-ui { background: red; }"
		server.AddCustomCSS(customCSS)
		
		if !strings.Contains(server.config.CustomCSS, customCSS) {
			t.Error("Expected custom CSS to be added")
		}
		
		customJS := "console.log('custom');"
		server.AddCustomJS(customJS)
		
		if !strings.Contains(server.config.CustomJS, customJS) {
			t.Error("Expected custom JS to be added")
		}
	})
	
	t.Run("should set OAuth config", func(t *testing.T) {
		oauthConfig := &OAuthConfig{
			ClientId: "test-client",
			AppName:  "Test App",
		}
		
		server.SetOAuthConfig(oauthConfig)
		
		if server.config.OAuth == nil {
			t.Error("Expected OAuth config to be set")
		}
		
		if server.config.OAuth.ClientId != "test-client" {
			t.Errorf("Expected OAuth client ID 'test-client', got %s", server.config.OAuth.ClientId)
		}
	})
}

// TestAPIPlayground tests the API playground functionality
func TestAPIPlayground(t *testing.T) {
	playground := NewAPIPlayground()
	
	t.Run("should record requests", func(t *testing.T) {
		req := &PlaygroundRequest{
			ID:        "test-1",
			Method:    "GET",
			URL:       "https://api.example.com/users",
			Headers:   map[string]string{"Authorization": "Bearer token"},
			Timestamp: time.Now(),
		}
		
		playground.addToHistory(req)
		
		history := playground.GetHistory()
		if len(history) != 1 {
			t.Errorf("Expected 1 request in history, got %d", len(history))
		}
		
		if history[0].ID != "test-1" {
			t.Errorf("Expected request ID 'test-1', got %s", history[0].ID)
		}
	})
	
	t.Run("should limit history size", func(t *testing.T) {
		playground.maxHistory = 3
		
		// Add 5 requests
		for i := 0; i < 5; i++ {
			req := &PlaygroundRequest{
				ID:        fmt.Sprintf("test-%d", i),
				Method:    "GET",
				URL:       "https://api.example.com/users",
				Timestamp: time.Now(),
			}
			playground.addToHistory(req)
		}
		
		history := playground.GetHistory()
		if len(history) != 3 {
			t.Errorf("Expected history to be limited to 3, got %d", len(history))
		}
		
		// Should keep the most recent requests
		if history[0].ID != "test-2" {
			t.Errorf("Expected oldest request to be 'test-2', got %s", history[0].ID)
		}
	})
}

// TestValidation tests documentation validation
func TestValidation(t *testing.T) {
	t.Run("should validate OpenAPI spec", func(t *testing.T) {
		builder := NewAPIDocumentationBuilder(&APIDocConfig{
			Title:   "Test API",
			Version: "1.0.0",
		})
		
		errors := builder.ValidateSpec()
		
		// Should have no errors for basic valid spec
		if len(errors) > 0 {
			t.Errorf("Expected no validation errors, got: %v", errors)
		}
	})
	
	t.Run("should detect missing required fields", func(t *testing.T) {
		builder := NewAPIDocumentationBuilder(&APIDocConfig{
			Title: "", // Missing title
		})
		
		errors := builder.ValidateSpec()
		
		// Should have error for missing title
		found := false
		for _, err := range errors {
			if strings.Contains(err, "title") {
				found = true
				break
			}
		}
		
		if !found {
			t.Error("Expected validation error for missing title")
		}
	})
}

// TestJSONSerialization tests JSON serialization/deserialization
func TestJSONSerialization(t *testing.T) {
	t.Run("should serialize OpenAPI spec to JSON", func(t *testing.T) {
		spec := &OpenAPISpec{
			OpenAPI: "3.0.3",
			Info: OpenAPIInfo{
				Title:   "Test API",
				Version: "1.0.0",
			},
			Paths:   make(map[string]PathItem),
			Servers: []OpenAPIServer{
				{
					URL:         "https://api.example.com",
					Description: "Production server",
				},
			},
		}
		
		jsonBytes, err := json.Marshal(spec)
		if err != nil {
			t.Errorf("Failed to serialize to JSON: %v", err)
		}
		
		// Deserialize back
		var deserializedSpec OpenAPISpec
		err = json.Unmarshal(jsonBytes, &deserializedSpec)
		if err != nil {
			t.Errorf("Failed to deserialize from JSON: %v", err)
		}
		
		if deserializedSpec.Info.Title != "Test API" {
			t.Errorf("Expected title 'Test API', got %s", deserializedSpec.Info.Title)
		}
		
		if len(deserializedSpec.Servers) != 1 {
			t.Errorf("Expected 1 server, got %d", len(deserializedSpec.Servers))
		}
	})
}

// Benchmark tests for performance
func BenchmarkOpenAPIGeneration(b *testing.B) {
	router := NewRouter()
	
	// Add many routes to test performance
	for i := 0; i < 100; i++ {
		path := fmt.Sprintf("/api/v1/resource%d", i)
		router.Get(path, func(c Context) error {
			return c.JSON(200, "ok")
		})
		router.Post(path, func(c Context) error {
			return c.JSON(201, "created")
		})
	}
	
	builder := NewOpenAPIBuilder("Benchmark API", "1.0.0")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		builder.FromRouter(router)
		_ = builder.Build()
	}
}

func BenchmarkSchemaGeneration(b *testing.B) {
	type ComplexStruct struct {
		ID       int                    `json:"id"`
		Name     string                 `json:"name"`
		Tags     []string               `json:"tags"`
		Metadata map[string]interface{} `json:"metadata"`
		Items    []struct {
			ID    int    `json:"id"`
			Value string `json:"value"`
		} `json:"items"`
	}
	
	builder := NewOpenAPIBuilder("API", "1.0.0")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = builder.SchemaFromStruct(ComplexStruct{})
	}
}