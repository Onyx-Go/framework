package onyx

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"math/big"
	"mime"
	"net/smtp"
	"os"
	"path/filepath"
	"strings"
	texttemplate "text/template"
	"time"
)

// Mail interfaces and types

// MailDriver interface defines the contract for mail drivers
type MailDriver interface {
	Send(message *Message) error
	GetName() string
}

// Mailable interface defines the contract for mailable structs
type Mailable interface {
	Envelope() *Envelope
	Content() *Content
	Attachments() []*Attachment
	GetData() map[string]interface{}
}

// MailManager manages mail configuration and drivers
type MailManager struct {
	config        *MailConfig
	drivers       map[string]MailDriver
	defaultDriver string
}

// Mail configuration structures

type MailConfig struct {
	DefaultMailer string                    `json:"default"`
	From          Address                   `json:"from"`
	Mailers       map[string]MailerConfig   `json:"mailers"`
}

type MailerConfig struct {
	Transport string            `json:"transport"`
	Host      string            `json:"host"`
	Port      int               `json:"port"`
	Username  string            `json:"username"`
	Password  string            `json:"password"`
	Encryption string           `json:"encryption"`
	Timeout   int               `json:"timeout"`
	Options   map[string]string `json:"options"`
}

// Message components

type Address struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

func (a Address) String() string {
	if a.Name != "" {
		return fmt.Sprintf("%s <%s>", a.Name, a.Email)
	}
	return a.Email
}

type Envelope struct {
	From     *Address   `json:"from"`
	To       []*Address `json:"to"`
	CC       []*Address `json:"cc"`
	BCC      []*Address `json:"bcc"`
	ReplyTo  []*Address `json:"reply_to"`
	Subject  string     `json:"subject"`
	Headers  map[string]string `json:"headers"`
	Tags     []string   `json:"tags"`
	Metadata map[string]string `json:"metadata"`
}

type Content struct {
	View     string                 `json:"view"`
	Text     string                 `json:"text"`
	HTML     string                 `json:"html"`
	Data     map[string]interface{} `json:"data"`
	ViewData map[string]interface{} `json:"view_data"`
}

type Attachment struct {
	Name        string            `json:"name"`
	Content     []byte            `json:"content"`
	ContentType string            `json:"content_type"`
	Disposition string            `json:"disposition"`
	Headers     map[string]string `json:"headers"`
	FilePath    string            `json:"file_path"`
}

// Message represents a complete email message
type Message struct {
	Envelope    *Envelope     `json:"envelope"`
	Content     *Content      `json:"content"`
	Attachments []*Attachment `json:"attachments"`
	RenderedHTML string       `json:"rendered_html"`
	RenderedText string       `json:"rendered_text"`
	MessageID   string        `json:"message_id"`
	Date        time.Time     `json:"date"`
}

// SMTP Driver implementation

type SMTPDriver struct {
	config MailerConfig
	name   string
}

func NewSMTPDriver(name string, config MailerConfig) *SMTPDriver {
	return &SMTPDriver{
		config: config,
		name:   name,
	}
}

func (d *SMTPDriver) GetName() string {
	return d.name
}

func (d *SMTPDriver) Send(message *Message) error {
	// Build the email message
	var buf bytes.Buffer
	
	// Write headers
	if err := d.writeHeaders(&buf, message); err != nil {
		return fmt.Errorf("failed to write headers: %w", err)
	}
	
	// Write body
	if err := d.writeBody(&buf, message); err != nil {
		return fmt.Errorf("failed to write body: %w", err)
	}
	
	// Get recipients
	recipients := d.getRecipients(message.Envelope)
	if len(recipients) == 0 {
		return fmt.Errorf("no recipients specified")
	}
	
	// Send via SMTP
	return d.sendSMTP(message.Envelope.From.Email, recipients, buf.Bytes())
}

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
	
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", env.Subject))
	buf.WriteString(fmt.Sprintf("Date: %s\r\n", message.Date.Format(time.RFC1123Z)))
	buf.WriteString(fmt.Sprintf("Message-ID: %s\r\n", message.MessageID))
	buf.WriteString("MIME-Version: 1.0\r\n")
	
	// Custom headers
	for key, value := range env.Headers {
		buf.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
	}
	
	return nil
}

func (d *SMTPDriver) writeBody(buf *bytes.Buffer, message *Message) error {
	hasAttachments := len(message.Attachments) > 0
	hasHTML := message.RenderedHTML != ""
	hasText := message.RenderedText != ""
	
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

func (d *SMTPDriver) writeMultipartMixed(buf *bytes.Buffer, message *Message) error {
	boundary := d.generateBoundary()
	buf.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n\r\n", boundary))
	
	// Write message content
	buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	if message.RenderedHTML != "" && message.RenderedText != "" {
		if err := d.writeMultipartAlternative(buf, message); err != nil {
			return err
		}
	} else if message.RenderedHTML != "" {
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

func (d *SMTPDriver) writeMultipartAlternative(buf *bytes.Buffer, message *Message) error {
	boundary := d.generateBoundary()
	buf.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n\r\n", boundary))
	
	// Text part
	buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	buf.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
	buf.WriteString(message.RenderedText)
	
	// HTML part
	buf.WriteString(fmt.Sprintf("\r\n--%s\r\n", boundary))
	buf.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
	buf.WriteString(message.RenderedHTML)
	
	buf.WriteString(fmt.Sprintf("\r\n--%s--\r\n", boundary))
	return nil
}

func (d *SMTPDriver) writeHTMLOnly(buf *bytes.Buffer, message *Message) error {
	buf.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
	buf.WriteString(message.RenderedHTML)
	return nil
}

func (d *SMTPDriver) writeTextOnly(buf *bytes.Buffer, message *Message) error {
	buf.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
	buf.WriteString(message.RenderedText)
	return nil
}

func (d *SMTPDriver) writeAttachment(buf *bytes.Buffer, attachment *Attachment) error {
	contentType := attachment.ContentType
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(attachment.Name))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
	}
	
	disposition := attachment.Disposition
	if disposition == "" {
		disposition = "attachment"
	}
	
	buf.WriteString(fmt.Sprintf("Content-Type: %s\r\n", contentType))
	buf.WriteString(fmt.Sprintf("Content-Disposition: %s; filename=\"%s\"\r\n", disposition, attachment.Name))
	buf.WriteString("Content-Transfer-Encoding: base64\r\n")
	
	// Custom headers
	for key, value := range attachment.Headers {
		buf.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
	}
	
	buf.WriteString("\r\n")
	
	// Encode content in base64
	content := attachment.Content
	if len(content) == 0 && attachment.FilePath != "" {
		data, err := os.ReadFile(attachment.FilePath)
		if err != nil {
			return fmt.Errorf("failed to read attachment file %s: %w", attachment.FilePath, err)
		}
		content = data
	}
	
	encoded := encodeBase64(content)
	buf.WriteString(encoded)
	
	return nil
}

func (d *SMTPDriver) formatAddresses(addresses []*Address) string {
	var formatted []string
	for _, addr := range addresses {
		formatted = append(formatted, addr.String())
	}
	return strings.Join(formatted, ", ")
}

func (d *SMTPDriver) getRecipients(env *Envelope) []string {
	var recipients []string
	
	for _, addr := range env.To {
		recipients = append(recipients, addr.Email)
	}
	for _, addr := range env.CC {
		recipients = append(recipients, addr.Email)
	}
	for _, addr := range env.BCC {
		recipients = append(recipients, addr.Email)
	}
	
	return recipients
}

func (d *SMTPDriver) generateBoundary() string {
	return fmt.Sprintf("boundary_%d", time.Now().UnixNano())
}

func (d *SMTPDriver) sendSMTP(from string, to []string, message []byte) error {
	addr := fmt.Sprintf("%s:%d", d.config.Host, d.config.Port)
	
	var auth smtp.Auth
	if d.config.Username != "" && d.config.Password != "" {
		auth = smtp.PlainAuth("", d.config.Username, d.config.Password, d.config.Host)
	}
	
	if d.config.Encryption == "tls" {
		return d.sendTLS(addr, auth, from, to, message)
	}
	
	return smtp.SendMail(addr, auth, from, to, message)
}

func (d *SMTPDriver) sendTLS(addr string, auth smtp.Auth, from string, to []string, message []byte) error {
	host := strings.Split(addr, ":")[0]
	
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         host,
	}
	
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return err
	}
	defer conn.Close()
	
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Quit()
	
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	
	if err := client.Mail(from); err != nil {
		return err
	}
	
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return err
		}
	}
	
	writer, err := client.Data()
	if err != nil {
		return err
	}
	defer writer.Close()
	
	_, err = writer.Write(message)
	return err
}

// Base Mailable implementation

type BaseMailable struct {
	envelope    *Envelope
	content     *Content
	attachments []*Attachment
	data        map[string]interface{}
}

func NewBaseMailable() *BaseMailable {
	return &BaseMailable{
		envelope: &Envelope{
			Headers:  make(map[string]string),
			Metadata: make(map[string]string),
		},
		content: &Content{
			Data:     make(map[string]interface{}),
			ViewData: make(map[string]interface{}),
		},
		attachments: make([]*Attachment, 0),
		data:        make(map[string]interface{}),
	}
}

func (m *BaseMailable) Envelope() *Envelope {
	return m.envelope
}

func (m *BaseMailable) Content() *Content {
	return m.content
}

func (m *BaseMailable) Attachments() []*Attachment {
	return m.attachments
}

func (m *BaseMailable) GetData() map[string]interface{} {
	return m.data
}

// Fluent interface methods

func (m *BaseMailable) From(email, name string) *BaseMailable {
	m.envelope.From = &Address{Email: email, Name: name}
	return m
}

func (m *BaseMailable) To(email, name string) *BaseMailable {
	m.envelope.To = append(m.envelope.To, &Address{Email: email, Name: name})
	return m
}

func (m *BaseMailable) CC(email, name string) *BaseMailable {
	m.envelope.CC = append(m.envelope.CC, &Address{Email: email, Name: name})
	return m
}

func (m *BaseMailable) BCC(email, name string) *BaseMailable {
	m.envelope.BCC = append(m.envelope.BCC, &Address{Email: email, Name: name})
	return m
}

func (m *BaseMailable) ReplyTo(email, name string) *BaseMailable {
	m.envelope.ReplyTo = append(m.envelope.ReplyTo, &Address{Email: email, Name: name})
	return m
}

func (m *BaseMailable) Subject(subject string) *BaseMailable {
	m.envelope.Subject = subject
	return m
}

func (m *BaseMailable) View(view string) *BaseMailable {
	m.content.View = view
	return m
}

func (m *BaseMailable) Text(text string) *BaseMailable {
	m.content.Text = text
	return m
}

func (m *BaseMailable) HTML(html string) *BaseMailable {
	m.content.HTML = html
	return m
}

func (m *BaseMailable) With(key string, value interface{}) *BaseMailable {
	m.content.ViewData[key] = value
	return m
}

func (m *BaseMailable) WithData(data map[string]interface{}) *BaseMailable {
	for k, v := range data {
		m.content.ViewData[k] = v
	}
	return m
}

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

func (m *BaseMailable) AttachData(content []byte, name, contentType string) *BaseMailable {
	attachment := &Attachment{
		Content:     content,
		Name:        name,
		ContentType: contentType,
		Headers:     make(map[string]string),
	}
	m.attachments = append(m.attachments, attachment)
	return m
}

func (m *BaseMailable) Header(key, value string) *BaseMailable {
	m.envelope.Headers[key] = value
	return m
}

func (m *BaseMailable) Tag(tag string) *BaseMailable {
	m.envelope.Tags = append(m.envelope.Tags, tag)
	return m
}

func (m *BaseMailable) Meta(key, value string) *BaseMailable {
	m.envelope.Metadata[key] = value
	return m
}

// Mail Manager implementation

func NewMailManager(config *MailConfig) *MailManager {
	return &MailManager{
		config:        config,
		drivers:       make(map[string]MailDriver),
		defaultDriver: config.DefaultMailer,
	}
}

func (mm *MailManager) RegisterDriver(name string, driver MailDriver) {
	mm.drivers[name] = driver
}

func (mm *MailManager) GetDriver(name string) (MailDriver, error) {
	if name == "" {
		name = mm.defaultDriver
	}
	
	driver, exists := mm.drivers[name]
	if !exists {
		return nil, fmt.Errorf("mail driver '%s' not found", name)
	}
	
	return driver, nil
}

func (mm *MailManager) Send(mailable Mailable, driverName ...string) error {
	driver, err := mm.getDriverToUse(driverName...)
	if err != nil {
		return err
	}
	
	message, err := mm.buildMessage(mailable)
	if err != nil {
		return fmt.Errorf("failed to build message: %w", err)
	}
	
	return driver.Send(message)
}

func (mm *MailManager) getDriverToUse(driverName ...string) (MailDriver, error) {
	var name string
	if len(driverName) > 0 {
		name = driverName[0]
	}
	return mm.GetDriver(name)
}

func (mm *MailManager) buildMessage(mailable Mailable) (*Message, error) {
	envelope := mailable.Envelope()
	content := mailable.Content()
	attachments := mailable.Attachments()
	
	// Set default from address if not specified
	if envelope.From == nil {
		envelope.From = &mm.config.From
	}
	
	message := &Message{
		Envelope:    envelope,
		Content:     content,
		Attachments: attachments,
		MessageID:   generateMessageID(),
		Date:        time.Now(),
	}
	
	// Render templates
	if err := mm.renderContent(message, mailable); err != nil {
		return nil, fmt.Errorf("failed to render content: %w", err)
	}
	
	return message, nil
}

func (mm *MailManager) renderContent(message *Message, mailable Mailable) error {
	content := message.Content
	data := mailable.GetData()
	
	// Merge view data with mailable data
	for k, v := range content.ViewData {
		data[k] = v
	}
	
	// Render HTML template
	if content.View != "" {
		rendered, err := mm.renderTemplate(content.View+".html", data)
		if err == nil {
			message.RenderedHTML = rendered
		}
		
		// Try to render text version
		textRendered, err := mm.renderTemplate(content.View+".txt", data)
		if err == nil {
			message.RenderedText = textRendered
		}
	}
	
	// Use direct HTML/text if no view specified
	if content.HTML != "" {
		message.RenderedHTML = content.HTML
	}
	if content.Text != "" {
		message.RenderedText = content.Text
	}
	
	return nil
}

func (mm *MailManager) renderTemplate(templateName string, data map[string]interface{}) (string, error) {
	// This is a simplified template rendering
	// In a real implementation, you'd integrate with your template system
	templateContent := "Hello {{.Name}}, this is a test email."
	
	tmpl, err := texttemplate.New("email").Parse(templateContent)
	if err != nil {
		return "", err
	}
	
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	
	return buf.String(), nil
}

// Utility functions

func generateMessageID() string {
	now := time.Now()
	return fmt.Sprintf("<%d.%d.%d@onyx>", now.UnixNano(), now.UnixMicro(), fastRand())
}

func fastRand() int64 {
	// Generate a cryptographically secure random number for uniqueness
	max := big.NewInt(1000000) // 1 million possibilities
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		// Fallback to time-based random if crypto/rand fails
		return time.Now().UnixNano() % 1000000
	}
	return n.Int64()
}

func encodeBase64(data []byte) string {
	const lineLength = 76
	
	encoded := make([]byte, 0, len(data)*4/3+4)
	
	// Base64 encode
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	
	for i := 0; i < len(data); i += 3 {
		var b0, b1, b2 byte
		b0 = data[i]
		if i+1 < len(data) {
			b1 = data[i+1]
		}
		if i+2 < len(data) {
			b2 = data[i+2]
		}
		
		encoded = append(encoded, chars[b0>>2])
		encoded = append(encoded, chars[((b0&0x03)<<4)|((b1&0xf0)>>4)])
		if i+1 < len(data) {
			encoded = append(encoded, chars[((b1&0x0f)<<2)|((b2&0xc0)>>6)])
		} else {
			encoded = append(encoded, '=')
		}
		if i+2 < len(data) {
			encoded = append(encoded, chars[b2&0x3f])
		} else {
			encoded = append(encoded, '=')
		}
		
		// Add line breaks every 76 characters
		if len(encoded)%lineLength == 0 {
			encoded = append(encoded, '\r', '\n')
		}
	}
	
	return string(encoded)
}

// Global mail function
var globalMailManager *MailManager

func SetGlobalMailManager(manager *MailManager) {
	globalMailManager = manager
}

func Mail() *MailManager {
	if globalMailManager == nil {
		// Create default mail manager if none exists
		config := &MailConfig{
			DefaultMailer: "smtp",
			From:          Address{Email: "noreply@example.com", Name: "Onyx"},
			Mailers: map[string]MailerConfig{
				"smtp": {
					Transport:  "smtp",
					Host:       "localhost",
					Port:       587,
					Username:   "",
					Password:   "",
					Encryption: "",
					Timeout:    30,
				},
			},
		}
		globalMailManager = NewMailManager(config)
		
		// Register SMTP driver
		smtpDriver := NewSMTPDriver("smtp", config.Mailers["smtp"])
		globalMailManager.RegisterDriver("smtp", smtpDriver)
	}
	
	return globalMailManager
}

// Convenience functions for creating mailables

func NewMail() *BaseMailable {
	return NewBaseMailable()
}

func SendMail(mailable Mailable, driverName ...string) error {
	return Mail().Send(mailable, driverName...)
}

// Context helper function for getting mail manager
func GetMailFromContext(c Context) *MailManager {
	// For now, use the global mail manager since Application interface doesn't expose Container
	// TODO: Extend Application interface to provide access to Container
	return Mail()
}