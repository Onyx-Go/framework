package template

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDefaultEngine_NewEngine(t *testing.T) {
	opts := DefaultOptions()
	engine := NewEngine(opts)

	if engine == nil {
		t.Fatal("NewEngine returned nil")
	}

	if engine.options != opts {
		t.Error("Engine options not set correctly")
	}

	if engine.templates == nil {
		t.Error("Templates map not initialized")
	}

	if engine.compiled == nil {
		t.Error("Compiled map not initialized")
	}

	if engine.functions == nil {
		t.Error("Functions map not initialized")
	}

	if engine.fileSystem == nil {
		t.Error("FileSystem not initialized")
	}

	if engine.cache == nil && opts.EnableCache {
		t.Error("Cache not initialized when enabled")
	}

	if engine.loader == nil {
		t.Error("Loader not initialized")
	}

	if engine.validator == nil {
		t.Error("Validator not initialized")
	}

	if engine.compiler == nil {
		t.Error("Compiler not initialized")
	}

	if engine.errorHandler == nil {
		t.Error("ErrorHandler not initialized")
	}

	if engine.stats == nil {
		t.Error("Stats not initialized")
	}
}

func TestDefaultEngine_AddFunctions(t *testing.T) {
	engine := NewEngine(DefaultOptions())

	// Test AddFunction
	testFunc := func(s string) string { return strings.ToUpper(s) }
	result := engine.AddFunction("test", testFunc)

	if result != engine {
		t.Error("AddFunction should return engine for chaining")
	}

	functions := engine.GetFunctions()
	if _, exists := functions["test"]; !exists {
		t.Error("Function not added")
	}

	// Test AddFunctions
	newFuncs := FuncMap{
		"test2": func(s string) string { return strings.ToLower(s) },
		"test3": func(a, b int) int { return a + b },
	}

	result = engine.AddFunctions(newFuncs)
	if result != engine {
		t.Error("AddFunctions should return engine for chaining")
	}

	functions = engine.GetFunctions()
	if _, exists := functions["test2"]; !exists {
		t.Error("Function test2 not added")
	}
	if _, exists := functions["test3"]; !exists {
		t.Error("Function test3 not added")
	}

	// Test RemoveFunction
	result = engine.RemoveFunction("test2")
	if result != engine {
		t.Error("RemoveFunction should return engine for chaining")
	}

	functions = engine.GetFunctions()
	if _, exists := functions["test2"]; exists {
		t.Error("Function test2 not removed")
	}
}

func TestDefaultEngine_SetOptions(t *testing.T) {
	engine := NewEngine(DefaultOptions())

	newOpts := &Options{
		ViewsPath:   "custom/views",
		LayoutsPath: "custom/layouts",
		LeftDelim:   "[[",
		RightDelim:  "]]",
	}

	result := engine.SetOptions(newOpts)
	if result != engine {
		t.Error("SetOptions should return engine for chaining")
	}

	if engine.GetOptions() != newOpts {
		t.Error("Options not set correctly")
	}

	// Test SetDelimiters
	result = engine.SetDelimiters("<%", "%>")
	if result != engine {
		t.Error("SetDelimiters should return engine for chaining")
	}

	opts := engine.GetOptions()
	if opts.LeftDelim != "<%" || opts.RightDelim != "%>" {
		t.Error("Delimiters not set correctly")
	}
}

func TestDefaultEngine_TemplateManagement(t *testing.T) {
	// Create temporary directory structure
	tempDir := createTempTemplateDir(t)
	defer os.RemoveAll(tempDir)

	opts := DefaultOptions()
	opts.ViewsPath = filepath.Join(tempDir, "views")
	opts.LayoutsPath = filepath.Join(tempDir, "layouts")
	opts.PartialsPath = filepath.Join(tempDir, "partials")

	engine := NewEngine(opts)
	ctx := context.Background()

	// Test LoadTemplates
	err := engine.LoadTemplates(ctx)
	if err != nil {
		t.Fatalf("LoadTemplates failed: %v", err)
	}

	// Test HasTemplate
	if !engine.HasTemplate("home") {
		t.Error("Template 'home' should exist")
	}

	if engine.HasTemplate("nonexistent") {
		t.Error("Template 'nonexistent' should not exist")
	}

	// Test GetTemplateNames
	names := engine.GetTemplateNames()
	if len(names) == 0 {
		t.Error("No templates loaded")
	}

	foundHome := false
	for _, name := range names {
		if name == "home" {
			foundHome = true
			break
		}
	}
	if !foundHome {
		t.Error("Template 'home' not found in names")
	}

	// Test GetTemplateInfo
	info, err := engine.GetTemplateInfo("home")
	if err != nil {
		t.Errorf("GetTemplateInfo failed: %v", err)
	}
	if info == nil {
		t.Error("Template info is nil")
	}
	if info.Name != "home" {
		t.Errorf("Expected template name 'home', got '%s'", info.Name)
	}

	// Test non-existent template
	_, err = engine.GetTemplateInfo("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent template")
	}
}

func TestDefaultEngine_Rendering(t *testing.T) {
	// Create temporary directory structure
	tempDir := createTempTemplateDir(t)
	defer os.RemoveAll(tempDir)

	opts := DefaultOptions()
	opts.ViewsPath = filepath.Join(tempDir, "views")
	opts.LayoutsPath = "" // Disable layouts for this test
	opts.PartialsPath = filepath.Join(tempDir, "partials")

	engine := NewEngine(opts)
	ctx := context.Background()

	err := engine.LoadTemplates(ctx)
	if err != nil {
		t.Fatalf("LoadTemplates failed: %v", err)
	}

	// Test Render
	data := Data{
		"title": "Test Title",
		"name":  "John Doe",
	}

	result, err := engine.Render(ctx, "home", data)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(result, "Test Title") {
		t.Error("Rendered content should contain title")
	}

	if !strings.Contains(result, "John Doe") {
		t.Error("Rendered content should contain name")
	}

	// Test RenderTo
	var buf strings.Builder
	err = engine.RenderTo(ctx, &buf, "home", data)
	if err != nil {
		t.Fatalf("RenderTo failed: %v", err)
	}

	if buf.String() != result {
		t.Error("RenderTo result should match Render result")
	}

	// Skip layout tests since layouts are disabled for this test
	// Test RenderWithLayout would require layout templates to be loaded

	// Test error case - non-existent template
	_, err = engine.Render(ctx, "nonexistent", data)
	if err == nil {
		t.Error("Expected error for non-existent template")
	}
}

func TestDefaultEngine_Cache(t *testing.T) {
	opts := DefaultOptions()
	opts.EnableCache = true
	opts.CacheSize = 10
	opts.CacheTTL = 1 * time.Hour

	engine := NewEngine(opts)

	// Test cache operations
	err := engine.ClearCache()
	if err != nil {
		t.Errorf("ClearCache failed: %v", err)
	}

	stats := engine.GetCacheStats()
	if stats == nil {
		t.Error("Cache stats should not be nil")
	}

	if stats.Size != 0 {
		t.Error("Cache size should be 0 after clear")
	}
}

func TestDefaultEngine_HotReload(t *testing.T) {
	// Create temporary directory structure
	tempDir := createTempTemplateDir(t)
	defer os.RemoveAll(tempDir)

	opts := DefaultOptions()
	opts.ViewsPath = filepath.Join(tempDir, "views")
	opts.PartialsPath = filepath.Join(tempDir, "partials")
	opts.EnableHotReload = false

	engine := NewEngine(opts)
	ctx := context.Background()

	// Test that hot reload is initially disabled
	if engine.IsHotReloadEnabled() {
		t.Error("Hot reload should be disabled initially")
	}

	// Test EnableHotReload
	err := engine.EnableHotReload(ctx)
	if err != nil {
		t.Errorf("EnableHotReload failed: %v", err)
	}

	if !engine.IsHotReloadEnabled() {
		t.Error("Hot reload should be enabled")
	}

	// Test DisableHotReload
	err = engine.DisableHotReload()
	if err != nil {
		t.Errorf("DisableHotReload failed: %v", err)
	}

	if engine.IsHotReloadEnabled() {
		t.Error("Hot reload should be disabled")
	}

	// Test enabling twice (should not error)
	err = engine.EnableHotReload(ctx)
	if err != nil {
		t.Errorf("First EnableHotReload failed: %v", err)
	}

	err = engine.EnableHotReload(ctx)
	if err != nil {
		t.Errorf("Second EnableHotReload failed: %v", err)
	}
}

func TestDefaultEngine_Precompile(t *testing.T) {
	// Create temporary directory structure
	tempDir := createTempTemplateDir(t)
	defer os.RemoveAll(tempDir)

	opts := DefaultOptions()
	opts.ViewsPath = filepath.Join(tempDir, "views")
	opts.LayoutsPath = filepath.Join(tempDir, "layouts")
	opts.PartialsPath = filepath.Join(tempDir, "partials")

	engine := NewEngine(opts)
	ctx := context.Background()

	err := engine.LoadTemplates(ctx)
	if err != nil {
		t.Fatalf("LoadTemplates failed: %v", err)
	}

	// Test Precompile with specific templates
	err = engine.Precompile(ctx, "home")
	if err != nil {
		t.Errorf("Precompile failed: %v", err)
	}

	// Test Precompile all templates
	err = engine.Precompile(ctx)
	if err != nil {
		t.Errorf("Precompile all failed: %v", err)
	}

	// Test precompile non-existent template
	err = engine.Precompile(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent template")
	}
}

func TestDefaultEngine_ErrorHandling(t *testing.T) {
	engine := NewEngine(DefaultOptions())

	// Test SetErrorHandler
	customHandler := NewDefaultErrorHandler()
	result := engine.SetErrorHandler(customHandler)

	if result != engine {
		t.Error("SetErrorHandler should return engine for chaining")
	}

	if engine.GetErrorHandler() != customHandler {
		t.Error("Error handler not set correctly")
	}
}

func TestDefaultEngine_Close(t *testing.T) {
	opts := DefaultOptions()
	opts.EnableCache = true

	engine := NewEngine(opts)
	ctx := context.Background()

	// Enable hot reload to test cleanup
	err := engine.EnableHotReload(ctx)
	if err != nil {
		t.Errorf("EnableHotReload failed: %v", err)
	}

	// Test Close
	err = engine.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Verify hot reload is disabled
	if engine.IsHotReloadEnabled() {
		t.Error("Hot reload should be disabled after close")
	}
}

func TestLRUCache(t *testing.T) {
	cache := NewLRUCache(3, 1*time.Hour)

	// Test basic operations
	template1 := &CachedTemplate{
		Content:   "content1",
		ExpiresAt: time.Now().Add(1 * time.Hour),
		CreatedAt: time.Now(),
	}

	cache.Set("key1", template1, 1*time.Hour)

	// Test Get
	retrieved, exists := cache.Get("key1")
	if !exists {
		t.Error("Template should exist in cache")
	}
	if retrieved.Content != "content1" {
		t.Error("Retrieved template content doesn't match")
	}

	// Test cache miss
	_, exists = cache.Get("nonexistent")
	if exists {
		t.Error("Non-existent key should not exist")
	}

	// Test LRU eviction
	template2 := &CachedTemplate{Content: "content2", CreatedAt: time.Now()}
	template3 := &CachedTemplate{Content: "content3", CreatedAt: time.Now()}
	template4 := &CachedTemplate{Content: "content4", CreatedAt: time.Now()}

	cache.Set("key2", template2, 1*time.Hour)
	cache.Set("key3", template3, 1*time.Hour)
	cache.Set("key4", template4, 1*time.Hour) // Should evict key1

	_, exists = cache.Get("key1")
	if exists {
		t.Error("key1 should have been evicted")
	}

	// Test Delete
	cache.Delete("key2")
	_, exists = cache.Get("key2")
	if exists {
		t.Error("key2 should have been deleted")
	}

	// Test Clear
	cache.Clear()
	stats := cache.Stats()
	if stats.Size != 0 {
		t.Error("Cache should be empty after clear")
	}

	// Test SetMaxSize
	cache.SetMaxSize(1)
	if cache.Stats().MaxSize != 1 {
		t.Error("Max size not updated")
	}

	// Test SetDefaultTTL
	cache.SetDefaultTTL(30 * time.Minute)

	// Test Close
	err := cache.Close()
	if err != nil {
		t.Errorf("Cache close failed: %v", err)
	}
}

func TestOSFileSystem(t *testing.T) {
	fs := NewOSFileSystem()

	// Create temporary file for testing
	tempFile := filepath.Join(os.TempDir(), "test_template.html")
	content := "Test template content"
	err := os.WriteFile(tempFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile)

	// Test ReadFile
	data, err := fs.ReadFile(tempFile)
	if err != nil {
		t.Errorf("ReadFile failed: %v", err)
	}
	if string(data) != content {
		t.Error("ReadFile content doesn't match")
	}

	// Test Exists
	if !fs.Exists(tempFile) {
		t.Error("File should exist")
	}

	if fs.Exists("/nonexistent/file.html") {
		t.Error("Non-existent file should not exist")
	}

	// Test IsDir
	if fs.IsDir(tempFile) {
		t.Error("File should not be a directory")
	}

	if !fs.IsDir(os.TempDir()) {
		t.Error("Temp dir should be a directory")
	}

	// Test Stat
	info, err := fs.Stat(tempFile)
	if err != nil {
		t.Errorf("Stat failed: %v", err)
	}
	if info.Name() != filepath.Base(tempFile) {
		t.Error("File name doesn't match")
	}

	// Test Open
	file, err := fs.Open(tempFile)
	if err != nil {
		t.Errorf("Open failed: %v", err)
	}
	defer file.Close()

	// Read from opened file
	fileData, err := io.ReadAll(file)
	if err != nil {
		t.Errorf("ReadAll failed: %v", err)
	}
	if string(fileData) != content {
		t.Error("File content doesn't match")
	}

	// Test Glob
	pattern := filepath.Join(os.TempDir(), "test_*.html")
	matches, err := fs.Glob(pattern)
	if err != nil {
		t.Errorf("Glob failed: %v", err)
	}
	if len(matches) != 1 || matches[0] != tempFile {
		t.Error("Glob didn't return expected match")
	}
}

func TestDefaultLoader(t *testing.T) {
	// Create temporary directory structure
	tempDir := createTempTemplateDir(t)
	defer os.RemoveAll(tempDir)

	fs := NewOSFileSystem()
	loader := NewDefaultLoader(fs)
	ctx := context.Background()

	viewsDir := filepath.Join(tempDir, "views")

	// Test LoadFromDirectory
	templates, err := loader.LoadFromDirectory(ctx, viewsDir)
	if err != nil {
		t.Fatalf("LoadFromDirectory failed: %v", err)
	}

	if len(templates) == 0 {
		t.Error("No templates loaded")
	}

	if _, exists := templates["home"]; !exists {
		t.Error("Template 'home' not found")
	}

	// Test LoadFromFiles
	homeFile := filepath.Join(viewsDir, "home.html")
	files := []string{homeFile}
	fileTemplates, err := loader.LoadFromFiles(ctx, files)
	if err != nil {
		t.Fatalf("LoadFromFiles failed: %v", err)
	}

	if _, exists := fileTemplates["home"]; !exists {
		t.Error("Template 'home' not found in file templates")
	}

	// Test LoadByPattern
	pattern := filepath.Join(viewsDir, "*.html")
	patternTemplates, err := loader.LoadByPattern(ctx, pattern)
	if err != nil {
		t.Fatalf("LoadByPattern failed: %v", err)
	}

	if len(patternTemplates) == 0 {
		t.Error("No templates loaded by pattern")
	}

	// Test LoadByExtension
	extTemplates, err := loader.LoadByExtension(ctx, viewsDir, ".html")
	if err != nil {
		t.Fatalf("LoadByExtension failed: %v", err)
	}

	if len(extTemplates) == 0 {
		t.Error("No templates loaded by extension")
	}

	// Test LoadFromConfig
	config := LoaderConfig{
		Sources: []SourceConfig{
			{Type: "directory", Path: viewsDir},
		},
	}
	configTemplates, err := loader.LoadFromConfig(ctx, config)
	if err != nil {
		t.Fatalf("LoadFromConfig failed: %v", err)
	}

	if len(configTemplates) == 0 {
		t.Error("No templates loaded from config")
	}
}

func TestDefaultValidator(t *testing.T) {
	validator := NewDefaultValidator()
	ctx := context.Background()

	// Test ValidateSyntax with valid template
	validTemplate := `<h1>{{.Title}}</h1><p>{{.Content}}</p>`
	err := validator.ValidateSyntax(ctx, validTemplate)
	if err != nil {
		t.Errorf("ValidateSyntax failed for valid template: %v", err)
	}

	// Test ValidateSyntax with invalid template
	invalidTemplate := `<h1>{{.Title}</h1><p>{{.Content}}</p>` // Missing closing }}
	err = validator.ValidateSyntax(ctx, invalidTemplate)
	if err == nil {
		t.Error("ValidateSyntax should fail for invalid template")
	}

	// Test ValidateBatch
	templates := map[string]string{
		"valid":   validTemplate,
		"invalid": invalidTemplate,
	}
	errors := validator.ValidateBatch(ctx, templates)
	if len(errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(errors))
	}
	if _, exists := errors["invalid"]; !exists {
		t.Error("Expected error for invalid template")
	}
}

func TestDefaultCompiler(t *testing.T) {
	compiler := NewDefaultCompiler()
	ctx := context.Background()

	// Test Compile
	source := `<h1>{{.Title}}</h1><p>{{.Content}}</p>`
	compiled, err := compiler.Compile(ctx, source)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	if compiled.Source != source {
		t.Error("Compiled source doesn't match original")
	}

	if compiled.Template == nil {
		t.Error("Compiled template is nil")
	}

	if compiled.CompiledAt.IsZero() {
		t.Error("Compiled timestamp not set")
	}

	// Test CompileBatch
	sources := map[string]string{
		"template1": `<h1>{{.Title}}</h1>`,
		"template2": `<p>{{.Content}}</p>`,
	}
	compiledTemplates, err := compiler.CompileBatch(ctx, sources)
	if err != nil {
		t.Fatalf("CompileBatch failed: %v", err)
	}

	if len(compiledTemplates) != 2 {
		t.Errorf("Expected 2 compiled templates, got %d", len(compiledTemplates))
	}

	// Test Analyze
	analysis, err := compiler.Analyze(ctx, source)
	if err != nil {
		t.Errorf("Analyze failed: %v", err)
	}

	if analysis == nil {
		t.Error("Analysis is nil")
	}

	// Test Optimize
	optimized, err := compiler.Optimize(ctx, compiled)
	if err != nil {
		t.Errorf("Optimize failed: %v", err)
	}

	if optimized == nil {
		t.Error("Optimized template is nil")
	}
}

func TestDefaultErrorHandler(t *testing.T) {
	handler := NewDefaultErrorHandler()
	ctx := context.Background()

	// Test HandleError
	testErr := fmt.Errorf("test error")
	errorContext := ErrorContext{
		Template: "test",
		Line:     1,
		Column:   1,
	}

	handledErr := handler.HandleError(ctx, testErr, errorContext)
	if handledErr != testErr {
		t.Error("HandleError should return the original error")
	}

	// Test HandleRenderError
	data := Data{"key": "value"}
	renderErr := handler.HandleRenderError(ctx, testErr, "test-template", data)
	if renderErr == nil {
		t.Error("HandleRenderError should return an error")
	}

	expectedMsg := "render error in template test-template: test error"
	if renderErr.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, renderErr.Error())
	}

	// Test HandleLoadError
	loadErr := handler.HandleLoadError(ctx, testErr, "/path/to/template")
	if loadErr == nil {
		t.Error("HandleLoadError should return an error")
	}

	expectedLoadMsg := "load error for /path/to/template: test error"
	if loadErr.Error() != expectedLoadMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedLoadMsg, loadErr.Error())
	}
}

func TestHotReloader(t *testing.T) {
	fs := NewOSFileSystem()
	reloadFunc := func() {
		// Reload function for testing
	}

	reloader := NewHotReloader(fs, reloadFunc)

	// Test Watch
	tempDir := os.TempDir()
	err := reloader.Watch(tempDir)
	if err != nil {
		t.Errorf("Watch failed: %v", err)
	}

	// Test Stop
	reloader.Stop()

	// Test watching after stop should fail
	err = reloader.Watch(tempDir)
	if err == nil {
		t.Error("Watch should fail after stop")
	}
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	if opts == nil {
		t.Fatal("DefaultOptions returned nil")
	}

	if opts.ViewsPath == "" {
		t.Error("ViewsPath should not be empty")
	}

	if opts.LayoutsPath == "" {
		t.Error("LayoutsPath should not be empty")
	}

	if opts.PartialsPath == "" {
		t.Error("PartialsPath should not be empty")
	}

	if len(opts.Extensions) == 0 {
		t.Error("Extensions should not be empty")
	}

	if opts.LeftDelim == "" || opts.RightDelim == "" {
		t.Error("Delimiters should not be empty")
	}

	if opts.CacheSize <= 0 {
		t.Error("CacheSize should be positive")
	}

	if opts.CacheTTL <= 0 {
		t.Error("CacheTTL should be positive")
	}
}

func TestDataHelpers(t *testing.T) {
	data := Data{
		"string": "test",
		"int":    42,
		"bool":   true,
		"float":  3.14,
	}

	// Test Get
	if data.Get("string") != "test" {
		t.Error("Get string failed")
	}

	// Test GetString
	if data.GetString("string") != "test" {
		t.Error("GetString failed")
	}

	if data.GetString("int") != "" {
		t.Error("GetString should return empty for non-string")
	}

	// Test GetInt
	if data.GetInt("int") != 42 {
		t.Error("GetInt failed")
	}

	if data.GetInt("string") != 0 {
		t.Error("GetInt should return 0 for non-int")
	}

	// Test GetBool
	if !data.GetBool("bool") {
		t.Error("GetBool failed")
	}

	if data.GetBool("string") {
		t.Error("GetBool should return false for non-bool")
	}

	// Test Set
	data.Set("new", "value")
	if data.Get("new") != "value" {
		t.Error("Set failed")
	}

	// Test Has
	if !data.Has("string") {
		t.Error("Has should return true for existing key")
	}

	if data.Has("nonexistent") {
		t.Error("Has should return false for non-existing key")
	}

	// Test Merge
	other := Data{
		"other": "value",
		"int":   100, // Should override
	}

	merged := data.Merge(other)
	if merged.GetString("other") != "value" {
		t.Error("Merge should include new key")
	}

	if merged.GetInt("int") != 100 {
		t.Error("Merge should override existing key")
	}

	// Original data should not be modified
	if data.GetInt("int") != 42 {
		t.Error("Original data should not be modified")
	}
}

// Helper function to create temporary template directory structure
func createTempTemplateDir(t *testing.T) string {
	tempDir, err := os.MkdirTemp("", "onyx_template_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create views directory
	viewsDir := filepath.Join(tempDir, "views")
	err = os.MkdirAll(viewsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create views dir: %v", err)
	}

	// Create layouts directory
	layoutsDir := filepath.Join(tempDir, "layouts")
	err = os.MkdirAll(layoutsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create layouts dir: %v", err)
	}

	// Create partials directory
	partialsDir := filepath.Join(tempDir, "partials")
	err = os.MkdirAll(partialsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create partials dir: %v", err)
	}

	// Create sample templates
	homeTemplate := `<h1>{{.title}}</h1>
<p>Welcome, {{.name}}!</p>`

	err = os.WriteFile(filepath.Join(viewsDir, "home.html"), []byte(homeTemplate), 0644)
	if err != nil {
		t.Fatalf("Failed to create home template: %v", err)
	}

	// Create sample layout
	layoutTemplate := `<!DOCTYPE html>
<html>
<head>
    <title>{{.title}}</title>
</head>
<body>
    {{template "content" .}}
</body>
</html>
{{define "content"}}{{end}}`

	err = os.WriteFile(filepath.Join(layoutsDir, "app.html"), []byte(layoutTemplate), 0644)
	if err != nil {
		t.Fatalf("Failed to create layout template: %v", err)
	}

	return tempDir
}