package onyx

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

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

// Config manages application configuration with multiple providers and caching
type Config struct {
	values     map[string]interface{}
	providers  []ConfigProvider
	validators map[string]ConfigValidator
	cache      *ConfigCache
	mutex      sync.RWMutex
	loaded     bool
	env        string
	appName    string
	debug      bool
}

// ConfigCache handles configuration caching
type ConfigCache struct {
	enabled   bool
	ttl       time.Duration
	cached    map[string]cachedValue
	mutex     sync.RWMutex
	lastLoad  time.Time
}

type cachedValue struct {
	value     interface{}
	expiresAt time.Time
}

// NewConfig creates a new configuration manager
func NewConfig() *Config {
	c := &Config{
		values:     make(map[string]interface{}),
		providers:  make([]ConfigProvider, 0),
		validators: make(map[string]ConfigValidator),
		cache: &ConfigCache{
			enabled: true,
			ttl:     5 * time.Minute,
			cached:  make(map[string]cachedValue),
		},
		env:     GetEnv("APP_ENV", "production"),
		appName: GetEnv("APP_NAME", "Onyx"),
		debug:   GetEnv("APP_DEBUG", "false") == "true",
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
	c.cache.lastLoad = time.Now()
	
	return nil
}

// Get retrieves a configuration value with type conversion
func (c *Config) Get(key string, defaultValue ...interface{}) interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	// Check cache first
	if c.cache.enabled {
		if cached, exists := c.getCached(key); exists {
			return cached
		}
	}
	
	// Get from values
	value := c.getNestedValue(key)
	if value != nil {
		// Cache the value
		if c.cache.enabled {
			c.setCached(key, value)
		}
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
	if c.cache.enabled {
		c.cache.mutex.Lock()
		delete(c.cache.cached, key)
		c.cache.mutex.Unlock()
	}
	
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
	if c.cache.enabled {
		c.cache.mutex.Lock()
		c.cache.cached = make(map[string]cachedValue)
		c.cache.mutex.Unlock()
	}
}

// EnableCache enables configuration caching
func (c *Config) EnableCache(ttl time.Duration) {
	c.cache.enabled = true
	c.cache.ttl = ttl
}

// DisableCache disables configuration caching
func (c *Config) DisableCache() {
	c.cache.enabled = false
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

func (c *Config) getCached(key string) (interface{}, bool) {
	c.cache.mutex.RLock()
	defer c.cache.mutex.RUnlock()
	
	if cached, exists := c.cache.cached[key]; exists {
		if time.Now().Before(cached.expiresAt) {
			return cached.value, true
		}
		// Expired, remove it
		delete(c.cache.cached, key)
	}
	
	return nil, false
}

func (c *Config) setCached(key string, value interface{}) {
	c.cache.mutex.Lock()
	defer c.cache.mutex.Unlock()
	
	c.cache.cached[key] = cachedValue{
		value:     value,
		expiresAt: time.Now().Add(c.cache.ttl),
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

// EnvProvider loads configuration from environment variables
type EnvProvider struct{}

func (ep *EnvProvider) Name() string {
	return "env"
}

func (ep *EnvProvider) Load() (map[string]interface{}, error) {
	result := make(map[string]interface{})
	
	for _, env := range os.Environ() {
		if parts := strings.SplitN(env, "=", 2); len(parts) == 2 {
			key := strings.ToLower(parts[0])
			value := parts[1]
			
			// Try to parse as different types
			if parsed := parseValue(value); parsed != nil {
				result[key] = parsed
			} else {
				result[key] = value
			}
		}
	}
	
	return result, nil
}

func (ep *EnvProvider) Watch() (<-chan ConfigEvent, error) {
	// Environment variables can't be watched easily
	return nil, errors.New("environment provider doesn't support watching")
}

// DotEnvProvider loads configuration from .env files
type DotEnvProvider struct {
	filepath string
}

func NewDotEnvProvider(filepath string) (*DotEnvProvider, error) {
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return nil, fmt.Errorf("env file %s does not exist", filepath)
	}
	
	return &DotEnvProvider{filepath: filepath}, nil
}

func (dep *DotEnvProvider) Name() string {
	return fmt.Sprintf("dotenv:%s", dep.filepath)
}

func (dep *DotEnvProvider) Load() (map[string]interface{}, error) {
	file, err := os.Open(dep.filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	result := make(map[string]interface{})
	scanner := bufio.NewScanner(file)
	lineNum := 0
	
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// Parse key=value pairs
		if parts := strings.SplitN(line, "=", 2); len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			
			// Remove quotes if present
			value = removeQuotes(value)
			
			// Expand variables
			value = expandVariables(value, result)
			
			// Parse value
			if parsed := parseValue(value); parsed != nil {
				result[strings.ToLower(key)] = parsed
			} else {
				result[strings.ToLower(key)] = value
			}
		} else {
			return nil, fmt.Errorf("invalid syntax in %s at line %d: %s", dep.filepath, lineNum, line)
		}
	}
	
	return result, scanner.Err()
}

func (dep *DotEnvProvider) Watch() (<-chan ConfigEvent, error) {
	// File watching would require a file watcher implementation
	return nil, errors.New("dotenv provider doesn't support watching yet")
}

// FileProvider loads configuration from JSON/YAML files in a directory
type FileProvider struct {
	BasePath string
}

func (fp *FileProvider) Name() string {
	return fmt.Sprintf("file:%s", fp.BasePath)
}

func (fp *FileProvider) Load() (map[string]interface{}, error) {
	result := make(map[string]interface{})
	
	if _, err := os.Stat(fp.BasePath); os.IsNotExist(err) {
		return result, nil // No config directory is fine
	}
	
	err := filepath.WalkDir(fp.BasePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		if d.IsDir() {
			return nil
		}
		
		// Only process JSON files for now (YAML support can be added later)
		if !strings.HasSuffix(path, ".json") {
			return nil
		}
		
		// Read and parse file
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		
		var config map[string]interface{}
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("failed to parse %s: %w", path, err)
		}
		
		// Use filename (without extension) as key
		filename := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		result[filename] = config
		
		return nil
	})
	
	return result, err
}

func (fp *FileProvider) Watch() (<-chan ConfigEvent, error) {
	return nil, errors.New("file provider doesn't support watching yet")
}

// Utility functions
func parseValue(value string) interface{} {
	// Try bool
	if lower := strings.ToLower(value); lower == "true" || lower == "false" {
		return lower == "true"
	}
	
	// Try int
	if intVal, err := strconv.Atoi(value); err == nil {
		return intVal
	}
	
	// Try float
	if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
		return floatVal
	}
	
	// Try duration
	if duration, err := time.ParseDuration(value); err == nil {
		return duration
	}
	
	// Return as string
	return value
}

func removeQuotes(value string) string {
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') ||
		   (value[0] == '\'' && value[len(value)-1] == '\'') {
			return value[1 : len(value)-1]
		}
	}
	return value
}

func expandVariables(value string, env map[string]interface{}) string {
	re := regexp.MustCompile(`\$\{([^}]+)\}`)
	return re.ReplaceAllStringFunc(value, func(match string) string {
		varName := match[2 : len(match)-1] // Remove ${ and }
		if envValue, exists := env[strings.ToLower(varName)]; exists {
			return fmt.Sprintf("%v", envValue)
		}
		// Check system environment
		if sysValue := os.Getenv(varName); sysValue != "" {
			return sysValue
		}
		return match // Return unchanged if not found
	})
}

// GetEnv is a utility function to get environment variables
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Common validators
func RequiredValidator(key string, value interface{}) error {
	if value == nil {
		return fmt.Errorf("configuration key %s is required", key)
	}
	
	if str, ok := value.(string); ok && str == "" {
		return fmt.Errorf("configuration key %s cannot be empty", key)
	}
	
	return nil
}

func IntRangeValidator(min, max int) ConfigValidator {
	return func(key string, value interface{}) error {
		var intVal int
		
		switch v := value.(type) {
		case int:
			intVal = v
		case string:
			var err error
			if intVal, err = strconv.Atoi(v); err != nil {
				return fmt.Errorf("configuration key %s must be an integer", key)
			}
		default:
			return fmt.Errorf("configuration key %s must be an integer", key)
		}
		
		if intVal < min || intVal > max {
			return fmt.Errorf("configuration key %s must be between %d and %d", key, min, max)
		}
		
		return nil
	}
}

func RegexValidator(pattern string) ConfigValidator {
	re := regexp.MustCompile(pattern)
	return func(key string, value interface{}) error {
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("configuration key %s must be a string", key)
		}
		
		if !re.MatchString(str) {
			return fmt.Errorf("configuration key %s does not match required pattern", key)
		}
		
		return nil
	}
}

func OneOfValidator(validValues ...interface{}) ConfigValidator {
	return func(key string, value interface{}) error {
		for _, valid := range validValues {
			if reflect.DeepEqual(value, valid) {
				return nil
			}
		}
		return fmt.Errorf("configuration key %s must be one of: %v", key, validValues)
	}
}