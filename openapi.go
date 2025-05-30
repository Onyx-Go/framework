package onyx

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"
)

// OpenAPI 3.0 Specification Structures
type OpenAPISpec struct {
	OpenAPI      string                 `json:"openapi"`
	Info         OpenAPIInfo            `json:"info"`
	Servers      []OpenAPIServer        `json:"servers,omitempty"`
	Paths        map[string]PathItem    `json:"paths"`
	Components   *Components            `json:"components,omitempty"`
	Security     []SecurityRequirement  `json:"security,omitempty"`
	Tags         []Tag                  `json:"tags,omitempty"`
	ExternalDocs *ExternalDocumentation `json:"externalDocs,omitempty"`
}

type OpenAPIInfo struct {
	Title          string   `json:"title"`
	Description    string   `json:"description,omitempty"`
	TermsOfService string   `json:"termsOfService,omitempty"`
	Contact        *Contact `json:"contact,omitempty"`
	License        *License `json:"license,omitempty"`
	Version        string   `json:"version"`
}

type Contact struct {
	Name  string `json:"name,omitempty"`
	URL   string `json:"url,omitempty"`
	Email string `json:"email,omitempty"`
}

type License struct {
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}

type OpenAPIServer struct {
	URL         string                    `json:"url"`
	Description string                    `json:"description,omitempty"`
	Variables   map[string]ServerVariable `json:"variables,omitempty"`
}

type ServerVariable struct {
	Enum        []string `json:"enum,omitempty"`
	Default     string   `json:"default"`
	Description string   `json:"description,omitempty"`
}

type PathItem struct {
	Ref         string     `json:"$ref,omitempty"`
	Summary     string     `json:"summary,omitempty"`
	Description string     `json:"description,omitempty"`
	Get         *Operation `json:"get,omitempty"`
	Put         *Operation `json:"put,omitempty"`
	Post        *Operation `json:"post,omitempty"`
	Delete      *Operation `json:"delete,omitempty"`
	Options     *Operation `json:"options,omitempty"`
	Head        *Operation `json:"head,omitempty"`
	Patch       *Operation `json:"patch,omitempty"`
	Trace       *Operation `json:"trace,omitempty"`
	Servers     []OpenAPIServer `json:"servers,omitempty"`
	Parameters  []Parameter `json:"parameters,omitempty"`
}

type Operation struct {
	Tags         []string              `json:"tags,omitempty"`
	Summary      string                `json:"summary,omitempty"`
	Description  string                `json:"description,omitempty"`
	OperationID  string                `json:"operationId,omitempty"`
	Parameters   []Parameter           `json:"parameters,omitempty"`
	RequestBody  *RequestBody          `json:"requestBody,omitempty"`
	Responses    map[string]Response   `json:"responses"`
	Callbacks    map[string]Callback   `json:"callbacks,omitempty"`
	Deprecated   bool                  `json:"deprecated,omitempty"`
	Security     []SecurityRequirement `json:"security,omitempty"`
	Servers      []OpenAPIServer       `json:"servers,omitempty"`
	ExternalDocs *ExternalDocumentation `json:"externalDocs,omitempty"`
}

type Parameter struct {
	Name            string      `json:"name"`
	In              string      `json:"in"` // "query", "header", "path", "cookie"
	Description     string      `json:"description,omitempty"`
	Required        bool        `json:"required,omitempty"`
	Deprecated      bool        `json:"deprecated,omitempty"`
	AllowEmptyValue bool        `json:"allowEmptyValue,omitempty"`
	Style           string      `json:"style,omitempty"`
	Explode         *bool       `json:"explode,omitempty"`
	AllowReserved   bool        `json:"allowReserved,omitempty"`
	Schema          *OpenAPISchema `json:"schema,omitempty"`
	Example         interface{} `json:"example,omitempty"`
	Examples        map[string]Example `json:"examples,omitempty"`
	Content         map[string]MediaType `json:"content,omitempty"`
}

type RequestBody struct {
	Description string               `json:"description,omitempty"`
	Content     map[string]MediaType `json:"content"`
	Required    bool                 `json:"required,omitempty"`
}

type Response struct {
	Description string               `json:"description"`
	Headers     map[string]Header    `json:"headers,omitempty"`
	Content     map[string]MediaType `json:"content,omitempty"`
	Links       map[string]Link      `json:"links,omitempty"`
}

type MediaType struct {
	Schema   *OpenAPISchema      `json:"schema,omitempty"`
	Example  interface{}         `json:"example,omitempty"`
	Examples map[string]Example  `json:"examples,omitempty"`
	Encoding map[string]Encoding `json:"encoding,omitempty"`
}

type OpenAPISchema struct {
	Type                 string                 `json:"type,omitempty"`
	AllOf                []*OpenAPISchema       `json:"allOf,omitempty"`
	OneOf                []*OpenAPISchema       `json:"oneOf,omitempty"`
	AnyOf                []*OpenAPISchema       `json:"anyOf,omitempty"`
	Not                  *OpenAPISchema         `json:"not,omitempty"`
	Items                *OpenAPISchema         `json:"items,omitempty"`
	Properties           map[string]*OpenAPISchema `json:"properties,omitempty"`
	AdditionalProperties interface{}            `json:"additionalProperties,omitempty"`
	Description          string                 `json:"description,omitempty"`
	Format               string                 `json:"format,omitempty"`
	Default              interface{}            `json:"default,omitempty"`
	Title                string                 `json:"title,omitempty"`
	MultipleOf           *float64               `json:"multipleOf,omitempty"`
	Maximum              *float64               `json:"maximum,omitempty"`
	ExclusiveMaximum     *bool                  `json:"exclusiveMaximum,omitempty"`
	Minimum              *float64               `json:"minimum,omitempty"`
	ExclusiveMinimum     *bool                  `json:"exclusiveMinimum,omitempty"`
	MaxLength            *int                   `json:"maxLength,omitempty"`
	MinLength            *int                   `json:"minLength,omitempty"`
	Pattern              string                 `json:"pattern,omitempty"`
	MaxItems             *int                   `json:"maxItems,omitempty"`
	MinItems             *int                   `json:"minItems,omitempty"`
	UniqueItems          *bool                  `json:"uniqueItems,omitempty"`
	MaxProperties        *int                   `json:"maxProperties,omitempty"`
	MinProperties        *int                   `json:"minProperties,omitempty"`
	Required             []string               `json:"required,omitempty"`
	Enum                 []interface{}          `json:"enum,omitempty"`
	Nullable             bool                   `json:"nullable,omitempty"`
	Discriminator        *Discriminator         `json:"discriminator,omitempty"`
	ReadOnly             bool                   `json:"readOnly,omitempty"`
	WriteOnly            bool                   `json:"writeOnly,omitempty"`
	XML                  *XML                   `json:"xml,omitempty"`
	ExternalDocs         *ExternalDocumentation `json:"externalDocs,omitempty"`
	Example              interface{}            `json:"example,omitempty"`
	Deprecated           bool                   `json:"deprecated,omitempty"`
	Ref                  string                 `json:"$ref,omitempty"`
}

type Components struct {
	Schemas         map[string]*OpenAPISchema  `json:"schemas,omitempty"`
	Responses       map[string]Response        `json:"responses,omitempty"`
	Parameters      map[string]Parameter       `json:"parameters,omitempty"`
	Examples        map[string]Example         `json:"examples,omitempty"`
	RequestBodies   map[string]RequestBody     `json:"requestBodies,omitempty"`
	Headers         map[string]Header          `json:"headers,omitempty"`
	SecuritySchemes map[string]SecurityScheme  `json:"securitySchemes,omitempty"`
	Links           map[string]Link            `json:"links,omitempty"`
	Callbacks       map[string]Callback        `json:"callbacks,omitempty"`
}

type SecurityScheme struct {
	Type             string      `json:"type"`
	Description      string      `json:"description,omitempty"`
	Name             string      `json:"name,omitempty"`
	In               string      `json:"in,omitempty"`
	Scheme           string      `json:"scheme,omitempty"`
	BearerFormat     string      `json:"bearerFormat,omitempty"`
	Flows            OAuthFlows  `json:"flows,omitempty"`
	OpenIDConnectURL string      `json:"openIdConnectUrl,omitempty"`
}

type OAuthFlows struct {
	Implicit          *OAuthFlow `json:"implicit,omitempty"`
	Password          *OAuthFlow `json:"password,omitempty"`
	ClientCredentials *OAuthFlow `json:"clientCredentials,omitempty"`
	AuthorizationCode *OAuthFlow `json:"authorizationCode,omitempty"`
}

type OAuthFlow struct {
	AuthorizationURL string            `json:"authorizationUrl,omitempty"`
	TokenURL         string            `json:"tokenUrl,omitempty"`
	RefreshURL       string            `json:"refreshUrl,omitempty"`
	Scopes           map[string]string `json:"scopes"`
}

type SecurityRequirement map[string][]string

type Tag struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description,omitempty"`
	ExternalDocs *ExternalDocumentation `json:"externalDocs,omitempty"`
}

type ExternalDocumentation struct {
	Description string `json:"description,omitempty"`
	URL         string `json:"url"`
}

type Header struct {
	Description     string               `json:"description,omitempty"`
	Required        bool                 `json:"required,omitempty"`
	Deprecated      bool                 `json:"deprecated,omitempty"`
	AllowEmptyValue bool                 `json:"allowEmptyValue,omitempty"`
	Style           string               `json:"style,omitempty"`
	Explode         *bool                `json:"explode,omitempty"`
	AllowReserved   bool                 `json:"allowReserved,omitempty"`
	Schema          *OpenAPISchema       `json:"schema,omitempty"`
	Example         interface{}          `json:"example,omitempty"`
	Examples        map[string]Example   `json:"examples,omitempty"`
	Content         map[string]MediaType `json:"content,omitempty"`
}

type Example struct {
	Summary       string      `json:"summary,omitempty"`
	Description   string      `json:"description,omitempty"`
	Value         interface{} `json:"value,omitempty"`
	ExternalValue string      `json:"externalValue,omitempty"`
}

type Link struct {
	OperationRef string                 `json:"operationRef,omitempty"`
	OperationID  string                 `json:"operationId,omitempty"`
	Parameters   map[string]interface{} `json:"parameters,omitempty"`
	RequestBody  interface{}            `json:"requestBody,omitempty"`
	Description  string                 `json:"description,omitempty"`
	Server       *OpenAPIServer         `json:"server,omitempty"`
}

type Callback map[string]PathItem

type Encoding struct {
	ContentType   string              `json:"contentType,omitempty"`
	Headers       map[string]Header   `json:"headers,omitempty"`
	Style         string              `json:"style,omitempty"`
	Explode       *bool               `json:"explode,omitempty"`
	AllowReserved bool                `json:"allowReserved,omitempty"`
}

type Discriminator struct {
	PropertyName string            `json:"propertyName"`
	Mapping      map[string]string `json:"mapping,omitempty"`
}

type XML struct {
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Prefix    string `json:"prefix,omitempty"`
	Attribute bool   `json:"attribute,omitempty"`
	Wrapped   bool   `json:"wrapped,omitempty"`
}

// OpenAPI Documentation Builder
type OpenAPIBuilder struct {
	spec        *OpenAPISpec
	router      *Router
	components  *Components
	tagMap      map[string]*Tag
	schemaCache map[reflect.Type]*OpenAPISchema
}

// NewOpenAPIBuilder creates a new OpenAPI specification builder
func NewOpenAPIBuilder(title, version string) *OpenAPIBuilder {
	return &OpenAPIBuilder{
		spec: &OpenAPISpec{
			OpenAPI: "3.0.3",
			Info: OpenAPIInfo{
				Title:   title,
				Version: version,
			},
			Paths: make(map[string]PathItem),
		},
		components: &Components{
			Schemas:         make(map[string]*OpenAPISchema),
			Responses:       make(map[string]Response),
			Parameters:      make(map[string]Parameter),
			Examples:        make(map[string]Example),
			RequestBodies:   make(map[string]RequestBody),
			Headers:         make(map[string]Header),
			SecuritySchemes: make(map[string]SecurityScheme),
			Links:           make(map[string]Link),
			Callbacks:       make(map[string]Callback),
		},
		tagMap:      make(map[string]*Tag),
		schemaCache: make(map[reflect.Type]*OpenAPISchema),
	}
}

// SetInfo sets the OpenAPI info section
func (b *OpenAPIBuilder) SetInfo(info OpenAPIInfo) *OpenAPIBuilder {
	b.spec.Info = info
	return b
}

// AddServer adds a server to the OpenAPI specification
func (b *OpenAPIBuilder) AddServer(url, description string) *OpenAPIBuilder {
	server := OpenAPIServer{
		URL:         url,
		Description: description,
	}
	b.spec.Servers = append(b.spec.Servers, server)
	return b
}

// AddTag adds a tag to the OpenAPI specification
func (b *OpenAPIBuilder) AddTag(name, description string) *OpenAPIBuilder {
	tag := &Tag{
		Name:        name,
		Description: description,
	}
	b.spec.Tags = append(b.spec.Tags, *tag)
	b.tagMap[name] = tag
	return b
}

// AddSecurityScheme adds a security scheme to the components
func (b *OpenAPIBuilder) AddSecurityScheme(name string, scheme SecurityScheme) *OpenAPIBuilder {
	if b.components.SecuritySchemes == nil {
		b.components.SecuritySchemes = make(map[string]SecurityScheme)
	}
	b.components.SecuritySchemes[name] = scheme
	return b
}

// AddBearerAuth adds JWT Bearer authentication
func (b *OpenAPIBuilder) AddBearerAuth() *OpenAPIBuilder {
	return b.AddSecurityScheme("bearerAuth", SecurityScheme{
		Type:         "http",
		Scheme:       "bearer",
		BearerFormat: "JWT",
		Description:  "JWT Bearer token authentication",
	})
}

// AddAPIKeyAuth adds API Key authentication
func (b *OpenAPIBuilder) AddAPIKeyAuth(name, in string) *OpenAPIBuilder {
	return b.AddSecurityScheme("apiKeyAuth", SecurityScheme{
		Type:        "apiKey",
		Name:        name,
		In:          in,
		Description: "API Key authentication",
	})
}

// FromRouter generates OpenAPI specification from router
func (b *OpenAPIBuilder) FromRouter(router *Router) *OpenAPIBuilder {
	b.router = router
	
	// TODO: Implement route extraction with new interface-based router
	// For now, skip auto-discovery from router routes since the interface doesn't expose them
	// OpenAPI generation will need to work through documentation middleware recording instead
	_ = router // Suppress unused variable warning
	
	b.spec.Components = b.components
	return b
}

// Build returns the completed OpenAPI specification
func (b *OpenAPIBuilder) Build() *OpenAPISpec {
	// Set default responses if none exist
	if b.components.Responses == nil || len(b.components.Responses) == 0 {
		b.addDefaultResponses()
	}
	
	b.spec.Components = b.components
	return b.spec
}

// JSON returns the OpenAPI specification as JSON
func (b *OpenAPIBuilder) JSON() ([]byte, error) {
	spec := b.Build()
	return json.MarshalIndent(spec, "", "  ")
}

// addRouteToSpec adds a route to the OpenAPI specification
func (b *OpenAPIBuilder) addRouteToSpec(route Route) {
	path := b.convertPathToOpenAPI(route.Pattern())
	
	if _, exists := b.spec.Paths[path]; !exists {
		b.spec.Paths[path] = PathItem{}
	}
	
	pathItem := b.spec.Paths[path]
	operation := b.createOperationFromRoute(route)
	
	switch strings.ToLower(route.Method()) {
	case "get":
		pathItem.Get = operation
	case "post":
		pathItem.Post = operation
	case "put":
		pathItem.Put = operation
	case "delete":
		pathItem.Delete = operation
	case "patch":
		pathItem.Patch = operation
	case "options":
		pathItem.Options = operation
	case "head":
		pathItem.Head = operation
	}
	
	b.spec.Paths[path] = pathItem
}

// convertPathToOpenAPI converts Onyx path patterns to OpenAPI format
func (b *OpenAPIBuilder) convertPathToOpenAPI(pattern string) string {
	// Convert {param} to {param}
	// Convert {param:type} to {param}
	re := regexp.MustCompile(`\{([^:}]+):[^}]+\}`)
	pattern = re.ReplaceAllString(pattern, `{$1}`)
	
	return pattern
}

// createOperationFromRoute creates an OpenAPI operation from a route
func (b *OpenAPIBuilder) createOperationFromRoute(route Route) *Operation {
	operation := &Operation{
		OperationID: b.generateOperationID(route),
		Summary:     b.generateSummary(route),
		Parameters:  b.extractParameters(route),
		Responses:   b.generateDefaultResponses(route),
	}
	
	// Add tags based on path
	if tag := b.extractTagFromPath(route.Pattern()); tag != "" {
		operation.Tags = []string{tag}
	}
	
	// Add request body for POST, PUT, PATCH
	if route.Method() == "POST" || route.Method() == "PUT" || route.Method() == "PATCH" {
		operation.RequestBody = b.generateRequestBody()
	}
	
	return operation
}

// generateOperationID generates a unique operation ID
func (b *OpenAPIBuilder) generateOperationID(route Route) string {
	method := strings.ToLower(route.Method())
	path := strings.ReplaceAll(route.Pattern(), "/", "_")
	path = strings.ReplaceAll(path, "{", "")
	path = strings.ReplaceAll(path, "}", "")
	path = regexp.MustCompile(`[^a-zA-Z0-9_]`).ReplaceAllString(path, "")
	
	return method + "_" + path
}

// generateSummary generates a summary for the operation
func (b *OpenAPIBuilder) generateSummary(route Route) string {
	method := strings.ToUpper(route.Method())
	path := route.Pattern()
	
	return fmt.Sprintf("%s %s", method, path)
}

// extractParameters extracts parameters from route pattern
func (b *OpenAPIBuilder) extractParameters(route Route) []Parameter {
	var parameters []Parameter
	
	for _, paramName := range route.ParamNames() {
		param := Parameter{
			Name:        paramName,
			In:          "path",
			Required:    true,
			Description: fmt.Sprintf("Path parameter: %s", paramName),
			Schema: &OpenAPISchema{
				Type: "string",
			},
		}
		
		// Try to infer type from pattern
		if b.isIntParameter(route.Pattern(), paramName) {
			param.Schema.Type = "integer"
			param.Schema.Format = "int64"
		}
		
		parameters = append(parameters, param)
	}
	
	return parameters
}

// isIntParameter checks if a parameter is constrained to integers
func (b *OpenAPIBuilder) isIntParameter(pattern, paramName string) bool {
	re := regexp.MustCompile(fmt.Sprintf(`\{%s:(int|number)\}`, regexp.QuoteMeta(paramName)))
	return re.MatchString(pattern)
}

// extractTagFromPath extracts a tag name from the route path
func (b *OpenAPIBuilder) extractTagFromPath(pattern string) string {
	// Extract first path segment as tag
	parts := strings.Split(strings.Trim(pattern, "/"), "/")
	if len(parts) > 0 && parts[0] != "" {
		// Skip common prefixes
		if parts[0] == "api" && len(parts) > 1 {
			if len(parts) > 2 && (parts[1] == "v1" || parts[1] == "v2") {
				return strings.Title(parts[2])
			}
			return strings.Title(parts[1])
		}
		return strings.Title(parts[0])
	}
	return ""
}

// generateDefaultResponses generates default responses for an operation
func (b *OpenAPIBuilder) generateDefaultResponses(route Route) map[string]Response {
	responses := make(map[string]Response)
	
	// Default success response
	if route.Method() == "POST" {
		responses["201"] = Response{
			Description: "Created",
			Content: map[string]MediaType{
				"application/json": {
					Schema: &OpenAPISchema{
						Type: "object",
						Properties: map[string]*OpenAPISchema{
							"message": {Type: "string"},
							"data":    {Type: "object"},
						},
					},
				},
			},
		}
	} else if route.Method() == "DELETE" {
		responses["204"] = Response{
			Description: "No Content",
		}
	} else {
		responses["200"] = Response{
			Description: "Success",
			Content: map[string]MediaType{
				"application/json": {
					Schema: &OpenAPISchema{
						Type: "object",
						Properties: map[string]*OpenAPISchema{
							"message": {Type: "string"},
							"data":    {Type: "object"},
						},
					},
				},
			},
		}
	}
	
	// Common error responses
	responses["400"] = Response{Description: "Bad Request"}
	responses["401"] = Response{Description: "Unauthorized"}
	responses["403"] = Response{Description: "Forbidden"}
	responses["404"] = Response{Description: "Not Found"}
	responses["500"] = Response{Description: "Internal Server Error"}
	
	return responses
}

// generateRequestBody generates a default request body
func (b *OpenAPIBuilder) generateRequestBody() *RequestBody {
	return &RequestBody{
		Description: "Request body",
		Required:    true,
		Content: map[string]MediaType{
			"application/json": {
				Schema: &OpenAPISchema{
					Type: "object",
				},
			},
		},
	}
}

// addDefaultResponses adds common response schemas to components
func (b *OpenAPIBuilder) addDefaultResponses() {
	b.components.Responses["Success"] = Response{
		Description: "Successful response",
		Content: map[string]MediaType{
			"application/json": {
				Schema: &OpenAPISchema{
					Type: "object",
					Properties: map[string]*OpenAPISchema{
						"message": {Type: "string"},
						"data":    {Type: "object"},
					},
				},
			},
		},
	}
	
	b.components.Responses["Error"] = Response{
		Description: "Error response",
		Content: map[string]MediaType{
			"application/json": {
				Schema: &OpenAPISchema{
					Type: "object",
					Properties: map[string]*OpenAPISchema{
						"error":   {Type: "string"},
						"message": {Type: "string"},
						"code":    {Type: "integer"},
					},
				},
			},
		},
	}
}

// SchemaFromStruct creates an OpenAPI schema from a Go struct
func (b *OpenAPIBuilder) SchemaFromStruct(v interface{}) *OpenAPISchema {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	
	if schema, exists := b.schemaCache[t]; exists {
		return schema
	}
	
	schema := &OpenAPISchema{
		Type:       "object",
		Properties: make(map[string]*OpenAPISchema),
	}
	
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		
		// Skip unexported fields
		if !field.IsExported() {
			continue
		}
		
		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}
		
		fieldName := field.Name
		if jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" {
				fieldName = parts[0]
			}
		}
		
		fieldSchema := b.schemaFromType(field.Type)
		schema.Properties[fieldName] = fieldSchema
	}
	
	b.schemaCache[t] = schema
	return schema
}

// schemaFromType creates a schema from a reflect.Type
func (b *OpenAPIBuilder) schemaFromType(t reflect.Type) *OpenAPISchema {
	switch t.Kind() {
	case reflect.String:
		return &OpenAPISchema{Type: "string"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return &OpenAPISchema{Type: "integer"}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &OpenAPISchema{Type: "integer", Minimum: &[]float64{0}[0]}
	case reflect.Float32, reflect.Float64:
		return &OpenAPISchema{Type: "number"}
	case reflect.Bool:
		return &OpenAPISchema{Type: "boolean"}
	case reflect.Array, reflect.Slice:
		return &OpenAPISchema{
			Type:  "array",
			Items: b.schemaFromType(t.Elem()),
		}
	case reflect.Map:
		return &OpenAPISchema{
			Type:                 "object",
			AdditionalProperties: b.schemaFromType(t.Elem()),
		}
	case reflect.Struct:
		if t == reflect.TypeOf(time.Time{}) {
			return &OpenAPISchema{Type: "string", Format: "date-time"}
		}
		return b.SchemaFromStruct(reflect.Zero(t).Interface())
	case reflect.Ptr:
		schema := b.schemaFromType(t.Elem())
		schema.Nullable = true
		return schema
	default:
		return &OpenAPISchema{Type: "object"}
	}
}

// Utility functions for common OpenAPI patterns

// CreateAPIResponse creates a standard API response schema
func CreateAPIResponse(dataSchema *OpenAPISchema) *OpenAPISchema {
	return &OpenAPISchema{
		Type: "object",
		Properties: map[string]*OpenAPISchema{
			"success": &OpenAPISchema{Type: "boolean"},
			"message": &OpenAPISchema{Type: "string"},
			"data":    dataSchema,
		},
		Required: []string{"success"},
	}
}

// CreateErrorResponse creates a standard error response schema
func CreateErrorResponse() *OpenAPISchema {
	return &OpenAPISchema{
		Type: "object",
		Properties: map[string]*OpenAPISchema{
			"success": &OpenAPISchema{Type: "boolean"},
			"error":   &OpenAPISchema{Type: "string"},
			"message": &OpenAPISchema{Type: "string"},
			"code":    &OpenAPISchema{Type: "integer"},
		},
		Required: []string{"success", "error"},
	}
}

// CreatePaginatedResponse creates a paginated response schema
func CreatePaginatedResponse(itemSchema *OpenAPISchema) *OpenAPISchema {
	return &OpenAPISchema{
		Type: "object",
		Properties: map[string]*OpenAPISchema{
			"success": &OpenAPISchema{Type: "boolean"},
			"data": &OpenAPISchema{
				Type: "array",
				Items: itemSchema,
			},
			"meta": &OpenAPISchema{
				Type: "object",
				Properties: map[string]*OpenAPISchema{
					"current_page": &OpenAPISchema{Type: "integer"},
					"per_page":     &OpenAPISchema{Type: "integer"},
					"total":        &OpenAPISchema{Type: "integer"},
					"total_pages":  &OpenAPISchema{Type: "integer"},
					"has_next":     &OpenAPISchema{Type: "boolean"},
					"has_prev":     &OpenAPISchema{Type: "boolean"},
				},
			},
		},
		Required: []string{"success", "data", "meta"},
	}
}