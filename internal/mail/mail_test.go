package mail

import (
	"testing"
	"time"
)

func TestAddress_String(t *testing.T) {
	tests := []struct {
		name     string
		address  Address
		expected string
	}{
		{
			name:     "email only",
			address:  Address{Email: "test@example.com"},
			expected: "test@example.com",
		},
		{
			name:     "email with name",
			address:  Address{Email: "test@example.com", Name: "Test User"},
			expected: "Test User <test@example.com>",
		},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := test.address.String()
			if result != test.expected {
				t.Errorf("Expected %s, got %s", test.expected, result)
			}
		})
	}
}

func TestAddress_Validate(t *testing.T) {
	tests := []struct {
		name    string
		address Address
		wantErr bool
	}{
		{
			name:    "valid email",
			address: Address{Email: "test@example.com", Name: "Test"},
			wantErr: false,
		},
		{
			name:    "empty email",
			address: Address{Email: "", Name: "Test"},
			wantErr: true,
		},
		{
			name:    "invalid email - no @",
			address: Address{Email: "testexample.com", Name: "Test"},
			wantErr: true,
		},
		{
			name:    "invalid email - no domain",
			address: Address{Email: "test@", Name: "Test"},
			wantErr: true,
		},
		{
			name:    "invalid email - no local part",
			address: Address{Email: "@example.com", Name: "Test"},
			wantErr: true,
		},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.address.Validate()
			if test.wantErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !test.wantErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestNewMailable(t *testing.T) {
	mailable := NewMailable()
	
	if mailable.envelope == nil {
		t.Error("Envelope should be initialized")
	}
	
	if mailable.content == nil {
		t.Error("Content should be initialized")
	}
	
	if mailable.attachments == nil {
		t.Error("Attachments should be initialized")
	}
	
	if mailable.data == nil {
		t.Error("Data should be initialized")
	}
	
	if len(mailable.envelope.Headers) != 0 {
		t.Error("Headers should be empty initially")
	}
	
	if mailable.envelope.Priority != PriorityNormal {
		t.Error("Priority should default to normal")
	}
}

func TestMailable_FluentInterface(t *testing.T) {
	mailable := NewMailable().
		From("sender@example.com", "Sender").
		To("recipient@example.com", "Recipient").
		CC("cc@example.com", "CC User").
		BCC("bcc@example.com", "BCC User").
		ReplyTo("reply@example.com", "Reply").
		Subject("Test Subject").
		Priority(PriorityHigh).
		Text("Test text content").
		HTML("<p>Test HTML content</p>").
		With("name", "John").
		Header("X-Custom", "custom-value").
		Tag("newsletter").
		Meta("campaign", "test")
	
	envelope := mailable.Envelope()
	content := mailable.Content()
	
	// Test envelope
	if envelope.From.Email != "sender@example.com" {
		t.Error("From email not set correctly")
	}
	
	if len(envelope.To) != 1 || envelope.To[0].Email != "recipient@example.com" {
		t.Error("To address not set correctly")
	}
	
	if len(envelope.CC) != 1 || envelope.CC[0].Email != "cc@example.com" {
		t.Error("CC address not set correctly")
	}
	
	if len(envelope.BCC) != 1 || envelope.BCC[0].Email != "bcc@example.com" {
		t.Error("BCC address not set correctly")
	}
	
	if len(envelope.ReplyTo) != 1 || envelope.ReplyTo[0].Email != "reply@example.com" {
		t.Error("Reply-To address not set correctly")
	}
	
	if envelope.Subject != "Test Subject" {
		t.Error("Subject not set correctly")
	}
	
	if envelope.Priority != PriorityHigh {
		t.Error("Priority not set correctly")
	}
	
	if envelope.Headers["X-Custom"] != "custom-value" {
		t.Error("Custom header not set correctly")
	}
	
	if len(envelope.Tags) != 1 || envelope.Tags[0] != "newsletter" {
		t.Error("Tag not set correctly")
	}
	
	if envelope.Metadata["campaign"] != "test" {
		t.Error("Metadata not set correctly")
	}
	
	// Test content
	if content.Text != "Test text content" {
		t.Error("Text content not set correctly")
	}
	
	if content.HTML != "<p>Test HTML content</p>" {
		t.Error("HTML content not set correctly")
	}
	
	if content.TemplateData["name"] != "John" {
		t.Error("Template data not set correctly")
	}
}

func TestMailable_Attachments(t *testing.T) {
	mailable := NewMailable().
		Attach("/path/to/file.pdf", "document.pdf").
		AttachData([]byte("test content"), "test.txt", "text/plain").
		AttachInline([]byte("image data"), "image.png", "image/png", "img1")
	
	attachments := mailable.Attachments()
	
	if len(attachments) != 3 {
		t.Errorf("Expected 3 attachments, got %d", len(attachments))
	}
	
	// File attachment
	if attachments[0].FilePath != "/path/to/file.pdf" {
		t.Error("File path not set correctly")
	}
	if attachments[0].Name != "document.pdf" {
		t.Error("File name not set correctly")
	}
	
	// Data attachment
	if string(attachments[1].Content) != "test content" {
		t.Error("Attachment content not set correctly")
	}
	if attachments[1].ContentType != "text/plain" {
		t.Error("Content type not set correctly")
	}
	
	// Inline attachment
	if attachments[2].ContentID != "img1" {
		t.Error("Content ID not set correctly")
	}
	if attachments[2].Disposition != "inline" {
		t.Error("Disposition not set correctly")
	}
}

func TestMailable_Validation(t *testing.T) {
	// Valid mailable
	mailable := NewMailable().
		From("sender@example.com", "Sender").
		To("recipient@example.com", "Recipient").
		Subject("Test Subject").
		Text("Test content")
	
	if err := mailable.IsValid(); err != nil {
		t.Errorf("Valid mailable should pass validation: %v", err)
	}
	
	// Invalid - no from
	invalid1 := NewMailable().
		To("recipient@example.com", "Recipient").
		Subject("Test Subject").
		Text("Test content")
	
	if err := invalid1.IsValid(); err == nil {
		t.Error("Mailable without from should fail validation")
	}
	
	// Invalid - no recipients
	invalid2 := NewMailable().
		From("sender@example.com", "Sender").
		Subject("Test Subject").
		Text("Test content")
	
	if err := invalid2.IsValid(); err == nil {
		t.Error("Mailable without recipients should fail validation")
	}
	
	// Invalid - no subject
	invalid3 := NewMailable().
		From("sender@example.com", "Sender").
		To("recipient@example.com", "Recipient").
		Text("Test content")
	
	if err := invalid3.IsValid(); err == nil {
		t.Error("Mailable without subject should fail validation")
	}
	
	// Invalid - no content
	invalid4 := NewMailable().
		From("sender@example.com", "Sender").
		To("recipient@example.com", "Recipient").
		Subject("Test Subject")
	
	if err := invalid4.IsValid(); err == nil {
		t.Error("Mailable without content should fail validation")
	}
}

func TestMessage_Validation(t *testing.T) {
	// Valid message
	message := &Message{
		Envelope: &Envelope{
			From:    &Address{Email: "sender@example.com", Name: "Sender"},
			To:      []*Address{{Email: "recipient@example.com", Name: "Recipient"}},
			Subject: "Test Subject",
		},
		Content: &Content{
			Text: "Test content",
		},
		Attachments: []*Attachment{},
	}
	
	if err := message.Validate(); err != nil {
		t.Errorf("Valid message should pass validation: %v", err)
	}
	
	// Invalid - no envelope
	invalid1 := &Message{
		Content: &Content{Text: "Test content"},
	}
	
	if err := invalid1.Validate(); err == nil {
		t.Error("Message without envelope should fail validation")
	}
	
	// Invalid - no content
	invalid2 := &Message{
		Envelope: &Envelope{
			From:    &Address{Email: "sender@example.com"},
			To:      []*Address{{Email: "recipient@example.com"}},
			Subject: "Test Subject",
		},
	}
	
	if err := invalid2.Validate(); err == nil {
		t.Error("Message without content should fail validation")
	}
}

func TestMessage_GetSize(t *testing.T) {
	message := &Message{
		Envelope: &Envelope{
			Subject: "Test Subject",
			To:      []*Address{{Email: "test@example.com"}},
		},
		Content: &Content{
			Text: "Test content",
			HTML: "<p>Test content</p>",
		},
		Attachments: []*Attachment{
			{
				Content: []byte("attachment content"),
				Size:    18,
			},
		},
	}
	
	size := message.GetSize()
	if size <= 0 {
		t.Error("Message size should be greater than 0")
	}
	
	// Size should include content and attachments
	expectedMinSize := int64(len("Test content") + len("<p>Test content</p>") + 18)
	if size < expectedMinSize {
		t.Errorf("Message size %d should be at least %d", size, expectedMinSize)
	}
}

func TestMessage_Clone(t *testing.T) {
	original := &Message{
		ID:        "original-id",
		MessageID: "original-message-id",
		Envelope: &Envelope{
			From:    &Address{Email: "sender@example.com", Name: "Sender"},
			To:      []*Address{{Email: "recipient@example.com", Name: "Recipient"}},
			Subject: "Original Subject",
			Headers: map[string]string{"X-Custom": "value"},
		},
		Content: &Content{
			Text:         "Original text",
			TemplateData: map[string]interface{}{"name": "John"},
		},
		Attachments: []*Attachment{
			{
				Name:    "test.txt",
				Content: []byte("test content"),
			},
		},
	}
	
	clone := original.Clone()
	
	// Verify clone has same data
	if clone.ID != original.ID {
		t.Error("Clone should have same ID")
	}
	
	if clone.Envelope.Subject != original.Envelope.Subject {
		t.Error("Clone should have same subject")
	}
	
	// Verify they are separate objects
	clone.Envelope.Subject = "Modified Subject"
	if original.Envelope.Subject == "Modified Subject" {
		t.Error("Modifying clone should not affect original")
	}
	
	// Verify deep copy of attachments
	clone.Attachments[0].Content[0] = 'X'
	if original.Attachments[0].Content[0] == 'X' {
		t.Error("Modifying clone attachment should not affect original")
	}
}

func TestGenerateMessageID(t *testing.T) {
	id1 := GenerateMessageID()
	id2 := GenerateMessageID()
	
	if id1 == id2 {
		t.Error("Generated message IDs should be unique")
	}
	
	if id1 == "" || id2 == "" {
		t.Error("Generated message IDs should not be empty")
	}
	
	// Check format
	if !containsString(id1, "@onyx>") {
		t.Error("Message ID should contain @onyx>")
	}
}

func TestStatsCollector(t *testing.T) {
	collector := NewStatsCollector()
	
	// Record some events
	collector.RecordSuccess("smtp", 100*time.Millisecond)
	collector.RecordSuccess("smtp", 200*time.Millisecond)
	collector.RecordFailure("smtp", 50*time.Millisecond)
	collector.RecordQueued()
	collector.RecordRateLimitHit("smtp")
	
	stats := collector.GetStats()
	
	if stats.SentCount != 2 {
		t.Errorf("Expected 2 sent messages, got %d", stats.SentCount)
	}
	
	if stats.FailedCount != 1 {
		t.Errorf("Expected 1 failed message, got %d", stats.FailedCount)
	}
	
	if stats.QueuedCount != 1 {
		t.Errorf("Expected 1 queued message, got %d", stats.QueuedCount)
	}
	
	if stats.RateLimitHits != 1 {
		t.Errorf("Expected 1 rate limit hit, got %d", stats.RateLimitHits)
	}
	
	if stats.SuccessRate != 2.0/3.0 {
		t.Errorf("Expected success rate of 0.667, got %f", stats.SuccessRate)
	}
	
	// Check driver stats
	driverStats, exists := stats.DriverStats["smtp"]
	if !exists {
		t.Error("Expected SMTP driver stats")
	}
	
	if driverStats.SentCount != 2 {
		t.Errorf("Expected 2 sent for SMTP driver, got %d", driverStats.SentCount)
	}
	
	if driverStats.FailedCount != 1 {
		t.Errorf("Expected 1 failed for SMTP driver, got %d", driverStats.FailedCount)
	}
}

func TestRateLimiter(t *testing.T) {
	limiter := NewRateLimiter(3, 1*time.Second)
	
	// Should allow first 3 requests
	for i := 0; i < 3; i++ {
		if !limiter.Allow() {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}
	
	// 4th request should be denied
	if limiter.Allow() {
		t.Error("4th request should be denied")
	}
	
	// After waiting, should allow again
	time.Sleep(1100 * time.Millisecond) // Wait for refill
	if !limiter.Allow() {
		t.Error("Request after refill should be allowed")
	}
}

func TestValidationHelper(t *testing.T) {
	helper := &ValidationHelper{}
	
	// Test email validation
	if err := helper.ValidateEmail("valid@example.com"); err != nil {
		t.Errorf("Valid email should pass validation: %v", err)
	}
	
	if err := helper.ValidateEmail("invalid-email"); err == nil {
		t.Error("Invalid email should fail validation")
	}
	
	// Test mailable validation
	validMailable := NewMailable().
		From("sender@example.com", "Sender").
		To("recipient@example.com", "Recipient").
		Subject("Test").
		Text("Content")
	
	if err := helper.ValidateMailable(validMailable); err != nil {
		t.Errorf("Valid mailable should pass validation: %v", err)
	}
	
	invalidMailable := NewMailable()
	if err := helper.ValidateMailable(invalidMailable); err == nil {
		t.Error("Invalid mailable should fail validation")
	}
}

// Helper function for testing
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s[:len(substr)] == substr || 
		(len(s) > len(substr) && s[len(s)-len(substr):] == substr) ||
		contains(s, substr))
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}