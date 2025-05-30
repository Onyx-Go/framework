package onyx

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// HTTPError represents an HTTP error with status code and message
type HTTPError struct {
	Code     int                    `json:"code"`
	Message  string                 `json:"message"`
	Internal error                 `json:"-"` // Hidden from JSON output
	Context  map[string]interface{} `json:"context,omitempty"`
}

func (e *HTTPError) Error() string {
	if e.Internal != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Internal)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// ValidationError represents validation errors
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   string `json:"value,omitempty"`
}

// ValidationErrors represents multiple validation errors
type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

func (ve *ValidationErrors) Error() string {
	var messages []string
	for _, err := range ve.Errors {
		messages = append(messages, fmt.Sprintf("%s: %s", err.Field, err.Message))
	}
	return "Validation failed: " + strings.Join(messages, ", ")
}

// ErrorReporter interface for custom error reporting
type ErrorReporter interface {
	Report(error, *Context) error
}

// ErrorRenderer interface for custom error rendering
type ErrorRenderer interface {
	Render(*Context, error) error
}

// ErrorHandler manages centralized error handling
type ErrorHandler struct {
	debug     bool
	reporters []ErrorReporter
	renderers map[string]ErrorRenderer
	templates map[int]string // HTTP status code to template mapping
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(debug bool) *ErrorHandler {
	return &ErrorHandler{
		debug:     debug,
		reporters: make([]ErrorReporter, 0),
		renderers: make(map[string]ErrorRenderer),
		templates: make(map[int]string),
	}
}

// AddReporter adds an error reporter
func (eh *ErrorHandler) AddReporter(reporter ErrorReporter) {
	eh.reporters = append(eh.reporters, reporter)
}

// AddRenderer adds a custom error renderer for specific content types
func (eh *ErrorHandler) AddRenderer(contentType string, renderer ErrorRenderer) {
	eh.renderers[contentType] = renderer
}

// SetTemplate sets a template for a specific HTTP status code
func (eh *ErrorHandler) SetTemplate(statusCode int, template string) {
	eh.templates[statusCode] = template
}

// Handle processes an error and renders appropriate response
func (eh *ErrorHandler) Handle(c *Context, err error) {
	if err == nil {
		return
	}

	// Report the error
	for _, reporter := range eh.reporters {
		if reportErr := reporter.Report(err, c); reportErr != nil {
			Error("Failed to report error", map[string]interface{}{
				"original_error": err.Error(),
				"report_error":   reportErr.Error(),
			})
		}
	}

	// Determine response format based on Accept header
	acceptHeader := c.GetHeader("Accept")
	contentType := eh.determineContentType(acceptHeader)

	// Try custom renderer first
	if renderer, exists := eh.renderers[contentType]; exists {
		if renderErr := renderer.Render(c, err); renderErr != nil {
			Error("Failed to render error with custom renderer", map[string]interface{}{
				"error":        err.Error(),
				"render_error": renderErr.Error(),
			})
			eh.renderDefault(c, err)
		}
		return
	}

	// Use default rendering
	eh.renderDefault(c, err)
}

func (eh *ErrorHandler) determineContentType(acceptHeader string) string {
	if strings.Contains(acceptHeader, "application/json") {
		return "application/json"
	}
	if strings.Contains(acceptHeader, "text/html") {
		return "text/html"
	}
	return "text/html" // Default
}

func (eh *ErrorHandler) renderDefault(c *Context, err error) {
	var statusCode int
	var message string
	var context map[string]interface{}

	// Extract error information
	switch e := err.(type) {
	case *HTTPError:
		statusCode = e.Code
		message = e.Message
		context = e.Context
	case *ValidationErrors:
		statusCode = 422
		message = "Validation failed"
		context = map[string]interface{}{
			"validation_errors": e.Errors,
		}
	default:
		statusCode = 500
		message = "Internal Server Error"
		if eh.debug {
			message = err.Error()
		}
	}

	// Determine response format
	acceptHeader := c.GetHeader("Accept")

	if strings.Contains(acceptHeader, "application/json") {
		eh.renderJSON(c, statusCode, message, context, err)
	} else {
		eh.renderHTML(c, statusCode, message, context, err)
	}
}

func (eh *ErrorHandler) renderJSON(c *Context, statusCode int, message string, context map[string]interface{}, originalErr error) {
	response := map[string]interface{}{
		"error": map[string]interface{}{
			"message":     message,
			"status_code": statusCode,
			"timestamp":   time.Now().Format(time.RFC3339),
		},
	}

	// Add context if available
	if context != nil {
		for k, v := range context {
			response["error"].(map[string]interface{})[k] = v
		}
	}

	// Add debug information if in debug mode
	if eh.debug {
		debugInfo := eh.getDebugInfo(originalErr)
		response["debug"] = debugInfo
	}

	c.ResponseWriter.Header().Set("Content-Type", "application/json")
	c.ResponseWriter.WriteHeader(statusCode)
	json.NewEncoder(c.ResponseWriter).Encode(response)
}

func (eh *ErrorHandler) renderHTML(c *Context, statusCode int, message string, context map[string]interface{}, originalErr error) {
	// Try to use custom template if available
	if template, exists := eh.templates[statusCode]; exists {
		if c.app.TemplateEngine() != nil {
			data := map[string]interface{}{
				"status_code": statusCode,
				"message":     message,
				"context":     context,
			}

			if eh.debug {
				data["debug"] = eh.getDebugInfo(originalErr)
			}

			if rendered, err := c.app.TemplateEngine().Render(template, ViewData(data)); err == nil {
				c.ResponseWriter.Header().Set("Content-Type", "text/html")
				c.ResponseWriter.WriteHeader(statusCode)
				c.ResponseWriter.Write([]byte(rendered))
				return
			}
		}
	}

	// Fall back to simple HTML response
	html := eh.generateDefaultHTML(statusCode, message, context, originalErr)
	c.ResponseWriter.Header().Set("Content-Type", "text/html")
	c.ResponseWriter.WriteHeader(statusCode)
	c.ResponseWriter.Write([]byte(html))
}

func (eh *ErrorHandler) generateDefaultHTML(statusCode int, message string, context map[string]interface{}, originalErr error) string {
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Error %d</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; background: #f8f9fa; }
        .error-container { background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .error-code { color: #dc3545; font-size: 48px; font-weight: bold; margin-bottom: 10px; }
        .error-message { color: #495057; font-size: 24px; margin-bottom: 20px; }
        .error-details { background: #f8f9fa; padding: 15px; border-radius: 4px; margin-top: 20px; }
        .debug-info { background: #fff3cd; padding: 15px; border-radius: 4px; margin-top: 20px; border-left: 4px solid #ffc107; }
        .stack-trace { background: #f8f9fa; padding: 10px; border-radius: 4px; font-family: monospace; white-space: pre-wrap; font-size: 12px; }
    </style>
</head>
<body>
    <div class="error-container">
        <div class="error-code">%d</div>
        <div class="error-message">%s</div>`, statusCode, statusCode, message)

	// Add context information
	if context != nil && len(context) > 0 {
		html += `<div class="error-details"><h4>Additional Information:</h4><ul>`
		for k, v := range context {
			html += fmt.Sprintf("<li><strong>%s:</strong> %v</li>", k, v)
		}
		html += `</ul></div>`
	}

	// Add debug information if in debug mode
	if eh.debug && originalErr != nil {
		debugInfo := eh.getDebugInfo(originalErr)
		html += `<div class="debug-info">
            <h4>Debug Information:</h4>
            <p><strong>Error:</strong> ` + fmt.Sprintf("%v", originalErr) + `</p>`

		if stackTrace, ok := debugInfo["stack_trace"].(string); ok {
			html += `<h4>Stack Trace:</h4><div class="stack-trace">` + stackTrace + `</div>`
		}

		if file, ok := debugInfo["file"].(string); ok {
			if line, ok := debugInfo["line"].(int); ok {
				html += fmt.Sprintf(`<p><strong>Location:</strong> %s:%d</p>`, file, line)
			}
		}

		html += `</div>`
	}

	html += `
    </div>
</body>
</html>`

	return html
}

func (eh *ErrorHandler) getDebugInfo(err error) map[string]interface{} {
	debugInfo := map[string]interface{}{
		"error_type": fmt.Sprintf("%T", err),
		"timestamp":  time.Now().Format(time.RFC3339),
	}

	// Get stack trace
	stack := make([]byte, 4096)
	length := runtime.Stack(stack, false)
	debugInfo["stack_trace"] = string(stack[:length])

	// Get caller information
	if pc, file, line, ok := runtime.Caller(4); ok {
		if fn := runtime.FuncForPC(pc); fn != nil {
			debugInfo["function"] = fn.Name()
		}
		debugInfo["file"] = file
		debugInfo["line"] = line
	}

	// Add Go runtime info
	debugInfo["go_version"] = runtime.Version()
	debugInfo["num_goroutines"] = runtime.NumGoroutine()

	return debugInfo
}

// Built-in error reporters
type LogErrorReporter struct{}

func (ler *LogErrorReporter) Report(err error, c *Context) error {
	errorContext := map[string]interface{}{
		"error":      err.Error(),
		"method":     c.Method(),
		"url":        c.URL(),
		"user_agent": c.UserAgent(),
		"remote_ip":  c.RemoteIP(),
		"timestamp":  time.Now().Format(time.RFC3339),
	}

	// Add HTTP status code if it's an HTTPError
	if httpErr, ok := err.(*HTTPError); ok {
		errorContext["status_code"] = httpErr.Code
		errorContext["http_message"] = httpErr.Message
		if httpErr.Context != nil {
			errorContext["error_context"] = httpErr.Context
		}
	}

	// Log at appropriate level based on error type
	if httpErr, ok := err.(*HTTPError); ok {
		if httpErr.Code >= 500 {
			Error("HTTP Error (5xx)", errorContext)
		} else if httpErr.Code >= 400 {
			Warn("HTTP Error (4xx)", errorContext)
		} else {
			Info("HTTP Response", errorContext)
		}
	} else {
		Error("Application Error", errorContext)
	}

	return nil
}

// Convenience functions for creating common HTTP errors
func NewHTTPError(code int, message string) *HTTPError {
	return &HTTPError{
		Code:    code,
		Message: message,
	}
}

func NewHTTPErrorWithContext(code int, message string, context map[string]interface{}) *HTTPError {
	return &HTTPError{
		Code:    code,
		Message: message,
		Context: context,
	}
}

func NewHTTPErrorWithInternal(code int, message string, internal error) *HTTPError {
	return &HTTPError{
		Code:     code,
		Message:  message,
		Internal: internal,
	}
}

// Common HTTP errors
func BadRequest(message string) *HTTPError {
	return NewHTTPError(400, message)
}

func Unauthorized(message string) *HTTPError {
	return NewHTTPError(401, message)
}

func Forbidden(message string) *HTTPError {
	return NewHTTPError(403, message)
}

func NotFound(message string) *HTTPError {
	return NewHTTPError(404, message)
}

func MethodNotAllowed(message string) *HTTPError {
	return NewHTTPError(405, message)
}

func UnprocessableEntity(message string) *HTTPError {
	return NewHTTPError(422, message)
}

func InternalServerError(message string) *HTTPError {
	return NewHTTPError(500, message)
}

func ServiceUnavailable(message string) *HTTPError {
	return NewHTTPError(503, message)
}

// Validation error helpers
func NewValidationError(field, message, value string) ValidationError {
	return ValidationError{
		Field:   field,
		Message: message,
		Value:   value,
	}
}

func NewValidationErrors(errors ...ValidationError) *ValidationErrors {
	return &ValidationErrors{
		Errors: errors,
	}
}

// Error handling middleware
func ErrorHandlerMiddleware(handler *ErrorHandler) MiddlewareFunc {
	return func(c *Context) error {
		err := c.Next()
		if err != nil {
			// Always use the current global handler to support runtime debug changes
			GetErrorHandler().Handle(c, err)
			c.Abort()
		}
		return nil
	}
}

// Global error handler instance
var globalErrorHandler *ErrorHandler

// SetupErrorHandling configures global error handling
func SetupErrorHandling(debug bool) {
	globalErrorHandler = NewErrorHandler(debug)
	
	// Add default log reporter
	globalErrorHandler.AddReporter(&LogErrorReporter{})
}

// GetErrorHandler returns the global error handler
func GetErrorHandler() *ErrorHandler {
	if globalErrorHandler == nil {
		SetupErrorHandling(false)
	}
	return globalErrorHandler
}

// Context helper methods for error handling
func (c *Context) Error(err error) error {
	if globalErrorHandler != nil {
		globalErrorHandler.Handle(c, err)
		c.Abort()
		return nil
	}
	return err
}

func (c *Context) BadRequest(message string) error {
	return c.Error(BadRequest(message))
}

func (c *Context) Unauthorized(message string) error {
	return c.Error(Unauthorized(message))
}

func (c *Context) Forbidden(message string) error {
	return c.Error(Forbidden(message))
}

func (c *Context) NotFound(message string) error {
	return c.Error(NotFound(message))
}

func (c *Context) UnprocessableEntity(message string) error {
	return c.Error(UnprocessableEntity(message))
}

func (c *Context) InternalServerError(message string) error {
	return c.Error(InternalServerError(message))
}

func (c *Context) ValidationErrors(errors ...ValidationError) error {
	return c.Error(NewValidationErrors(errors...))
}

// AbortWithError aborts the request chain and handles the error
func (c *Context) AbortWithError(err error) {
	c.Error(err)
}

// AbortWithStatus aborts with a status code and default message
func (c *Context) AbortWithStatus(code int) {
	message := http.StatusText(code)
	if message == "" {
		message = "HTTP Error " + strconv.Itoa(code)
	}
	c.Error(NewHTTPError(code, message))
}