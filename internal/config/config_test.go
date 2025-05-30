package config

import (
	"os"
	"testing"
	"time"
)

func TestNewConfig(t *testing.T) {
	config := NewConfig()
	
	if config == nil {
		t.Fatal("NewConfig() returned nil")
	}
	
	if config.values == nil {
		t.Error("config.values should be initialized")
	}
	
	if config.providers == nil {
		t.Error("config.providers should be initialized")
	}
	
	if config.validators == nil {
		t.Error("config.validators should be initialized")
	}
	
	if config.cache == nil {
		t.Error("config.cache should be initialized")
	}
}

func TestConfigGetString(t *testing.T) {
	config := NewConfig()
	config.values = map[string]interface{}{
		"test_string": "hello",
		"test_int":    42,
		"test_bool":   true,
	}
	
	// Test string value
	if val := config.GetString("test_string"); val != "hello" {
		t.Errorf("expected 'hello', got '%s'", val)
	}
	
	// Test conversion from int
	if val := config.GetString("test_int"); val != "42" {
		t.Errorf("expected '42', got '%s'", val)
	}
	
	// Test default value
	if val := config.GetString("nonexistent", "default"); val != "default" {
		t.Errorf("expected 'default', got '%s'", val)
	}
	
	// Test empty default
	if val := config.GetString("nonexistent"); val != "" {
		t.Errorf("expected empty string, got '%s'", val)
	}
}

func TestConfigGetInt(t *testing.T) {
	config := NewConfig()
	config.values = map[string]interface{}{
		"test_int":    42,
		"test_string": "123",
		"test_float":  45.6,
	}
	
	// Test int value
	if val := config.GetInt("test_int"); val != 42 {
		t.Errorf("expected 42, got %d", val)
	}
	
	// Test conversion from string
	if val := config.GetInt("test_string"); val != 123 {
		t.Errorf("expected 123, got %d", val)
	}
	
	// Test conversion from float
	if val := config.GetInt("test_float"); val != 45 {
		t.Errorf("expected 45, got %d", val)
	}
	
	// Test default value
	if val := config.GetInt("nonexistent", 99); val != 99 {
		t.Errorf("expected 99, got %d", val)
	}
}

func TestConfigGetBool(t *testing.T) {
	config := NewConfig()
	config.values = map[string]interface{}{
		"test_bool_true":  true,
		"test_bool_false": false,
		"test_string_true": "true",
		"test_string_1":    "1",
		"test_string_yes":  "yes",
		"test_string_false": "false",
		"test_string_0":     "0",
		"test_string_no":    "no",
	}
	
	tests := []struct {
		key      string
		expected bool
	}{
		{"test_bool_true", true},
		{"test_bool_false", false},
		{"test_string_true", true},
		{"test_string_1", true},
		{"test_string_yes", true},
		{"test_string_false", false},
		{"test_string_0", false},
		{"test_string_no", false},
	}
	
	for _, test := range tests {
		if val := config.GetBool(test.key); val != test.expected {
			t.Errorf("key %s: expected %v, got %v", test.key, test.expected, val)
		}
	}
	
	// Test default value
	if val := config.GetBool("nonexistent", true); val != true {
		t.Errorf("expected true, got %v", val)
	}
}

func TestConfigGetDuration(t *testing.T) {
	config := NewConfig()
	config.values = map[string]interface{}{
		"test_duration": 5 * time.Second,
		"test_string":   "10s",
	}
	
	// Test duration value
	if val := config.GetDuration("test_duration"); val != 5*time.Second {
		t.Errorf("expected 5s, got %v", val)
	}
	
	// Test conversion from string
	if val := config.GetDuration("test_string"); val != 10*time.Second {
		t.Errorf("expected 10s, got %v", val)
	}
	
	// Test default value
	expected := 30 * time.Second
	if val := config.GetDuration("nonexistent", expected); val != expected {
		t.Errorf("expected %v, got %v", expected, val)
	}
}

func TestConfigGetStringSlice(t *testing.T) {
	config := NewConfig()
	config.values = map[string]interface{}{
		"test_slice":     []string{"a", "b", "c"},
		"test_interface": []interface{}{"x", "y", "z"},
		"test_string":    "1,2,3",
	}
	
	// Test string slice
	val := config.GetStringSlice("test_slice")
	expected := []string{"a", "b", "c"}
	if len(val) != len(expected) {
		t.Errorf("expected length %d, got %d", len(expected), len(val))
	}
	for i, v := range val {
		if v != expected[i] {
			t.Errorf("expected %s at index %d, got %s", expected[i], i, v)
		}
	}
	
	// Test interface slice conversion
	val = config.GetStringSlice("test_interface")
	expected = []string{"x", "y", "z"}
	if len(val) != len(expected) {
		t.Errorf("expected length %d, got %d", len(expected), len(val))
	}
	
	// Test string splitting
	val = config.GetStringSlice("test_string")
	expected = []string{"1", "2", "3"}
	if len(val) != len(expected) {
		t.Errorf("expected length %d, got %d", len(expected), len(val))
	}
}

func TestConfigNestedValues(t *testing.T) {
	config := NewConfig()
	config.values = map[string]interface{}{
		"database": map[string]interface{}{
			"host": "localhost",
			"port": 5432,
			"credentials": map[string]interface{}{
				"username": "admin",
				"password": "secret",
			},
		},
	}
	
	// Test nested string
	if val := config.GetString("database.host"); val != "localhost" {
		t.Errorf("expected 'localhost', got '%s'", val)
	}
	
	// Test nested int
	if val := config.GetInt("database.port"); val != 5432 {
		t.Errorf("expected 5432, got %d", val)
	}
	
	// Test deeply nested value
	if val := config.GetString("database.credentials.username"); val != "admin" {
		t.Errorf("expected 'admin', got '%s'", val)
	}
	
	// Test nonexistent nested key
	if val := config.GetString("database.nonexistent"); val != "" {
		t.Errorf("expected empty string, got '%s'", val)
	}
}

func TestConfigSet(t *testing.T) {
	config := NewConfig()
	
	// Test setting simple value
	err := config.Set("test_key", "test_value")
	if err != nil {
		t.Fatalf("Set() returned error: %v", err)
	}
	
	if val := config.GetString("test_key"); val != "test_value" {
		t.Errorf("expected 'test_value', got '%s'", val)
	}
	
	// Test setting nested value
	err = config.Set("nested.key", "nested_value")
	if err != nil {
		t.Fatalf("Set() returned error: %v", err)
	}
	
	if val := config.GetString("nested.key"); val != "nested_value" {
		t.Errorf("expected 'nested_value', got '%s'", val)
	}
}

func TestConfigHas(t *testing.T) {
	config := NewConfig()
	config.values = map[string]interface{}{
		"existing": "value",
		"nested": map[string]interface{}{
			"key": "value",
		},
	}
	
	// Test existing key
	if !config.Has("existing") {
		t.Error("Has() should return true for existing key")
	}
	
	// Test nested key
	if !config.Has("nested.key") {
		t.Error("Has() should return true for nested key")
	}
	
	// Test nonexistent key
	if config.Has("nonexistent") {
		t.Error("Has() should return false for nonexistent key")
	}
}

func TestConfigCache(t *testing.T) {
	config := NewConfig()
	config.values = map[string]interface{}{
		"test_key": "test_value",
	}
	
	// First access should cache the value
	val1 := config.Get("test_key")
	if val1 != "test_value" {
		t.Errorf("expected 'test_value', got '%v'", val1)
	}
	
	// Change underlying value
	config.values["test_key"] = "changed_value"
	
	// Second access should return cached value
	val2 := config.Get("test_key")
	if val2 != "test_value" {
		t.Errorf("expected cached value 'test_value', got '%v'", val2)
	}
	
	// Clear cache and access again
	config.ClearCache()
	val3 := config.Get("test_key")
	if val3 != "changed_value" {
		t.Errorf("expected 'changed_value' after cache clear, got '%v'", val3)
	}
}

func TestConfigValidation(t *testing.T) {
	config := NewConfig()
	
	// Add a required validator
	config.AddValidator("required_key", RequiredValidator)
	
	// Try to set nil value (should fail)
	err := config.Set("required_key", nil)
	if err == nil {
		t.Error("Set() should return error for nil value with required validator")
	}
	
	// Try to set empty string (should fail)
	err = config.Set("required_key", "")
	if err == nil {
		t.Error("Set() should return error for empty string with required validator")
	}
	
	// Set valid value (should succeed)
	err = config.Set("required_key", "valid_value")
	if err != nil {
		t.Errorf("Set() should not return error for valid value: %v", err)
	}
}

func TestConfigAll(t *testing.T) {
	config := NewConfig()
	config.values = map[string]interface{}{
		"key1": "value1",
		"key2": 42,
		"nested": map[string]interface{}{
			"key3": "value3",
		},
	}
	
	all := config.All()
	
	// Check that we get a copy, not the original
	if &all == &config.values {
		t.Error("All() should return a copy, not the original map")
	}
	
	// Check values
	if all["key1"] != "value1" {
		t.Errorf("expected 'value1', got '%v'", all["key1"])
	}
	
	if all["key2"] != 42 {
		t.Errorf("expected 42, got '%v'", all["key2"])
	}
	
	// Check nested value
	nested, ok := all["nested"].(map[string]interface{})
	if !ok {
		t.Error("nested value should be a map")
	}
	
	if nested["key3"] != "value3" {
		t.Errorf("expected 'value3', got '%v'", nested["key3"])
	}
}

func TestConfigEnvironmentValues(t *testing.T) {
	config := NewConfig()
	
	// Test environment detection
	if config.Env() == "" {
		t.Error("Env() should return a non-empty value")
	}
	
	if config.AppName() == "" {
		t.Error("AppName() should return a non-empty value")
	}
	
	// Debug should be false by default (unless APP_DEBUG is set)
	if config.Debug() && os.Getenv("APP_DEBUG") != "true" {
		t.Error("Debug() should return false by default")
	}
}