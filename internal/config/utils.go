package config

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ParseValue attempts to parse a string value into its appropriate type
func ParseValue(value string) interface{} {
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

// RemoveQuotes removes surrounding quotes from a string value
func RemoveQuotes(value string) string {
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') ||
		   (value[0] == '\'' && value[len(value)-1] == '\'') {
			return value[1 : len(value)-1]
		}
	}
	return value
}

// ExpandVariables expands ${VAR} style variables in a string
func ExpandVariables(value string, env map[string]interface{}) string {
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

// NormalizeKey normalizes a configuration key to lowercase with dots
func NormalizeKey(key string) string {
	// Convert underscores to dots and make lowercase
	return strings.ToLower(strings.ReplaceAll(key, "_", "."))
}

// SplitKey splits a dot-separated key into its components
func SplitKey(key string) []string {
	return strings.Split(key, ".")
}

// JoinKey joins key components with dots
func JoinKey(parts ...string) string {
	return strings.Join(parts, ".")
}

// DeepCopyMap creates a deep copy of a map[string]interface{}
func DeepCopyMap(src map[string]interface{}) map[string]interface{} {
	dst := make(map[string]interface{})
	for k, v := range src {
		switch val := v.(type) {
		case map[string]interface{}:
			dst[k] = DeepCopyMap(val)
		case []interface{}:
			newSlice := make([]interface{}, len(val))
			copy(newSlice, val)
			dst[k] = newSlice
		default:
			dst[k] = v
		}
	}
	return dst
}

// MergeMaps merges multiple maps, with later maps overriding earlier ones
func MergeMaps(maps ...map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	
	for _, m := range maps {
		for k, v := range m {
			if existingMap, ok := result[k].(map[string]interface{}); ok {
				if newMap, ok := v.(map[string]interface{}); ok {
					result[k] = MergeMaps(existingMap, newMap)
					continue
				}
			}
			result[k] = v
		}
	}
	
	return result
}

// FlattenMap flattens a nested map using dot notation
func FlattenMap(m map[string]interface{}, prefix string) map[string]interface{} {
	result := make(map[string]interface{})
	
	for k, v := range m {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		
		if nested, ok := v.(map[string]interface{}); ok {
			for nestedKey, nestedValue := range FlattenMap(nested, key) {
				result[nestedKey] = nestedValue
			}
		} else {
			result[key] = v
		}
	}
	
	return result
}

// UnflattenMap converts a flattened map back to nested structure
func UnflattenMap(flat map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	
	for key, value := range flat {
		setNestedValue(result, key, value)
	}
	
	return result
}

func setNestedValue(m map[string]interface{}, key string, value interface{}) {
	keys := strings.Split(key, ".")
	current := m
	
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