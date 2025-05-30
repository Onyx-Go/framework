package mail

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SMTPDriver implements the Driver interface for SMTP delivery
type SMTPDriver struct {
	name   string
	config DriverConfig
	pool   *ConnectionPool
}

// ConnectionPool manages SMTP connections for reuse
type ConnectionPool struct {
	config      DriverConfig
	connections chan *smtp.Client
	maxConns    int
	timeout     time.Duration
}

// NewSMTPDriver creates a new SMTP driver
func NewSMTPDriver(name string, config DriverConfig) *SMTPDriver {
	maxConns := config.MaxConnections
	if maxConns <= 0 {
		maxConns = 10
	}
	
	pool := &ConnectionPool{
		config:      config,
		connections: make(chan *smtp.Client, maxConns),
		maxConns:    maxConns,
		timeout:     config.IdleTimeout,
	}
	
	return &SMTPDriver{
		name:   name,
		config: config,
		pool:   pool,
	}
}

// Interface implementations

// Send delivers a message via SMTP
func (d *SMTPDriver) Send(ctx context.Context, message *Message) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	
	// Validate message
	if err := message.Validate(); err != nil {
		return fmt.Errorf("invalid message: %w", err)
	}
	
	// Build the email content
	content, err := d.buildMessage(message)
	if err != nil {
		return fmt.Errorf("failed to build message: %w", err)
	}
	
	// Send via SMTP
	return d.sendSMTP(ctx, message.Envelope.From.Email, message.Envelope.GetRecipientEmails(), content)
}

// GetName returns the driver name
func (d *SMTPDriver) GetName() string {
	return d.name
}

// Validate checks if the driver configuration is valid
func (d *SMTPDriver) Validate() error {
	if d.config.Host == "" {
		return fmt.Errorf("SMTP host is required")
	}
	
	if d.config.Port <= 0 || d.config.Port > 65535 {
		return fmt.Errorf("invalid SMTP port: %d", d.config.Port)
	}
	
	if d.config.Encryption != "" {
		validEncryptions := []string{"tls", "ssl", "starttls"}
		valid := false
		for _, enc := range validEncryptions {
			if d.config.Encryption == enc {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid encryption type: %s", d.config.Encryption)
		}
	}
	
	return nil
}

// Close shuts down the driver and releases resources
func (d *SMTPDriver) Close(ctx context.Context) error {
	// Close all pooled connections
	close(d.pool.connections)
	
	for client := range d.pool.connections {
		if client != nil {
			client.Quit()
		}
	}
	
	return nil
}

// HealthCheck verifies the driver can connect to the SMTP server
func (d *SMTPDriver) HealthCheck(ctx context.Context) error {
	conn, err := d.pool.getConnection(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer d.pool.returnConnection(conn)
	
	// Test connection by sending NOOP
	if err := conn.Noop(); err != nil {
		return fmt.Errorf("SMTP server not responding: %w", err)
	}
	
	return nil
}

// GetCapabilities returns what features this driver supports
func (d *SMTPDriver) GetCapabilities() Capabilities {
	return Capabilities{
		HTML:               true,
		Text:               true,
		Attachments:        true,
		InlineImages:       true,
		MultipleRecipients: true,
		CCAndBCC:           true,
		CustomHeaders:      true,
		DeliveryTracking:   false, // Basic SMTP doesn't support tracking
		ReadReceipts:       true,
		Priority:           true,
		Encryption:         d.config.Encryption != "",
	}
}

// Message building methods

// buildMessage creates the full SMTP message content
func (d *SMTPDriver) buildMessage(message *Message) ([]byte, error) {
	var buf bytes.Buffer
	
	// Write headers
	if err := d.writeHeaders(&buf, message); err != nil {
		return nil, fmt.Errorf("failed to write headers: %w", err)
	}
	
	// Write body
	if err := d.writeBody(&buf, message); err != nil {
		return nil, fmt.Errorf("failed to write body: %w", err)
	}
	
	return buf.Bytes(), nil
}

// writeHeaders writes the email headers
func (d *SMTPDriver) writeHeaders(buf *bytes.Buffer, message *Message) error {
	env := message.Envelope
	
	// Basic headers
	buf.WriteString(fmt.Sprintf("From: %s\r\n", env.From.String()))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", d.formatAddresses(env.To)))
	
	if len(env.CC) > 0 {
		buf.WriteString(fmt.Sprintf("Cc: %s\r\n", d.formatAddresses(env.CC)))
	}
	
	if len(env.ReplyTo) > 0 {
		buf.WriteString(fmt.Sprintf("Reply-To: %s\r\n", d.formatAddresses(env.ReplyTo)))
	}
	
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", d.encodeSubject(env.Subject)))
	buf.WriteString(fmt.Sprintf("Date: %s\r\n", message.Date.Format(time.RFC1123Z)))
	buf.WriteString(fmt.Sprintf("Message-ID: %s\r\n", message.MessageID))
	buf.WriteString("MIME-Version: 1.0\r\n")
	
	// Priority header
	if env.Priority != PriorityNormal {
		priorityStr := "3" // Normal
		if env.Priority == PriorityHigh {
			priorityStr = "1"
		} else if env.Priority == PriorityLow {
			priorityStr = "5"
		}
		buf.WriteString(fmt.Sprintf("X-Priority: %s\r\n", priorityStr))
	}
	
	// Delivery receipt
	if env.DeliveryReceipt {
		buf.WriteString(fmt.Sprintf("Disposition-Notification-To: %s\r\n", env.From.Email))
	}
	
	// Read receipt
	if env.ReadReceipt {
		buf.WriteString(fmt.Sprintf("Return-Receipt-To: %s\r\n", env.From.Email))
	}
	
	// Custom headers
	for key, value := range env.Headers {
		buf.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
	}
	
	// Tracking headers
	if env.TrackingID != "" {
		buf.WriteString(fmt.Sprintf("X-Tracking-ID: %s\r\n", env.TrackingID))
	}
	
	if len(env.Tags) > 0 {
		buf.WriteString(fmt.Sprintf("X-Tags: %s\r\n", strings.Join(env.Tags, ", ")))
	}
	
	return nil
}

// writeBody writes the email body with appropriate MIME structure
func (d *SMTPDriver) writeBody(buf *bytes.Buffer, message *Message) error {
	hasAttachments := len(message.Attachments) > 0
	hasHTML := message.Content.RenderedHTML != ""
	hasText := message.Content.RenderedText != ""
	
	if hasAttachments {
		return d.writeMultipartMixed(buf, message)
	} else if hasHTML && hasText {
		return d.writeMultipartAlternative(buf, message)
	} else if hasHTML {
		return d.writeHTMLOnly(buf, message)
	} else {
		return d.writeTextOnly(buf, message)
	}
}

// writeMultipartMixed writes a multipart/mixed message (with attachments)
func (d *SMTPDriver) writeMultipartMixed(buf *bytes.Buffer, message *Message) error {
	boundary := d.generateBoundary()
	buf.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n\r\n", boundary))
	
	// Write message content part
	buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	
	hasHTML := message.Content.RenderedHTML != ""
	hasText := message.Content.RenderedText != ""
	
	if hasHTML && hasText {
		if err := d.writeMultipartAlternative(buf, message); err != nil {
			return err
		}
	} else if hasHTML {
		if err := d.writeHTMLOnly(buf, message); err != nil {
			return err
		}
	} else {
		if err := d.writeTextOnly(buf, message); err != nil {
			return err
		}
	}
	
	// Write attachments
	for _, attachment := range message.Attachments {
		buf.WriteString(fmt.Sprintf("\r\n--%s\r\n", boundary))
		if err := d.writeAttachment(buf, attachment); err != nil {
			return err
		}
	}
	
	buf.WriteString(fmt.Sprintf("\r\n--%s--\r\n", boundary))
	return nil
}

// writeMultipartAlternative writes a multipart/alternative message (HTML + text)
func (d *SMTPDriver) writeMultipartAlternative(buf *bytes.Buffer, message *Message) error {
	boundary := d.generateBoundary()
	buf.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n\r\n", boundary))
	
	// Text part first
	buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	buf.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
	buf.WriteString(d.encodeQuotedPrintable(message.Content.RenderedText))
	
	// HTML part
	buf.WriteString(fmt.Sprintf("\r\n--%s\r\n", boundary))
	buf.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
	buf.WriteString(d.encodeQuotedPrintable(message.Content.RenderedHTML))
	
	buf.WriteString(fmt.Sprintf("\r\n--%s--\r\n", boundary))
	return nil
}

// writeHTMLOnly writes an HTML-only message
func (d *SMTPDriver) writeHTMLOnly(buf *bytes.Buffer, message *Message) error {
	buf.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
	buf.WriteString(d.encodeQuotedPrintable(message.Content.RenderedHTML))
	return nil
}

// writeTextOnly writes a text-only message
func (d *SMTPDriver) writeTextOnly(buf *bytes.Buffer, message *Message) error {
	buf.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
	buf.WriteString(d.encodeQuotedPrintable(message.Content.RenderedText))
	return nil
}

// writeAttachment writes an attachment
func (d *SMTPDriver) writeAttachment(buf *bytes.Buffer, attachment *Attachment) error {
	// Determine content type
	contentType := attachment.ContentType
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(attachment.Name))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
	}
	
	// Determine disposition
	disposition := attachment.Disposition
	if disposition == "" {
		disposition = "attachment"
	}
	
	// Write headers
	buf.WriteString(fmt.Sprintf("Content-Type: %s\r\n", contentType))
	buf.WriteString(fmt.Sprintf("Content-Disposition: %s; filename=\"%s\"\r\n", disposition, attachment.Name))
	buf.WriteString("Content-Transfer-Encoding: base64\r\n")
	
	// Inline attachments get Content-ID
	if attachment.ContentID != "" {
		buf.WriteString(fmt.Sprintf("Content-ID: <%s>\r\n", attachment.ContentID))
	}
	
	// Custom headers
	for key, value := range attachment.Headers {
		buf.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
	}
	
	buf.WriteString("\r\n")
	
	// Get attachment content
	content := attachment.Content
	if len(content) == 0 && attachment.FilePath != "" {
		data, err := os.ReadFile(attachment.FilePath)
		if err != nil {
			return fmt.Errorf("failed to read attachment file %s: %w", attachment.FilePath, err)
		}
		content = data
	}
	
	// Encode in base64 with line breaks
	encoded := base64.StdEncoding.EncodeToString(content)
	for i := 0; i < len(encoded); i += 76 {
		end := i + 76
		if end > len(encoded) {
			end = len(encoded)
		}
		buf.WriteString(encoded[i:end])
		buf.WriteString("\r\n")
	}
	
	return nil
}

// SMTP connection methods

// sendSMTP sends the message via SMTP
func (d *SMTPDriver) sendSMTP(ctx context.Context, from string, to []string, message []byte) error {
	client, err := d.pool.getConnection(ctx)
	if err != nil {
		return fmt.Errorf("failed to get SMTP connection: %w", err)
	}
	defer d.pool.returnConnection(client)
	
	// Set sender
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}
	
	// Set recipients
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", recipient, err)
		}
	}
	
	// Send message data
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to start data transfer: %w", err)
	}
	defer writer.Close()
	
	if _, err := writer.Write(message); err != nil {
		return fmt.Errorf("failed to write message data: %w", err)
	}
	
	return nil
}

// Connection pool methods

// getConnection gets a connection from the pool or creates a new one
func (cp *ConnectionPool) getConnection(ctx context.Context) (*smtp.Client, error) {
	select {
	case client := <-cp.connections:
		// Test if connection is still valid
		if err := client.Noop(); err == nil {
			return client, nil
		}
		// Connection is dead, create a new one
		client.Quit()
	default:
		// No connections available, create a new one
	}
	
	return cp.createConnection(ctx)
}

// returnConnection returns a connection to the pool
func (cp *ConnectionPool) returnConnection(client *smtp.Client) {
	if client == nil {
		return
	}
	
	select {
	case cp.connections <- client:
		// Successfully returned to pool
	default:
		// Pool is full, close the connection
		client.Quit()
	}
}

// createConnection creates a new SMTP connection
func (cp *ConnectionPool) createConnection(ctx context.Context) (*smtp.Client, error) {
	addr := fmt.Sprintf("%s:%d", cp.config.Host, cp.config.Port)
	
	// Set timeout for connection
	timeout := cp.config.ConnectTimeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	
	// Create connection with timeout
	dialer := &net.Dialer{Timeout: timeout}
	
	var conn net.Conn
	var err error
	
	if cp.config.Encryption == "ssl" || cp.config.Encryption == "tls" {
		// Direct TLS connection
		tlsConfig := &tls.Config{
			ServerName:         cp.config.Host,
			InsecureSkipVerify: false,
		}
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, tlsConfig)
	} else {
		// Plain connection (may upgrade to TLS later)
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", addr, err)
	}
	
	// Create SMTP client
	client, err := smtp.NewClient(conn, cp.config.Host)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create SMTP client: %w", err)
	}
	
	// Upgrade to TLS if needed
	if cp.config.Encryption == "starttls" {
		tlsConfig := &tls.Config{
			ServerName:         cp.config.Host,
			InsecureSkipVerify: false,
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			client.Quit()
			return nil, fmt.Errorf("failed to start TLS: %w", err)
		}
	}
	
	// Authenticate if credentials provided
	if cp.config.Username != "" && cp.config.Password != "" {
		auth := smtp.PlainAuth("", cp.config.Username, cp.config.Password, cp.config.Host)
		if err := client.Auth(auth); err != nil {
			client.Quit()
			return nil, fmt.Errorf("SMTP authentication failed: %w", err)
		}
	}
	
	return client, nil
}

// Utility methods

// formatAddresses formats a slice of addresses for headers
func (d *SMTPDriver) formatAddresses(addresses []*Address) string {
	var formatted []string
	for _, addr := range addresses {
		formatted = append(formatted, addr.String())
	}
	return strings.Join(formatted, ", ")
}

// generateBoundary generates a unique boundary for multipart messages
func (d *SMTPDriver) generateBoundary() string {
	return fmt.Sprintf("boundary_%d_%d", time.Now().UnixNano(), time.Now().Unix())
}

// encodeSubject encodes the subject header according to RFC 2047
func (d *SMTPDriver) encodeSubject(subject string) string {
	// Simple implementation - in production you'd use proper RFC 2047 encoding
	return subject
}

// encodeQuotedPrintable encodes text in quoted-printable format
func (d *SMTPDriver) encodeQuotedPrintable(text string) string {
	// Simplified implementation
	// In production, you'd use proper quoted-printable encoding
	return text
}