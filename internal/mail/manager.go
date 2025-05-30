package mail

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DefaultManager implements the Manager interface
type DefaultManager struct {
	config         *Config
	drivers        map[string]Driver
	renderer       Renderer
	queue          Queue
	logger         Logger
	middleware     []Middleware
	stats          *StatsCollector
	rateLimiters   map[string]*RateLimiter
	defaultDriver  string
	mutex          sync.RWMutex
}

// NewManager creates a new mail manager
func NewManager(config *Config) *DefaultManager {
	return &DefaultManager{
		config:       config,
		drivers:      make(map[string]Driver),
		middleware:   make([]Middleware, 0),
		rateLimiters: make(map[string]*RateLimiter),
		defaultDriver: config.DefaultDriver,
		stats:        NewStatsCollector(),
	}
}

// Driver management

// RegisterDriver registers a mail driver
func (m *DefaultManager) RegisterDriver(name string, driver Driver) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if err := driver.Validate(); err != nil {
		return fmt.Errorf("invalid driver configuration: %w", err)
	}
	
	m.drivers[name] = driver
	
	// Set up rate limiter if configured
	if m.config.RateLimit.Enabled {
		if driverLimit, exists := m.config.RateLimit.DriverLimits[name]; exists {
			m.rateLimiters[name] = NewRateLimiter(driverLimit.MaxMessages, driverLimit.Period)
		} else {
			m.rateLimiters[name] = NewRateLimiter(m.config.RateLimit.MaxMessages, m.config.RateLimit.Period)
		}
	}
	
	return nil
}

// GetDriver returns a driver by name
func (m *DefaultManager) GetDriver(name string) (Driver, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	if name == "" {
		name = m.defaultDriver
	}
	
	driver, exists := m.drivers[name]
	if !exists {
		return nil, fmt.Errorf("mail driver '%s' not found", name)
	}
	
	return driver, nil
}

// SetDefaultDriver sets the default driver
func (m *DefaultManager) SetDefaultDriver(name string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.defaultDriver = name
}

// GetDefaultDriver returns the default driver name
func (m *DefaultManager) GetDefaultDriver() string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.defaultDriver
}

// Message sending

// Send sends a message using the specified or default driver
func (m *DefaultManager) Send(ctx context.Context, mailable Mailable, driverName ...string) error {
	startTime := time.Now()
	
	// Get driver
	driver, err := m.getDriverToUse(driverName...)
	if err != nil {
		return err
	}
	
	driverNameStr := driver.GetName()
	
	// Check rate limiting
	if m.config.RateLimit.Enabled {
		if rateLimiter, exists := m.rateLimiters[driverNameStr]; exists {
			if !rateLimiter.Allow() {
				m.stats.RecordRateLimitHit(driverNameStr)
				return fmt.Errorf("rate limit exceeded for driver %s", driverNameStr)
			}
		}
	}
	
	// Build message
	message, err := m.buildMessage(ctx, mailable)
	if err != nil {
		m.stats.RecordFailure(driverNameStr, time.Since(startTime))
		return fmt.Errorf("failed to build message: %w", err)
	}
	
	// Execute middleware chain
	err = m.executeMiddleware(ctx, mailable, func(ctx context.Context, mailable Mailable) error {
		return m.sendMessage(ctx, driver, message, mailable)
	})
	
	duration := time.Since(startTime)
	
	if err != nil {
		m.stats.RecordFailure(driverNameStr, duration)
		if m.logger != nil {
			m.logger.LogFailed(ctx, message, driverNameStr, err, duration)
		}
		
		// Handle retry logic
		if m.config.Retry.Enabled && m.shouldRetry(message, err) {
			return m.retryMessage(ctx, mailable, driverNameStr, err)
		}
		
		return err
	}
	
	m.stats.RecordSuccess(driverNameStr, duration)
	if m.logger != nil {
		m.logger.LogSent(ctx, message, driverNameStr, duration)
	}
	
	return nil
}

// SendNow sends a message immediately without queuing
func (m *DefaultManager) SendNow(ctx context.Context, mailable Mailable, driverName ...string) error {
	// This is the same as Send for non-queued operation
	return m.Send(ctx, mailable, driverName...)
}

// Queue queues a message for later delivery
func (m *DefaultManager) Queue(ctx context.Context, mailable Mailable, driverName ...string) error {
	if !m.config.Queue.Enabled || m.queue == nil {
		return fmt.Errorf("queuing is not enabled")
	}
	
	if m.logger != nil {
		m.logger.LogQueued(ctx, mailable, 0)
	}
	
	return m.queue.Push(ctx, mailable, 0)
}

// SendBulk sends multiple messages
func (m *DefaultManager) SendBulk(ctx context.Context, mailables []Mailable, driverName ...string) error {
	var firstError error
	successCount := 0
	
	for _, mailable := range mailables {
		if err := m.Send(ctx, mailable, driverName...); err != nil {
			if firstError == nil {
				firstError = err
			}
		} else {
			successCount++
		}
	}
	
	if firstError != nil {
		return fmt.Errorf("bulk send completed with %d successes out of %d total: %w", 
			successCount, len(mailables), firstError)
	}
	
	return nil
}

// Configuration

// GetConfig returns the current configuration
func (m *DefaultManager) GetConfig() *Config {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.config
}

// UpdateConfig updates the configuration
func (m *DefaultManager) UpdateConfig(config *Config) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.config = config
	m.defaultDriver = config.DefaultDriver
	
	// Update rate limiters
	if config.RateLimit.Enabled {
		for name := range m.drivers {
			if driverLimit, exists := config.RateLimit.DriverLimits[name]; exists {
				m.rateLimiters[name] = NewRateLimiter(driverLimit.MaxMessages, driverLimit.Period)
			} else {
				m.rateLimiters[name] = NewRateLimiter(config.RateLimit.MaxMessages, config.RateLimit.Period)
			}
		}
	} else {
		m.rateLimiters = make(map[string]*RateLimiter)
	}
	
	return nil
}

// Health and monitoring

// HealthCheck checks the health of drivers
func (m *DefaultManager) HealthCheck(ctx context.Context, driverName ...string) error {
	if len(driverName) > 0 {
		// Check specific driver
		driver, err := m.GetDriver(driverName[0])
		if err != nil {
			return err
		}
		return driver.HealthCheck(ctx)
	}
	
	// Check all drivers
	m.mutex.RLock()
	drivers := make(map[string]Driver)
	for name, driver := range m.drivers {
		drivers[name] = driver
	}
	m.mutex.RUnlock()
	
	var firstError error
	for name, driver := range drivers {
		if err := driver.HealthCheck(ctx); err != nil {
			if firstError == nil {
				firstError = fmt.Errorf("driver %s health check failed: %w", name, err)
			}
		}
	}
	
	return firstError
}

// GetStats returns mail statistics
func (m *DefaultManager) GetStats(ctx context.Context) (*Stats, error) {
	return m.stats.GetStats(), nil
}

// Lifecycle

// Close closes the manager and all drivers
func (m *DefaultManager) Close(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	var firstError error
	
	// Close all drivers
	for name, driver := range m.drivers {
		if err := driver.Close(ctx); err != nil {
			if firstError == nil {
				firstError = fmt.Errorf("error closing driver %s: %w", name, err)
			}
		}
	}
	
	// Close queue if present
	if m.queue != nil {
		if err := m.queue.Clear(ctx); err != nil {
			if firstError == nil {
				firstError = fmt.Errorf("error closing queue: %w", err)
			}
		}
	}
	
	return firstError
}

// Component setters

// SetRenderer sets the template renderer
func (m *DefaultManager) SetRenderer(renderer Renderer) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.renderer = renderer
}

// SetQueue sets the message queue
func (m *DefaultManager) SetQueue(queue Queue) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.queue = queue
}

// SetLogger sets the mail logger
func (m *DefaultManager) SetLogger(logger Logger) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.logger = logger
}

// AddMiddleware adds middleware to the execution chain
func (m *DefaultManager) AddMiddleware(middleware Middleware) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.middleware = append(m.middleware, middleware)
}

// Private methods

// getDriverToUse returns the driver to use for sending
func (m *DefaultManager) getDriverToUse(driverName ...string) (Driver, error) {
	var name string
	if len(driverName) > 0 {
		name = driverName[0]
	}
	return m.GetDriver(name)
}

// buildMessage creates a Message from a Mailable
func (m *DefaultManager) buildMessage(ctx context.Context, mailable Mailable) (*Message, error) {
	// Create message
	message := &Message{
		ID:        GenerateMessageID(),
		MessageID: GenerateMessageID(),
		Date:      time.Now(),
		Envelope:  mailable.Envelope(),
		Content:   mailable.Content(),
		Attachments: mailable.Attachments(),
		Metadata:  make(map[string]interface{}),
	}
	
	// Set default from address if not specified
	if message.Envelope.From == nil {
		message.Envelope.From = &m.config.From
	}
	
	// Render templates if needed
	if err := m.renderContent(ctx, message, mailable); err != nil {
		return nil, fmt.Errorf("failed to render content: %w", err)
	}
	
	// Validate message
	if err := message.Validate(); err != nil {
		return nil, fmt.Errorf("invalid message: %w", err)
	}
	
	return message, nil
}

// renderContent renders template content
func (m *DefaultManager) renderContent(ctx context.Context, message *Message, mailable Mailable) error {
	content := message.Content
	
	if content.View != "" && m.renderer != nil {
		data := mailable.GetData()
		
		// Render HTML template
		if html, err := m.renderer.RenderHTML(ctx, content.View, data); err == nil {
			content.RenderedHTML = html
		}
		
		// Render text template
		if text, err := m.renderer.RenderText(ctx, content.View, data); err == nil {
			content.RenderedText = text
		}
	}
	
	// Use direct content if no templates rendered
	if content.RenderedHTML == "" && content.HTML != "" {
		content.RenderedHTML = content.HTML
	}
	
	if content.RenderedText == "" && content.Text != "" {
		content.RenderedText = content.Text
	}
	
	return nil
}

// sendMessage sends a message using a driver
func (m *DefaultManager) sendMessage(ctx context.Context, driver Driver, message *Message, mailable Mailable) error {
	// Call before send hook
	if err := mailable.BeforeSend(ctx); err != nil {
		return fmt.Errorf("before send hook failed: %w", err)
	}
	
	// Send the message
	message.Driver = driver.GetName()
	startTime := time.Now()
	err := driver.Send(ctx, message)
	message.ProcessingTime = time.Since(startTime)
	
	// Update message status
	if err != nil {
		message.DeliveryStatus = StatusFailed
		message.LastError = err.Error()
		message.ErrorCount++
		message.LastAttempt = time.Now()
	} else {
		message.DeliveryStatus = StatusSent
		message.DeliveredAt = time.Now()
	}
	
	// Call after send hook
	hookErr := mailable.AfterSend(ctx, err)
	if hookErr != nil && err == nil {
		// If send succeeded but hook failed, we still report the hook error
		return fmt.Errorf("after send hook failed: %w", hookErr)
	}
	
	return err
}

// executeMiddleware executes the middleware chain
func (m *DefaultManager) executeMiddleware(ctx context.Context, mailable Mailable, final func(ctx context.Context, mailable Mailable) error) error {
	if len(m.middleware) == 0 {
		return final(ctx, mailable)
	}
	
	// Build middleware chain
	handler := final
	for i := len(m.middleware) - 1; i >= 0; i-- {
		middleware := m.middleware[i]
		currentHandler := handler
		handler = func(ctx context.Context, mailable Mailable) error {
			return middleware.Handle(ctx, mailable, currentHandler)
		}
	}
	
	return handler(ctx, mailable)
}

// shouldRetry determines if a message should be retried
func (m *DefaultManager) shouldRetry(message *Message, err error) bool {
	if !m.config.Retry.Enabled {
		return false
	}
	
	if message.ErrorCount >= m.config.Retry.MaxAttempts {
		return false
	}
	
	// Add logic here to determine if specific errors should be retried
	// For now, retry all errors
	return true
}

// retryMessage handles message retry logic
func (m *DefaultManager) retryMessage(ctx context.Context, mailable Mailable, driverName string, err error) error {
	if m.queue == nil {
		return fmt.Errorf("retry failed: queue not available: %w", err)
	}
	
	// Calculate delay
	delay := m.calculateRetryDelay(1) // TODO: get actual attempt number
	
	if m.logger != nil {
		m.logger.LogRetry(ctx, "message-id", 1, err) // TODO: use actual message ID and attempt
	}
	
	// Queue for retry
	return m.queue.Push(ctx, mailable, delay)
}

// calculateRetryDelay calculates the delay for a retry attempt
func (m *DefaultManager) calculateRetryDelay(attempt int) time.Duration {
	if !m.config.Retry.ExponentialBackoff {
		return m.config.Retry.InitialDelay
	}
	
	delay := time.Duration(float64(m.config.Retry.InitialDelay) * 
		(m.config.Retry.Multiplier * float64(attempt)))
	
	if delay > m.config.Retry.MaxDelay {
		delay = m.config.Retry.MaxDelay
	}
	
	return delay
}