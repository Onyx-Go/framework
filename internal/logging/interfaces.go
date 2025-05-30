package logging

import (
	"context"
	"io"
	"time"
)

// LogLevel represents the severity level of a log entry
type LogLevel int

const (
	DebugLevel LogLevel = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

// LogEntry represents a single log entry
type LogEntry struct {
	Level     LogLevel               `json:"level"`
	Message   string                 `json:"message"`
	Timestamp time.Time              `json:"timestamp"`
	Context   map[string]interface{} `json:"context,omitempty"`
	Channel   string                 `json:"channel,omitempty"`
	Extra     map[string]interface{} `json:"extra,omitempty"`
}

// Logger interface defines the logging contract with context support
type Logger interface {
	// Context-aware logging methods
	DebugContext(ctx context.Context, message string, args ...map[string]interface{})
	InfoContext(ctx context.Context, message string, args ...map[string]interface{})
	WarnContext(ctx context.Context, message string, args ...map[string]interface{})
	ErrorContext(ctx context.Context, message string, args ...map[string]interface{})
	FatalContext(ctx context.Context, message string, args ...map[string]interface{})
	LogContext(ctx context.Context, level LogLevel, message string, args ...map[string]interface{})
	
	// Legacy methods (for backward compatibility)
	Debug(message string, context ...map[string]interface{})
	Info(message string, context ...map[string]interface{})
	Warn(message string, context ...map[string]interface{})
	Error(message string, context ...map[string]interface{})
	Fatal(message string, context ...map[string]interface{})
	Log(level LogLevel, message string, context ...map[string]interface{})
	
	// Logger modifiers
	WithContext(context map[string]interface{}) Logger
	WithChannel(channel string) Logger
}

// Driver interface for different logging backends
type Driver interface {
	Write(ctx context.Context, entry LogEntry) error
	Close() error
}

// Manager interface for managing logging channels and drivers
type Manager interface {
	// Channel management
	AddChannel(name string, driver Driver, level LogLevel)
	Channel(name string) Logger
	SetDefaultChannel(name string)
	Default() Logger
	
	// Lifecycle
	Close() error
}

// Formatter interface for formatting log entries
type Formatter interface {
	Format(entry LogEntry) ([]byte, error)
}

// Config represents logging configuration
type Config struct {
	DefaultChannel string         `json:"default_channel"`
	Console        ConsoleConfig  `json:"console"`
	File           FileConfig     `json:"file"`
	JSON           JSONConfig     `json:"json"`
}

// ConsoleConfig represents console logging configuration
type ConsoleConfig struct {
	Level    LogLevel `json:"level"`
	Colorize bool     `json:"colorize"`
}

// FileConfig represents file logging configuration
type FileConfig struct {
	Enabled  bool     `json:"enabled"`
	Path     string   `json:"path"`
	Level    LogLevel `json:"level"`
	MaxSize  int64    `json:"max_size"`
	MaxFiles int      `json:"max_files"`
}

// JSONConfig represents JSON logging configuration
type JSONConfig struct {
	Enabled bool     `json:"enabled"`
	Path    string   `json:"path"`
	Level   LogLevel `json:"level"`
}

// WriterDriver interface for drivers that support custom writers
type WriterDriver interface {
	Driver
	SetWriter(writer io.Writer)
}

// RotatingDriver interface for drivers that support log rotation
type RotatingDriver interface {
	Driver
	Rotate() error
}

// BufferedDriver interface for drivers that support buffering
type BufferedDriver interface {
	Driver
	Flush() error
}