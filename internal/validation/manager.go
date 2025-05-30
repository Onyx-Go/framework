package validation

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"
)

// DefaultManager implements the Manager interface
type DefaultManager struct {
	rules            map[string]Rule
	customMessages   map[string]string
	customAttributes map[string]string
	defaultOptions   *Options
	messageProvider  MessageProvider
	structValidator  StructValidator
	requestValidator RequestValidator
	transformer      Transformer
	eventListeners   []EventListener
	mutex            sync.RWMutex
}

// NewManager creates a new validation manager
func NewManager() *DefaultManager {
	manager := &DefaultManager{
		rules:            make(map[string]Rule),
		customMessages:   make(map[string]string),
		customAttributes: make(map[string]string),
		defaultOptions:   DefaultOptions(),
		eventListeners:   make([]EventListener, 0),
	}
	
	// Register default rules
	manager.registerDefaultRules()
	
	// Initialize components
	manager.messageProvider = NewDefaultMessageProvider()
	manager.structValidator = NewDefaultStructValidator(manager)
	manager.requestValidator = NewDefaultRequestValidator(manager)
	manager.transformer = NewDefaultTransformer()
	
	return manager
}

// Make creates a new validator instance
func (m *DefaultManager) Make(data map[string]interface{}, rules map[string][]Rule) Validator {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	validator := NewValidator(data, rules)
	validator.WithOptions(m.defaultOptions)
	validator.WithMessages(m.customMessages)
	
	for field, attr := range m.customAttributes {
		validator.WithAttribute(field, attr)
	}
	
	return validator
}

// MakeRequest creates a validator from an HTTP request
func (m *DefaultManager) MakeRequest(r *http.Request, rules map[string][]Rule) Validator {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	validator := NewValidator(nil, rules)
	validator.WithRequest(r)
	validator.WithOptions(m.defaultOptions)
	validator.WithMessages(m.customMessages)
	
	for field, attr := range m.customAttributes {
		validator.WithAttribute(field, attr)
	}
	
	return validator
}

// MakeStruct creates a validator for struct validation
func (m *DefaultManager) MakeStruct(s interface{}) Validator {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	// Convert struct to map
	data, err := structToMap(s)
	if err != nil {
		// Return validator with error
		validator := NewValidator(nil, nil)
		result := NewResult()
		result.AddError("struct", fmt.Sprintf("failed to parse struct: %v", err))
		return validator
	}
	
	// Parse struct tags for validation rules
	structType := reflect.ValueOf(s).Type()
	if structType.Kind() == reflect.Ptr {
		structType = structType.Elem()
	}
	
	rulesMap := make(map[string][]Rule)
	if structType.Kind() == reflect.Struct {
		for i := 0; i < structType.NumField(); i++ {
			field := structType.Field(i)
			if !field.IsExported() {
				continue
			}
			
			// Get field name (use json tag if available)
			fieldName := field.Name
			if jsonTag := field.Tag.Get("json"); jsonTag != "" {
				if parts := strings.Split(jsonTag, ","); parts[0] != "" && parts[0] != "-" {
					fieldName = parts[0]
				}
			}
			
			// Parse validation tag
			tag := field.Tag.Get("validate")
			if tag == "" {
				continue
			}
			
			fieldRules, err := parseValidationTag(tag)
			if err != nil {
				// Return validator with error
				validator := NewValidator(data, nil)
				return validator
			}
			
			rulesMap[fieldName] = fieldRules
		}
	}
	
	validator := NewValidator(data, rulesMap)
	validator.WithOptions(m.defaultOptions)
	validator.WithMessages(m.customMessages)
	
	for field, attr := range m.customAttributes {
		validator.WithAttribute(field, attr)
	}
	
	return validator
}

// RegisterRule registers a custom validation rule function
func (m *DefaultManager) RegisterRule(name string, rule RuleFunc) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if name == "" {
		return fmt.Errorf("rule name cannot be empty")
	}
	
	m.rules[name] = &FuncRule{
		name: name,
		fn:   rule,
	}
	
	return nil
}

// RegisterCustomRule registers a custom validation rule
func (m *DefaultManager) RegisterCustomRule(name string, rule Rule) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if name == "" {
		return fmt.Errorf("rule name cannot be empty")
	}
	
	m.rules[name] = rule
	return nil
}

// UnregisterRule removes a validation rule
func (m *DefaultManager) UnregisterRule(name string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	delete(m.rules, name)
}

// GetRule returns a validation rule by name
func (m *DefaultManager) GetRule(name string) (Rule, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	rule, exists := m.rules[name]
	return rule, exists
}

// GetRules returns all registered rules
func (m *DefaultManager) GetRules() map[string]Rule {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	// Return a copy to prevent external modification
	rules := make(map[string]Rule)
	for name, rule := range m.rules {
		rules[name] = rule
	}
	
	return rules
}

// SetMessages sets custom error messages
func (m *DefaultManager) SetMessages(messages map[string]string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if m.customMessages == nil {
		m.customMessages = make(map[string]string)
	}
	
	for key, message := range messages {
		m.customMessages[key] = message
	}
}

// SetAttributes sets custom field attributes
func (m *DefaultManager) SetAttributes(attributes map[string]string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if m.customAttributes == nil {
		m.customAttributes = make(map[string]string)
	}
	
	for field, attribute := range attributes {
		m.customAttributes[field] = attribute
	}
}

// GetMessage returns a custom message for a rule and field
func (m *DefaultManager) GetMessage(rule, field string) string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	// Check field-specific message
	if message, exists := m.customMessages[field+"."+rule]; exists {
		return message
	}
	
	// Check rule-specific message
	if message, exists := m.customMessages[rule]; exists {
		return message
	}
	
	// Use message provider
	return m.messageProvider.GetMessage(rule, field, nil)
}

// SetDefaultOptions sets default validation options
func (m *DefaultManager) SetDefaultOptions(opts *Options) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.defaultOptions = opts
}

// GetDefaultOptions returns default validation options
func (m *DefaultManager) GetDefaultOptions() *Options {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	return m.defaultOptions
}

// Middleware returns HTTP middleware for validation
func (m *DefaultManager) Middleware(rules map[string][]Rule) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create validator for request
			validator := m.MakeRequest(r, rules)
			
			// Perform validation
			result := validator.Validate(r.Context())
			
			if result.HasErrors() {
				// Handle validation errors
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				
				if jsonData, err := result.ToJSON(); err == nil {
					w.Write(jsonData)
				} else {
					w.Write([]byte(`{"error": "validation failed"}`))
				}
				return
			}
			
			// Validation passed, continue
			next.ServeHTTP(w, r)
		})
	}
}

// ValidateRequest validates an HTTP request
func (m *DefaultManager) ValidateRequest(ctx context.Context, r *http.Request, rules map[string][]Rule) Result {
	validator := m.MakeRequest(r, rules)
	return validator.Validate(ctx)
}

// ValidateStruct validates a struct
func (m *DefaultManager) ValidateStruct(ctx context.Context, s interface{}) Result {
	validator := m.MakeStruct(s)
	return validator.ValidateStruct(ctx, s)
}

// ValidateData validates data with rules
func (m *DefaultManager) ValidateData(ctx context.Context, data map[string]interface{}, rules map[string][]Rule) Result {
	validator := m.Make(data, rules)
	return validator.Validate(ctx)
}

// AddEventListener adds a validation event listener
func (m *DefaultManager) AddEventListener(listener EventListener) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.eventListeners = append(m.eventListeners, listener)
}

// RemoveEventListener removes a validation event listener
func (m *DefaultManager) RemoveEventListener(listener EventListener) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	for i, l := range m.eventListeners {
		// Compare pointers (this is a simple implementation)
		if fmt.Sprintf("%p", l) == fmt.Sprintf("%p", listener) {
			m.eventListeners = append(m.eventListeners[:i], m.eventListeners[i+1:]...)
			break
		}
	}
}

// GetEventListeners returns all event listeners
func (m *DefaultManager) GetEventListeners() []EventListener {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	// Return a copy
	listeners := make([]EventListener, len(m.eventListeners))
	copy(listeners, m.eventListeners)
	return listeners
}

// Private methods

func (m *DefaultManager) registerDefaultRules() {
	// Register built-in validation rules
	m.rules["required"] = NewRequiredRule()
	m.rules["min"] = NewMinRule("0")
	m.rules["max"] = NewMaxRule("255")
	m.rules["email"] = NewEmailRule()
	m.rules["numeric"] = NewNumericRule()
	m.rules["alpha"] = NewAlphaRule()
	m.rules["alpha_num"] = NewAlphaNumRule()
	m.rules["url"] = NewURLRule()
	m.rules["in"] = NewInRule([]string{})
	m.rules["confirmed"] = NewConfirmedRule()
	m.rules["between"] = NewBetweenRule("0", "100")
	m.rules["size"] = NewSizeRule("0")
	m.rules["uuid"] = NewUUIDRule()
	m.rules["date"] = NewDateRule("")
}

// Default implementations for embedded interfaces

// DefaultStructValidator implements StructValidator
type DefaultStructValidator struct {
	manager *DefaultManager
	tagName string
	mutex   sync.RWMutex
}

// NewDefaultStructValidator creates a new struct validator
func NewDefaultStructValidator(manager *DefaultManager) *DefaultStructValidator {
	return &DefaultStructValidator{
		manager: manager,
		tagName: "validate",
	}
}

// ValidateStruct validates a struct with validation tags
func (sv *DefaultStructValidator) ValidateStruct(ctx context.Context, s interface{}) Result {
	return sv.manager.ValidateStruct(ctx, s)
}

// ParseTags parses validation tags from struct type
func (sv *DefaultStructValidator) ParseTags(structType reflect.Type) (map[string][]Rule, error) {
	sv.mutex.RLock()
	defer sv.mutex.RUnlock()
	
	rules := make(map[string][]Rule)
	
	if structType.Kind() == reflect.Ptr {
		structType = structType.Elem()
	}
	
	if structType.Kind() != reflect.Struct {
		return rules, fmt.Errorf("expected struct, got %s", structType.Kind())
	}
	
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		if !field.IsExported() {
			continue
		}
		
		tag := field.Tag.Get(sv.tagName)
		if tag == "" {
			continue
		}
		
		fieldRules, err := sv.parseTagString(tag)
		if err != nil {
			return nil, fmt.Errorf("error parsing tag for field %s: %v", field.Name, err)
		}
		
		// Use json tag for field name if available
		fieldName := field.Name
		if jsonTag := field.Tag.Get("json"); jsonTag != "" {
			if parts := strings.Split(jsonTag, ","); parts[0] != "" && parts[0] != "-" {
				fieldName = parts[0]
			}
		}
		
		rules[fieldName] = fieldRules
	}
	
	return rules, nil
}

// SetTagName sets the validation tag name
func (sv *DefaultStructValidator) SetTagName(tagName string) {
	sv.mutex.Lock()
	defer sv.mutex.Unlock()
	
	sv.tagName = tagName
}

// GetTagName returns the validation tag name
func (sv *DefaultStructValidator) GetTagName() string {
	sv.mutex.RLock()
	defer sv.mutex.RUnlock()
	
	return sv.tagName
}

// RegisterTagHandler registers a custom tag handler
func (sv *DefaultStructValidator) RegisterTagHandler(tag string, handler TagHandler) {
	// Implementation for custom tag handlers
	// This would be extended in a full implementation
}

// GetTagHandler returns a tag handler
func (sv *DefaultStructValidator) GetTagHandler(tag string) (TagHandler, bool) {
	// Implementation for custom tag handlers
	// This would be extended in a full implementation
	return nil, false
}

func (sv *DefaultStructValidator) parseTagString(tag string) ([]Rule, error) {
	return parseValidationTag(tag)
}

// DefaultRequestValidator implements RequestValidator
type DefaultRequestValidator struct {
	manager *DefaultManager
}

// NewDefaultRequestValidator creates a new request validator
func NewDefaultRequestValidator(manager *DefaultManager) *DefaultRequestValidator {
	return &DefaultRequestValidator{
		manager: manager,
	}
}

// ValidateRequest validates an HTTP request
func (rv *DefaultRequestValidator) ValidateRequest(ctx context.Context, r *http.Request, rules map[string][]Rule) Result {
	return rv.manager.ValidateRequest(ctx, r, rules)
}

// ValidateJSON validates JSON data
func (rv *DefaultRequestValidator) ValidateJSON(ctx context.Context, data []byte, rules map[string][]Rule) Result {
	validator := NewValidator(nil, rules)
	validator.WithJSON(data)
	return validator.Validate(ctx)
}

// ValidateQuery validates query parameters
func (rv *DefaultRequestValidator) ValidateQuery(ctx context.Context, r *http.Request, rules map[string][]Rule) Result {
	data := make(map[string]interface{})
	for key, values := range r.URL.Query() {
		if len(values) == 1 {
			data[key] = values[0]
		} else {
			data[key] = values
		}
	}
	
	validator := NewValidator(data, rules)
	return validator.Validate(ctx)
}

// ValidateForm validates form data
func (rv *DefaultRequestValidator) ValidateForm(ctx context.Context, r *http.Request, rules map[string][]Rule) Result {
	return rv.manager.ValidateRequest(ctx, r, rules)
}

// ValidateFiles validates uploaded files
func (rv *DefaultRequestValidator) ValidateFiles(ctx context.Context, r *http.Request, rules map[string][]Rule) Result {
	// Implementation for file validation
	// This would be extended in a full implementation
	return NewResult()
}

// ValidateHeaders validates request headers
func (rv *DefaultRequestValidator) ValidateHeaders(ctx context.Context, r *http.Request, rules map[string][]Rule) Result {
	data := make(map[string]interface{})
	for key, values := range r.Header {
		if len(values) == 1 {
			data[key] = values[0]
		} else {
			data[key] = values
		}
	}
	
	validator := NewValidator(data, rules)
	return validator.Validate(ctx)
}

// DefaultTransformer implements Transformer
type DefaultTransformer struct {
	transformers map[string]FieldTransformer
	mutex        sync.RWMutex
}

// NewDefaultTransformer creates a new transformer
func NewDefaultTransformer() *DefaultTransformer {
	return &DefaultTransformer{
		transformers: make(map[string]FieldTransformer),
	}
}

// Transform transforms data before validation
func (t *DefaultTransformer) Transform(data map[string]interface{}) map[string]interface{} {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	
	if len(t.transformers) == 0 {
		return data
	}
	
	transformed := make(map[string]interface{})
	for key, value := range data {
		if transformer, exists := t.transformers[key]; exists && transformer.ShouldTransform(value) {
			transformed[key] = transformer.Transform(value)
		} else {
			transformed[key] = value
		}
	}
	
	return transformed
}

// TransformField transforms a single field
func (t *DefaultTransformer) TransformField(field string, value interface{}) interface{} {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	
	if transformer, exists := t.transformers[field]; exists && transformer.ShouldTransform(value) {
		return transformer.Transform(value)
	}
	
	return value
}

// RegisterTransformer registers a field transformer
func (t *DefaultTransformer) RegisterTransformer(field string, transformer FieldTransformer) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	
	t.transformers[field] = transformer
}

// GetTransformer returns a field transformer
func (t *DefaultTransformer) GetTransformer(field string) (FieldTransformer, bool) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	
	transformer, exists := t.transformers[field]
	return transformer, exists
}

// DefaultMessageProvider implements MessageProvider
type DefaultMessageProvider struct {
	messages   map[string]string
	attributes map[string]string
	locale     string
	mutex      sync.RWMutex
}

// NewDefaultMessageProvider creates a new message provider
func NewDefaultMessageProvider() *DefaultMessageProvider {
	provider := &DefaultMessageProvider{
		messages:   make(map[string]string),
		attributes: make(map[string]string),
		locale:     "en",
	}
	
	provider.loadDefaultMessages()
	return provider
}

// GetMessage returns a message for a rule and field
func (mp *DefaultMessageProvider) GetMessage(rule, field string, params []string) string {
	mp.mutex.RLock()
	defer mp.mutex.RUnlock()
	
	// Check for custom message
	if message, exists := mp.messages[field+"."+rule]; exists {
		return message
	}
	
	if message, exists := mp.messages[rule]; exists {
		return message
	}
	
	// Return default message
	return fmt.Sprintf("The %s field is invalid", field)
}

// GetAttribute returns an attribute name for a field
func (mp *DefaultMessageProvider) GetAttribute(field string) string {
	mp.mutex.RLock()
	defer mp.mutex.RUnlock()
	
	if attribute, exists := mp.attributes[field]; exists {
		return attribute
	}
	
	return field
}

// SetMessages sets custom messages
func (mp *DefaultMessageProvider) SetMessages(messages map[string]string) {
	mp.mutex.Lock()
	defer mp.mutex.Unlock()
	
	for key, message := range messages {
		mp.messages[key] = message
	}
}

// SetAttributes sets custom attributes
func (mp *DefaultMessageProvider) SetAttributes(attributes map[string]string) {
	mp.mutex.Lock()
	defer mp.mutex.Unlock()
	
	for field, attribute := range attributes {
		mp.attributes[field] = attribute
	}
}

// SetLocale sets the locale
func (mp *DefaultMessageProvider) SetLocale(locale string) {
	mp.mutex.Lock()
	defer mp.mutex.Unlock()
	
	mp.locale = locale
}

// GetLocale returns the current locale
func (mp *DefaultMessageProvider) GetLocale() string {
	mp.mutex.RLock()
	defer mp.mutex.RUnlock()
	
	return mp.locale
}

func (mp *DefaultMessageProvider) loadDefaultMessages() {
	mp.messages = map[string]string{
		"required":  "The :field field is required",
		"min":       "The :field field must be at least :param0",
		"max":       "The :field field may not be greater than :param0",
		"email":     "The :field field must be a valid email address",
		"numeric":   "The :field field must be numeric",
		"alpha":     "The :field field may only contain letters",
		"alpha_num": "The :field field may only contain letters and numbers",
		"url":       "The :field field must be a valid URL",
		"in":        "The :field field must be one of the allowed values",
		"confirmed": "The :field confirmation does not match",
		"between":   "The :field field must be between :param0 and :param1",
		"size":      "The :field field must be exactly :param0",
		"uuid":      "The :field field must be a valid UUID",
		"date":      "The :field field must be a valid date",
	}
}

// Global manager instance
var globalManager = NewManager()

// Package-level functions for convenience

// Make creates a validator using the global manager
func Make(data map[string]interface{}, rules map[string][]Rule) Validator {
	return globalManager.Make(data, rules)
}

// MakeRequest creates a validator from HTTP request using the global manager
func MakeRequest(r *http.Request, rules map[string][]Rule) Validator {
	return globalManager.MakeRequest(r, rules)
}

// MakeStruct creates a validator for struct using the global manager
func MakeStruct(s interface{}) Validator {
	return globalManager.MakeStruct(s)
}

// ValidateStruct validates a struct using the global manager
func ValidateStruct(ctx context.Context, s interface{}) Result {
	return globalManager.ValidateStruct(ctx, s)
}

// ValidateData validates data using the global manager
func ValidateData(ctx context.Context, data map[string]interface{}, rules map[string][]Rule) Result {
	return globalManager.ValidateData(ctx, data, rules)
}

// RegisterRule registers a custom rule with the global manager
func RegisterRule(name string, rule RuleFunc) error {
	return globalManager.RegisterRule(name, rule)
}

// SetMessages sets custom messages with the global manager
func SetMessages(messages map[string]string) {
	globalManager.SetMessages(messages)
}

// SetAttributes sets custom attributes with the global manager
func SetAttributes(attributes map[string]string) {
	globalManager.SetAttributes(attributes)
}

// GetManager returns the global manager instance
func GetManager() *DefaultManager {
	return globalManager
}

// parseValidationTag parses a validation tag string into rules
func parseValidationTag(tag string) ([]Rule, error) {
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
		rule, err := createRuleFromTag(ruleName, params)
		if err != nil {
			return nil, err
		}
		
		rules = append(rules, rule)
	}
	
	return rules, nil
}

// createRuleFromTag creates a rule from tag name and parameters
func createRuleFromTag(ruleName string, params []string) (Rule, error) {
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
	case "confirmed":
		return NewConfirmedRule(), nil
	case "between":
		if len(params) < 2 {
			return nil, fmt.Errorf("between rule requires two parameters")
		}
		return NewBetweenRule(params[0], params[1]), nil
	case "size":
		if len(params) == 0 {
			return nil, fmt.Errorf("size rule requires a parameter")
		}
		return NewSizeRule(params[0]), nil
	case "uuid":
		return NewUUIDRule(), nil
	case "date":
		format := ""
		if len(params) > 0 {
			format = params[0]
		}
		return NewDateRule(format), nil
	case "regex":
		if len(params) == 0 {
			return nil, fmt.Errorf("regex rule requires a pattern parameter")
		}
		return NewRegexRule(params[0])
	default:
		return nil, fmt.Errorf("unknown validation rule: %s", ruleName)
	}
}