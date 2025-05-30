package template

import (
	"context"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// OSFileSystem implements FileSystem for the operating system
type OSFileSystem struct{}

// NewOSFileSystem creates a new OS-based file system
func NewOSFileSystem() *OSFileSystem {
	return &OSFileSystem{}
}

// Open opens a file
func (fs *OSFileSystem) Open(name string) (File, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	return &OSFile{f}, nil
}

// ReadFile reads a file completely
func (fs *OSFileSystem) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

// Glob returns files matching a pattern
func (fs *OSFileSystem) Glob(pattern string) ([]string, error) {
	return filepath.Glob(pattern)
}

// ReadDir reads a directory
func (fs *OSFileSystem) ReadDir(name string) ([]DirEntry, error) {
	entries, err := os.ReadDir(name)
	if err != nil {
		return nil, err
	}
	
	result := make([]DirEntry, len(entries))
	for i, entry := range entries {
		result[i] = &OSDirEntry{entry}
	}
	
	return result, nil
}

// Walk walks a directory tree
func (fs *OSFileSystem) Walk(root string, fn WalkFunc) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return fn(path, nil, err)
		}
		if d == nil {
			return fn(path, nil, nil)
		}
		info, infoErr := d.Info()
		if infoErr != nil {
			return fn(path, nil, infoErr)
		}
		return fn(path, &OSFileInfo{info}, err)
	})
}

// Stat returns file information
func (fs *OSFileSystem) Stat(name string) (FileInfo, error) {
	info, err := os.Stat(name)
	if err != nil {
		return nil, err
	}
	return &OSFileInfo{info}, nil
}

// IsDir checks if path is a directory
func (fs *OSFileSystem) IsDir(name string) bool {
	info, err := os.Stat(name)
	return err == nil && info.IsDir()
}

// Exists checks if file exists
func (fs *OSFileSystem) Exists(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}

// Watch watches a path for changes
func (fs *OSFileSystem) Watch(path string, handler WatchHandler) error {
	// This would integrate with a file watcher library like fsnotify
	// For now, return nil
	return nil
}

// Unwatch stops watching a path
func (fs *OSFileSystem) Unwatch(path string) error {
	// This would stop watching with a file watcher library
	// For now, return nil
	return nil
}

// OSFile wraps os.File to implement File interface
type OSFile struct {
	*os.File
}

// Name returns the file name
func (f *OSFile) Name() string {
	return f.File.Name()
}

// Size returns the file size
func (f *OSFile) Size() int64 {
	if info, err := f.File.Stat(); err == nil {
		return info.Size()
	}
	return 0
}

// ModTime returns the modification time
func (f *OSFile) ModTime() time.Time {
	if info, err := f.File.Stat(); err == nil {
		return info.ModTime()
	}
	return time.Time{}
}

// OSFileInfo wraps os.FileInfo to implement FileInfo interface
type OSFileInfo struct {
	os.FileInfo
}

// Mode returns the file mode type
func (fi *OSFileInfo) Mode() FileMode {
	return FileMode(fi.FileInfo.Mode())
}

// Type returns the file mode type
func (fi *OSFileInfo) Type() FileMode {
	return FileMode(fi.FileInfo.Mode())
}

// OSDirEntry wraps os.DirEntry to implement DirEntry interface
type OSDirEntry struct {
	os.DirEntry
}

// Type returns the file mode type
func (de *OSDirEntry) Type() FileMode {
	return FileMode(de.DirEntry.Type())
}

// Info returns file info
func (de *OSDirEntry) Info() (FileInfo, error) {
	info, err := de.DirEntry.Info()
	if err != nil {
		return nil, err
	}
	return &OSFileInfo{info}, nil
}

// LRUCache implements Cache using LRU eviction
type LRUCache struct {
	items    map[string]*cacheItem
	maxSize  int
	ttl      time.Duration
	head     *cacheItem
	tail     *cacheItem
	mutex    sync.RWMutex
	stats    *CacheStats
}

type cacheItem struct {
	key       string
	template  *CachedTemplate
	prev      *cacheItem
	next      *cacheItem
	createdAt time.Time
	accessedAt time.Time
}

// NewLRUCache creates a new LRU cache
func NewLRUCache(maxSize int, ttl time.Duration) *LRUCache {
	cache := &LRUCache{
		items:   make(map[string]*cacheItem),
		maxSize: maxSize,
		ttl:     ttl,
		stats:   &CacheStats{MaxSize: maxSize},
	}
	
	// Initialize sentinel nodes
	cache.head = &cacheItem{}
	cache.tail = &cacheItem{}
	cache.head.next = cache.tail
	cache.tail.prev = cache.head
	
	return cache
}

// Get retrieves a cached template
func (c *LRUCache) Get(key string) (*CachedTemplate, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	item, exists := c.items[key]
	if !exists {
		c.stats.Misses++
		return nil, false
	}
	
	// Check expiration
	if time.Since(item.createdAt) > c.ttl {
		c.removeItem(item)
		c.stats.Misses++
		return nil, false
	}
	
	// Move to front (most recently used)
	c.moveToFront(item)
	item.accessedAt = time.Now()
	item.template.Hits++
	
	c.stats.Hits++
	c.updateHitRatio()
	
	return item.template, true
}

// Set stores a template in cache
func (c *LRUCache) Set(key string, template *CachedTemplate, ttl time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	// Check if item already exists
	if existing, exists := c.items[key]; exists {
		existing.template = template
		existing.createdAt = time.Now()
		existing.accessedAt = time.Now()
		c.moveToFront(existing)
		return
	}
	
	// Create new item
	item := &cacheItem{
		key:        key,
		template:   template,
		createdAt:  time.Now(),
		accessedAt: time.Now(),
	}
	
	// Add to front
	c.addToFront(item)
	c.items[key] = item
	
	// Evict if necessary
	if len(c.items) > c.maxSize {
		oldest := c.tail.prev
		c.removeItem(oldest)
		c.stats.EvictionCount++
	}
	
	c.stats.Size = len(c.items)
}

// Delete removes a template from cache
func (c *LRUCache) Delete(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	if item, exists := c.items[key]; exists {
		c.removeItem(item)
	}
}

// Clear removes all templates from cache
func (c *LRUCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	c.items = make(map[string]*cacheItem)
	c.head.next = c.tail
	c.tail.prev = c.head
	c.stats.Size = 0
}

// Stats returns cache statistics
func (c *LRUCache) Stats() *CacheStats {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	statsCopy := *c.stats
	return &statsCopy
}

// SetMaxSize sets the maximum cache size
func (c *LRUCache) SetMaxSize(size int) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	c.maxSize = size
	c.stats.MaxSize = size
	
	// Evict items if necessary
	for len(c.items) > c.maxSize {
		oldest := c.tail.prev
		c.removeItem(oldest)
		c.stats.EvictionCount++
	}
}

// SetDefaultTTL sets the default TTL
func (c *LRUCache) SetDefaultTTL(ttl time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	c.ttl = ttl
}

// Close closes the cache
func (c *LRUCache) Close() error {
	c.Clear()
	return nil
}

// Private methods for LRUCache

func (c *LRUCache) addToFront(item *cacheItem) {
	item.prev = c.head
	item.next = c.head.next
	c.head.next.prev = item
	c.head.next = item
}

func (c *LRUCache) removeItem(item *cacheItem) {
	item.prev.next = item.next
	item.next.prev = item.prev
	delete(c.items, item.key)
	c.stats.Size = len(c.items)
}

func (c *LRUCache) moveToFront(item *cacheItem) {
	c.removeFromList(item)
	c.addToFront(item)
}

func (c *LRUCache) removeFromList(item *cacheItem) {
	item.prev.next = item.next
	item.next.prev = item.prev
}

func (c *LRUCache) updateHitRatio() {
	total := c.stats.Hits + c.stats.Misses
	if total > 0 {
		c.stats.HitRatio = float64(c.stats.Hits) / float64(total)
	}
}

// DefaultLoader implements Loader interface
type DefaultLoader struct {
	fileSystem FileSystem
	mutex      sync.RWMutex
}

// NewDefaultLoader creates a new default loader
func NewDefaultLoader(fs FileSystem) *DefaultLoader {
	return &DefaultLoader{
		fileSystem: fs,
	}
}

// LoadFromDirectory loads templates from a directory
func (l *DefaultLoader) LoadFromDirectory(ctx context.Context, dir string) (map[string]string, error) {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	
	templates := make(map[string]string)
	
	err := l.fileSystem.Walk(dir, func(path string, info FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if info.IsDir() {
			return nil
		}
		
		// Check file extension
		if !strings.HasSuffix(path, ".html") && !strings.HasSuffix(path, ".tmpl") {
			return nil
		}
		
		content, err := l.fileSystem.ReadFile(path)
		if err != nil {
			return err
		}
		
		// Get relative path as template name
		relativePath, _ := filepath.Rel(dir, path)
		name := strings.TrimSuffix(relativePath, filepath.Ext(relativePath))
		name = strings.ReplaceAll(name, string(filepath.Separator), ".")
		
		templates[name] = string(content)
		
		return nil
	})
	
	return templates, err
}

// LoadFromFiles loads templates from specific files
func (l *DefaultLoader) LoadFromFiles(ctx context.Context, files []string) (map[string]string, error) {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	
	templates := make(map[string]string)
	
	for _, file := range files {
		content, err := l.fileSystem.ReadFile(file)
		if err != nil {
			return nil, err
		}
		
		name := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		templates[name] = string(content)
	}
	
	return templates, nil
}

// LoadFromFS loads templates from file system
func (l *DefaultLoader) LoadFromFS(ctx context.Context, fs FileSystem, root string) (map[string]string, error) {
	oldFS := l.fileSystem
	l.fileSystem = fs
	defer func() { l.fileSystem = oldFS }()
	
	return l.LoadFromDirectory(ctx, root)
}

// LoadFromMap loads templates from a map
func (l *DefaultLoader) LoadFromMap(templates map[string]string) error {
	// This is a no-op for the default loader since templates are already in memory
	return nil
}

// LoadFromConfig loads templates based on configuration
func (l *DefaultLoader) LoadFromConfig(ctx context.Context, config LoaderConfig) (map[string]string, error) {
	templates := make(map[string]string)
	
	for _, source := range config.Sources {
		switch source.Type {
		case "directory":
			dirTemplates, err := l.LoadFromDirectory(ctx, source.Path)
			if err != nil {
				return nil, err
			}
			for name, content := range dirTemplates {
				templates[name] = content
			}
		case "file":
			content, err := l.fileSystem.ReadFile(source.Path)
			if err != nil {
				return nil, err
			}
			name := strings.TrimSuffix(filepath.Base(source.Path), filepath.Ext(source.Path))
			templates[name] = string(content)
		}
	}
	
	return templates, nil
}

// LoadByPattern loads templates matching a pattern
func (l *DefaultLoader) LoadByPattern(ctx context.Context, pattern string) (map[string]string, error) {
	files, err := l.fileSystem.Glob(pattern)
	if err != nil {
		return nil, err
	}
	
	return l.LoadFromFiles(ctx, files)
}

// LoadByExtension loads templates with specific extension
func (l *DefaultLoader) LoadByExtension(ctx context.Context, dir string, ext string) (map[string]string, error) {
	pattern := filepath.Join(dir, "*"+ext)
	return l.LoadByPattern(ctx, pattern)
}

// Watch watches for template changes
func (l *DefaultLoader) Watch(ctx context.Context, handler func(name, content string)) error {
	// This would implement file watching
	// For now, return nil
	return nil
}

// StopWatching stops watching for changes
func (l *DefaultLoader) StopWatching() {
	// This would stop file watching
}

// DefaultValidator implements Validator interface
type DefaultValidator struct{}

// NewDefaultValidator creates a new default validator
func NewDefaultValidator() *DefaultValidator {
	return &DefaultValidator{}
}

// ValidateSyntax validates template syntax
func (v *DefaultValidator) ValidateSyntax(ctx context.Context, source string) error {
	// Basic syntax validation - try to parse template
	_, err := template.New("test").Parse(source)
	return err
}

// ValidateFile validates a template file
func (v *DefaultValidator) ValidateFile(ctx context.Context, path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	
	return v.ValidateSyntax(ctx, string(content))
}

// ValidateBatch validates multiple templates
func (v *DefaultValidator) ValidateBatch(ctx context.Context, templates map[string]string) map[string]error {
	errors := make(map[string]error)
	
	for name, content := range templates {
		if err := v.ValidateSyntax(ctx, content); err != nil {
			errors[name] = err
		}
	}
	
	return errors
}

// ValidateVariables validates template variables
func (v *DefaultValidator) ValidateVariables(ctx context.Context, source string, variables []string) error {
	// This would check if all required variables are present
	return nil
}

// ValidateFunctions validates template functions
func (v *DefaultValidator) ValidateFunctions(ctx context.Context, source string, functions FuncMap) error {
	// This would check if all used functions are available
	return nil
}

// ValidateLayouts validates template layouts
func (v *DefaultValidator) ValidateLayouts(ctx context.Context, template, layout string) error {
	// This would validate template-layout compatibility
	return nil
}

// ValidateSecurity validates template security
func (v *DefaultValidator) ValidateSecurity(ctx context.Context, source string) error {
	// This would check for security issues
	return nil
}

// CheckForVulnerabilities checks for security vulnerabilities
func (v *DefaultValidator) CheckForVulnerabilities(ctx context.Context, source string) []SecurityIssue {
	// This would scan for security issues
	return []SecurityIssue{}
}

// CheckBestPractices checks for best practice violations
func (v *DefaultValidator) CheckBestPractices(ctx context.Context, source string) []BestPracticeIssue {
	// This would check for best practice violations
	return []BestPracticeIssue{}
}

// DefaultCompiler implements Compiler interface
type DefaultCompiler struct{}

// NewDefaultCompiler creates a new default compiler
func NewDefaultCompiler() *DefaultCompiler {
	return &DefaultCompiler{}
}

// Compile compiles a template source
func (c *DefaultCompiler) Compile(ctx context.Context, source string) (*CompiledTemplate, error) {
	tmpl, err := template.New("compiled").Parse(source)
	if err != nil {
		return nil, err
	}
	
	compiled := &CompiledTemplate{
		Source:     source,
		Template:   tmpl,
		CompiledAt: time.Now(),
		Metadata:   make(map[string]interface{}),
	}
	
	return compiled, nil
}

// CompileFile compiles a template file
func (c *DefaultCompiler) CompileFile(ctx context.Context, path string) (*CompiledTemplate, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	
	compiled, err := c.Compile(ctx, string(content))
	if err != nil {
		return nil, err
	}
	
	compiled.Name = filepath.Base(path)
	return compiled, nil
}

// CompileBatch compiles multiple templates
func (c *DefaultCompiler) CompileBatch(ctx context.Context, sources map[string]string) (map[string]*CompiledTemplate, error) {
	compiled := make(map[string]*CompiledTemplate)
	
	for name, source := range sources {
		tmpl, err := c.Compile(ctx, source)
		if err != nil {
			return nil, fmt.Errorf("failed to compile template %s: %w", name, err)
		}
		
		tmpl.Name = name
		compiled[name] = tmpl
	}
	
	return compiled, nil
}

// Analyze analyzes a template
func (c *DefaultCompiler) Analyze(ctx context.Context, source string) (*TemplateAnalysis, error) {
	// This would perform deep template analysis
	return &TemplateAnalysis{
		Variables:    []string{},
		Functions:    []string{},
		Dependencies: []string{},
		Complexity:   0,
		Blocks:       []string{},
		Includes:     []string{},
		Issues:       []AnalysisIssue{},
		Metadata:     make(map[string]interface{}),
	}, nil
}

// ExtractDependencies extracts template dependencies
func (c *DefaultCompiler) ExtractDependencies(ctx context.Context, source string) ([]string, error) {
	// This would extract template dependencies
	return []string{}, nil
}

// Optimize optimizes a compiled template
func (c *DefaultCompiler) Optimize(ctx context.Context, compiled *CompiledTemplate) (*CompiledTemplate, error) {
	// This would perform template optimization
	return compiled, nil
}

// EnableOptimizations enables compiler optimizations
func (c *DefaultCompiler) EnableOptimizations(opts OptimizationOptions) {
	// This would configure optimization options
}

// SetCache sets the compiler cache
func (c *DefaultCompiler) SetCache(cache CompilerCache) {
	// This would set the compiler cache
}

// ClearCache clears the compiler cache
func (c *DefaultCompiler) ClearCache() {
	// This would clear the compiler cache
}

// DefaultErrorHandler implements ErrorHandler interface
type DefaultErrorHandler struct{}

// NewDefaultErrorHandler creates a new default error handler
func NewDefaultErrorHandler() *DefaultErrorHandler {
	return &DefaultErrorHandler{}
}

// HandleError handles general errors
func (h *DefaultErrorHandler) HandleError(ctx context.Context, err error, context ErrorContext) error {
	// Default behavior is to return the error as-is
	return err
}

// HandleRenderError handles render errors
func (h *DefaultErrorHandler) HandleRenderError(ctx context.Context, err error, template string, data Data) error {
	return fmt.Errorf("render error in template %s: %w", template, err)
}

// HandleLoadError handles load errors
func (h *DefaultErrorHandler) HandleLoadError(ctx context.Context, err error, path string) error {
	return fmt.Errorf("load error for %s: %w", path, err)
}

// HotReloader handles hot reloading of templates
type HotReloader struct {
	fileSystem   FileSystem
	watchedPaths map[string]bool
	reloadFunc   func()
	mutex        sync.RWMutex
	stopped      bool
}

// NewHotReloader creates a new hot reloader
func NewHotReloader(fs FileSystem, reloadFunc func()) *HotReloader {
	return &HotReloader{
		fileSystem:   fs,
		watchedPaths: make(map[string]bool),
		reloadFunc:   reloadFunc,
	}
}

// Watch starts watching a path
func (hr *HotReloader) Watch(path string) error {
	hr.mutex.Lock()
	defer hr.mutex.Unlock()
	
	if hr.stopped {
		return fmt.Errorf("hot reloader is stopped")
	}
	
	hr.watchedPaths[path] = true
	
	// This would start watching the path with a file watcher
	return hr.fileSystem.Watch(path, func(event WatchEvent) {
		if !hr.stopped {
			hr.reloadFunc()
		}
	})
}

// Stop stops the hot reloader
func (hr *HotReloader) Stop() {
	hr.mutex.Lock()
	defer hr.mutex.Unlock()
	
	hr.stopped = true
	
	// Stop watching all paths
	for path := range hr.watchedPaths {
		hr.fileSystem.Unwatch(path)
	}
	
	hr.watchedPaths = make(map[string]bool)
}