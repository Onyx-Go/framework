package migrations

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Common SQL foreign key actions
const (
	ForeignKeyRestrict = "RESTRICT"
	ForeignKeyCascade  = "CASCADE"
	ForeignKeySetNull  = "SET NULL"
	ForeignKeyNoAction = "NO ACTION"
)

// Common SQL index algorithms
const (
	IndexAlgorithmBTree = "BTREE"
	IndexAlgorithmHash  = "HASH"
)

// Common column types
const (
	ColumnTypeString    = "string"
	ColumnTypeText      = "text"
	ColumnTypeLongText  = "longtext"
	ColumnTypeMediumText = "mediumtext"
	ColumnTypeTinyText  = "tinytext"
	ColumnTypeChar      = "char"
	ColumnTypeInteger   = "integer"
	ColumnTypeBigInteger = "biginteger"
	ColumnTypeSmallInteger = "smallinteger"
	ColumnTypeTinyInteger = "tinyinteger"
	ColumnTypeFloat     = "float"
	ColumnTypeDouble    = "double"
	ColumnTypeDecimal   = "decimal"
	ColumnTypeBoolean   = "boolean"
	ColumnTypeDate      = "date"
	ColumnTypeDateTime  = "datetime"
	ColumnTypeDateTimeTz = "datetimetz"
	ColumnTypeTime      = "time"
	ColumnTypeTimeTz    = "timetz"
	ColumnTypeTimestamp = "timestamp"
	ColumnTypeTimestampTz = "timestamptz"
	ColumnTypeYear      = "year"
	ColumnTypeJSON      = "json"
	ColumnTypeJSONB     = "jsonb"
	ColumnTypeUUID      = "uuid"
	ColumnTypeBinary    = "binary"
	ColumnTypeEnum      = "enum"
)

// MigrationFile represents a migration file
type MigrationFile struct {
	Name      string
	Path      string
	Timestamp string
	Class     string
}

// NewMigrationFile creates a new migration file instance
func NewMigrationFile(path string) *MigrationFile {
	name := filepath.Base(path)
	timestamp := ExtractTimestamp(name)
	class := ExtractClassName(name)
	
	return &MigrationFile{
		Name:      name,
		Path:      path,
		Timestamp: timestamp,
		Class:     class,
	}
}

// GetTimestamp returns the timestamp from the migration file name
func (mf *MigrationFile) GetTimestamp() string {
	return mf.Timestamp
}

// GetClassName returns the class name from the migration file name
func (mf *MigrationFile) GetClassName() string {
	return mf.Class
}

// ExtractTimestamp extracts timestamp from migration filename
func ExtractTimestamp(filename string) string {
	// Match pattern like "2023_01_01_000000_create_users_table.go"
	re := regexp.MustCompile(`^(\d{4}_\d{2}_\d{2}_\d{6})_`)
	matches := re.FindStringSubmatch(filename)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// ExtractClassName extracts class name from migration filename
func ExtractClassName(filename string) string {
	// Remove extension
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	
	// Remove timestamp prefix
	re := regexp.MustCompile(`^\d{4}_\d{2}_\d{2}_\d{6}_(.+)$`)
	matches := re.FindStringSubmatch(name)
	if len(matches) > 1 {
		return ToPascalCase(matches[1])
	}
	return ToPascalCase(name)
}

// ToPascalCase converts snake_case to PascalCase
func ToPascalCase(s string) string {
	words := strings.Split(s, "_")
	var result strings.Builder
	
	for _, word := range words {
		if len(word) > 0 {
			result.WriteString(strings.ToUpper(word[:1]))
			if len(word) > 1 {
				result.WriteString(strings.ToLower(word[1:]))
			}
		}
	}
	
	return result.String()
}

// ToSnakeCase converts PascalCase/camelCase to snake_case
func ToSnakeCase(s string) string {
	var result strings.Builder
	
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	
	return strings.ToLower(result.String())
}

// GenerateTimestamp generates a timestamp for migration filename
func GenerateTimestamp() string {
	return time.Now().Format("2006_01_02_150405")
}

// GenerateMigrationName generates a migration filename
func GenerateMigrationName(name string) string {
	timestamp := GenerateTimestamp()
	snakeName := ToSnakeCase(name)
	return fmt.Sprintf("%s_%s.go", timestamp, snakeName)
}

// MigrationBatch represents a batch of migrations
type MigrationBatch struct {
	Number     int
	Migrations []string
	CreatedAt  time.Time
}

// NewMigrationBatch creates a new migration batch
func NewMigrationBatch(number int) *MigrationBatch {
	return &MigrationBatch{
		Number:     number,
		Migrations: make([]string, 0),
		CreatedAt:  time.Now(),
	}
}

// Add adds a migration to the batch
func (mb *MigrationBatch) Add(migration string) {
	mb.Migrations = append(mb.Migrations, migration)
}

// Size returns the number of migrations in the batch
func (mb *MigrationBatch) Size() int {
	return len(mb.Migrations)
}

// Contains checks if the batch contains a specific migration
func (mb *MigrationBatch) Contains(migration string) bool {
	for _, m := range mb.Migrations {
		if m == migration {
			return true
		}
	}
	return false
}

// MigrationConfig holds configuration for the migration system
type MigrationConfig struct {
	// Directory containing migration files
	Path string
	
	// Table name for tracking migrations
	Table string
	
	// Connection for database operations
	Connection *sql.DB
	
	// Driver name (mysql, postgres, sqlite3)
	Driver string
	
	// Lock timeout for migration locking
	LockTimeout time.Duration
	
	// Enable/disable transactions
	UseTransactions bool
	
	// Custom SQL generators
	SQLGenerators map[string]SQLGenerator
}

// DefaultMigrationConfig returns default migration configuration
func DefaultMigrationConfig() *MigrationConfig {
	return &MigrationConfig{
		Path:            "database/migrations",
		Table:           "migrations",
		LockTimeout:     15 * time.Minute,
		UseTransactions: true,
		SQLGenerators:   make(map[string]SQLGenerator),
	}
}

// Validate validates the migration configuration
func (mc *MigrationConfig) Validate() error {
	if mc.Connection == nil {
		return fmt.Errorf("database connection is required")
	}
	
	if mc.Driver == "" {
		return fmt.Errorf("database driver is required")
	}
	
	if mc.Path == "" {
		return fmt.Errorf("migration path is required")
	}
	
	if mc.Table == "" {
		return fmt.Errorf("migration table name is required")
	}
	
	return nil
}

// GetSQLGenerator returns the SQL generator for the configured driver
func (mc *MigrationConfig) GetSQLGenerator() SQLGenerator {
	if generator, exists := mc.SQLGenerators[mc.Driver]; exists {
		return generator
	}
	
	// Return default generator based on driver
	switch mc.Driver {
	case "mysql":
		return NewMySQLGenerator()
	case "postgres":
		return NewPostgreSQLGenerator()
	case "sqlite3":
		return NewSQLiteGenerator()
	default:
		return NewMySQLGenerator() // Default fallback
	}
}

// MigrationError represents an error that occurred during migration
type MigrationError struct {
	Migration string
	Operation string
	Err       error
}

func (me *MigrationError) Error() string {
	return fmt.Sprintf("migration error in %s during %s: %v", me.Migration, me.Operation, me.Err)
}

func (me *MigrationError) Unwrap() error {
	return me.Err
}

// NewMigrationError creates a new migration error
func NewMigrationError(migration, operation string, err error) *MigrationError {
	return &MigrationError{
		Migration: migration,
		Operation: operation,
		Err:       err,
	}
}

// SchemaState represents the current state of the database schema
type SchemaState struct {
	Tables      []string
	Columns     map[string][]string
	Indexes     map[string][]string
	ForeignKeys map[string][]string
	Version     string
	UpdatedAt   time.Time
}

// NewSchemaState creates a new schema state
func NewSchemaState() *SchemaState {
	return &SchemaState{
		Tables:      make([]string, 0),
		Columns:     make(map[string][]string),
		Indexes:     make(map[string][]string),
		ForeignKeys: make(map[string][]string),
		UpdatedAt:   time.Now(),
	}
}

// HasTable checks if a table exists in the schema state
func (ss *SchemaState) HasTable(tableName string) bool {
	for _, table := range ss.Tables {
		if table == tableName {
			return true
		}
	}
	return false
}

// HasColumn checks if a column exists in the schema state
func (ss *SchemaState) HasColumn(tableName, columnName string) bool {
	if columns, exists := ss.Columns[tableName]; exists {
		for _, column := range columns {
			if column == columnName {
				return true
			}
		}
	}
	return false
}

// AddTable adds a table to the schema state
func (ss *SchemaState) AddTable(tableName string) {
	if !ss.HasTable(tableName) {
		ss.Tables = append(ss.Tables, tableName)
		ss.Columns[tableName] = make([]string, 0)
		ss.Indexes[tableName] = make([]string, 0)
		ss.ForeignKeys[tableName] = make([]string, 0)
		ss.UpdatedAt = time.Now()
	}
}

// AddColumn adds a column to a table in the schema state
func (ss *SchemaState) AddColumn(tableName, columnName string) {
	if !ss.HasTable(tableName) {
		ss.AddTable(tableName)
	}
	
	if !ss.HasColumn(tableName, columnName) {
		ss.Columns[tableName] = append(ss.Columns[tableName], columnName)
		ss.UpdatedAt = time.Now()
	}
}

// RemoveTable removes a table from the schema state
func (ss *SchemaState) RemoveTable(tableName string) {
	// Remove from tables slice
	for i, table := range ss.Tables {
		if table == tableName {
			ss.Tables = append(ss.Tables[:i], ss.Tables[i+1:]...)
			break
		}
	}
	
	// Remove from maps
	delete(ss.Columns, tableName)
	delete(ss.Indexes, tableName)
	delete(ss.ForeignKeys, tableName)
	ss.UpdatedAt = time.Now()
}

// ParseMigrationNumber parses a migration number from a filename or string
func ParseMigrationNumber(s string) (int, error) {
	re := regexp.MustCompile(`\d+`)
	match := re.FindString(s)
	if match == "" {
		return 0, fmt.Errorf("no migration number found in: %s", s)
	}
	return strconv.Atoi(match)
}

// ValidateMigrationName validates a migration name
func ValidateMigrationName(name string) error {
	if name == "" {
		return fmt.Errorf("migration name cannot be empty")
	}
	
	// Check for valid characters (letters, numbers, underscores)
	re := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	if !re.MatchString(name) {
		return fmt.Errorf("migration name can only contain letters, numbers, and underscores")
	}
	
	return nil
}

// SortMigrationFiles sorts migration files by timestamp
func SortMigrationFiles(files []string) []string {
	// Simple bubble sort by timestamp
	sorted := make([]string, len(files))
	copy(sorted, files)
	
	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			timestamp1 := ExtractTimestamp(sorted[j])
			timestamp2 := ExtractTimestamp(sorted[j+1])
			
			if timestamp1 > timestamp2 {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}
	
	return sorted
}