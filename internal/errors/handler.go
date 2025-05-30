package errors

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"time"
)

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

// SetDebug enables or disables debug mode
func (eh *ErrorHandler) SetDebug(debug bool) {
	eh.debug = debug
}

// IsDebug returns current debug mode state
func (eh *ErrorHandler) IsDebug() bool {
	return eh.debug
}

// Handle processes an error and renders appropriate response
func (eh *ErrorHandler) Handle(c Context, err error) {
	if err == nil {
		return
	}

	// Report the error
	for _, reporter := range eh.reporters {
		if reportErr := reporter.Report(err, c); reportErr != nil {
			// Log reporter error (would need logger interface)
			// For now, we'll skip to avoid circular dependency
		}
	}

	// Determine response format based on Accept header
	acceptHeader := c.GetHeader("Accept")
	contentType := eh.determineContentType(acceptHeader)

	// Try custom renderer first
	if renderer, exists := eh.renderers[contentType]; exists {
		if renderErr := renderer.Render(c, err); renderErr != nil {
			// Fallback to default rendering
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

func (eh *ErrorHandler) renderDefault(c Context, err error) {
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

func (eh *ErrorHandler) renderJSON(c Context, statusCode int, message string, context map[string]interface{}, originalErr error) {
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

	writer := c.ResponseWriter()
	if headers := writer.Header(); headers != nil {
		headers["Content-Type"] = []string{"application/json"}
	}
	writer.WriteHeader(statusCode)
	json.NewEncoder(writer).Encode(response)
}

func (eh *ErrorHandler) renderHTML(c Context, statusCode int, message string, context map[string]interface{}, originalErr error) {
	// Try to use custom template if available
	if template, exists := eh.templates[statusCode]; exists {
		if app := c.App(); app != nil {
			if engine := app.TemplateEngine(); engine != nil {
				data := map[string]interface{}{
					"status_code": statusCode,
					"message":     message,
					"context":     context,
				}

				if eh.debug {
					data["debug"] = eh.getDebugInfo(originalErr)
				}

				if rendered, err := engine.Render(template, data); err == nil {
					writer := c.ResponseWriter()
					if headers := writer.Header(); headers != nil {
						headers["Content-Type"] = []string{"text/html"}
					}
					writer.WriteHeader(statusCode)
					writer.Write([]byte(rendered))
					return
				}
			}
		}
	}

	// Fall back to simple HTML response
	html := eh.generateDefaultHTML(statusCode, message, context, originalErr)
	writer := c.ResponseWriter()
	if headers := writer.Header(); headers != nil {
		headers["Content-Type"] = []string{"text/html"}
	}
	writer.WriteHeader(statusCode)
	writer.Write([]byte(html))
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