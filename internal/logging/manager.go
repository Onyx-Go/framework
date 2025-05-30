package logging

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// manager implements the Manager interface
type manager struct {
	channels       map[string]*channel
	defaultChannel string
	mutex          sync.RWMutex
}

// NewManager creates a new log manager
func NewManager() Manager {
	return &manager{
		channels:       make(map[string]*channel),
		defaultChannel: "default",
	}
}

// AddChannel adds a new logging channel
func (m *manager) AddChannel(name string, driver Driver, level LogLevel) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.channels[name] = &channel{
		name:    name,
		driver:  driver,
		level:   level,
		context: make(map[string]interface{}),
	}
}

// Channel gets a specific logging channel
func (m *manager) Channel(name string) Logger {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	if ch, exists := m.channels[name]; exists {
		return ch
	}
	
	// Return default channel if specified channel doesn't exist
	if defaultChannel, exists := m.channels[m.defaultChannel]; exists {
		return defaultChannel
	}
	
	// Return a null logger if no channels exist
	return &nullLogger{}
}

// SetDefaultChannel sets the default logging channel
func (m *manager) SetDefaultChannel(name string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.defaultChannel = name
}

// Default returns the default logging channel
func (m *manager) Default() Logger {
	return m.Channel(m.defaultChannel)
}

// Close closes all logging channels
func (m *manager) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	var errors []string
	for name, ch := range m.channels {
		if err := ch.driver.Close(); err != nil {
			errors = append(errors, fmt.Sprintf("channel %s: %v", name, err))
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("errors closing channels: %s", strings.Join(errors, ", "))
	}
	
	return nil
}

// nullLogger discards all log entries (for testing/disabled logging)
type nullLogger struct{}

func (nl *nullLogger) DebugContext(ctx context.Context, message string, args ...map[string]interface{}) {}
func (nl *nullLogger) InfoContext(ctx context.Context, message string, args ...map[string]interface{})  {}
func (nl *nullLogger) WarnContext(ctx context.Context, message string, args ...map[string]interface{})  {}
func (nl *nullLogger) ErrorContext(ctx context.Context, message string, args ...map[string]interface{}) {}
func (nl *nullLogger) FatalContext(ctx context.Context, message string, args ...map[string]interface{}) {}
func (nl *nullLogger) LogContext(ctx context.Context, level LogLevel, message string, args ...map[string]interface{}) {}

func (nl *nullLogger) Debug(message string, context ...map[string]interface{}) {}
func (nl *nullLogger) Info(message string, context ...map[string]interface{})  {}
func (nl *nullLogger) Warn(message string, context ...map[string]interface{})  {}
func (nl *nullLogger) Error(message string, context ...map[string]interface{}) {}
func (nl *nullLogger) Fatal(message string, context ...map[string]interface{}) {}
func (nl *nullLogger) Log(level LogLevel, message string, context ...map[string]interface{}) {}
func (nl *nullLogger) WithContext(context map[string]interface{}) Logger       { return nl }
func (nl *nullLogger) WithChannel(channel string) Logger                       { return nl }