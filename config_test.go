package onyx

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewConfig(t *testing.T) {
	config := NewConfig()
	
	if config == nil {
		t.Fatal("NewConfig() returned nil")
	}
	
	if config.env == "" {
		t.Error("Environment not set")
	}
	
	if config.appName == "" {
		t.Error("App name not set")
	}
	
	if len(config.providers) == 0 {
		t.Error("No providers added by default")
	}
}

func TestConfigGetString(t *testing.T) {
	config := NewConfig()
	
	// Test with default value
	value := config.GetString("nonexistent", "default")
	if value != "default" {
		t.Errorf("Expected 'default', got '%s'", value)
	}
	
	// Test without default value
	value = config.GetString("nonexistent")
	if value != "" {
		t.Errorf("Expected empty string, got '%s'", value)
	}
	
	// Test setting and getting
	config.Set("test_string", "hello world")
	value = config.GetString("test_string")
	if value != "hello world" {
		t.Errorf("Expected 'hello world', got '%s'", value)
	}
}

func TestConfigGetInt(t *testing.T) {
	config := NewConfig()
	
	// Test with default value
	value := config.GetInt("nonexistent", 42)
	if value != 42 {
		t.Errorf("Expected 42, got %d", value)
	}
	
	// Test without default value
	value = config.GetInt("nonexistent")
	if value != 0 {
		t.Errorf("Expected 0, got %d", value)
	}
	
	// Test setting and getting
	config.Set("test_int", 123)
	value = config.GetInt("test_int")
	if value != 123 {
		t.Errorf("Expected 123, got %d", value)
	}
	
	// Test string to int conversion
	config.Set("test_string_int", "456")
	value = config.GetInt("test_string_int")
	if value != 456 {
		t.Errorf("Expected 456, got %d", value)
	}
}

func TestConfigGetBool(t *testing.T) {
	config := NewConfig()
	
	// Test true values
	trueValues := []string{"true", "1", "yes", "on", "enable", "enabled", "TRUE", "YES"}
	for _, val := range trueValues {
		config.Set("test_bool", val)
		if !config.GetBool("test_bool") {
			t.Errorf("Expected true for value '%s'", val)
		}
	}
	
	// Test false values
	falseValues := []string{"false", "0", "no", "off", "disable", "disabled", "FALSE", "NO"}
	for _, val := range falseValues {
		config.Set("test_bool", val)
		if config.GetBool("test_bool") {
			t.Errorf("Expected false for value '%s'", val)
		}
	}
	
	// Test default value
	if !config.GetBool("nonexistent", true) {
		t.Error("Expected true default value")
	}
}

func TestConfigGetDuration(t *testing.T) {
	config := NewConfig()
	
	// Test duration parsing
	config.Set("test_duration", "5m30s")
	duration := config.GetDuration("test_duration")
	expected := 5*time.Minute + 30*time.Second
	if duration != expected {
		t.Errorf("Expected %v, got %v", expected, duration)
	}
	
	// Test default value
	defaultDuration := 10 * time.Second
	duration = config.GetDuration("nonexistent", defaultDuration)
	if duration != defaultDuration {
		t.Errorf("Expected %v, got %v", defaultDuration, duration)
	}
}

func TestConfigGetStringSlice(t *testing.T) {
	config := NewConfig()
	
	// Test comma-separated string
	config.Set("test_slice", "one,two,three")
	slice := config.GetStringSlice("test_slice")
	expected := []string{"one", "two", "three"}
	
	if len(slice) != len(expected) {
		t.Fatalf("Expected slice length %d, got %d", len(expected), len(slice))
	}
	
	for i, val := range expected {
		if slice[i] != val {
			t.Errorf("Expected slice[%d] = '%s', got '%s'", i, val, slice[i])
		}
	}
	
	// Test default value
	defaultSlice := []string{"default1", "default2"}
	slice = config.GetStringSlice("nonexistent", defaultSlice)
	if len(slice) != len(defaultSlice) {
		t.Errorf("Expected default slice length %d, got %d", len(defaultSlice), len(slice))
	}
}

func TestConfigNestedValues(t *testing.T) {
	config := NewConfig()
	
	// Test nested setting
	config.Set("database.host", "localhost")
	config.Set("database.port", 5432)
	config.Set("database.options.ssl", true)
	
	// Test nested getting
	host := config.GetString("database.host")
	if host != "localhost" {
		t.Errorf("Expected 'localhost', got '%s'", host)
	}
	
	port := config.GetInt("database.port")
	if port != 5432 {
		t.Errorf("Expected 5432, got %d", port)
	}
	
	ssl := config.GetBool("database.options.ssl")
	if !ssl {
		t.Error("Expected true for SSL option")
	}
}

func TestConfigValidation(t *testing.T) {
	config := NewConfig()
	
	// Add required validator
	config.AddValidator("required_field", RequiredValidator)
	
	// Test validation failure
	err := config.Set("required_field", "")
	if err == nil {
		t.Error("Expected validation error for empty required field")
	}
	
	// Test validation success
	err = config.Set("required_field", "valid_value")
	if err != nil {
		t.Errorf("Unexpected validation error: %v", err)
	}
	
	// Add range validator
	config.AddValidator("port", IntRangeValidator(1, 65535))
	
	// Test range validation failure
	err = config.Set("port", "70000")
	if err == nil {
		t.Error("Expected validation error for port out of range")
	}
	
	// Test range validation success
	err = config.Set("port", "8080")
	if err != nil {
		t.Errorf("Unexpected validation error: %v", err)
	}
}

func TestConfigCache(t *testing.T) {
	config := NewConfig()
	config.EnableCache(100 * time.Millisecond)
	
	// Set a value
	config.Set("cached_value", "test")
	
	// Get it (should be cached)
	value1 := config.GetString("cached_value")
	
	// Change the underlying value directly
	config.values["cached_value"] = "changed"
	
	// Get it again (should still return cached value)
	value2 := config.GetString("cached_value")
	
	if value1 != value2 {
		t.Error("Cache not working - values should be the same")
	}
	
	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)
	
	// Get it again (should return new value)
	value3 := config.GetString("cached_value")
	
	if value3 != "changed" {
		t.Errorf("Expected 'changed' after cache expiry, got '%s'", value3)
	}
}

func TestEnvProvider(t *testing.T) {
	// Set environment variable
	os.Setenv("TEST_ENV_VAR", "test_value")
	defer os.Unsetenv("TEST_ENV_VAR")
	
	provider := &EnvProvider{}
	values, err := provider.Load()
	
	if err != nil {
		t.Fatalf("EnvProvider.Load() failed: %v", err)
	}
	
	if testValue, exists := values["test_env_var"]; !exists || testValue != "test_value" {
		t.Errorf("Expected 'test_value' for TEST_ENV_VAR, got '%v'", testValue)
	}
	
	if provider.Name() != "env" {
		t.Errorf("Expected provider name 'env', got '%s'", provider.Name())
	}
}

func TestDotEnvProvider(t *testing.T) {
	// Create temporary .env file
	tempDir := t.TempDir()
	envFile := filepath.Join(tempDir, ".env")
	
	envContent := `# This is a comment
APP_NAME=TestApp
APP_DEBUG=true
DB_PORT=5432
DB_TIMEOUT=30s
EMPTY_VALUE=
QUOTED_VALUE="hello world"
VARIABLE_EXPANSION=${APP_NAME}_expanded
`
	
	err := os.WriteFile(envFile, []byte(envContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test .env file: %v", err)
	}
	
	provider, err := NewDotEnvProvider(envFile)
	if err != nil {
		t.Fatalf("NewDotEnvProvider failed: %v", err)
	}
	
	values, err := provider.Load()
	if err != nil {
		t.Fatalf("DotEnvProvider.Load() failed: %v", err)
	}
	
	// Test string value
	if appName, exists := values["app_name"]; !exists || appName != "TestApp" {
		t.Errorf("Expected 'TestApp' for app_name, got '%v'", appName)
	}
	
	// Test boolean value
	if debug, exists := values["app_debug"]; !exists || debug != true {
		t.Errorf("Expected true for app_debug, got '%v'", debug)
	}
	
	// Test integer value
	if port, exists := values["db_port"]; !exists || port != 5432 {
		t.Errorf("Expected 5432 for db_port, got '%v'", port)
	}
	
	// Test duration value
	if timeout, exists := values["db_timeout"]; !exists {
		t.Error("Expected db_timeout to exist")
	} else if duration, ok := timeout.(time.Duration); !ok || duration != 30*time.Second {
		t.Errorf("Expected 30s duration for db_timeout, got '%v'", timeout)
	}
	
	// Test quoted value
	if quoted, exists := values["quoted_value"]; !exists || quoted != "hello world" {
		t.Errorf("Expected 'hello world' for quoted_value, got '%v'", quoted)
	}
	
	// Test variable expansion
	if expanded, exists := values["variable_expansion"]; !exists || expanded != "TestApp_expanded" {
		t.Errorf("Expected 'TestApp_expanded' for variable_expansion, got '%v'", expanded)
	}
}

func TestFileProvider(t *testing.T) {
	// Create temporary config directory
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "config")
	err := os.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}
	
	// Create a JSON config file
	configContent := `{
		"host": "localhost",
		"port": 8080,
		"ssl": true,
		"features": ["feature1", "feature2"]
	}`
	
	configFile := filepath.Join(configDir, "database.json")
	err = os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}
	
	provider := &FileProvider{BasePath: configDir}
	values, err := provider.Load()
	if err != nil {
		t.Fatalf("FileProvider.Load() failed: %v", err)
	}
	
	// Check if database config was loaded
	if dbConfig, exists := values["database"]; !exists {
		t.Error("Expected database config to be loaded")
	} else if config, ok := dbConfig.(map[string]interface{}); !ok {
		t.Error("Expected database config to be a map")
	} else {
		if host := config["host"]; host != "localhost" {
			t.Errorf("Expected 'localhost' for host, got '%v'", host)
		}
		
		if port := config["port"]; port != float64(8080) { // JSON numbers are float64
			t.Errorf("Expected 8080 for port, got '%v'", port)
		}
		
		if ssl := config["ssl"]; ssl != true {
			t.Errorf("Expected true for ssl, got '%v'", ssl)
		}
	}
}

func TestConfigLoad(t *testing.T) {
	// Create temporary environment
	tempDir := t.TempDir()
	
	// Create .env file
	envFile := filepath.Join(tempDir, ".env")
	envContent := "APP_NAME=LoadTest\nAPP_DEBUG=true"
	err := os.WriteFile(envFile, []byte(envContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}
	
	// Create config directory and file
	configDir := filepath.Join(tempDir, "config")
	err = os.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}
	
	configContent := `{"database": {"host": "localhost", "port": 5432}}`
	configFile := filepath.Join(configDir, "app.json")
	err = os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}
	
	// Create config with custom providers
	config := &Config{
		values:     make(map[string]interface{}),
		providers:  make([]ConfigProvider, 0),
		validators: make(map[string]ConfigValidator),
		cache: &ConfigCache{
			enabled: true,
			ttl:     5 * time.Minute,
			cached:  make(map[string]cachedValue),
		},
	}
	
	// Add providers
	envProvider, err := NewDotEnvProvider(envFile)
	if err != nil {
		t.Fatalf("Failed to create DotEnvProvider: %v", err)
	}
	config.AddProvider(envProvider)
	config.AddProvider(&FileProvider{BasePath: configDir})
	
	// Load configuration
	err = config.Load()
	if err != nil {
		t.Fatalf("Config.Load() failed: %v", err)
	}
	
	// Test values from .env
	if appName := config.GetString("app_name"); appName != "LoadTest" {
		t.Errorf("Expected 'LoadTest' from .env, got '%s'", appName)
	}
	
	if !config.GetBool("app_debug") {
		t.Error("Expected true for app_debug from .env")
	}
	
	// Test values from JSON file
	if host := config.GetString("app.database.host"); host != "localhost" {
		t.Errorf("Expected 'localhost' from JSON config, got '%s'", host)
	}
	
	if port := config.GetInt("app.database.port"); port != 5432 {
		t.Errorf("Expected 5432 from JSON config, got %d", port)
	}
}

func TestConfigHas(t *testing.T) {
	config := NewConfig()
	
	// Test non-existent key
	if config.Has("nonexistent") {
		t.Error("Expected false for non-existent key")
	}
	
	// Test existing key
	config.Set("existing", "value")
	if !config.Has("existing") {
		t.Error("Expected true for existing key")
	}
	
	// Test nested key
	config.Set("nested.key", "value")
	if !config.Has("nested.key") {
		t.Error("Expected true for nested key")
	}
}

func TestConfigAll(t *testing.T) {
	config := NewConfig()
	
	// Set some values
	config.Set("key1", "value1")
	config.Set("key2", 42)
	config.Set("nested.key", true)
	
	all := config.All()
	
	if all["key1"] != "value1" {
		t.Errorf("Expected 'value1' for key1, got '%v'", all["key1"])
	}
	
	if all["key2"] != 42 {
		t.Errorf("Expected 42 for key2, got '%v'", all["key2"])
	}
	
	// Check nested structure
	if nested, ok := all["nested"].(map[string]interface{}); !ok {
		t.Error("Expected nested to be a map")
	} else if nested["key"] != true {
		t.Errorf("Expected true for nested.key, got '%v'", nested["key"])
	}
}

func TestParseValue(t *testing.T) {
	tests := []struct {
		input    string
		expected interface{}
	}{
		{"true", true},
		{"false", false},
		{"123", 123},
		{"123.45", 123.45},
		{"5m30s", 5*time.Minute + 30*time.Second},
		{"hello", "hello"},
	}
	
	for _, test := range tests {
		result := parseValue(test.input)
		if result != test.expected {
			t.Errorf("parseValue(%s): expected %v (%T), got %v (%T)",
				test.input, test.expected, test.expected, result, result)
		}
	}
}

func TestRemoveQuotes(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`"hello"`, "hello"},
		{`'world'`, "world"},
		{`"unclosed`, `"unclosed`},
		{`no quotes`, `no quotes`},
		{`""`, ``},
		{`''`, ``},
	}
	
	for _, test := range tests {
		result := removeQuotes(test.input)
		if result != test.expected {
			t.Errorf("removeQuotes(%s): expected '%s', got '%s'",
				test.input, test.expected, result)
		}
	}
}

func TestExpandVariables(t *testing.T) {
	env := map[string]interface{}{
		"app_name": "TestApp",
		"version":  "1.0",
	}
	
	tests := []struct {
		input    string
		expected string
	}{
		{"${app_name}", "TestApp"},
		{"${app_name}_${version}", "TestApp_1.0"},
		{"prefix_${app_name}_suffix", "prefix_TestApp_suffix"},
		{"${nonexistent}", "${nonexistent}"},
		{"no variables", "no variables"},
	}
	
	for _, test := range tests {
		result := expandVariables(test.input, env)
		if result != test.expected {
			t.Errorf("expandVariables(%s): expected '%s', got '%s'",
				test.input, test.expected, result)
		}
	}
}

func TestValidators(t *testing.T) {
	// Test RequiredValidator
	err := RequiredValidator("test", nil)
	if err == nil {
		t.Error("Expected error for nil value")
	}
	
	err = RequiredValidator("test", "")
	if err == nil {
		t.Error("Expected error for empty string")
	}
	
	err = RequiredValidator("test", "valid")
	if err != nil {
		t.Errorf("Unexpected error for valid value: %v", err)
	}
	
	// Test IntRangeValidator
	validator := IntRangeValidator(1, 100)
	
	err = validator("test", "0")
	if err == nil {
		t.Error("Expected error for value below range")
	}
	
	err = validator("test", "101")
	if err == nil {
		t.Error("Expected error for value above range")
	}
	
	err = validator("test", "50")
	if err != nil {
		t.Errorf("Unexpected error for value in range: %v", err)
	}
	
	// Test RegexValidator
	emailValidator := RegexValidator(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	
	err = emailValidator("email", "invalid-email")
	if err == nil {
		t.Error("Expected error for invalid email")
	}
	
	err = emailValidator("email", "test@example.com")
	if err != nil {
		t.Errorf("Unexpected error for valid email: %v", err)
	}
	
	// Test OneOfValidator
	oneOfValidator := OneOfValidator("dev", "prod", "test")
	
	err = oneOfValidator("env", "invalid")
	if err == nil {
		t.Error("Expected error for invalid value")
	}
	
	err = oneOfValidator("env", "dev")
	if err != nil {
		t.Errorf("Unexpected error for valid value: %v", err)
	}
}

func TestConfigReload(t *testing.T) {
	config := NewConfig()
	
	// Set initial value
	config.Set("test", "initial")
	
	// Clear values to simulate external changes
	config.values = make(map[string]interface{})
	
	// Reload should work (though it won't restore our manually set value
	// since we don't have providers that would reload it)
	err := config.Reload()
	if err != nil {
		t.Errorf("Unexpected error during reload: %v", err)
	}
}

func TestConfigClearCache(t *testing.T) {
	config := NewConfig()
	config.EnableCache(1 * time.Minute)
	
	// Set and get a value to cache it
	config.Set("cached", "value")
	config.GetString("cached")
	
	// Verify cache has entries
	config.cache.mutex.RLock()
	cacheSize := len(config.cache.cached)
	config.cache.mutex.RUnlock()
	
	if cacheSize == 0 {
		t.Error("Expected cache to have entries")
	}
	
	// Clear cache
	config.ClearCache()
	
	// Verify cache is empty
	config.cache.mutex.RLock()
	cacheSize = len(config.cache.cached)
	config.cache.mutex.RUnlock()
	
	if cacheSize != 0 {
		t.Error("Expected cache to be empty after clearing")
	}
}

func ExampleConfig_basic() {
	config := NewConfig()
	
	// Set some configuration values
	config.Set("app.name", "MyApp")
	config.Set("app.debug", true)
	config.Set("database.port", 5432)
	
	// Get configuration values with type conversion
	appName := config.GetString("app.name")
	debug := config.GetBool("app.debug")
	port := config.GetInt("database.port")
	
	fmt.Printf("App: %s, Debug: %t, Port: %d\n", appName, debug, port)
	// Output: App: MyApp, Debug: true, Port: 5432
}

func ExampleConfig_validation() {
	config := NewConfig()
	
	// Add validators
	config.AddValidator("port", IntRangeValidator(1, 65535))
	config.AddValidator("email", RegexValidator(`^[^@]+@[^@]+\.[^@]+$`))
	
	// Valid values
	config.Set("port", "8080")
	config.Set("email", "user@example.com")
	
	// Invalid values will return errors
	if err := config.Set("port", "70000"); err != nil {
		fmt.Printf("Port validation failed: %v\n", err)
	}
	
	if err := config.Set("email", "invalid-email"); err != nil {
		fmt.Printf("Email validation failed: %v\n", err)
	}
	// Output: Port validation failed: validation failed for key port: configuration key port must be between 1 and 65535
	// Email validation failed: validation failed for key email: configuration key email does not match required pattern
}