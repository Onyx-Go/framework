package mail

import (
	"context"
	"fmt"
	"path/filepath"
)

// BaseMailable provides a base implementation of the Mailable interface
type BaseMailable struct {
	envelope    *Envelope
	content     *Content
	attachments []*Attachment
	data        map[string]interface{}
	hooks       MailableHooks
}

// MailableHooks defines callback functions for mailable lifecycle events
type MailableHooks struct {
	BeforeSend func(ctx context.Context, mailable Mailable) error
	AfterSend  func(ctx context.Context, mailable Mailable, err error) error
}

// NewMailable creates a new base mailable
func NewMailable() *BaseMailable {
	return &BaseMailable{
		envelope: &Envelope{
			Headers:  make(map[string]string),
			Metadata: make(map[string]string),
			Tags:     make([]string, 0),
			Priority: PriorityNormal,
		},
		content: &Content{
			TemplateData: make(map[string]interface{}),
			Charset:      "UTF-8",
			Encoding:     "quoted-printable",
		},
		attachments: make([]*Attachment, 0),
		data:        make(map[string]interface{}),
	}
}

// Interface implementations

// Envelope returns the message envelope
func (m *BaseMailable) Envelope() *Envelope {
	return m.envelope
}

// Content returns the message content
func (m *BaseMailable) Content() *Content {
	return m.content
}

// Attachments returns the message attachments
func (m *BaseMailable) Attachments() []*Attachment {
	return m.attachments
}

// GetData returns the template data
func (m *BaseMailable) GetData() map[string]interface{} {
	// Merge content template data with mailable data
	merged := make(map[string]interface{})
	
	// Add mailable data first
	for k, v := range m.data {
		merged[k] = v
	}
	
	// Add content template data (overwrites mailable data if keys conflict)
	for k, v := range m.content.TemplateData {
		merged[k] = v
	}
	
	return merged
}

// BeforeSend is called before the message is sent
func (m *BaseMailable) BeforeSend(ctx context.Context) error {
	if m.hooks.BeforeSend != nil {
		return m.hooks.BeforeSend(ctx, m)
	}
	return nil
}

// AfterSend is called after the message is sent
func (m *BaseMailable) AfterSend(ctx context.Context, err error) error {
	if m.hooks.AfterSend != nil {
		return m.hooks.AfterSend(ctx, m, err)
	}
	return nil
}

// Fluent interface methods for building emails

// From sets the sender address
func (m *BaseMailable) From(email, name string) *BaseMailable {
	m.envelope.From = &Address{Email: email, Name: name}
	return m
}

// To adds a recipient
func (m *BaseMailable) To(email, name string) *BaseMailable {
	m.envelope.To = append(m.envelope.To, &Address{Email: email, Name: name})
	return m
}

// ToMany adds multiple recipients
func (m *BaseMailable) ToMany(addresses []Address) *BaseMailable {
	for _, addr := range addresses {
		m.envelope.To = append(m.envelope.To, &addr)
	}
	return m
}

// CC adds a carbon copy recipient
func (m *BaseMailable) CC(email, name string) *BaseMailable {
	m.envelope.CC = append(m.envelope.CC, &Address{Email: email, Name: name})
	return m
}

// BCC adds a blind carbon copy recipient
func (m *BaseMailable) BCC(email, name string) *BaseMailable {
	m.envelope.BCC = append(m.envelope.BCC, &Address{Email: email, Name: name})
	return m
}

// ReplyTo sets the reply-to address
func (m *BaseMailable) ReplyTo(email, name string) *BaseMailable {
	m.envelope.ReplyTo = append(m.envelope.ReplyTo, &Address{Email: email, Name: name})
	return m
}

// Subject sets the email subject
func (m *BaseMailable) Subject(subject string) *BaseMailable {
	m.envelope.Subject = subject
	return m
}

// Priority sets the message priority
func (m *BaseMailable) Priority(priority int) *BaseMailable {
	m.envelope.Priority = priority
	return m
}

// View sets the template view to use
func (m *BaseMailable) View(view string) *BaseMailable {
	m.content.View = view
	return m
}

// Text sets the plain text content
func (m *BaseMailable) Text(text string) *BaseMailable {
	m.content.Text = text
	return m
}

// HTML sets the HTML content
func (m *BaseMailable) HTML(html string) *BaseMailable {
	m.content.HTML = html
	return m
}

// With adds data for template rendering
func (m *BaseMailable) With(key string, value interface{}) *BaseMailable {
	m.content.TemplateData[key] = value
	return m
}

// WithData adds multiple data values for template rendering
func (m *BaseMailable) WithData(data map[string]interface{}) *BaseMailable {
	for k, v := range data {
		m.content.TemplateData[k] = v
	}
	return m
}

// SetData sets the mailable data (used for hooks and custom logic)
func (m *BaseMailable) SetData(key string, value interface{}) *BaseMailable {
	m.data[key] = value
	return m
}

// Attach adds a file attachment
func (m *BaseMailable) Attach(filePath, name string) *BaseMailable {
	attachment := &Attachment{
		FilePath: filePath,
		Name:     name,
		Headers:  make(map[string]string),
	}
	
	if name == "" {
		attachment.Name = filepath.Base(filePath)
	}
	
	m.attachments = append(m.attachments, attachment)
	return m
}

// AttachData adds an attachment from raw data
func (m *BaseMailable) AttachData(content []byte, name, contentType string) *BaseMailable {
	attachment := &Attachment{
		Content:     content,
		Name:        name,
		ContentType: contentType,
		Headers:     make(map[string]string),
		Size:        int64(len(content)),
	}
	
	m.attachments = append(m.attachments, attachment)
	return m
}

// AttachInline adds an inline attachment (for embedding in HTML)
func (m *BaseMailable) AttachInline(content []byte, name, contentType, contentID string) *BaseMailable {
	attachment := &Attachment{
		Content:     content,
		Name:        name,
		ContentType: contentType,
		Disposition: "inline",
		ContentID:   contentID,
		Headers:     make(map[string]string),
		Size:        int64(len(content)),
	}
	
	m.attachments = append(m.attachments, attachment)
	return m
}

// Header adds a custom header
func (m *BaseMailable) Header(key, value string) *BaseMailable {
	m.envelope.Headers[key] = value
	return m
}

// Tag adds a tag for categorization
func (m *BaseMailable) Tag(tag string) *BaseMailable {
	m.envelope.Tags = append(m.envelope.Tags, tag)
	return m
}

// Tags adds multiple tags
func (m *BaseMailable) Tags(tags []string) *BaseMailable {
	m.envelope.Tags = append(m.envelope.Tags, tags...)
	return m
}

// Meta adds metadata
func (m *BaseMailable) Meta(key, value string) *BaseMailable {
	m.envelope.Metadata[key] = value
	return m
}

// TrackingID sets the tracking ID for delivery tracking
func (m *BaseMailable) TrackingID(id string) *BaseMailable {
	m.envelope.TrackingID = id
	return m
}

// RequestDeliveryReceipt requests a delivery receipt
func (m *BaseMailable) RequestDeliveryReceipt(request bool) *BaseMailable {
	m.envelope.DeliveryReceipt = request
	return m
}

// RequestReadReceipt requests a read receipt
func (m *BaseMailable) RequestReadReceipt(request bool) *BaseMailable {
	m.envelope.ReadReceipt = request
	return m
}

// Charset sets the character encoding
func (m *BaseMailable) Charset(charset string) *BaseMailable {
	m.content.Charset = charset
	return m
}

// Encoding sets the content transfer encoding
func (m *BaseMailable) Encoding(encoding string) *BaseMailable {
	m.content.Encoding = encoding
	return m
}

// Hook methods for lifecycle events

// OnBeforeSend sets a callback to be executed before sending
func (m *BaseMailable) OnBeforeSend(fn func(ctx context.Context, mailable Mailable) error) *BaseMailable {
	m.hooks.BeforeSend = fn
	return m
}

// OnAfterSend sets a callback to be executed after sending
func (m *BaseMailable) OnAfterSend(fn func(ctx context.Context, mailable Mailable, err error) error) *BaseMailable {
	m.hooks.AfterSend = fn
	return m
}

// Utility methods

// IsValid checks if the mailable is valid and ready to send
func (m *BaseMailable) IsValid() error {
	if m.envelope.From == nil {
		return fmt.Errorf("from address is required")
	}
	
	if len(m.envelope.To) == 0 && len(m.envelope.CC) == 0 && len(m.envelope.BCC) == 0 {
		return fmt.Errorf("at least one recipient is required")
	}
	
	if m.envelope.Subject == "" {
		return fmt.Errorf("subject is required")
	}
	
	if !m.content.HasContent() && !m.content.HasTemplate() {
		return fmt.Errorf("content or template is required")
	}
	
	return nil
}

// GetRecipientCount returns the total number of recipients
func (m *BaseMailable) GetRecipientCount() int {
	return len(m.envelope.To) + len(m.envelope.CC) + len(m.envelope.BCC)
}

// HasAttachments returns true if the mailable has attachments
func (m *BaseMailable) HasAttachments() bool {
	return len(m.attachments) > 0
}

// GetEstimatedSize returns the estimated size of the email in bytes
func (m *BaseMailable) GetEstimatedSize() int64 {
	var size int64
	
	// Content size
	size += int64(len(m.content.Text))
	size += int64(len(m.content.HTML))
	size += int64(len(m.envelope.Subject))
	
	// Attachment size
	for _, attachment := range m.attachments {
		if attachment.Size > 0 {
			size += attachment.Size
		} else {
			size += int64(len(attachment.Content))
		}
	}
	
	// Overhead for headers and encoding
	size += int64(m.GetRecipientCount() * 100) // Approximate per recipient
	
	return size
}

// Clone creates a copy of the mailable
func (m *BaseMailable) Clone() *BaseMailable {
	clone := NewMailable()
	
	// Copy envelope
	if m.envelope.From != nil {
		clone.envelope.From = &Address{
			Email: m.envelope.From.Email,
			Name:  m.envelope.From.Name,
		}
	}
	
	clone.envelope.To = cloneAddresses(m.envelope.To)
	clone.envelope.CC = cloneAddresses(m.envelope.CC)
	clone.envelope.BCC = cloneAddresses(m.envelope.BCC)
	clone.envelope.ReplyTo = cloneAddresses(m.envelope.ReplyTo)
	
	clone.envelope.Subject = m.envelope.Subject
	clone.envelope.Priority = m.envelope.Priority
	clone.envelope.TrackingID = m.envelope.TrackingID
	clone.envelope.DeliveryReceipt = m.envelope.DeliveryReceipt
	clone.envelope.ReadReceipt = m.envelope.ReadReceipt
	
	clone.envelope.Headers = cloneStringMap(m.envelope.Headers)
	clone.envelope.Metadata = cloneStringMap(m.envelope.Metadata)
	clone.envelope.Tags = cloneStringSlice(m.envelope.Tags)
	
	// Copy content
	clone.content.Text = m.content.Text
	clone.content.HTML = m.content.HTML
	clone.content.View = m.content.View
	clone.content.Charset = m.content.Charset
	clone.content.Encoding = m.content.Encoding
	
	// Copy template data
	for k, v := range m.content.TemplateData {
		clone.content.TemplateData[k] = v
	}
	
	// Copy mailable data
	for k, v := range m.data {
		clone.data[k] = v
	}
	
	// Copy attachments
	for _, att := range m.attachments {
		cloneAtt := &Attachment{
			Name:        att.Name,
			ContentType: att.ContentType,
			Disposition: att.Disposition,
			FilePath:    att.FilePath,
			Size:        att.Size,
			ContentID:   att.ContentID,
			Headers:     cloneStringMap(att.Headers),
		}
		
		if len(att.Content) > 0 {
			cloneAtt.Content = make([]byte, len(att.Content))
			copy(cloneAtt.Content, att.Content)
		}
		
		clone.attachments = append(clone.attachments, cloneAtt)
	}
	
	// Copy hooks
	clone.hooks = m.hooks
	
	return clone
}