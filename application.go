package onyx

import (
	"fmt"
	"net/http"
	"time"

	httpInternal "github.com/onyx-go/framework/internal/http"
	contextImpl "github.com/onyx-go/framework/internal/http/context"
	routerImpl "github.com/onyx-go/framework/internal/http/router"
)

// Wrapper types for backward compatibility
type Context = *contextImpl.Context
type HandlerFunc func(Context) error  
type MiddlewareFunc func(Context) error

// Router wrapper
type Router struct {
	*routerImpl.Router
}

// Router wrapper methods
func (r *Router) Group(prefix string, middleware ...MiddlewareFunc) *RouteGroup {
	// Convert middleware to internal types
	var internalMiddleware []httpInternal.MiddlewareFunc
	for _, mw := range middleware {
		internalMw := func(c httpInternal.Context) error {
			return mw(c.(*contextImpl.Context))
		}
		internalMiddleware = append(internalMiddleware, internalMw)
	}
	
	internalGroup := r.Router.Group(prefix, internalMiddleware...).(*routerImpl.RouteGroup)
	return wrapRouteGroup(internalGroup)
}

// RouteGroup wrapper  
type RouteGroup struct {
	*routerImpl.RouteGroup
	prefix string
}

type Route = httpInternal.Route

// RouteGroup compatibility methods
func (rg *RouteGroup) Get(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	// Convert handler and middleware to internal types
	internalHandler := func(c httpInternal.Context) error {
		return handler(c.(*contextImpl.Context))
	}
	
	var internalMiddleware []httpInternal.MiddlewareFunc
	for _, mw := range middleware {
		internalMw := func(c httpInternal.Context) error {
			return mw(c.(*contextImpl.Context))
		}
		internalMiddleware = append(internalMiddleware, internalMw)
	}
	
	rg.RouteGroup.Get(pattern, internalHandler, internalMiddleware...)
}

func (rg *RouteGroup) Post(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	// Convert handler and middleware to internal types
	internalHandler := func(c httpInternal.Context) error {
		return handler(c.(*contextImpl.Context))
	}
	
	var internalMiddleware []httpInternal.MiddlewareFunc
	for _, mw := range middleware {
		internalMw := func(c httpInternal.Context) error {
			return mw(c.(*contextImpl.Context))
		}
		internalMiddleware = append(internalMiddleware, internalMw)
	}
	
	rg.RouteGroup.Post(pattern, internalHandler, internalMiddleware...)
}

func (rg *RouteGroup) Put(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	// Convert handler and middleware to internal types
	internalHandler := func(c httpInternal.Context) error {
		return handler(c.(*contextImpl.Context))
	}
	
	var internalMiddleware []httpInternal.MiddlewareFunc
	for _, mw := range middleware {
		internalMw := func(c httpInternal.Context) error {
			return mw(c.(*contextImpl.Context))
		}
		internalMiddleware = append(internalMiddleware, internalMw)
	}
	
	rg.RouteGroup.Put(pattern, internalHandler, internalMiddleware...)
}

func (rg *RouteGroup) Delete(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	// Convert handler and middleware to internal types
	internalHandler := func(c httpInternal.Context) error {
		return handler(c.(*contextImpl.Context))
	}
	
	var internalMiddleware []httpInternal.MiddlewareFunc
	for _, mw := range middleware {
		internalMw := func(c httpInternal.Context) error {
			return mw(c.(*contextImpl.Context))
		}
		internalMiddleware = append(internalMiddleware, internalMw)
	}
	
	rg.RouteGroup.Delete(pattern, internalHandler, internalMiddleware...)
}

// Initialize the prefix field when creating RouteGroup wrappers
func wrapRouteGroup(internal *routerImpl.RouteGroup) *RouteGroup {
	return &RouteGroup{
		RouteGroup: internal,
		prefix:     internal.Prefix_,
	}
}

type Application struct {
	router         httpInternal.Router
	server         *http.Server
	config         *Config
	container      *Container
	templateEngine *TemplateEngine
}

func New() *Application {
	r := router.NewRouter()
	app := &Application{
		router:    r,
		config:    NewConfig(),
		container: NewContainer(),
	}
	
	r.SetApplication(app)
	
	// Setup default logging configuration
	config := LoggingConfig{
		DefaultChannel: "console",
		Console: struct {
			Level    LogLevel `json:"level"`
			Colorize bool     `json:"colorize"`
		}{
			Level:    InfoLevel,
			Colorize: true,
		},
		File: struct {
			Enabled  bool     `json:"enabled"`
			Path     string   `json:"path"`
			Level    LogLevel `json:"level"`
			MaxSize  int64    `json:"max_size"`
			MaxFiles int      `json:"max_files"`
		}{
			Enabled:  false,
			Path:     "storage/logs/onyx.log",
			Level:    InfoLevel,
			MaxSize:  10 * 1024 * 1024, // 10MB
			MaxFiles: 5,
		},
		JSON: struct {
			Enabled bool     `json:"enabled"`
			Path    string   `json:"path"`
			Level   LogLevel `json:"level"`
		}{
			Enabled: false,
			Path:    "",
			Level:   InfoLevel,
		},
	}
	
	if err := SetupLogging(config); err != nil {
		fmt.Printf("Warning: Failed to setup logging: %v\n", err)
	}
	
	// Setup error handling
	SetupErrorHandling(false) // Set to true for debug mode
	
	app.router.Use(LoggerMiddleware())
	app.router.Use(RecoveryMiddleware())
	app.router.Use(ErrorHandlerMiddleware(GetErrorHandler()))
	
	return app
}

func (app *Application) Start(address string) error {
	app.server = &http.Server{
		Addr:         address,
		Handler:      app.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	
	fmt.Printf("🚀 Onyx server starting on %s\n", address)
	return app.server.ListenAndServe()
}

func (app *Application) Config() *Config {
	return app.config
}

func (app *Application) Container() *Container {
	return app.container
}

func (app *Application) SetTemplateEngine(viewsPath, layoutsPath string) error {
	app.templateEngine = NewTemplateEngine(viewsPath, layoutsPath)
	return app.templateEngine.LoadTemplates()
}

func (app *Application) GetTemplateEngine() *TemplateEngine {
	return app.templateEngine
}

// Router delegation methods
func (app *Application) GET(pattern string, handler httpInternal.HandlerFunc, middleware ...httpInternal.MiddlewareFunc) {
	app.router.GET(pattern, handler, middleware...)
}

func (app *Application) POST(pattern string, handler httpInternal.HandlerFunc, middleware ...httpInternal.MiddlewareFunc) {
	app.router.POST(pattern, handler, middleware...)
}

func (app *Application) PUT(pattern string, handler httpInternal.HandlerFunc, middleware ...httpInternal.MiddlewareFunc) {
	app.router.PUT(pattern, handler, middleware...)
}

func (app *Application) DELETE(pattern string, handler httpInternal.HandlerFunc, middleware ...httpInternal.MiddlewareFunc) {
	app.router.DELETE(pattern, handler, middleware...)
}

func (app *Application) PATCH(pattern string, handler httpInternal.HandlerFunc, middleware ...httpInternal.MiddlewareFunc) {
	app.router.PATCH(pattern, handler, middleware...)
}

func (app *Application) OPTIONS(pattern string, handler httpInternal.HandlerFunc, middleware ...httpInternal.MiddlewareFunc) {
	app.router.OPTIONS(pattern, handler, middleware...)
}

func (app *Application) HEAD(pattern string, handler httpInternal.HandlerFunc, middleware ...httpInternal.MiddlewareFunc) {
	app.router.HEAD(pattern, handler, middleware...)
}

func (app *Application) ANY(pattern string, handler httpInternal.HandlerFunc, middleware ...httpInternal.MiddlewareFunc) {
	app.router.ANY(pattern, handler, middleware...)
}

func (app *Application) Use(middleware ...httpInternal.MiddlewareFunc) {
	app.router.Use(middleware...)
}

// UseMiddleware accepts old-style middleware for backward compatibility
func (app *Application) UseMiddleware(middleware MiddlewareFunc) {
	// Convert old middleware to new style
	converted := func(c httpInternal.Context) error {
		return middleware(c.(*contextImpl.Context))
	}
	app.router.Use(converted)
}


func (app *Application) Group(prefix string, middleware ...httpInternal.MiddlewareFunc) httpInternal.RouteGroup {
	return app.router.Group(prefix, middleware...)
}

func (app *Application) SetNotFound(handler httpInternal.HandlerFunc) {
	app.router.SetNotFound(handler)
}

// Backward compatibility methods with lowercase names
func (app *Application) Get(pattern string, handler httpInternal.HandlerFunc, middleware ...httpInternal.MiddlewareFunc) {
	app.router.GET(pattern, handler, middleware...)
}

func (app *Application) Post(pattern string, handler httpInternal.HandlerFunc, middleware ...httpInternal.MiddlewareFunc) {
	app.router.POST(pattern, handler, middleware...)
}

func (app *Application) Put(pattern string, handler httpInternal.HandlerFunc, middleware ...httpInternal.MiddlewareFunc) {
	app.router.PUT(pattern, handler, middleware...)
}

func (app *Application) Delete(pattern string, handler httpInternal.HandlerFunc, middleware ...httpInternal.MiddlewareFunc) {
	app.router.DELETE(pattern, handler, middleware...)
}

func (app *Application) Patch(pattern string, handler httpInternal.HandlerFunc, middleware ...httpInternal.MiddlewareFunc) {
	app.router.PATCH(pattern, handler, middleware...)
}

func (app *Application) Options(pattern string, handler httpInternal.HandlerFunc, middleware ...httpInternal.MiddlewareFunc) {
	app.router.OPTIONS(pattern, handler, middleware...)
}

// Handlers with old-style function signatures for backward compatibility
func (app *Application) GetHandler(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	app.Get(pattern, app.convertHandler(handler), app.convertMiddleware(middleware...)...)
}

func (app *Application) PostHandler(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	app.Post(pattern, app.convertHandler(handler), app.convertMiddleware(middleware...)...)
}

func (app *Application) PutHandler(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	app.Put(pattern, app.convertHandler(handler), app.convertMiddleware(middleware...)...)
}

func (app *Application) DeleteHandler(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	app.Delete(pattern, app.convertHandler(handler), app.convertMiddleware(middleware...)...)
}

func (app *Application) PatchHandler(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	app.Patch(pattern, app.convertHandler(handler), app.convertMiddleware(middleware...)...)
}

func (app *Application) OptionsHandler(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	app.Options(pattern, app.convertHandler(handler), app.convertMiddleware(middleware...)...)
}

func (app *Application) HeadHandler(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	app.HEAD(pattern, app.convertHandler(handler), app.convertMiddleware(middleware...)...)
}

// Helper functions to convert between old and new function types
func (app *Application) convertHandler(handler HandlerFunc) httpInternal.HandlerFunc {
	return func(c httpInternal.Context) error {
		return handler(c.(*contextImpl.Context))
	}
}

func (app *Application) convertMiddleware(middleware ...MiddlewareFunc) []httpInternal.MiddlewareFunc {
	var converted []httpInternal.MiddlewareFunc
	for _, mw := range middleware {
		convertedMw := func(c httpInternal.Context) error {
			return mw(c.(*contextImpl.Context))
		}
		converted = append(converted, convertedMw)
	}
	return converted
}

// Router property for backward compatibility  
func (app *Application) Router() *Router {
	return &Router{Router: app.router.(*routerImpl.Router)}
}

// Constructor functions for backward compatibility
func NewRouter() *Router {
	return &Router{Router: routerImpl.NewRouter()}
}

func NewContext(w http.ResponseWriter, r *http.Request, app httpInternal.Application) Context {
	return contextImpl.NewContext(w, r, app)
}

func (app *Application) ConfigureLogging(config LoggingConfig) error {
	return SetupLogging(config)
}

func (app *Application) Logger() Logger {
	return Log()
}

func (app *Application) SetDebug(debug bool) {
	SetupErrorHandling(debug)
}

func (app *Application) GetErrorHandler() *ErrorHandler {
	return GetErrorHandler()
}

// HTTP Application interface implementation methods
func (app *Application) ErrorHandler() httpInternal.ErrorHandler {
	return &HTTPErrorHandlerAdapter{handler: GetErrorHandler()}
}

func (app *Application) TemplateEngine() httpInternal.TemplateEngine {
	if app.templateEngine != nil {
		return &TemplateEngineAdapter{engine: app.templateEngine}
	}
	return nil
}

// Adapter types to bridge old interfaces with new HTTP interfaces
type HTTPErrorHandlerAdapter struct {
	handler *ErrorHandler
}

func (a *HTTPErrorHandlerAdapter) Handle(ctx httpInternal.Context, err error) {
	// For now, use a simple error response until we fully migrate
	if httpErr, ok := err.(*HTTPError); ok {
		ctx.String(httpErr.Code, httpErr.Message)
	} else {
		ctx.String(500, "Internal Server Error")
	}
}

type TemplateEngineAdapter struct {
	engine *TemplateEngine
}

func (a *TemplateEngineAdapter) Render(template string, data interface{}) (string, error) {
	return a.engine.Render(template, data)
}

func LoggerMiddleware() httpInternal.MiddlewareFunc {
	return func(c httpInternal.Context) error {
		start := time.Now()
		
		err := c.Next()
		
		duration := time.Since(start)
		
		// Get response status if available (we'll need to enhance Context for this)
		status := 200 // Default assumption
		if err != nil {
			status = 500
		}
		
		// Create structured log context
		logContext := map[string]interface{}{
			"method":       c.Method(),
			"url":          c.URL(),
			"user_agent":   c.Header("User-Agent"),
			"remote_ip":    c.RemoteIP(),
			"duration_ms":  duration.Milliseconds(),
			"status_code":  status,
		}
		
		// Log at different levels based on status code
		message := fmt.Sprintf("%s %s", c.Method(), c.URL())
		
		if status >= 500 {
			Error(message, logContext)
		} else if status >= 400 {
			Warn(message, logContext)
		} else {
			Info(message, logContext)
		}
		
		return err
	}
}

func RecoveryMiddleware() httpInternal.MiddlewareFunc {
	return func(c httpInternal.Context) error {
		defer func() {
			if err := recover(); err != nil {
				// Create panic error with context
				panicErr := fmt.Errorf("panic recovered: %v", err)
				
				// Log the panic with full context
				panicContext := map[string]interface{}{
					"panic":      fmt.Sprintf("%v", err),
					"method":     c.Method(),
					"url":        c.URL(),
					"user_agent": c.Header("User-Agent"),
					"remote_ip":  c.RemoteIP(),
				}
				
				Fatal("Panic recovered in request handler", panicContext)
				
				// Create HTTP error and let error handler deal with it
				httpErr := NewHTTPErrorWithContext(500, "Internal Server Error", panicContext)
				httpErr.Internal = panicErr
				
				GetErrorHandler().Handle(c, httpErr)
				c.Abort()
			}
		}()
		
		return c.Next()
	}
}

func CORSMiddleware(origins ...string) httpInternal.MiddlewareFunc {
	allowedOrigins := make(map[string]bool)
	for _, origin := range origins {
		allowedOrigins[origin] = true
	}
	
	return func(c httpInternal.Context) error {
		origin := c.Header("Origin")
		
		if len(allowedOrigins) == 0 || allowedOrigins["*"] || allowedOrigins[origin] {
			c.SetHeader("Access-Control-Allow-Origin", origin)
		}
		
		c.SetHeader("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.SetHeader("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if c.Method() == "OPTIONS" {
			c.Status(204)
			c.Abort()
			return nil
		}
		
		return c.Next()
	}
}

// Schedule returns the task scheduler instance
func (app *Application) Schedule() *Schedule {
	// Try to get from container using Make (which handles singletons properly)
	if scheduler, err := app.container.Make("scheduler"); err == nil {
		return scheduler.(*Schedule)
	}

	// Get logger
	logger, _ := app.container.Make("logger")
	if logger == nil {
		logManager := NewLogManager()
		console := NewConsoleDriver(true)
		logManager.AddChannel("console", console, InfoLevel)
		logger = logManager.Channel("console")
	}

	// Get or create queue manager
	var queueManager QueueManager
	if qm, err := app.container.Make("queue"); err == nil {
		queueManager = qm.(QueueManager)
	}

	schedule := NewSchedule(logger.(Logger), queueManager)
	app.container.Singleton("scheduler", schedule)
	return schedule
}

// StartScheduler starts the task scheduler in the background
func (app *Application) StartScheduler() error {
	schedule := app.Schedule()
	if schedule != nil {
		return schedule.Start()
	}
	return fmt.Errorf("scheduler not configured")
}

// StopScheduler stops the task scheduler gracefully
func (app *Application) StopScheduler() error {
	schedule := app.Schedule()
	if schedule != nil {
		return schedule.Stop()
	}
	return nil
}