package database

import (
	"time"
)

// SoftDeletes provides soft delete functionality for query builders
type SoftDeletes struct {
	qb QueryBuilder
}

// NewSoftDeletes creates a new soft deletes instance
func NewSoftDeletes(qb QueryBuilder) *SoftDeletes {
	return &SoftDeletes{qb: qb}
}

// WithTrashed includes soft-deleted records in the query
func (sd *SoftDeletes) WithTrashed() QueryBuilder {
	return sd.qb.WithTrashed()
}

// OnlyTrashed only returns soft-deleted records
func (sd *SoftDeletes) OnlyTrashed() QueryBuilder {
	return sd.qb.OnlyTrashed()
}

// WithoutTrashed excludes soft-deleted records (default behavior)
func (sd *SoftDeletes) WithoutTrashed() QueryBuilder {
	// This is the default behavior, so we just return the query builder
	return sd.qb
}

// Delete performs a soft delete
func (sd *SoftDeletes) Delete() error {
	now := time.Now()
	updateData := map[string]interface{}{
		"deleted_at": now,
		"updated_at": now,
	}
	
	_, err := sd.qb.Update(updateData)
	return err
}

// Restore restores soft-deleted records
func (sd *SoftDeletes) Restore() error {
	updateData := map[string]interface{}{
		"deleted_at": nil,
		"updated_at": time.Now(),
	}
	
	_, err := sd.qb.Update(updateData)
	return err
}

// ForceDelete permanently deletes records
func (sd *SoftDeletes) ForceDelete() error {
	_, err := sd.qb.ForceDelete()
	return err
}

// SoftDeleteQueryBuilder extends QueryBuilder with soft delete methods
type SoftDeleteQueryBuilder struct {
	QueryBuilder
	*SoftDeletes
}

// NewSoftDeleteQueryBuilder creates a query builder with soft delete capabilities
func NewSoftDeleteQueryBuilder(qb QueryBuilder) *SoftDeleteQueryBuilder {
	return &SoftDeleteQueryBuilder{
		QueryBuilder: qb,
		SoftDeletes:  NewSoftDeletes(qb),
	}
}

// Soft delete helper functions for models

// SoftDeleteModel performs a soft delete on a model
func SoftDeleteModel(db Database, model EventableModel) error {
	return DeleteModel(db, model)
}

// RestoreSoftDeletedModel restores a soft-deleted model
func RestoreSoftDeletedModel(db Database, model EventableModel) error {
	return RestoreModel(db, model)
}

// IsSoftDeleted checks if a model is soft deleted
func IsSoftDeleted(model EventableModel) bool {
	deletedAt := model.GetDeletedAt()
	return deletedAt.Valid
}

// SoftDeleteScope applies soft delete filtering to a query
func SoftDeleteScope(qb QueryBuilder, includeTrashed bool) QueryBuilder {
	if includeTrashed {
		return qb.WithTrashed()
	}
	// Default behavior excludes soft-deleted records
	return qb
}

// OnlyTrashedScope applies only-trashed filtering to a query
func OnlyTrashedScope(qb QueryBuilder) QueryBuilder {
	return qb.OnlyTrashed()
}

// RestoreScope for bulk restoring records
func RestoreScope(qb QueryBuilder) error {
	updateData := map[string]interface{}{
		"deleted_at": nil,
		"updated_at": time.Now(),
	}
	
	_, err := qb.Update(updateData)
	return err
}

// ForceDeleteScope for bulk force deleting records
func ForceDeleteScope(qb QueryBuilder) error {
	_, err := qb.ForceDelete()
	return err
}

// SoftDeleteConfig holds configuration for soft deletes
type SoftDeleteConfig struct {
	DeletedAtColumn string
	UpdatedAtColumn string
	Enabled         bool
}

// DefaultSoftDeleteConfig returns the default soft delete configuration
func DefaultSoftDeleteConfig() *SoftDeleteConfig {
	return &SoftDeleteConfig{
		DeletedAtColumn: "deleted_at",
		UpdatedAtColumn: "updated_at",
		Enabled:         true,
	}
}

// SoftDeleteTracker tracks soft delete operations
type SoftDeleteTracker struct {
	config  *SoftDeleteConfig
	deleted []interface{}
	restored []interface{}
}

// NewSoftDeleteTracker creates a new soft delete tracker
func NewSoftDeleteTracker(config *SoftDeleteConfig) *SoftDeleteTracker {
	if config == nil {
		config = DefaultSoftDeleteConfig()
	}
	
	return &SoftDeleteTracker{
		config:   config,
		deleted:  make([]interface{}, 0),
		restored: make([]interface{}, 0),
	}
}

// TrackDeleted tracks a deleted model
func (sdt *SoftDeleteTracker) TrackDeleted(model interface{}) {
	sdt.deleted = append(sdt.deleted, model)
}

// TrackRestored tracks a restored model
func (sdt *SoftDeleteTracker) TrackRestored(model interface{}) {
	sdt.restored = append(sdt.restored, model)
}

// GetDeleted returns all tracked deleted models
func (sdt *SoftDeleteTracker) GetDeleted() []interface{} {
	return sdt.deleted
}

// GetRestored returns all tracked restored models
func (sdt *SoftDeleteTracker) GetRestored() []interface{} {
	return sdt.restored
}

// Clear clears all tracked operations
func (sdt *SoftDeleteTracker) Clear() {
	sdt.deleted = make([]interface{}, 0)
	sdt.restored = make([]interface{}, 0)
}

// HasDeleted checks if any models were deleted
func (sdt *SoftDeleteTracker) HasDeleted() bool {
	return len(sdt.deleted) > 0
}

// HasRestored checks if any models were restored
func (sdt *SoftDeleteTracker) HasRestored() bool {
	return len(sdt.restored) > 0
}

// GetConfig returns the soft delete configuration
func (sdt *SoftDeleteTracker) GetConfig() *SoftDeleteConfig {
	return sdt.config
}