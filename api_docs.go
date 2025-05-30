package onyx

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"sync"
)

// APIDocConfig configuration for API documentation
type APIDocConfig struct {
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Version     string            `json:"version"`
	Host        string            `json:"host"`
	BasePath    string            `json:"base_path"`
	Schemes     []string          `json:"schemes"`
	Contact     *Contact          `json:"contact,omitempty"`
	License     *License          `json:"license,omitempty"`
	Tags        []Tag             `json:"tags"`
	Security    []SecurityScheme  `json:"security"`
	Servers     []OpenAPIServer   `json:"servers"`
	Extensions  map[string]interface{} `json:"extensions,omitempty"`
}

// RouteDocumentation holds documentation for a specific route
type RouteDocumentation struct {
	Summary     string                 `json:"summary,omitempty"`
	Description string                 `json:"description,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Parameters  []ParameterDoc         `json:"parameters,omitempty"`
	RequestBody *RequestBodyDoc        `json:"request_body,omitempty"`
	Responses   map[string]ResponseDoc `json:"responses,omitempty"`
	Security    []SecurityRequirement  `json:"security,omitempty"`
	Deprecated  bool                   `json:"deprecated,omitempty"`
	Hidden      bool                   `json:"hidden,omitempty"`
	Examples    map[string]interface{} `json:"examples,omitempty"`
}

// ParameterDoc documentation for parameters
type ParameterDoc struct {
	Name        string      `json:"name"`
	In          string      `json:"in"` // path, query, header, cookie
	Type        string      `json:"type"`
	Required    bool        `json:"required"`
	Description string      `json:"description,omitempty"`
	Example     interface{} `json:"example,omitempty"`
	Schema      *OpenAPISchema `json:"schema,omitempty"`
}

// RequestBodyDoc documentation for request body
type RequestBodyDoc struct {
	Description string                      `json:"description,omitempty"`
	Required    bool                        `json:"required"`
	Content     map[string]MediaTypeDoc     `json:"content"`
}

// MediaTypeDoc documentation for media types
type MediaTypeDoc struct {
	Schema  *OpenAPISchema `json:"schema,omitempty"`
	Example interface{} `json:"example,omitempty"`
}

// ResponseDoc documentation for responses
type ResponseDoc struct {
	Description string                  `json:"description"`
	Schema      *OpenAPISchema          `json:"schema,omitempty"`
	Headers     map[string]ParameterDoc `json:"headers,omitempty"`
	Examples    map[string]interface{}  `json:"examples,omitempty"`
}

// APIDocumentationBuilder builds comprehensive API documentation
type APIDocumentationBuilder struct {
	config          *APIDocConfig
	router          *Router
	routes          map[string]*RouteDocumentation
	schemas         map[string]*OpenAPISchema
	tags            map[string]*Tag
	securitySchemes map[string]SecurityScheme
	mutex           sync.RWMutex
	annotationCache map[string]*RouteDocumentation
}

// NewAPIDocumentationBuilder creates a new API documentation builder
func NewAPIDocumentationBuilder(config *APIDocConfig) *APIDocumentationBuilder {
	if config == nil {
		config = &APIDocConfig{
			Title:   "API Documentation",
			Version: "1.0.0",
			Schemes: []string{"http", "https"},
		}
	}

	return &APIDocumentationBuilder{
		config:          config,
		routes:          make(map[string]*RouteDocumentation),
		schemas:         make(map[string]*OpenAPISchema),
		tags:            make(map[string]*Tag),
		securitySchemes: make(map[string]SecurityScheme),
		annotationCache: make(map[string]*RouteDocumentation),
	}
}

// SetRouter sets the router for documentation generation
func (b *APIDocumentationBuilder) SetRouter(router *Router) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.router = router
}

// AddSchema adds a schema to the documentation
func (b *APIDocumentationBuilder) AddSchema(name string, schema *OpenAPISchema) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.schemas[name] = schema
}

// AddSchemaFromStruct adds a schema from a Go struct
func (b *APIDocumentationBuilder) AddSchemaFromStruct(name string, v interface{}) {
	builder := NewOpenAPIBuilder("", "")
	schema := builder.SchemaFromStruct(v)
	b.AddSchema(name, schema)
}

// AddTag adds a tag to the documentation
func (b *APIDocumentationBuilder) AddTag(tag *Tag) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.tags[tag.Name] = tag
}

// AddSecurityScheme adds a security scheme
func (b *APIDocumentationBuilder) AddSecurityScheme(name string, scheme SecurityScheme) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.securitySchemes[name] = scheme
}

// DocumentRoute adds documentation for a specific route
func (b *APIDocumentationBuilder) DocumentRoute(method, pattern string, doc *RouteDocumentation) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	
	key := fmt.Sprintf("%s %s", strings.ToUpper(method), pattern)
	b.routes[key] = doc
}

// AnnotateHandler adds annotations to a handler function
func (b *APIDocumentationBuilder) AnnotateHandler(handler HandlerFunc, doc *RouteDocumentation) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	
	handlerName := b.getFunctionName(handler)
	b.annotationCache[handlerName] = doc
}

// getFunctionName gets the name of a function
func (b *APIDocumentationBuilder) getFunctionName(f interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
}

// GenerateOpenAPISpec generates the complete OpenAPI specification
func (b *APIDocumentationBuilder) GenerateOpenAPISpec() (*OpenAPISpec, error) {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	if b.router == nil {
		return nil, fmt.Errorf("router not set")
	}

	builder := NewOpenAPIBuilder(b.config.Title, b.config.Version)

	// Set info
	builder.SetInfo(OpenAPIInfo{
		Title:          b.config.Title,
		Description:    b.config.Description,
		Version:        b.config.Version,
		Contact:        b.config.Contact,
		License:        b.config.License,
		TermsOfService: "",
	})

	// Add servers
	for _, server := range b.config.Servers {
		builder.AddServer(server.URL, server.Description)
	}

	// Add tags
	for _, tag := range b.tags {
		builder.AddTag(tag.Name, tag.Description)
	}

	// Add security schemes
	for name, scheme := range b.securitySchemes {
		builder.AddSecurityScheme(name, scheme)
	}

	// Generate from router with annotations
	spec := builder.FromRouter(b.router).Build()

	// Apply custom documentation
	b.applyCustomDocumentation(spec)

	// Add custom schemas
	if spec.Components == nil {
		spec.Components = &Components{}
	}
	if spec.Components.Schemas == nil {
		spec.Components.Schemas = make(map[string]*OpenAPISchema)
	}
	for name, schema := range b.schemas {
		spec.Components.Schemas[name] = schema
	}

	return spec, nil
}

// applyCustomDocumentation applies custom documentation to the spec
func (b *APIDocumentationBuilder) applyCustomDocumentation(spec *OpenAPISpec) {
	for path, pathItem := range spec.Paths {
		b.applyOperationDoc(path, "GET", pathItem.Get)
		b.applyOperationDoc(path, "POST", pathItem.Post)
		b.applyOperationDoc(path, "PUT", pathItem.Put)
		b.applyOperationDoc(path, "DELETE", pathItem.Delete)
		b.applyOperationDoc(path, "PATCH", pathItem.Patch)
		b.applyOperationDoc(path, "OPTIONS", pathItem.Options)
		b.applyOperationDoc(path, "HEAD", pathItem.Head)
	}
}

// applyOperationDoc applies documentation to an operation
func (b *APIDocumentationBuilder) applyOperationDoc(path, method string, operation *Operation) {
	if operation == nil {
		return
	}

	key := fmt.Sprintf("%s %s", method, path)
	doc, exists := b.routes[key]
	if !exists {
		return
	}

	if doc.Summary != "" {
		operation.Summary = doc.Summary
	}
	if doc.Description != "" {
		operation.Description = doc.Description
	}
	if len(doc.Tags) > 0 {
		operation.Tags = doc.Tags
	}
	if doc.Deprecated {
		operation.Deprecated = true
	}

	// Apply parameter documentation
	for i, param := range operation.Parameters {
		for _, docParam := range doc.Parameters {
			if param.Name == docParam.Name && param.In == docParam.In {
				if docParam.Description != "" {
					operation.Parameters[i].Description = docParam.Description
				}
				if docParam.Example != nil {
					operation.Parameters[i].Example = docParam.Example
				}
				break
			}
		}
	}

	// Apply request body documentation
	if operation.RequestBody != nil && doc.RequestBody != nil {
		if doc.RequestBody.Description != "" {
			operation.RequestBody.Description = doc.RequestBody.Description
		}
		operation.RequestBody.Required = doc.RequestBody.Required
	}

	// Apply response documentation
	for code, docResponse := range doc.Responses {
		if response, exists := operation.Responses[code]; exists {
			if docResponse.Description != "" {
				response.Description = docResponse.Description
			}
			operation.Responses[code] = response
		} else {
			// Add new response
			newResponse := Response{
				Description: docResponse.Description,
			}
			if docResponse.Schema != nil {
				newResponse.Content = map[string]MediaType{
					"application/json": {
						Schema: docResponse.Schema,
					},
				}
			}
			operation.Responses[code] = newResponse
		}
	}

	// Apply security documentation
	if len(doc.Security) > 0 {
		operation.Security = doc.Security
	}
}

// ParseAnnotationsFromSource parses documentation from source code comments
func (b *APIDocumentationBuilder) ParseAnnotationsFromSource(sourceCode string) map[string]*RouteDocumentation {
	annotations := make(map[string]*RouteDocumentation)
	
	// Regex patterns for parsing annotations
	patterns := map[string]*regexp.Regexp{
		"summary":     regexp.MustCompile(`@Summary\s+(.+)`),
		"description": regexp.MustCompile(`@Description\s+(.+)`),
		"tags":        regexp.MustCompile(`@Tags\s+(.+)`),
		"param":       regexp.MustCompile(`@Param\s+(\w+)\s+(\w+)\s+(\w+)\s+(\w+)\s+"([^"]+)"`),
		"success":     regexp.MustCompile(`@Success\s+(\d+)\s+\{(\w+)\}\s+(\w+)\s+"([^"]+)"`),
		"failure":     regexp.MustCompile(`@Failure\s+(\d+)\s+\{(\w+)\}\s+(\w+)\s+"([^"]+)"`),
		"router":      regexp.MustCompile(`@Router\s+(\S+)\s+\[(\w+)\]`),
		"security":    regexp.MustCompile(`@Security\s+(\w+)`),
		"deprecated":  regexp.MustCompile(`@Deprecated`),
	}

	lines := strings.Split(sourceCode, "\n")
	var currentDoc *RouteDocumentation
	var currentRoute string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Check for router annotation to identify the route
		if match := patterns["router"].FindStringSubmatch(line); match != nil {
			if currentDoc != nil && currentRoute != "" {
				annotations[currentRoute] = currentDoc
			}
			currentRoute = fmt.Sprintf("%s %s", strings.ToUpper(match[2]), match[1])
			currentDoc = &RouteDocumentation{
				Responses: make(map[string]ResponseDoc),
			}
			continue
		}

		if currentDoc == nil {
			continue
		}

		// Parse other annotations
		if match := patterns["summary"].FindStringSubmatch(line); match != nil {
			currentDoc.Summary = strings.TrimSpace(match[1])
		} else if match := patterns["description"].FindStringSubmatch(line); match != nil {
			currentDoc.Description = strings.TrimSpace(match[1])
		} else if match := patterns["tags"].FindStringSubmatch(line); match != nil {
			tags := strings.Split(match[1], ",")
			for i, tag := range tags {
				tags[i] = strings.TrimSpace(tag)
			}
			currentDoc.Tags = tags
		} else if match := patterns["param"].FindStringSubmatch(line); match != nil {
			param := ParameterDoc{
				Name:        match[1],
				In:          match[2],
				Type:        match[3],
				Required:    match[4] == "true",
				Description: match[5],
			}
			currentDoc.Parameters = append(currentDoc.Parameters, param)
		} else if match := patterns["success"].FindStringSubmatch(line); match != nil {
			code := match[1]
			currentDoc.Responses[code] = ResponseDoc{
				Description: match[4],
				Schema: &OpenAPISchema{
					Type: match[3],
				},
			}
		} else if match := patterns["failure"].FindStringSubmatch(line); match != nil {
			code := match[1]
			currentDoc.Responses[code] = ResponseDoc{
				Description: match[4],
				Schema: &OpenAPISchema{
					Type: match[3],
				},
			}
		} else if patterns["deprecated"].MatchString(line) {
			currentDoc.Deprecated = true
		}
	}

	// Add the last route if exists
	if currentDoc != nil && currentRoute != "" {
		annotations[currentRoute] = currentDoc
	}

	return annotations
}

// ApplyAnnotationsFromSource applies annotations from source code
func (b *APIDocumentationBuilder) ApplyAnnotationsFromSource(sourceCode string) {
	annotations := b.ParseAnnotationsFromSource(sourceCode)
	
	b.mutex.Lock()
	defer b.mutex.Unlock()
	
	for route, doc := range annotations {
		b.routes[route] = doc
	}
}

// GenerateMarkdown generates markdown documentation
func (b *APIDocumentationBuilder) GenerateMarkdown() (string, error) {
	spec, err := b.GenerateOpenAPISpec()
	if err != nil {
		return "", err
	}

	var md strings.Builder
	
	// Title and info
	md.WriteString(fmt.Sprintf("# %s\n\n", spec.Info.Title))
	if spec.Info.Description != "" {
		md.WriteString(fmt.Sprintf("%s\n\n", spec.Info.Description))
	}
	md.WriteString(fmt.Sprintf("**Version:** %s\n\n", spec.Info.Version))

	// Servers
	if len(spec.Servers) > 0 {
		md.WriteString("## Servers\n\n")
		for _, server := range spec.Servers {
			md.WriteString(fmt.Sprintf("- **%s** - %s\n", server.URL, server.Description))
		}
		md.WriteString("\n")
	}

	// Tags
	if len(spec.Tags) > 0 {
		md.WriteString("## Tags\n\n")
		for _, tag := range spec.Tags {
			md.WriteString(fmt.Sprintf("### %s\n", tag.Name))
			if tag.Description != "" {
				md.WriteString(fmt.Sprintf("%s\n", tag.Description))
			}
			md.WriteString("\n")
		}
	}

	// Paths
	md.WriteString("## Endpoints\n\n")
	for path, pathItem := range spec.Paths {
		md.WriteString(fmt.Sprintf("### %s\n\n", path))
		
		operations := map[string]*Operation{
			"GET":     pathItem.Get,
			"POST":    pathItem.Post,
			"PUT":     pathItem.Put,
			"DELETE":  pathItem.Delete,
			"PATCH":   pathItem.Patch,
			"OPTIONS": pathItem.Options,
			"HEAD":    pathItem.Head,
		}

		for method, operation := range operations {
			if operation == nil {
				continue
			}

			md.WriteString(fmt.Sprintf("#### %s %s\n\n", method, path))
			
			if operation.Summary != "" {
				md.WriteString(fmt.Sprintf("**Summary:** %s\n\n", operation.Summary))
			}
			
			if operation.Description != "" {
				md.WriteString(fmt.Sprintf("**Description:** %s\n\n", operation.Description))
			}

			if len(operation.Tags) > 0 {
				md.WriteString(fmt.Sprintf("**Tags:** %s\n\n", strings.Join(operation.Tags, ", ")))
			}

			// Parameters
			if len(operation.Parameters) > 0 {
				md.WriteString("**Parameters:**\n\n")
				md.WriteString("| Name | Type | In | Required | Description |\n")
				md.WriteString("|------|------|----|---------|-----------|\n")
				for _, param := range operation.Parameters {
					required := "No"
					if param.Required {
						required = "Yes"
					}
					paramType := "string"
					if param.Schema != nil && param.Schema.Type != "" {
						paramType = param.Schema.Type
					}
					md.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s |\n",
						param.Name, paramType, param.In, required, param.Description))
				}
				md.WriteString("\n")
			}

			// Responses
			if len(operation.Responses) > 0 {
				md.WriteString("**Responses:**\n\n")
				md.WriteString("| Code | Description |\n")
				md.WriteString("|------|-------------|\n")
				for code, response := range operation.Responses {
					md.WriteString(fmt.Sprintf("| %s | %s |\n", code, response.Description))
				}
				md.WriteString("\n")
			}

			md.WriteString("---\n\n")
		}
	}

	return md.String(), nil
}

// ExportToJSON exports the documentation to JSON format
func (b *APIDocumentationBuilder) ExportToJSON() ([]byte, error) {
	spec, err := b.GenerateOpenAPISpec()
	if err != nil {
		return nil, err
	}
	
	return json.MarshalIndent(spec, "", "  ")
}

// ExportToYAML exports the documentation to YAML format
func (b *APIDocumentationBuilder) ExportToYAML() ([]byte, error) {
	spec, err := b.GenerateOpenAPISpec()
	if err != nil {
		return nil, err
	}
	
	// Simple YAML-like output (would use yaml package in production)
	jsonBytes, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return nil, err
	}
	
	// Convert JSON to basic YAML format
	yamlStr := string(jsonBytes)
	yamlStr = strings.ReplaceAll(yamlStr, "{", "")
	yamlStr = strings.ReplaceAll(yamlStr, "}", "")
	yamlStr = strings.ReplaceAll(yamlStr, "\"", "")
	yamlStr = strings.ReplaceAll(yamlStr, ",", "")
	
	return []byte(yamlStr), nil
}

// GetRouteDocumentation gets documentation for a specific route
func (b *APIDocumentationBuilder) GetRouteDocumentation(method, pattern string) *RouteDocumentation {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	
	key := fmt.Sprintf("%s %s", strings.ToUpper(method), pattern)
	return b.routes[key]
}

// GetAllRoutes gets all documented routes
func (b *APIDocumentationBuilder) GetAllRoutes() map[string]*RouteDocumentation {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	
	result := make(map[string]*RouteDocumentation)
	for k, v := range b.routes {
		result[k] = v
	}
	return result
}

// Validation methods

// ValidateSpec validates the generated OpenAPI specification
func (b *APIDocumentationBuilder) ValidateSpec() []string {
	var errors []string
	
	spec, err := b.GenerateOpenAPISpec()
	if err != nil {
		errors = append(errors, fmt.Sprintf("Failed to generate spec: %v", err))
		return errors
	}
	
	// Basic validation
	if spec.Info.Title == "" {
		errors = append(errors, "API title is required")
	}
	
	if spec.Info.Version == "" {
		errors = append(errors, "API version is required")
	}
	
	// Validate paths
	for path, pathItem := range spec.Paths {
		if !strings.HasPrefix(path, "/") {
			errors = append(errors, fmt.Sprintf("Path '%s' must start with '/'", path))
		}
		
		operations := []*Operation{
			pathItem.Get, pathItem.Post, pathItem.Put, pathItem.Delete,
			pathItem.Patch, pathItem.Options, pathItem.Head,
		}
		
		for _, op := range operations {
			if op == nil {
				continue
			}
			
			if len(op.Responses) == 0 {
				errors = append(errors, fmt.Sprintf("Operation %s %s has no responses defined", path, "METHOD"))
			}
		}
	}
	
	return errors
}

// AddBearerAuth adds JWT Bearer authentication
func (b *APIDocumentationBuilder) AddBearerAuth() *APIDocumentationBuilder {
	b.AddSecurityScheme("bearerAuth", SecurityScheme{
		Type:         "http",
		Scheme:       "bearer",
		BearerFormat: "JWT",
		Description:  "JWT Bearer token authentication",
	})
	return b
}

// AddAPIKeyAuth adds API Key authentication
func (b *APIDocumentationBuilder) AddAPIKeyAuth(name, in string) *APIDocumentationBuilder {
	b.AddSecurityScheme("apiKeyAuth", SecurityScheme{
		Type:        "apiKey",
		Name:        name,
		In:          in,
		Description: "API Key authentication",
	})
	return b
}