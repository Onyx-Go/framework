package mail

import (
	"fmt"
	"strings"
	"time"
)

// Address represents an email address with optional name
type Address struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

// String returns the formatted address string
func (a Address) String() string {
	if a.Name != "" {
		return fmt.Sprintf("%s <%s>", a.Name, a.Email)
	}
	return a.Email
}

// Validate checks if the address is valid
func (a Address) Validate() error {
	if a.Email == "" {
		return fmt.Errorf("email address is required")
	}
	
	// Basic email validation
	if !strings.Contains(a.Email, "@") {
		return fmt.Errorf("invalid email address: %s", a.Email)
	}
	
	parts := strings.Split(a.Email, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("invalid email address: %s", a.Email)
	}
	
	return nil
}

// Envelope contains the message addressing and metadata
type Envelope struct {
	// Addressing
	From    *Address   `json:"from"`
	To      []*Address `json:"to"`
	CC      []*Address `json:"cc"`
	BCC     []*Address `json:"bcc"`
	ReplyTo []*Address `json:"reply_to"`
	
	// Message metadata
	Subject  string `json:"subject"`
	Priority int    `json:"priority"` // 1=high, 3=normal, 5=low
	
	// Headers and tracking
	Headers     map[string]string `json:"headers"`
	Tags        []string          `json:"tags"`
	Metadata    map[string]string `json:"metadata"`
	TrackingID  string            `json:"tracking_id"`
	
	// Delivery options
	DeliveryReceipt bool `json:"delivery_receipt"`
	ReadReceipt     bool `json:"read_receipt"`
}

// Validate checks if the envelope is valid
func (e *Envelope) Validate() error {
	if e.From == nil {
		return fmt.Errorf("from address is required")
	}
	
	if err := e.From.Validate(); err != nil {
		return fmt.Errorf("invalid from address: %w", err)
	}
	
	if len(e.To) == 0 && len(e.CC) == 0 && len(e.BCC) == 0 {
		return fmt.Errorf("at least one recipient is required")
	}
	
	// Validate all addresses
	for _, addr := range e.To {
		if err := addr.Validate(); err != nil {
			return fmt.Errorf("invalid to address: %w", err)
		}
	}
	
	for _, addr := range e.CC {
		if err := addr.Validate(); err != nil {
			return fmt.Errorf("invalid cc address: %w", err)
		}
	}
	
	for _, addr := range e.BCC {
		if err := addr.Validate(); err != nil {
			return fmt.Errorf("invalid bcc address: %w", err)
		}
	}
	
	for _, addr := range e.ReplyTo {
		if err := addr.Validate(); err != nil {
			return fmt.Errorf("invalid reply-to address: %w", err)
		}
	}
	
	if e.Subject == "" {
		return fmt.Errorf("subject is required")
	}
	
	return nil
}

// GetAllRecipients returns all recipient addresses
func (e *Envelope) GetAllRecipients() []*Address {
	var recipients []*Address
	recipients = append(recipients, e.To...)
	recipients = append(recipients, e.CC...)
	recipients = append(recipients, e.BCC...)
	return recipients
}

// GetRecipientEmails returns all recipient email addresses as strings
func (e *Envelope) GetRecipientEmails() []string {
	recipients := e.GetAllRecipients()
	emails := make([]string, len(recipients))
	for i, addr := range recipients {
		emails[i] = addr.Email
	}
	return emails
}

// Content contains the message body and template information
type Content struct {
	// Direct content
	Text string `json:"text"`
	HTML string `json:"html"`
	
	// Template-based content
	View         string                 `json:"view"`
	TemplateData map[string]interface{} `json:"template_data"`
	
	// Rendered content (populated after template processing)
	RenderedText string `json:"rendered_text"`
	RenderedHTML string `json:"rendered_html"`
	
	// Content encoding
	Charset  string `json:"charset"`
	Encoding string `json:"encoding"`
}

// HasContent returns true if the content has text or HTML
func (c *Content) HasContent() bool {
	return c.Text != "" || c.HTML != "" || c.RenderedText != "" || c.RenderedHTML != ""
}

// HasTemplate returns true if the content uses a template
func (c *Content) HasTemplate() bool {
	return c.View != ""
}

// Attachment represents a file attachment
type Attachment struct {
	// Basic attachment info
	Name        string `json:"name"`
	ContentType string `json:"content_type"`
	Disposition string `json:"disposition"` // attachment, inline
	
	// Content (either raw data or file path)
	Content  []byte `json:"content,omitempty"`
	FilePath string `json:"file_path,omitempty"`
	
	// Additional metadata
	Headers map[string]string `json:"headers"`
	Size    int64             `json:"size"`
	
	// For inline attachments
	ContentID string `json:"content_id,omitempty"`
}

// IsInline returns true if this is an inline attachment
func (a *Attachment) IsInline() bool {
	return a.Disposition == "inline" || a.ContentID != ""
}

// Validate checks if the attachment is valid
func (a *Attachment) Validate() error {
	if a.Name == "" {
		return fmt.Errorf("attachment name is required")
	}
	
	if len(a.Content) == 0 && a.FilePath == "" {
		return fmt.Errorf("attachment must have either content or file path")
	}
	
	if len(a.Content) > 0 && a.FilePath != "" {
		return fmt.Errorf("attachment cannot have both content and file path")
	}
	
	return nil
}

// Message represents a complete email message
type Message struct {
	// Core components
	Envelope    *Envelope     `json:"envelope"`
	Content     *Content      `json:"content"`
	Attachments []*Attachment `json:"attachments"`
	
	// Message metadata
	ID        string    `json:"id"`
	MessageID string    `json:"message_id"`
	Date      time.Time `json:"date"`
	
	// Delivery tracking
	DeliveryStatus string    `json:"delivery_status"`
	DeliveredAt    time.Time `json:"delivered_at,omitempty"`
	
	// Error information
	LastError    string    `json:"last_error,omitempty"`
	ErrorCount   int       `json:"error_count"`
	LastAttempt  time.Time `json:"last_attempt,omitempty"`
	
	// Processing metadata
	Driver       string                 `json:"driver,omitempty"`
	ProcessingTime time.Duration        `json:"processing_time,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// Validate checks if the message is valid and ready to send
func (m *Message) Validate() error {
	if m.Envelope == nil {
		return fmt.Errorf("message envelope is required")
	}
	
	if err := m.Envelope.Validate(); err != nil {
		return fmt.Errorf("invalid envelope: %w", err)
	}
	
	if m.Content == nil {
		return fmt.Errorf("message content is required")
	}
	
	if !m.Content.HasContent() && !m.Content.HasTemplate() {
		return fmt.Errorf("message must have either content or template")
	}
	
	// Validate attachments
	for i, attachment := range m.Attachments {
		if err := attachment.Validate(); err != nil {
			return fmt.Errorf("invalid attachment %d: %w", i, err)
		}
	}
	
	return nil
}

// HasAttachments returns true if the message has attachments
func (m *Message) HasAttachments() bool {
	return len(m.Attachments) > 0
}

// HasInlineAttachments returns true if the message has inline attachments
func (m *Message) HasInlineAttachments() bool {
	for _, attachment := range m.Attachments {
		if attachment.IsInline() {
			return true
		}
	}
	return false
}

// GetSize returns the approximate size of the message in bytes
func (m *Message) GetSize() int64 {
	var size int64
	
	// Content size
	if m.Content != nil {
		size += int64(len(m.Content.Text))
		size += int64(len(m.Content.HTML))
		size += int64(len(m.Content.RenderedText))
		size += int64(len(m.Content.RenderedHTML))
	}
	
	// Attachment size
	for _, attachment := range m.Attachments {
		if attachment.Size > 0 {
			size += attachment.Size
		} else {
			size += int64(len(attachment.Content))
		}
	}
	
	// Envelope overhead (approximate)
	if m.Envelope != nil {
		size += int64(len(m.Envelope.Subject) * 2) // Subject can be encoded
		size += int64(len(m.Envelope.GetRecipientEmails()) * 50) // Approximate per recipient
	}
	
	return size
}

// Clone creates a deep copy of the message
func (m *Message) Clone() *Message {
	clone := &Message{
		ID:             m.ID,
		MessageID:      m.MessageID,
		Date:           m.Date,
		DeliveryStatus: m.DeliveryStatus,
		DeliveredAt:    m.DeliveredAt,
		LastError:      m.LastError,
		ErrorCount:     m.ErrorCount,
		LastAttempt:    m.LastAttempt,
		Driver:         m.Driver,
		ProcessingTime: m.ProcessingTime,
	}
	
	// Clone envelope
	if m.Envelope != nil {
		clone.Envelope = &Envelope{
			Subject:         m.Envelope.Subject,
			Priority:        m.Envelope.Priority,
			TrackingID:      m.Envelope.TrackingID,
			DeliveryReceipt: m.Envelope.DeliveryReceipt,
			ReadReceipt:     m.Envelope.ReadReceipt,
		}
		
		// Clone addresses
		if m.Envelope.From != nil {
			clone.Envelope.From = &Address{
				Email: m.Envelope.From.Email,
				Name:  m.Envelope.From.Name,
			}
		}
		
		clone.Envelope.To = cloneAddresses(m.Envelope.To)
		clone.Envelope.CC = cloneAddresses(m.Envelope.CC)
		clone.Envelope.BCC = cloneAddresses(m.Envelope.BCC)
		clone.Envelope.ReplyTo = cloneAddresses(m.Envelope.ReplyTo)
		
		// Clone maps and slices
		clone.Envelope.Headers = cloneStringMap(m.Envelope.Headers)
		clone.Envelope.Metadata = cloneStringMap(m.Envelope.Metadata)
		clone.Envelope.Tags = cloneStringSlice(m.Envelope.Tags)
	}
	
	// Clone content
	if m.Content != nil {
		clone.Content = &Content{
			Text:         m.Content.Text,
			HTML:         m.Content.HTML,
			View:         m.Content.View,
			RenderedText: m.Content.RenderedText,
			RenderedHTML: m.Content.RenderedHTML,
			Charset:      m.Content.Charset,
			Encoding:     m.Content.Encoding,
		}
		
		// Clone template data
		clone.Content.TemplateData = make(map[string]interface{})
		for k, v := range m.Content.TemplateData {
			clone.Content.TemplateData[k] = v
		}
	}
	
	// Clone attachments
	clone.Attachments = make([]*Attachment, len(m.Attachments))
	for i, att := range m.Attachments {
		clone.Attachments[i] = &Attachment{
			Name:        att.Name,
			ContentType: att.ContentType,
			Disposition: att.Disposition,
			FilePath:    att.FilePath,
			Size:        att.Size,
			ContentID:   att.ContentID,
			Headers:     cloneStringMap(att.Headers),
		}
		
		// Clone content
		if len(att.Content) > 0 {
			clone.Attachments[i].Content = make([]byte, len(att.Content))
			copy(clone.Attachments[i].Content, att.Content)
		}
	}
	
	// Clone metadata
	if m.Metadata != nil {
		clone.Metadata = make(map[string]interface{})
		for k, v := range m.Metadata {
			clone.Metadata[k] = v
		}
	}
	
	return clone
}

// Helper functions for cloning

func cloneAddresses(addresses []*Address) []*Address {
	if addresses == nil {
		return nil
	}
	
	clone := make([]*Address, len(addresses))
	for i, addr := range addresses {
		clone[i] = &Address{
			Email: addr.Email,
			Name:  addr.Name,
		}
	}
	return clone
}

func cloneStringMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	
	clone := make(map[string]string)
	for k, v := range m {
		clone[k] = v
	}
	return clone
}

func cloneStringSlice(s []string) []string {
	if s == nil {
		return nil
	}
	
	clone := make([]string, len(s))
	copy(clone, s)
	return clone
}

// Priority constants
const (
	PriorityHigh   = 1
	PriorityNormal = 3
	PriorityLow    = 5
)

// Delivery status constants
const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusSent       = "sent"
	StatusDelivered  = "delivered"
	StatusFailed     = "failed"
	StatusBounced    = "bounced"
)