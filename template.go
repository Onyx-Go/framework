package onyx

import (
	"fmt"
	"html/template"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"
)

type TemplateEngine struct {
	templates   map[string]*template.Template
	viewsPath   string
	layoutsPath string
	mutex       sync.RWMutex
	functions   template.FuncMap
}

type ViewData map[string]interface{}

func NewTemplateEngine(viewsPath, layoutsPath string) *TemplateEngine {
	te := &TemplateEngine{
		templates:   make(map[string]*template.Template),
		viewsPath:   viewsPath,
		layoutsPath: layoutsPath,
		functions:   make(template.FuncMap),
	}
	
	te.registerDefaultFunctions()
	return te
}

func (te *TemplateEngine) registerDefaultFunctions() {
	te.functions["upper"] = strings.ToUpper
	te.functions["lower"] = strings.ToLower
	te.functions["title"] = strings.Title
	te.functions["join"] = func(sep string, items []string) string {
		return strings.Join(items, sep)
	}
	te.functions["add"] = func(a, b int) int {
		return a + b
	}
	te.functions["sub"] = func(a, b int) int {
		return a - b
	}
	te.functions["mul"] = func(a, b int) int {
		return a * b
	}
	te.functions["div"] = func(a, b int) int {
		if b != 0 {
			return a / b
		}
		return 0
	}
	te.functions["mod"] = func(a, b int) int {
		if b != 0 {
			return a % b
		}
		return 0
	}
	te.functions["eq"] = func(a, b interface{}) bool {
		return a == b
	}
	te.functions["ne"] = func(a, b interface{}) bool {
		return a != b
	}
	te.functions["lt"] = func(a, b int) bool {
		return a < b
	}
	te.functions["le"] = func(a, b int) bool {
		return a <= b
	}
	te.functions["gt"] = func(a, b int) bool {
		return a > b
	}
	te.functions["ge"] = func(a, b int) bool {
		return a >= b
	}
}

func (te *TemplateEngine) AddFunction(name string, fn interface{}) {
	te.mutex.Lock()
	defer te.mutex.Unlock()
	te.functions[name] = fn
}

func (te *TemplateEngine) LoadTemplates() error {
	te.mutex.Lock()
	defer te.mutex.Unlock()
	
	return filepath.WalkDir(te.viewsPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		if d.IsDir() || !strings.HasSuffix(path, ".html") {
			return nil
		}
		
		relativePath, err := filepath.Rel(te.viewsPath, path)
		if err != nil {
			return err
		}
		
		templateName := strings.TrimSuffix(relativePath, ".html")
		templateName = strings.ReplaceAll(templateName, string(filepath.Separator), ".")
		
		tmpl := template.New(templateName).Funcs(te.functions)
		
		if te.layoutsPath != "" {
			layoutFiles, err := filepath.Glob(filepath.Join(te.layoutsPath, "*.html"))
			if err == nil && len(layoutFiles) > 0 {
				tmpl, err = tmpl.ParseFiles(append(layoutFiles, path)...)
				if err != nil {
					return err
				}
			} else {
				tmpl, err = tmpl.ParseFiles(path)
				if err != nil {
					return err
				}
			}
		} else {
			tmpl, err = tmpl.ParseFiles(path)
			if err != nil {
				return err
			}
		}
		
		te.templates[templateName] = tmpl
		return nil
	})
}

func (te *TemplateEngine) Render(templateName string, data ViewData) (string, error) {
	te.mutex.RLock()
	tmpl, exists := te.templates[templateName]
	te.mutex.RUnlock()
	
	if !exists {
		return "", fmt.Errorf("template %s not found", templateName)
	}
	
	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	
	return buf.String(), nil
}

func (te *TemplateEngine) RenderWithLayout(templateName, layoutName string, data ViewData) (string, error) {
	te.mutex.RLock()
	tmpl, exists := te.templates[templateName]
	te.mutex.RUnlock()
	
	if !exists {
		return "", fmt.Errorf("template %s not found", templateName)
	}
	
	var buf strings.Builder
	if err := tmpl.ExecuteTemplate(&buf, layoutName, data); err != nil {
		return "", err
	}
	
	return buf.String(), nil
}

func (te *TemplateEngine) HasTemplate(templateName string) bool {
	te.mutex.RLock()
	defer te.mutex.RUnlock()
	_, exists := te.templates[templateName]
	return exists
}

func RenderView(c Context, templateName string, data ViewData) error {
	// Use global template engine since Application interface doesn't expose TemplateEngine
	// TODO: Extend Application interface to provide access to TemplateEngine
	engine := GetGlobalTemplateEngine()
	if engine == nil {
		return fmt.Errorf("template engine not configured")
	}
	
	html, err := engine.Render(templateName, data)
	if err != nil {
		return err
	}
	
	return c.HTML(200, html)
}

func RenderViewWithLayout(c Context, templateName, layoutName string, data ViewData) error {
	// Use global template engine since Application interface doesn't expose TemplateEngine
	// TODO: Extend Application interface to provide access to TemplateEngine
	engine := GetGlobalTemplateEngine()
	if engine == nil {
		return fmt.Errorf("template engine not configured")
	}
	
	html, err := engine.RenderWithLayout(templateName, layoutName, data)
	if err != nil {
		return err
	}
	
	return c.HTML(200, html)
}

// Global template engine instance
var globalTemplateEngine *TemplateEngine

// GetGlobalTemplateEngine returns the global template engine
func GetGlobalTemplateEngine() *TemplateEngine {
	return globalTemplateEngine
}

// SetGlobalTemplateEngine sets the global template engine
func SetGlobalTemplateEngine(engine *TemplateEngine) {
	globalTemplateEngine = engine
}