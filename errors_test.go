package onyx

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPError(t *testing.T) {
	err := NewHTTPError(404, "Not Found")
	
	if err.Code != 404 {
		t.Errorf("Expected code 404, got %d", err.Code)
	}
	
	if err.Message != "Not Found" {
		t.Errorf("Expected message 'Not Found', got '%s'", err.Message)
	}
	
	expectedError := "[404] Not Found"
	if err.Error() != expectedError {
		t.Errorf("Expected error string '%s', got '%s'", expectedError, err.Error())
	}
}

func TestHTTPErrorWithContext(t *testing.T) {
	context := map[string]interface{}{
		"resource": "user",
		"id":       123,
	}
	
	err := NewHTTPErrorWithContext(404, "User not found", context)
	
	if err.Context["resource"] != "user" {
		t.Error("Context not properly set")
	}
	
	if err.Context["id"] != 123 {
		t.Error("Context ID not properly set")
	}
}

func TestValidationErrors(t *testing.T) {
	errors := NewValidationErrors(
		NewValidationError("email", "Email is required", ""),
		NewValidationError("age", "Age must be a number", "abc"),
	)
	
	if len(errors.Errors) != 2 {
		t.Errorf("Expected 2 validation errors, got %d", len(errors.Errors))
	}
	
	errorMsg := errors.Error()
	if !strings.Contains(errorMsg, "email: Email is required") {
		t.Error("Email validation error not found in error message")
	}
	
	if !strings.Contains(errorMsg, "age: Age must be a number") {
		t.Error("Age validation error not found in error message")
	}
}

func TestErrorHandlerJSONResponse(t *testing.T) {
	app := New()
	app.SetDebug(true)
	
	app.GetHandler("/error", func(c Context) error {
		return BadRequest("Invalid request data")
	})
	
	req := httptest.NewRequest("GET", "/error", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	
	app.Router().ServeHTTP(w, req)
	
	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
	
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}
	
	errorData, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Fatal("Error data not found in response")
	}
	
	if errorData["message"] != "Invalid request data" {
		t.Errorf("Expected message 'Invalid request data', got '%v'", errorData["message"])
	}
	
	if errorData["status_code"] != float64(400) {
		t.Errorf("Expected status_code 400, got %v", errorData["status_code"])
	}
	
	// Debug mode should include debug information
	if _, exists := response["debug"]; !exists {
		t.Logf("Response: %+v", response)
		t.Error("Debug information not found in response (debug mode enabled)")
	}
}

func TestErrorHandlerHTMLResponse(t *testing.T) {
	app := New()
	app.SetDebug(false)
	
	app.GetHandler("/error", func(c Context) error {
		return InternalServerError("Something went wrong")
	})
	
	req := httptest.NewRequest("GET", "/error", nil)
	req.Header.Set("Accept", "text/html")
	w := httptest.NewRecorder()
	
	app.Router().ServeHTTP(w, req)
	
	if w.Code != 500 {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
	
	body := w.Body.String()
	if !strings.Contains(body, "Something went wrong") {
		t.Error("Error message not found in HTML response")
	}
	
	if !strings.Contains(body, "500") {
		t.Error("Status code not found in HTML response")
	}
	
	if !strings.Contains(body, "<!DOCTYPE html>") {
		t.Error("HTML structure not found in response")
	}
}

func TestValidationErrorResponse(t *testing.T) {
	app := New()
	
	app.PostHandler("/validate", func(c Context) error {
		validationErrors := NewValidationErrors(
			NewValidationError("name", "Name is required", ""),
			NewValidationError("email", "Email format is invalid", "invalid-email"),
		)
		return HandleError(c, validationErrors)
	})
	
	req := httptest.NewRequest("POST", "/validate", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	
	app.Router().ServeHTTP(w, req)
	
	if w.Code != 422 {
		t.Errorf("Expected status 422, got %d", w.Code)
	}
	
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}
	
	errorData := response["error"].(map[string]interface{})
	if errorData["message"] != "Validation failed" {
		t.Error("Validation failed message not found")
	}
	
	validationErrors := errorData["validation_errors"].([]interface{})
	if len(validationErrors) != 2 {
		t.Errorf("Expected 2 validation errors, got %d", len(validationErrors))
	}
}

func TestContextErrorMethods(t *testing.T) {
	app := New()
	
	// Test various context error methods
	app.GetHandler("/bad-request", func(c Context) error {
		return HandleBadRequest(c, "Bad request")
	})
	
	app.GetHandler("/unauthorized", func(c Context) error {
		return HandleUnauthorized(c, "Unauthorized")
	})
	
	app.GetHandler("/forbidden", func(c Context) error {
		return HandleForbidden(c, "Forbidden")
	})
	
	app.GetHandler("/not-found", func(c Context) error {
		return HandleNotFound(c, "Not found")
	})
	
	tests := []struct {
		path           string
		expectedStatus int
		expectedMessage string
	}{
		{"/bad-request", 400, "Bad request"},
		{"/unauthorized", 401, "Unauthorized"},
		{"/forbidden", 403, "Forbidden"},
		{"/not-found", 404, "Not found"},
	}
	
	for _, test := range tests {
		req := httptest.NewRequest("GET", test.path, nil)
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		
		app.Router().ServeHTTP(w, req)
		
		if w.Code != test.expectedStatus {
			t.Errorf("For %s: expected status %d, got %d", test.path, test.expectedStatus, w.Code)
		}
		
		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Errorf("Failed to parse JSON response for %s: %v", test.path, err)
			continue
		}
		
		errorData := response["error"].(map[string]interface{})
		if errorData["message"] != test.expectedMessage {
			t.Errorf("For %s: expected message '%s', got '%v'", test.path, test.expectedMessage, errorData["message"])
		}
	}
}

func TestPanicRecovery(t *testing.T) {
	app := New()
	app.SetDebug(true)
	
	app.GetHandler("/panic", func(c Context) error {
		panic("Test panic")
	})
	
	req := httptest.NewRequest("GET", "/panic", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	
	app.Router().ServeHTTP(w, req)
	
	if w.Code != 500 {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
	
	// Debug: Print actual response
	t.Logf("Response body: %s", w.Body.String())
	t.Logf("Response headers: %v", w.Header())
	
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}
	
	errorData := response["error"].(map[string]interface{})
	if errorData["status_code"] != float64(500) {
		t.Error("Panic not handled as 500 error")
	}
	
	// Should have panic context
	if _, exists := errorData["panic"]; !exists {
		t.Error("Panic context not found in error response")
	}
}

type TestErrorReporter struct {
	ReportedErrors []error
}

func (ter *TestErrorReporter) Report(err error, c Context) error {
	ter.ReportedErrors = append(ter.ReportedErrors, err)
	return nil
}

func TestCustomErrorReporter(t *testing.T) {
	// Test custom error reporter
	customReporter := &TestErrorReporter{
		ReportedErrors: make([]error, 0),
	}
	
	errorHandler := NewErrorHandler(false)
	errorHandler.AddReporter(customReporter)
	
	app := New()
	
	// Use the custom error handler temporarily
	originalHandler := globalErrorHandler
	globalErrorHandler = errorHandler
	defer func() { globalErrorHandler = originalHandler }()
	
	app.GetHandler("/error", func(c Context) error {
		return BadRequest("Test error")
	})
	
	req := httptest.NewRequest("GET", "/error", nil)
	w := httptest.NewRecorder()
	
	app.Router().ServeHTTP(w, req)
	
	if len(customReporter.ReportedErrors) < 1 {
		t.Errorf("Expected at least 1 reported error, got %d", len(customReporter.ReportedErrors))
		return
	}
	
	if customReporter.ReportedErrors[0].Error() != "[400] Test error" {
		t.Errorf("Expected '[400] Test error', got '%s'", customReporter.ReportedErrors[0].Error())
	}
}

func TestNotFoundRoutes(t *testing.T) {
	app := New()
	
	req := httptest.NewRequest("GET", "/nonexistent", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	
	app.Router().ServeHTTP(w, req)
	
	if w.Code != 404 {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
	
	// Debug: Print actual response
	t.Logf("Response body: %s", w.Body.String())
	t.Logf("Response headers: %v", w.Header())
	
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}
	
	errorData := response["error"].(map[string]interface{})
	if errorData["message"] != "Page not found" {
		t.Errorf("Expected 'Page not found', got '%v'", errorData["message"])
	}
}

func TestCommonHTTPErrors(t *testing.T) {
	tests := []struct {
		errorFunc      func(string) *HTTPError
		expectedCode   int
		expectedPrefix string
	}{
		{BadRequest, 400, "[400]"},
		{Unauthorized, 401, "[401]"},
		{Forbidden, 403, "[403]"},
		{NotFound, 404, "[404]"},
		{MethodNotAllowed, 405, "[405]"},
		{UnprocessableEntity, 422, "[422]"},
		{InternalServerError, 500, "[500]"},
		{ServiceUnavailable, 503, "[503]"},
	}
	
	for _, test := range tests {
		err := test.errorFunc("Test message")
		
		if err.Code != test.expectedCode {
			t.Errorf("Expected code %d, got %d", test.expectedCode, err.Code)
		}
		
		if !strings.HasPrefix(err.Error(), test.expectedPrefix) {
			t.Errorf("Expected error to start with '%s', got '%s'", test.expectedPrefix, err.Error())
		}
	}
}

func TestAbortMethods(t *testing.T) {
	app := New()
	
	app.GetHandler("/abort-with-error", func(c Context) error {
		return HandleError(c, BadRequest("Custom abort error"))
	})
	
	app.GetHandler("/abort-with-status", func(c Context) error {
		return HandleError(c, ServiceUnavailable("Service Unavailable"))
	})
	
	// Test AbortWithError
	req := httptest.NewRequest("GET", "/abort-with-error", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	
	app.Router().ServeHTTP(w, req)
	
	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
	
	// Test AbortWithStatus
	req = httptest.NewRequest("GET", "/abort-with-status", nil)
	req.Header.Set("Accept", "application/json")
	w = httptest.NewRecorder()
	
	app.Router().ServeHTTP(w, req)
	
	if w.Code != 503 {
		t.Errorf("Expected status 503, got %d", w.Code)
	}
}