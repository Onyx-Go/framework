package logging

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestLogLevels(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{DebugLevel, "debug"},
		{InfoLevel, "info"},
		{WarnLevel, "warning"},
		{ErrorLevel, "error"},
		{FatalLevel, "fatal"},
	}
	
	for _, test := range tests {
		if GetLevelName(test.level) != test.expected {
			t.Errorf("Expected level name %s, got %s", test.expected, GetLevelName(test.level))
		}
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected LogLevel
	}{
		{"debug", DebugLevel},
		{"DEBUG", DebugLevel},
		{"info", InfoLevel},
		{"INFO", InfoLevel},
		{"warn", WarnLevel},
		{"warning", WarnLevel},
		{"error", ErrorLevel},
		{"fatal", FatalLevel},
		{"unknown", InfoLevel}, // default
	}
	
	for _, test := range tests {
		if ParseLogLevel(test.input) != test.expected {
			t.Errorf("Expected level %v for input %s, got %v", test.expected, test.input, ParseLogLevel(test.input))
		}
	}
}

func TestManager(t *testing.T) {
	manager := NewManager()
	
	// Test adding channels
	buffer := &bytes.Buffer{}
	consoleDriver := NewConsoleDriver(false)
	consoleDriver.SetWriter(buffer)
	
	manager.AddChannel("test", consoleDriver, InfoLevel)
	
	// Test getting channel
	logger := manager.Channel("test")
	if logger == nil {
		t.Fatal("Expected logger, got nil")
	}
	
	// Test logging
	logger.Info("test message")
	
	output := buffer.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("Expected output to contain 'test message', got: %s", output)
	}
	
	// Test default channel
	manager.SetDefaultChannel("test")
	defaultLogger := manager.Default()
	if defaultLogger == nil {
		t.Fatal("Expected default logger, got nil")
	}
	
	// Test non-existent channel returns default
	nonExistentLogger := manager.Channel("nonexistent")
	if nonExistentLogger == nil {
		t.Fatal("Expected logger, got nil")
	}
}

func TestChannelWithContext(t *testing.T) {
	buffer := &bytes.Buffer{}
	consoleDriver := NewConsoleDriver(false)
	consoleDriver.SetWriter(buffer)
	
	manager := NewManager()
	manager.AddChannel("test", consoleDriver, DebugLevel)
	
	logger := manager.Channel("test")
	
	// Test context-aware logging
	ctx := context.WithValue(context.Background(), "request_id", "test-123")
	logger.InfoContext(ctx, "test message with context")
	
	output := buffer.String()
	if !strings.Contains(output, "test message with context") {
		t.Errorf("Expected output to contain message, got: %s", output)
	}
}

func TestChannelWithContextData(t *testing.T) {
	buffer := &bytes.Buffer{}
	consoleDriver := NewConsoleDriver(false)
	consoleDriver.SetWriter(buffer)
	
	manager := NewManager()
	manager.AddChannel("test", consoleDriver, DebugLevel)
	
	logger := manager.Channel("test")
	
	// Test with context data
	contextData := map[string]interface{}{
		"user_id": 123,
		"action":  "login",
	}
	
	logger.Info("user action", contextData)
	
	output := buffer.String()
	if !strings.Contains(output, "user action") {
		t.Errorf("Expected output to contain message, got: %s", output)
	}
	if !strings.Contains(output, "user_id") {
		t.Errorf("Expected output to contain context data, got: %s", output)
	}
}

func TestLogLevelFiltering(t *testing.T) {
	buffer := &bytes.Buffer{}
	consoleDriver := NewConsoleDriver(false)
	consoleDriver.SetWriter(buffer)
	
	manager := NewManager()
	manager.AddChannel("test", consoleDriver, WarnLevel) // Only warn and above
	
	logger := manager.Channel("test")
	
	// These should not appear
	logger.Debug("debug message")
	logger.Info("info message")
	
	// These should appear
	logger.Warn("warn message")
	logger.Error("error message")
	
	output := buffer.String()
	if strings.Contains(output, "debug message") {
		t.Error("Debug message should be filtered out")
	}
	if strings.Contains(output, "info message") {
		t.Error("Info message should be filtered out")
	}
	if !strings.Contains(output, "warn message") {
		t.Errorf("Warn message should appear in output: %s", output)
	}
	if !strings.Contains(output, "error message") {
		t.Errorf("Error message should appear in output: %s", output)
	}
}

func TestConsoleDriver(t *testing.T) {
	buffer := &bytes.Buffer{}
	driver := NewConsoleDriver(false)
	driver.SetWriter(buffer)
	
	entry := LogEntry{
		Level:     InfoLevel,
		Message:   "test message",
		Timestamp: time.Now(),
		Channel:   "test",
		Context:   map[string]interface{}{"key": "value"},
	}
	
	err := driver.Write(context.Background(), entry)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	output := buffer.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("Expected output to contain 'test message', got: %s", output)
	}
	if !strings.Contains(output, "[INFO]") {
		t.Errorf("Expected output to contain '[INFO]', got: %s", output)
	}
}

func TestJSONDriver(t *testing.T) {
	buffer := &bytes.Buffer{}
	driver := NewJSONDriver(buffer)
	
	entry := LogEntry{
		Level:     InfoLevel,
		Message:   "test message",
		Timestamp: time.Now(),
		Channel:   "test",
		Context:   map[string]interface{}{"key": "value"},
	}
	
	err := driver.Write(context.Background(), entry)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	output := buffer.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("Expected JSON output to contain 'test message', got: %s", output)
	}
	if !strings.Contains(output, "\"level\":\"info\"") {
		t.Errorf("Expected JSON output to contain level info, got: %s", output)
	}
}

func TestService(t *testing.T) {
	config := DefaultConfig()
	service, err := NewService(config)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}
	defer service.Close()
	
	// Test basic logging
	service.Info("test service message")
	
	// Test context-aware logging
	ctx := context.WithValue(context.Background(), "request_id", "test-456")
	service.InfoContext(ctx, "test context message")
	
	// Test channel access
	logger := service.Channel("console")
	if logger == nil {
		t.Fatal("Expected console logger, got nil")
	}
}

func TestGlobalLogging(t *testing.T) {
	// Setup global logging
	config := DefaultConfig()
	err := Setup(config)
	if err != nil {
		t.Fatalf("Failed to setup global logging: %v", err)
	}
	
	// Test global functions
	Debug("debug message")
	Info("info message")
	Warn("warn message")
	Error("error message")
	
	// Test context-aware global functions
	ctx := context.WithValue(context.Background(), "user_id", "user-789")
	InfoContext(ctx, "context info message")
	
	// Test logger access
	logger := GetLogger()
	if logger == nil {
		t.Fatal("Expected global logger, got nil")
	}
	
	// Test channel access
	consoleLogger := Channel("console")
	if consoleLogger == nil {
		t.Fatal("Expected console logger, got nil")
	}
}

func TestFileDriver(t *testing.T) {
	// Create a temporary file for testing
	tempFile := "/tmp/test_onyx.log"
	defer os.Remove(tempFile)
	defer os.Remove(tempFile + ".1") // cleanup rotation file
	
	driver, err := NewFileDriver(tempFile, 1024, 2)
	if err != nil {
		t.Fatalf("Failed to create file driver: %v", err)
	}
	defer driver.Close()
	
	entry := LogEntry{
		Level:     InfoLevel,
		Message:   "test file message",
		Timestamp: time.Now(),
		Channel:   "test",
		Context:   map[string]interface{}{"key": "value"},
	}
	
	err = driver.Write(context.Background(), entry)
	if err != nil {
		t.Fatalf("Unexpected error writing to file: %v", err)
	}
	
	// Check if file was created and contains content
	if _, err := os.Stat(tempFile); os.IsNotExist(err) {
		t.Fatal("Expected log file to be created")
	}
	
	content, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	
	if !strings.Contains(string(content), "test file message") {
		t.Errorf("Expected file content to contain 'test file message', got: %s", string(content))
	}
}