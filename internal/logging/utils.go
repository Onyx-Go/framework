package logging

import (
	"strings"
)

// LogLevel string mapping
var logLevelNames = map[LogLevel]string{
	DebugLevel: "debug",
	InfoLevel:  "info",
	WarnLevel:  "warning",
	ErrorLevel: "error",
	FatalLevel: "fatal",
}

// Color mapping for console output
var logLevelColors = map[LogLevel]string{
	DebugLevel: "\033[36m", // Cyan
	InfoLevel:  "\033[32m", // Green
	WarnLevel:  "\033[33m", // Yellow
	ErrorLevel: "\033[31m", // Red
	FatalLevel: "\033[35m", // Magenta
}

const colorReset = "\033[0m"

// GetLevelName returns the string name for a log level
func GetLevelName(level LogLevel) string {
	if name, exists := logLevelNames[level]; exists {
		return name
	}
	return "unknown"
}

// GetLevelColor returns the color code for a log level
func GetLevelColor(level LogLevel) string {
	if color, exists := logLevelColors[level]; exists {
		return color
	}
	return ""
}

// GetColorReset returns the color reset code
func GetColorReset() string {
	return colorReset
}

// ParseLogLevel parses a string into a LogLevel
func ParseLogLevel(level string) LogLevel {
	switch strings.ToLower(level) {
	case "debug":
		return DebugLevel
	case "info":
		return InfoLevel
	case "warn", "warning":
		return WarnLevel
	case "error":
		return ErrorLevel
	case "fatal":
		return FatalLevel
	default:
		return InfoLevel
	}
}

// DefaultConfig returns a default logging configuration
func DefaultConfig() Config {
	return Config{
		DefaultChannel: "console",
		Console: ConsoleConfig{
			Level:    InfoLevel,
			Colorize: true,
		},
		File: FileConfig{
			Enabled:  false,
			Path:     "storage/logs/onyx.log",
			Level:    InfoLevel,
			MaxSize:  10 * 1024 * 1024, // 10MB
			MaxFiles: 5,
		},
		JSON: JSONConfig{
			Enabled: false,
			Path:    "",
			Level:   InfoLevel,
		},
	}
}