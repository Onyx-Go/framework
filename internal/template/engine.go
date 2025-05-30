package template

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// DefaultEngine implements the Engine interface using Go's html/template
type DefaultEngine struct {
	templates   map[string]*template.Template
	compiled    map[string]*CompiledTemplate
	options     *Options
	functions   FuncMap
	cache       Cache
	loader      Loader
	validator   Validator
	compiler    Compiler
	fileSystem  FileSystem
	errorHandler ErrorHandler
	hotReload   *HotReloader
	mutex       sync.RWMutex
	stats       *EngineStats
}

// EngineStats tracks engine performance metrics
type EngineStats struct {
	RenderCount     int64         `json:"render_count"`
	TotalRenderTime time.Duration `json:"total_render_time"`
	AverageRenderTime time.Duration `json:"average_render_time"`
	TemplateCount   int           `json:"template_count"`
	CacheStats      *CacheStats   `json:"cache_stats"`
	ErrorCount      int64         `json:"error_count"`
	LastReload      time.Time     `json:"last_reload"`
	LoadCount       int64         `json:"load_count"`
}

// NewEngine creates a new template engine
func NewEngine(opts *Options) *DefaultEngine {
	if opts == nil {
		opts = DefaultOptions()
	}
	
	engine := &DefaultEngine{
		templates:  make(map[string]*template.Template),
		compiled:   make(map[string]*CompiledTemplate),
		options:    opts,
		functions:  make(FuncMap),
		fileSystem: NewOSFileSystem(),
		stats:      &EngineStats{},
	}
	
	// Initialize components
	if opts.EnableCache {
		engine.cache = NewLRUCache(opts.CacheSize, opts.CacheTTL)
	}
	
	engine.loader = NewDefaultLoader(engine.fileSystem)
	engine.validator = NewDefaultValidator()
	engine.compiler = NewDefaultCompiler()
	engine.errorHandler = NewDefaultErrorHandler()
	
	// Register default functions
	engine.registerDefaultFunctions()
	
	return engine
}

// LoadTemplates loads all templates from configured paths
func (e *DefaultEngine) LoadTemplates(ctx context.Context) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	
	defer func() {
		e.stats.LoadCount++
		e.stats.LastReload = time.Now()
	}()
	
	// Clear existing templates
	e.templates = make(map[string]*template.Template)
	e.compiled = make(map[string]*CompiledTemplate)
	
	// Load templates from views path
	if e.options.ViewsPath != "" {
		if err := e.loadFromDirectory(ctx, e.options.ViewsPath); err != nil {
			return fmt.Errorf("failed to load templates from views path: %w", err)
		}
	}
	
	// Load layouts
	if e.options.LayoutsPath != "" {
		if err := e.loadFromDirectory(ctx, e.options.LayoutsPath); err != nil {
			return fmt.Errorf("failed to load layouts: %w", err)
		}
	}
	
	// Load partials
	if e.options.PartialsPath != "" {
		if err := e.loadFromDirectory(ctx, e.options.PartialsPath); err != nil {
			return fmt.Errorf("failed to load partials: %w", err)
		}
	}
	
	// Precompile if enabled
	if e.options.PrecompileAll {
		if err := e.precompileAll(ctx); err != nil {
			return fmt.Errorf("failed to precompile templates: %w", err)
		}
	}
	
	e.stats.TemplateCount = len(e.templates)
	
	return nil
}

// LoadFromFS loads templates from a file system
func (e *DefaultEngine) LoadFromFS(ctx context.Context, fileSystem FileSystem) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	
	e.fileSystem = fileSystem
	return e.LoadTemplates(ctx)
}

// LoadFromPaths loads templates from specific paths
func (e *DefaultEngine) LoadFromPaths(ctx context.Context, paths []string) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	
	for _, path := range paths {
		if err := e.loadFromDirectory(ctx, path); err != nil {
			return fmt.Errorf("failed to load templates from path %s: %w", path, err)
		}
	}
	
	return nil
}

// ReloadTemplates reloads all templates
func (e *DefaultEngine) ReloadTemplates(ctx context.Context) error {
	return e.LoadTemplates(ctx)
}

// Render renders a template with data
func (e *DefaultEngine) Render(ctx context.Context, name string, data Data) (string, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		e.mutex.Lock()
		e.stats.RenderCount++
		e.stats.TotalRenderTime += duration
		e.stats.AverageRenderTime = e.stats.TotalRenderTime / time.Duration(e.stats.RenderCount)
		e.mutex.Unlock()
	}()
	
	// Check cache first
	if e.cache != nil {
		cacheKey := e.getCacheKey(name, data)
		if cached, exists := e.cache.Get(cacheKey); exists {
			return cached.Content, nil
		}
	}
	
	// Get template
	tmpl, err := e.getTemplate(name)
	if err != nil {
		e.mutex.Lock()
		e.stats.ErrorCount++
		e.mutex.Unlock()
		return "", e.errorHandler.HandleRenderError(ctx, err, name, data)
	}
	
	// Prepare data
	templateData := e.prepareData(data)
	
	// Render template
	var buf strings.Builder
	if err := tmpl.Execute(&buf, templateData); err != nil {
		e.mutex.Lock()
		e.stats.ErrorCount++
		e.mutex.Unlock()
		return "", e.errorHandler.HandleRenderError(ctx, err, name, data)
	}
	
	result := buf.String()
	
	// Cache result
	if e.cache != nil {
		cacheKey := e.getCacheKey(name, data)
		cached := &CachedTemplate{
			Content:   result,
			ExpiresAt: time.Now().Add(e.options.CacheTTL),
			Hits:      1,
			Size:      int64(len(result)),
			CreatedAt: time.Now(),
		}
		e.cache.Set(cacheKey, cached, e.options.CacheTTL)
	}
	
	return result, nil
}

// RenderTo renders a template directly to a writer
func (e *DefaultEngine) RenderTo(ctx context.Context, w io.Writer, name string, data Data) error {
	content, err := e.Render(ctx, name, data)
	if err != nil {
		return err
	}
	
	_, err = w.Write([]byte(content))
	return err
}

// RenderWithLayout renders a template with a specific layout
func (e *DefaultEngine) RenderWithLayout(ctx context.Context, name, layout string, data Data) (string, error) {
	// Get template
	tmpl, err := e.getTemplate(name)
	if err != nil {
		return "", err
	}
	
	// Prepare data
	templateData := e.prepareData(data)
	
	// Render with layout
	var buf strings.Builder
	if err := tmpl.ExecuteTemplate(&buf, layout, templateData); err != nil {
		return "", e.errorHandler.HandleRenderError(ctx, err, name, data)
	}
	
	return buf.String(), nil
}

// RenderToWithLayout renders a template with layout directly to a writer
func (e *DefaultEngine) RenderToWithLayout(ctx context.Context, w io.Writer, name, layout string, data Data) error {
	content, err := e.RenderWithLayout(ctx, name, layout, data)
	if err != nil {
		return err
	}
	
	_, err = w.Write([]byte(content))
	return err
}

// HasTemplate checks if a template exists
func (e *DefaultEngine) HasTemplate(name string) bool {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	
	_, exists := e.templates[name]
	return exists
}

// GetTemplateNames returns all template names
func (e *DefaultEngine) GetTemplateNames() []string {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	
	names := make([]string, 0, len(e.templates))
	for name := range e.templates {
		names = append(names, name)
	}
	
	return names
}

// GetTemplateInfo returns template information
func (e *DefaultEngine) GetTemplateInfo(name string) (*TemplateInfo, error) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	
	tmpl, exists := e.templates[name]
	if !exists {
		return nil, fmt.Errorf("template %s not found", name)
	}
	
	// Build template info
	info := &TemplateInfo{
		Name:      name,
		Variables: e.extractVariables(tmpl),
		Functions: e.extractFunctions(tmpl),
		Metadata:  make(map[string]interface{}),
	}
	
	return info, nil
}

// AddFunction adds a template function
func (e *DefaultEngine) AddFunction(name string, fn interface{}) Engine {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	
	e.functions[name] = fn
	return e
}

// AddFunctions adds multiple template functions
func (e *DefaultEngine) AddFunctions(funcs FuncMap) Engine {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	
	for name, fn := range funcs {
		e.functions[name] = fn
	}
	return e
}

// RemoveFunction removes a template function
func (e *DefaultEngine) RemoveFunction(name string) Engine {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	
	delete(e.functions, name)
	return e
}

// GetFunctions returns all template functions
func (e *DefaultEngine) GetFunctions() FuncMap {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	
	funcs := make(FuncMap)
	for name, fn := range e.functions {
		funcs[name] = fn
	}
	
	return funcs
}

// SetDelimiters sets template delimiters
func (e *DefaultEngine) SetDelimiters(left, right string) Engine {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	
	e.options.LeftDelim = left
	e.options.RightDelim = right
	return e
}

// SetOptions sets engine options
func (e *DefaultEngine) SetOptions(opts *Options) Engine {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	
	e.options = opts
	return e
}

// GetOptions returns engine options
func (e *DefaultEngine) GetOptions() *Options {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	
	return e.options
}

// EnableHotReload enables hot reloading
func (e *DefaultEngine) EnableHotReload(ctx context.Context) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	
	if e.hotReload != nil {
		return nil // Already enabled
	}
	
	hotReloader := NewHotReloader(e.fileSystem, func() {
		e.ReloadTemplates(ctx)
	})
	
	// Watch configured paths
	paths := []string{e.options.ViewsPath, e.options.LayoutsPath, e.options.PartialsPath}
	paths = append(paths, e.options.WatchPaths...)
	
	for _, path := range paths {
		if path != "" {
			if err := hotReloader.Watch(path); err != nil {
				return fmt.Errorf("failed to watch path %s: %w", path, err)
			}
		}
	}
	
	e.hotReload = hotReloader
	e.options.EnableHotReload = true
	
	return nil
}

// DisableHotReload disables hot reloading
func (e *DefaultEngine) DisableHotReload() error {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	
	if e.hotReload != nil {
		e.hotReload.Stop()
		e.hotReload = nil
	}
	
	e.options.EnableHotReload = false
	return nil
}

// IsHotReloadEnabled returns true if hot reload is enabled
func (e *DefaultEngine) IsHotReloadEnabled() bool {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	
	return e.hotReload != nil
}

// ClearCache clears the template cache
func (e *DefaultEngine) ClearCache() error {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	
	if e.cache != nil {
		e.cache.Clear()
	}
	
	return nil
}

// GetCacheStats returns cache statistics
func (e *DefaultEngine) GetCacheStats() *CacheStats {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	
	if e.cache != nil {
		return e.cache.Stats()
	}
	
	return &CacheStats{}
}

// Precompile precompiles specified templates
func (e *DefaultEngine) Precompile(ctx context.Context, names ...string) error {
	if len(names) == 0 {
		names = e.GetTemplateNames()
	}
	
	e.mutex.Lock()
	defer e.mutex.Unlock()
	
	for _, name := range names {
		if err := e.precompileTemplateUnlocked(ctx, name); err != nil {
			return fmt.Errorf("failed to precompile template %s: %w", name, err)
		}
	}
	
	return nil
}

// SetErrorHandler sets the error handler
func (e *DefaultEngine) SetErrorHandler(handler ErrorHandler) Engine {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	
	e.errorHandler = handler
	return e
}

// GetErrorHandler returns the error handler
func (e *DefaultEngine) GetErrorHandler() ErrorHandler {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	
	return e.errorHandler
}

// Close closes the engine and releases resources
func (e *DefaultEngine) Close() error {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	
	if e.hotReload != nil {
		e.hotReload.Stop()
		e.hotReload = nil
	}
	
	if e.cache != nil {
		e.cache.Close()
	}
	
	return nil
}

// Private methods

func (e *DefaultEngine) loadFromDirectory(ctx context.Context, dir string) error {
	return e.fileSystem.Walk(dir, func(path string, info FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if info.IsDir() {
			return nil
		}
		
		// Check if file has valid extension
		if !e.hasValidExtension(path) {
			return nil
		}
		
		// Read template content
		content, err := e.fileSystem.ReadFile(path)
		if err != nil {
			return err
		}
		
		// Get template name
		name := e.getTemplateName(path, dir)
		
		// Validate template
		if e.validator != nil {
			if err := e.validator.ValidateSyntax(ctx, string(content)); err != nil {
				return e.errorHandler.HandleLoadError(ctx, err, path)
			}
		}
		
		// Parse template
		tmpl := template.New(name).Funcs(template.FuncMap(e.functions))
		tmpl.Delims(e.options.LeftDelim, e.options.RightDelim)
		
		// Load layouts if available
		if e.options.LayoutsPath != "" {
			layoutFiles, err := e.getLayoutFiles()
			if err == nil && len(layoutFiles) > 0 {
				tmpl, err = tmpl.ParseFiles(append(layoutFiles, path)...)
				if err != nil {
					return err
				}
			} else {
				tmpl, err = tmpl.Parse(string(content))
				if err != nil {
					return err
				}
			}
		} else {
			tmpl, err = tmpl.Parse(string(content))
			if err != nil {
				return err
			}
		}
		
		e.templates[name] = tmpl
		
		return nil
	})
}

func (e *DefaultEngine) getLayoutFiles() ([]string, error) {
	return e.fileSystem.Glob(filepath.Join(e.options.LayoutsPath, "*.html"))
}

func (e *DefaultEngine) hasValidExtension(path string) bool {
	ext := filepath.Ext(path)
	for _, validExt := range e.options.Extensions {
		if ext == validExt {
			return true
		}
	}
	return false
}

func (e *DefaultEngine) getTemplateName(path, baseDir string) string {
	relativePath, _ := filepath.Rel(baseDir, path)
	name := strings.TrimSuffix(relativePath, filepath.Ext(relativePath))
	return strings.ReplaceAll(name, string(filepath.Separator), ".")
}

func (e *DefaultEngine) getTemplate(name string) (*template.Template, error) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	
	tmpl, exists := e.templates[name]
	if !exists {
		return nil, fmt.Errorf("template %s not found", name)
	}
	
	return tmpl, nil
}

func (e *DefaultEngine) prepareData(data Data) Data {
	result := make(Data)
	
	// Add layout data
	for k, v := range e.options.LayoutData {
		result[k] = v
	}
	
	// Add provided data
	for k, v := range data {
		result[k] = v
	}
	
	return result
}

func (e *DefaultEngine) getCacheKey(name string, data Data) string {
	// Simple cache key generation
	// In production, you might want a more sophisticated approach
	return fmt.Sprintf("%s:%v", name, data)
}

func (e *DefaultEngine) precompileTemplate(ctx context.Context, name string) error {
	tmpl, err := e.getTemplate(name)
	if err != nil {
		return err
	}
	
	compiled := &CompiledTemplate{
		Name:       name,
		Template:   tmpl,
		CompiledAt: time.Now(),
		Metadata:   make(map[string]interface{}),
	}
	
	e.compiled[name] = compiled
	return nil
}

func (e *DefaultEngine) precompileTemplateUnlocked(ctx context.Context, name string) error {
	tmpl, exists := e.templates[name]
	if !exists {
		return fmt.Errorf("template %s not found", name)
	}
	
	compiled := &CompiledTemplate{
		Name:       name,
		Template:   tmpl,
		CompiledAt: time.Now(),
		Metadata:   make(map[string]interface{}),
	}
	
	e.compiled[name] = compiled
	return nil
}

func (e *DefaultEngine) precompileAll(ctx context.Context) error {
	for name := range e.templates {
		if err := e.precompileTemplateUnlocked(ctx, name); err != nil {
			return err
		}
	}
	return nil
}

func (e *DefaultEngine) extractVariables(tmpl *template.Template) []string {
	// This would require template AST analysis
	// For now, return empty slice
	return []string{}
}

func (e *DefaultEngine) extractFunctions(tmpl *template.Template) []string {
	// This would require template AST analysis
	// For now, return empty slice
	return []string{}
}

func (e *DefaultEngine) registerDefaultFunctions() {
	e.functions["upper"] = strings.ToUpper
	e.functions["lower"] = strings.ToLower
	e.functions["title"] = strings.Title
	e.functions["join"] = func(sep string, items []string) string {
		return strings.Join(items, sep)
	}
	e.functions["add"] = func(a, b int) int {
		return a + b
	}
	e.functions["sub"] = func(a, b int) int {
		return a - b
	}
	e.functions["mul"] = func(a, b int) int {
		return a * b
	}
	e.functions["div"] = func(a, b int) int {
		if b != 0 {
			return a / b
		}
		return 0
	}
	e.functions["mod"] = func(a, b int) int {
		if b != 0 {
			return a % b
		}
		return 0
	}
	e.functions["eq"] = func(a, b interface{}) bool {
		return a == b
	}
	e.functions["ne"] = func(a, b interface{}) bool {
		return a != b
	}
	e.functions["lt"] = func(a, b int) bool {
		return a < b
	}
	e.functions["le"] = func(a, b int) bool {
		return a <= b
	}
	e.functions["gt"] = func(a, b int) bool {
		return a > b
	}
	e.functions["ge"] = func(a, b int) bool {
		return a >= b
	}
	e.functions["now"] = time.Now
	e.functions["formatTime"] = func(t time.Time, layout string) string {
		return t.Format(layout)
	}
	e.functions["contains"] = strings.Contains
	e.functions["hasPrefix"] = strings.HasPrefix
	e.functions["hasSuffix"] = strings.HasSuffix
	e.functions["trim"] = strings.TrimSpace
	e.functions["replace"] = strings.ReplaceAll
}