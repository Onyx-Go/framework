package database

import (
	"database/sql"
	"reflect"
	"time"
)

// BaseModel provides common fields and functionality for all models
type BaseModel struct {
	ID        uint         `db:"id" json:"id"`
	CreatedAt time.Time    `db:"created_at" json:"created_at"`
	UpdatedAt time.Time    `db:"updated_at" json:"updated_at"`
	DeletedAt *time.Time   `db:"deleted_at" json:"deleted_at,omitempty"`
	
	// Internal state for tracking changes
	original    map[string]interface{} // Original field values
	dirty       map[string]interface{} // Changed field values
	exists      bool                   // Whether record exists in database
}

// TableName returns the default table name (should be overridden by embedding models)
func (bm *BaseModel) TableName() string {
	return "base_models"
}

// GetID returns the model ID
func (bm *BaseModel) GetID() interface{} {
	return bm.ID
}

// SetID sets the model ID
func (bm *BaseModel) SetID(id interface{}) {
	if idUint, ok := id.(uint); ok {
		bm.ID = idUint
		if idUint > 0 {
			bm.MarkAsExisting()
		}
	}
}

// GetCreatedAt returns the created at timestamp
func (bm *BaseModel) GetCreatedAt() *time.Time {
	return &bm.CreatedAt
}

// GetUpdatedAt returns the updated at timestamp
func (bm *BaseModel) GetUpdatedAt() *time.Time {
	return &bm.UpdatedAt
}

// GetDeletedAt returns the deleted at timestamp
func (bm *BaseModel) GetDeletedAt() *sql.NullTime {
	if bm.DeletedAt == nil {
		return &sql.NullTime{Valid: false}
	}
	return &sql.NullTime{Time: *bm.DeletedAt, Valid: true}
}

// IsEventable returns whether the model supports events
func (bm *BaseModel) IsEventable() bool {
	return true
}

// GetModelName returns the model name for event dispatching
func (bm *BaseModel) GetModelName() string {
	// This will be overridden by embedding models
	return "BaseModel"
}

// InitializeModel initializes the model for change tracking
func (bm *BaseModel) InitializeModel() {
	if bm.original == nil {
		bm.original = make(map[string]interface{})
	}
	if bm.dirty == nil {
		bm.dirty = make(map[string]interface{})
	}
}

// MarkAsExisting marks the model as existing in the database
func (bm *BaseModel) MarkAsExisting() {
	bm.exists = true
	bm.InitializeModel()
	// Store original values
	bm.syncOriginal()
}

// MarkAsDirty marks a field as dirty (changed)
func (bm *BaseModel) MarkAsDirty(field string, value interface{}) {
	bm.InitializeModel()
	bm.dirty[field] = value
}

// IsDirty checks if the model has any dirty fields
func (bm *BaseModel) IsDirty() bool {
	return len(bm.dirty) > 0
}

// GetDirtyFields returns the dirty fields
func (bm *BaseModel) GetDirtyFields() map[string]interface{} {
	if bm.dirty == nil {
		return make(map[string]interface{})
	}
	return bm.dirty
}

// syncOriginal syncs the current state to original
func (bm *BaseModel) syncOriginal() {
	if bm.original == nil {
		bm.original = make(map[string]interface{})
	}
	
	// Use reflection to get current field values
	v := reflect.ValueOf(bm).Elem()
	t := v.Type()
	
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		if dbTag := field.Tag.Get("db"); dbTag != "" && dbTag != "-" {
			bm.original[dbTag] = v.Field(i).Interface()
		}
	}
	
	// Clear dirty fields
	bm.dirty = make(map[string]interface{})
}

// Exists returns whether the model exists in the database
func (bm *BaseModel) Exists() bool {
	return bm.exists
}

// SetIDValue sets the model ID and marks as existing (for uint IDs)
func (bm *BaseModel) SetIDValue(id uint) {
	bm.ID = id
	if id > 0 {
		bm.MarkAsExisting()
	}
}

// IsSoftDeleted checks if the model has been soft deleted
func (bm *BaseModel) IsSoftDeleted() bool {
	return bm.DeletedAt != nil
}

// IsNew checks if the model is new (not persisted to database)
func (bm *BaseModel) IsNew() bool {
	return !bm.exists
}

// GetOriginalValue returns the original value of a field
func (bm *BaseModel) GetOriginalValue(field string) (interface{}, bool) {
	if bm.original == nil {
		return nil, false
	}
	value, exists := bm.original[field]
	return value, exists
}

// GetDirtyValue returns the dirty value of a field
func (bm *BaseModel) GetDirtyValue(field string) (interface{}, bool) {
	if bm.dirty == nil {
		return nil, false
	}
	value, exists := bm.dirty[field]
	return value, exists
}

// RestoreField restores a field to its original value
func (bm *BaseModel) RestoreField(field string) {
	if bm.dirty != nil {
		delete(bm.dirty, field)
	}
}

// RestoreAll restores all fields to their original values
func (bm *BaseModel) RestoreAll() {
	if bm.dirty != nil {
		bm.dirty = make(map[string]interface{})
	}
}

// Touch updates the updated_at timestamp
func (bm *BaseModel) Touch() {
	bm.UpdatedAt = time.Now()
	bm.MarkAsDirty("updated_at", bm.UpdatedAt)
}

// Model operation helper functions

// SaveModel saves a model to the database
func SaveModel(db Database, model EventableModel) error {
	if model.IsNew() {
		return CreateModel(db, model)
	}
	return UpdateModel(db, model)
}

// CreateModel creates a new model in the database
func CreateModel(db Database, model EventableModel) error {
	// Set timestamps
	now := time.Now()
	if baseModel, ok := model.(*BaseModel); ok {
		baseModel.CreatedAt = now
		baseModel.UpdatedAt = now
	}
	
	// Prepare data for insertion
	data := extractModelFields(model)
	
	// Execute insert
	result, err := db.Table(model.TableName()).Insert(data)
	if err != nil {
		return err
	}
	
	// Set the ID
	if id, err := result.LastInsertId(); err == nil {
		model.SetID(uint(id))
	}
	
	return nil
}

// UpdateModel updates an existing model in the database
func UpdateModel(db Database, model EventableModel) error {
	// Only update if there are dirty fields
	if baseModel, ok := model.(*BaseModel); ok {
		if !baseModel.IsDirty() {
			return nil // No changes to save
		}
		
		// Update timestamp
		baseModel.Touch()
		
		// Get dirty fields
		data := baseModel.GetDirtyFields()
		
		// Execute update
		_, err := db.Table(model.TableName()).Where("id", "=", model.GetID()).Update(data)
		if err != nil {
			return err
		}
		
		// Sync original values
		baseModel.syncOriginal()
	}
	
	return nil
}

// DeleteModel soft deletes a model
func DeleteModel(db Database, model EventableModel) error {
	if baseModel, ok := model.(*BaseModel); ok {
		now := time.Now()
		baseModel.DeletedAt = &now
		baseModel.Touch()
		
		data := map[string]interface{}{
			"deleted_at": now,
			"updated_at": baseModel.UpdatedAt,
		}
		
		_, err := db.Table(model.TableName()).Where("id", "=", model.GetID()).Update(data)
		return err
	}
	
	_, err := db.Table(model.TableName()).Where("id", "=", model.GetID()).Delete()
	return err
}

// ForceDeleteModel permanently deletes a model
func ForceDeleteModel(db Database, model EventableModel) error {
	_, err := db.Table(model.TableName()).Where("id", "=", model.GetID()).ForceDelete()
	return err
}

// RestoreModel restores a soft-deleted model
func RestoreModel(db Database, model EventableModel) error {
	if baseModel, ok := model.(*BaseModel); ok {
		baseModel.DeletedAt = nil
		baseModel.Touch()
		
		data := map[string]interface{}{
			"deleted_at": nil,
			"updated_at": baseModel.UpdatedAt,
		}
		
		_, err := db.Table(model.TableName()).Where("id", "=", model.GetID()).Update(data)
		return err
	}
	
	_, err := db.Table(model.TableName()).Where("id", "=", model.GetID()).Restore()
	return err
}

// extractModelFields extracts field values from a model using reflection
func extractModelFields(model interface{}) map[string]interface{} {
	data := make(map[string]interface{})
	
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	
	extractFieldsFromValue(v, data)
	return data
}

// extractFieldsFromValue recursively extracts fields from a reflect.Value
func extractFieldsFromValue(v reflect.Value, data map[string]interface{}) {
	t := v.Type()
	
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)
		
		// Skip unexported fields
		if !field.IsExported() {
			continue
		}
		
		// Handle embedded structs
		if field.Anonymous && fieldValue.Kind() == reflect.Struct {
			extractFieldsFromValue(fieldValue, data)
			continue
		}
		
		// Get database column name from tag
		dbTag := field.Tag.Get("db")
		if dbTag == "" || dbTag == "-" {
			continue
		}
		
		// Skip ID field for new records
		if dbTag == "id" {
			if idValue := fieldValue.Interface(); idValue == uint(0) {
				continue
			}
		}
		
		// Handle different field types
		switch fieldValue.Kind() {
		case reflect.Ptr:
			if fieldValue.IsNil() {
				data[dbTag] = nil
			} else {
				data[dbTag] = fieldValue.Elem().Interface()
			}
		default:
			data[dbTag] = fieldValue.Interface()
		}
	}
}

// getBaseModel extracts BaseModel from an interface
func getBaseModel(model interface{}) *BaseModel {
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	
	// Look for BaseModel field
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if field.Type() == reflect.TypeOf(BaseModel{}) {
			return field.Addr().Interface().(*BaseModel)
		}
	}
	
	// Check if it's directly a BaseModel
	if v.Type() == reflect.TypeOf(BaseModel{}) {
		return v.Addr().Interface().(*BaseModel)
	}
	
	return nil
}

// Ensure BaseModel implements required interfaces
var _ Model = (*BaseModel)(nil)
var _ EventableModel = (*BaseModel)(nil)