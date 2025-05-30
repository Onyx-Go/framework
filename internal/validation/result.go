package validation

import (
	"encoding/json"
	"strings"
	"sync"
)

// DefaultResult implements the Result interface
type DefaultResult struct {
	errors ErrorBag
	mutex  sync.RWMutex
}

// NewResult creates a new validation result
func NewResult() *DefaultResult {
	return &DefaultResult{
		errors: NewErrorBag(),
	}
}

// IsValid returns true if validation passed
func (r *DefaultResult) IsValid() bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.errors.IsEmpty()
}

// HasErrors returns true if there are validation errors
func (r *DefaultResult) HasErrors() bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return !r.errors.IsEmpty()
}

// Failed returns true if validation failed
func (r *DefaultResult) Failed() bool {
	return r.HasErrors()
}

// Passes returns true if validation passed
func (r *DefaultResult) Passes() bool {
	return r.IsValid()
}

// Errors returns the error bag
func (r *DefaultResult) Errors() ErrorBag {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.errors
}

// GetErrors returns all errors as a map
func (r *DefaultResult) GetErrors() map[string][]string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.errors.All()
}

// FirstError returns the first error for a field
func (r *DefaultResult) FirstError(field string) string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.errors.First(field)
}

// AllErrors returns all error messages as a slice
func (r *DefaultResult) AllErrors() []string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.errors.ToSlice()
}

// AddError adds an error for a field
func (r *DefaultResult) AddError(field, message string) Result {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.errors.Add(field, message)
	return r
}

// AddErrors adds multiple errors for a field
func (r *DefaultResult) AddErrors(field string, messages []string) Result {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.errors.AddMany(field, messages)
	return r
}

// MergeErrors merges errors from another result
func (r *DefaultResult) MergeErrors(other Result) Result {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.errors.Merge(other.Errors())
	return r
}

// HasError returns true if field has errors
func (r *DefaultResult) HasError(field string) bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.errors.Has(field)
}

// GetFieldErrors returns errors for a specific field
func (r *DefaultResult) GetFieldErrors(field string) []string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.errors.Get(field)
}

// ToJSON serializes the result to JSON
func (r *DefaultResult) ToJSON() ([]byte, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.errors.ToJSON()
}

// ToMap converts the result to a map
func (r *DefaultResult) ToMap() map[string]interface{} {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	return map[string]interface{}{
		"valid":  r.IsValid(),
		"errors": r.errors.All(),
		"count":  r.errors.Count(),
	}
}

// DefaultErrorBag implements the ErrorBag interface
type DefaultErrorBag struct {
	errors map[string][]string
	mutex  sync.RWMutex
}

// NewErrorBag creates a new error bag
func NewErrorBag() *DefaultErrorBag {
	return &DefaultErrorBag{
		errors: make(map[string][]string),
	}
}

// Add adds an error for a field
func (eb *DefaultErrorBag) Add(field, message string) {
	eb.mutex.Lock()
	defer eb.mutex.Unlock()
	
	if eb.errors == nil {
		eb.errors = make(map[string][]string)
	}
	
	eb.errors[field] = append(eb.errors[field], message)
}

// AddMany adds multiple errors for a field
func (eb *DefaultErrorBag) AddMany(field string, messages []string) {
	eb.mutex.Lock()
	defer eb.mutex.Unlock()
	
	if eb.errors == nil {
		eb.errors = make(map[string][]string)
	}
	
	eb.errors[field] = append(eb.errors[field], messages...)
}

// Merge merges errors from another error bag
func (eb *DefaultErrorBag) Merge(other ErrorBag) {
	eb.mutex.Lock()
	defer eb.mutex.Unlock()
	
	if eb.errors == nil {
		eb.errors = make(map[string][]string)
	}
	
	for field, messages := range other.All() {
		eb.errors[field] = append(eb.errors[field], messages...)
	}
}

// Get returns errors for a field
func (eb *DefaultErrorBag) Get(field string) []string {
	eb.mutex.RLock()
	defer eb.mutex.RUnlock()
	
	if errors, exists := eb.errors[field]; exists {
		// Return a copy to prevent external modification
		result := make([]string, len(errors))
		copy(result, errors)
		return result
	}
	
	return []string{}
}

// First returns the first error for a field
func (eb *DefaultErrorBag) First(field string) string {
	eb.mutex.RLock()
	defer eb.mutex.RUnlock()
	
	if errors, exists := eb.errors[field]; exists && len(errors) > 0 {
		return errors[0]
	}
	
	return ""
}

// All returns all errors
func (eb *DefaultErrorBag) All() map[string][]string {
	eb.mutex.RLock()
	defer eb.mutex.RUnlock()
	
	// Return a deep copy to prevent external modification
	result := make(map[string][]string)
	for field, messages := range eb.errors {
		result[field] = make([]string, len(messages))
		copy(result[field], messages)
	}
	
	return result
}

// Count returns the total number of errors
func (eb *DefaultErrorBag) Count() int {
	eb.mutex.RLock()
	defer eb.mutex.RUnlock()
	
	count := 0
	for _, messages := range eb.errors {
		count += len(messages)
	}
	
	return count
}

// Has returns true if field has errors
func (eb *DefaultErrorBag) Has(field string) bool {
	eb.mutex.RLock()
	defer eb.mutex.RUnlock()
	
	errors, exists := eb.errors[field]
	return exists && len(errors) > 0
}

// IsEmpty returns true if there are no errors
func (eb *DefaultErrorBag) IsEmpty() bool {
	eb.mutex.RLock()
	defer eb.mutex.RUnlock()
	
	return len(eb.errors) == 0 || eb.Count() == 0
}

// Clear removes all errors
func (eb *DefaultErrorBag) Clear() {
	eb.mutex.Lock()
	defer eb.mutex.Unlock()
	
	eb.errors = make(map[string][]string)
}

// Remove removes errors for a field
func (eb *DefaultErrorBag) Remove(field string) {
	eb.mutex.Lock()
	defer eb.mutex.Unlock()
	
	delete(eb.errors, field)
}

// Filter filters errors based on a predicate function
func (eb *DefaultErrorBag) Filter(fn func(field, message string) bool) ErrorBag {
	eb.mutex.RLock()
	defer eb.mutex.RUnlock()
	
	filtered := NewErrorBag()
	
	for field, messages := range eb.errors {
		for _, message := range messages {
			if fn(field, message) {
				filtered.Add(field, message)
			}
		}
	}
	
	return filtered
}

// ToJSON serializes errors to JSON
func (eb *DefaultErrorBag) ToJSON() ([]byte, error) {
	eb.mutex.RLock()
	defer eb.mutex.RUnlock()
	
	return json.Marshal(eb.errors)
}

// ToSlice returns all error messages as a flat slice
func (eb *DefaultErrorBag) ToSlice() []string {
	eb.mutex.RLock()
	defer eb.mutex.RUnlock()
	
	var result []string
	for _, messages := range eb.errors {
		result = append(result, messages...)
	}
	
	return result
}

// ToString returns all errors as a formatted string
func (eb *DefaultErrorBag) ToString() string {
	eb.mutex.RLock()
	defer eb.mutex.RUnlock()
	
	var lines []string
	for field, messages := range eb.errors {
		for _, message := range messages {
			lines = append(lines, field+": "+message)
		}
	}
	
	return strings.Join(lines, "\n")
}

// ValidationError represents a structured validation error
type ValidationError struct {
	Field   string      `json:"field"`
	Value   interface{} `json:"value,omitempty"`
	Rule    string      `json:"rule"`
	Message string      `json:"message"`
	Params  []string    `json:"params,omitempty"`
}

// Error implements the error interface
func (ve *ValidationError) Error() string {
	return ve.Message
}

// NewValidationError creates a new validation error
func NewValidationError(field, rule, message string, value interface{}, params []string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Value:   value,
		Rule:    rule,
		Message: message,
		Params:  params,
	}
}

// ValidationErrors represents multiple validation errors
type ValidationErrors []*ValidationError

// Error implements the error interface
func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return "validation failed"
	}
	
	if len(ve) == 1 {
		return ve[0].Error()
	}
	
	var messages []string
	for _, err := range ve {
		messages = append(messages, err.Error())
	}
	
	return strings.Join(messages, "; ")
}

// ToMap converts validation errors to a map
func (ve ValidationErrors) ToMap() map[string][]string {
	result := make(map[string][]string)
	
	for _, err := range ve {
		result[err.Field] = append(result[err.Field], err.Message)
	}
	
	return result
}

// ToJSON serializes validation errors to JSON
func (ve ValidationErrors) ToJSON() ([]byte, error) {
	return json.Marshal(ve.ToMap())
}

// First returns the first error for a field
func (ve ValidationErrors) First(field string) string {
	for _, err := range ve {
		if err.Field == field {
			return err.Message
		}
	}
	return ""
}

// Get returns all errors for a field
func (ve ValidationErrors) Get(field string) []string {
	var messages []string
	
	for _, err := range ve {
		if err.Field == field {
			messages = append(messages, err.Message)
		}
	}
	
	return messages
}

// Has returns true if there are errors for a field
func (ve ValidationErrors) Has(field string) bool {
	for _, err := range ve {
		if err.Field == field {
			return true
		}
	}
	return false
}

// Count returns the total number of errors
func (ve ValidationErrors) Count() int {
	return len(ve)
}

// IsEmpty returns true if there are no errors
func (ve ValidationErrors) IsEmpty() bool {
	return len(ve) == 0
}

// Detailed validation result with enhanced information
type DetailedResult struct {
	*DefaultResult
	validatedFields []string
	skippedFields   []string
	duration        int64 // microseconds
	metadata        map[string]interface{}
}

// NewDetailedResult creates a new detailed validation result
func NewDetailedResult() *DetailedResult {
	return &DetailedResult{
		DefaultResult:   NewResult(),
		validatedFields: make([]string, 0),
		skippedFields:   make([]string, 0),
		metadata:        make(map[string]interface{}),
	}
}

// GetValidatedFields returns the list of validated fields
func (dr *DetailedResult) GetValidatedFields() []string {
	return dr.validatedFields
}

// GetSkippedFields returns the list of skipped fields
func (dr *DetailedResult) GetSkippedFields() []string {
	return dr.skippedFields
}

// GetDuration returns the validation duration in microseconds
func (dr *DetailedResult) GetDuration() int64 {
	return dr.duration
}

// GetMetadata returns validation metadata
func (dr *DetailedResult) GetMetadata() map[string]interface{} {
	return dr.metadata
}

// SetDuration sets the validation duration
func (dr *DetailedResult) SetDuration(duration int64) {
	dr.duration = duration
}

// AddValidatedField adds a field to the validated list
func (dr *DetailedResult) AddValidatedField(field string) {
	dr.validatedFields = append(dr.validatedFields, field)
}

// AddSkippedField adds a field to the skipped list
func (dr *DetailedResult) AddSkippedField(field string) {
	dr.skippedFields = append(dr.skippedFields, field)
}

// SetMetadata sets validation metadata
func (dr *DetailedResult) SetMetadata(key string, value interface{}) {
	dr.metadata[key] = value
}

// ToDetailedMap converts the result to a detailed map
func (dr *DetailedResult) ToDetailedMap() map[string]interface{} {
	result := dr.ToMap()
	result["validated_fields"] = dr.validatedFields
	result["skipped_fields"] = dr.skippedFields
	result["duration_microseconds"] = dr.duration
	result["metadata"] = dr.metadata
	
	return result
}