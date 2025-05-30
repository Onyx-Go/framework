package onyx

import (
	"bytes"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestQuickLoggingFunctionality(t *testing.T) {
	// Test that logging doesn't break basic application functionality
	app := New()
	
	// Override with test logging
	var logBuffer bytes.Buffer
	globalLogManager = NewLogManager()
	jsonDriver := NewJSONDriver(&logBuffer)
	globalLogManager.AddChannel("test", jsonDriver, DebugLevel)
	globalLogManager.SetDefaultChannel("test")
	
	// Add test route
	app.GetHandler("/test", func(c Context) error {
		Log().Info("Test route hit")
		return c.String(200, "OK")
	})
	
	// Make request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	
	app.Router().ServeHTTP(w, req)
	
	// Check response
	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}
	
	if w.Body.String() != "OK" {
		t.Errorf("Expected 'OK', got '%s'", w.Body.String())
	}
	
	// Check that logs were generated
	logOutput := logBuffer.String()
	if logOutput == "" {
		t.Error("No logs generated")
	}
	
	// Should contain both middleware log and custom log
	if !strings.Contains(logOutput, "Test route hit") {
		t.Error("Custom log not found")
	}
	
	if !strings.Contains(logOutput, "GET /test") {
		t.Error("Middleware log not found")
	}
}

func TestLoggingPerformance(t *testing.T) {
	// Test that logging doesn't significantly impact performance
	var buffer bytes.Buffer
	driver := NewJSONDriver(&buffer)
	
	manager := NewLogManager()
	manager.AddChannel("perf", driver, InfoLevel)
	logger := manager.Channel("perf")
	
	// Time 1000 log operations
	start := time.Now()
	for i := 0; i < 1000; i++ {
		logger.Info("Performance test message", map[string]interface{}{
			"iteration": i,
			"data":      "test_data",
		})
	}
	duration := time.Since(start)
	
	// Should complete reasonably quickly (less than 100ms for 1000 logs)
	if duration > 100*time.Millisecond {
		t.Errorf("Logging too slow: %v for 1000 operations", duration)
	}
	
	// Verify all logs were written
	logLines := strings.Split(strings.TrimSpace(buffer.String()), "\n")
	if len(logLines) != 1000 {
		t.Errorf("Expected 1000 log lines, got %d", len(logLines))
	}
	
	fmt.Printf("Logging performance: %v for 1000 operations (%.2f μs per log)\n", 
		duration, float64(duration.Nanoseconds())/1000/1000)
}