package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Config manages application configuration with multiple providers and caching
type Config struct {
	values     map[string]interface{}
	providers  []ConfigProvider
	validators map[string]ConfigValidator
	cache      *Cache
	mutex      sync.RWMutex
	loaded     bool
	env        string
	appName    string
	debug      bool
}

// NewConfig creates a new configuration manager
func NewConfig() *Config {
	c := &Config{
		values:     make(map[string]interface{}),
		providers:  make([]ConfigProvider, 0),
		validators: make(map[string]ConfigValidator),
		cache:      NewCache(),
		env:        GetEnv("APP_ENV", "production"),
		appName:    GetEnv("APP_NAME", "Onyx"),
		debug:      GetEnv("APP_DEBUG", "false") == "true",
	}
	
	// Add default providers
	c.AddProvider(&EnvProvider{})
	c.AddProvider(&FileProvider{BasePath: "config"})
	
	// Load .env file if it exists
	if envProvider, err := NewDotEnvProvider(".env"); err == nil {
		c.AddProvider(envProvider)
	}
	
	// Load environment-specific .env file
	envFile := fmt.Sprintf(".env.%s", c.env)
	if envProvider, err := NewDotEnvProvider(envFile); err == nil {
		c.AddProvider(envProvider)
	}
	
	return c
}

// AddProvider adds a configuration provider
func (c *Config) AddProvider(provider ConfigProvider) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.providers = append(c.providers, provider)
}

// AddValidator adds a validator for a configuration key
func (c *Config) AddValidator(key string, validator ConfigValidator) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.validators[key] = validator
}

// Load loads configuration from all providers
func (c *Config) Load() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	// Clear existing values
	c.values = make(map[string]interface{})
	
	// Load from each provider
	for _, provider := range c.providers {
		values, err := provider.Load()
		if err != nil {
			return fmt.Errorf("failed to load from provider %s: %w", provider.Name(), err)
		}
		
		// Merge values (later providers override earlier ones)
		for key, value := range values {
			c.values[key] = value
		}
	}
	
	// Validate all values
	for key, value := range c.values {
		if validator, exists := c.validators[key]; exists {
			if err := validator(key, value); err != nil {
				return fmt.Errorf("validation failed for key %s: %w", key, err)
			}
		}
	}
	
	c.loaded = true
	c.cache.SetLastLoad(time.Now())
	
	return nil
}

// Get retrieves a configuration value with type conversion
func (c *Config) Get(key string, defaultValue ...interface{}) interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	// Check cache first
	if cached, exists := c.cache.Get(key); exists {
		return cached
	}
	
	// Get from values
	value := c.getNestedValue(key)
	if value != nil {
		// Cache the value
		c.cache.Set(key, value)
		return value
	}
	
	// Return default value
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	
	return nil
}

// GetString retrieves a string configuration value
func (c *Config) GetString(key string, defaultValue ...string) string {
	value := c.Get(key)
	if str, ok := value.(string); ok {
		return str
	}
	
	if value != nil {
		return fmt.Sprintf("%v", value)
	}
	
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	
	return ""
}

// GetInt retrieves an integer configuration value
func (c *Config) GetInt(key string, defaultValue ...int) int {
	value := c.Get(key)
	
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		if intValue, err := strconv.Atoi(v); err == nil {
			return intValue
		}
	}
	
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	
	return 0
}

// GetInt64 retrieves an int64 configuration value
func (c *Config) GetInt64(key string, defaultValue ...int64) int64 {
	value := c.Get(key)
	
	switch v := value.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	case string:
		if intValue, err := strconv.ParseInt(v, 10, 64); err == nil {
			return intValue
		}
	}
	
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	
	return 0
}

// GetFloat64 retrieves a float64 configuration value
func (c *Config) GetFloat64(key string, defaultValue ...float64) float64 {
	value := c.Get(key)
	
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		if floatValue, err := strconv.ParseFloat(v, 64); err == nil {
			return floatValue
		}
	}
	
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	
	return 0.0
}

// GetBool retrieves a boolean configuration value
func (c *Config) GetBool(key string, defaultValue ...bool) bool {
	value := c.Get(key)
	
	if b, ok := value.(bool); ok {
		return b
	}
	
	if str, ok := value.(string); ok {
		switch strings.ToLower(str) {
		case "true", "1", "yes", "on", "enable", "enabled":
			return true
		case "false", "0", "no", "off", "disable", "disabled":
			return false
		}
	}
	
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	
	return false
}

// GetDuration retrieves a time.Duration configuration value
func (c *Config) GetDuration(key string, defaultValue ...time.Duration) time.Duration {
	value := c.Get(key)
	
	if d, ok := value.(time.Duration); ok {
		return d
	}
	
	if str, ok := value.(string); ok {
		if duration, err := time.ParseDuration(str); err == nil {
			return duration
		}
	}
	
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	
	return 0
}

// GetStringSlice retrieves a string slice configuration value
func (c *Config) GetStringSlice(key string, defaultValue ...[]string) []string {
	value := c.Get(key)
	
	if slice, ok := value.([]string); ok {
		return slice
	}
	
	if slice, ok := value.([]interface{}); ok {
		result := make([]string, len(slice))
		for i, v := range slice {
			result[i] = fmt.Sprintf("%v", v)
		}
		return result
	}
	
	if str, ok := value.(string); ok {
		// Try to parse as comma-separated values
		return strings.Split(str, ",")
	}
	
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	
	return nil
}

// GetStringMap retrieves a string map configuration value
func (c *Config) GetStringMap(key string, defaultValue ...map[string]string) map[string]string {
	value := c.Get(key)
	
	if m, ok := value.(map[string]string); ok {
		return m
	}
	
	if m, ok := value.(map[string]interface{}); ok {
		result := make(map[string]string)
		for k, v := range m {
			result[k] = fmt.Sprintf("%v", v)
		}
		return result
	}
	
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	
	return nil
}

// Set sets a configuration value
func (c *Config) Set(key string, value interface{}) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	// Validate the value if validator exists
	if validator, exists := c.validators[key]; exists {
		if err := validator(key, value); err != nil {
			return fmt.Errorf("validation failed for key %s: %w", key, err)
		}
	}
	
	c.setNestedValue(key, value)
	
	// Clear cache
	c.cache.Delete(key)
	
	return nil
}

// Has checks if a configuration key exists
func (c *Config) Has(key string) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.getNestedValue(key) != nil
}

// All returns all configuration values
func (c *Config) All() map[string]interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	result := make(map[string]interface{})
	c.copyMap(c.values, result)
	return result
}

// Env returns the current environment
func (c *Config) Env() string {
	return c.env
}

// AppName returns the application name
func (c *Config) AppName() string {
	return c.appName
}

// Debug returns whether debug mode is enabled
func (c *Config) Debug() bool {
	return c.debug
}

// Reload reloads configuration from all providers
func (c *Config) Reload() error {
	return c.Load()
}

// ClearCache clears the configuration cache
func (c *Config) ClearCache() {
	c.cache.Clear()
}

// EnableCache enables configuration caching
func (c *Config) EnableCache(ttl time.Duration) {
	c.cache.Enable(ttl)
}

// DisableCache disables configuration caching
func (c *Config) DisableCache() {
	c.cache.Disable()
}

// Helper methods
func (c *Config) getNestedValue(key string) interface{} {
	keys := strings.Split(key, ".")
	current := c.values
	
	for i, k := range keys {
		if i == len(keys)-1 {
			return current[k]
		}
		
		if next, ok := current[k].(map[string]interface{}); ok {
			current = next
		} else {
			return nil
		}
	}
	
	return nil
}

func (c *Config) setNestedValue(key string, value interface{}) {
	keys := strings.Split(key, ".")
	current := c.values
	
	for i, k := range keys {
		if i == len(keys)-1 {
			current[k] = value
			return
		}
		
		if next, ok := current[k].(map[string]interface{}); ok {
			current = next
		} else {
			next = make(map[string]interface{})
			current[k] = next
			current = next
		}
	}
}

func (c *Config) copyMap(src, dst map[string]interface{}) {
	for k, v := range src {
		if m, ok := v.(map[string]interface{}); ok {
			nested := make(map[string]interface{})
			c.copyMap(m, nested)
			dst[k] = nested
		} else {
			dst[k] = v
		}
	}
}

// GetEnv is a utility function to get environment variables
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}