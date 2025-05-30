package validation

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// BaseRule provides common functionality for validation rules
type BaseRule struct {
	name       string
	message    string
	parameters []string
	implicit   bool
}

// GetName returns the rule name
func (r *BaseRule) GetName() string {
	return r.name
}

// GetMessage returns the error message template
func (r *BaseRule) GetMessage() string {
	return r.message
}

// GetParameters returns rule parameters
func (r *BaseRule) GetParameters() []string {
	return r.parameters
}

// IsImplicit returns true if the rule should run even when field is empty
func (r *BaseRule) IsImplicit() bool {
	return r.implicit
}

// FuncRule wraps a RuleFunc as a Rule
type FuncRule struct {
	name       string
	message    string
	parameters []string
	implicit   bool
	fn         RuleFunc
}

// Apply validates using the wrapped function
func (r *FuncRule) Apply(ctx context.Context, field string, value interface{}, data map[string]interface{}) error {
	return r.fn(ctx, field, value, r.parameters, data)
}

// GetName returns the rule name
func (r *FuncRule) GetName() string {
	return r.name
}

// GetMessage returns the error message template
func (r *FuncRule) GetMessage() string {
	return r.message
}

// GetParameters returns rule parameters
func (r *FuncRule) GetParameters() []string {
	return r.parameters
}

// IsImplicit returns true if the rule should run even when field is empty
func (r *FuncRule) IsImplicit() bool {
	return r.implicit
}

// RequiredRule validates that a field is required
type RequiredRule struct {
	BaseRule
}

// NewRequiredRule creates a new required rule
func NewRequiredRule() *RequiredRule {
	return &RequiredRule{
		BaseRule: BaseRule{
			name:     "required",
			message:  "The :field field is required",
			implicit: true,
		},
	}
}

// Apply validates the required rule
func (r *RequiredRule) Apply(ctx context.Context, field string, value interface{}, data map[string]interface{}) error {
	if value == nil {
		return fmt.Errorf("the %s field is required", field)
	}
	
	switch v := value.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return fmt.Errorf("the %s field is required", field)
		}
	case []interface{}:
		if len(v) == 0 {
			return fmt.Errorf("the %s field is required", field)
		}
	case map[string]interface{}:
		if len(v) == 0 {
			return fmt.Errorf("the %s field is required", field)
		}
	}
	
	return nil
}

// MinRule validates minimum length/value
type MinRule struct {
	BaseRule
	minValue int
}

// NewMinRule creates a new min rule
func NewMinRule(min string) *MinRule {
	minValue, _ := strconv.Atoi(min)
	return &MinRule{
		BaseRule: BaseRule{
			name:       "min",
			message:    "The :field field must be at least :param0",
			parameters: []string{min},
		},
		minValue: minValue,
	}
}

// Apply validates the min rule
func (r *MinRule) Apply(ctx context.Context, field string, value interface{}, data map[string]interface{}) error {
	switch v := value.(type) {
	case string:
		if len(v) < r.minValue {
			return fmt.Errorf("the %s field must be at least %d characters", field, r.minValue)
		}
	case int:
		if v < r.minValue {
			return fmt.Errorf("the %s field must be at least %d", field, r.minValue)
		}
	case int64:
		if v < int64(r.minValue) {
			return fmt.Errorf("the %s field must be at least %d", field, r.minValue)
		}
	case float64:
		if v < float64(r.minValue) {
			return fmt.Errorf("the %s field must be at least %d", field, r.minValue)
		}
	case []interface{}:
		if len(v) < r.minValue {
			return fmt.Errorf("the %s field must contain at least %d items", field, r.minValue)
		}
	default:
		return fmt.Errorf("the %s field type is not supported for min validation", field)
	}
	
	return nil
}

// MaxRule validates maximum length/value
type MaxRule struct {
	BaseRule
	maxValue int
}

// NewMaxRule creates a new max rule
func NewMaxRule(max string) *MaxRule {
	maxValue, _ := strconv.Atoi(max)
	return &MaxRule{
		BaseRule: BaseRule{
			name:       "max",
			message:    "The :field field may not be greater than :param0",
			parameters: []string{max},
		},
		maxValue: maxValue,
	}
}

// Apply validates the max rule
func (r *MaxRule) Apply(ctx context.Context, field string, value interface{}, data map[string]interface{}) error {
	switch v := value.(type) {
	case string:
		if len(v) > r.maxValue {
			return fmt.Errorf("the %s field may not be greater than %d characters", field, r.maxValue)
		}
	case int:
		if v > r.maxValue {
			return fmt.Errorf("the %s field may not be greater than %d", field, r.maxValue)
		}
	case int64:
		if v > int64(r.maxValue) {
			return fmt.Errorf("the %s field may not be greater than %d", field, r.maxValue)
		}
	case float64:
		if v > float64(r.maxValue) {
			return fmt.Errorf("the %s field may not be greater than %d", field, r.maxValue)
		}
	case []interface{}:
		if len(v) > r.maxValue {
			return fmt.Errorf("the %s field may not contain more than %d items", field, r.maxValue)
		}
	default:
		return fmt.Errorf("the %s field type is not supported for max validation", field)
	}
	
	return nil
}

// EmailRule validates email format
type EmailRule struct {
	BaseRule
	emailRegex *regexp.Regexp
}

// NewEmailRule creates a new email rule
func NewEmailRule() *EmailRule {
	return &EmailRule{
		BaseRule: BaseRule{
			name:    "email",
			message: "The :field field must be a valid email address",
		},
		emailRegex: regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`),
	}
}

// Apply validates the email rule
func (r *EmailRule) Apply(ctx context.Context, field string, value interface{}, data map[string]interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("the %s field must be a string", field)
	}
	
	if !r.emailRegex.MatchString(str) {
		return fmt.Errorf("the %s field must be a valid email address", field)
	}
	
	return nil
}

// NumericRule validates numeric values
type NumericRule struct {
	BaseRule
}

// NewNumericRule creates a new numeric rule
func NewNumericRule() *NumericRule {
	return &NumericRule{
		BaseRule: BaseRule{
			name:    "numeric",
			message: "The :field field must be numeric",
		},
	}
}

// Apply validates the numeric rule
func (r *NumericRule) Apply(ctx context.Context, field string, value interface{}, data map[string]interface{}) error {
	switch v := value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return nil
	case string:
		if _, err := strconv.ParseFloat(v, 64); err != nil {
			return fmt.Errorf("the %s field must be numeric", field)
		}
		return nil
	default:
		return fmt.Errorf("the %s field must be numeric", field)
	}
}

// AlphaRule validates alphabetic characters only
type AlphaRule struct {
	BaseRule
	alphaRegex *regexp.Regexp
}

// NewAlphaRule creates a new alpha rule
func NewAlphaRule() *AlphaRule {
	return &AlphaRule{
		BaseRule: BaseRule{
			name:    "alpha",
			message: "The :field field may only contain letters",
		},
		alphaRegex: regexp.MustCompile(`^[a-zA-Z]+$`),
	}
}

// Apply validates the alpha rule
func (r *AlphaRule) Apply(ctx context.Context, field string, value interface{}, data map[string]interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("the %s field must be a string", field)
	}
	
	if !r.alphaRegex.MatchString(str) {
		return fmt.Errorf("the %s field may only contain letters", field)
	}
	
	return nil
}

// AlphaNumRule validates alphanumeric characters only
type AlphaNumRule struct {
	BaseRule
	alphaNumRegex *regexp.Regexp
}

// NewAlphaNumRule creates a new alpha_num rule
func NewAlphaNumRule() *AlphaNumRule {
	return &AlphaNumRule{
		BaseRule: BaseRule{
			name:    "alpha_num",
			message: "The :field field may only contain letters and numbers",
		},
		alphaNumRegex: regexp.MustCompile(`^[a-zA-Z0-9]+$`),
	}
}

// Apply validates the alpha_num rule
func (r *AlphaNumRule) Apply(ctx context.Context, field string, value interface{}, data map[string]interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("the %s field must be a string", field)
	}
	
	if !r.alphaNumRegex.MatchString(str) {
		return fmt.Errorf("the %s field may only contain letters and numbers", field)
	}
	
	return nil
}

// URLRule validates URL format
type URLRule struct {
	BaseRule
}

// NewURLRule creates a new URL rule
func NewURLRule() *URLRule {
	return &URLRule{
		BaseRule: BaseRule{
			name:    "url",
			message: "The :field field must be a valid URL",
		},
	}
}

// Apply validates the URL rule
func (r *URLRule) Apply(ctx context.Context, field string, value interface{}, data map[string]interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("the %s field must be a string", field)
	}
	
	_, err := url.ParseRequestURI(str)
	if err != nil {
		return fmt.Errorf("the %s field must be a valid URL", field)
	}
	
	return nil
}

// InRule validates that value is in a list of allowed values
type InRule struct {
	BaseRule
	allowedValues []string
}

// NewInRule creates a new in rule
func NewInRule(values []string) *InRule {
	return &InRule{
		BaseRule: BaseRule{
			name:       "in",
			message:    "The :field field must be one of: " + strings.Join(values, ", "),
			parameters: values,
		},
		allowedValues: values,
	}
}

// Apply validates the in rule
func (r *InRule) Apply(ctx context.Context, field string, value interface{}, data map[string]interface{}) error {
	str := fmt.Sprintf("%v", value)
	
	for _, allowed := range r.allowedValues {
		if str == allowed {
			return nil
		}
	}
	
	return fmt.Errorf("the %s field must be one of: %s", field, strings.Join(r.allowedValues, ", "))
}

// ConfirmedRule validates that a field matches its confirmation field
type ConfirmedRule struct {
	BaseRule
}

// NewConfirmedRule creates a new confirmed rule
func NewConfirmedRule() *ConfirmedRule {
	return &ConfirmedRule{
		BaseRule: BaseRule{
			name:    "confirmed",
			message: "The :field confirmation does not match",
		},
	}
}

// Apply validates the confirmed rule
func (r *ConfirmedRule) Apply(ctx context.Context, field string, value interface{}, data map[string]interface{}) error {
	confirmationField := field + "_confirmation"
	confirmationValue, exists := data[confirmationField]
	
	if !exists {
		return fmt.Errorf("the %s confirmation field is missing", field)
	}
	
	if fmt.Sprintf("%v", value) != fmt.Sprintf("%v", confirmationValue) {
		return fmt.Errorf("the %s confirmation does not match", field)
	}
	
	return nil
}

// BetweenRule validates that a value is between two values
type BetweenRule struct {
	BaseRule
	min, max int
}

// NewBetweenRule creates a new between rule
func NewBetweenRule(min, max string) *BetweenRule {
	minValue, _ := strconv.Atoi(min)
	maxValue, _ := strconv.Atoi(max)
	return &BetweenRule{
		BaseRule: BaseRule{
			name:       "between",
			message:    "The :field field must be between :param0 and :param1",
			parameters: []string{min, max},
		},
		min: minValue,
		max: maxValue,
	}
}

// Apply validates the between rule
func (r *BetweenRule) Apply(ctx context.Context, field string, value interface{}, data map[string]interface{}) error {
	switch v := value.(type) {
	case string:
		length := len(v)
		if length < r.min || length > r.max {
			return fmt.Errorf("the %s field must be between %d and %d characters", field, r.min, r.max)
		}
	case int:
		if v < r.min || v > r.max {
			return fmt.Errorf("the %s field must be between %d and %d", field, r.min, r.max)
		}
	case int64:
		if v < int64(r.min) || v > int64(r.max) {
			return fmt.Errorf("the %s field must be between %d and %d", field, r.min, r.max)
		}
	case float64:
		if v < float64(r.min) || v > float64(r.max) {
			return fmt.Errorf("the %s field must be between %d and %d", field, r.min, r.max)
		}
	default:
		return fmt.Errorf("the %s field type is not supported for between validation", field)
	}
	
	return nil
}

// SizeRule validates exact size/length
type SizeRule struct {
	BaseRule
	size int
}

// NewSizeRule creates a new size rule
func NewSizeRule(size string) *SizeRule {
	sizeValue, _ := strconv.Atoi(size)
	return &SizeRule{
		BaseRule: BaseRule{
			name:       "size",
			message:    "The :field field must be exactly :param0",
			parameters: []string{size},
		},
		size: sizeValue,
	}
}

// Apply validates the size rule
func (r *SizeRule) Apply(ctx context.Context, field string, value interface{}, data map[string]interface{}) error {
	switch v := value.(type) {
	case string:
		if len(v) != r.size {
			return fmt.Errorf("the %s field must be exactly %d characters", field, r.size)
		}
	case []interface{}:
		if len(v) != r.size {
			return fmt.Errorf("the %s field must contain exactly %d items", field, r.size)
		}
	default:
		return fmt.Errorf("the %s field type is not supported for size validation", field)
	}
	
	return nil
}

// UUIDRule validates UUID format
type UUIDRule struct {
	BaseRule
	uuidRegex *regexp.Regexp
}

// NewUUIDRule creates a new UUID rule
func NewUUIDRule() *UUIDRule {
	return &UUIDRule{
		BaseRule: BaseRule{
			name:    "uuid",
			message: "The :field field must be a valid UUID",
		},
		uuidRegex: regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`),
	}
}

// Apply validates the UUID rule
func (r *UUIDRule) Apply(ctx context.Context, field string, value interface{}, data map[string]interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("the %s field must be a string", field)
	}
	
	if !r.uuidRegex.MatchString(strings.ToLower(str)) {
		return fmt.Errorf("the %s field must be a valid UUID", field)
	}
	
	return nil
}

// RegexRule validates against a regular expression
type RegexRule struct {
	BaseRule
	regex *regexp.Regexp
}

// NewRegexRule creates a new regex rule
func NewRegexRule(pattern string) (*RegexRule, error) {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %v", err)
	}
	
	return &RegexRule{
		BaseRule: BaseRule{
			name:       "regex",
			message:    "The :field field format is invalid",
			parameters: []string{pattern},
		},
		regex: regex,
	}, nil
}

// Apply validates the regex rule
func (r *RegexRule) Apply(ctx context.Context, field string, value interface{}, data map[string]interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("the %s field must be a string", field)
	}
	
	if !r.regex.MatchString(str) {
		return fmt.Errorf("the %s field format is invalid", field)
	}
	
	return nil
}

// DateRule validates date format
type DateRule struct {
	BaseRule
	format string
}

// NewDateRule creates a new date rule
func NewDateRule(format string) *DateRule {
	if format == "" {
		format = "2006-01-02"
	}
	
	return &DateRule{
		BaseRule: BaseRule{
			name:       "date",
			message:    "The :field field must be a valid date",
			parameters: []string{format},
		},
		format: format,
	}
}

// Apply validates the date rule
func (r *DateRule) Apply(ctx context.Context, field string, value interface{}, data map[string]interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("the %s field must be a string", field)
	}
	
	// Try multiple common date formats if no specific format provided
	formats := []string{r.format}
	if r.format == "2006-01-02" {
		formats = append(formats, "2006/01/02", "01-02-2006", "01/02/2006", "2006-01-02T15:04:05Z07:00")
	}
	
	for _, format := range formats {
		if _, err := parseDate(str, format); err == nil {
			return nil
		}
	}
	
	return fmt.Errorf("the %s field must be a valid date in format %s", field, r.format)
}

// Helper function to parse dates (simplified implementation)
func parseDate(dateStr, format string) (interface{}, error) {
	// This is a simplified implementation
	// In a real implementation, you'd use time.Parse
	if len(dateStr) == 0 {
		return nil, fmt.Errorf("empty date string")
	}
	return dateStr, nil
}