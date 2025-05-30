package database

import (
	"testing"
	"time"
)

// Test model for testing
type TestModel struct {
	BaseModel
	Name  string `db:"name" json:"name"`
	Email string `db:"email" json:"email"`
}

func (tm *TestModel) TableName() string {
	return "test_models"
}

func TestBaseModel(t *testing.T) {
	model := &TestModel{
		Name:  "Test User",
		Email: "test@example.com",
	}

	// Test initial state
	if !model.IsNew() {
		t.Error("New model should be marked as new")
	}

	if model.IsSoftDeleted() {
		t.Error("New model should not be soft deleted")
	}

	if model.IsDirty() {
		t.Error("New model should not have dirty fields")
	}

	// Test marking as existing
	model.SetIDValue(1)
	if model.IsNew() {
		t.Error("Model with ID should not be marked as new")
	}

	if !model.Exists() {
		t.Error("Model with ID should be marked as existing")
	}

	// Test dirty tracking
	model.MarkAsDirty("name", "Updated Name")
	if !model.IsDirty() {
		t.Error("Model should be dirty after marking field as dirty")
	}

	dirtyFields := model.GetDirtyFields()
	if len(dirtyFields) != 1 {
		t.Errorf("Expected 1 dirty field, got %d", len(dirtyFields))
	}

	if dirtyFields["name"] != "Updated Name" {
		t.Errorf("Expected dirty field name to be 'Updated Name', got %v", dirtyFields["name"])
	}
}

func TestModelInterfaces(t *testing.T) {
	model := &TestModel{}

	// Test Model interface
	if model.TableName() != "test_models" {
		t.Errorf("Expected table name 'test_models', got %s", model.TableName())
	}

	// Test EventableModel interface methods
	if model.GetID() != uint(0) {
		t.Errorf("Expected ID 0, got %v", model.GetID())
	}

	model.SetID(uint(42))
	if model.GetID() != uint(42) {
		t.Errorf("Expected ID 42, got %v", model.GetID())
	}

	if !model.IsEventable() {
		t.Error("BaseModel should be eventable")
	}
}

func TestSoftDeletes(t *testing.T) {
	model := &TestModel{}
	model.SetIDValue(1)

	// Test initial state
	if model.IsSoftDeleted() {
		t.Error("Model should not be soft deleted initially")
	}

	// Test soft delete
	now := time.Now()
	model.DeletedAt = &now

	if !model.IsSoftDeleted() {
		t.Error("Model should be soft deleted after setting DeletedAt")
	}

	// Test restore
	model.DeletedAt = nil
	if model.IsSoftDeleted() {
		t.Error("Model should not be soft deleted after clearing DeletedAt")
	}
}

func TestExtractModelFields(t *testing.T) {
	model := &TestModel{
		BaseModel: BaseModel{
			ID:        1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Name:  "Test User",
		Email: "test@example.com",
	}

	fields := extractModelFields(model)

	// Debug: Print what fields were actually extracted
	t.Logf("Extracted fields: %+v", fields)

	// Note: ID field with value 1 should be extracted, along with timestamps and custom fields
	expectedFields := []string{"id", "created_at", "updated_at", "name", "email"}
	for _, field := range expectedFields {
		if _, exists := fields[field]; !exists {
			t.Errorf("Expected field %s to be extracted", field)
		}
	}

	if fields["name"] != "Test User" {
		t.Errorf("Expected name field to be 'Test User', got %v", fields["name"])
	}

	if fields["email"] != "test@example.com" {
		t.Errorf("Expected email field to be 'test@example.com', got %v", fields["email"])
	}
}

func TestScanner(t *testing.T) {
	scanner := NewScanner()

	if scanner == nil {
		t.Error("NewScanner should return a scanner instance")
	}

	// Just test that we can create a scanner
	// More detailed testing would require actual database connections
}

func TestQueryBuilderCreation(t *testing.T) {
	// This is a basic test to ensure the structure compiles
	// More comprehensive tests would require a database connection
	
	db := &DB{driver: "sqlite3"}
	qb := NewQueryBuilder(db)

	if qb == nil {
		t.Error("NewQueryBuilder should return a query builder instance")
	}

	// Test method chaining
	qb = qb.Select("id", "name").
		Where("active", "=", true).
		OrderBy("created_at", "DESC").
		Limit(10)

	if qb == nil {
		t.Error("Query builder method chaining should work")
	}
}

func TestSoftDeleteConfig(t *testing.T) {
	config := DefaultSoftDeleteConfig()

	if config.DeletedAtColumn != "deleted_at" {
		t.Errorf("Expected deleted_at column, got %s", config.DeletedAtColumn)
	}

	if config.UpdatedAtColumn != "updated_at" {
		t.Errorf("Expected updated_at column, got %s", config.UpdatedAtColumn)
	}

	if !config.Enabled {
		t.Error("Soft deletes should be enabled by default")
	}
}

func TestSoftDeleteTracker(t *testing.T) {
	tracker := NewSoftDeleteTracker(nil)

	if tracker.HasDeleted() {
		t.Error("New tracker should not have deleted models")
	}

	if tracker.HasRestored() {
		t.Error("New tracker should not have restored models")
	}

	// Track some operations
	model1 := &TestModel{Name: "Model 1"}
	model2 := &TestModel{Name: "Model 2"}

	tracker.TrackDeleted(model1)
	tracker.TrackRestored(model2)

	if !tracker.HasDeleted() {
		t.Error("Tracker should have deleted models")
	}

	if !tracker.HasRestored() {
		t.Error("Tracker should have restored models")
	}

	deleted := tracker.GetDeleted()
	if len(deleted) != 1 {
		t.Errorf("Expected 1 deleted model, got %d", len(deleted))
	}

	restored := tracker.GetRestored()
	if len(restored) != 1 {
		t.Errorf("Expected 1 restored model, got %d", len(restored))
	}

	// Clear and test
	tracker.Clear()
	if tracker.HasDeleted() || tracker.HasRestored() {
		t.Error("Tracker should be empty after clearing")
	}
}