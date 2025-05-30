package onyx

import (
	"bytes"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestApplicationLogging(t *testing.T) {
	// Create application
	app := New()
	
	// Configure logging to capture output
	var logBuffer bytes.Buffer
	
	// Override global log manager for testing
	globalLogManager = NewLogManager()
	jsonDriver := NewJSONDriver(&logBuffer)
	globalLogManager.AddChannel("json", jsonDriver, DebugLevel)
	globalLogManager.SetDefaultChannel("json")
	
	// Add test route
	app.Get("/test", func(c *Context) error {
		c.Log().Info("Test route accessed", map[string]interface{}{
			"custom_data": "test_value",
		})
		return c.String(200, "OK")
	})
	
	// Add route that causes an error
	app.Get("/error", func(c *Context) error {
		c.Log().Error("Simulated error", map[string]interface{}{
			"error_type": "test_error",
		})
		return c.String(500, "Internal Server Error")
	})
	
	// Test successful request
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "test-agent")
	w := httptest.NewRecorder()
	
	app.ServeHTTP(w, req)
	
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	// Check that logs were generated
	logOutput := logBuffer.String()
	if logOutput == "" {
		t.Error("No log output generated")
	}
	
	// Should contain request log and custom log
	logLines := strings.Split(strings.TrimSpace(logOutput), "\n")
	if len(logLines) < 2 {
		t.Errorf("Expected at least 2 log entries, got %d", len(logLines))
	}
	
	// Test error request
	logBuffer.Reset()
	req = httptest.NewRequest("GET", "/error", nil)
	w = httptest.NewRecorder()
	
	app.ServeHTTP(w, req)
	
	logOutput = logBuffer.String()
	if !strings.Contains(logOutput, "Simulated error") {
		t.Error("Custom error log not found")
	}
	
	if !strings.Contains(logOutput, "GET /error") {
		t.Error("Request log not found")
	}
}

func TestContextLogging(t *testing.T) {
	// Create application
	app := New()
	
	// Capture logs
	var logBuffer bytes.Buffer
	globalLogManager = NewLogManager()
	jsonDriver := NewJSONDriver(&logBuffer)
	globalLogManager.AddChannel("test", jsonDriver, DebugLevel)
	globalLogManager.SetDefaultChannel("test")
	
	// Add test route that uses context logging
	app.Post("/user", func(c *Context) error {
		logger := c.Log()
		logger.Info("User creation attempt", map[string]interface{}{
			"action": "create_user",
			"ip":     c.RemoteIP(),
		})
		return c.JSON(201, map[string]string{"status": "created"})
	})
	
	// Make request
	req := httptest.NewRequest("POST", "/user", nil)
	req.Header.Set("User-Agent", "test-client")
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	
	app.ServeHTTP(w, req)
	
	// Check logs contain request context
	logOutput := logBuffer.String()
	if !strings.Contains(logOutput, "POST") {
		t.Error("HTTP method not found in logs")
	}
	
	if !strings.Contains(logOutput, "/user") {
		t.Error("URL not found in logs")
	}
	
	if !strings.Contains(logOutput, "test-client") {
		t.Error("User agent not found in logs")
	}
	
	if !strings.Contains(logOutput, "User creation attempt") {
		t.Error("Custom log message not found")
	}
}

func TestRecoveryMiddlewareLogging(t *testing.T) {
	// Create application
	app := New()
	
	// Capture logs
	var logBuffer bytes.Buffer
	globalLogManager = NewLogManager()
	jsonDriver := NewJSONDriver(&logBuffer)
	globalLogManager.AddChannel("test", jsonDriver, DebugLevel)
	globalLogManager.SetDefaultChannel("test")
	
	// Add route that panics
	app.Get("/panic", func(c *Context) error {
		panic("Test panic")
	})
	
	// Make request
	req := httptest.NewRequest("GET", "/panic", nil)
	w := httptest.NewRecorder()
	
	app.ServeHTTP(w, req)
	
	// Should return 500
	if w.Code != 500 {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
	
	// Check that panic was logged
	logOutput := logBuffer.String()
	if !strings.Contains(logOutput, "Panic recovered") {
		t.Error("Panic recovery log not found")
	}
	
	if !strings.Contains(logOutput, "Test panic") {
		t.Error("Panic message not found in logs")
	}
	
	if !strings.Contains(logOutput, "fatal") {
		t.Error("Fatal log level not found")
	}
}

func TestLoggingConfiguration(t *testing.T) {
	app := New()
	
	// Test configuring logging after app creation
	config := LoggingConfig{
		DefaultChannel: "file",
		File: struct {
			Enabled  bool     `json:"enabled"`
			Path     string   `json:"path"`
			Level    LogLevel `json:"level"`
			MaxSize  int64    `json:"max_size"`
			MaxFiles int      `json:"max_files"`
		}{
			Enabled:  true,
			Path:     t.TempDir() + "/test.log",
			Level:    WarnLevel,
			MaxSize:  1024,
			MaxFiles: 2,
		},
	}
	
	err := app.ConfigureLogging(config)
	if err != nil {
		t.Fatalf("Failed to configure logging: %v", err)
	}
	
	// Test that logger is accessible
	logger := app.Logger()
	if logger == nil {
		t.Error("Logger should not be nil")
	}
	
	// Test logging (this will go to file)
	logger.Warn("Test warning message")
}