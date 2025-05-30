package logging

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// channel represents a logging channel with specific configuration
type channel struct {
	name    string
	driver  Driver
	level   LogLevel
	context map[string]interface{}
	mutex   sync.RWMutex
}

// Context-aware logging methods

func (c *channel) DebugContext(ctx context.Context, message string, args ...map[string]interface{}) {
	c.LogContext(ctx, DebugLevel, message, args...)
}

func (c *channel) InfoContext(ctx context.Context, message string, args ...map[string]interface{}) {
	c.LogContext(ctx, InfoLevel, message, args...)
}

func (c *channel) WarnContext(ctx context.Context, message string, args ...map[string]interface{}) {
	c.LogContext(ctx, WarnLevel, message, args...)
}

func (c *channel) ErrorContext(ctx context.Context, message string, args ...map[string]interface{}) {
	c.LogContext(ctx, ErrorLevel, message, args...)
}

func (c *channel) FatalContext(ctx context.Context, message string, args ...map[string]interface{}) {
	c.LogContext(ctx, FatalLevel, message, args...)
}

func (c *channel) LogContext(ctx context.Context, level LogLevel, message string, args ...map[string]interface{}) {
	if level < c.level {
		return // Skip if below minimum level
	}
	
	entry := LogEntry{
		Level:     level,
		Message:   message,
		Timestamp: time.Now(),
		Channel:   c.name,
		Context:   c.mergeContextWithRequestContext(ctx, args...),
		Extra:     c.getExtraInfo(),
	}
	
	c.driver.Write(ctx, entry)
}

// Legacy logging methods (for backward compatibility)

func (c *channel) Debug(message string, context ...map[string]interface{}) {
	c.Log(DebugLevel, message, context...)
}

func (c *channel) Info(message string, context ...map[string]interface{}) {
	c.Log(InfoLevel, message, context...)
}

func (c *channel) Warn(message string, context ...map[string]interface{}) {
	c.Log(WarnLevel, message, context...)
}

func (c *channel) Error(message string, context ...map[string]interface{}) {
	c.Log(ErrorLevel, message, context...)
}

func (c *channel) Fatal(message string, context ...map[string]interface{}) {
	c.Log(FatalLevel, message, context...)
}

func (c *channel) Log(level LogLevel, message string, contextMaps ...map[string]interface{}) {
	c.LogContext(context.Background(), level, message, contextMaps...)
}

// Logger modifiers

func (c *channel) WithContext(context map[string]interface{}) Logger {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	newChannel := &channel{
		name:    c.name,
		driver:  c.driver,
		level:   c.level,
		context: c.mergeContext(context),
	}
	
	return newChannel
}

func (c *channel) WithChannel(channelName string) Logger {
	newChannel := &channel{
		name:    channelName,
		driver:  c.driver,
		level:   c.level,
		context: c.context,
	}
	
	return newChannel
}

// Helper methods

func (c *channel) mergeContext(contexts ...map[string]interface{}) map[string]interface{} {
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

func (c *channel) mergeContextWithRequestContext(ctx context.Context, contexts ...map[string]interface{}) map[string]interface{} {
	merged := c.mergeContext(contexts...)
	
	// Extract request context if available
	if ctx != nil {
		if requestID := ctx.Value("request_id"); requestID != nil {
			merged["request_id"] = requestID
		}
		if userID := ctx.Value("user_id"); userID != nil {
			merged["user_id"] = userID
		}
		if traceID := ctx.Value("trace_id"); traceID != nil {
			merged["trace_id"] = traceID
		}
		if spanID := ctx.Value("span_id"); spanID != nil {
			merged["span_id"] = spanID
		}
	}
	
	return merged
}

func (c *channel) getExtraInfo() map[string]interface{} {
	extra := make(map[string]interface{})
	
	// Add caller information
	if pc, file, line, ok := runtime.Caller(4); ok {
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