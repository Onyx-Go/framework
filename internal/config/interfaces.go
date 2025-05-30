package config

import "time"

// ConfigProvider interface for different configuration sources
type ConfigProvider interface {
	Load() (map[string]interface{}, error)
	Watch() (<-chan ConfigEvent, error)
	Name() string
}

// ConfigEvent represents a configuration change event
type ConfigEvent struct {
	Type   string // "added", "modified", "deleted"
	Key    string
	Value  interface{}
	Source string
}

// ConfigValidator function type for validating configuration values
type ConfigValidator func(key string, value interface{}) error

// Repository interface for configuration storage and retrieval
type Repository interface {
	Get(key string, defaultValue ...interface{}) interface{}
	GetString(key string, defaultValue ...string) string
	GetInt(key string, defaultValue ...int) int
	GetInt64(key string, defaultValue ...int64) int64
	GetFloat64(key string, defaultValue ...float64) float64
	GetBool(key string, defaultValue ...bool) bool
	GetDuration(key string, defaultValue ...time.Duration) time.Duration
	GetStringSlice(key string, defaultValue ...[]string) []string
	GetStringMap(key string, defaultValue ...map[string]string) map[string]string
	Set(key string, value interface{}) error
	Has(key string) bool
	All() map[string]interface{}
}

// Manager interface for configuration management
type Manager interface {
	Repository
	
	// Management methods
	AddProvider(provider ConfigProvider)
	AddValidator(key string, validator ConfigValidator)
	Load() error
	Reload() error
	
	// Environment info
	Env() string
	AppName() string
	Debug() bool
	
	// Cache control
	ClearCache()
	EnableCache(ttl time.Duration)
	DisableCache()
}