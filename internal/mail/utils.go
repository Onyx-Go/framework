package mail

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"
	"time"
)

// GenerateMessageID generates a unique message ID
func GenerateMessageID() string {
	now := time.Now()
	random := fastRand()
	return fmt.Sprintf("<%d.%d.%d@onyx>", now.UnixNano(), now.UnixMicro(), random)
}

// fastRand generates a cryptographically secure random number
func fastRand() int64 {
	max := big.NewInt(1000000) // 1 million possibilities
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		// Fallback to time-based random if crypto/rand fails
		return time.Now().UnixNano() % 1000000
	}
	return n.Int64()
}

// StatsCollector collects and manages mail statistics
type StatsCollector struct {
	stats Stats
	mutex sync.RWMutex
}

// NewStatsCollector creates a new statistics collector
func NewStatsCollector() *StatsCollector {
	return &StatsCollector{
		stats: Stats{
			DriverStats: make(map[string]DriverStats),
			UpdatedAt:   time.Now(),
		},
	}
}

// RecordSuccess records a successful message delivery
func (sc *StatsCollector) RecordSuccess(driverName string, duration time.Duration) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	
	sc.stats.SentCount++
	sc.updateDriverStats(driverName, true, duration)
	sc.updateAverageDeliveryTime(duration)
	sc.stats.UpdatedAt = time.Now()
}

// RecordFailure records a failed message delivery
func (sc *StatsCollector) RecordFailure(driverName string, duration time.Duration) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	
	sc.stats.FailedCount++
	sc.updateDriverStats(driverName, false, duration)
	sc.stats.UpdatedAt = time.Now()
}

// RecordQueued records a queued message
func (sc *StatsCollector) RecordQueued() {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	
	sc.stats.QueuedCount++
	sc.stats.QueueSize++
	sc.stats.PendingCount++
	sc.stats.UpdatedAt = time.Now()
}

// RecordRateLimitHit records a rate limit hit
func (sc *StatsCollector) RecordRateLimitHit(driverName string) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	
	sc.stats.RateLimitHits++
	sc.stats.UpdatedAt = time.Now()
}

// GetStats returns a copy of the current statistics
func (sc *StatsCollector) GetStats() *Stats {
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()
	
	// Create a copy to avoid concurrent access issues
	statsCopy := sc.stats
	statsCopy.DriverStats = make(map[string]DriverStats)
	for name, stats := range sc.stats.DriverStats {
		statsCopy.DriverStats[name] = stats
	}
	
	// Calculate success rate
	total := statsCopy.SentCount + statsCopy.FailedCount
	if total > 0 {
		statsCopy.SuccessRate = float64(statsCopy.SentCount) / float64(total)
	}
	
	return &statsCopy
}

// updateDriverStats updates statistics for a specific driver
func (sc *StatsCollector) updateDriverStats(driverName string, success bool, duration time.Duration) {
	driverStats, exists := sc.stats.DriverStats[driverName]
	if !exists {
		driverStats = DriverStats{
			Name:         driverName,
			HealthStatus: "unknown",
		}
	}
	
	if success {
		driverStats.SentCount++
	} else {
		driverStats.FailedCount++
	}
	
	// Update average time
	totalMessages := driverStats.SentCount + driverStats.FailedCount
	if totalMessages == 1 {
		driverStats.AverageTime = duration
	} else {
		// Calculate running average
		oldAvg := driverStats.AverageTime
		driverStats.AverageTime = time.Duration(
			(int64(oldAvg)*int64(totalMessages-1) + int64(duration)) / int64(totalMessages),
		)
	}
	
	driverStats.LastUsed = time.Now()
	sc.stats.DriverStats[driverName] = driverStats
}

// updateAverageDeliveryTime updates the overall average delivery time
func (sc *StatsCollector) updateAverageDeliveryTime(duration time.Duration) {
	totalMessages := sc.stats.SentCount + sc.stats.FailedCount
	if totalMessages == 1 {
		sc.stats.AverageDeliveryTime = duration
	} else {
		// Calculate running average
		oldAvg := sc.stats.AverageDeliveryTime
		sc.stats.AverageDeliveryTime = time.Duration(
			(int64(oldAvg)*int64(totalMessages-1) + int64(duration)) / int64(totalMessages),
		)
	}
}

// RateLimiter implements a simple rate limiter
type RateLimiter struct {
	maxMessages int
	period      time.Duration
	tokens      int
	lastRefill  time.Time
	mutex       sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxMessages int, period time.Duration) *RateLimiter {
	return &RateLimiter{
		maxMessages: maxMessages,
		period:      period,
		tokens:      maxMessages,
		lastRefill:  time.Now(),
	}
}

// Allow checks if an operation is allowed under the rate limit
func (rl *RateLimiter) Allow() bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	// Refill tokens based on elapsed time
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)
	
	if elapsed >= rl.period {
		// Refill all tokens
		rl.tokens = rl.maxMessages
		rl.lastRefill = now
	} else {
		// Partial refill based on elapsed time
		tokensToAdd := int(float64(rl.maxMessages) * (float64(elapsed) / float64(rl.period)))
		rl.tokens += tokensToAdd
		if rl.tokens > rl.maxMessages {
			rl.tokens = rl.maxMessages
		}
		if tokensToAdd > 0 {
			rl.lastRefill = now
		}
	}
	
	// Check if we have tokens available
	if rl.tokens > 0 {
		rl.tokens--
		return true
	}
	
	return false
}

// GetRemaining returns the number of remaining tokens
func (rl *RateLimiter) GetRemaining() int {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	return rl.tokens
}

// AddressBuilder helps build email addresses fluently
type AddressBuilder struct {
	addresses []*Address
}

// NewAddressBuilder creates a new address builder
func NewAddressBuilder() *AddressBuilder {
	return &AddressBuilder{
		addresses: make([]*Address, 0),
	}
}

// Add adds an address
func (ab *AddressBuilder) Add(email, name string) *AddressBuilder {
	ab.addresses = append(ab.addresses, &Address{Email: email, Name: name})
	return ab
}

// AddAddress adds an Address object
func (ab *AddressBuilder) AddAddress(address Address) *AddressBuilder {
	ab.addresses = append(ab.addresses, &address)
	return ab
}

// Build returns the built addresses
func (ab *AddressBuilder) Build() []*Address {
	return ab.addresses
}

// MessageBuilder provides a fluent interface for building messages
type MessageBuilder struct {
	message *Message
}

// NewMessageBuilder creates a new message builder
func NewMessageBuilder() *MessageBuilder {
	return &MessageBuilder{
		message: &Message{
			ID:        GenerateMessageID(),
			MessageID: GenerateMessageID(),
			Date:      time.Now(),
			Envelope: &Envelope{
				Headers:  make(map[string]string),
				Metadata: make(map[string]string),
				Tags:     make([]string, 0),
				Priority: PriorityNormal,
			},
			Content: &Content{
				TemplateData: make(map[string]interface{}),
				Charset:      "UTF-8",
				Encoding:     "quoted-printable",
			},
			Attachments: make([]*Attachment, 0),
			Metadata:    make(map[string]interface{}),
		},
	}
}

// From sets the sender
func (mb *MessageBuilder) From(email, name string) *MessageBuilder {
	mb.message.Envelope.From = &Address{Email: email, Name: name}
	return mb
}

// To adds a recipient
func (mb *MessageBuilder) To(email, name string) *MessageBuilder {
	mb.message.Envelope.To = append(mb.message.Envelope.To, &Address{Email: email, Name: name})
	return mb
}

// Subject sets the subject
func (mb *MessageBuilder) Subject(subject string) *MessageBuilder {
	mb.message.Envelope.Subject = subject
	return mb
}

// HTML sets the HTML content
func (mb *MessageBuilder) HTML(html string) *MessageBuilder {
	mb.message.Content.HTML = html
	mb.message.Content.RenderedHTML = html
	return mb
}

// Text sets the text content
func (mb *MessageBuilder) Text(text string) *MessageBuilder {
	mb.message.Content.Text = text
	mb.message.Content.RenderedText = text
	return mb
}

// Build returns the built message
func (mb *MessageBuilder) Build() *Message {
	return mb.message
}

// ValidationHelper provides email validation utilities
type ValidationHelper struct{}

// ValidateEmail performs basic email validation
func (vh *ValidationHelper) ValidateEmail(email string) error {
	addr := Address{Email: email}
	return addr.Validate()
}

// ValidateMessage validates a complete message
func (vh *ValidationHelper) ValidateMessage(message *Message) error {
	return message.Validate()
}

// ValidateMailable validates a mailable
func (vh *ValidationHelper) ValidateMailable(mailable Mailable) error {
	if envelope := mailable.Envelope(); envelope != nil {
		if err := envelope.Validate(); err != nil {
			return err
		}
	}
	
	if content := mailable.Content(); content != nil {
		if !content.HasContent() && !content.HasTemplate() {
			return fmt.Errorf("content or template is required")
		}
	}
	
	for i, attachment := range mailable.Attachments() {
		if err := attachment.Validate(); err != nil {
			return fmt.Errorf("invalid attachment %d: %w", i, err)
		}
	}
	
	return nil
}

// TemplateHelper provides template-related utilities
type TemplateHelper struct{}

// ExtractVariables extracts variable names from template content
func (th *TemplateHelper) ExtractVariables(template string) []string {
	// Simplified implementation - in practice you'd parse the template properly
	var variables []string
	// This would extract {{.Variable}} patterns from the template
	return variables
}

// ValidateTemplateData checks if template data contains required variables
func (th *TemplateHelper) ValidateTemplateData(template string, data map[string]interface{}) error {
	variables := th.ExtractVariables(template)
	
	for _, variable := range variables {
		if _, exists := data[variable]; !exists {
			return fmt.Errorf("required template variable '%s' is missing", variable)
		}
	}
	
	return nil
}

// SizeHelper provides utilities for calculating message sizes
type SizeHelper struct{}

// CalculateMessageSize calculates the approximate size of a message
func (sh *SizeHelper) CalculateMessageSize(message *Message) int64 {
	return message.GetSize()
}

// CalculateMailableSize calculates the approximate size of a mailable
func (sh *SizeHelper) CalculateMailableSize(mailable Mailable) int64 {
	var size int64
	
	if content := mailable.Content(); content != nil {
		size += int64(len(content.Text))
		size += int64(len(content.HTML))
		size += int64(len(content.RenderedText))
		size += int64(len(content.RenderedHTML))
	}
	
	for _, attachment := range mailable.Attachments() {
		if attachment.Size > 0 {
			size += attachment.Size
		} else {
			size += int64(len(attachment.Content))
		}
	}
	
	if envelope := mailable.Envelope(); envelope != nil {
		size += int64(len(envelope.Subject) * 2) // Subject can be encoded
		size += int64(len(envelope.GetRecipientEmails()) * 50) // Approximate per recipient
	}
	
	return size
}

// Global instances for convenience
var (
	defaultValidationHelper = &ValidationHelper{}
	defaultTemplateHelper   = &TemplateHelper{}
	defaultSizeHelper       = &SizeHelper{}
)

// Package-level convenience functions

// ValidateEmail validates an email address
func ValidateEmail(email string) error {
	return defaultValidationHelper.ValidateEmail(email)
}

// ValidateMessage validates a message
func ValidateMessage(message *Message) error {
	return defaultValidationHelper.ValidateMessage(message)
}

// ValidateMailable validates a mailable
func ValidateMailable(mailable Mailable) error {
	return defaultValidationHelper.ValidateMailable(mailable)
}

// CalculateSize calculates the size of a message or mailable
func CalculateSize(item interface{}) int64 {
	switch v := item.(type) {
	case *Message:
		return defaultSizeHelper.CalculateMessageSize(v)
	case Mailable:
		return defaultSizeHelper.CalculateMailableSize(v)
	default:
		return 0
	}
}