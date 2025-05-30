package onyx

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	
	// Enable foreign key constraints for SQLite
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		t.Fatalf("Failed to enable foreign key constraints: %v", err)
	}
	
	cleanup := func() {
		// Clean up any remaining tables before closing
		rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'")
		if err == nil {
			var tables []string
			for rows.Next() {
				var tableName string
				if err := rows.Scan(&tableName); err == nil {
					tables = append(tables, tableName)
				}
			}
			rows.Close()
			
			// Drop tables in reverse order to handle foreign key dependencies
			for i := len(tables) - 1; i >= 0; i-- {
				db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", tables[i]))
			}
		}
		
		db.Close()
		os.Remove(dbPath)
	}
	
	return db, cleanup
}

func TestSchemaBuilderCreate(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	schema := NewSchemaBuilder(db, "sqlite3")
	
	err := schema.Create("test_table", func(table Table) {
		table.ID()
		table.String("name")
		table.Integer("age")
		table.Boolean("active")
		table.Timestamps()
	})
	
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
	
	// Verify table exists
	exists, err := schema.HasTable("test_table")
	if err != nil {
		t.Fatalf("Failed to check table existence: %v", err)
	}
	
	if !exists {
		t.Error("Table was not created")
	}
	
	// Verify columns exist
	columns, err := schema.GetColumnListing("test_table")
	if err != nil {
		t.Fatalf("Failed to get column listing: %v", err)
	}
	
	expectedColumns := []string{"id", "name", "age", "active", "created_at", "updated_at"}
	if len(columns) != len(expectedColumns) {
		t.Errorf("Expected %d columns, got %d", len(expectedColumns), len(columns))
	}
	
	for _, expected := range expectedColumns {
		found := false
		for _, column := range columns {
			if column == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected column '%s' not found", expected)
		}
	}
}

func TestSchemaBuilderAlterTable(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	schema := NewSchemaBuilder(db, "sqlite3")
	
	// Create initial table
	err := schema.Create("alter_test", func(table Table) {
		table.ID()
		table.String("name")
	})
	if err != nil {
		t.Fatalf("Failed to create initial table: %v", err)
	}
	
	// Alter table to add columns
	err = schema.Table("alter_test", func(table Table) {
		table.String("email")
		table.Integer("age")
	})
	if err != nil {
		t.Fatalf("Failed to alter table: %v", err)
	}
	
	// Verify new columns exist
	hasEmail, err := schema.HasColumn("alter_test", "email")
	if err != nil {
		t.Fatalf("Failed to check email column: %v", err)
	}
	if !hasEmail {
		t.Error("Email column was not added")
	}
	
	hasAge, err := schema.HasColumn("alter_test", "age")
	if err != nil {
		t.Fatalf("Failed to check age column: %v", err)
	}
	if !hasAge {
		t.Error("Age column was not added")
	}
}

func TestSchemaBuilderDropTable(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	schema := NewSchemaBuilder(db, "sqlite3")
	
	// Create table
	err := schema.Create("drop_test", func(table Table) {
		table.ID()
		table.String("name")
	})
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
	
	// Verify table exists
	exists, err := schema.HasTable("drop_test")
	if err != nil {
		t.Fatalf("Failed to check table existence: %v", err)
	}
	if !exists {
		t.Fatal("Table was not created")
	}
	
	// Drop table
	err = schema.Drop("drop_test")
	if err != nil {
		t.Fatalf("Failed to drop table: %v", err)
	}
	
	// Verify table no longer exists
	exists, err = schema.HasTable("drop_test")
	if err != nil {
		t.Fatalf("Failed to check table existence after drop: %v", err)
	}
	if exists {
		t.Error("Table still exists after drop")
	}
}

func TestSchemaBuilderDropIfExists(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	schema := NewSchemaBuilder(db, "sqlite3")
	
	// Drop non-existent table (should not error)
	err := schema.DropIfExists("nonexistent_table")
	if err != nil {
		t.Fatalf("DropIfExists failed on non-existent table: %v", err)
	}
	
	// Create and drop existing table
	err = schema.Create("existing_table", func(table Table) {
		table.ID()
	})
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
	
	err = schema.DropIfExists("existing_table")
	if err != nil {
		t.Fatalf("DropIfExists failed on existing table: %v", err)
	}
	
	// Verify table is gone
	exists, err := schema.HasTable("existing_table")
	if err != nil {
		t.Fatalf("Failed to check table existence: %v", err)
	}
	if exists {
		t.Error("Table still exists after DropIfExists")
	}
}

func TestColumnTypes(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	schema := NewSchemaBuilder(db, "sqlite3")
	
	err := schema.Create("column_types_test", func(table Table) {
		table.ID()
		table.String("varchar_col", 100)
		table.Text("text_col")
		table.Integer("int_col")
		table.BigInteger("bigint_col")
		table.Boolean("bool_col")
		table.Float("float_col", 8, 2)
		table.Decimal("decimal_col", 10, 2)
		table.Date("date_col")
		table.DateTime("datetime_col")
		table.Timestamp("timestamp_col")
		table.JSON("json_col")
		table.UUID("uuid_col")
		table.Enum("enum_col", []string{"option1", "option2", "option3"})
	})
	
	if err != nil {
		t.Fatalf("Failed to create table with various column types: %v", err)
	}
	
	// Verify table was created successfully
	exists, err := schema.HasTable("column_types_test")
	if err != nil {
		t.Fatalf("Failed to check table existence: %v", err)
	}
	if !exists {
		t.Error("Table with various column types was not created")
	}
}

func TestColumnModifiers(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	schema := NewSchemaBuilder(db, "sqlite3")
	
	err := schema.Create("modifiers_test", func(table Table) {
		table.ID()
		table.String("required_col").NotNull()
		table.String("nullable_col").Nullable()
		table.String("default_col").Default("default_value")
		table.String("unique_col").Unique()
		table.Integer("auto_inc_col").AutoIncrement()
		table.String("comment_col").Comment("This is a comment")
	})
	
	if err != nil {
		t.Fatalf("Failed to create table with column modifiers: %v", err)
	}
	
	// Verify table was created
	exists, err := schema.HasTable("modifiers_test")
	if err != nil {
		t.Fatalf("Failed to check table existence: %v", err)
	}
	if !exists {
		t.Error("Table with column modifiers was not created")
	}
}

func TestIndexes(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	schema := NewSchemaBuilder(db, "sqlite3")
	
	err := schema.Create("indexes_test", func(table Table) {
		table.ID()
		table.String("name")
		table.String("email")
		table.String("slug")
		
		// Add indexes
		table.Index([]string{"name"})
		table.Unique([]string{"email"})
		table.Index([]string{"name", "email"}, "compound_index")
	})
	
	if err != nil {
		t.Fatalf("Failed to create table with indexes: %v", err)
	}
	
	// Verify table was created
	exists, err := schema.HasTable("indexes_test")
	if err != nil {
		t.Fatalf("Failed to check table existence: %v", err)
	}
	if !exists {
		t.Error("Table with indexes was not created")
	}
}

func TestForeignKeys(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	schema := NewSchemaBuilder(db, "sqlite3")
	
	// Create parent table
	err := schema.Create("users", func(table Table) {
		table.ID()
		table.String("name")
		table.String("email")
	})
	if err != nil {
		t.Fatalf("Failed to create users table: %v", err)
	}
	
	// Create child table with foreign key
	err = schema.Create("posts", func(table Table) {
		table.ID()
		table.String("title")
		table.Text("content")
		table.ForeignID("user_id")
		
		table.Foreign("user_id").References("id").On("users").OnDelete("CASCADE")
	})
	if err != nil {
		t.Fatalf("Failed to create posts table with foreign key: %v", err)
	}
	
	// Verify tables were created
	usersExists, err := schema.HasTable("users")
	if err != nil {
		t.Fatalf("Failed to check users table existence: %v", err)
	}
	if !usersExists {
		t.Error("Users table was not created")
	}
	
	postsExists, err := schema.HasTable("posts")
	if err != nil {
		t.Fatalf("Failed to check posts table existence: %v", err)
	}
	if !postsExists {
		t.Error("Posts table was not created")
	}
}

func TestSpecialMethods(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	schema := NewSchemaBuilder(db, "sqlite3")
	
	err := schema.Create("special_test", func(table Table) {
		table.ID()
		table.String("name")
		table.Timestamps()
		table.SoftDeletes()
		table.RememberToken()
		table.Morphs("taggable")
	})
	
	if err != nil {
		t.Fatalf("Failed to create table with special methods: %v", err)
	}
	
	// Verify table was created
	exists, err := schema.HasTable("special_test")
	if err != nil {
		t.Fatalf("Failed to check table existence: %v", err)
	}
	if !exists {
		t.Error("Table with special methods was not created")
	}
	
	// Check for specific columns added by special methods
	columns, err := schema.GetColumnListing("special_test")
	if err != nil {
		t.Fatalf("Failed to get column listing: %v", err)
	}
	
	expectedColumns := []string{
		"id", "name", "created_at", "updated_at", "deleted_at", 
		"remember_token", "taggable_type", "taggable_id",
	}
	
	for _, expected := range expectedColumns {
		found := false
		for _, column := range columns {
			if column == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected column '%s' from special method not found", expected)
		}
	}
}

func TestBaseMigration(t *testing.T) {
	migration := NewBaseMigration("2024_01_01_120000_create_test_table")
	
	if migration.GetName() != "2024_01_01_120000_create_test_table" {
		t.Errorf("Expected name '2024_01_01_120000_create_test_table', got '%s'", migration.GetName())
	}
	
	expectedTimestamp := "2024_01_01_120000"
	if migration.GetTimestamp() != expectedTimestamp {
		t.Errorf("Expected timestamp '%s', got '%s'", expectedTimestamp, migration.GetTimestamp())
	}
	
	if migration.GetBatch() != 0 {
		t.Errorf("Expected batch 0, got %d", migration.GetBatch())
	}
	
	migration.SetBatch(5)
	if migration.GetBatch() != 5 {
		t.Errorf("Expected batch 5 after setting, got %d", migration.GetBatch())
	}
}

func TestMigrator(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	migrator := NewMigrator(db, "sqlite3")
	
	// Test migration registration
	migration1 := &TestMigration{
		BaseMigration: NewBaseMigration("2024_01_01_000001_create_users"),
		schema:        migrator.schema,
		tableName:     "test_users",
	}
	migration2 := &TestMigration{
		BaseMigration: NewBaseMigration("2024_01_01_000002_create_posts"),
		schema:        migrator.schema,
		tableName:     "test_posts",
	}
	
	migrator.Register(migration1)
	migrator.Register(migration2)
	
	// Test migration run
	err := migrator.Run()
	if err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}
	
	// Verify migrations table was created
	exists, err := migrator.schema.HasTable("migrations")
	if err != nil {
		t.Fatalf("Failed to check migrations table: %v", err)
	}
	if !exists {
		t.Error("Migrations table was not created")
	}
	
	// Verify migrations were logged
	ranMigrations, err := migrator.getRanMigrations()
	if err != nil {
		t.Fatalf("Failed to get ran migrations: %v", err)
	}
	
	if len(ranMigrations) != 2 {
		t.Errorf("Expected 2 ran migrations, got %d", len(ranMigrations))
	}
	
	// Test migration status
	statuses, err := migrator.Status()
	if err != nil {
		t.Fatalf("Failed to get migration status: %v", err)
	}
	
	if len(statuses) != 2 {
		t.Errorf("Expected 2 migration statuses, got %d", len(statuses))
	}
	
	for _, status := range statuses {
		if !status.Ran {
			t.Errorf("Migration %s should be marked as ran", status.Name)
		}
		if status.Batch != 1 {
			t.Errorf("Expected batch 1 for migration %s, got %d", status.Name, status.Batch)
		}
	}
}

func TestMigratorRollback(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	migrator := NewMigrator(db, "sqlite3")
	
	// Create test migrations
	migration1 := &TestMigration{
		BaseMigration: NewBaseMigration("2024_01_01_000001_create_test1"),
		schema:        migrator.schema,
		tableName:     "test1",
	}
	migration2 := &TestMigration{
		BaseMigration: NewBaseMigration("2024_01_01_000002_create_test2"),
		schema:        migrator.schema,
		tableName:     "test2",
	}
	
	migrator.Register(migration1)
	migrator.Register(migration2)
	
	// Run migrations
	err := migrator.Run()
	if err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}
	
	// Verify tables exist
	exists1, _ := migrator.schema.HasTable("test1")
	exists2, _ := migrator.schema.HasTable("test2")
	if !exists1 || !exists2 {
		t.Fatal("Migration tables were not created")
	}
	
	// Rollback last batch
	err = migrator.Rollback(1)
	if err != nil {
		t.Fatalf("Failed to rollback migrations: %v", err)
	}
	
	// Verify tables are gone
	exists1, _ = migrator.schema.HasTable("test1")
	exists2, _ = migrator.schema.HasTable("test2")
	if exists1 || exists2 {
		t.Error("Tables still exist after rollback")
	}
	
	// Verify no migrations are recorded as ran
	ranMigrations, err := migrator.getRanMigrations()
	if err != nil {
		t.Fatalf("Failed to get ran migrations: %v", err)
	}
	
	if len(ranMigrations) != 0 {
		t.Errorf("Expected 0 ran migrations after rollback, got %d", len(ranMigrations))
	}
}

func TestMigratorReset(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	migrator := NewMigrator(db, "sqlite3")
	
	// Create and run test migrations
	migration := &TestMigration{
		BaseMigration: NewBaseMigration("2024_01_01_000001_create_reset_test"),
		schema:        migrator.schema,
		tableName:     "reset_test",
	}
	
	migrator.Register(migration)
	
	err := migrator.Run()
	if err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}
	
	// Verify table exists
	exists, _ := migrator.schema.HasTable("reset_test")
	if !exists {
		t.Fatal("Migration table was not created")
	}
	
	// Reset all migrations
	err = migrator.Reset()
	if err != nil {
		t.Fatalf("Failed to reset migrations: %v", err)
	}
	
	// Verify table is gone
	exists, _ = migrator.schema.HasTable("reset_test")
	if exists {
		t.Error("Table still exists after reset")
	}
	
	// Verify no migrations are recorded
	ranMigrations, err := migrator.getRanMigrations()
	if err != nil {
		t.Fatalf("Failed to get ran migrations: %v", err)
	}
	
	if len(ranMigrations) != 0 {
		t.Errorf("Expected 0 ran migrations after reset, got %d", len(ranMigrations))
	}
}

func TestSQLGeneration(t *testing.T) {
	tests := []struct {
		name   string
		driver string
	}{
		{"MySQL", "mysql"},
		{"PostgreSQL", "postgres"},
		{"SQLite", "sqlite3"},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			table := NewTableBuilder("test_table", "create")
			table.ID()
			table.String("name", 100)
			table.Integer("age")
			table.Boolean("active")
			table.Timestamps()
			
			sql := table.ToSQL(test.driver)
			
			if len(sql) == 0 {
				t.Errorf("No SQL generated for driver %s", test.driver)
			}
			
			// Check that main CREATE TABLE statement is present
			if !strings.Contains(sql[0], "CREATE TABLE test_table") {
				t.Errorf("CREATE TABLE statement not found in SQL for driver %s", test.driver)
			}
			
			// Check that columns are present
			if !strings.Contains(sql[0], "id") ||
			   !strings.Contains(sql[0], "name") ||
			   !strings.Contains(sql[0], "age") ||
			   !strings.Contains(sql[0], "active") {
				t.Errorf("Required columns not found in SQL for driver %s", test.driver)
			}
		})
	}
}

func TestColumnSQLGeneration(t *testing.T) {
	tests := []struct {
		name     string
		driver   string
		column   *ColumnBuilder
		expected string
	}{
		{
			name:   "VARCHAR with length",
			driver: "mysql",
			column: &ColumnBuilder{
				name:     "test_col",
				dataType: "VARCHAR",
				length:   255,
				nullable: true,
			},
			expected: "test_col VARCHAR(255)",
		},
		{
			name:   "INT with NOT NULL",
			driver: "mysql",
			column: &ColumnBuilder{
				name:     "test_col",
				dataType: "INT",
				nullable: false,
			},
			expected: "test_col INT NOT NULL",
		},
		{
			name:   "AUTO_INCREMENT",
			driver: "mysql",
			column: &ColumnBuilder{
				name:          "test_col",
				dataType:      "INT",
				autoIncrement: true,
				nullable:      false,
			},
			expected: "test_col INT NOT NULL AUTO_INCREMENT",
		},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sql := test.column.ToSQL(test.driver)
			if !strings.Contains(sql, test.expected) {
				t.Errorf("Expected SQL to contain '%s', got '%s'", test.expected, sql)
			}
		})
	}
}

func TestMakeMigration(t *testing.T) {
	tempDir := t.TempDir()
	
	err := MakeMigration("create_test_table", tempDir)
	if err != nil {
		t.Fatalf("Failed to make migration: %v", err)
	}
	
	// Check that migration file was created
	files, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read temp directory: %v", err)
	}
	
	if len(files) != 1 {
		t.Errorf("Expected 1 migration file, got %d", len(files))
	}
	
	migrationFile := files[0]
	if !strings.HasSuffix(migrationFile.Name(), "_create_test_table.go") {
		t.Errorf("Migration file has wrong name: %s", migrationFile.Name())
	}
	
	// Check file contents
	content, err := os.ReadFile(filepath.Join(tempDir, migrationFile.Name()))
	if err != nil {
		t.Fatalf("Failed to read migration file: %v", err)
	}
	
	contentStr := string(content)
	if !strings.Contains(contentStr, "CreateTestTable") {
		t.Error("Migration file doesn't contain proper struct name")
	}
	
	if !strings.Contains(contentStr, "func (m *CreateTestTable) Up()") {
		t.Error("Migration file doesn't contain Up method")
	}
	
	if !strings.Contains(contentStr, "func (m *CreateTestTable) Down()") {
		t.Error("Migration file doesn't contain Down method")
	}
}

func TestExtractTimestamp(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"2024_01_01_120000_create_users_table", "2024_01_01_120000"},
		{"2023_12_31_235959_update_posts", "2023_12_31_235959"},
		{"invalid_format", ""},
	}
	
	for _, test := range tests {
		result := extractTimestamp(test.input)
		if result != test.expected {
			t.Errorf("extractTimestamp(%s): expected '%s', got '%s'", test.input, test.expected, result)
		}
	}
}

func TestGenerateIndexName(t *testing.T) {
	result := generateIndexName("users", "idx", []string{"name", "email"})
	expected := "idx_users_name_email"
	if result != expected {
		t.Errorf("generateIndexName: expected '%s', got '%s'", expected, result)
	}
}

func TestGenerateForeignKeyName(t *testing.T) {
	result := generateForeignKeyName("posts", "user_id")
	expected := "fk_posts_user_id"
	if result != expected {
		t.Errorf("generateForeignKeyName: expected '%s', got '%s'", expected, result)
	}
}

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"create_users_table", "CreateUsersTable"},
		{"update_posts", "UpdatePosts"},
		{"single", "Single"},
		{"", ""},
	}
	
	for _, test := range tests {
		result := toCamelCase(test.input)
		if result != test.expected {
			t.Errorf("toCamelCase(%s): expected '%s', got '%s'", test.input, test.expected, result)
		}
	}
}

// Test migration implementation for testing
type TestMigration struct {
	*BaseMigration
	schema    SchemaBuilder
	tableName string
}

func (tm *TestMigration) Up() error {
	tableName := tm.tableName
	if tableName == "" {
		tableName = "test_table"
	}
	
	return tm.schema.Create(tableName, func(table Table) {
		table.ID()
		table.String("name")
		table.Timestamps()
	})
}

func (tm *TestMigration) Down() error {
	tableName := tm.tableName
	if tableName == "" {
		tableName = "test_table"
	}
	
	return tm.schema.Drop(tableName)
}

// Example tests for the provided CreateUsersTable migration
func TestCreateUsersTable(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	schema := NewSchemaBuilder(db, "sqlite3")
	migration := NewCreateUsersTable(schema)
	
	// Test Up migration
	err := migration.Up()
	if err != nil {
		t.Fatalf("Failed to run Up migration: %v", err)
	}
	
	// Verify table exists
	exists, err := schema.HasTable("users")
	if err != nil {
		t.Fatalf("Failed to check table existence: %v", err)
	}
	if !exists {
		t.Error("Users table was not created")
	}
	
	// Verify columns
	expectedColumns := []string{"id", "name", "email", "password", "remember_token", "created_at", "updated_at"}
	for _, column := range expectedColumns {
		hasColumn, err := schema.HasColumn("users", column)
		if err != nil {
			t.Fatalf("Failed to check column %s: %v", column, err)
		}
		if !hasColumn {
			t.Errorf("Column %s was not created", column)
		}
	}
	
	// Test Down migration
	err = migration.Down()
	if err != nil {
		t.Fatalf("Failed to run Down migration: %v", err)
	}
	
	// Verify table is gone
	exists, err = schema.HasTable("users")
	if err != nil {
		t.Fatalf("Failed to check table existence after Down: %v", err)
	}
	if exists {
		t.Error("Users table still exists after Down migration")
	}
}

func TestSQLiteAutoIncrement(t *testing.T) {
	// Test SQLite ID column generation
	table := NewTableBuilder("test_table", "create")
	table.ID()
	
	sql := table.ToSQL("sqlite3")
	if len(sql) == 0 {
		t.Fatal("No SQL generated")
	}
	
	createSQL := sql[0]
	t.Logf("SQLite ID column SQL: %s", createSQL)
	
	// Verify it contains "INTEGER PRIMARY KEY AUTOINCREMENT" instead of "BIGINT NOT NULL AUTOINCREMENT"
	if !strings.Contains(createSQL, "INTEGER PRIMARY KEY AUTOINCREMENT") {
		t.Errorf("Expected 'INTEGER PRIMARY KEY AUTOINCREMENT' in SQL, got: %s", createSQL)
	}
	
	if strings.Contains(createSQL, "BIGINT NOT NULL AUTOINCREMENT") {
		t.Errorf("Found incorrect 'BIGINT NOT NULL AUTOINCREMENT' syntax in SQL: %s", createSQL)
	}
	
	// Test BigIncrements column generation
	table2 := NewTableBuilder("test_table2", "create")
	table2.BigIncrements("my_id")
	
	sql2 := table2.ToSQL("sqlite3")
	if len(sql2) == 0 {
		t.Fatal("No SQL generated for BigIncrements")
	}
	
	createSQL2 := sql2[0]
	t.Logf("SQLite BigIncrements column SQL: %s", createSQL2)
	
	// Verify it contains "INTEGER PRIMARY KEY AUTOINCREMENT" instead of "BIGINT NOT NULL AUTOINCREMENT"
	if !strings.Contains(createSQL2, "INTEGER PRIMARY KEY AUTOINCREMENT") {
		t.Errorf("Expected 'INTEGER PRIMARY KEY AUTOINCREMENT' in BigIncrements SQL, got: %s", createSQL2)
	}
	
	if strings.Contains(createSQL2, "BIGINT NOT NULL AUTOINCREMENT") {
		t.Errorf("Found incorrect 'BIGINT NOT NULL AUTOINCREMENT' syntax in BigIncrements SQL: %s", createSQL2)
	}
}

// Integration test
func TestMigrationIntegration(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	migrator := NewMigrator(db, "sqlite3")
	
	// Register multiple migrations
	usersTable := NewCreateUsersTable(migrator.schema)
	postsTable := &TestMigration{
		BaseMigration: NewBaseMigration("2024_01_01_000002_create_posts_table"),
		schema:        migrator.schema,
		tableName:     "posts",
	}
	
	migrator.Register(usersTable)
	migrator.Register(postsTable)
	
	// Run migrations
	err := migrator.Run()
	if err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}
	
	// Verify both tables exist
	usersExists, _ := migrator.schema.HasTable("users")
	postsExists, _ := migrator.schema.HasTable("posts")
	
	if !usersExists {
		t.Error("Users table was not created")
	}
	if !postsExists {
		t.Error("Posts table was not created")
	}
	
	// Test status
	statuses, err := migrator.Status()
	if err != nil {
		t.Fatalf("Failed to get migration status: %v", err)
	}
	
	if len(statuses) != 2 {
		t.Errorf("Expected 2 migration statuses, got %d", len(statuses))
	}
	
	// All should be ran
	for _, status := range statuses {
		if !status.Ran {
			t.Errorf("Migration %s should be marked as ran", status.Name)
		}
	}
	
	// Test rollback
	err = migrator.Rollback(1)
	if err != nil {
		t.Fatalf("Failed to rollback: %v", err)
	}
	
	// Verify tables are gone
	usersExists, _ = migrator.schema.HasTable("users")
	postsExists, _ = migrator.schema.HasTable("posts")
	
	if usersExists || postsExists {
		t.Error("Tables still exist after rollback")
	}
}

// Test SQLite-specific features and fixes
func TestSQLiteSpecificFeatures(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	schema := NewSchemaBuilder(db, "sqlite3")

	// Test 1: Multiple ADD COLUMN operations (should be separate statements)
	err := schema.Create("multi_column_test", func(table Table) {
		table.ID()
		table.String("name")
	})
	if err != nil {
		t.Fatalf("Failed to create initial table: %v", err)
	}

	// This should generate separate ALTER TABLE statements for SQLite
	err = schema.Table("multi_column_test", func(table Table) {
		table.String("email")
		table.Integer("age")
		table.Boolean("active")
	})
	if err != nil {
		t.Fatalf("Failed to alter table with multiple columns: %v", err)
	}

	// Verify all columns exist
	expectedColumns := []string{"id", "name", "email", "age", "active"}
	for _, column := range expectedColumns {
		hasColumn, err := schema.HasColumn("multi_column_test", column)
		if err != nil {
			t.Fatalf("Failed to check column %s: %v", column, err)
		}
		if !hasColumn {
			t.Errorf("Column %s was not added", column)
		}
	}

	// Test 2: ENUM types (should use TEXT with CHECK constraint)
	err = schema.Create("enum_test", func(table Table) {
		table.ID()
		table.String("name")
		table.Enum("status", []string{"active", "inactive", "pending"})
	})
	if err != nil {
		t.Fatalf("Failed to create table with ENUM: %v", err)
	}

	// Verify table was created successfully
	exists, err := schema.HasTable("enum_test")
	if err != nil {
		t.Fatalf("Failed to check enum_test table existence: %v", err)
	}
	if !exists {
		t.Error("Table with ENUM column was not created")
	}

	// Test 3: Foreign keys (should be inline for table creation)
	err = schema.Create("users_fk_test", func(table Table) {
		table.ID()
		table.String("name")
	})
	if err != nil {
		t.Fatalf("Failed to create users table: %v", err)
	}

	err = schema.Create("posts_fk_test", func(table Table) {
		table.ID()
		table.String("title")
		table.ForeignID("user_id")
		table.Foreign("user_id").References("id").On("users_fk_test").OnDelete("CASCADE")
	})
	if err != nil {
		t.Fatalf("Failed to create table with foreign key: %v", err)
	}

	// Verify tables exist
	usersExists, err := schema.HasTable("users_fk_test")
	if err != nil {
		t.Fatalf("Failed to check users_fk_test table: %v", err)
	}
	if !usersExists {
		t.Error("Users FK test table was not created")
	}

	postsExists, err := schema.HasTable("posts_fk_test")
	if err != nil {
		t.Fatalf("Failed to check posts_fk_test table: %v", err)
	}
	if !postsExists {
		t.Error("Posts FK test table was not created")
	}

	t.Log("All SQLite-specific features work correctly")
}

// Test SQL generation for different scenarios
func TestSQLiteSpecificSQLGeneration(t *testing.T) {
	// Test ALTER TABLE with multiple columns
	table := NewTableBuilder("test_table", "alter")
	table.String("email")
	table.Integer("age")

	sql := table.ToSQL("sqlite3")
	
	// Should generate separate ALTER TABLE statements
	if len(sql) < 2 {
		t.Errorf("Expected at least 2 separate ALTER TABLE statements, got %d", len(sql))
	}

	for i, stmt := range sql {
		t.Logf("SQLite ALTER statement %d: %s", i+1, stmt)
		if !strings.Contains(stmt, "ALTER TABLE test_table ADD COLUMN") {
			t.Errorf("Statement %d doesn't contain proper ALTER TABLE syntax: %s", i+1, stmt)
		}
	}

	// Test ENUM column generation
	enumTable := NewTableBuilder("enum_table", "create")
	enumTable.ID()
	enumTable.Enum("status", []string{"active", "inactive", "pending"})

	enumSQL := enumTable.ToSQL("sqlite3")
	if len(enumSQL) == 0 {
		t.Fatal("No SQL generated for ENUM table")
	}

	createSQL := enumSQL[0]
	t.Logf("SQLite ENUM table SQL: %s", createSQL)

	// Should contain TEXT type with CHECK constraint
	if !strings.Contains(createSQL, "TEXT CHECK") {
		t.Errorf("ENUM column should use TEXT with CHECK constraint, got: %s", createSQL)
	}

	// Test foreign key generation
	fkTable := NewTableBuilder("fk_table", "create")
	fkTable.ID()
	fkTable.ForeignID("user_id")
	fkTable.Foreign("user_id").References("id").On("users").OnDelete("CASCADE")

	fkSQL := fkTable.ToSQL("sqlite3")
	if len(fkSQL) == 0 {
		t.Fatal("No SQL generated for FK table")
	}

	fkCreateSQL := fkSQL[0]
	t.Logf("SQLite FK table SQL: %s", fkCreateSQL)

	// Should contain inline FOREIGN KEY constraint
	if !strings.Contains(fkCreateSQL, "FOREIGN KEY") || !strings.Contains(fkCreateSQL, "REFERENCES") {
		t.Errorf("Foreign key constraint should be inline for SQLite, got: %s", fkCreateSQL)
	}
}