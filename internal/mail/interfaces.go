package mail

import (
	"context"
	"time"
)

// Driver defines the interface for mail delivery drivers
type Driver interface {
	// Send delivers a message using the driver
	Send(ctx context.Context, message *Message) error
	
	// GetName returns the driver name
	GetName() string
	
	// Validate checks if the driver configuration is valid
	Validate() error
	
	// Close shuts down the driver and releases resources
	Close(ctx context.Context) error
	
	// HealthCheck verifies the driver can connect to its service
	HealthCheck(ctx context.Context) error
	
	// GetCapabilities returns what features this driver supports
	GetCapabilities() Capabilities
}

// Mailable defines the interface for objects that can be sent as email
type Mailable interface {
	// Envelope returns the message envelope (addressing, subject, headers)
	Envelope() *Envelope
	
	// Content returns the message content (body, templates, data)
	Content() *Content
	
	// Attachments returns the message attachments
	Attachments() []*Attachment
	
	// GetData returns the data to be passed to templates
	GetData() map[string]interface{}
	
	// BeforeSend is called before the message is sent (hook for modifications)
	BeforeSend(ctx context.Context) error
	
	// AfterSend is called after the message is sent (hook for logging, etc.)
	AfterSend(ctx context.Context, err error) error
}

// Manager handles mail configuration, drivers, and sending
type Manager interface {
	// Driver management
	RegisterDriver(name string, driver Driver) error
	GetDriver(name string) (Driver, error)
	SetDefaultDriver(name string)
	GetDefaultDriver() string
	
	// Message sending
	Send(ctx context.Context, mailable Mailable, driverName ...string) error
	SendNow(ctx context.Context, mailable Mailable, driverName ...string) error
	Queue(ctx context.Context, mailable Mailable, driverName ...string) error
	
	// Bulk operations
	SendBulk(ctx context.Context, mailables []Mailable, driverName ...string) error
	
	// Configuration
	GetConfig() *Config
	UpdateConfig(config *Config) error
	
	// Health and monitoring
	HealthCheck(ctx context.Context, driverName ...string) error
	GetStats(ctx context.Context) (*Stats, error)
	
	// Lifecycle
	Close(ctx context.Context) error
}

// Renderer handles template rendering for email content
type Renderer interface {
	// Render processes a template with data and returns the result
	Render(ctx context.Context, template string, data map[string]interface{}) (string, error)
	
	// RenderHTML renders an HTML template
	RenderHTML(ctx context.Context, template string, data map[string]interface{}) (string, error)
	
	// RenderText renders a text template
	RenderText(ctx context.Context, template string, data map[string]interface{}) (string, error)
	
	// SetTemplateDir sets the base directory for templates
	SetTemplateDir(dir string)
	
	// GetTemplateDir returns the current template directory
	GetTemplateDir() string
	
	// AddFunction adds a custom function to the template engine
	AddFunction(name string, fn interface{})
}

// Queue defines the interface for queuing emails for later delivery
type Queue interface {
	// Push adds a message to the queue
	Push(ctx context.Context, mailable Mailable, delay time.Duration) error
	
	// Process processes queued messages
	Process(ctx context.Context) error
	
	// Failed handles failed message delivery
	Failed(ctx context.Context, mailable Mailable, err error) error
	
	// Retry attempts to resend a failed message
	Retry(ctx context.Context, messageID string) error
	
	// GetPending returns pending messages in the queue
	GetPending(ctx context.Context) ([]QueuedMessage, error)
	
	// GetFailed returns failed messages
	GetFailed(ctx context.Context) ([]QueuedMessage, error)
	
	// Clear removes all messages from the queue
	Clear(ctx context.Context) error
}

// Logger defines the interface for mail-specific logging
type Logger interface {
	// LogSent logs a successfully sent message
	LogSent(ctx context.Context, message *Message, driver string, duration time.Duration)
	
	// LogFailed logs a failed message delivery
	LogFailed(ctx context.Context, message *Message, driver string, err error, duration time.Duration)
	
	// LogQueued logs a message being queued
	LogQueued(ctx context.Context, mailable Mailable, delay time.Duration)
	
	// LogRetry logs a retry attempt
	LogRetry(ctx context.Context, messageID string, attempt int, err error)
}

// Middleware defines the interface for mail middleware
type Middleware interface {
	// Handle processes the mailable before/after sending
	Handle(ctx context.Context, mailable Mailable, next func(ctx context.Context, mailable Mailable) error) error
}

// Config holds mail system configuration
type Config struct {
	// Default driver to use
	DefaultDriver string `json:"default_driver"`
	
	// Global from address
	From Address `json:"from"`
	
	// Driver configurations
	Drivers map[string]DriverConfig `json:"drivers"`
	
	// Template configuration
	Templates TemplateConfig `json:"templates"`
	
	// Queue configuration
	Queue QueueConfig `json:"queue"`
	
	// Retry configuration
	Retry RetryConfig `json:"retry"`
	
	// Rate limiting
	RateLimit RateLimitConfig `json:"rate_limit"`
	
	// Logging configuration
	Logging LoggingConfig `json:"logging"`
}

// DriverConfig holds configuration for a mail driver
type DriverConfig struct {
	// Driver type (smtp, sendmail, log, etc.)
	Type string `json:"type"`
	
	// Connection settings
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	Encryption string `json:"encryption"` // tls, ssl, starttls
	
	// Timeouts
	Timeout        time.Duration `json:"timeout"`
	ConnectTimeout time.Duration `json:"connect_timeout"`
	SendTimeout    time.Duration `json:"send_timeout"`
	
	// Connection pooling
	MaxConnections int           `json:"max_connections"`
	IdleTimeout    time.Duration `json:"idle_timeout"`
	
	// Additional options
	Options map[string]interface{} `json:"options"`
}

// TemplateConfig holds template rendering configuration
type TemplateConfig struct {
	// Base directory for templates
	Directory string `json:"directory"`
	
	// Template file extensions
	HTMLExtension string `json:"html_extension"`
	TextExtension string `json:"text_extension"`
	
	// Template caching
	Cache   bool          `json:"cache"`
	CacheTTL time.Duration `json:"cache_ttl"`
	
	// Custom functions
	Functions map[string]interface{} `json:"functions"`
}

// QueueConfig holds queue configuration
type QueueConfig struct {
	// Enable queuing
	Enabled bool `json:"enabled"`
	
	// Queue driver (memory, redis, database)
	Driver string `json:"driver"`
	
	// Queue name
	Name string `json:"name"`
	
	// Worker configuration
	Workers    int           `json:"workers"`
	BatchSize  int           `json:"batch_size"`
	PollInterval time.Duration `json:"poll_interval"`
	
	// Connection string for queue driver
	Connection string `json:"connection"`
}

// RetryConfig holds retry configuration
type RetryConfig struct {
	// Enable retries
	Enabled bool `json:"enabled"`
	
	// Maximum retry attempts
	MaxAttempts int `json:"max_attempts"`
	
	// Delay between retries
	InitialDelay time.Duration `json:"initial_delay"`
	MaxDelay     time.Duration `json:"max_delay"`
	Multiplier   float64       `json:"multiplier"`
	
	// Exponential backoff
	ExponentialBackoff bool `json:"exponential_backoff"`
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	// Enable rate limiting
	Enabled bool `json:"enabled"`
	
	// Maximum messages per period
	MaxMessages int `json:"max_messages"`
	
	// Period for rate limiting
	Period time.Duration `json:"period"`
	
	// Per-driver rate limits
	DriverLimits map[string]RateLimit `json:"driver_limits"`
}

// RateLimit defines rate limiting parameters
type RateLimit struct {
	MaxMessages int           `json:"max_messages"`
	Period      time.Duration `json:"period"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	// Enable logging
	Enabled bool `json:"enabled"`
	
	// Log level (debug, info, warn, error)
	Level string `json:"level"`
	
	// Log sent messages
	LogSent bool `json:"log_sent"`
	
	// Log failed messages
	LogFailed bool `json:"log_failed"`
	
	// Log queued messages
	LogQueued bool `json:"log_queued"`
	
	// Include message content in logs
	IncludeContent bool `json:"include_content"`
	
	// Maximum content length to log
	MaxContentLength int `json:"max_content_length"`
}

// Capabilities defines what features a driver supports
type Capabilities struct {
	// HTML content support
	HTML bool `json:"html"`
	
	// Plain text support
	Text bool `json:"text"`
	
	// Attachment support
	Attachments bool `json:"attachments"`
	
	// Inline images support
	InlineImages bool `json:"inline_images"`
	
	// Multiple recipients support
	MultipleRecipients bool `json:"multiple_recipients"`
	
	// CC/BCC support
	CCAndBCC bool `json:"cc_and_bcc"`
	
	// Custom headers support
	CustomHeaders bool `json:"custom_headers"`
	
	// Delivery status tracking
	DeliveryTracking bool `json:"delivery_tracking"`
	
	// Read receipt support
	ReadReceipts bool `json:"read_receipts"`
	
	// Priority support
	Priority bool `json:"priority"`
	
	// Encryption support
	Encryption bool `json:"encryption"`
}

// Stats holds mail system statistics
type Stats struct {
	// Message counts
	SentCount   int64 `json:"sent_count"`
	FailedCount int64 `json:"failed_count"`
	QueuedCount int64 `json:"queued_count"`
	
	// Driver-specific stats
	DriverStats map[string]DriverStats `json:"driver_stats"`
	
	// Performance metrics
	AverageDeliveryTime time.Duration `json:"average_delivery_time"`
	SuccessRate         float64       `json:"success_rate"`
	
	// Queue stats
	QueueSize    int64 `json:"queue_size"`
	PendingCount int64 `json:"pending_count"`
	
	// Rate limiting stats
	RateLimitHits int64 `json:"rate_limit_hits"`
	
	// Collection period
	Period    time.Duration `json:"period"`
	UpdatedAt time.Time     `json:"updated_at"`
}

// DriverStats holds statistics for a specific driver
type DriverStats struct {
	Name         string        `json:"name"`
	SentCount    int64         `json:"sent_count"`
	FailedCount  int64         `json:"failed_count"`
	AverageTime  time.Duration `json:"average_time"`
	LastUsed     time.Time     `json:"last_used"`
	HealthStatus string        `json:"health_status"`
}

// QueuedMessage represents a message in the queue
type QueuedMessage struct {
	ID          string    `json:"id"`
	Mailable    Mailable  `json:"mailable"`
	Driver      string    `json:"driver"`
	Attempts    int       `json:"attempts"`
	MaxAttempts int       `json:"max_attempts"`
	QueuedAt    time.Time `json:"queued_at"`
	ScheduledAt time.Time `json:"scheduled_at"`
	LastAttempt time.Time `json:"last_attempt"`
	LastError   string    `json:"last_error"`
	Status      string    `json:"status"` // pending, processing, failed, completed
}

// DefaultConfig returns a default mail configuration
func DefaultConfig() *Config {
	return &Config{
		DefaultDriver: "smtp",
		From: Address{
			Email: "noreply@example.com",
			Name:  "Onyx Application",
		},
		Drivers: map[string]DriverConfig{
			"smtp": {
				Type:           "smtp",
				Host:           "localhost",
				Port:           587,
				Encryption:     "tls",
				Timeout:        30 * time.Second,
				ConnectTimeout: 10 * time.Second,
				SendTimeout:    30 * time.Second,
				MaxConnections: 10,
				IdleTimeout:    5 * time.Minute,
				Options:        make(map[string]interface{}),
			},
			"log": {
				Type:    "log",
				Options: make(map[string]interface{}),
			},
		},
		Templates: TemplateConfig{
			Directory:     "resources/views/emails",
			HTMLExtension: ".html",
			TextExtension: ".txt",
			Cache:         true,
			CacheTTL:      10 * time.Minute,
			Functions:     make(map[string]interface{}),
		},
		Queue: QueueConfig{
			Enabled:      false,
			Driver:       "memory",
			Name:         "emails",
			Workers:      1,
			BatchSize:    10,
			PollInterval: 1 * time.Second,
		},
		Retry: RetryConfig{
			Enabled:            true,
			MaxAttempts:        3,
			InitialDelay:       1 * time.Second,
			MaxDelay:           5 * time.Minute,
			Multiplier:         2.0,
			ExponentialBackoff: true,
		},
		RateLimit: RateLimitConfig{
			Enabled:      false,
			MaxMessages:  100,
			Period:       1 * time.Hour,
			DriverLimits: make(map[string]RateLimit),
		},
		Logging: LoggingConfig{
			Enabled:          true,
			Level:            "info",
			LogSent:          true,
			LogFailed:        true,
			LogQueued:        true,
			IncludeContent:   false,
			MaxContentLength: 1000,
		},
	}
}