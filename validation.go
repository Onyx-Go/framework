package onyx

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type Validator struct {
	data   map[string]interface{}
	rules  map[string][]string
	errors map[string][]string
}

type ValidationRule func(field string, value interface{}, params []string) error

var validationRules = map[string]ValidationRule{
	"required":  requiredRule,
	"min":       minRule,
	"max":       maxRule,
	"email":     emailRule,
	"numeric":   numericRule,
	"alpha":     alphaRule,
	"alpha_num": alphaNumRule,
	"url":       urlRule,
	"in":        inRule,
	"confirmed": confirmedRule,
}

func NewValidator(data map[string]interface{}, rules map[string][]string) *Validator {
	return &Validator{
		data:   data,
		rules:  rules,
		errors: make(map[string][]string),
	}
}

func (v *Validator) Validate() bool {
	v.errors = make(map[string][]string)
	
	for field, fieldRules := range v.rules {
		value := v.getValue(field)
		
		for _, ruleStr := range fieldRules {
			rule, params := v.parseRule(ruleStr)
			
			if validationFunc, exists := validationRules[rule]; exists {
				if err := validationFunc(field, value, params); err != nil {
					v.addError(field, err.Error())
				}
			}
		}
	}
	
	return len(v.errors) == 0
}

func (v *Validator) Errors() map[string][]string {
	return v.errors
}

func (v *Validator) FirstError(field string) string {
	if errors, exists := v.errors[field]; exists && len(errors) > 0 {
		return errors[0]
	}
	return ""
}

func (v *Validator) HasErrors() bool {
	return len(v.errors) > 0
}

func (v *Validator) getValue(field string) interface{} {
	if value, exists := v.data[field]; exists {
		return value
	}
	return nil
}

func (v *Validator) parseRule(ruleStr string) (string, []string) {
	parts := strings.Split(ruleStr, ":")
	rule := parts[0]
	var params []string
	
	if len(parts) > 1 {
		params = strings.Split(parts[1], ",")
	}
	
	return rule, params
}

func (v *Validator) addError(field, message string) {
	if _, exists := v.errors[field]; !exists {
		v.errors[field] = []string{}
	}
	v.errors[field] = append(v.errors[field], message)
}

func requiredRule(field string, value interface{}, params []string) error {
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
	}
	
	return nil
}

func minRule(field string, value interface{}, params []string) error {
	if len(params) == 0 {
		return fmt.Errorf("min rule requires a parameter")
	}
	
	minValue, err := strconv.Atoi(params[0])
	if err != nil {
		return fmt.Errorf("invalid min parameter")
	}
	
	switch v := value.(type) {
	case string:
		if len(v) < minValue {
			return fmt.Errorf("the %s field must be at least %d characters", field, minValue)
		}
	case int:
		if v < minValue {
			return fmt.Errorf("the %s field must be at least %d", field, minValue)
		}
	case float64:
		if v < float64(minValue) {
			return fmt.Errorf("the %s field must be at least %d", field, minValue)
		}
	}
	
	return nil
}

func maxRule(field string, value interface{}, params []string) error {
	if len(params) == 0 {
		return fmt.Errorf("max rule requires a parameter")
	}
	
	maxValue, err := strconv.Atoi(params[0])
	if err != nil {
		return fmt.Errorf("invalid max parameter")
	}
	
	switch v := value.(type) {
	case string:
		if len(v) > maxValue {
			return fmt.Errorf("the %s field may not be greater than %d characters", field, maxValue)
		}
	case int:
		if v > maxValue {
			return fmt.Errorf("the %s field may not be greater than %d", field, maxValue)
		}
	case float64:
		if v > float64(maxValue) {
			return fmt.Errorf("the %s field may not be greater than %d", field, maxValue)
		}
	}
	
	return nil
}

func emailRule(field string, value interface{}, params []string) error {
	if value == nil {
		return nil
	}
	
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("the %s field must be a string", field)
	}
	
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(str) {
		return fmt.Errorf("the %s field must be a valid email address", field)
	}
	
	return nil
}

func numericRule(field string, value interface{}, params []string) error {
	if value == nil {
		return nil
	}
	
	switch value.(type) {
	case int, int32, int64, float32, float64:
		return nil
	case string:
		str := value.(string)
		if _, err := strconv.ParseFloat(str, 64); err != nil {
			return fmt.Errorf("the %s field must be numeric", field)
		}
		return nil
	default:
		return fmt.Errorf("the %s field must be numeric", field)
	}
}

func alphaRule(field string, value interface{}, params []string) error {
	if value == nil {
		return nil
	}
	
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("the %s field must be a string", field)
	}
	
	alphaRegex := regexp.MustCompile(`^[a-zA-Z]+$`)
	if !alphaRegex.MatchString(str) {
		return fmt.Errorf("the %s field may only contain letters", field)
	}
	
	return nil
}

func alphaNumRule(field string, value interface{}, params []string) error {
	if value == nil {
		return nil
	}
	
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("the %s field must be a string", field)
	}
	
	alphaNumRegex := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	if !alphaNumRegex.MatchString(str) {
		return fmt.Errorf("the %s field may only contain letters and numbers", field)
	}
	
	return nil
}

func urlRule(field string, value interface{}, params []string) error {
	if value == nil {
		return nil
	}
	
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("the %s field must be a string", field)
	}
	
	urlRegex := regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
	if !urlRegex.MatchString(str) {
		return fmt.Errorf("the %s field must be a valid URL", field)
	}
	
	return nil
}

func inRule(field string, value interface{}, params []string) error {
	if value == nil {
		return nil
	}
	
	str := fmt.Sprintf("%v", value)
	
	for _, param := range params {
		if str == param {
			return nil
		}
	}
	
	return fmt.Errorf("the %s field must be one of: %s", field, strings.Join(params, ", "))
}

func confirmedRule(field string, value interface{}, params []string) error {
	return fmt.Errorf("confirmed rule not yet implemented for %s field", field)
}

func (c *Context) Validate(rules map[string][]string) (*Validator, error) {
	data := make(map[string]interface{})
	
	if err := c.Request.ParseForm(); err != nil {
		return nil, err
	}
	
	for key, values := range c.Request.Form {
		if len(values) == 1 {
			data[key] = values[0]
		} else {
			data[key] = values
		}
	}
	
	validator := NewValidator(data, rules)
	return validator, nil
}

func RegisterValidationRule(name string, rule ValidationRule) {
	validationRules[name] = rule
}