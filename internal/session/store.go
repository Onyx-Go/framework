package session

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"
)

// DefaultStore implements the Store interface for session data serialization
type DefaultStore struct{}

// NewDefaultStore creates a new default store
func NewDefaultStore() *DefaultStore {
	return &DefaultStore{}
}

// Serialize converts session data to bytes
func (ds *DefaultStore) Serialize(ctx context.Context, data map[string]interface{}) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	
	// Convert data to JSON-serializable format
	serializable := make(map[string]interface{})
	
	for key, value := range data {
		serialized, err := ds.makeSerializable(value)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize key %s: %w", key, err)
		}
		serializable[key] = serialized
	}
	
	// Marshal to JSON
	bytes, err := json.Marshal(serializable)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal session data: %w", err)
	}
	
	return bytes, nil
}

// Deserialize converts bytes back to session data
func (ds *DefaultStore) Deserialize(ctx context.Context, data []byte) (map[string]interface{}, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	
	if len(data) == 0 {
		return make(map[string]interface{}), nil
	}
	
	var serialized map[string]interface{}
	if err := json.Unmarshal(data, &serialized); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session data: %w", err)
	}
	
	// Convert back from serialized format
	result := make(map[string]interface{})
	
	for key, value := range serialized {
		deserialized, err := ds.makeDeserializable(value)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize key %s: %w", key, err)
		}
		result[key] = deserialized
	}
	
	return result, nil
}

// Validate checks if session data is valid
func (ds *DefaultStore) Validate(ctx context.Context, data map[string]interface{}) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	
	// Basic validation rules
	for key, value := range data {
		// Check key constraints
		if len(key) > 255 {
			return fmt.Errorf("session key %s exceeds maximum length", key)
		}
		
		// Check value constraints
		if err := ds.validateValue(key, value); err != nil {
			return err
		}
	}
	
	return nil
}

// Transform applies transformations to session data
func (ds *DefaultStore) Transform(ctx context.Context, data map[string]interface{}) (map[string]interface{}, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	
	result := make(map[string]interface{})
	
	for key, value := range data {
		// Apply transformations (e.g., encryption, compression)
		transformed, err := ds.transformValue(key, value)
		if err != nil {
			return nil, fmt.Errorf("failed to transform key %s: %w", key, err)
		}
		result[key] = transformed
	}
	
	return result, nil
}

// makeSerializable converts values to JSON-serializable format
func (ds *DefaultStore) makeSerializable(value interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	
	switch v := value.(type) {
	case time.Time:
		return map[string]interface{}{
			"_type":  "time.Time",
			"_value": v.Format(time.RFC3339Nano),
		}, nil
	case time.Duration:
		return map[string]interface{}{
			"_type":  "time.Duration",
			"_value": int64(v),
		}, nil
	case []byte:
		return map[string]interface{}{
			"_type":  "[]byte",
			"_value": string(v),
		}, nil
	case complex64, complex128:
		return nil, fmt.Errorf("complex numbers are not supported")
	case chan interface{}:
		return nil, fmt.Errorf("channels are not supported")
	case func():
		return nil, fmt.Errorf("functions are not supported")
	default:
		// Check if it's a pointer, slice, or map that might contain unsupported types
		val := reflect.ValueOf(value)
		switch val.Kind() {
		case reflect.Ptr:
			if val.IsNil() {
				return nil, nil
			}
			return ds.makeSerializable(val.Elem().Interface())
		case reflect.Slice, reflect.Array:
			result := make([]interface{}, val.Len())
			for i := 0; i < val.Len(); i++ {
				serialized, err := ds.makeSerializable(val.Index(i).Interface())
				if err != nil {
					return nil, err
				}
				result[i] = serialized
			}
			return result, nil
		case reflect.Map:
			if val.Type().Key().Kind() != reflect.String {
				return nil, fmt.Errorf("only string-keyed maps are supported")
			}
			result := make(map[string]interface{})
			for _, key := range val.MapKeys() {
				serialized, err := ds.makeSerializable(val.MapIndex(key).Interface())
				if err != nil {
					return nil, err
				}
				result[key.String()] = serialized
			}
			return result, nil
		default:
			// For basic types (string, int, float, bool), return as-is
			return value, nil
		}
	}
}

// makeDeserializable converts values back from serialized format
func (ds *DefaultStore) makeDeserializable(value interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	
	// Check if it's a special serialized type
	if valueMap, ok := value.(map[string]interface{}); ok {
		if typeStr, hasType := valueMap["_type"]; hasType {
			switch typeStr {
			case "time.Time":
				if timeStr, ok := valueMap["_value"].(string); ok {
					return time.Parse(time.RFC3339Nano, timeStr)
				}
			case "time.Duration":
				if durationInt, ok := valueMap["_value"].(float64); ok {
					return time.Duration(int64(durationInt)), nil
				}
			case "[]byte":
				if byteStr, ok := valueMap["_value"].(string); ok {
					return []byte(byteStr), nil
				}
			}
		} else {
			// Regular map, deserialize recursively
			result := make(map[string]interface{})
			for k, v := range valueMap {
				deserialized, err := ds.makeDeserializable(v)
				if err != nil {
					return nil, err
				}
				result[k] = deserialized
			}
			return result, nil
		}
	}
	
	// Check if it's a slice
	if valueSlice, ok := value.([]interface{}); ok {
		result := make([]interface{}, len(valueSlice))
		for i, item := range valueSlice {
			deserialized, err := ds.makeDeserializable(item)
			if err != nil {
				return nil, err
			}
			result[i] = deserialized
		}
		return result, nil
	}
	
	// For basic types, return as-is
	return value, nil
}

// validateValue validates a session value
func (ds *DefaultStore) validateValue(key string, value interface{}) error {
	if value == nil {
		return nil
	}
	
	// Check value size (estimate)
	if ds.estimateSize(value) > 1024*1024 { // 1MB limit
		return fmt.Errorf("session value for key %s exceeds size limit", key)
	}
	
	// Check for unsupported types
	return ds.checkSupportedType(value)
}

// estimateSize estimates the memory size of a value
func (ds *DefaultStore) estimateSize(value interface{}) int {
	if value == nil {
		return 0
	}
	
	switch v := value.(type) {
	case string:
		return len(v)
	case []byte:
		return len(v)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return 8
	case float32, float64:
		return 8
	case bool:
		return 1
	case time.Time:
		return 32
	case time.Duration:
		return 8
	default:
		val := reflect.ValueOf(value)
		switch val.Kind() {
		case reflect.Slice, reflect.Array:
			size := 0
			for i := 0; i < val.Len(); i++ {
				size += ds.estimateSize(val.Index(i).Interface())
			}
			return size
		case reflect.Map:
			size := 0
			for _, key := range val.MapKeys() {
				size += ds.estimateSize(key.Interface())
				size += ds.estimateSize(val.MapIndex(key).Interface())
			}
			return size
		case reflect.Ptr:
			if val.IsNil() {
				return 0
			}
			return ds.estimateSize(val.Elem().Interface())
		default:
			return 100 // Default estimate for unknown types
		}
	}
}

// checkSupportedType checks if a type is supported for session storage
func (ds *DefaultStore) checkSupportedType(value interface{}) error {
	if value == nil {
		return nil
	}
	
	val := reflect.ValueOf(value)
	switch val.Kind() {
	case reflect.Chan, reflect.Func, reflect.UnsafePointer:
		return fmt.Errorf("unsupported type: %T", value)
	case reflect.Ptr:
		if val.IsNil() {
			return nil
		}
		return ds.checkSupportedType(val.Elem().Interface())
	case reflect.Slice, reflect.Array:
		for i := 0; i < val.Len(); i++ {
			if err := ds.checkSupportedType(val.Index(i).Interface()); err != nil {
				return err
			}
		}
	case reflect.Map:
		for _, key := range val.MapKeys() {
			if err := ds.checkSupportedType(key.Interface()); err != nil {
				return err
			}
			if err := ds.checkSupportedType(val.MapIndex(key).Interface()); err != nil {
				return err
			}
		}
	}
	
	return nil
}

// transformValue applies transformations to a value (placeholder for encryption, etc.)
func (ds *DefaultStore) transformValue(key string, value interface{}) (interface{}, error) {
	// For now, just return the value as-is
	// This is where you would implement encryption, compression, etc.
	return value, nil
}