package logging

import (
	"context"
	"fmt"
	"io"
	"os"
)

// Service provides centralized logging functionality
type Service struct {
	manager Manager
	config  Config
}

// NewService creates a new logging service
func NewService(config Config) (*Service, error) {
	service := &Service{
		manager: NewManager(),
		config:  config,
	}
	
	if err := service.setupChannels(); err != nil {
		return nil, fmt.Errorf("failed to setup logging channels: %w", err)
	}
	
	return service, nil
}

// setupChannels sets up the logging channels based on configuration
func (s *Service) setupChannels() error {
	// Setup console channel
	consoleDriver := NewConsoleDriver(s.config.Console.Colorize)
	s.manager.AddChannel("console", consoleDriver, s.config.Console.Level)
	
	// Setup file channel if enabled
	if s.config.File.Enabled {
		fileDriver, err := NewFileDriver(s.config.File.Path, s.config.File.MaxSize, s.config.File.MaxFiles)
		if err != nil {
			return fmt.Errorf("failed to setup file logging: %w", err)
		}
		s.manager.AddChannel("file", fileDriver, s.config.File.Level)
	}
	
	// Setup JSON channel if enabled
	if s.config.JSON.Enabled {
		var writer io.Writer = os.Stdout
		if s.config.JSON.Path != "" {
			file, err := os.OpenFile(s.config.JSON.Path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				return fmt.Errorf("failed to setup JSON logging: %w", err)
			}
			writer = file
		}
		jsonDriver := NewJSONDriver(writer)
		s.manager.AddChannel("json", jsonDriver, s.config.JSON.Level)
	}
	
	// Set default channel
	s.manager.SetDefaultChannel(s.config.DefaultChannel)
	return nil
}

// Manager returns the underlying log manager
func (s *Service) Manager() Manager {
	return s.manager
}

// Logger returns the default logger
func (s *Service) Logger() Logger {
	return s.manager.Default()
}

// Channel returns a specific logging channel
func (s *Service) Channel(name string) Logger {
	return s.manager.Channel(name)
}

// Context-aware logging methods using default logger

func (s *Service) DebugContext(ctx context.Context, message string, args ...map[string]interface{}) {
	s.Logger().DebugContext(ctx, message, args...)
}

func (s *Service) InfoContext(ctx context.Context, message string, args ...map[string]interface{}) {
	s.Logger().InfoContext(ctx, message, args...)
}

func (s *Service) WarnContext(ctx context.Context, message string, args ...map[string]interface{}) {
	s.Logger().WarnContext(ctx, message, args...)
}

func (s *Service) ErrorContext(ctx context.Context, message string, args ...map[string]interface{}) {
	s.Logger().ErrorContext(ctx, message, args...)
}

func (s *Service) FatalContext(ctx context.Context, message string, args ...map[string]interface{}) {
	s.Logger().FatalContext(ctx, message, args...)
}

// Legacy logging methods (for backward compatibility)

func (s *Service) Debug(message string, context ...map[string]interface{}) {
	s.Logger().Debug(message, context...)
}

func (s *Service) Info(message string, context ...map[string]interface{}) {
	s.Logger().Info(message, context...)
}

func (s *Service) Warn(message string, context ...map[string]interface{}) {
	s.Logger().Warn(message, context...)
}

func (s *Service) Error(message string, context ...map[string]interface{}) {
	s.Logger().Error(message, context...)
}

func (s *Service) Fatal(message string, context ...map[string]interface{}) {
	s.Logger().Fatal(message, context...)
}

// Close closes the logging service
func (s *Service) Close() error {
	return s.manager.Close()
}

// Global logging service instance
var globalService *Service

// Setup sets up the global logging service
func Setup(config Config) error {
	service, err := NewService(config)
	if err != nil {
		return err
	}
	
	globalService = service
	return nil
}

// GetService returns the global logging service
func GetService() *Service {
	if globalService == nil {
		// Create a default service if none exists
		config := DefaultConfig()
		service, _ := NewService(config)
		globalService = service
	}
	return globalService
}

// Global context-aware logging functions

func DebugContext(ctx context.Context, message string, args ...map[string]interface{}) {
	GetService().DebugContext(ctx, message, args...)
}

func InfoContext(ctx context.Context, message string, args ...map[string]interface{}) {
	GetService().InfoContext(ctx, message, args...)
}

func WarnContext(ctx context.Context, message string, args ...map[string]interface{}) {
	GetService().WarnContext(ctx, message, args...)
}

func ErrorContext(ctx context.Context, message string, args ...map[string]interface{}) {
	GetService().ErrorContext(ctx, message, args...)
}

func FatalContext(ctx context.Context, message string, args ...map[string]interface{}) {
	GetService().FatalContext(ctx, message, args...)
}

// Global legacy logging functions (for backward compatibility)

func Debug(message string, context ...map[string]interface{}) {
	GetService().Debug(message, context...)
}

func Info(message string, context ...map[string]interface{}) {
	GetService().Info(message, context...)
}

func Warn(message string, context ...map[string]interface{}) {
	GetService().Warn(message, context...)
}

func Error(message string, context ...map[string]interface{}) {
	GetService().Error(message, context...)
}

func Fatal(message string, context ...map[string]interface{}) {
	GetService().Fatal(message, context...)
}

// GetLogger returns the global logger
func GetLogger() Logger {
	return GetService().Logger()
}

// Channel returns a specific logging channel
func Channel(name string) Logger {
	return GetService().Channel(name)
}