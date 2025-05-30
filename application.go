package onyx

import (
	"fmt"
	"net/http"
	"time"
)

type Application struct {
	*Router
	server         *http.Server
	config         *Config
	container      *Container
	templateEngine *TemplateEngine
}

func New() *Application {
	router := NewRouter()
	app := &Application{
		Router:    router,
		config:    NewConfig(),
		container: NewContainer(),
	}
	
	router.app = app
	
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
	
	app.Use(LoggerMiddleware())
	app.Use(RecoveryMiddleware())
	app.Use(ErrorHandlerMiddleware(GetErrorHandler()))
	
	return app
}

func (app *Application) Start(address string) error {
	app.server = &http.Server{
		Addr:         address,
		Handler:      app.Router,
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

func (app *Application) TemplateEngine() *TemplateEngine {
	return app.templateEngine
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

func (app *Application) ErrorHandler() *ErrorHandler {
	return GetErrorHandler()
}

func LoggerMiddleware() MiddlewareFunc {
	return func(c *Context) error {
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
			"user_agent":   c.UserAgent(),
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

func RecoveryMiddleware() MiddlewareFunc {
	return func(c *Context) error {
		defer func() {
			if err := recover(); err != nil {
				// Create panic error with context
				panicErr := fmt.Errorf("panic recovered: %v", err)
				
				// Log the panic with full context
				panicContext := map[string]interface{}{
					"panic":      fmt.Sprintf("%v", err),
					"method":     c.Method(),
					"url":        c.URL(),
					"user_agent": c.UserAgent(),
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

func CORSMiddleware(origins ...string) MiddlewareFunc {
	allowedOrigins := make(map[string]bool)
	for _, origin := range origins {
		allowedOrigins[origin] = true
	}
	
	return func(c *Context) error {
		origin := c.GetHeader("Origin")
		
		if len(allowedOrigins) == 0 || allowedOrigins["*"] || allowedOrigins[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
		}
		
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
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