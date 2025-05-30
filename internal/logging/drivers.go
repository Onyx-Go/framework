package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// ConsoleDriver outputs to stdout/stderr with colors
type ConsoleDriver struct {
	colorize bool
	writer   io.Writer
}

// NewConsoleDriver creates a new console driver
func NewConsoleDriver(colorize bool) *ConsoleDriver {
	return &ConsoleDriver{
		colorize: colorize,
		writer:   os.Stdout,
	}
}

// SetWriter sets the output writer
func (cd *ConsoleDriver) SetWriter(writer io.Writer) {
	cd.writer = writer
}

// Write writes a log entry to the console
func (cd *ConsoleDriver) Write(ctx context.Context, entry LogEntry) error {
	var output string
	
	if cd.colorize {
		color := GetLevelColor(entry.Level)
		levelName := strings.ToUpper(GetLevelName(entry.Level))
		
		output = fmt.Sprintf("%s[%s]%s [%s] [%s] %s",
			color,
			levelName,
			GetColorReset(),
			entry.Timestamp.Format("2006-01-02 15:04:05"),
			entry.Channel,
			entry.Message,
		)
	} else {
		output = fmt.Sprintf("[%s] [%s] [%s] %s",
			strings.ToUpper(GetLevelName(entry.Level)),
			entry.Timestamp.Format("2006-01-02 15:04:05"),
			entry.Channel,
			entry.Message,
		)
	}
	
	// Add context if present
	if len(entry.Context) > 0 {
		if contextJson, err := json.Marshal(entry.Context); err == nil {
			output += fmt.Sprintf(" Context: %s", string(contextJson))
		}
	}
	
	output += "\n"
	
	// Use stderr for errors and fatal, stdout for others
	writer := cd.writer
	if entry.Level >= ErrorLevel && cd.writer == os.Stdout {
		writer = os.Stderr
	}
	
	_, err := writer.Write([]byte(output))
	return err
}

// Close closes the console driver (no-op)
func (cd *ConsoleDriver) Close() error {
	return nil // Nothing to close for console
}

// FileDriver outputs to rotating log files
type FileDriver struct {
	filepath    string
	file        *os.File
	maxSize     int64 // Max file size in bytes
	maxFiles    int   // Max number of rotated files to keep
	currentSize int64
	mutex       sync.Mutex
}

// NewFileDriver creates a new file driver
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

// Write writes a log entry to the file
func (fd *FileDriver) Write(ctx context.Context, entry LogEntry) error {
	fd.mutex.Lock()
	defer fd.mutex.Unlock()
	
	// Format as JSON for file storage
	logData := map[string]interface{}{
		"level":     GetLevelName(entry.Level),
		"message":   entry.Message,
		"timestamp": entry.Timestamp.Format(time.RFC3339),
		"channel":   entry.Channel,
		"context":   entry.Context,
		"extra":     entry.Extra,
	}
	
	// Add context values if available
	if ctx != nil {
		if requestID := ctx.Value("request_id"); requestID != nil {
			logData["request_id"] = requestID
		}
		if traceID := ctx.Value("trace_id"); traceID != nil {
			logData["trace_id"] = traceID
		}
	}
	
	jsonData, err := json.Marshal(logData)
	if err != nil {
		return err
	}
	
	jsonData = append(jsonData, '\n')
	
	// Check if rotation is needed
	if fd.currentSize+int64(len(jsonData)) > fd.maxSize {
		if err := fd.Rotate(); err != nil {
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

// Rotate rotates the log files
func (fd *FileDriver) Rotate() error {
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

// Close closes the file driver
func (fd *FileDriver) Close() error {
	fd.mutex.Lock()
	defer fd.mutex.Unlock()
	
	if fd.file != nil {
		return fd.file.Close()
	}
	return nil
}

// openFile opens the log file for writing
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

// JSONDriver outputs structured JSON logs
type JSONDriver struct {
	writer io.Writer
}

// NewJSONDriver creates a new JSON driver
func NewJSONDriver(writer io.Writer) *JSONDriver {
	return &JSONDriver{writer: writer}
}

// SetWriter sets the output writer
func (jd *JSONDriver) SetWriter(writer io.Writer) {
	jd.writer = writer
}

// Write writes a log entry as JSON
func (jd *JSONDriver) Write(ctx context.Context, entry LogEntry) error {
	logData := map[string]interface{}{
		"level":     GetLevelName(entry.Level),
		"message":   entry.Message,
		"timestamp": entry.Timestamp.Format(time.RFC3339),
		"channel":   entry.Channel,
		"context":   entry.Context,
		"extra":     entry.Extra,
	}
	
	// Add context values if available
	if ctx != nil {
		contextData := make(map[string]interface{})
		if requestID := ctx.Value("request_id"); requestID != nil {
			contextData["request_id"] = requestID
		}
		if userID := ctx.Value("user_id"); userID != nil {
			contextData["user_id"] = userID
		}
		if traceID := ctx.Value("trace_id"); traceID != nil {
			contextData["trace_id"] = traceID
		}
		if spanID := ctx.Value("span_id"); spanID != nil {
			contextData["span_id"] = spanID
		}
		
		if len(contextData) > 0 {
			logData["request_context"] = contextData
		}
	}
	
	jsonData, err := json.Marshal(logData)
	if err != nil {
		return err
	}
	
	jsonData = append(jsonData, '\n')
	_, err = jd.writer.Write(jsonData)
	return err
}

// Close closes the JSON driver (no-op)
func (jd *JSONDriver) Close() error {
	return nil
}