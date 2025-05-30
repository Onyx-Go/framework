package errors

// ErrorReporter interface for custom error reporting
type ErrorReporter interface {
	Report(error, Context) error
}

// ErrorRenderer interface for custom error rendering
type ErrorRenderer interface {
	Render(Context, error) error
}

// Context interface represents the minimal context needed for error handling
// This prevents circular imports while maintaining functionality
type Context interface {
	GetHeader(string) string
	Method() string
	URL() string
	UserAgent() string
	RemoteIP() string
	ResponseWriter() ResponseWriter
	App() Application
	Abort()
	Next() error
}

// ResponseWriter interface for HTTP response writing
type ResponseWriter interface {
	Header() map[string][]string
	WriteHeader(int)
	Write([]byte) (int, error)
}

// Application interface for accessing app services
type Application interface {
	TemplateEngine() TemplateEngine
}

// TemplateEngine interface for template rendering
type TemplateEngine interface {
	Render(string, interface{}) (string, error)
}