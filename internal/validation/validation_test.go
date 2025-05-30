package validation

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestNewValidator(t *testing.T) {
	data := map[string]interface{}{
		"name":  "John Doe",
		"email": "john@example.com",
		"age":   30,
	}
	
	rules := map[string][]Rule{
		"name":  {NewRequiredRule(), NewMinRule("2")},
		"email": {NewRequiredRule(), NewEmailRule()},
		"age":   {NewRequiredRule(), NewNumericRule()},
	}
	
	validator := NewValidator(data, rules)
	if validator == nil {
		t.Fatal("Expected validator to be created")
	}
	
	result := validator.Validate(context.Background())
	if !result.IsValid() {
		t.Errorf("Expected validation to pass, got errors: %v", result.GetErrors())
	}
}

func TestRequiredRule(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		hasError bool
	}{
		{"valid string", "hello", false},
		{"empty string", "", true},
		{"nil value", nil, true},
		{"zero int", 0, false},
		{"empty slice", []interface{}{}, true},
		{"non-empty slice", []interface{}{"item"}, false},
	}
	
	rule := NewRequiredRule()
	ctx := context.Background()
	data := make(map[string]interface{})
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := rule.Apply(ctx, "field", test.value, data)
			hasError := err != nil
			
			if hasError != test.hasError {
				t.Errorf("Expected hasError=%v, got hasError=%v, error=%v", test.hasError, hasError, err)
			}
		})
	}
}

func TestMinRule(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		minValue string
		hasError bool
	}{
		{"string valid", "hello", "3", false},
		{"string too short", "hi", "3", true},
		{"int valid", 10, "5", false},
		{"int too small", 3, "5", true},
		{"float valid", 10.5, "5", false},
		{"float too small", 3.2, "5", true},
		{"slice valid", []interface{}{1, 2, 3}, "2", false},
		{"slice too short", []interface{}{1}, "2", true},
	}
	
	ctx := context.Background()
	data := make(map[string]interface{})
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rule := NewMinRule(test.minValue)
			err := rule.Apply(ctx, "field", test.value, data)
			hasError := err != nil
			
			if hasError != test.hasError {
				t.Errorf("Expected hasError=%v, got hasError=%v, error=%v", test.hasError, hasError, err)
			}
		})
	}
}

func TestMaxRule(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		maxValue string
		hasError bool
	}{
		{"string valid", "hi", "5", false},
		{"string too long", "hello world", "5", true},
		{"int valid", 3, "5", false},
		{"int too large", 10, "5", true},
		{"float valid", 3.2, "5", false},
		{"float too large", 10.5, "5", true},
		{"slice valid", []interface{}{1}, "2", false},
		{"slice too long", []interface{}{1, 2, 3}, "2", true},
	}
	
	ctx := context.Background()
	data := make(map[string]interface{})
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rule := NewMaxRule(test.maxValue)
			err := rule.Apply(ctx, "field", test.value, data)
			hasError := err != nil
			
			if hasError != test.hasError {
				t.Errorf("Expected hasError=%v, got hasError=%v, error=%v", test.hasError, hasError, err)
			}
		})
	}
}

func TestEmailRule(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		hasError bool
	}{
		{"valid email", "test@example.com", false},
		{"valid email with subdomain", "user@mail.example.com", false},
		{"invalid email no @", "testexample.com", true},
		{"invalid email no domain", "test@", true},
		{"invalid email no local part", "@example.com", true},
		{"empty string", "", true},
		{"non-string value", 123, true},
	}
	
	rule := NewEmailRule()
	ctx := context.Background()
	data := make(map[string]interface{})
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := rule.Apply(ctx, "field", test.value, data)
			hasError := err != nil
			
			if hasError != test.hasError {
				t.Errorf("Expected hasError=%v, got hasError=%v, error=%v", test.hasError, hasError, err)
			}
		})
	}
}

func TestNumericRule(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		hasError bool
	}{
		{"int", 123, false},
		{"int64", int64(123), false},
		{"float64", 123.45, false},
		{"numeric string", "123.45", false},
		{"non-numeric string", "abc", true},
		{"empty string", "", true},
		{"non-numeric type", []int{1, 2, 3}, true},
	}
	
	rule := NewNumericRule()
	ctx := context.Background()
	data := make(map[string]interface{})
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := rule.Apply(ctx, "field", test.value, data)
			hasError := err != nil
			
			if hasError != test.hasError {
				t.Errorf("Expected hasError=%v, got hasError=%v, error=%v", test.hasError, hasError, err)
			}
		})
	}
}

func TestAlphaRule(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		hasError bool
	}{
		{"valid alpha", "hello", false},
		{"uppercase", "HELLO", false},
		{"mixed case", "Hello", false},
		{"with numbers", "hello123", true},
		{"with spaces", "hello world", true},
		{"with symbols", "hello!", true},
		{"empty string", "", true},
		{"non-string", 123, true},
	}
	
	rule := NewAlphaRule()
	ctx := context.Background()
	data := make(map[string]interface{})
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := rule.Apply(ctx, "field", test.value, data)
			hasError := err != nil
			
			if hasError != test.hasError {
				t.Errorf("Expected hasError=%v, got hasError=%v, error=%v", test.hasError, hasError, err)
			}
		})
	}
}

func TestAlphaNumRule(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		hasError bool
	}{
		{"valid alpha", "hello", false},
		{"valid numeric", "123", false},
		{"valid alphanumeric", "hello123", false},
		{"with spaces", "hello 123", true},
		{"with symbols", "hello!", true},
		{"empty string", "", true},
		{"non-string", 123, true},
	}
	
	rule := NewAlphaNumRule()
	ctx := context.Background()
	data := make(map[string]interface{})
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := rule.Apply(ctx, "field", test.value, data)
			hasError := err != nil
			
			if hasError != test.hasError {
				t.Errorf("Expected hasError=%v, got hasError=%v, error=%v", test.hasError, hasError, err)
			}
		})
	}
}

func TestURLRule(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		hasError bool
	}{
		{"valid http URL", "http://example.com", false},
		{"valid https URL", "https://example.com", false},
		{"URL with path", "https://example.com/path", false},
		{"URL with query", "https://example.com/path?query=1", false},
		{"invalid URL", "not-a-url", true},
		{"empty string", "", true},
		{"non-string", 123, true},
	}
	
	rule := NewURLRule()
	ctx := context.Background()
	data := make(map[string]interface{})
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := rule.Apply(ctx, "field", test.value, data)
			hasError := err != nil
			
			if hasError != test.hasError {
				t.Errorf("Expected hasError=%v, got hasError=%v, error=%v", test.hasError, hasError, err)
			}
		})
	}
}

func TestInRule(t *testing.T) {
	rule := NewInRule([]string{"red", "green", "blue"})
	ctx := context.Background()
	data := make(map[string]interface{})
	
	tests := []struct {
		name     string
		value    interface{}
		hasError bool
	}{
		{"valid value", "red", false},
		{"valid value 2", "blue", false},
		{"invalid value", "yellow", true},
		{"empty string", "", true},
		{"numeric value", 123, true},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := rule.Apply(ctx, "field", test.value, data)
			hasError := err != nil
			
			if hasError != test.hasError {
				t.Errorf("Expected hasError=%v, got hasError=%v, error=%v", test.hasError, hasError, err)
			}
		})
	}
}

func TestConfirmedRule(t *testing.T) {
	rule := NewConfirmedRule()
	ctx := context.Background()
	
	tests := []struct {
		name     string
		data     map[string]interface{}
		field    string
		value    interface{}
		hasError bool
	}{
		{
			"matching confirmation",
			map[string]interface{}{"password": "secret", "password_confirmation": "secret"},
			"password",
			"secret",
			false,
		},
		{
			"non-matching confirmation",
			map[string]interface{}{"password": "secret", "password_confirmation": "different"},
			"password",
			"secret",
			true,
		},
		{
			"missing confirmation",
			map[string]interface{}{"password": "secret"},
			"password",
			"secret",
			true,
		},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := rule.Apply(ctx, test.field, test.value, test.data)
			hasError := err != nil
			
			if hasError != test.hasError {
				t.Errorf("Expected hasError=%v, got hasError=%v, error=%v", test.hasError, hasError, err)
			}
		})
	}
}

func TestBetweenRule(t *testing.T) {
	rule := NewBetweenRule("5", "10")
	ctx := context.Background()
	data := make(map[string]interface{})
	
	tests := []struct {
		name     string
		value    interface{}
		hasError bool
	}{
		{"string valid length", "hello", false},
		{"string too short", "hi", true},
		{"string too long", "hello world", true},
		{"int valid", 7, false},
		{"int too small", 3, true},
		{"int too large", 15, true},
		{"float valid", 7.5, false},
		{"float too small", 3.2, true},
		{"float too large", 15.8, true},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := rule.Apply(ctx, "field", test.value, data)
			hasError := err != nil
			
			if hasError != test.hasError {
				t.Errorf("Expected hasError=%v, got hasError=%v, error=%v", test.hasError, hasError, err)
			}
		})
	}
}

func TestUUIDRule(t *testing.T) {
	rule := NewUUIDRule()
	ctx := context.Background()
	data := make(map[string]interface{})
	
	tests := []struct {
		name     string
		value    interface{}
		hasError bool
	}{
		{"valid UUID", "550e8400-e29b-41d4-a716-446655440000", false},
		{"valid UUID uppercase", "550E8400-E29B-41D4-A716-446655440000", false},
		{"invalid UUID", "not-a-uuid", true},
		{"UUID without dashes", "550e8400e29b41d4a716446655440000", true},
		{"empty string", "", true},
		{"non-string", 123, true},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := rule.Apply(ctx, "field", test.value, data)
			hasError := err != nil
			
			if hasError != test.hasError {
				t.Errorf("Expected hasError=%v, got hasError=%v, error=%v", test.hasError, hasError, err)
			}
		})
	}
}

func TestErrorBag(t *testing.T) {
	bag := NewErrorBag()
	
	// Test Add
	bag.Add("name", "Name is required")
	bag.Add("email", "Email is invalid")
	bag.Add("name", "Name is too short")
	
	// Test Count
	if count := bag.Count(); count != 3 {
		t.Errorf("Expected count 3, got %d", count)
	}
	
	// Test Has
	if !bag.Has("name") {
		t.Error("Expected bag to have 'name' errors")
	}
	
	if bag.Has("nonexistent") {
		t.Error("Expected bag not to have 'nonexistent' errors")
	}
	
	// Test Get
	nameErrors := bag.Get("name")
	if len(nameErrors) != 2 {
		t.Errorf("Expected 2 name errors, got %d", len(nameErrors))
	}
	
	// Test First
	firstError := bag.First("name")
	if firstError != "Name is required" {
		t.Errorf("Expected first error 'Name is required', got '%s'", firstError)
	}
	
	// Test All
	all := bag.All()
	if len(all) != 2 {
		t.Errorf("Expected 2 fields with errors, got %d", len(all))
	}
	
	// Test IsEmpty
	if bag.IsEmpty() {
		t.Error("Expected bag not to be empty")
	}
	
	// Test Clear
	bag.Clear()
	if !bag.IsEmpty() {
		t.Error("Expected bag to be empty after clear")
	}
}

func TestResult(t *testing.T) {
	result := NewResult()
	
	// Test initial state
	if !result.IsValid() {
		t.Error("Expected new result to be valid")
	}
	
	if result.HasErrors() {
		t.Error("Expected new result to have no errors")
	}
	
	// Test adding errors
	result.AddError("name", "Name is required")
	
	if result.IsValid() {
		t.Error("Expected result to be invalid after adding error")
	}
	
	if !result.HasErrors() {
		t.Error("Expected result to have errors after adding error")
	}
	
	// Test error retrieval
	if !result.HasError("name") {
		t.Error("Expected result to have 'name' error")
	}
	
	firstError := result.FirstError("name")
	if firstError != "Name is required" {
		t.Errorf("Expected first error 'Name is required', got '%s'", firstError)
	}
	
	// Test JSON serialization
	jsonData, err := result.ToJSON()
	if err != nil {
		t.Errorf("Expected JSON serialization to succeed, got error: %v", err)
	}
	
	if len(jsonData) == 0 {
		t.Error("Expected JSON data to be non-empty")
	}
}

func TestManager(t *testing.T) {
	manager := NewManager()
	
	// Test rule registration
	customRule := func(ctx context.Context, field string, value interface{}, params []string, data map[string]interface{}) error {
		if value == "invalid" {
			return NewValidationError(field, "custom", "Custom validation failed", value, params)
		}
		return nil
	}
	
	err := manager.RegisterRule("custom", customRule)
	if err != nil {
		t.Errorf("Expected rule registration to succeed, got error: %v", err)
	}
	
	// Test getting registered rule
	if _, exists := manager.GetRule("custom"); !exists {
		t.Error("Expected custom rule to be registered")
	}
	
	// Test validator creation
	data := map[string]interface{}{
		"test": "valid",
	}
	
	rules := map[string][]Rule{
		"test": {NewRequiredRule()},
	}
	
	validator := manager.Make(data, rules)
	if validator == nil {
		t.Fatal("Expected validator to be created")
	}
	
	result := validator.Validate(context.Background())
	if !result.IsValid() {
		t.Errorf("Expected validation to pass, got errors: %v", result.GetErrors())
	}
}

func TestValidatorWithRequest(t *testing.T) {
	// Create a mock HTTP request
	form := url.Values{}
	form.Add("name", "John Doe")
	form.Add("email", "john@example.com")
	
	req, err := http.NewRequest("POST", "/test", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	
	// Create validator
	validator := NewValidator(nil, nil)
	validator.WithRequest(req)
	
	// Add rules
	validator.AddRule("name", NewRequiredRule(), NewMinRule("2"))
	validator.AddRule("email", NewRequiredRule(), NewEmailRule())
	
	// Validate
	result := validator.Validate(context.Background())
	if !result.IsValid() {
		t.Errorf("Expected validation to pass, got errors: %v", result.GetErrors())
	}
}

func TestValidatorWithCustomMessages(t *testing.T) {
	data := map[string]interface{}{
		"name": "",
	}
	
	rules := map[string][]Rule{
		"name": {NewRequiredRule()},
	}
	
	customMessages := map[string]string{
		"name.required": "Please provide your name",
	}
	
	validator := NewValidator(data, rules)
	validator.WithMessages(customMessages)
	
	result := validator.Validate(context.Background())
	
	if result.IsValid() {
		t.Error("Expected validation to fail")
	}
	
	firstError := result.FirstError("name")
	if !strings.Contains(firstError, "Please provide your name") {
		t.Errorf("Expected custom message, got: %s", firstError)
	}
}

func TestValidatorSometimes(t *testing.T) {
	data := map[string]interface{}{
		"optional_field": "present",
	}
	
	validator := NewValidator(data, nil)
	validator.Sometimes("optional_field", NewMinRule("10"))
	validator.Sometimes("missing_field", NewRequiredRule())
	
	result := validator.Validate(context.Background())
	
	// Should fail because optional_field is present but too short
	if result.IsValid() {
		t.Error("Expected validation to fail for optional_field")
	}
	
	// Should not fail for missing_field because it's not present
	if result.HasError("missing_field") {
		t.Error("Expected no error for missing_field")
	}
}

func TestValidatorStopOnFirstFailure(t *testing.T) {
	data := map[string]interface{}{
		"field1": "",
		"field2": "",
	}
	
	rules := map[string][]Rule{
		"field1": {NewRequiredRule()},
		"field2": {NewRequiredRule()},
	}
	
	validator := NewValidator(data, rules)
	validator.StopOnFirstFailure()
	
	result := validator.Validate(context.Background())
	
	if result.IsValid() {
		t.Error("Expected validation to fail")
	}
	
	// Should only have one error due to StopOnFirstFailure
	errorCount := len(result.AllErrors())
	if errorCount > 1 {
		t.Errorf("Expected at most 1 error due to StopOnFirstFailure, got %d", errorCount)
	}
}

func TestStructToMap(t *testing.T) {
	type TestStruct struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Age   int    `json:"age"`
	}
	
	s := TestStruct{
		Name:  "John Doe",
		Email: "john@example.com",
		Age:   30,
	}
	
	data, err := structToMap(s)
	if err != nil {
		t.Fatalf("Expected struct conversion to succeed, got error: %v", err)
	}
	
	if data["name"] != "John Doe" {
		t.Errorf("Expected name 'John Doe', got '%v'", data["name"])
	}
	
	if data["email"] != "john@example.com" {
		t.Errorf("Expected email 'john@example.com', got '%v'", data["email"])
	}
	
	if data["age"] != 30 {
		t.Errorf("Expected age 30, got %v", data["age"])
	}
}

func TestIsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected bool
	}{
		{"nil", nil, true},
		{"empty string", "", true},
		{"whitespace string", "   ", true},
		{"non-empty string", "hello", false},
		{"empty slice", []interface{}{}, true},
		{"non-empty slice", []interface{}{1}, false},
		{"empty map", map[string]interface{}{}, true},
		{"non-empty map", map[string]interface{}{"key": "value"}, false},
		{"zero int", 0, false},
		{"non-zero int", 5, false},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := isEmpty(test.value)
			if result != test.expected {
				t.Errorf("Expected isEmpty(%v) = %v, got %v", test.value, test.expected, result)
			}
		})
	}
}

func TestRegexRule(t *testing.T) {
	// Test valid regex
	rule, err := NewRegexRule(`^\d{3}-\d{3}-\d{4}$`)
	if err != nil {
		t.Fatalf("Expected valid regex to create rule, got error: %v", err)
	}
	
	ctx := context.Background()
	data := make(map[string]interface{})
	
	tests := []struct {
		name     string
		value    interface{}
		hasError bool
	}{
		{"valid phone", "123-456-7890", false},
		{"invalid phone", "123-45-6789", true},
		{"non-string", 123, true},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := rule.Apply(ctx, "field", test.value, data)
			hasError := err != nil
			
			if hasError != test.hasError {
				t.Errorf("Expected hasError=%v, got hasError=%v, error=%v", test.hasError, hasError, err)
			}
		})
	}
	
	// Test invalid regex
	_, err = NewRegexRule(`[invalid`)
	if err == nil {
		t.Error("Expected invalid regex to return error")
	}
}

func TestValidationErrors(t *testing.T) {
	errors := ValidationErrors{
		NewValidationError("name", "required", "Name is required", nil, nil),
		NewValidationError("email", "email", "Email is invalid", "invalid-email", nil),
	}
	
	// Test Error method
	errorMsg := errors.Error()
	if !strings.Contains(errorMsg, "Name is required") {
		t.Error("Expected error message to contain 'Name is required'")
	}
	
	// Test ToMap
	errorMap := errors.ToMap()
	if len(errorMap) != 2 {
		t.Errorf("Expected 2 fields in error map, got %d", len(errorMap))
	}
	
	// Test First
	firstError := errors.First("name")
	if firstError != "Name is required" {
		t.Errorf("Expected first error 'Name is required', got '%s'", firstError)
	}
	
	// Test Has
	if !errors.Has("email") {
		t.Error("Expected errors to have 'email' field")
	}
	
	// Test Count
	if errors.Count() != 2 {
		t.Errorf("Expected count 2, got %d", errors.Count())
	}
	
	// Test IsEmpty
	if errors.IsEmpty() {
		t.Error("Expected errors not to be empty")
	}
}

func BenchmarkValidation(b *testing.B) {
	data := map[string]interface{}{
		"name":  "John Doe",
		"email": "john@example.com",
		"age":   30,
	}
	
	rules := map[string][]Rule{
		"name":  {NewRequiredRule(), NewMinRule("2"), NewMaxRule("50")},
		"email": {NewRequiredRule(), NewEmailRule()},
		"age":   {NewRequiredRule(), NewNumericRule(), NewMinRule("18")},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator := NewValidator(data, rules)
		validator.Validate(context.Background())
	}
}

func BenchmarkErrorBag(b *testing.B) {
	bag := NewErrorBag()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bag.Add("field", "error message")
		bag.Has("field")
		bag.Get("field")
		bag.Clear()
	}
}