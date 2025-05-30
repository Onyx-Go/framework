package config

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
)

// RequiredValidator ensures a configuration value is not nil or empty
func RequiredValidator(key string, value interface{}) error {
	if value == nil {
		return fmt.Errorf("configuration key %s is required", key)
	}
	
	if str, ok := value.(string); ok && str == "" {
		return fmt.Errorf("configuration key %s cannot be empty", key)
	}
	
	return nil
}

// IntRangeValidator validates that an integer value is within the specified range
func IntRangeValidator(min, max int) ConfigValidator {
	return func(key string, value interface{}) error {
		var intVal int
		
		switch v := value.(type) {
		case int:
			intVal = v
		case int64:
			intVal = int(v)
		case float64:
			intVal = int(v)
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

// FloatRangeValidator validates that a float value is within the specified range
func FloatRangeValidator(min, max float64) ConfigValidator {
	return func(key string, value interface{}) error {
		var floatVal float64
		
		switch v := value.(type) {
		case float64:
			floatVal = v
		case float32:
			floatVal = float64(v)
		case int:
			floatVal = float64(v)
		case int64:
			floatVal = float64(v)
		case string:
			var err error
			if floatVal, err = strconv.ParseFloat(v, 64); err != nil {
				return fmt.Errorf("configuration key %s must be a number", key)
			}
		default:
			return fmt.Errorf("configuration key %s must be a number", key)
		}
		
		if floatVal < min || floatVal > max {
			return fmt.Errorf("configuration key %s must be between %.2f and %.2f", key, min, max)
		}
		
		return nil
	}
}

// RegexValidator validates that a string value matches the specified regex pattern
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

// OneOfValidator validates that a value is one of the allowed values
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

// StringLengthValidator validates the length of a string value
func StringLengthValidator(minLen, maxLen int) ConfigValidator {
	return func(key string, value interface{}) error {
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("configuration key %s must be a string", key)
		}
		
		length := len(str)
		if length < minLen {
			return fmt.Errorf("configuration key %s must be at least %d characters", key, minLen)
		}
		
		if maxLen > 0 && length > maxLen {
			return fmt.Errorf("configuration key %s must be at most %d characters", key, maxLen)
		}
		
		return nil
	}
}

// URLValidator validates that a string is a valid URL
func URLValidator(key string, value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("configuration key %s must be a string", key)
	}
	
	// Simple URL validation - could be enhanced with net/url
	urlPattern := `^https?://[^\s/$.?#].[^\s]*$`
	re := regexp.MustCompile(urlPattern)
	
	if !re.MatchString(str) {
		return fmt.Errorf("configuration key %s must be a valid URL", key)
	}
	
	return nil
}

// EmailValidator validates that a string is a valid email address
func EmailValidator(key string, value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("configuration key %s must be a string", key)
	}
	
	// Basic email validation pattern
	emailPattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	re := regexp.MustCompile(emailPattern)
	
	if !re.MatchString(str) {
		return fmt.Errorf("configuration key %s must be a valid email address", key)
	}
	
	return nil
}

// BoolValidator validates that a value can be converted to a boolean
func BoolValidator(key string, value interface{}) error {
	switch value.(type) {
	case bool:
		return nil
	case string:
		str := value.(string)
		switch str {
		case "true", "false", "1", "0", "yes", "no", "on", "off", "enable", "disable", "enabled", "disabled":
			return nil
		default:
			return fmt.Errorf("configuration key %s must be a valid boolean value", key)
		}
	case int:
		intVal := value.(int)
		if intVal == 0 || intVal == 1 {
			return nil
		}
		return fmt.Errorf("configuration key %s boolean integer must be 0 or 1", key)
	default:
		return fmt.Errorf("configuration key %s must be a boolean", key)
	}
}

// TypeValidator validates that a value is of the specified type
func TypeValidator(expectedType reflect.Type) ConfigValidator {
	return func(key string, value interface{}) error {
		if value == nil {
			return fmt.Errorf("configuration key %s cannot be nil", key)
		}
		
		actualType := reflect.TypeOf(value)
		if actualType != expectedType {
			return fmt.Errorf("configuration key %s must be of type %s, got %s", 
				key, expectedType.String(), actualType.String())
		}
		
		return nil
	}
}

// ChainValidator allows chaining multiple validators
func ChainValidator(validators ...ConfigValidator) ConfigValidator {
	return func(key string, value interface{}) error {
		for _, validator := range validators {
			if err := validator(key, value); err != nil {
				return err
			}
		}
		return nil
	}
}