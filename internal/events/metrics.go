package events

import (
	"sync"
	"time"
)

// SimpleMetrics provides basic metrics collection for events
type SimpleMetrics struct {
	events     map[string]*EventMetrics
	listeners  map[string]*ListenerMetrics
	errors     map[string]int64
	totalCount int64
	mutex      sync.RWMutex
}

// EventMetrics tracks metrics for specific events
type EventMetrics struct {
	Count        int64         `json:"count"`
	TotalTime    int64         `json:"total_time_ns"`
	AverageTime  float64       `json:"average_time_ms"`
	MinTime      int64         `json:"min_time_ns"`
	MaxTime      int64         `json:"max_time_ns"`
	LastFired    time.Time     `json:"last_fired"`
	ErrorCount   int64         `json:"error_count"`
}

// ListenerMetrics tracks metrics for listeners
type ListenerMetrics struct {
	Count        int64     `json:"count"`
	SuccessCount int64     `json:"success_count"`
	ErrorCount   int64     `json:"error_count"`
	TotalTime    int64     `json:"total_time_ns"`
	AverageTime  float64   `json:"average_time_ms"`
	LastRun      time.Time `json:"last_run"`
}

// NewSimpleMetrics creates a new simple metrics instance
func NewSimpleMetrics() *SimpleMetrics {
	return &SimpleMetrics{
		events:    make(map[string]*EventMetrics),
		listeners: make(map[string]*ListenerMetrics),
		errors:    make(map[string]int64),
	}
}

// RecordEvent records metrics for an event
func (sm *SimpleMetrics) RecordEvent(eventName string, duration int64) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	metrics := sm.getOrCreateEventMetrics(eventName)
	metrics.Count++
	metrics.TotalTime += duration
	metrics.AverageTime = float64(metrics.TotalTime) / float64(metrics.Count) / 1e6 // Convert to milliseconds
	metrics.LastFired = time.Now()

	if metrics.MinTime == 0 || duration < metrics.MinTime {
		metrics.MinTime = duration
	}
	if duration > metrics.MaxTime {
		metrics.MaxTime = duration
	}

	sm.totalCount++
}

// RecordListener records metrics for a listener
func (sm *SimpleMetrics) RecordListener(listenerName string, duration int64, success bool) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	metrics := sm.getOrCreateListenerMetrics(listenerName)
	metrics.Count++
	metrics.TotalTime += duration
	metrics.AverageTime = float64(metrics.TotalTime) / float64(metrics.Count) / 1e6 // Convert to milliseconds
	metrics.LastRun = time.Now()

	if success {
		metrics.SuccessCount++
	} else {
		metrics.ErrorCount++
	}
}

// RecordError records an error for an event
func (sm *SimpleMetrics) RecordError(eventName string, err error) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sm.errors[eventName]++

	// Also update event metrics
	metrics := sm.getOrCreateEventMetrics(eventName)
	metrics.ErrorCount++
}

// GetStats returns all collected statistics
func (sm *SimpleMetrics) GetStats() map[string]interface{} {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	stats := map[string]interface{}{
		"total_events": sm.totalCount,
		"events":       make(map[string]*EventMetrics),
		"listeners":    make(map[string]*ListenerMetrics),
		"errors":       make(map[string]int64),
	}

	// Copy event metrics
	for name, metrics := range sm.events {
		stats["events"].(map[string]*EventMetrics)[name] = &EventMetrics{
			Count:       metrics.Count,
			TotalTime:   metrics.TotalTime,
			AverageTime: metrics.AverageTime,
			MinTime:     metrics.MinTime,
			MaxTime:     metrics.MaxTime,
			LastFired:   metrics.LastFired,
			ErrorCount:  metrics.ErrorCount,
		}
	}

	// Copy listener metrics
	for name, metrics := range sm.listeners {
		stats["listeners"].(map[string]*ListenerMetrics)[name] = &ListenerMetrics{
			Count:        metrics.Count,
			SuccessCount: metrics.SuccessCount,
			ErrorCount:   metrics.ErrorCount,
			TotalTime:    metrics.TotalTime,
			AverageTime:  metrics.AverageTime,
			LastRun:      metrics.LastRun,
		}
	}

	// Copy error counts
	for name, count := range sm.errors {
		stats["errors"].(map[string]int64)[name] = count
	}

	return stats
}

// GetEventStats returns statistics for a specific event
func (sm *SimpleMetrics) GetEventStats(eventName string) *EventMetrics {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	if metrics, exists := sm.events[eventName]; exists {
		return &EventMetrics{
			Count:       metrics.Count,
			TotalTime:   metrics.TotalTime,
			AverageTime: metrics.AverageTime,
			MinTime:     metrics.MinTime,
			MaxTime:     metrics.MaxTime,
			LastFired:   metrics.LastFired,
			ErrorCount:  metrics.ErrorCount,
		}
	}
	return nil
}

// GetListenerStats returns statistics for a specific listener
func (sm *SimpleMetrics) GetListenerStats(listenerName string) *ListenerMetrics {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	if metrics, exists := sm.listeners[listenerName]; exists {
		return &ListenerMetrics{
			Count:        metrics.Count,
			SuccessCount: metrics.SuccessCount,
			ErrorCount:   metrics.ErrorCount,
			TotalTime:    metrics.TotalTime,
			AverageTime:  metrics.AverageTime,
			LastRun:      metrics.LastRun,
		}
	}
	return nil
}

// Reset resets all metrics
func (sm *SimpleMetrics) Reset() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sm.events = make(map[string]*EventMetrics)
	sm.listeners = make(map[string]*ListenerMetrics)
	sm.errors = make(map[string]int64)
	sm.totalCount = 0
}

// GetTopEvents returns the most frequently fired events
func (sm *SimpleMetrics) GetTopEvents(limit int) []string {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	type eventCount struct {
		name  string
		count int64
	}

	events := make([]eventCount, 0, len(sm.events))
	for name, metrics := range sm.events {
		events = append(events, eventCount{name: name, count: metrics.Count})
	}

	// Simple bubble sort for small datasets
	for i := 0; i < len(events)-1; i++ {
		for j := 0; j < len(events)-i-1; j++ {
			if events[j].count < events[j+1].count {
				events[j], events[j+1] = events[j+1], events[j]
			}
		}
	}

	result := make([]string, 0, limit)
	for i, event := range events {
		if i >= limit {
			break
		}
		result = append(result, event.name)
	}

	return result
}

// GetSlowestListeners returns the slowest listeners
func (sm *SimpleMetrics) GetSlowestListeners(limit int) []string {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	type listenerTime struct {
		name        string
		averageTime float64
	}

	listeners := make([]listenerTime, 0, len(sm.listeners))
	for name, metrics := range sm.listeners {
		listeners = append(listeners, listenerTime{name: name, averageTime: metrics.AverageTime})
	}

	// Simple bubble sort for small datasets
	for i := 0; i < len(listeners)-1; i++ {
		for j := 0; j < len(listeners)-i-1; j++ {
			if listeners[j].averageTime < listeners[j+1].averageTime {
				listeners[j], listeners[j+1] = listeners[j+1], listeners[j]
			}
		}
	}

	result := make([]string, 0, limit)
	for i, listener := range listeners {
		if i >= limit {
			break
		}
		result = append(result, listener.name)
	}

	return result
}

// getOrCreateEventMetrics gets or creates event metrics
func (sm *SimpleMetrics) getOrCreateEventMetrics(eventName string) *EventMetrics {
	if metrics, exists := sm.events[eventName]; exists {
		return metrics
	}

	metrics := &EventMetrics{}
	sm.events[eventName] = metrics
	return metrics
}

// getOrCreateListenerMetrics gets or creates listener metrics
func (sm *SimpleMetrics) getOrCreateListenerMetrics(listenerName string) *ListenerMetrics {
	if metrics, exists := sm.listeners[listenerName]; exists {
		return metrics
	}

	metrics := &ListenerMetrics{}
	sm.listeners[listenerName] = metrics
	return metrics
}

// NoopMetrics provides a no-operation metrics implementation
type NoopMetrics struct{}

// NewNoopMetrics creates a new no-op metrics instance
func NewNoopMetrics() *NoopMetrics {
	return &NoopMetrics{}
}

// RecordEvent does nothing
func (nm *NoopMetrics) RecordEvent(eventName string, duration int64) {}

// RecordListener does nothing
func (nm *NoopMetrics) RecordListener(listenerName string, duration int64, success bool) {}

// RecordError does nothing
func (nm *NoopMetrics) RecordError(eventName string, err error) {}

// GetStats returns empty stats
func (nm *NoopMetrics) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"total_events": 0,
		"events":       map[string]*EventMetrics{},
		"listeners":    map[string]*ListenerMetrics{},
		"errors":       map[string]int64{},
	}
}