package onyx

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLogLevels(t *testing.T) {
	var buffer bytes.Buffer
	jsonDriver := NewJSONDriver(&buffer)
	
	manager := NewLogManager()
	manager.AddChannel("test", jsonDriver, DebugLevel)
	
	logger := manager.Channel("test")
	
	// Test all log levels
	logger.Debug("Debug message")
	logger.Info("Info message")
	logger.Warn("Warning message")
	logger.Error("Error message")
	logger.Fatal("Fatal message")
	
	// Parse logged entries
	logs := strings.Split(strings.TrimSpace(buffer.String()), "\n")
	if len(logs) != 5 {
		t.Errorf("Expected 5 log entries, got %d", len(logs))
	}
	
	expectedLevels := []string{"debug", "info", "warning", "error", "fatal"}
	expectedMessages := []string{"Debug message", "Info message", "Warning message", "Error message", "Fatal message"}
	
	for i, logLine := range logs {
		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(logLine), &entry); err != nil {
			t.Errorf("Failed to parse log entry %d: %v", i, err)
			continue
		}
		
		if entry["level"] != expectedLevels[i] {
			t.Errorf("Expected level %s, got %s", expectedLevels[i], entry["level"])
		}
		
		if entry["message"] != expectedMessages[i] {
			t.Errorf("Expected message %s, got %s", expectedMessages[i], entry["message"])
		}
		
		if entry["channel"] != "test" {
			t.Errorf("Expected channel 'test', got %s", entry["channel"])
		}
	}
}

func TestLogLevelFiltering(t *testing.T) {
	var buffer bytes.Buffer
	jsonDriver := NewJSONDriver(&buffer)
	
	manager := NewLogManager()
	manager.AddChannel("test", jsonDriver, WarnLevel) // Only warn and above
	
	logger := manager.Channel("test")
	
	// These should be filtered out
	logger.Debug("Debug message")
	logger.Info("Info message")
	
	// These should pass through
	logger.Warn("Warning message")
	logger.Error("Error message")
	
	logs := strings.Split(strings.TrimSpace(buffer.String()), "\n")
	if len(logs) != 2 {
		t.Errorf("Expected 2 log entries (warn/error only), got %d", len(logs))
	}
}

func TestLogWithContext(t *testing.T) {
	var buffer bytes.Buffer
	jsonDriver := NewJSONDriver(&buffer)
	
	manager := NewLogManager()
	manager.AddChannel("test", jsonDriver, DebugLevel)
	
	logger := manager.Channel("test")
	
	context := map[string]interface{}{
		"user_id": 123,
		"action":  "login",
	}
	
	logger.Info("User logged in", context)
	
	var entry map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(buffer.String())), &entry); err != nil {
		t.Fatalf("Failed to parse log entry: %v", err)
	}
	
	contextData, ok := entry["context"].(map[string]interface{})
	if !ok {
		t.Fatal("Context data not found or wrong type")
	}
	
	if contextData["user_id"] != float64(123) { // JSON numbers are float64
		t.Errorf("Expected user_id 123, got %v", contextData["user_id"])
	}
	
	if contextData["action"] != "login" {
		t.Errorf("Expected action 'login', got %v", contextData["action"])
	}
}

func TestMultipleChannels(t *testing.T) {
	var consoleBuffer bytes.Buffer
	var fileBuffer bytes.Buffer
	
	consoleDriver := NewJSONDriver(&consoleBuffer)
	fileDriver := NewJSONDriver(&fileBuffer)
	
	manager := NewLogManager()
	manager.AddChannel("console", consoleDriver, InfoLevel)
	manager.AddChannel("file", fileDriver, DebugLevel)
	manager.SetDefaultChannel("console")
	
	// Log to default channel (console)
	manager.Default().Info("Console message")
	
	// Log to specific channel (file)
	manager.Channel("file").Debug("File debug message")
	
	// Check console buffer
	consoleLogs := strings.TrimSpace(consoleBuffer.String())
	if !strings.Contains(consoleLogs, "Console message") {
		t.Error("Console message not found in console buffer")
	}
	
	// Check file buffer
	fileLogs := strings.TrimSpace(fileBuffer.String())
	if !strings.Contains(fileLogs, "File debug message") {
		t.Error("File debug message not found in file buffer")
	}
}

func TestFileLogging(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")
	
	// Create file driver
	driver, err := NewFileDriver(logFile, 1024*1024, 3) // 1MB max, 3 files
	if err != nil {
		t.Fatalf("Failed to create file driver: %v", err)
	}
	defer driver.Close()
	
	manager := NewLogManager()
	manager.AddChannel("file", driver, DebugLevel)
	
	logger := manager.Channel("file")
	
	// Write some log entries
	logger.Info("Test message 1")
	logger.Error("Test message 2", map[string]interface{}{
		"error_code": 500,
		"details":    "Something went wrong",
	})
	
	// Close driver to flush
	driver.Close()
	
	// Check if file exists and contains expected content
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Fatal("Log file was not created")
	}
	
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	
	logContent := string(content)
	if !strings.Contains(logContent, "Test message 1") {
		t.Error("First log message not found in file")
	}
	
	if !strings.Contains(logContent, "Test message 2") {
		t.Error("Second log message not found in file")
	}
	
	if !strings.Contains(logContent, "error_code") {
		t.Error("Context data not found in file")
	}
}

func TestConsoleDriver(t *testing.T) {
	var buffer bytes.Buffer
	driver := &ConsoleDriver{
		colorize: false,
		writer:   &buffer,
	}
	
	entry := LogEntry{
		Level:     InfoLevel,
		Message:   "Test console message",
		Timestamp: time.Now(),
		Channel:   "test",
		Context: map[string]interface{}{
			"key": "value",
		},
	}
	
	err := driver.Write(entry)
	if err != nil {
		t.Fatalf("Failed to write to console driver: %v", err)
	}
	
	output := buffer.String()
	if !strings.Contains(output, "Test console message") {
		t.Error("Message not found in console output")
	}
	
	if !strings.Contains(output, "[INFO]") {
		t.Error("Log level not found in console output")
	}
	
	if !strings.Contains(output, "[test]") {
		t.Error("Channel not found in console output")
	}
}

func TestLoggerWithContext(t *testing.T) {
	var buffer bytes.Buffer
	jsonDriver := NewJSONDriver(&buffer)
	
	manager := NewLogManager()
	manager.AddChannel("test", jsonDriver, DebugLevel)
	
	logger := manager.Channel("test")
	
	// Create logger with context
	contextLogger := logger.WithContext(map[string]interface{}{
		"request_id": "req-123",
		"user_id":    456,
	})
	
	// Log with additional context
	contextLogger.Info("Operation completed", map[string]interface{}{
		"duration": "150ms",
		"status":   "success",
	})
	
	var entry map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(buffer.String())), &entry); err != nil {
		t.Fatalf("Failed to parse log entry: %v", err)
	}
	
	contextData, ok := entry["context"].(map[string]interface{})
	if !ok {
		t.Fatal("Context data not found")
	}
	
	// Check that both base context and additional context are present
	if contextData["request_id"] != "req-123" {
		t.Error("Base context not preserved")
	}
	
	if contextData["duration"] != "150ms" {
		t.Error("Additional context not added")
	}
}

func TestGlobalLogging(t *testing.T) {
	// Setup test logging
	config := LoggingConfig{
		DefaultChannel: "console",
		Console: struct {
			Level    LogLevel `json:"level"`
			Colorize bool     `json:"colorize"`
		}{
			Level:    DebugLevel,
			Colorize: false,
		},
	}
	
	err := SetupLogging(config)
	if err != nil {
		t.Fatalf("Failed to setup logging: %v", err)
	}
	
	// Test global functions
	Debug("Debug message")
	Info("Info message")
	Warn("Warning message")
	Error("Error message")
	
	// Test that they don't panic (output goes to console)
	// In a real test environment, you'd capture the output
}

func TestNullLogger(t *testing.T) {
	logger := &NullLogger{}
	
	// These should not panic
	logger.Debug("debug")
	logger.Info("info")
	logger.Warn("warn")
	logger.Error("error")
	logger.Fatal("fatal")
	
	contextLogger := logger.WithContext(map[string]interface{}{"key": "value"})
	channelLogger := logger.WithChannel("test")
	
	// Should return the same null logger
	if contextLogger != logger {
		t.Error("WithContext should return the same null logger instance")
	}
	
	if channelLogger != logger {
		t.Error("WithChannel should return the same null logger instance")
	}
}