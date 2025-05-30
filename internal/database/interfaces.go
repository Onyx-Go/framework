package database

import (
	"database/sql"
	"time"
)

// Model interface defines the contract for database models
type Model interface {
	TableName() string
}

// EventableModel interface for models that support events
type EventableModel interface {
	Model
	GetID() interface{}
	SetID(id interface{})
	GetCreatedAt() *time.Time
	GetUpdatedAt() *time.Time
	GetDeletedAt() *sql.NullTime
	IsEventable() bool
	IsNew() bool
}

// Database represents the database connection and operations
type Database interface {
	// Connection management
	Close() error
	Ping() error
	
	// Query building
	Table(tableName string) QueryBuilder
	Model(model Model) QueryBuilder
	
	// Raw SQL
	Raw(query string, args ...interface{}) QueryBuilder
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

// QueryBuilder interface defines the contract for query building
type QueryBuilder interface {
	// Selection
	Select(columns ...string) QueryBuilder
	Table(tableName string) QueryBuilder
	
	// Conditions
	Where(column string, operator string, value interface{}) QueryBuilder
	WhereIn(column string, values []interface{}) QueryBuilder
	WhereNotIn(column string, values []interface{}) QueryBuilder
	WhereNull(column string) QueryBuilder
	WhereNotNull(column string) QueryBuilder
	WhereBetween(column string, start, end interface{}) QueryBuilder
	WhereNotBetween(column string, start, end interface{}) QueryBuilder
	WhereLike(column string, value string) QueryBuilder
	WhereNotLike(column string, value string) QueryBuilder
	
	// Logical operators
	OrWhere(column string, operator string, value interface{}) QueryBuilder
	OrWhereIn(column string, values []interface{}) QueryBuilder
	OrWhereNotIn(column string, values []interface{}) QueryBuilder
	OrWhereNull(column string) QueryBuilder
	OrWhereNotNull(column string) QueryBuilder
	
	// Joins
	Join(table string, first string, operator string, second string) QueryBuilder
	LeftJoin(table string, first string, operator string, second string) QueryBuilder
	RightJoin(table string, first string, operator string, second string) QueryBuilder
	InnerJoin(table string, first string, operator string, second string) QueryBuilder
	
	// Ordering and grouping
	OrderBy(column string, direction ...string) QueryBuilder
	GroupBy(columns ...string) QueryBuilder
	Having(column string, operator string, value interface{}) QueryBuilder
	
	// Limiting
	Limit(limit int) QueryBuilder
	Offset(offset int) QueryBuilder
	
	// Eager loading
	With(relations ...string) QueryBuilder
	WithCount(relations ...string) QueryBuilder
	
	// Soft deletes
	WithTrashed() QueryBuilder
	OnlyTrashed() QueryBuilder
	
	// Execution
	Get(dest interface{}) error
	First(dest interface{}) error
	Find(dest interface{}, id interface{}) error
	Exists() (bool, error)
	Count() (int64, error)
	Sum(column string) (float64, error)
	Avg(column string) (float64, error)
	Min(column string) (interface{}, error)
	Max(column string) (interface{}, error)
	
	// Mutation
	Insert(data interface{}) (sql.Result, error)
	Update(data interface{}) (sql.Result, error)
	Delete() (sql.Result, error)
	ForceDelete() (sql.Result, error)
	Restore() (sql.Result, error)
	
	// SQL building
	ToSQL() (string, []interface{}, error)
}

// Scanner interface for database scanning utilities
type Scanner interface {
	ScanRows(rows *sql.Rows, dest interface{}) error
	ScanRow(row *sql.Row, dest interface{}) error
	ScanIntoStruct(scanner interface{}, dest interface{}) error
}

// Migrator interface for database migrations
type Migrator interface {
	// Migration execution
	Up() error
	Down() error
	Rollback(steps int) error
	Reset() error
	Refresh() error
	Fresh() error
	
	// Migration status
	Status() ([]MigrationStatus, error)
	Pending() ([]string, error)
	Ran() ([]RanMigration, error)
	
	// Migration management
	Install() error
	Uninstall() error
	
	// File operations
	MakeMigration(name string) (string, error)
	GetMigrationFiles() ([]string, error)
}

// Migration represents a single database migration
type Migration interface {
	Up() error
	Down() error
	GetName() string
	GetBatch() int
}

// SchemaBuilder interface for database schema operations
type SchemaBuilder interface {
	// Table operations
	Create(tableName string, callback func(Table)) error
	Table(tableName string, callback func(Table)) error
	Alter(tableName string, callback func(Table)) error
	Drop(tableName string) error
	DropIfExists(tableName string) error
	Rename(from, to string) error
	
	// Table inspection
	HasTable(tableName string) (bool, error)
	HasColumn(tableName, columnName string) (bool, error)
	HasIndex(tableName, indexName string) (bool, error)
	HasForeignKey(tableName, keyName string) (bool, error)
	
	// Column operations
	GetColumnListing(tableName string) ([]string, error)
	GetColumnType(tableName, columnName string) (string, error)
}

// Table interface for table schema building
type Table interface {
	// Primary key
	ID() Table
	BigID() Table
	UUIID() Table
	
	// Basic column types
	String(name string, length ...int) Column
	Text(name string) Column
	LongText(name string) Column
	Integer(name string) Column
	BigInteger(name string) Column
	Float(name string) Column
	Double(name string) Column
	Decimal(name string, precision, scale int) Column
	Boolean(name string) Column
	Date(name string) Column
	DateTime(name string) Column
	Time(name string) Column
	Timestamp(name string) Column
	JSON(name string) Column
	
	// Special columns
	Timestamps() Table
	SoftDeletes() Table
	
	// Column modification
	DropColumn(names ...string) Table
	RenameColumn(from, to string) Table
	
	// Indexes
	Index(columns ...string) Index
	Unique(columns ...string) Index
	Primary(columns ...string) Index
	DropIndex(name string) Table
	DropPrimary() Table
	DropUnique(name string) Table
	
	// Foreign keys
	Foreign(column string) ForeignKey
	DropForeign(name string) Table
}

// Column interface for column schema building
type Column interface {
	// Constraints
	Nullable() Column
	NotNull() Column
	Default(value interface{}) Column
	
	// Indexes
	Primary() Column
	Unique() Column
	Index() Column
	
	// Auto increment
	AutoIncrement() Column
	
	// Comments
	Comment(comment string) Column
	
	// Placement
	After(column string) Column
	First() Column
}

// Index interface for index schema building
type Index interface {
	Name(name string) Index
	Unique() Index
	Partial(condition string) Index
}

// ForeignKey interface for foreign key schema building
type ForeignKey interface {
	References(column string) ForeignKey
	On(table string) ForeignKey
	OnDelete(action string) ForeignKey
	OnUpdate(action string) ForeignKey
	Name(name string) ForeignKey
}

// MigrationStatus represents the status of a migration
type MigrationStatus struct {
	Name   string
	Batch  int
	Ran    bool
	RanAt  *time.Time
}

// RanMigration represents a migration that has been executed
type RanMigration struct {
	Migration string
	Batch     int
	RanAt     time.Time
}

// whereClause represents a WHERE condition in a SQL query
type whereClause struct {
	Column   string
	Operator string
	Value    interface{}
	Boolean  string
}