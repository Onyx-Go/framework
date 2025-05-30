package onyx

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
)

// Documentation decorators and helpers for easy route annotation

// DocBuilder provides a fluent interface for building route documentation
type DocBuilder struct {
	doc *RouteDocumentation
}

// NewDocBuilder creates a new documentation builder
func NewDocBuilder() *DocBuilder {
	return &DocBuilder{
		doc: &RouteDocumentation{
			Parameters: make([]ParameterDoc, 0),
			Responses:  make(map[string]ResponseDoc),
		},
	}
}

// Summary sets the operation summary
func (d *DocBuilder) Summary(summary string) *DocBuilder {
	d.doc.Summary = summary
	return d
}

// Description sets the operation description
func (d *DocBuilder) Description(description string) *DocBuilder {
	d.doc.Description = description
	return d
}

// Tags sets the operation tags
func (d *DocBuilder) Tags(tags ...string) *DocBuilder {
	d.doc.Tags = tags
	return d
}

// Tag adds a single tag
func (d *DocBuilder) Tag(tag string) *DocBuilder {
	d.doc.Tags = append(d.doc.Tags, tag)
	return d
}

// Deprecated marks the operation as deprecated
func (d *DocBuilder) Deprecated() *DocBuilder {
	d.doc.Deprecated = true
	return d
}

// Hidden marks the operation as hidden from documentation
func (d *DocBuilder) Hidden() *DocBuilder {
	d.doc.Hidden = true
	return d
}

// PathParam adds a path parameter
func (d *DocBuilder) PathParam(name, description string) *DocBuilder {
	return d.addParam(name, "path", "string", true, description, nil)
}

// PathParamInt adds an integer path parameter
func (d *DocBuilder) PathParamInt(name, description string) *DocBuilder {
	return d.addParam(name, "path", "integer", true, description, nil)
}

// QueryParam adds a query parameter
func (d *DocBuilder) QueryParam(name, paramType, description string, required bool) *DocBuilder {
	return d.addParam(name, "query", paramType, required, description, nil)
}

// QueryParamWithExample adds a query parameter with example
func (d *DocBuilder) QueryParamWithExample(name, paramType, description string, required bool, example interface{}) *DocBuilder {
	return d.addParam(name, "query", paramType, required, description, example)
}

// HeaderParam adds a header parameter
func (d *DocBuilder) HeaderParam(name, description string, required bool) *DocBuilder {
	return d.addParam(name, "header", "string", required, description, nil)
}

// addParam adds a parameter to the documentation
func (d *DocBuilder) addParam(name, in, paramType string, required bool, description string, example interface{}) *DocBuilder {
	param := ParameterDoc{
		Name:        name,
		In:          in,
		Type:        paramType,
		Required:    required,
		Description: description,
		Example:     example,
		Schema: &OpenAPISchema{
			Type: paramType,
		},
	}
	
	if example != nil {
		param.Schema.Example = example
	}
	
	d.doc.Parameters = append(d.doc.Parameters, param)
	return d
}

// RequestBody sets the request body documentation
func (d *DocBuilder) RequestBody(description string, required bool, contentType string, schema *OpenAPISchema) *DocBuilder {
	if d.doc.RequestBody == nil {
		d.doc.RequestBody = &RequestBodyDoc{
			Content: make(map[string]MediaTypeDoc),
		}
	}
	
	d.doc.RequestBody.Description = description
	d.doc.RequestBody.Required = required
	d.doc.RequestBody.Content[contentType] = MediaTypeDoc{
		Schema: schema,
	}
	
	return d
}

// JSONRequestBody sets a JSON request body
func (d *DocBuilder) JSONRequestBody(description string, required bool, schema *OpenAPISchema) *DocBuilder {
	return d.RequestBody(description, required, "application/json", schema)
}

// Response adds a response documentation
func (d *DocBuilder) Response(code, description string, schema *OpenAPISchema) *DocBuilder {
	d.doc.Responses[code] = ResponseDoc{
		Description: description,
		Schema:      schema,
	}
	return d
}

// SuccessResponse adds a 200 success response
func (d *DocBuilder) SuccessResponse(description string, schema *OpenAPISchema) *DocBuilder {
	return d.Response("200", description, schema)
}

// CreatedResponse adds a 201 created response
func (d *DocBuilder) CreatedResponse(description string, schema *OpenAPISchema) *DocBuilder {
	return d.Response("201", description, schema)
}

// NoContentResponse adds a 204 no content response
func (d *DocBuilder) NoContentResponse() *DocBuilder {
	return d.Response("204", "No Content", nil)
}

// BadRequestResponse adds a 400 bad request response
func (d *DocBuilder) BadRequestResponse(description string) *DocBuilder {
	return d.Response("400", description, CreateErrorResponse())
}

// UnauthorizedResponse adds a 401 unauthorized response
func (d *DocBuilder) UnauthorizedResponse() *DocBuilder {
	return d.Response("401", "Unauthorized", CreateErrorResponse())
}

// ForbiddenResponse adds a 403 forbidden response
func (d *DocBuilder) ForbiddenResponse() *DocBuilder {
	return d.Response("403", "Forbidden", CreateErrorResponse())
}

// NotFoundResponse adds a 404 not found response
func (d *DocBuilder) NotFoundResponse() *DocBuilder {
	return d.Response("404", "Not Found", CreateErrorResponse())
}

// InternalErrorResponse adds a 500 internal server error response
func (d *DocBuilder) InternalErrorResponse() *DocBuilder {
	return d.Response("500", "Internal Server Error", CreateErrorResponse())
}

// Security adds security requirements
func (d *DocBuilder) Security(scheme string, scopes ...string) *DocBuilder {
	if d.doc.Security == nil {
		d.doc.Security = make([]SecurityRequirement, 0)
	}
	
	requirement := make(SecurityRequirement)
	requirement[scheme] = scopes
	d.doc.Security = append(d.doc.Security, requirement)
	
	return d
}

// BearerAuth adds bearer token authentication requirement
func (d *DocBuilder) BearerAuth() *DocBuilder {
	return d.Security("bearerAuth")
}

// APIKeyAuth adds API key authentication requirement
func (d *DocBuilder) APIKeyAuth() *DocBuilder {
	return d.Security("apiKeyAuth")
}

// Example adds an example to the documentation
func (d *DocBuilder) Example(name string, value interface{}) *DocBuilder {
	if d.doc.Examples == nil {
		d.doc.Examples = make(map[string]interface{})
	}
	d.doc.Examples[name] = value
	return d
}

// Build returns the completed route documentation
func (d *DocBuilder) Build() *RouteDocumentation {
	return d.doc
}

// Annotation helper functions for common patterns

// Doc creates a documentation builder
func Doc() *DocBuilder {
	return NewDocBuilder()
}

// Documented creates a documented handler wrapper
func Documented(handler HandlerFunc, doc *RouteDocumentation) HandlerFunc {
	// Store documentation in global registry if needed
	return handler
}

// DocumentedHandler is a decorator that adds documentation to a handler
type DocumentedHandler struct {
	Handler HandlerFunc
	Doc     *RouteDocumentation
}

// ServeHTTP implements the handler interface
func (dh *DocumentedHandler) ServeHTTP(c *Context) error {
	return dh.Handler(c)
}

// WithDoc creates a documented handler
func WithDoc(handler HandlerFunc, doc *RouteDocumentation) *DocumentedHandler {
	return &DocumentedHandler{
		Handler: handler,
		Doc:     doc,
	}
}

// Schema builders for common patterns

// StringSchema creates a string schema
func StringSchema() *OpenAPISchema {
	return &OpenAPISchema{Type: "string"}
}

// IntSchema creates an integer schema
func IntSchema() *OpenAPISchema {
	return &OpenAPISchema{Type: "integer", Format: "int64"}
}

// BoolSchema creates a boolean schema
func BoolSchema() *OpenAPISchema {
	return &OpenAPISchema{Type: "boolean"}
}

// ArraySchema creates an array schema
func ArraySchema(itemSchema *OpenAPISchema) *OpenAPISchema {
	return &OpenAPISchema{
		Type:  "array",
		Items: itemSchema,
	}
}

// ObjectSchema creates an object schema with properties
func ObjectSchema(properties map[string]*OpenAPISchema, required ...string) *OpenAPISchema {
	return &OpenAPISchema{
		Type:       "object",
		Properties: properties,
		Required:   required,
	}
}

// RefSchema creates a reference to another schema
func RefSchema(ref string) *OpenAPISchema {
	return &OpenAPISchema{Ref: fmt.Sprintf("#/components/schemas/%s", ref)}
}

// Common response schemas

// UserSchema example user schema
func UserSchema() *OpenAPISchema {
	return ObjectSchema(map[string]*OpenAPISchema{
		"id":         IntSchema(),
		"name":       StringSchema(),
		"email":      StringSchema(),
		"created_at": StringSchema(),
		"updated_at": StringSchema(),
	}, "id", "name", "email")
}

// PaginationMetaSchema creates pagination metadata schema
func PaginationMetaSchema() *OpenAPISchema {
	return ObjectSchema(map[string]*OpenAPISchema{
		"current_page": IntSchema(),
		"per_page":     IntSchema(),
		"total":        IntSchema(),
		"total_pages":  IntSchema(),
		"has_next":     BoolSchema(),
		"has_prev":     BoolSchema(),
	})
}

// ListResponseSchema creates a paginated list response schema
func ListResponseSchema(itemSchema *OpenAPISchema) *OpenAPISchema {
	return ObjectSchema(map[string]*OpenAPISchema{
		"success": BoolSchema(),
		"data":    ArraySchema(itemSchema),
		"meta":    PaginationMetaSchema(),
	}, "success", "data", "meta")
}

// SingleResponseSchema creates a single item response schema
func SingleResponseSchema(itemSchema *OpenAPISchema) *OpenAPISchema {
	return ObjectSchema(map[string]*OpenAPISchema{
		"success": BoolSchema(),
		"data":    itemSchema,
	}, "success", "data")
}

// Annotation parsing from comments

// RouteAnnotation represents a parsed route annotation
type RouteAnnotation struct {
	Method      string
	Path        string
	Summary     string
	Description string
	Tags        []string
	Parameters  []ParameterAnnotation
	Responses   []ResponseAnnotation
	Security    []string
	Deprecated  bool
}

// ParameterAnnotation represents a parameter annotation
type ParameterAnnotation struct {
	Name        string
	In          string
	Type        string
	Required    bool
	Description string
	Example     interface{}
}

// ResponseAnnotation represents a response annotation
type ResponseAnnotation struct {
	Code        string
	Description string
	Type        string
	Schema      string
}

// ParseRouteAnnotations parses annotations from a function
func ParseRouteAnnotations(handler interface{}) *RouteAnnotation {
	// Get function info
	fn := runtime.FuncForPC(reflect.ValueOf(handler).Pointer())
	if fn == nil {
		return nil
	}

	// This would typically read the source file and parse comments
	// For now, return a basic annotation
	return &RouteAnnotation{
		Summary: "Auto-generated documentation",
	}
}

// Helper functions for route registration with documentation

// DocumentedRoute represents a route with documentation
type DocumentedRoute struct {
	Method  string
	Pattern string
	Handler HandlerFunc
	Doc     *RouteDocumentation
	Middleware []MiddlewareFunc
}

// RouteGroup with documentation support
type DocumentedRouteGroup struct {
	group   *RouteGroup
	builder *APIDocumentationBuilder
	prefix  string
}

// NewDocumentedRouteGroup creates a documented route group
func NewDocumentedRouteGroup(group *RouteGroup, builder *APIDocumentationBuilder) *DocumentedRouteGroup {
	return &DocumentedRouteGroup{
		group:   group,
		builder: builder,
		prefix:  group.prefix,
	}
}

// Get adds a documented GET route
func (g *DocumentedRouteGroup) Get(pattern string, handler HandlerFunc, doc *RouteDocumentation, middleware ...MiddlewareFunc) {
	fullPattern := g.prefix + pattern
	g.group.Get(pattern, handler, middleware...)
	if doc != nil {
		g.builder.DocumentRoute("GET", fullPattern, doc)
	}
}

// Post adds a documented POST route
func (g *DocumentedRouteGroup) Post(pattern string, handler HandlerFunc, doc *RouteDocumentation, middleware ...MiddlewareFunc) {
	fullPattern := g.prefix + pattern
	g.group.Post(pattern, handler, middleware...)
	if doc != nil {
		g.builder.DocumentRoute("POST", fullPattern, doc)
	}
}

// Put adds a documented PUT route
func (g *DocumentedRouteGroup) Put(pattern string, handler HandlerFunc, doc *RouteDocumentation, middleware ...MiddlewareFunc) {
	fullPattern := g.prefix + pattern
	g.group.Put(pattern, handler, middleware...)
	if doc != nil {
		g.builder.DocumentRoute("PUT", fullPattern, doc)
	}
}

// Delete adds a documented DELETE route
func (g *DocumentedRouteGroup) Delete(pattern string, handler HandlerFunc, doc *RouteDocumentation, middleware ...MiddlewareFunc) {
	fullPattern := g.prefix + pattern
	g.group.Delete(pattern, handler, middleware...)
	if doc != nil {
		g.builder.DocumentRoute("DELETE", fullPattern, doc)
	}
}

// Common documentation patterns

// CRUDDocumentation generates standard CRUD documentation
func CRUDDocumentation(resource string, itemSchema *OpenAPISchema) map[string]*RouteDocumentation {
	resourceTitle := strings.Title(resource)
	
	return map[string]*RouteDocumentation{
		"GET": Doc().
			Summary(fmt.Sprintf("List %s", resource)).
			Description(fmt.Sprintf("Get a paginated list of %s", resource)).
			Tag(resourceTitle).
			QueryParam("page", "integer", "Page number", false).
			QueryParam("per_page", "integer", "Items per page", false).
			QueryParam("sort", "string", "Sort field", false).
			QueryParam("order", "string", "Sort order (asc, desc)", false).
			SuccessResponse("Success", ListResponseSchema(itemSchema)).
			BadRequestResponse("Invalid parameters").
			Build(),
			
		"POST": Doc().
			Summary(fmt.Sprintf("Create %s", strings.TrimSuffix(resource, "s"))).
			Description(fmt.Sprintf("Create a new %s", strings.TrimSuffix(resource, "s"))).
			Tag(resourceTitle).
			JSONRequestBody("Request body", true, itemSchema).
			CreatedResponse("Created", SingleResponseSchema(itemSchema)).
			BadRequestResponse("Invalid input").
			Build(),
			
		"GET_ID": Doc().
			Summary(fmt.Sprintf("Get %s", strings.TrimSuffix(resource, "s"))).
			Description(fmt.Sprintf("Get a specific %s by ID", strings.TrimSuffix(resource, "s"))).
			Tag(resourceTitle).
			PathParamInt("id", fmt.Sprintf("%s ID", resourceTitle)).
			SuccessResponse("Success", SingleResponseSchema(itemSchema)).
			NotFoundResponse().
			Build(),
			
		"PUT_ID": Doc().
			Summary(fmt.Sprintf("Update %s", strings.TrimSuffix(resource, "s"))).
			Description(fmt.Sprintf("Update a specific %s", strings.TrimSuffix(resource, "s"))).
			Tag(resourceTitle).
			PathParamInt("id", fmt.Sprintf("%s ID", resourceTitle)).
			JSONRequestBody("Request body", true, itemSchema).
			SuccessResponse("Updated", SingleResponseSchema(itemSchema)).
			BadRequestResponse("Invalid input").
			NotFoundResponse().
			Build(),
			
		"DELETE_ID": Doc().
			Summary(fmt.Sprintf("Delete %s", strings.TrimSuffix(resource, "s"))).
			Description(fmt.Sprintf("Delete a specific %s", strings.TrimSuffix(resource, "s"))).
			Tag(resourceTitle).
			PathParamInt("id", fmt.Sprintf("%s ID", resourceTitle)).
			NoContentResponse().
			NotFoundResponse().
			Build(),
	}
}

// AuthDocumentation generates authentication endpoint documentation
func AuthDocumentation() map[string]*RouteDocumentation {
	loginSchema := ObjectSchema(map[string]*OpenAPISchema{
		"email":    StringSchema(),
		"password": StringSchema(),
	}, "email", "password")
	
	tokenSchema := ObjectSchema(map[string]*OpenAPISchema{
		"token":      StringSchema(),
		"expires_at": StringSchema(),
	})
	
	return map[string]*RouteDocumentation{
		"POST_LOGIN": Doc().
			Summary("User login").
			Description("Authenticate user and return JWT token").
			Tag("Authentication").
			JSONRequestBody("Login credentials", true, loginSchema).
			SuccessResponse("Login successful", SingleResponseSchema(tokenSchema)).
			UnauthorizedResponse().
			Build(),
			
		"POST_LOGOUT": Doc().
			Summary("User logout").
			Description("Logout user and invalidate token").
			Tag("Authentication").
			BearerAuth().
			NoContentResponse().
			UnauthorizedResponse().
			Build(),
			
		"GET_ME": Doc().
			Summary("Get current user").
			Description("Get current authenticated user information").
			Tag("Authentication").
			BearerAuth().
			SuccessResponse("User information", SingleResponseSchema(UserSchema())).
			UnauthorizedResponse().
			Build(),
	}
}