package onyx

import (
	"context"
	"testing"
	"time"
)

// SoftDeleteUser model for soft delete testing
type SoftDeleteUser struct {
	BaseModel
	Name  string `db:"name" json:"name"`
	Email string `db:"email" json:"email"`
}

func (u *SoftDeleteUser) TableName() string {
	return "soft_delete_users"
}

func (u *SoftDeleteUser) GetModelName() string {
	return "SoftDeleteUser"
}

func setupSoftDeleteTest(t *testing.T) (*DB, func()) {
	db, err := NewDB("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Create test table with soft delete column
	createTableSQL := `
		CREATE TABLE soft_delete_users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			email TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			deleted_at DATETIME
		)
	`
	
	if _, err := db.Exec(createTableSQL); err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	cleanup := func() {
		db.Close()
	}

	return db, cleanup
}

func TestSoftDelete(t *testing.T) {
	db, cleanup := setupSoftDeleteTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create a test user
	user := &SoftDeleteUser{
		Name:  "John Doe",
		Email: "john@example.com",
	}

	err := CreateModel(ctx, db, user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	if user.ID == 0 {
		t.Fatal("User ID should be set after creation")
	}

	// Verify user exists and is not soft deleted
	if user.IsSoftDeleted() {
		t.Fatal("User should not be soft deleted initially")
	}

	// Soft delete the user
	err = DeleteModel(ctx, db, user)
	if err != nil {
		t.Fatalf("Failed to soft delete user: %v", err)
	}

	// Verify user is now soft deleted
	if !user.IsSoftDeleted() {
		t.Fatal("User should be soft deleted after DeleteModel")
	}

	if user.DeletedAt == nil {
		t.Fatal("DeletedAt should be set after soft delete")
	}

	// Try to find the user - should not be returned in normal queries
	var foundUsers []SoftDeleteUser
	err = db.Table("soft_delete_users").Where("id", "=", user.ID).Get(&foundUsers)
	if err != nil {
		t.Fatalf("Failed to query for user: %v", err)
	}

	if len(foundUsers) != 0 {
		t.Fatal("Soft deleted user should not be returned in normal queries")
	}

	// Find with trashed - should return the user
	err = db.Table("soft_delete_users").WithTrashed().Where("id", "=", user.ID).Get(&foundUsers)
	if err != nil {
		t.Fatalf("Failed to query for user with trashed: %v", err)
	}

	if len(foundUsers) != 1 {
		t.Fatal("Should find user when including trashed records")
	}

	if foundUsers[0].DeletedAt == nil {
		t.Fatal("Found user should have DeletedAt set")
	}
}

func TestOnlyTrashed(t *testing.T) {
	db, cleanup := setupSoftDeleteTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create two users
	user1 := &SoftDeleteUser{Name: "John Doe", Email: "john@example.com"}
	user2 := &SoftDeleteUser{Name: "Jane Doe", Email: "jane@example.com"}

	CreateModel(ctx, db, user1)
	CreateModel(ctx, db, user2)

	// Soft delete only the first user
	DeleteModel(ctx, db, user1)

	// Query only trashed records
	var trashedUsers []SoftDeleteUser
	err := db.Table("soft_delete_users").OnlyTrashed().Get(&trashedUsers)
	if err != nil {
		t.Fatalf("Failed to query only trashed users: %v", err)
	}

	if len(trashedUsers) != 1 {
		t.Fatalf("Expected 1 trashed user, got %d", len(trashedUsers))
	}

	if trashedUsers[0].ID != user1.ID {
		t.Fatal("Wrong user returned in trashed query")
	}
}

func TestRestoreModel(t *testing.T) {
	db, cleanup := setupSoftDeleteTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create and soft delete a user
	user := &SoftDeleteUser{Name: "John Doe", Email: "john@example.com"}
	CreateModel(ctx, db, user)
	DeleteModel(ctx, db, user)

	// Verify user is soft deleted
	if !user.IsSoftDeleted() {
		t.Fatal("User should be soft deleted")
	}

	// Restore the user
	err := RestoreModel(ctx, db, user)
	if err != nil {
		t.Fatalf("Failed to restore user: %v", err)
	}

	// Verify user is no longer soft deleted
	if user.IsSoftDeleted() {
		t.Fatal("User should not be soft deleted after restore")
	}

	if user.DeletedAt != nil {
		t.Fatal("DeletedAt should be nil after restore")
	}

	// Verify user can be found in normal queries
	var foundUsers []SoftDeleteUser
	err = db.Table("soft_delete_users").Where("id", "=", user.ID).Get(&foundUsers)
	if err != nil {
		t.Fatalf("Failed to query for restored user: %v", err)
	}

	if len(foundUsers) != 1 {
		t.Fatal("Should find restored user in normal queries")
	}
}

func TestForceDelete(t *testing.T) {
	db, cleanup := setupSoftDeleteTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create a user
	user := &SoftDeleteUser{Name: "John Doe", Email: "john@example.com"}
	CreateModel(ctx, db, user)

	// Force delete the user (permanent delete)
	err := ForceDeleteModel(ctx, db, user)
	if err != nil {
		t.Fatalf("Failed to force delete user: %v", err)
	}

	// Verify user cannot be found even with trashed
	var foundUsers []SoftDeleteUser
	err = db.Table("soft_delete_users").WithTrashed().Where("id", "=", user.ID).Get(&foundUsers)
	if err != nil {
		t.Fatalf("Failed to query for force deleted user: %v", err)
	}

	if len(foundUsers) != 0 {
		t.Fatal("Force deleted user should not be found even with trashed")
	}
}

func TestQueryBuilderRestore(t *testing.T) {
	db, cleanup := setupSoftDeleteTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create and soft delete multiple users
	user1 := &SoftDeleteUser{Name: "John Doe", Email: "john@example.com"}
	user2 := &SoftDeleteUser{Name: "Jane Doe", Email: "jane@example.com"}

	CreateModel(ctx, db, user1)
	CreateModel(ctx, db, user2)
	DeleteModel(ctx, db, user1)
	DeleteModel(ctx, db, user2)

	// Restore all users using QueryBuilder
	rowsAffected, err := db.Table("soft_delete_users").OnlyTrashed().Restore()
	if err != nil {
		t.Fatalf("Failed to restore users via QueryBuilder: %v", err)
	}

	if rowsAffected != 2 {
		t.Fatalf("Expected 2 rows affected, got %d", rowsAffected)
	}

	// Verify both users are restored
	var activeUsers []SoftDeleteUser
	err = db.Table("soft_delete_users").Get(&activeUsers)
	if err != nil {
		t.Fatalf("Failed to query active users: %v", err)
	}

	if len(activeUsers) != 2 {
		t.Fatalf("Expected 2 active users after restore, got %d", len(activeUsers))
	}
}

func TestQueryBuilderForceDelete(t *testing.T) {
	db, cleanup := setupSoftDeleteTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create and soft delete a user
	user := &SoftDeleteUser{Name: "John Doe", Email: "john@example.com"}
	CreateModel(ctx, db, user)
	DeleteModel(ctx, db, user)

	// Force delete using QueryBuilder
	rowsAffected, err := db.Table("soft_delete_users").OnlyTrashed().ForceDelete()
	if err != nil {
		t.Fatalf("Failed to force delete via QueryBuilder: %v", err)
	}

	if rowsAffected != 1 {
		t.Fatalf("Expected 1 row affected, got %d", rowsAffected)
	}

	// Verify user is permanently deleted
	var foundUsers []SoftDeleteUser
	err = db.Table("soft_delete_users").WithTrashed().Get(&foundUsers)
	if err != nil {
		t.Fatalf("Failed to query for users: %v", err)
	}

	if len(foundUsers) != 0 {
		t.Fatal("Should not find any users after force delete")
	}
}

func TestBaseModelSoftDeleteMethods(t *testing.T) {
	user := &SoftDeleteUser{Name: "John Doe", Email: "john@example.com"}

	// Initially not soft deleted
	if user.IsSoftDeleted() {
		t.Fatal("New user should not be soft deleted")
	}

	// Soft delete the model
	user.SoftDelete()

	if !user.IsSoftDeleted() {
		t.Fatal("User should be soft deleted after SoftDelete()")
	}

	if user.DeletedAt == nil {
		t.Fatal("DeletedAt should be set after SoftDelete()")
	}

	deletedTime := *user.DeletedAt

	// Restore the model
	user.Restore()

	if user.IsSoftDeleted() {
		t.Fatal("User should not be soft deleted after Restore()")
	}

	if user.DeletedAt != nil {
		t.Fatal("DeletedAt should be nil after Restore()")
	}

	// Check that the time was reasonable
	if time.Since(deletedTime) > time.Second {
		t.Fatal("DeletedAt time seems unreasonable")
	}
}