package template

import (
	"context"
	"html/template"
	"io"
	"time"
)

// Engine defines the interface for template engines
type Engine interface {
	// Template loading and management
	LoadTemplates(ctx context.Context) error
	LoadFromFS(ctx context.Context, fs FileSystem) error
	LoadFromPaths(ctx context.Context, paths []string) error
	ReloadTemplates(ctx context.Context) error
	
	// Template rendering
	Render(ctx context.Context, name string, data Data) (string, error)
	RenderTo(ctx context.Context, w io.Writer, name string, data Data) error
	RenderWithLayout(ctx context.Context, name, layout string, data Data) (string, error)
	RenderToWithLayout(ctx context.Context, w io.Writer, name, layout string, data Data) error
	
	// Template existence and metadata
	HasTemplate(name string) bool
	GetTemplateNames() []string
	GetTemplateInfo(name string) (*TemplateInfo, error)
	
	// Function management
	AddFunction(name string, fn interface{}) Engine
	AddFunctions(funcs FuncMap) Engine
	RemoveFunction(name string) Engine
	GetFunctions() FuncMap
	
	// Configuration
	SetDelimiters(left, right string) Engine
	SetOptions(opts *Options) Engine
	GetOptions() *Options
	
	// Hot reload and watching
	EnableHotReload(ctx context.Context) error
	DisableHotReload() error
	IsHotReloadEnabled() bool
	
	// Performance and caching
	ClearCache() error
	GetCacheStats() *CacheStats
	Precompile(ctx context.Context, names ...string) error
	
	// Error handling
	SetErrorHandler(handler ErrorHandler) Engine
	GetErrorHandler() ErrorHandler
	
	// Lifecycle
	Close() error
}

// Manager manages multiple template engines
type Manager interface {
	// Engine management
	RegisterEngine(name string, engine Engine) error
	GetEngine(name string) (Engine, error)
	SetDefaultEngine(name string) error
	GetDefaultEngine() Engine
	ListEngines() []string
	
	// Global operations
	Render(ctx context.Context, name string, data Data, engineName ...string) (string, error)
	RenderTo(ctx context.Context, w io.Writer, name string, data Data, engineName ...string) error
	
	// Global configuration
	SetGlobalFunctions(funcs FuncMap)
	AddGlobalFunction(name string, fn interface{})
	SetGlobalOptions(opts *Options)
	
	// Lifecycle
	LoadAll(ctx context.Context) error
	ReloadAll(ctx context.Context) error
	Close() error
}

// Renderer provides high-level rendering operations
type Renderer interface {
	// Basic rendering
	RenderTemplate(ctx context.Context, name string, data Data) (*RenderResult, error)
	RenderString(ctx context.Context, tmplStr string, data Data) (*RenderResult, error)
	RenderFile(ctx context.Context, path string, data Data) (*RenderResult, error)
	
	// Batch rendering
	RenderBatch(ctx context.Context, requests []RenderRequest) ([]RenderResult, error)
	
	// Streaming
	RenderStream(ctx context.Context, w io.Writer, name string, data Data) error
	
	// Email templates
	RenderEmail(ctx context.Context, name string, data Data) (*EmailTemplate, error)
	
	// Partial rendering
	RenderPartial(ctx context.Context, name string, data Data) (string, error)
	
	// Component rendering
	RenderComponent(ctx context.Context, component Component, data Data) (string, error)
}

// FileSystem interface for template file access
type FileSystem interface {
	// File operations
	Open(name string) (File, error)
	ReadFile(name string) ([]byte, error)
	Glob(pattern string) ([]string, error)
	
	// Directory operations
	ReadDir(name string) ([]DirEntry, error)
	Walk(root string, fn WalkFunc) error
	
	// File information
	Stat(name string) (FileInfo, error)
	IsDir(name string) bool
	Exists(name string) bool
	
	// Watching (for hot reload)
	Watch(path string, handler WatchHandler) error
	Unwatch(path string) error
}

// File represents a template file
type File interface {
	io.ReadCloser
	Name() string
	Size() int64
	ModTime() time.Time
}

// DirEntry represents a directory entry
type DirEntry interface {
	Name() string
	IsDir() bool
	Type() FileMode
	Info() (FileInfo, error)
}

// FileInfo represents file metadata
type FileInfo interface {
	Name() string
	Size() int64
	Mode() FileMode
	ModTime() time.Time
	IsDir() bool
}

// WalkFunc is called for each file during directory traversal
type WalkFunc func(path string, info FileInfo, err error) error

// WatchHandler handles file system events
type WatchHandler func(event WatchEvent)

// Compiler compiles templates for better performance
type Compiler interface {
	// Compilation
	Compile(ctx context.Context, source string) (*CompiledTemplate, error)
	CompileFile(ctx context.Context, path string) (*CompiledTemplate, error)
	CompileBatch(ctx context.Context, sources map[string]string) (map[string]*CompiledTemplate, error)
	
	// Template analysis
	Analyze(ctx context.Context, source string) (*TemplateAnalysis, error)
	ExtractDependencies(ctx context.Context, source string) ([]string, error)
	
	// Optimization
	Optimize(ctx context.Context, compiled *CompiledTemplate) (*CompiledTemplate, error)
	EnableOptimizations(opts OptimizationOptions)
	
	// Caching
	SetCache(cache CompilerCache)
	ClearCache()
}

// Cache provides template caching capabilities
type Cache interface {
	// Basic operations
	Get(key string) (*CachedTemplate, bool)
	Set(key string, template *CachedTemplate, ttl time.Duration)
	Delete(key string)
	Clear()
	
	// Statistics
	Stats() *CacheStats
	
	// Configuration
	SetMaxSize(size int)
	SetDefaultTTL(ttl time.Duration)
	
	// Lifecycle
	Close() error
}

// CompilerCache caches compiled templates
type CompilerCache interface {
	Get(key string) (*CompiledTemplate, bool)
	Set(key string, template *CompiledTemplate)
	Delete(key string)
	Clear()
	Stats() *CacheStats
}

// Loader loads templates from various sources
type Loader interface {
	// Loading from filesystem
	LoadFromDirectory(ctx context.Context, dir string) (map[string]string, error)
	LoadFromFiles(ctx context.Context, files []string) (map[string]string, error)
	LoadFromFS(ctx context.Context, fs FileSystem, root string) (map[string]string, error)
	
	// Loading from other sources
	LoadFromMap(templates map[string]string) error
	LoadFromConfig(ctx context.Context, config LoaderConfig) (map[string]string, error)
	
	// Pattern matching
	LoadByPattern(ctx context.Context, pattern string) (map[string]string, error)
	LoadByExtension(ctx context.Context, dir string, ext string) (map[string]string, error)
	
	// Watching for changes
	Watch(ctx context.Context, handler func(name, content string)) error
	StopWatching()
}

// Validator validates template syntax and structure
type Validator interface {
	// Syntax validation
	ValidateSyntax(ctx context.Context, source string) error
	ValidateFile(ctx context.Context, path string) error
	ValidateBatch(ctx context.Context, templates map[string]string) map[string]error
	
	// Semantic validation
	ValidateVariables(ctx context.Context, source string, variables []string) error
	ValidateFunctions(ctx context.Context, source string, functions FuncMap) error
	ValidateLayouts(ctx context.Context, template, layout string) error
	
	// Security validation
	ValidateSecurity(ctx context.Context, source string) error
	CheckForVulnerabilities(ctx context.Context, source string) []SecurityIssue
	
	// Best practices
	CheckBestPractices(ctx context.Context, source string) []BestPracticeIssue
}

// ErrorHandler handles template errors
type ErrorHandler interface {
	HandleError(ctx context.Context, err error, context ErrorContext) error
	HandleRenderError(ctx context.Context, err error, template string, data Data) error
	HandleLoadError(ctx context.Context, err error, path string) error
}

// Component represents a reusable template component
type Component interface {
	// Basic information
	GetName() string
	GetTemplate() string
	GetSchema() *ComponentSchema
	
	// Rendering
	Render(ctx context.Context, data Data) (string, error)
	RenderTo(ctx context.Context, w io.Writer, data Data) error
	
	// Validation
	ValidateData(data Data) error
	GetDefaultData() Data
	
	// Dependencies
	GetDependencies() []string
	GetSlots() []string
}

// Data represents template data
type Data map[string]interface{}

// FuncMap represents template functions
type FuncMap map[string]interface{}

// Options holds template engine configuration
type Options struct {
	// Path configuration
	ViewsPath    string   `json:"views_path"`
	LayoutsPath  string   `json:"layouts_path"`
	PartialsPath string   `json:"partials_path"`
	Extensions   []string `json:"extensions"`
	
	// Delimiters
	LeftDelim  string `json:"left_delim"`
	RightDelim string `json:"right_delim"`
	
	// Caching
	EnableCache    bool          `json:"enable_cache"`
	CacheSize      int           `json:"cache_size"`
	CacheTTL       time.Duration `json:"cache_ttl"`
	
	// Hot reload
	EnableHotReload bool `json:"enable_hot_reload"`
	WatchPaths      []string `json:"watch_paths"`
	
	// Performance
	EnableOptimization bool `json:"enable_optimization"`
	PrecompileAll      bool `json:"precompile_all"`
	EnableMinification bool `json:"enable_minification"`
	
	// Security
	AutoEscape     bool     `json:"auto_escape"`
	AllowedDomains []string `json:"allowed_domains"`
	CSPEnabled     bool     `json:"csp_enabled"`
	
	// Debug
	Debug         bool `json:"debug"`
	StrictMode    bool `json:"strict_mode"`
	ShowErrors    bool `json:"show_errors"`
	EnableProfiling bool `json:"enable_profiling"`
	
	// Layout
	DefaultLayout string            `json:"default_layout"`
	LayoutData    map[string]interface{} `json:"layout_data"`
	
	// Custom options
	Custom map[string]interface{} `json:"custom"`
}

// TemplateInfo contains template metadata
type TemplateInfo struct {
	Name         string            `json:"name"`
	Path         string            `json:"path"`
	Size         int64             `json:"size"`
	ModTime      time.Time         `json:"mod_time"`
	Dependencies []string          `json:"dependencies"`
	Variables    []string          `json:"variables"`
	Functions    []string          `json:"functions"`
	Layouts      []string          `json:"layouts"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// CacheStats provides cache statistics
type CacheStats struct {
	Hits          int64         `json:"hits"`
	Misses        int64         `json:"misses"`
	Size          int           `json:"size"`
	MaxSize       int           `json:"max_size"`
	HitRatio      float64       `json:"hit_ratio"`
	AverageLoadTime time.Duration `json:"average_load_time"`
	TotalLoadTime time.Duration   `json:"total_load_time"`
	EvictionCount int64         `json:"eviction_count"`
}

// RenderRequest represents a render request
type RenderRequest struct {
	Template string `json:"template"`
	Layout   string `json:"layout,omitempty"`
	Data     Data   `json:"data"`
	Options  *RenderOptions `json:"options,omitempty"`
}

// RenderResult represents a render result
type RenderResult struct {
	Content     string        `json:"content"`
	Error       error         `json:"error,omitempty"`
	Duration    time.Duration `json:"duration"`
	Size        int           `json:"size"`
	Template    string        `json:"template"`
	CacheHit    bool          `json:"cache_hit"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// RenderOptions provides rendering options
type RenderOptions struct {
	Layout      string            `json:"layout,omitempty"`
	Minify      bool              `json:"minify"`
	Compress    bool              `json:"compress"`
	Headers     map[string]string `json:"headers,omitempty"`
	StatusCode  int               `json:"status_code,omitempty"`
	ContentType string            `json:"content_type,omitempty"`
}

// EmailTemplate represents an email template
type EmailTemplate struct {
	Subject   string `json:"subject"`
	HTMLBody  string `json:"html_body"`
	TextBody  string `json:"text_body"`
	Headers   map[string]string `json:"headers,omitempty"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

// CompiledTemplate represents a compiled template
type CompiledTemplate struct {
	Name         string                 `json:"name"`
	Source       string                 `json:"source"`
	Template     *template.Template     `json:"-"`
	Dependencies []string               `json:"dependencies"`
	Variables    []string               `json:"variables"`
	Functions    []string               `json:"functions"`
	CompiledAt   time.Time              `json:"compiled_at"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// CachedTemplate represents a cached template
type CachedTemplate struct {
	Template  *template.Template `json:"-"`
	Content   string             `json:"content"`
	ExpiresAt time.Time          `json:"expires_at"`
	Hits      int64              `json:"hits"`
	Size      int64              `json:"size"`
	CreatedAt time.Time          `json:"created_at"`
}

// TemplateAnalysis provides template analysis results
type TemplateAnalysis struct {
	Variables     []string          `json:"variables"`
	Functions     []string          `json:"functions"`
	Dependencies  []string          `json:"dependencies"`
	Complexity    int               `json:"complexity"`
	Blocks        []string          `json:"blocks"`
	Includes      []string          `json:"includes"`
	Extends       string            `json:"extends,omitempty"`
	Issues        []AnalysisIssue   `json:"issues,omitempty"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// AnalysisIssue represents an analysis issue
type AnalysisIssue struct {
	Type        string `json:"type"`
	Level       string `json:"level"`
	Message     string `json:"message"`
	Line        int    `json:"line"`
	Column      int    `json:"column"`
	Suggestion  string `json:"suggestion,omitempty"`
}

// SecurityIssue represents a security issue
type SecurityIssue struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Message     string `json:"message"`
	Line        int    `json:"line"`
	Column      int    `json:"column"`
	Mitigation  string `json:"mitigation"`
}

// BestPracticeIssue represents a best practice issue
type BestPracticeIssue struct {
	Rule        string `json:"rule"`
	Level       string `json:"level"`
	Message     string `json:"message"`
	Line        int    `json:"line"`
	Column      int    `json:"column"`
	Improvement string `json:"improvement"`
}

// ComponentSchema defines component data schema
type ComponentSchema struct {
	Properties map[string]PropertySchema `json:"properties"`
	Required   []string                  `json:"required"`
	Optional   []string                  `json:"optional"`
}

// PropertySchema defines property schema
type PropertySchema struct {
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Default     interface{} `json:"default,omitempty"`
	Required    bool        `json:"required"`
	Validation  interface{} `json:"validation,omitempty"`
}

// LoaderConfig configures template loading
type LoaderConfig struct {
	Sources     []SourceConfig `json:"sources"`
	Extensions  []string       `json:"extensions"`
	Exclude     []string       `json:"exclude"`
	Recursive   bool           `json:"recursive"`
	FollowLinks bool           `json:"follow_links"`
}

// SourceConfig configures a template source
type SourceConfig struct {
	Type   string                 `json:"type"`
	Path   string                 `json:"path"`
	Config map[string]interface{} `json:"config"`
}

// OptimizationOptions configures template optimization
type OptimizationOptions struct {
	MinifyHTML       bool `json:"minify_html"`
	MinifyCSS        bool `json:"minify_css"`
	MinifyJS         bool `json:"minify_js"`
	RemoveComments   bool `json:"remove_comments"`
	RemoveWhitespace bool `json:"remove_whitespace"`
	OptimizeImages   bool `json:"optimize_images"`
	InlineCSS        bool `json:"inline_css"`
	InlineJS         bool `json:"inline_js"`
}

// ErrorContext provides error context information
type ErrorContext struct {
	Template string                 `json:"template"`
	Line     int                    `json:"line"`
	Column   int                    `json:"column"`
	Data     Data                   `json:"data"`
	Context  map[string]interface{} `json:"context"`
}

// WatchEvent represents a file system event
type WatchEvent struct {
	Type string    `json:"type"`
	Path string    `json:"path"`
	Time time.Time `json:"time"`
}

// FileMode represents file permissions
type FileMode uint32

// Default configurations
func DefaultOptions() *Options {
	return &Options{
		ViewsPath:           "resources/views",
		LayoutsPath:         "resources/views/layouts",
		PartialsPath:        "resources/views/partials",
		Extensions:          []string{".html", ".tmpl", ".tpl"},
		LeftDelim:           "{{",
		RightDelim:          "}}",
		EnableCache:         true,
		CacheSize:           1000,
		CacheTTL:            1 * time.Hour,
		EnableHotReload:     false,
		EnableOptimization:  true,
		PrecompileAll:       false,
		EnableMinification:  false,
		AutoEscape:          true,
		Debug:               false,
		StrictMode:          false,
		ShowErrors:          true,
		EnableProfiling:     false,
		DefaultLayout:       "app",
		LayoutData:          make(map[string]interface{}),
		Custom:              make(map[string]interface{}),
	}
}

// Helper functions for working with Data
func (d Data) Get(key string) interface{} {
	return d[key]
}

func (d Data) GetString(key string) string {
	if v, ok := d[key].(string); ok {
		return v
	}
	return ""
}

func (d Data) GetInt(key string) int {
	if v, ok := d[key].(int); ok {
		return v
	}
	return 0
}

func (d Data) GetBool(key string) bool {
	if v, ok := d[key].(bool); ok {
		return v
	}
	return false
}

func (d Data) Set(key string, value interface{}) {
	d[key] = value
}

func (d Data) Has(key string) bool {
	_, exists := d[key]
	return exists
}

func (d Data) Merge(other Data) Data {
	result := make(Data)
	for k, v := range d {
		result[k] = v
	}
	for k, v := range other {
		result[k] = v
	}
	return result
}