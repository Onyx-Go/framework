package onyx

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// Mock mail driver for testing
type MockMailDriver struct {
	name         string
	sentMessages []*Message
	shouldFail   bool
	failMessage  string
}

func NewMockMailDriver(name string) *MockMailDriver {
	return &MockMailDriver{
		name:         name,
		sentMessages: make([]*Message, 0),
		shouldFail:   false,
	}
}

func (d *MockMailDriver) GetName() string {
	return d.name
}

func (d *MockMailDriver) Send(message *Message) error {
	if d.shouldFail {
		return fmt.Errorf("%s", d.failMessage)
	}
	
	d.sentMessages = append(d.sentMessages, message)
	return nil
}

func (d *MockMailDriver) SetShouldFail(shouldFail bool, message string) {
	d.shouldFail = shouldFail
	d.failMessage = message
}

func (d *MockMailDriver) GetSentMessages() []*Message {
	return d.sentMessages
}

func (d *MockMailDriver) Reset() {
	d.sentMessages = make([]*Message, 0)
	d.shouldFail = false
	d.failMessage = ""
}

// Test Address functionality
func TestAddress(t *testing.T) {
	tests := []struct {
		name     string
		address  Address
		expected string
	}{
		{
			name:     "Email only",
			address:  Address{Email: "test@example.com"},
			expected: "test@example.com",
		},
		{
			name:     "Email with name",
			address:  Address{Email: "test@example.com", Name: "Test User"},
			expected: "Test User <test@example.com>",
		},
		{
			name:     "Empty name",
			address:  Address{Email: "test@example.com", Name: ""},
			expected: "test@example.com",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.address.String()
			if result != tt.expected {
				t.Errorf("Address.String() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// Test BaseMailable functionality
func TestBaseMailable(t *testing.T) {
	mailable := NewBaseMailable()
	
	// Test fluent interface
	mailable.From("sender@example.com", "Sender Name").
		To("recipient@example.com", "Recipient Name").
		CC("cc@example.com", "CC User").
		BCC("bcc@example.com", "BCC User").
		ReplyTo("reply@example.com", "Reply User").
		Subject("Test Subject").
		View("test-template").
		Text("Plain text content").
		HTML("<h1>HTML content</h1>").
		With("name", "John").
		WithData(map[string]interface{}{
			"age":  30,
			"city": "New York",
		}).
		Attach("/path/to/file.pdf", "attachment.pdf").
		AttachData([]byte("file content"), "data.txt", "text/plain").
		Header("X-Custom-Header", "custom-value").
		Tag("newsletter").
		Meta("campaign_id", "123")
	
	// Test envelope
	envelope := mailable.Envelope()
	if envelope.From.Email != "sender@example.com" {
		t.Errorf("From email = %q, want %q", envelope.From.Email, "sender@example.com")
	}
	if envelope.From.Name != "Sender Name" {
		t.Errorf("From name = %q, want %q", envelope.From.Name, "Sender Name")
	}
	if len(envelope.To) != 1 || envelope.To[0].Email != "recipient@example.com" {
		t.Errorf("To address not set correctly")
	}
	if envelope.Subject != "Test Subject" {
		t.Errorf("Subject = %q, want %q", envelope.Subject, "Test Subject")
	}
	if envelope.Headers["X-Custom-Header"] != "custom-value" {
		t.Errorf("Custom header not set correctly")
	}
	if len(envelope.Tags) != 1 || envelope.Tags[0] != "newsletter" {
		t.Errorf("Tags not set correctly")
	}
	if envelope.Metadata["campaign_id"] != "123" {
		t.Errorf("Metadata not set correctly")
	}
	
	// Test content
	content := mailable.Content()
	if content.View != "test-template" {
		t.Errorf("View = %q, want %q", content.View, "test-template")
	}
	if content.Text != "Plain text content" {
		t.Errorf("Text = %q, want %q", content.Text, "Plain text content")
	}
	if content.HTML != "<h1>HTML content</h1>" {
		t.Errorf("HTML = %q, want %q", content.HTML, "<h1>HTML content</h1>")
	}
	if content.ViewData["name"] != "John" {
		t.Errorf("ViewData[name] = %v, want %q", content.ViewData["name"], "John")
	}
	if content.ViewData["age"] != 30 {
		t.Errorf("ViewData[age] = %v, want %d", content.ViewData["age"], 30)
	}
	
	// Test attachments
	attachments := mailable.Attachments()
	if len(attachments) != 2 {
		t.Errorf("Attachments count = %d, want %d", len(attachments), 2)
	}
	if attachments[0].FilePath != "/path/to/file.pdf" {
		t.Errorf("First attachment FilePath = %q, want %q", attachments[0].FilePath, "/path/to/file.pdf")
	}
	if attachments[1].Name != "data.txt" {
		t.Errorf("Second attachment Name = %q, want %q", attachments[1].Name, "data.txt")
	}
}

// Test MailManager functionality
func TestMailManager(t *testing.T) {
	config := &MailConfig{
		DefaultMailer: "mock",
		From:          Address{Email: "default@example.com", Name: "Default Sender"},
		Mailers: map[string]MailerConfig{
			"mock": {
				Transport: "mock",
			},
		},
	}
	
	manager := NewMailManager(config)
	mockDriver := NewMockMailDriver("mock")
	manager.RegisterDriver("mock", mockDriver)
	
	// Test driver registration and retrieval
	driver, err := manager.GetDriver("mock")
	if err != nil {
		t.Fatalf("Failed to get driver: %v", err)
	}
	if driver.GetName() != "mock" {
		t.Errorf("Driver name = %q, want %q", driver.GetName(), "mock")
	}
	
	// Test getting default driver
	defaultDriver, err := manager.GetDriver("")
	if err != nil {
		t.Fatalf("Failed to get default driver: %v", err)
	}
	if defaultDriver.GetName() != "mock" {
		t.Errorf("Default driver name = %q, want %q", defaultDriver.GetName(), "mock")
	}
	
	// Test getting non-existent driver
	_, err = manager.GetDriver("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent driver")
	}
}

// Test sending mail
func TestSendMail(t *testing.T) {
	config := &MailConfig{
		DefaultMailer: "mock",
		From:          Address{Email: "default@example.com", Name: "Default Sender"},
		Mailers: map[string]MailerConfig{
			"mock": {
				Transport: "mock",
			},
		},
	}
	
	manager := NewMailManager(config)
	mockDriver := NewMockMailDriver("mock")
	manager.RegisterDriver("mock", mockDriver)
	
	// Create a test mailable
	mailable := NewBaseMailable().
		To("recipient@example.com", "Recipient").
		Subject("Test Email").
		Text("This is a test email")
	
	// Send the email
	err := manager.Send(mailable)
	if err != nil {
		t.Fatalf("Failed to send email: %v", err)
	}
	
	// Check that the email was sent
	sentMessages := mockDriver.GetSentMessages()
	if len(sentMessages) != 1 {
		t.Fatalf("Expected 1 sent message, got %d", len(sentMessages))
	}
	
	message := sentMessages[0]
	
	// Check envelope
	if message.Envelope.From.Email != "default@example.com" {
		t.Errorf("From email = %q, want %q", message.Envelope.From.Email, "default@example.com")
	}
	if len(message.Envelope.To) != 1 || message.Envelope.To[0].Email != "recipient@example.com" {
		t.Error("To address not set correctly")
	}
	if message.Envelope.Subject != "Test Email" {
		t.Errorf("Subject = %q, want %q", message.Envelope.Subject, "Test Email")
	}
	
	// Check content
	if message.Content.Text != "This is a test email" {
		t.Errorf("Text content = %q, want %q", message.Content.Text, "This is a test email")
	}
	
	// Check message metadata
	if message.MessageID == "" {
		t.Error("MessageID should not be empty")
	}
	if message.Date.IsZero() {
		t.Error("Date should not be zero")
	}
}

// Test sending mail with custom driver
func TestSendMailWithCustomDriver(t *testing.T) {
	config := &MailConfig{
		DefaultMailer: "mock1",
		From:          Address{Email: "default@example.com", Name: "Default Sender"},
		Mailers: map[string]MailerConfig{
			"mock1": {Transport: "mock"},
			"mock2": {Transport: "mock"},
		},
	}
	
	manager := NewMailManager(config)
	mockDriver1 := NewMockMailDriver("mock1")
	mockDriver2 := NewMockMailDriver("mock2")
	manager.RegisterDriver("mock1", mockDriver1)
	manager.RegisterDriver("mock2", mockDriver2)
	
	mailable := NewBaseMailable().
		To("recipient@example.com", "Recipient").
		Subject("Test Email").
		Text("This is a test email")
	
	// Send with specific driver
	err := manager.Send(mailable, "mock2")
	if err != nil {
		t.Fatalf("Failed to send email: %v", err)
	}
	
	// Check that mock2 received the email, not mock1
	if len(mockDriver1.GetSentMessages()) != 0 {
		t.Error("mock1 should not have received any messages")
	}
	if len(mockDriver2.GetSentMessages()) != 1 {
		t.Error("mock2 should have received 1 message")
	}
}

// Test mail sending failure
func TestSendMailFailure(t *testing.T) {
	config := &MailConfig{
		DefaultMailer: "mock",
		From:          Address{Email: "default@example.com", Name: "Default Sender"},
		Mailers: map[string]MailerConfig{
			"mock": {Transport: "mock"},
		},
	}
	
	manager := NewMailManager(config)
	mockDriver := NewMockMailDriver("mock")
	mockDriver.SetShouldFail(true, "SMTP connection failed")
	manager.RegisterDriver("mock", mockDriver)
	
	mailable := NewBaseMailable().
		To("recipient@example.com", "Recipient").
		Subject("Test Email").
		Text("This is a test email")
	
	err := manager.Send(mailable)
	if err == nil {
		t.Error("Expected send to fail")
	}
	if !strings.Contains(err.Error(), "SMTP connection failed") {
		t.Errorf("Error message should contain 'SMTP connection failed', got: %v", err)
	}
}

// Test SMTP driver formatting
func TestSMTPDriverFormatting(t *testing.T) {
	config := MailerConfig{
		Transport: "smtp",
		Host:      "smtp.example.com",
		Port:      587,
		Username:  "user@example.com",
		Password:  "password",
	}
	
	driver := NewSMTPDriver("smtp", config)
	
	// Test address formatting
	addresses := []*Address{
		{Email: "test1@example.com", Name: "Test User 1"},
		{Email: "test2@example.com", Name: ""},
		{Email: "test3@example.com", Name: "Test User 3"},
	}
	
	formatted := driver.formatAddresses(addresses)
	expected := "Test User 1 <test1@example.com>, test2@example.com, Test User 3 <test3@example.com>"
	if formatted != expected {
		t.Errorf("formatAddresses() = %q, want %q", formatted, expected)
	}
}

// Test SMTP driver recipients
func TestSMTPDriverRecipients(t *testing.T) {
	config := MailerConfig{
		Transport: "smtp",
		Host:      "smtp.example.com",
		Port:      587,
	}
	
	driver := NewSMTPDriver("smtp", config)
	
	envelope := &Envelope{
		To: []*Address{
			{Email: "to1@example.com"},
			{Email: "to2@example.com"},
		},
		CC: []*Address{
			{Email: "cc1@example.com"},
		},
		BCC: []*Address{
			{Email: "bcc1@example.com"},
			{Email: "bcc2@example.com"},
		},
	}
	
	recipients := driver.getRecipients(envelope)
	expected := []string{
		"to1@example.com",
		"to2@example.com",
		"cc1@example.com",
		"bcc1@example.com",
		"bcc2@example.com",
	}
	
	if len(recipients) != len(expected) {
		t.Errorf("Recipients count = %d, want %d", len(recipients), len(expected))
	}
	
	for i, recipient := range recipients {
		if recipient != expected[i] {
			t.Errorf("Recipient[%d] = %q, want %q", i, recipient, expected[i])
		}
	}
}

// Test attachment functionality
func TestAttachments(t *testing.T) {
	mailable := NewBaseMailable()
	
	// Test file attachment
	mailable.Attach("/path/to/file.pdf", "custom-name.pdf")
	
	// Test data attachment
	data := []byte("This is test data")
	mailable.AttachData(data, "test.txt", "text/plain")
	
	attachments := mailable.Attachments()
	if len(attachments) != 2 {
		t.Fatalf("Expected 2 attachments, got %d", len(attachments))
	}
	
	// Test file attachment
	fileAttachment := attachments[0]
	if fileAttachment.FilePath != "/path/to/file.pdf" {
		t.Errorf("FilePath = %q, want %q", fileAttachment.FilePath, "/path/to/file.pdf")
	}
	if fileAttachment.Name != "custom-name.pdf" {
		t.Errorf("Name = %q, want %q", fileAttachment.Name, "custom-name.pdf")
	}
	
	// Test data attachment
	dataAttachment := attachments[1]
	if string(dataAttachment.Content) != "This is test data" {
		t.Errorf("Content = %q, want %q", string(dataAttachment.Content), "This is test data")
	}
	if dataAttachment.Name != "test.txt" {
		t.Errorf("Name = %q, want %q", dataAttachment.Name, "test.txt")
	}
	if dataAttachment.ContentType != "text/plain" {
		t.Errorf("ContentType = %q, want %q", dataAttachment.ContentType, "text/plain")
	}
}

// Test message building
func TestMessageBuilding(t *testing.T) {
	config := &MailConfig{
		DefaultMailer: "mock",
		From:          Address{Email: "default@example.com", Name: "Default Sender"},
		Mailers: map[string]MailerConfig{
			"mock": {Transport: "mock"},
		},
	}
	
	manager := NewMailManager(config)
	
	mailable := NewBaseMailable().
		From("sender@example.com", "Custom Sender").
		To("recipient@example.com", "Recipient").
		Subject("Test Subject").
		Text("Plain text content").
		HTML("<h1>HTML content</h1>").
		With("name", "John")
	
	message, err := manager.buildMessage(mailable)
	if err != nil {
		t.Fatalf("Failed to build message: %v", err)
	}
	
	// Test that custom from overrides default
	if message.Envelope.From.Email != "sender@example.com" {
		t.Errorf("From email = %q, want %q", message.Envelope.From.Email, "sender@example.com")
	}
	
	// Test that message ID is generated
	if message.MessageID == "" {
		t.Error("MessageID should not be empty")
	}
	
	// Test that date is set
	if message.Date.IsZero() {
		t.Error("Date should not be zero")
	}
	
	// Test that date is recent (within last minute)
	if time.Since(message.Date) > time.Minute {
		t.Error("Date should be recent")
	}
}

// Test default from address
func TestDefaultFromAddress(t *testing.T) {
	config := &MailConfig{
		DefaultMailer: "mock",
		From:          Address{Email: "default@example.com", Name: "Default Sender"},
		Mailers: map[string]MailerConfig{
			"mock": {Transport: "mock"},
		},
	}
	
	manager := NewMailManager(config)
	
	// Mailable without explicit from address
	mailable := NewBaseMailable().
		To("recipient@example.com", "Recipient").
		Subject("Test Subject").
		Text("Test content")
	
	message, err := manager.buildMessage(mailable)
	if err != nil {
		t.Fatalf("Failed to build message: %v", err)
	}
	
	// Should use default from address
	if message.Envelope.From.Email != "default@example.com" {
		t.Errorf("From email = %q, want %q", message.Envelope.From.Email, "default@example.com")
	}
	if message.Envelope.From.Name != "Default Sender" {
		t.Errorf("From name = %q, want %q", message.Envelope.From.Name, "Default Sender")
	}
}

// Test Base64 encoding
func TestBase64Encoding(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "Empty data",
			input:    []byte{},
			expected: "",
		},
		{
			name:     "Simple text",
			input:    []byte("Hello"),
			expected: "SGVsbG8=",
		},
		{
			name:     "Text with padding",
			input:    []byte("Hello World!"),
			expected: "SGVsbG8gV29ybGQh",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := encodeBase64(tt.input)
			// Remove line breaks for comparison
			result = strings.ReplaceAll(result, "\r\n", "")
			if result != tt.expected {
				t.Errorf("encodeBase64() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// Test message ID generation
func TestMessageIDGeneration(t *testing.T) {
	id1 := generateMessageID()
	id2 := generateMessageID()
	
	// IDs should not be empty
	if id1 == "" {
		t.Error("First message ID should not be empty")
	}
	if id2 == "" {
		t.Error("Second message ID should not be empty")
	}
	
	// IDs should be different
	if id1 == id2 {
		t.Error("Message IDs should be unique")
	}
	
	// IDs should have proper format
	if !strings.HasPrefix(id1, "<") || !strings.HasSuffix(id1, "@onyx>") {
		t.Errorf("Message ID format incorrect: %s", id1)
	}
}

// Integration test with custom mailable
type TestMailable struct {
	*BaseMailable
	UserName string
	OrderID  string
}

func NewTestMailable(userName, orderID string) *TestMailable {
	mailable := &TestMailable{
		BaseMailable: NewBaseMailable(),
		UserName:     userName,
		OrderID:      orderID,
	}
	
	mailable.Subject(fmt.Sprintf("Order Confirmation #%s", orderID)).
		Text(fmt.Sprintf("Hello %s, your order #%s has been confirmed.", userName, orderID)).
		HTML(fmt.Sprintf("<h1>Hello %s</h1><p>Your order #%s has been confirmed.</p>", userName, orderID))
	
	return mailable
}

func (m *TestMailable) GetData() map[string]interface{} {
	data := m.BaseMailable.GetData()
	data["UserName"] = m.UserName
	data["OrderID"] = m.OrderID
	return data
}

func TestCustomMailable(t *testing.T) {
	config := &MailConfig{
		DefaultMailer: "mock",
		From:          Address{Email: "orders@example.com", Name: "Order System"},
		Mailers: map[string]MailerConfig{
			"mock": {Transport: "mock"},
		},
	}
	
	manager := NewMailManager(config)
	mockDriver := NewMockMailDriver("mock")
	manager.RegisterDriver("mock", mockDriver)
	
	// Create custom mailable
	mailable := NewTestMailable("John Doe", "12345")
	mailable.To("john@example.com", "John Doe")
	
	// Send the email
	err := manager.Send(mailable)
	if err != nil {
		t.Fatalf("Failed to send email: %v", err)
	}
	
	// Verify the email was sent correctly
	sentMessages := mockDriver.GetSentMessages()
	if len(sentMessages) != 1 {
		t.Fatalf("Expected 1 sent message, got %d", len(sentMessages))
	}
	
	message := sentMessages[0]
	
	// Check subject includes order ID
	expectedSubject := "Order Confirmation #12345"
	if message.Envelope.Subject != expectedSubject {
		t.Errorf("Subject = %q, want %q", message.Envelope.Subject, expectedSubject)
	}
	
	// Check text content includes user name and order ID
	if !strings.Contains(message.Content.Text, "John Doe") {
		t.Error("Text content should contain user name")
	}
	if !strings.Contains(message.Content.Text, "12345") {
		t.Error("Text content should contain order ID")
	}
	
	// Check HTML content
	if !strings.Contains(message.Content.HTML, "<h1>Hello John Doe</h1>") {
		t.Error("HTML content should contain proper greeting")
	}
}