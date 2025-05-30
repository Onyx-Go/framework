package errors

import (
	"fmt"
	"net/http"
	"strconv"
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

// Unwrap returns the internal error for error wrapping support
func (e *HTTPError) Unwrap() error {
	return e.Internal
}

// NewHTTPError creates a new HTTP error
func NewHTTPError(code int, message string) *HTTPError {
	return &HTTPError{
		Code:    code,
		Message: message,
	}
}

// NewHTTPErrorWithContext creates a new HTTP error with context
func NewHTTPErrorWithContext(code int, message string, context map[string]interface{}) *HTTPError {
	return &HTTPError{
		Code:    code,
		Message: message,
		Context: context,
	}
}

// NewHTTPErrorWithInternal creates a new HTTP error with internal error
func NewHTTPErrorWithInternal(code int, message string, internal error) *HTTPError {
	return &HTTPError{
		Code:     code,
		Message:  message,
		Internal: internal,
	}
}

// Common HTTP error constructors
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

// StatusText returns HTTP status text with fallback
func StatusText(code int) string {
	message := http.StatusText(code)
	if message == "" {
		message = "HTTP Error " + strconv.Itoa(code)
	}
	return message
}