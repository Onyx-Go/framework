package onyx

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
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

var logLevelNames = map[LogLevel]string{
	DebugLevel: "debug",
	InfoLevel:  "info",
	WarnLevel:  "warning",
	ErrorLevel: "error",
	FatalLevel: "fatal",
}

var logLevelColors = map[LogLevel]string{
	DebugLevel: "\033[36m", // Cyan
	InfoLevel:  "\033[32m", // Green
	WarnLevel:  "\033[33m", // Yellow
	ErrorLevel: "\033[31m", // Red
	FatalLevel: "\033[35m", // Magenta
}

const colorReset = "\033[0m"

// LogEntry represents a single log entry
type LogEntry struct {
	Level     LogLevel               `json:"level"`
	Message   string                 `json:"message"`
	Timestamp time.Time              `json:"timestamp"`
	Context   map[string]interface{} `json:"context,omitempty"`
	Channel   string                 `json:"channel,omitempty"`
	Extra     map[string]interface{} `json:"extra,omitempty"`
}

// Logger interface defines the logging contract
type Logger interface {
	Debug(message string, context ...map[string]interface{})
	Info(message string, context ...map[string]interface{})
	Warn(message string, context ...map[string]interface{})
	Error(message string, context ...map[string]interface{})
	Fatal(message string, context ...map[string]interface{})
	Log(level LogLevel, message string, context ...map[string]interface{})
	WithContext(context map[string]interface{}) Logger
	WithChannel(channel string) Logger
}

// Driver interface for different logging backends
type Driver interface {
	Write(entry LogEntry) error
	Close() error
}

// LogManager manages multiple logging channels and drivers
type LogManager struct {
	channels       map[string]*Channel
	defaultChannel string
	mutex          sync.RWMutex
}

// Channel represents a logging channel with specific configuration
type Channel struct {
	name    string
	driver  Driver
	level   LogLevel
	context map[string]interface{}
	mutex   sync.RWMutex
}

// NewLogManager creates a new log manager
func NewLogManager() *LogManager {
	return &LogManager{
		channels:       make(map[string]*Channel),
		defaultChannel: "default",
	}
}

// AddChannel adds a new logging channel
func (lm *LogManager) AddChannel(name string, driver Driver, level LogLevel) {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()
	
	lm.channels[name] = &Channel{
		name:    name,
		driver:  driver,
		level:   level,
		context: make(map[string]interface{}),
	}
}

// Channel gets a specific logging channel
func (lm *LogManager) Channel(name string) Logger {
	lm.mutex.RLock()
	defer lm.mutex.RUnlock()
	
	if channel, exists := lm.channels[name]; exists {
		return channel
	}
	
	// Return default channel if specified channel doesn't exist
	if defaultChannel, exists := lm.channels[lm.defaultChannel]; exists {
		return defaultChannel
	}
	
	// Return a null logger if no channels exist
	return &NullLogger{}
}

// SetDefaultChannel sets the default logging channel
func (lm *LogManager) SetDefaultChannel(name string) {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()
	lm.defaultChannel = name
}

// Default returns the default logging channel
func (lm *LogManager) Default() Logger {
	return lm.Channel(lm.defaultChannel)
}

// Close closes all logging channels
func (lm *LogManager) Close() error {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()
	
	var errors []string
	for name, channel := range lm.channels {
		if err := channel.driver.Close(); err != nil {
			errors = append(errors, fmt.Sprintf("channel %s: %v", name, err))
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("errors closing channels: %s", strings.Join(errors, ", "))
	}
	
	return nil
}

// Channel implementation
func (c *Channel) Debug(message string, context ...map[string]interface{}) {
	c.Log(DebugLevel, message, context...)
}

func (c *Channel) Info(message string, context ...map[string]interface{}) {
	c.Log(InfoLevel, message, context...)
}

func (c *Channel) Warn(message string, context ...map[string]interface{}) {
	c.Log(WarnLevel, message, context...)
}

func (c *Channel) Error(message string, context ...map[string]interface{}) {
	c.Log(ErrorLevel, message, context...)
}

func (c *Channel) Fatal(message string, context ...map[string]interface{}) {
	c.Log(FatalLevel, message, context...)
}

func (c *Channel) Log(level LogLevel, message string, context ...map[string]interface{}) {
	if level < c.level {
		return // Skip if below minimum level
	}
	
	entry := LogEntry{
		Level:     level,
		Message:   message,
		Timestamp: time.Now(),
		Channel:   c.name,
		Context:   c.mergeContext(context...),
		Extra:     c.getExtraInfo(),
	}
	
	c.driver.Write(entry)
}

func (c *Channel) WithContext(context map[string]interface{}) Logger {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	newChannel := &Channel{
		name:    c.name,
		driver:  c.driver,
		level:   c.level,
		context: c.mergeContext(context),
	}
	
	return newChannel
}

func (c *Channel) WithChannel(channel string) Logger {
	// For the channel implementation, this returns itself with updated name
	newChannel := &Channel{
		name:    channel,
		driver:  c.driver,
		level:   c.level,
		context: c.context,
	}
	
	return newChannel
}

func (c *Channel) mergeContext(contexts ...map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})
	
	// Start with channel context
	for k, v := range c.context {
		merged[k] = v
	}
	
	// Add provided contexts
	for _, ctx := range contexts {
		for k, v := range ctx {
			merged[k] = v
		}
	}
	
	return merged
}

func (c *Channel) getExtraInfo() map[string]interface{} {
	extra := make(map[string]interface{})
	
	// Add caller information
	if pc, file, line, ok := runtime.Caller(3); ok {
		if fn := runtime.FuncForPC(pc); fn != nil {
			extra["function"] = fn.Name()
		}
		extra["file"] = filepath.Base(file)
		extra["line"] = line
	}
	
	extra["go_version"] = runtime.Version()
	extra["process_id"] = os.Getpid()
	
	return extra
}

// Console Driver - outputs to stdout/stderr with colors
type ConsoleDriver struct {
	colorize bool
	writer   io.Writer
}

func NewConsoleDriver(colorize bool) *ConsoleDriver {
	return &ConsoleDriver{
		colorize: colorize,
		writer:   os.Stdout,
	}
}

func (cd *ConsoleDriver) Write(entry LogEntry) error {
	var output string
	
	if cd.colorize {
		color := logLevelColors[entry.Level]
		levelName := strings.ToUpper(logLevelNames[entry.Level])
		
		output = fmt.Sprintf("%s[%s]%s [%s] [%s] %s",
			color,
			levelName,
			colorReset,
			entry.Timestamp.Format("2006-01-02 15:04:05"),
			entry.Channel,
			entry.Message,
		)
	} else {
		output = fmt.Sprintf("[%s] [%s] [%s] %s",
			strings.ToUpper(logLevelNames[entry.Level]),
			entry.Timestamp.Format("2006-01-02 15:04:05"),
			entry.Channel,
			entry.Message,
		)
	}
	
	if len(entry.Context) > 0 {
		if contextJson, err := json.Marshal(entry.Context); err == nil {
			output += fmt.Sprintf(" Context: %s", string(contextJson))
		}
	}
	
	output += "\n"
	
	// Use stderr for errors and fatal, stdout for others
	writer := cd.writer
	if entry.Level >= ErrorLevel {
		writer = os.Stderr
	}
	
	_, err := writer.Write([]byte(output))
	return err
}

func (cd *ConsoleDriver) Close() error {
	return nil // Nothing to close for console
}

// File Driver - outputs to rotating log files
type FileDriver struct {
	filepath   string
	file       *os.File
	maxSize    int64 // Max file size in bytes
	maxFiles   int   // Max number of rotated files to keep
	currentSize int64
	mutex      sync.Mutex
}

func NewFileDriver(filepath string, maxSize int64, maxFiles int) (*FileDriver, error) {
	// Ensure directory exists
	dir := filepath[:strings.LastIndex(filepath, "/")]
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %v", err)
	}
	
	fd := &FileDriver{
		filepath: filepath,
		maxSize:  maxSize,
		maxFiles: maxFiles,
	}
	
	if err := fd.openFile(); err != nil {
		return nil, err
	}
	
	return fd, nil
}

func (fd *FileDriver) openFile() error {
	file, err := os.OpenFile(fd.filepath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	
	// Get current file size
	if stat, err := file.Stat(); err == nil {
		fd.currentSize = stat.Size()
	}
	
	fd.file = file
	return nil
}

func (fd *FileDriver) Write(entry LogEntry) error {
	fd.mutex.Lock()
	defer fd.mutex.Unlock()
	
	// Format as JSON for file storage
	logData := map[string]interface{}{
		"level":     logLevelNames[entry.Level],
		"message":   entry.Message,
		"timestamp": entry.Timestamp.Format(time.RFC3339),
		"channel":   entry.Channel,
		"context":   entry.Context,
		"extra":     entry.Extra,
	}
	
	jsonData, err := json.Marshal(logData)
	if err != nil {
		return err
	}
	
	jsonData = append(jsonData, '\n')
	
	// Check if rotation is needed
	if fd.currentSize+int64(len(jsonData)) > fd.maxSize {
		if err := fd.rotate(); err != nil {
			return err
		}
	}
	
	n, err := fd.file.Write(jsonData)
	if err != nil {
		return err
	}
	
	fd.currentSize += int64(n)
	return nil
}

func (fd *FileDriver) rotate() error {
	// Close current file
	fd.file.Close()
	
	// Rotate existing files
	for i := fd.maxFiles - 1; i > 0; i-- {
		oldPath := fmt.Sprintf("%s.%d", fd.filepath, i)
		newPath := fmt.Sprintf("%s.%d", fd.filepath, i+1)
		
		if i == fd.maxFiles-1 {
			// Remove the oldest file
			os.Remove(newPath)
		}
		
		if _, err := os.Stat(oldPath); err == nil {
			os.Rename(oldPath, newPath)
		}
	}
	
	// Move current file to .1
	if _, err := os.Stat(fd.filepath); err == nil {
		os.Rename(fd.filepath, fd.filepath+".1")
	}
	
	// Create new file
	fd.currentSize = 0
	return fd.openFile()
}

func (fd *FileDriver) Close() error {
	fd.mutex.Lock()
	defer fd.mutex.Unlock()
	
	if fd.file != nil {
		return fd.file.Close()
	}
	return nil
}

// JSON Driver - outputs structured JSON logs
type JSONDriver struct {
	writer io.Writer
}

func NewJSONDriver(writer io.Writer) *JSONDriver {
	return &JSONDriver{writer: writer}
}

func (jd *JSONDriver) Write(entry LogEntry) error {
	logData := map[string]interface{}{
		"level":     logLevelNames[entry.Level],
		"message":   entry.Message,
		"timestamp": entry.Timestamp.Format(time.RFC3339),
		"channel":   entry.Channel,
		"context":   entry.Context,
		"extra":     entry.Extra,
	}
	
	jsonData, err := json.Marshal(logData)
	if err != nil {
		return err
	}
	
	jsonData = append(jsonData, '\n')
	_, err = jd.writer.Write(jsonData)
	return err
}

func (jd *JSONDriver) Close() error {
	return nil
}

// NullLogger - discards all log entries (for testing/disabled logging)
type NullLogger struct{}

func (nl *NullLogger) Debug(message string, context ...map[string]interface{}) {}
func (nl *NullLogger) Info(message string, context ...map[string]interface{})  {}
func (nl *NullLogger) Warn(message string, context ...map[string]interface{})  {}
func (nl *NullLogger) Error(message string, context ...map[string]interface{}) {}
func (nl *NullLogger) Fatal(message string, context ...map[string]interface{}) {}
func (nl *NullLogger) Log(level LogLevel, message string, context ...map[string]interface{}) {}
func (nl *NullLogger) WithContext(context map[string]interface{}) Logger       { return nl }
func (nl *NullLogger) WithChannel(channel string) Logger                       { return nl }

// Helper functions for global logging
var globalLogManager *LogManager

func SetupLogging(config LoggingConfig) error {
	globalLogManager = NewLogManager()
	
	// Setup default console channel
	consoleDriver := NewConsoleDriver(config.Console.Colorize)
	globalLogManager.AddChannel("console", consoleDriver, config.Console.Level)
	
	// Setup file channel if configured
	if config.File.Enabled {
		fileDriver, err := NewFileDriver(config.File.Path, config.File.MaxSize, config.File.MaxFiles)
		if err != nil {
			return fmt.Errorf("failed to setup file logging: %v", err)
		}
		globalLogManager.AddChannel("file", fileDriver, config.File.Level)
	}
	
	// Setup JSON channel if configured  
	if config.JSON.Enabled {
		var writer io.Writer = os.Stdout
		if config.JSON.Path != "" {
			file, err := os.OpenFile(config.JSON.Path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				return fmt.Errorf("failed to setup JSON logging: %v", err)
			}
			writer = file
		}
		jsonDriver := NewJSONDriver(writer)
		globalLogManager.AddChannel("json", jsonDriver, config.JSON.Level)
	}
	
	globalLogManager.SetDefaultChannel(config.DefaultChannel)
	return nil
}

type LoggingConfig struct {
	DefaultChannel string `json:"default_channel"`
	Console        struct {
		Level    LogLevel `json:"level"`
		Colorize bool     `json:"colorize"`
	} `json:"console"`
	File struct {
		Enabled  bool     `json:"enabled"`
		Path     string   `json:"path"`
		Level    LogLevel `json:"level"`
		MaxSize  int64    `json:"max_size"`
		MaxFiles int      `json:"max_files"`
	} `json:"file"`
	JSON struct {
		Enabled bool     `json:"enabled"`
		Path    string   `json:"path"`
		Level   LogLevel `json:"level"`
	} `json:"json"`
}

// Global logging functions
func Log() Logger {
	if globalLogManager != nil {
		return globalLogManager.Default()
	}
	return &NullLogger{}
}

func Debug(message string, context ...map[string]interface{}) {
	Log().Debug(message, context...)
}

func Info(message string, context ...map[string]interface{}) {
	Log().Info(message, context...)
}

func Warn(message string, context ...map[string]interface{}) {
	Log().Warn(message, context...)
}

func Error(message string, context ...map[string]interface{}) {
	Log().Error(message, context...)
}

func Fatal(message string, context ...map[string]interface{}) {
	Log().Fatal(message, context...)
}

// Context helper for adding request context to logs
func (c *Context) Log() Logger {
	if globalLogManager != nil {
		requestContext := map[string]interface{}{
			"method":     c.Method(),
			"url":        c.URL(),
			"user_agent": c.UserAgent(),
			"remote_ip":  c.RemoteIP(),
		}
		
		return globalLogManager.Default().WithContext(requestContext)
	}
	return &NullLogger{}
}