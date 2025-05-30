package onyx

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Test models for event testing
type TestEventModel struct {
	BaseModel
	Name  string `db:"name" json:"name"`
	Email string `db:"email" json:"email"`
}

func (tem *TestEventModel) TableName() string {
	return "test_event_models"
}

func (tem *TestEventModel) GetModelName() string {
	return "TestEventModel"
}

// Test observer for tracking events
type TestObserver struct {
	BaseModelLifecycleObserver
	Events []string
}

func (to *TestObserver) Creating(ctx context.Context, model interface{}) error {
	to.Events = append(to.Events, "creating")
	return nil
}

func (to *TestObserver) Created(ctx context.Context, model interface{}) error {
	to.Events = append(to.Events, "created")
	return nil
}

func (to *TestObserver) Updating(ctx context.Context, model interface{}) error {
	to.Events = append(to.Events, "updating")
	return nil
}

func (to *TestObserver) Updated(ctx context.Context, model interface{}) error {
	to.Events = append(to.Events, "updated")
	return nil
}

func (to *TestObserver) Saving(ctx context.Context, model interface{}) error {
	to.Events = append(to.Events, "saving")
	return nil
}

func (to *TestObserver) Saved(ctx context.Context, model interface{}) error {
	to.Events = append(to.Events, "saved")
	return nil
}

func (to *TestObserver) Deleting(ctx context.Context, model interface{}) error {
	to.Events = append(to.Events, "deleting")
	return nil
}

func (to *TestObserver) Deleted(ctx context.Context, model interface{}) error {
	to.Events = append(to.Events, "deleted")
	return nil
}

// Test observer that prevents operations
type PreventingObserver struct {
	BaseModelLifecycleObserver
	PreventCreate bool
	PreventUpdate bool
	PreventDelete bool
}

func (po *PreventingObserver) Creating(ctx context.Context, model interface{}) error {
	if po.PreventCreate {
		return fmt.Errorf("creation prevented by observer")
	}
	return nil
}

func (po *PreventingObserver) Updating(ctx context.Context, model interface{}) error {
	if po.PreventUpdate {
		return fmt.Errorf("update prevented by observer")
	}
	return nil
}

func (po *PreventingObserver) Deleting(ctx context.Context, model interface{}) error {
	if po.PreventDelete {
		return fmt.Errorf("deletion prevented by observer")
	}
	return nil
}

// Setup test database for model events
func setupEventTestDB(t *testing.T) (*DB, func()) {
	// Create temporary database file
	tempFile := fmt.Sprintf("/tmp/test_events_%d.db", time.Now().UnixNano())
	
	db, err := NewDB("sqlite3", tempFile)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	
	// Create test table
	createTableSQL := `
	CREATE TABLE test_event_models (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name VARCHAR(255),
		email VARCHAR(255),
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`
	
	_, err = db.Exec(createTableSQL)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}
	
	// Cleanup function
	cleanup := func() {
		db.Close()
		os.Remove(tempFile)
	}
	
	return db, cleanup
}

func TestEventDispatcher(t *testing.T) {
	dispatcher := NewModelEventDispatcher()
	
	// Test observer registration
	observer := &TestObserver{Events: make([]string, 0)}
	dispatcher.RegisterObserver("TestEventModel", observer)
	
	// Test handler registration
	handlerCalled := false
	handler := func(ctx context.Context, model interface{}) error {
		handlerCalled = true
		return nil
	}
	dispatcher.RegisterHandler("TestEventModel", EventCreating, handler)
	
	// Test event dispatching  
	model := &TestEventModel{Name: "Test"}
	ctx := context.Background()
	
	// The model name should be extracted properly
	err := dispatcher.DispatchEvent(ctx, EventCreating, model)
	if err != nil {
		t.Errorf("DispatchEvent failed: %v", err)
	}
	
	// Check observer was called
	if len(observer.Events) != 1 || observer.Events[0] != "creating" {
		t.Errorf("Expected observer to be called with 'creating', got: %v", observer.Events)
	}
	
	// Check handler was called
	if !handlerCalled {
		t.Error("Handler was not called")
	}
}

func TestModelEventLifecycle(t *testing.T) {
	db, cleanup := setupEventTestDB(t)
	defer cleanup()
	
	// Clear any existing observers
	globalModelEventDispatcher = NewModelEventDispatcher()
	
	// Register test observer
	observer := &TestObserver{Events: make([]string, 0)}
	ObserveModel("TestEventModel", observer)
	
	ctx := context.Background()
	
	// Test CREATE lifecycle
	model := &TestEventModel{
		Name:  "John Doe",
		Email: "john@example.com",
	}
	
	err := CreateModel(ctx, db, model)
	if err != nil {
		t.Fatalf("CreateModel failed: %v", err)
	}
	
	expectedCreateEvents := []string{"saving", "creating", "created", "saved"}
	if len(observer.Events) != len(expectedCreateEvents) {
		t.Fatalf("Expected %d events, got %d: %v", len(expectedCreateEvents), len(observer.Events), observer.Events)
	}
	
	for i, expected := range expectedCreateEvents {
		if observer.Events[i] != expected {
			t.Errorf("Event %d: expected '%s', got '%s'", i, expected, observer.Events[i])
		}
	}
	
	// Verify model was created with ID
	if model.ID == 0 {
		t.Error("Model ID should be set after creation")
	}
	
	if !model.Exists() {
		t.Error("Model should be marked as existing after creation")
	}
	
	// Test UPDATE lifecycle
	observer.Events = make([]string, 0) // Reset events
	
	// Make a change
	model.Name = "Jane Doe"
	model.MarkAsDirty("name", model.Name)
	
	err = UpdateModel(ctx, db, model)
	if err != nil {
		t.Fatalf("UpdateModel failed: %v", err)
	}
	
	expectedUpdateEvents := []string{"saving", "updating", "updated", "saved"}
	if len(observer.Events) != len(expectedUpdateEvents) {
		t.Fatalf("Expected %d events, got %d: %v", len(expectedUpdateEvents), len(observer.Events), observer.Events)
	}
	
	for i, expected := range expectedUpdateEvents {
		if observer.Events[i] != expected {
			t.Errorf("Event %d: expected '%s', got '%s'", i, expected, observer.Events[i])
		}
	}
	
	// Test DELETE lifecycle
	observer.Events = make([]string, 0) // Reset events
	
	err = DeleteModel(ctx, db, model)
	if err != nil {
		t.Fatalf("DeleteModel failed: %v", err)
	}
	
	expectedDeleteEvents := []string{"deleting", "deleted"}
	if len(observer.Events) != len(expectedDeleteEvents) {
		t.Fatalf("Expected %d events, got %d: %v", len(expectedDeleteEvents), len(observer.Events), observer.Events)
	}
	
	for i, expected := range expectedDeleteEvents {
		if observer.Events[i] != expected {
			t.Errorf("Event %d: expected '%s', got '%s'", i, expected, observer.Events[i])
		}
	}
	
	// Verify model is no longer marked as existing
	if model.Exists() {
		t.Error("Model should not be marked as existing after deletion")
	}
}

func TestEventHandlerRegistration(t *testing.T) {
	// Clear global dispatcher
	globalModelEventDispatcher = NewModelEventDispatcher()
	
	// Test convenience functions
	creatingCalled := false
	OnCreating("TestEventModel", func(ctx context.Context, model interface{}) error {
		creatingCalled = true
		return nil
	})
	
	createdCalled := false
	OnCreated("TestEventModel", func(ctx context.Context, model interface{}) error {
		createdCalled = true
		return nil
	})
	
	updatingCalled := false
	OnUpdating("TestEventModel", func(ctx context.Context, model interface{}) error {
		updatingCalled = true
		return nil
	})
	
	updatedCalled := false
	OnUpdated("TestEventModel", func(ctx context.Context, model interface{}) error {
		updatedCalled = true
		return nil
	})
	
	savingCalled := false
	OnSaving("TestEventModel", func(ctx context.Context, model interface{}) error {
		savingCalled = true
		return nil
	})
	
	savedCalled := false
	OnSaved("TestEventModel", func(ctx context.Context, model interface{}) error {
		savedCalled = true
		return nil
	})
	
	deletingCalled := false
	OnDeleting("TestEventModel", func(ctx context.Context, model interface{}) error {
		deletingCalled = true
		return nil
	})
	
	deletedCalled := false
	OnDeleted("TestEventModel", func(ctx context.Context, model interface{}) error {
		deletedCalled = true
		return nil
	})
	
	// Test all events
	ctx := context.Background()
	model := &TestEventModel{}
	dispatcher := GetModelEventDispatcher()
	
	events := []ModelEvent{
		EventCreating, EventCreated, EventUpdating, EventUpdated,
		EventSaving, EventSaved, EventDeleting, EventDeleted,
	}
	
	for _, event := range events {
		err := dispatcher.DispatchEvent(ctx, event, model)
		if err != nil {
			t.Errorf("DispatchEvent failed for %s: %v", event, err)
		}
	}
	
	// Verify all handlers were called
	if !creatingCalled {
		t.Error("Creating handler was not called")
	}
	if !createdCalled {
		t.Error("Created handler was not called")
	}
	if !updatingCalled {
		t.Error("Updating handler was not called")
	}
	if !updatedCalled {
		t.Error("Updated handler was not called")
	}
	if !savingCalled {
		t.Error("Saving handler was not called")
	}
	if !savedCalled {
		t.Error("Saved handler was not called")
	}
	if !deletingCalled {
		t.Error("Deleting handler was not called")
	}
	if !deletedCalled {
		t.Error("Deleted handler was not called")
	}
}

func TestEventPrevention(t *testing.T) {
	db, cleanup := setupEventTestDB(t)
	defer cleanup()
	
	// Clear global dispatcher
	globalModelEventDispatcher = NewModelEventDispatcher()
	
	ctx := context.Background()
	
	// Test CREATE prevention
	preventingObserver := &PreventingObserver{PreventCreate: true}
	ObserveModel("TestEventModel", preventingObserver)
	
	model := &TestEventModel{
		Name:  "John Doe",
		Email: "john@example.com",
	}
	
	err := CreateModel(ctx, db, model)
	if err == nil {
		t.Error("Expected CreateModel to be prevented by observer")
	}
	
	if !IsModelEventError(err) {
		t.Error("Expected error to be a ModelEventError")
	}
	
	// Verify model was not created
	if model.ID != 0 {
		t.Error("Model ID should not be set when creation is prevented")
	}
	
	// Test UPDATE prevention
	globalModelEventDispatcher = NewModelEventDispatcher() // Clear
	preventingObserver = &PreventingObserver{PreventUpdate: true}
	ObserveModel("TestEventModel", preventingObserver)
	
	// First create a model successfully
	model = &TestEventModel{
		Name:  "John Doe",
		Email: "john@example.com",
	}
	
	// Manually create without events to test update prevention
	createSQL := `INSERT INTO test_event_models (name, email, created_at, updated_at) VALUES (?, ?, ?, ?)`
	result, err := db.Exec(createSQL, model.Name, model.Email, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("Failed to create test model: %v", err)
	}
	
	id, _ := result.LastInsertId()
	model.SetID(uint(id))
	model.MarkAsDirty("name", "Jane Doe")
	
	err = UpdateModel(ctx, db, model)
	if err == nil {
		t.Error("Expected UpdateModel to be prevented by observer")
	}
	
	// Test DELETE prevention
	globalModelEventDispatcher = NewModelEventDispatcher() // Clear
	preventingObserver = &PreventingObserver{PreventDelete: true}
	ObserveModel("TestEventModel", preventingObserver)
	
	err = DeleteModel(ctx, db, model)
	if err == nil {
		t.Error("Expected DeleteModel to be prevented by observer")
	}
}

func TestDirtyFieldTracking(t *testing.T) {
	model := &TestEventModel{
		Name:  "John Doe",
		Email: "john@example.com",
	}
	
	baseModel := &model.BaseModel
	baseModel.InitializeModel()
	
	// Test initial state
	if baseModel.IsDirty() {
		t.Error("New model should not be dirty")
	}
	
	// Test marking fields as dirty
	baseModel.MarkAsDirty("name", "Jane Doe")
	if !baseModel.IsDirty() {
		t.Error("Model should be dirty after marking field as dirty")
	}
	
	dirtyFields := baseModel.GetDirtyFields()
	if len(dirtyFields) != 1 {
		t.Errorf("Expected 1 dirty field, got %d", len(dirtyFields))
	}
	
	if dirtyFields["name"] != "Jane Doe" {
		t.Errorf("Expected dirty field 'name' to be 'Jane Doe', got %v", dirtyFields["name"])
	}
	
	// Test syncing original
	baseModel.syncOriginal()
	if baseModel.IsDirty() {
		t.Error("Model should not be dirty after syncing original")
	}
}

func TestModelEventError(t *testing.T) {
	err := &ModelEventError{
		Event:     EventCreating,
		ModelName: "TestModel",
		Err:       fmt.Errorf("test error"),
	}
	
	expectedMsg := "model event error: creating on TestModel: test error"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
	
	// Test error unwrapping
	if err.Unwrap().Error() != "test error" {
		t.Errorf("Expected unwrapped error to be 'test error', got '%s'", err.Unwrap().Error())
	}
	
	// Test IsModelEventError
	if !IsModelEventError(err) {
		t.Error("IsModelEventError should return true for ModelEventError")
	}
	
	if IsModelEventError(fmt.Errorf("normal error")) {
		t.Error("IsModelEventError should return false for normal error")
	}
}

func TestSaveModel(t *testing.T) {
	db, cleanup := setupEventTestDB(t)
	defer cleanup()
	
	// Clear global dispatcher
	globalModelEventDispatcher = NewModelEventDispatcher()
	
	observer := &TestObserver{Events: make([]string, 0)}
	ObserveModel("TestEventModel", observer)
	
	ctx := context.Background()
	
	// Test save new model (should create)
	model := &TestEventModel{
		Name:  "John Doe",
		Email: "john@example.com",
	}
	
	err := SaveModel(ctx, db, model)
	if err != nil {
		t.Fatalf("SaveModel failed: %v", err)
	}
	
	// Should have called create events
	expectedCreateEvents := []string{"saving", "creating", "created", "saved"}
	if len(observer.Events) != len(expectedCreateEvents) {
		t.Fatalf("Expected %d events, got %d: %v", len(expectedCreateEvents), len(observer.Events), observer.Events)
	}
	
	// Test save existing model (should update)
	observer.Events = make([]string, 0) // Reset
	
	model.Name = "Jane Doe"
	model.MarkAsDirty("name", model.Name)
	
	err = SaveModel(ctx, db, model)
	if err != nil {
		t.Fatalf("SaveModel failed: %v", err)
	}
	
	// Should have called update events
	expectedUpdateEvents := []string{"saving", "updating", "updated", "saved"}
	if len(observer.Events) != len(expectedUpdateEvents) {
		t.Fatalf("Expected %d events, got %d: %v", len(expectedUpdateEvents), len(observer.Events), observer.Events)
	}
}

func TestMultipleObservers(t *testing.T) {
	// Clear global dispatcher
	globalModelEventDispatcher = NewModelEventDispatcher()
	
	observer1 := &TestObserver{Events: make([]string, 0)}
	observer2 := &TestObserver{Events: make([]string, 0)}
	
	ObserveModel("TestEventModel", observer1)
	ObserveModel("TestEventModel", observer2)
	
	ctx := context.Background()
	model := &TestEventModel{}
	dispatcher := GetModelEventDispatcher()
	
	err := dispatcher.DispatchEvent(ctx, EventCreating, model)
	if err != nil {
		t.Errorf("DispatchEvent failed: %v", err)
	}
	
	// Both observers should have been called
	if len(observer1.Events) != 1 || observer1.Events[0] != "creating" {
		t.Errorf("Observer1 expected ['creating'], got %v", observer1.Events)
	}
	
	if len(observer2.Events) != 1 || observer2.Events[0] != "creating" {
		t.Errorf("Observer2 expected ['creating'], got %v", observer2.Events)
	}
}