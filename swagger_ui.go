package onyx

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strings"
	"time"
)

// SwaggerUIConfig configuration for Swagger UI
type SwaggerUIConfig struct {
	Title            string            `json:"title"`
	Path             string            `json:"path"`
	SpecURL          string            `json:"spec_url"`
	DeepLinking      bool              `json:"deep_linking"`
	DisplayOperationId bool            `json:"display_operation_id"`
	DefaultModelsExpandDepth int       `json:"default_models_expand_depth"`
	DefaultModelExpandDepth  int       `json:"default_model_expand_depth"`
	DefaultModelRendering    string    `json:"default_model_rendering"`
	DisplayRequestDuration   bool      `json:"display_request_duration"`
	DocExpansion            string     `json:"doc_expansion"`
	Filter                  bool       `json:"filter"`
	MaxDisplayedTags        int        `json:"max_displayed_tags"`
	ShowExtensions          bool       `json:"show_extensions"`
	ShowCommonExtensions    bool       `json:"show_common_extensions"`
	TryItOutEnabled         bool       `json:"try_it_out_enabled"`
	RequestInterceptor      string     `json:"request_interceptor,omitempty"`
	ResponseInterceptor     string     `json:"response_interceptor,omitempty"`
	Theme                   string     `json:"theme"`
	CustomCSS               string     `json:"custom_css,omitempty"`
	CustomJS                string     `json:"custom_js,omitempty"`
	OAuth                   *OAuthConfig `json:"oauth,omitempty"`
	Plugins                 []string   `json:"plugins,omitempty"`
	Presets                 []string   `json:"presets,omitempty"`
	SupportedSubmitMethods  []string   `json:"supported_submit_methods"`
}

// OAuthConfig OAuth configuration for Swagger UI
type OAuthConfig struct {
	ClientId     string `json:"clientId,omitempty"`
	ClientSecret string `json:"clientSecret,omitempty"`
	Realm        string `json:"realm,omitempty"`
	AppName      string `json:"appName,omitempty"`
	ScopeSeparator string `json:"scopeSeparator,omitempty"`
	Scopes       string `json:"scopes,omitempty"`
	AdditionalQueryStringParams map[string]string `json:"additionalQueryStringParams,omitempty"`
	UseBasicAuthenticationWithAccessCodeGrant bool `json:"useBasicAuthenticationWithAccessCodeGrant,omitempty"`
	UsePkceWithAuthorizationCodeGrant bool `json:"usePkceWithAuthorizationCodeGrant,omitempty"`
}

// SwaggerUIServer serves Swagger UI with enhanced features
type SwaggerUIServer struct {
	config           *SwaggerUIConfig
	docManager       *APIDocumentationManager
	versionedDocs    *VersionedDocumentation
	playground       *APIPlayground
	codeGenerator    *CodeGenerator
	customTemplates  map[string]string
	staticAssets     map[string][]byte
}

// NewSwaggerUIServer creates a new Swagger UI server
func NewSwaggerUIServer(config *SwaggerUIConfig, docManager *APIDocumentationManager) *SwaggerUIServer {
	if config == nil {
		config = DefaultSwaggerUIConfig()
	}

	server := &SwaggerUIServer{
		config:          config,
		docManager:      docManager,
		playground:      NewAPIPlayground(),
		codeGenerator:   NewCodeGenerator(),
		customTemplates: make(map[string]string),
		staticAssets:    make(map[string][]byte),
	}

	if docManager != nil {
		server.versionedDocs = docManager.GetVersionedDocumentation()
	}

	return server
}

// DefaultSwaggerUIConfig returns default Swagger UI configuration
func DefaultSwaggerUIConfig() *SwaggerUIConfig {
	return &SwaggerUIConfig{
		Title:                       "API Documentation",
		Path:                        "/docs",
		SpecURL:                     "/docs/openapi.json",
		DeepLinking:                 true,
		DisplayOperationId:          false,
		DefaultModelsExpandDepth:    1,
		DefaultModelExpandDepth:     1,
		DefaultModelRendering:       "example",
		DisplayRequestDuration:      true,
		DocExpansion:               "list",
		Filter:                     true,
		MaxDisplayedTags:           -1,
		ShowExtensions:             false,
		ShowCommonExtensions:       false,
		TryItOutEnabled:            true,
		Theme:                      "default",
		SupportedSubmitMethods:     []string{"get", "post", "put", "delete", "patch", "head", "options"},
		Plugins:                    []string{"SwaggerUIBundle.plugins.DownloadUrl"},
		Presets:                    []string{"SwaggerUIBundle.presets.apis", "SwaggerUIStandalonePreset"},
	}
}

// ServeHTTP serves Swagger UI and related endpoints
func (sui *SwaggerUIServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	
	// Remove base path
	if strings.HasPrefix(path, sui.config.Path) {
		path = strings.TrimPrefix(path, sui.config.Path)
	}
	
	switch {
	case path == "/" || path == "" || path == "/index.html":
		sui.serveIndex(w, r)
	case path == "/config.json":
		sui.serveConfig(w, r)
	case path == "/versions":
		sui.serveVersions(w, r)
	case strings.HasPrefix(path, "/version/"):
		sui.serveVersionedUI(w, r)
	case path == "/playground":
		sui.servePlayground(w, r)
	case strings.HasPrefix(path, "/playground/"):
		sui.handlePlaygroundRequest(w, r)
	case path == "/code-gen":
		sui.serveCodeGenerator(w, r)
	case strings.HasPrefix(path, "/code-gen/"):
		sui.handleCodeGenRequest(w, r)
	case path == "/health":
		sui.serveHealth(w, r)
	case strings.HasPrefix(path, "/assets/"):
		sui.serveAsset(w, r)
	default:
		http.NotFound(w, r)
	}
}

// serveIndex serves the main Swagger UI index page
func (sui *SwaggerUIServer) serveIndex(w http.ResponseWriter, r *http.Request) {
	// Check for version parameter
	version := r.URL.Query().Get("version")
	specURL := sui.config.SpecURL
	
	if version != "" && sui.versionedDocs != nil {
		specURL = fmt.Sprintf("/docs/%s/openapi.json", version)
	}
	
	tmpl := sui.getIndexTemplate()
	data := sui.getTemplateData(specURL, version)
	
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, data)
}

// serveConfig serves Swagger UI configuration
func (sui *SwaggerUIServer) serveConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sui.config)
}

// serveVersions serves version selection page
func (sui *SwaggerUIServer) serveVersions(w http.ResponseWriter, r *http.Request) {
	if sui.versionedDocs == nil {
		http.Redirect(w, r, sui.config.Path, http.StatusTemporaryRedirect)
		return
	}
	
	versions := sui.versionedDocs.GenerateVersionMatrix()
	
	tmpl := sui.getVersionsTemplate()
	data := map[string]interface{}{
		"Title":    "API Versions",
		"Versions": versions.Versions,
		"BasePath": sui.config.Path,
	}
	
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, data)
}

// serveVersionedUI serves version-specific Swagger UI
func (sui *SwaggerUIServer) serveVersionedUI(w http.ResponseWriter, r *http.Request) {
	version := strings.TrimPrefix(r.URL.Path, sui.config.Path+"/version/")
	version = strings.Split(version, "/")[0]
	
	if sui.versionedDocs == nil {
		http.NotFound(w, r)
		return
	}
	
	specURL := fmt.Sprintf("/docs/%s/openapi.json", version)
	tmpl := sui.getIndexTemplate()
	data := sui.getTemplateData(specURL, version)
	
	// Add version-specific information
	if versionInfo, exists := sui.versionedDocs.versionManager.GetVersion(version); exists {
		data["VersionInfo"] = versionInfo
	}
	
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, data)
}

// servePlayground serves the API playground
func (sui *SwaggerUIServer) servePlayground(w http.ResponseWriter, r *http.Request) {
	tmpl := sui.getPlaygroundTemplate()
	data := map[string]interface{}{
		"Title":   "API Playground",
		"Config":  sui.config,
		"BasePath": sui.config.Path,
	}
	
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, data)
}

// handlePlaygroundRequest handles API playground requests
func (sui *SwaggerUIServer) handlePlaygroundRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		sui.playground.HandleRequest(w, r)
	} else {
		http.NotFound(w, r)
	}
}

// serveCodeGenerator serves the code generator interface
func (sui *SwaggerUIServer) serveCodeGenerator(w http.ResponseWriter, r *http.Request) {
	tmpl := sui.getCodeGenTemplate()
	data := map[string]interface{}{
		"Title":         "Code Generator",
		"Languages":     sui.codeGenerator.GetSupportedLanguages(),
		"BasePath":      sui.config.Path,
		"AvailableSpecs": sui.getAvailableSpecs(),
	}
	
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, data)
}

// handleCodeGenRequest handles code generation requests
func (sui *SwaggerUIServer) handleCodeGenRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		sui.codeGenerator.HandleRequest(w, r, sui.docManager)
	} else {
		http.NotFound(w, r)
	}
}

// serveHealth serves health check endpoint
func (sui *SwaggerUIServer) serveHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":     "healthy",
		"timestamp":  time.Now().Format(time.RFC3339),
		"version":    "1.0.0",
		"features": map[string]bool{
			"swagger_ui":     true,
			"versioning":     sui.versionedDocs != nil,
			"playground":     true,
			"code_generator": true,
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// serveAsset serves static assets
func (sui *SwaggerUIServer) serveAsset(w http.ResponseWriter, r *http.Request) {
	assetPath := strings.TrimPrefix(r.URL.Path, sui.config.Path+"/assets/")
	
	if asset, exists := sui.staticAssets[assetPath]; exists {
		// Determine content type
		contentType := "application/octet-stream"
		if strings.HasSuffix(assetPath, ".css") {
			contentType = "text/css"
		} else if strings.HasSuffix(assetPath, ".js") {
			contentType = "application/javascript"
		} else if strings.HasSuffix(assetPath, ".png") {
			contentType = "image/png"
		} else if strings.HasSuffix(assetPath, ".ico") {
			contentType = "image/x-icon"
		}
		
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Write(asset)
		return
	}
	
	http.NotFound(w, r)
}

// Template methods

// getIndexTemplate returns the main Swagger UI template
func (sui *SwaggerUIServer) getIndexTemplate() *template.Template {
	if customTemplate, exists := sui.customTemplates["index"]; exists {
		return template.Must(template.New("index").Parse(customTemplate))
	}
	
	return template.Must(template.New("index").Parse(swaggerUIIndexTemplate))
}

// getVersionsTemplate returns the versions selection template
func (sui *SwaggerUIServer) getVersionsTemplate() *template.Template {
	if customTemplate, exists := sui.customTemplates["versions"]; exists {
		return template.Must(template.New("versions").Parse(customTemplate))
	}
	
	return template.Must(template.New("versions").Parse(versionsTemplate))
}

// getPlaygroundTemplate returns the playground template
func (sui *SwaggerUIServer) getPlaygroundTemplate() *template.Template {
	if customTemplate, exists := sui.customTemplates["playground"]; exists {
		return template.Must(template.New("playground").Parse(customTemplate))
	}
	
	return template.Must(template.New("playground").Parse(playgroundTemplate))
}

// getCodeGenTemplate returns the code generator template
func (sui *SwaggerUIServer) getCodeGenTemplate() *template.Template {
	if customTemplate, exists := sui.customTemplates["codegen"]; exists {
		return template.Must(template.New("codegen").Parse(customTemplate))
	}
	
	return template.Must(template.New("codegen").Parse(codeGenTemplate))
}

// getTemplateData returns template data for Swagger UI
func (sui *SwaggerUIServer) getTemplateData(specURL, version string) map[string]interface{} {
	config := *sui.config
	config.SpecURL = specURL
	
	data := map[string]interface{}{
		"Title":   config.Title,
		"Config":  config,
		"Version": version,
		"BasePath": sui.config.Path,
		"Features": map[string]bool{
			"Versioning":    sui.versionedDocs != nil,
			"Playground":    true,
			"CodeGenerator": true,
		},
	}
	
	if version != "" {
		data["CurrentVersion"] = version
		if sui.versionedDocs != nil {
			versions := sui.versionedDocs.versionManager.GetAllVersions()
			var sortedVersions []string
			for v := range versions {
				sortedVersions = append(sortedVersions, v)
			}
			sort.Strings(sortedVersions)
			data["AvailableVersions"] = sortedVersions
		}
	}
	
	return data
}

// getAvailableSpecs returns available OpenAPI specifications
func (sui *SwaggerUIServer) getAvailableSpecs() []map[string]string {
	specs := []map[string]string{
		{
			"name": "Default",
			"url":  "/docs/openapi.json",
		},
	}
	
	if sui.versionedDocs != nil {
		versions := sui.versionedDocs.versionManager.GetAllVersions()
		for version := range versions {
			specs = append(specs, map[string]string{
				"name": fmt.Sprintf("Version %s", version),
				"url":  fmt.Sprintf("/docs/%s/openapi.json", version),
			})
		}
	}
	
	return specs
}

// Customization methods

// SetCustomTemplate sets a custom template
func (sui *SwaggerUIServer) SetCustomTemplate(name, content string) {
	sui.customTemplates[name] = content
}

// SetCustomAsset sets a custom static asset
func (sui *SwaggerUIServer) SetCustomAsset(path string, content []byte) {
	sui.staticAssets[path] = content
}

// AddCustomCSS adds custom CSS
func (sui *SwaggerUIServer) AddCustomCSS(css string) {
	if sui.config.CustomCSS == "" {
		sui.config.CustomCSS = css
	} else {
		sui.config.CustomCSS += "\n" + css
	}
}

// AddCustomJS adds custom JavaScript
func (sui *SwaggerUIServer) AddCustomJS(js string) {
	if sui.config.CustomJS == "" {
		sui.config.CustomJS = js
	} else {
		sui.config.CustomJS += "\n" + js
	}
}

// SetOAuthConfig sets OAuth configuration
func (sui *SwaggerUIServer) SetOAuthConfig(config *OAuthConfig) {
	sui.config.OAuth = config
}

// SetTheme sets the UI theme
func (sui *SwaggerUIServer) SetTheme(theme string) {
	sui.config.Theme = theme
}

// Enhanced features

// EnableVersionComparison enables version comparison features
func (sui *SwaggerUIServer) EnableVersionComparison() {
	sui.AddCustomJS(`
		// Version comparison functionality
		function compareVersions(v1, v2) {
			fetch('/docs/api/compare/' + v1 + '/' + v2)
				.then(response => response.json())
				.then(data => {
					displayComparison(data);
				});
		}
		
		function displayComparison(data) {
			// Display comparison results
			console.log('Version comparison:', data);
		}
	`)
}

// EnableExportFeatures enables documentation export features
func (sui *SwaggerUIServer) EnableExportFeatures() {
	sui.AddCustomJS(`
		// Export functionality
		function exportDocumentation(format) {
			const url = '/docs/openapi.json';
			if (format === 'yaml') {
				// Convert to YAML (would need yaml library)
				fetch(url).then(r => r.json()).then(spec => {
					// Export as YAML
				});
			} else {
				window.open(url);
			}
		}
	`)
}

// EnableDarkMode enables dark mode support
func (sui *SwaggerUIServer) EnableDarkMode() {
	sui.AddCustomCSS(`
		@media (prefers-color-scheme: dark) {
			.swagger-ui {
				filter: invert(1) hue-rotate(180deg);
			}
			.swagger-ui img {
				filter: invert(1) hue-rotate(180deg);
			}
		}
		
		.dark-mode .swagger-ui {
			filter: invert(1) hue-rotate(180deg);
		}
		.dark-mode .swagger-ui img {
			filter: invert(1) hue-rotate(180deg);
		}
	`)
	
	sui.AddCustomJS(`
		// Dark mode toggle
		function toggleDarkMode() {
			document.body.classList.toggle('dark-mode');
			localStorage.setItem('darkMode', document.body.classList.contains('dark-mode'));
		}
		
		// Load dark mode preference
		if (localStorage.getItem('darkMode') === 'true') {
			document.body.classList.add('dark-mode');
		}
	`)
}

// Router integration

// RegisterSwaggerUIRoutes registers Swagger UI routes with the router
func RegisterSwaggerUIRoutes(app *Application, server *SwaggerUIServer) {
	basePath := server.config.Path
	
	// Register main handler
	app.Any(basePath+"/*", func(c *Context) error {
		server.ServeHTTP(c.ResponseWriter, c.Request)
		return nil
	})
	
	// Register root path
	app.Get(basePath, func(c *Context) error {
		server.ServeHTTP(c.ResponseWriter, c.Request)
		return nil
	})
}

// Factory function

// CreateEnhancedSwaggerUI creates a fully configured Swagger UI server
func CreateEnhancedSwaggerUI(
	docManager *APIDocumentationManager,
	config *SwaggerUIConfig,
) *SwaggerUIServer {
	if config == nil {
		config = DefaultSwaggerUIConfig()
	}
	
	server := NewSwaggerUIServer(config, docManager)
	
	// Enable enhanced features
	server.EnableVersionComparison()
	server.EnableExportFeatures()
	server.EnableDarkMode()
	
	// Add custom branding
	server.AddCustomCSS(`
		.topbar {
			background-color: #1f1f1f;
		}
		.swagger-ui .topbar .link {
			color: #fff;
		}
	`)
	
	return server
}