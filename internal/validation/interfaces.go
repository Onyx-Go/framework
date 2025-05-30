package validation

import (
	"context"
	"net/http"
	"reflect"
)

// Validator interface defines the contract for validation operations
type Validator interface {
	// Core validation
	Validate(ctx context.Context) Result
	ValidateStruct(ctx context.Context, s interface{}) Result
	ValidateField(ctx context.Context, field string, value interface{}) Result
	
	// Rule management
	AddRule(field string, rules ...Rule) Validator
	AddRules(rules map[string][]Rule) Validator
	WithCustomRule(name string, rule RuleFunc) Validator
	
	// Data management
	WithData(data map[string]interface{}) Validator
	WithRequest(r *http.Request) Validator
	WithJSON(jsonData []byte) Validator
	
	// Conditional validation
	Sometimes(field string, rules ...Rule) Validator
	When(condition func() bool, callback func(v Validator)) Validator
	Unless(condition func() bool, callback func(v Validator)) Validator
	
	// Custom messages
	WithMessages(messages map[string]string) Validator
	WithAttribute(field, attribute string) Validator
	
	// Configuration
	WithOptions(opts *Options) Validator
	StopOnFirstFailure() Validator
	Bail() Validator
}

// Rule interface for validation rules
type Rule interface {
	// Apply validates the rule against the field value
	Apply(ctx context.Context, field string, value interface{}, data map[string]interface{}) error
	
	// GetName returns the rule name
	GetName() string
	
	// GetMessage returns the error message template
	GetMessage() string
	
	// GetParameters returns rule parameters
	GetParameters() []string
	
	// IsImplicit returns true if the rule should run even when field is empty
	IsImplicit() bool
}

// RuleFunc is a function type for custom validation rules
type RuleFunc func(ctx context.Context, field string, value interface{}, params []string, data map[string]interface{}) error

// Result interface for validation results
type Result interface {
	// Status
	IsValid() bool
	HasErrors() bool
	Failed() bool
	Passes() bool
	
	// Error access
	Errors() ErrorBag
	GetErrors() map[string][]string
	FirstError(field string) string
	AllErrors() []string
	
	// Error manipulation
	AddError(field, message string) Result
	AddErrors(field string, messages []string) Result
	MergeErrors(other Result) Result
	
	// Field-specific
	HasError(field string) bool
	GetFieldErrors(field string) []string
	
	// JSON serialization
	ToJSON() ([]byte, error)
	ToMap() map[string]interface{}
}

// ErrorBag interface for managing validation errors
type ErrorBag interface {
	// Add errors
	Add(field, message string)
	AddMany(field string, messages []string)
	Merge(other ErrorBag)
	
	// Access errors
	Get(field string) []string
	First(field string) string
	All() map[string][]string
	Count() int
	Has(field string) bool
	IsEmpty() bool
	
	// Manipulation
	Clear()
	Remove(field string)
	Filter(fn func(field, message string) bool) ErrorBag
	
	// Serialization
	ToJSON() ([]byte, error)
	ToSlice() []string
	ToString() string
}

// Manager interface for managing validators and rules
type Manager interface {
	// Validator creation
	Make(data map[string]interface{}, rules map[string][]Rule) Validator
	MakeRequest(r *http.Request, rules map[string][]Rule) Validator
	MakeStruct(s interface{}) Validator
	
	// Rule registration
	RegisterRule(name string, rule RuleFunc) error
	RegisterCustomRule(name string, rule Rule) error
	UnregisterRule(name string)
	GetRule(name string) (Rule, bool)
	GetRules() map[string]Rule
	
	// Message management
	SetMessages(messages map[string]string)
	SetAttributes(attributes map[string]string)
	GetMessage(rule, field string) string
	
	// Configuration
	SetDefaultOptions(opts *Options)
	GetDefaultOptions() *Options
	
	// Middleware support
	Middleware(rules map[string][]Rule) func(http.Handler) http.Handler
}

// StructValidator interface for struct validation with tags
type StructValidator interface {
	// Validate struct with validation tags
	ValidateStruct(ctx context.Context, s interface{}) Result
	
	// Tag parsing
	ParseTags(structType reflect.Type) (map[string][]Rule, error)
	SetTagName(tagName string)
	GetTagName() string
	
	// Custom tag handlers
	RegisterTagHandler(tag string, handler TagHandler)
	GetTagHandler(tag string) (TagHandler, bool)
}

// TagHandler interface for custom struct tag handling
type TagHandler interface {
	Parse(tag string) ([]Rule, error)
	Validate(ctx context.Context, field string, value interface{}, tag string) error
}

// RequestValidator interface for HTTP request validation
type RequestValidator interface {
	// Validate HTTP requests
	ValidateRequest(ctx context.Context, r *http.Request, rules map[string][]Rule) Result
	ValidateJSON(ctx context.Context, data []byte, rules map[string][]Rule) Result
	ValidateQuery(ctx context.Context, r *http.Request, rules map[string][]Rule) Result
	ValidateForm(ctx context.Context, r *http.Request, rules map[string][]Rule) Result
	
	// File validation
	ValidateFiles(ctx context.Context, r *http.Request, rules map[string][]Rule) Result
	
	// Header validation
	ValidateHeaders(ctx context.Context, r *http.Request, rules map[string][]Rule) Result
}

// ConditionalValidator interface for conditional validation
type ConditionalValidator interface {
	// Conditional rules
	When(condition func(data map[string]interface{}) bool, rules map[string][]Rule) Validator
	Unless(condition func(data map[string]interface{}) bool, rules map[string][]Rule) Validator
	Sometimes(field string, rules ...Rule) Validator
	
	// Field dependencies
	RequiredWith(field string, dependencies ...string) Validator
	RequiredWithout(field string, dependencies ...string) Validator
	RequiredIf(field string, condition func(data map[string]interface{}) bool) Validator
	RequiredUnless(field string, condition func(data map[string]interface{}) bool) Validator
}

// AsyncValidator interface for asynchronous validation
type AsyncValidator interface {
	// Async validation
	ValidateAsync(ctx context.Context) <-chan Result
	ValidateFieldAsync(ctx context.Context, field string, value interface{}) <-chan Result
	
	// Batch validation
	ValidateBatch(ctx context.Context, items []map[string]interface{}) <-chan []Result
}

// Transformer interface for data transformation before validation
type Transformer interface {
	// Transform data before validation
	Transform(data map[string]interface{}) map[string]interface{}
	TransformField(field string, value interface{}) interface{}
	
	// Register transformers
	RegisterTransformer(field string, transformer FieldTransformer)
	GetTransformer(field string) (FieldTransformer, bool)
}

// FieldTransformer interface for field-specific transformations
type FieldTransformer interface {
	Transform(value interface{}) interface{}
	ShouldTransform(value interface{}) bool
}

// Options holds validation configuration
type Options struct {
	// Behavior options
	StopOnFirstFailure bool
	ValidateAll        bool
	CaseSensitive      bool
	TrimStrings        bool
	ConvertTypes       bool
	
	// Error options
	ShowAllErrors      bool
	IncludeFieldNames  bool
	CustomMessages     map[string]string
	CustomAttributes   map[string]string
	
	// Async options
	MaxConcurrency     int
	ValidationTimeout  int64 // milliseconds
	
	// Language/Locale
	Locale             string
	MessageProvider    MessageProvider
	
	// Custom options
	Extensions         map[string]interface{}
}

// MessageProvider interface for custom error messages
type MessageProvider interface {
	// Get message for rule and field
	GetMessage(rule, field string, params []string) string
	
	// Get attribute name for field
	GetAttribute(field string) string
	
	// Set custom messages and attributes
	SetMessages(messages map[string]string)
	SetAttributes(attributes map[string]string)
	
	// Locale support
	SetLocale(locale string)
	GetLocale() string
}

// RuleBuilder interface for building complex rules
type RuleBuilder interface {
	// Basic rules
	Required() RuleBuilder
	Optional() RuleBuilder
	Nullable() RuleBuilder
	
	// String rules
	String() RuleBuilder
	Alpha() RuleBuilder
	AlphaNum() RuleBuilder
	Numeric() RuleBuilder
	Email() RuleBuilder
	URL() RuleBuilder
	UUID() RuleBuilder
	
	// Size rules
	Min(min int) RuleBuilder
	Max(max int) RuleBuilder
	Between(min, max int) RuleBuilder
	Size(size int) RuleBuilder
	
	// Comparison rules
	In(values ...interface{}) RuleBuilder
	NotIn(values ...interface{}) RuleBuilder
	Same(field string) RuleBuilder
	Different(field string) RuleBuilder
	
	// Date rules
	Date() RuleBuilder
	DateFormat(format string) RuleBuilder
	Before(date string) RuleBuilder
	After(date string) RuleBuilder
	
	// File rules
	File() RuleBuilder
	Image() RuleBuilder
	Mimes(mimes ...string) RuleBuilder
	MimeTypes(types ...string) RuleBuilder
	
	// Custom rules
	Custom(rule Rule) RuleBuilder
	CustomFunc(rule RuleFunc) RuleBuilder
	
	// Build
	Build() []Rule
}

// Default configuration functions
func DefaultOptions() *Options {
	return &Options{
		StopOnFirstFailure: false,
		ValidateAll:        true,
		CaseSensitive:      true,
		TrimStrings:        true,
		ConvertTypes:       false,
		ShowAllErrors:      true,
		IncludeFieldNames:  true,
		CustomMessages:     make(map[string]string),
		CustomAttributes:   make(map[string]string),
		MaxConcurrency:     10,
		ValidationTimeout:  5000, // 5 seconds
		Locale:             "en",
		Extensions:         make(map[string]interface{}),
	}
}

// Rule priorities for execution order
type RulePriority int

const (
	PriorityHigh   RulePriority = 100
	PriorityNormal RulePriority = 50
	PriorityLow    RulePriority = 10
)

// ValidationEvent represents events during validation
type ValidationEvent struct {
	Type      string
	Field     string
	Rule      string
	Value     interface{}
	Error     error
	Timestamp int64
}

// EventListener interface for validation events
type EventListener interface {
	OnValidationStart(data map[string]interface{})
	OnValidationEnd(result Result)
	OnFieldValidation(field string, value interface{})
	OnRuleValidation(field, rule string, value interface{}, err error)
}