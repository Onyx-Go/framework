package validation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"
)

// DefaultValidator implements the Validator interface
type DefaultValidator struct {
	data            map[string]interface{}
	rules           map[string][]Rule
	conditionalRules map[string][]ConditionalRule
	customMessages  map[string]string
	customAttributes map[string]string
	options         *Options
	ruleRegistry    map[string]Rule
	transformers    map[string]FieldTransformer
	eventListeners  []EventListener
	mutex           sync.RWMutex
}

// ConditionalRule represents a rule with conditions
type ConditionalRule struct {
	Rule      Rule
	Condition func(data map[string]interface{}) bool
	Required  bool
}

// NewValidator creates a new validator instance
func NewValidator(data map[string]interface{}, rules map[string][]Rule) *DefaultValidator {
	return &DefaultValidator{
		data:             data,
		rules:            rules,
		conditionalRules: make(map[string][]ConditionalRule),
		customMessages:   make(map[string]string),
		customAttributes: make(map[string]string),
		options:          DefaultOptions(),
		ruleRegistry:     getDefaultRules(),
		transformers:     make(map[string]FieldTransformer),
		eventListeners:   make([]EventListener, 0),
	}
}

// Validate validates all fields with their rules
func (v *DefaultValidator) Validate(ctx context.Context) Result {
	v.mutex.RLock()
	defer v.mutex.RUnlock()
	
	result := NewResult()
	
	// Notify listeners
	for _, listener := range v.eventListeners {
		listener.OnValidationStart(v.data)
	}
	
	// Transform data if transformers are registered
	transformedData := v.transformData()
	
	// Validate regular rules
	for field, fieldRules := range v.rules {
		if ctx.Err() != nil {
			result.AddError("validation", "validation cancelled")
			break
		}
		
		value := v.getValue(field, transformedData)
		fieldResult := v.validateField(ctx, field, value, fieldRules, transformedData)
		result.MergeErrors(fieldResult)
		
		if v.options.StopOnFirstFailure && fieldResult.HasErrors() {
			break
		}
	}
	
	// Validate conditional rules
	for field, conditionalRules := range v.conditionalRules {
		if ctx.Err() != nil {
			break
		}
		
		value := v.getValue(field, transformedData)
		for _, condRule := range conditionalRules {
			if condRule.Condition(transformedData) {
				if err := condRule.Rule.Apply(ctx, field, value, transformedData); err != nil {
					message := v.formatErrorMessage(field, condRule.Rule.GetName(), err.Error(), condRule.Rule.GetParameters())
					result.AddError(field, message)
					
					if v.options.StopOnFirstFailure {
						break
					}
				}
			}
		}
	}
	
	// Notify listeners
	for _, listener := range v.eventListeners {
		listener.OnValidationEnd(result)
	}
	
	return result
}

// ValidateStruct validates a struct with validation tags
func (v *DefaultValidator) ValidateStruct(ctx context.Context, s interface{}) Result {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	
	// Convert struct to map
	structData, err := structToMap(s)
	if err != nil {
		result := NewResult()
		result.AddError("struct", fmt.Sprintf("failed to parse struct: %v", err))
		return result
	}
	
	// Parse validation tags
	structRules, err := v.parseStructTags(reflect.TypeOf(s))
	if err != nil {
		result := NewResult()
		result.AddError("struct", fmt.Sprintf("failed to parse validation tags: %v", err))
		return result
	}
	
	// Update validator data and rules
	oldData := v.data
	oldRules := v.rules
	
	v.data = structData
	v.rules = structRules
	
	// Validate
	result := v.Validate(ctx)
	
	// Restore original data and rules
	v.data = oldData
	v.rules = oldRules
	
	return result
}

// ValidateField validates a single field
func (v *DefaultValidator) ValidateField(ctx context.Context, field string, value interface{}) Result {
	v.mutex.RLock()
	defer v.mutex.RUnlock()
	
	result := NewResult()
	
	if fieldRules, exists := v.rules[field]; exists {
		fieldResult := v.validateField(ctx, field, value, fieldRules, v.data)
		result.MergeErrors(fieldResult)
	}
	
	return result
}

// validateField validates a single field with its rules
func (v *DefaultValidator) validateField(ctx context.Context, field string, value interface{}, rules []Rule, data map[string]interface{}) Result {
	result := NewResult()
	
	// Notify listeners
	for _, listener := range v.eventListeners {
		listener.OnFieldValidation(field, value)
	}
	
	for _, rule := range rules {
		if ctx.Err() != nil {
			result.AddError("validation", "validation cancelled")
			break
		}
		
		// Skip non-implicit rules if value is empty
		if !rule.IsImplicit() && isEmpty(value) {
			continue
		}
		
		err := rule.Apply(ctx, field, value, data)
		
		// Notify listeners
		for _, listener := range v.eventListeners {
			listener.OnRuleValidation(field, rule.GetName(), value, err)
		}
		
		if err != nil {
			message := v.formatErrorMessage(field, rule.GetName(), err.Error(), rule.GetParameters())
			result.AddError(field, message)
			
			if v.options.StopOnFirstFailure {
				break
			}
		}
	}
	
	return result
}

// AddRule adds validation rules for a field
func (v *DefaultValidator) AddRule(field string, rules ...Rule) Validator {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	
	if v.rules == nil {
		v.rules = make(map[string][]Rule)
	}
	
	v.rules[field] = append(v.rules[field], rules...)
	return v
}

// AddRules adds multiple validation rules
func (v *DefaultValidator) AddRules(rules map[string][]Rule) Validator {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	
	if v.rules == nil {
		v.rules = make(map[string][]Rule)
	}
	
	for field, fieldRules := range rules {
		v.rules[field] = append(v.rules[field], fieldRules...)
	}
	
	return v
}

// WithCustomRule adds a custom validation rule
func (v *DefaultValidator) WithCustomRule(name string, rule RuleFunc) Validator {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	
	v.ruleRegistry[name] = &FuncRule{
		name: name,
		fn:   rule,
	}
	
	return v
}

// WithData sets the validation data
func (v *DefaultValidator) WithData(data map[string]interface{}) Validator {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	
	v.data = data
	return v
}

// WithRequest sets data from HTTP request
func (v *DefaultValidator) WithRequest(r *http.Request) Validator {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	
	data := make(map[string]interface{})
	
	// Parse form data
	if err := r.ParseForm(); err == nil {
		for key, values := range r.Form {
			if len(values) == 1 {
				data[key] = values[0]
			} else {
				data[key] = values
			}
		}
	}
	
	// Parse multipart form
	if err := r.ParseMultipartForm(32 << 20); err == nil {
		if r.MultipartForm != nil {
			for key, values := range r.MultipartForm.Value {
				if len(values) == 1 {
					data[key] = values[0]
				} else {
					data[key] = values
				}
			}
		}
	}
	
	v.data = data
	return v
}

// WithJSON sets data from JSON
func (v *DefaultValidator) WithJSON(jsonData []byte) Validator {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	
	var data map[string]interface{}
	if err := json.Unmarshal(jsonData, &data); err == nil {
		v.data = data
	}
	
	return v
}

// Sometimes adds conditional validation rules
func (v *DefaultValidator) Sometimes(field string, rules ...Rule) Validator {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	
	if v.conditionalRules == nil {
		v.conditionalRules = make(map[string][]ConditionalRule)
	}
	
	condition := func(data map[string]interface{}) bool {
		value := v.getValue(field, data)
		return !isEmpty(value)
	}
	
	for _, rule := range rules {
		condRule := ConditionalRule{
			Rule:      rule,
			Condition: condition,
			Required:  false,
		}
		v.conditionalRules[field] = append(v.conditionalRules[field], condRule)
	}
	
	return v
}

// When adds conditional validation
func (v *DefaultValidator) When(condition func() bool, callback func(v Validator)) Validator {
	if condition() {
		callback(v)
	}
	return v
}

// Unless adds inverse conditional validation
func (v *DefaultValidator) Unless(condition func() bool, callback func(v Validator)) Validator {
	if !condition() {
		callback(v)
	}
	return v
}

// WithMessages sets custom error messages
func (v *DefaultValidator) WithMessages(messages map[string]string) Validator {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	
	if v.customMessages == nil {
		v.customMessages = make(map[string]string)
	}
	
	for key, message := range messages {
		v.customMessages[key] = message
	}
	
	return v
}

// WithAttribute sets custom field attributes
func (v *DefaultValidator) WithAttribute(field, attribute string) Validator {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	
	if v.customAttributes == nil {
		v.customAttributes = make(map[string]string)
	}
	
	v.customAttributes[field] = attribute
	return v
}

// WithOptions sets validation options
func (v *DefaultValidator) WithOptions(opts *Options) Validator {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	
	v.options = opts
	return v
}

// StopOnFirstFailure configures validator to stop on first failure
func (v *DefaultValidator) StopOnFirstFailure() Validator {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	
	v.options.StopOnFirstFailure = true
	return v
}

// Bail is an alias for StopOnFirstFailure
func (v *DefaultValidator) Bail() Validator {
	return v.StopOnFirstFailure()
}

// Helper methods

func (v *DefaultValidator) getValue(field string, data map[string]interface{}) interface{} {
	if value, exists := data[field]; exists {
		return value
	}
	
	// Support nested field access with dot notation
	if strings.Contains(field, ".") {
		return v.getNestedValue(field, data)
	}
	
	return nil
}

func (v *DefaultValidator) getNestedValue(field string, data map[string]interface{}) interface{} {
	parts := strings.Split(field, ".")
	current := data
	
	for i, part := range parts {
		if i == len(parts)-1 {
			// Last part, return the value
			if value, exists := current[part]; exists {
				return value
			}
			return nil
		}
		
		// Intermediate part, navigate deeper
		if value, exists := current[part]; exists {
			if nextMap, ok := value.(map[string]interface{}); ok {
				current = nextMap
			} else {
				return nil
			}
		} else {
			return nil
		}
	}
	
	return nil
}

func (v *DefaultValidator) transformData() map[string]interface{} {
	if len(v.transformers) == 0 {
		return v.data
	}
	
	transformed := make(map[string]interface{})
	for key, value := range v.data {
		if transformer, exists := v.transformers[key]; exists && transformer.ShouldTransform(value) {
			transformed[key] = transformer.Transform(value)
		} else {
			transformed[key] = value
		}
	}
	
	return transformed
}

func (v *DefaultValidator) formatErrorMessage(field, ruleName, defaultMessage string, params []string) string {
	// Check for custom message
	messageKey := fmt.Sprintf("%s.%s", field, ruleName)
	if message, exists := v.customMessages[messageKey]; exists {
		return v.interpolateMessage(message, field, params)
	}
	
	// Check for rule-specific message
	if message, exists := v.customMessages[ruleName]; exists {
		return v.interpolateMessage(message, field, params)
	}
	
	// Use field attribute if available
	fieldName := field
	if attribute, exists := v.customAttributes[field]; exists {
		fieldName = attribute
	}
	
	// Replace field name in default message
	return strings.ReplaceAll(defaultMessage, field, fieldName)
}

func (v *DefaultValidator) interpolateMessage(message, field string, params []string) string {
	result := strings.ReplaceAll(message, ":field", field)
	result = strings.ReplaceAll(result, ":attribute", field)
	
	for i, param := range params {
		placeholder := fmt.Sprintf(":param%d", i)
		result = strings.ReplaceAll(result, placeholder, param)
	}
	
	return result
}

func (v *DefaultValidator) parseStructTags(t reflect.Type) (map[string][]Rule, error) {
	rules := make(map[string][]Rule)
	
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	
	if t.Kind() != reflect.Struct {
		return rules, fmt.Errorf("expected struct, got %s", t.Kind())
	}
	
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		
		// Skip unexported fields
		if !field.IsExported() {
			continue
		}
		
		// Get validation tag
		tag := field.Tag.Get("validate")
		if tag == "" {
			continue
		}
		
		// Parse tag into rules
		fieldRules, err := v.parseTagRules(tag)
		if err != nil {
			return nil, fmt.Errorf("error parsing tag for field %s: %v", field.Name, err)
		}
		
		// Use json tag for field name if available
		fieldName := field.Name
		if jsonTag := field.Tag.Get("json"); jsonTag != "" {
			if parts := strings.Split(jsonTag, ","); parts[0] != "" {
				fieldName = parts[0]
			}
		}
		
		rules[fieldName] = fieldRules
	}
	
	return rules, nil
}

func (v *DefaultValidator) parseTagRules(tag string) ([]Rule, error) {
	var rules []Rule
	
	// Split by pipe for multiple rules
	ruleStrings := strings.Split(tag, "|")
	
	for _, ruleStr := range ruleStrings {
		ruleStr = strings.TrimSpace(ruleStr)
		if ruleStr == "" {
			continue
		}
		
		// Parse rule name and parameters
		parts := strings.Split(ruleStr, ":")
		ruleName := parts[0]
		var params []string
		
		if len(parts) > 1 {
			params = strings.Split(parts[1], ",")
			for i, param := range params {
				params[i] = strings.TrimSpace(param)
			}
		}
		
		// Create rule
		rule, err := v.createRuleFromTag(ruleName, params)
		if err != nil {
			return nil, err
		}
		
		rules = append(rules, rule)
	}
	
	return rules, nil
}

func (v *DefaultValidator) createRuleFromTag(ruleName string, params []string) (Rule, error) {
	switch ruleName {
	case "required":
		return NewRequiredRule(), nil
	case "min":
		if len(params) == 0 {
			return nil, fmt.Errorf("min rule requires a parameter")
		}
		return NewMinRule(params[0]), nil
	case "max":
		if len(params) == 0 {
			return nil, fmt.Errorf("max rule requires a parameter")
		}
		return NewMaxRule(params[0]), nil
	case "email":
		return NewEmailRule(), nil
	case "numeric":
		return NewNumericRule(), nil
	case "alpha":
		return NewAlphaRule(), nil
	case "alpha_num":
		return NewAlphaNumRule(), nil
	case "url":
		return NewURLRule(), nil
	case "in":
		if len(params) == 0 {
			return nil, fmt.Errorf("in rule requires parameters")
		}
		return NewInRule(params), nil
	default:
		// Check custom rules
		if rule, exists := v.ruleRegistry[ruleName]; exists {
			return rule, nil
		}
		return nil, fmt.Errorf("unknown validation rule: %s", ruleName)
	}
}

// Utility functions

func isEmpty(value interface{}) bool {
	if value == nil {
		return true
	}
	
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v) == ""
	case []interface{}:
		return len(v) == 0
	case map[string]interface{}:
		return len(v) == 0
	default:
		rv := reflect.ValueOf(value)
		switch rv.Kind() {
		case reflect.Array, reflect.Slice, reflect.Map, reflect.Chan:
			return rv.Len() == 0
		case reflect.Ptr, reflect.Interface:
			return rv.IsNil()
		}
	}
	
	return false
}

func structToMap(s interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	
	v := reflect.ValueOf(s)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct, got %s", v.Kind())
	}
	
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)
		
		if !field.IsExported() {
			continue
		}
		
		// Use json tag for field name if available
		fieldName := field.Name
		if jsonTag := field.Tag.Get("json"); jsonTag != "" {
			if parts := strings.Split(jsonTag, ","); parts[0] != "" {
				fieldName = parts[0]
			}
		}
		
		result[fieldName] = fieldValue.Interface()
	}
	
	return result, nil
}

func getDefaultRules() map[string]Rule {
	return map[string]Rule{
		"required":  NewRequiredRule(),
		"min":       NewMinRule("0"),
		"max":       NewMaxRule("255"),
		"email":     NewEmailRule(),
		"numeric":   NewNumericRule(),
		"alpha":     NewAlphaRule(),
		"alpha_num": NewAlphaNumRule(),
		"url":       NewURLRule(),
		"in":        NewInRule([]string{}),
	}
}