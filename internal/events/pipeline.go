package events

import (
	"context"
	"fmt"
)

// DefaultPipeline implements the Pipeline interface
type DefaultPipeline struct {
	filters      []Filter
	transformers []Transformer
}

// NewPipeline creates a new pipeline
func NewPipeline() *DefaultPipeline {
	return &DefaultPipeline{
		filters:      make([]Filter, 0),
		transformers: make([]Transformer, 0),
	}
}

// AddFilter adds a filter to the pipeline
func (p *DefaultPipeline) AddFilter(filter Filter) {
	p.filters = append(p.filters, filter)
}

// AddTransformer adds a transformer to the pipeline
func (p *DefaultPipeline) AddTransformer(transformer Transformer) {
	p.transformers = append(p.transformers, transformer)
}

// Process processes an event through the pipeline
func (p *DefaultPipeline) Process(ctx context.Context, event Event) (Event, error) {
	// Apply filters first
	for _, filter := range p.filters {
		if !filter.Filter(ctx, event) {
			return nil, fmt.Errorf("event filtered out by %T", filter)
		}
	}

	// Apply transformers
	currentEvent := event
	for _, transformer := range p.transformers {
		transformed, err := transformer.Transform(ctx, currentEvent)
		if err != nil {
			return nil, fmt.Errorf("transformation error by %T: %w", transformer, err)
		}
		currentEvent = transformed
	}

	return currentEvent, nil
}

// FilterFunc implements the Filter interface
type FilterFunc func(ctx context.Context, event Event) bool

// Filter implements the Filter interface
func (ff FilterFunc) Filter(ctx context.Context, event Event) bool {
	return ff(ctx, event)
}

// TransformerFunc implements the Transformer interface
type TransformerFunc func(ctx context.Context, event Event) (Event, error)

// Transform implements the Transformer interface
func (tf TransformerFunc) Transform(ctx context.Context, event Event) (Event, error) {
	return tf(ctx, event)
}

// Common filters

// EventNameFilter filters events by name pattern
type EventNameFilter struct {
	allowedPatterns []string
	blockedPatterns []string
}

// NewEventNameFilter creates a new event name filter
func NewEventNameFilter(allowedPatterns, blockedPatterns []string) *EventNameFilter {
	return &EventNameFilter{
		allowedPatterns: allowedPatterns,
		blockedPatterns: blockedPatterns,
	}
}

// Filter implements the Filter interface
func (f *EventNameFilter) Filter(ctx context.Context, event Event) bool {
	eventName := event.GetName()

	// Check blocked patterns first
	for _, pattern := range f.blockedPatterns {
		if matchesPattern(pattern, eventName) {
			return false
		}
	}

	// If no allowed patterns, allow all (unless blocked)
	if len(f.allowedPatterns) == 0 {
		return true
	}

	// Check allowed patterns
	for _, pattern := range f.allowedPatterns {
		if matchesPattern(pattern, eventName) {
			return true
		}
	}

	return false
}

// PriorityFilter filters events by priority
type PriorityFilter struct {
	minPriority Priority
}

// NewPriorityFilter creates a new priority filter
func NewPriorityFilter(minPriority Priority) *PriorityFilter {
	return &PriorityFilter{minPriority: minPriority}
}

// Filter implements the Filter interface
func (f *PriorityFilter) Filter(ctx context.Context, event Event) bool {
	if baseEvent, ok := event.(*BaseEvent); ok {
		return baseEvent.GetPriority() >= f.minPriority
	}
	return true // Allow events that don't have priority
}

// MetadataFilter filters events based on metadata
type MetadataFilter struct {
	requiredKeys   []string
	keyValuePairs  map[string]interface{}
}

// NewMetadataFilter creates a new metadata filter
func NewMetadataFilter(requiredKeys []string, keyValuePairs map[string]interface{}) *MetadataFilter {
	return &MetadataFilter{
		requiredKeys:  requiredKeys,
		keyValuePairs: keyValuePairs,
	}
}

// Filter implements the Filter interface
func (f *MetadataFilter) Filter(ctx context.Context, event Event) bool {
	metadata := event.GetMetadata()

	// Check required keys
	for _, key := range f.requiredKeys {
		if _, exists := metadata[key]; !exists {
			return false
		}
	}

	// Check key-value pairs
	for key, expectedValue := range f.keyValuePairs {
		if actualValue, exists := metadata[key]; !exists || actualValue != expectedValue {
			return false
		}
	}

	return true
}

// Common transformers

// MetadataTransformer adds or modifies event metadata
type MetadataTransformer struct {
	metadata map[string]interface{}
}

// NewMetadataTransformer creates a new metadata transformer
func NewMetadataTransformer(metadata map[string]interface{}) *MetadataTransformer {
	return &MetadataTransformer{metadata: metadata}
}

// Transform implements the Transformer interface
func (t *MetadataTransformer) Transform(ctx context.Context, event Event) (Event, error) {
	if baseEvent, ok := event.(*BaseEvent); ok {
		cloned := baseEvent.Clone()
		newEvent := cloned.(*BaseEvent)
		for key, value := range t.metadata {
			newEvent.SetMetadata(key, value)
		}
		return newEvent, nil
	}
	return event, nil
}

// PayloadTransformer transforms event payload
type PayloadTransformer struct {
	transformFunc func(payload interface{}) (interface{}, error)
}

// NewPayloadTransformer creates a new payload transformer
func NewPayloadTransformer(transformFunc func(payload interface{}) (interface{}, error)) *PayloadTransformer {
	return &PayloadTransformer{transformFunc: transformFunc}
}

// Transform implements the Transformer interface
func (t *PayloadTransformer) Transform(ctx context.Context, event Event) (Event, error) {
	if baseEvent, ok := event.(*BaseEvent); ok {
		newPayload, err := t.transformFunc(baseEvent.GetPayload())
		if err != nil {
			return nil, err
		}
		
		newEvent := NewEventWithContext(ctx, baseEvent.GetName(), newPayload)
		
		// Copy metadata
		for key, value := range baseEvent.GetMetadata() {
			newEvent.SetMetadata(key, value)
		}
		
		return newEvent, nil
	}
	return event, nil
}

// ContextTransformer adds context values as metadata
type ContextTransformer struct {
	contextKeys []string
}

// NewContextTransformer creates a new context transformer
func NewContextTransformer(contextKeys []string) *ContextTransformer {
	return &ContextTransformer{contextKeys: contextKeys}
}

// Transform implements the Transformer interface
func (t *ContextTransformer) Transform(ctx context.Context, event Event) (Event, error) {
	if baseEvent, ok := event.(*BaseEvent); ok {
		cloned := baseEvent.Clone()
		newEvent := cloned.(*BaseEvent)
		
		for _, key := range t.contextKeys {
			if value := ctx.Value(key); value != nil {
				newEvent.SetMetadata(key, value)
			}
		}
		
		return newEvent, nil
	}
	return event, nil
}

// ValidationTransformer validates events and adds validation metadata
type ValidationTransformer struct {
	validator Validator
}

// NewValidationTransformer creates a new validation transformer
func NewValidationTransformer(validator Validator) *ValidationTransformer {
	return &ValidationTransformer{validator: validator}
}

// Transform implements the Transformer interface
func (t *ValidationTransformer) Transform(ctx context.Context, event Event) (Event, error) {
	err := t.validator.Validate(ctx, event)
	
	if baseEvent, ok := event.(*BaseEvent); ok {
		cloned := baseEvent.Clone()
		newEvent := cloned.(*BaseEvent)
		newEvent.SetMetadata("validated", err == nil)
		if err != nil {
			newEvent.SetMetadata("validation_error", err.Error())
		}
		return newEvent, nil
	}
	
	return event, err
}

// matchesPattern performs simple pattern matching with wildcards
func matchesPattern(pattern, text string) bool {
	if pattern == "*" {
		return true
	}
	
	if pattern == text {
		return true
	}
	
	// Simple wildcard matching - only supports * at the end
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(text) >= len(prefix) && text[:len(prefix)] == prefix
	}
	
	return false
}

// ChainPipeline creates a pipeline that chains multiple pipelines
func ChainPipeline(pipelines ...Pipeline) Pipeline {
	return &chainPipeline{pipelines: pipelines}
}

type chainPipeline struct {
	pipelines []Pipeline
}

func (cp *chainPipeline) AddFilter(filter Filter) {
	// Add to the last pipeline
	if len(cp.pipelines) > 0 {
		cp.pipelines[len(cp.pipelines)-1].AddFilter(filter)
	}
}

func (cp *chainPipeline) AddTransformer(transformer Transformer) {
	// Add to the last pipeline
	if len(cp.pipelines) > 0 {
		cp.pipelines[len(cp.pipelines)-1].AddTransformer(transformer)
	}
}

func (cp *chainPipeline) Process(ctx context.Context, event Event) (Event, error) {
	currentEvent := event
	
	for _, pipeline := range cp.pipelines {
		processed, err := pipeline.Process(ctx, currentEvent)
		if err != nil {
			return nil, err
		}
		currentEvent = processed
	}
	
	return currentEvent, nil
}