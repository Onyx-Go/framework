package onyx

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type TestUser struct {
	ID        int        `db:"id" json:"id"`
	Name      string     `db:"name" json:"name"`
	Email     string     `db:"email" json:"email"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
	DeletedAt *time.Time `db:"deleted_at" json:"deleted_at,omitempty"`
}

func (u TestUser) TableName() string {
	return "users"
}

func TestDatabaseORMScanning(t *testing.T) {
	// Create in-memory SQLite database for testing
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create test table
	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			email TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			deleted_at DATETIME
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Insert test data
	_, err = db.Exec(`
		INSERT INTO users (name, email, created_at, updated_at) VALUES 
		('John Doe', 'john@example.com', '2023-01-01 12:00:00', '2023-01-01 12:00:00'),
		('Jane Smith', 'jane@example.com', '2023-01-02 13:00:00', '2023-01-02 13:00:00')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Create Onyx DB wrapper
	onyxDB := &DB{DB: db, driver: "sqlite3"}
	
	// Test First() method - single row scan
	var user TestUser
	err = onyxDB.Table("users").Where("email", "=", "john@example.com").First(&user)
	if err != nil {
		t.Fatalf("Failed to fetch first user: %v", err)
	}

	if user.Name != "John Doe" {
		t.Errorf("Expected name 'John Doe', got '%s'", user.Name)
	}
	if user.Email != "john@example.com" {
		t.Errorf("Expected email 'john@example.com', got '%s'", user.Email)
	}

	// Test Get() method - multiple rows scan
	var users []TestUser
	err = onyxDB.Table("users").OrderBy("id", "asc").Get(&users)
	if err != nil {
		t.Fatalf("Failed to fetch users: %v", err)
	}

	if len(users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(users))
	}

	if users[0].Name != "John Doe" {
		t.Errorf("Expected first user name 'John Doe', got '%s'", users[0].Name)
	}
	if users[1].Name != "Jane Smith" {
		t.Errorf("Expected second user name 'Jane Smith', got '%s'", users[1].Name)
	}

	// Test Model() method
	testUser := TestUser{}
	var modelUser TestUser
	err = onyxDB.Model(testUser).Where("id", "=", 1).First(&modelUser)
	if err != nil {
		t.Fatalf("Failed to fetch user using Model(): %v", err)
	}

	if modelUser.Name != "John Doe" {
		t.Errorf("Expected name 'John Doe', got '%s'", modelUser.Name)
	}
}

func TestDatabaseORMInsertUpdate(t *testing.T) {
	// Create in-memory SQLite database for testing
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create test table
	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			email TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			deleted_at DATETIME
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Create Onyx DB wrapper
	onyxDB := &DB{DB: db, driver: "sqlite3"}

	// Test Insert
	now := time.Now()
	data := map[string]interface{}{
		"name":       "Test User",
		"email":      "test@example.com",
		"created_at": now,
		"updated_at": now,
	}

	id, err := onyxDB.Table("users").Insert(data)
	if err != nil {
		t.Fatalf("Failed to insert user: %v", err)
	}

	if id <= 0 {
		t.Errorf("Expected positive ID, got %d", id)
	}

	// Test Update
	updateData := map[string]interface{}{
		"name": "Updated User",
	}

	affected, err := onyxDB.Table("users").Where("id", "=", id).Update(updateData)
	if err != nil {
		t.Fatalf("Failed to update user: %v", err)
	}

	if affected != 1 {
		t.Errorf("Expected 1 affected row, got %d", affected)
	}

	// Verify update
	var user TestUser
	err = onyxDB.Table("users").Where("id", "=", id).First(&user)
	if err != nil {
		t.Fatalf("Failed to fetch updated user: %v", err)
	}

	if user.Name != "Updated User" {
		t.Errorf("Expected name 'Updated User', got '%s'", user.Name)
	}

	// Test Delete
	affected, err = onyxDB.Table("users").Where("id", "=", id).Delete()
	if err != nil {
		t.Fatalf("Failed to delete user: %v", err)
	}

	if affected != 1 {
		t.Errorf("Expected 1 affected row, got %d", affected)
	}
}