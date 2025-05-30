package session

import (
	"context"
	"testing"
	"time"
)

func TestDefaultStore_BasicSerialization(t *testing.T) {
	store := NewDefaultStore()
	ctx := context.Background()
	
	// Test basic data types
	data := map[string]interface{}{
		"string":  "hello world",
		"int":     42,
		"float":   3.14,
		"bool":    true,
		"nil":     nil,
	}
	
	// Serialize
	serialized, err := store.Serialize(ctx, data)
	if err != nil {
		t.Fatalf("Failed to serialize data: %v", err)
	}
	
	// Deserialize
	deserialized, err := store.Deserialize(ctx, serialized)
	if err != nil {
		t.Fatalf("Failed to deserialize data: %v", err)
	}
	
	// Verify data
	if deserialized["string"] != "hello world" {
		t.Errorf("String value mismatch: expected 'hello world', got %v", deserialized["string"])
	}
	
	if deserialized["int"].(float64) != 42 { // JSON unmarshals numbers as float64
		t.Errorf("Int value mismatch: expected 42, got %v", deserialized["int"])
	}
	
	if deserialized["float"] != 3.14 {
		t.Errorf("Float value mismatch: expected 3.14, got %v", deserialized["float"])
	}
	
	if deserialized["bool"] != true {
		t.Errorf("Bool value mismatch: expected true, got %v", deserialized["bool"])
	}
	
	if deserialized["nil"] != nil {
		t.Errorf("Nil value mismatch: expected nil, got %v", deserialized["nil"])
	}
}

func TestDefaultStore_ComplexTypes(t *testing.T) {
	store := NewDefaultStore()
	ctx := context.Background()
	
	now := time.Now()
	duration := 2 * time.Hour
	byteData := []byte("test bytes")
	
	data := map[string]interface{}{
		"time":     now,
		"duration": duration,
		"bytes":    byteData,
		"slice":    []interface{}{"a", "b", "c"},
		"map":      map[string]interface{}{"nested": "value"},
	}
	
	// Serialize
	serialized, err := store.Serialize(ctx, data)
	if err != nil {
		t.Fatalf("Failed to serialize complex data: %v", err)
	}
	
	// Deserialize
	deserialized, err := store.Deserialize(ctx, serialized)
	if err != nil {
		t.Fatalf("Failed to deserialize complex data: %v", err)
	}
	
	// Verify time
	if deserializedTime, ok := deserialized["time"].(time.Time); ok {
		if !deserializedTime.Equal(now) {
			t.Errorf("Time mismatch: expected %v, got %v", now, deserializedTime)
		}
	} else {
		t.Errorf("Time deserialization failed: got %T", deserialized["time"])
	}
	
	// Verify duration
	if deserializedDuration, ok := deserialized["duration"].(time.Duration); ok {
		if deserializedDuration != duration {
			t.Errorf("Duration mismatch: expected %v, got %v", duration, deserializedDuration)
		}
	} else {
		t.Errorf("Duration deserialization failed: got %T", deserialized["duration"])
	}
	
	// Verify bytes
	if deserializedBytes, ok := deserialized["bytes"].([]byte); ok {
		if string(deserializedBytes) != string(byteData) {
			t.Errorf("Bytes mismatch: expected %s, got %s", string(byteData), string(deserializedBytes))
		}
	} else {
		t.Errorf("Bytes deserialization failed: got %T", deserialized["bytes"])
	}
	
	// Verify slice
	if deserializedSlice, ok := deserialized["slice"].([]interface{}); ok {
		if len(deserializedSlice) != 3 {
			t.Errorf("Slice length mismatch: expected 3, got %d", len(deserializedSlice))
		}
		if deserializedSlice[0] != "a" {
			t.Errorf("Slice element mismatch: expected 'a', got %v", deserializedSlice[0])
		}
	} else {
		t.Errorf("Slice deserialization failed: got %T", deserialized["slice"])
	}
	
	// Verify nested map
	if deserializedMap, ok := deserialized["map"].(map[string]interface{}); ok {
		if deserializedMap["nested"] != "value" {
			t.Errorf("Nested map mismatch: expected 'value', got %v", deserializedMap["nested"])
		}
	} else {
		t.Errorf("Map deserialization failed: got %T", deserialized["map"])
	}
}

func TestDefaultStore_Validation(t *testing.T) {
	store := NewDefaultStore()
	ctx := context.Background()
	
	// Test valid data
	validData := map[string]interface{}{
		"key": "value",
	}
	
	if err := store.Validate(ctx, validData); err != nil {
		t.Errorf("Valid data should pass validation: %v", err)
	}
	
	// Test key too long
	longKey := make([]byte, 300)
	for i := range longKey {
		longKey[i] = 'a'
	}
	
	invalidData := map[string]interface{}{
		string(longKey): "value",
	}
	
	if err := store.Validate(ctx, invalidData); err == nil {
		t.Error("Long key should fail validation")
	}
}

func TestDefaultStore_UnsupportedTypes(t *testing.T) {
	store := NewDefaultStore()
	ctx := context.Background()
	
	// Test unsupported types
	unsupportedData := map[string]interface{}{
		"channel":  make(chan int),
		"function": func() {},
	}
	
	// Should fail serialization
	_, err := store.Serialize(ctx, unsupportedData)
	if err == nil {
		t.Error("Unsupported types should fail serialization")
	}
}

func TestDefaultStore_EmptyData(t *testing.T) {
	store := NewDefaultStore()
	ctx := context.Background()
	
	// Test empty data
	emptyData := make(map[string]interface{})
	
	serialized, err := store.Serialize(ctx, emptyData)
	if err != nil {
		t.Fatalf("Failed to serialize empty data: %v", err)
	}
	
	deserialized, err := store.Deserialize(ctx, serialized)
	if err != nil {
		t.Fatalf("Failed to deserialize empty data: %v", err)
	}
	
	if len(deserialized) != 0 {
		t.Errorf("Expected empty map, got %d elements", len(deserialized))
	}
	
	// Test empty bytes
	deserialized, err = store.Deserialize(ctx, []byte{})
	if err != nil {
		t.Fatalf("Failed to deserialize empty bytes: %v", err)
	}
	
	if len(deserialized) != 0 {
		t.Errorf("Expected empty map from empty bytes, got %d elements", len(deserialized))
	}
}

func TestDefaultStore_ContextCancellation(t *testing.T) {
	store := NewDefaultStore()
	
	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	
	data := map[string]interface{}{"key": "value"}
	
	// All operations should return context.Canceled error
	if _, err := store.Serialize(ctx, data); err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
	
	if _, err := store.Deserialize(ctx, []byte("{}")); err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
	
	if err := store.Validate(ctx, data); err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
	
	if _, err := store.Transform(ctx, data); err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

func TestDefaultStore_NestedStructures(t *testing.T) {
	store := NewDefaultStore()
	ctx := context.Background()
	
	// Test deeply nested structures
	nestedData := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"level3": []interface{}{
					map[string]interface{}{
						"deep": "value",
					},
				},
			},
		},
	}
	
	// Serialize
	serialized, err := store.Serialize(ctx, nestedData)
	if err != nil {
		t.Fatalf("Failed to serialize nested data: %v", err)
	}
	
	// Deserialize
	deserialized, err := store.Deserialize(ctx, serialized)
	if err != nil {
		t.Fatalf("Failed to deserialize nested data: %v", err)
	}
	
	// Navigate to deep value
	level1, ok := deserialized["level1"].(map[string]interface{})
	if !ok {
		t.Fatal("Level1 should be a map")
	}
	
	level2, ok := level1["level2"].(map[string]interface{})
	if !ok {
		t.Fatal("Level2 should be a map")
	}
	
	level3, ok := level2["level3"].([]interface{})
	if !ok {
		t.Fatal("Level3 should be a slice")
	}
	
	deepMap, ok := level3[0].(map[string]interface{})
	if !ok {
		t.Fatal("Deep element should be a map")
	}
	
	if deepMap["deep"] != "value" {
		t.Errorf("Deep value mismatch: expected 'value', got %v", deepMap["deep"])
	}
}

func TestDefaultStore_SizeEstimation(t *testing.T) {
	store := NewDefaultStore()
	
	tests := []struct {
		name     string
		value    interface{}
		minSize  int
		maxSize  int
	}{
		{"string", "hello", 5, 5},
		{"int", 42, 8, 8},
		{"bool", true, 1, 1},
		{"bytes", []byte("test"), 4, 4},
		{"nil", nil, 0, 0},
		{"slice", []interface{}{"a", "b"}, 2, 10},
		{"map", map[string]interface{}{"key": "value"}, 8, 20},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			size := store.estimateSize(test.value)
			if size < test.minSize || size > test.maxSize {
				t.Errorf("Size estimation for %s: expected between %d and %d, got %d", test.name, test.minSize, test.maxSize, size)
			}
		})
	}
}

func TestDefaultStore_TypeValidation(t *testing.T) {
	store := NewDefaultStore()
	
	// Test supported types
	supportedTypes := []interface{}{
		"string",
		42,
		3.14,
		true,
		time.Now(),
		time.Hour,
		[]byte("bytes"),
		[]interface{}{"slice"},
		map[string]interface{}{"map": "value"},
	}
	
	for _, value := range supportedTypes {
		if err := store.checkSupportedType(value); err != nil {
			t.Errorf("Type %T should be supported: %v", value, err)
		}
	}
	
	// Test unsupported types
	unsupportedTypes := []interface{}{
		make(chan int),
		func() {},
	}
	
	for _, value := range unsupportedTypes {
		if err := store.checkSupportedType(value); err == nil {
			t.Errorf("Type %T should be unsupported", value)
		}
	}
}