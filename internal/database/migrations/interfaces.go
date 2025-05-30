package migrations

import (
	"database/sql"
)

// Migration defines the interface for database migrations
type Migration interface {
	// Up runs the migration
	Up() error
	// Down rolls back the migration
	Down() error
	// GetName returns the migration name
	GetName() string
	// GetBatch returns the batch number for this migration
	GetBatch() int
}

// SchemaBuilder provides database schema operations
type SchemaBuilder interface {
	// Table operations
	Create(tableName string, callback func(Table)) error
	Table(tableName string, callback func(Table)) error
	Alter(tableName string, callback func(Table)) error
	Drop(tableName string) error
	DropIfExists(tableName string) error
	Rename(from, to string) error
	
	// Column operations
	HasTable(tableName string) (bool, error)
	HasColumn(tableName, columnName string) (bool, error)
	HasIndex(tableName, indexName string) (bool, error)
	HasForeignKey(tableName, keyName string) (bool, error)
	
	// Information retrieval
	GetColumnListing(tableName string) ([]string, error)
	GetColumnType(tableName, columnName string) (string, error)
	GetTableListing() ([]string, error)
	
	// Raw SQL execution
	Raw(sql string, bindings ...interface{}) error
	
	// Driver information
	GetDriverName() string
	GetConnection() *sql.DB
}

// Table provides fluent interface for table schema definition
type Table interface {
	// Primary key columns
	ID() Column
	BigID() Column
	UUIID() Column
	
	// Column types
	String(name string, length ...int) Column
	Text(name string) Column
	LongText(name string) Column
	MediumText(name string) Column
	TinyText(name string) Column
	Char(name string, length int) Column
	
	// Numeric types
	Integer(name string) Column
	BigInteger(name string) Column
	SmallInteger(name string) Column
	TinyInteger(name string) Column
	UnsignedInteger(name string) Column
	UnsignedBigInteger(name string) Column
	UnsignedSmallInteger(name string) Column
	UnsignedTinyInteger(name string) Column
	
	// Decimal types
	Float(name string, precision ...int) Column
	Double(name string, precision ...int) Column
	Decimal(name string, precision, scale int) Column
	UnsignedFloat(name string, precision ...int) Column
	UnsignedDouble(name string, precision ...int) Column
	UnsignedDecimal(name string, precision, scale int) Column
	
	// Date and time types
	Date(name string) Column
	DateTime(name string) Column
	DateTimeTz(name string) Column
	Time(name string) Column
	TimeTz(name string) Column
	Timestamp(name string) Column
	TimestampTz(name string) Column
	Year(name string) Column
	
	// Boolean and binary types
	Boolean(name string) Column
	Binary(name string) Column
	
	// JSON types
	JSON(name string) Column
	JSONB(name string) Column
	
	// UUID type
	UUID(name string) Column
	
	// Enum type
	Enum(name string, values []string) Column
	
	// Special columns
	Timestamps() Table
	SoftDeletes() Table
	SoftDeletesTz() Table
	RememberToken() Table
	
	// Column modifications
	DropColumn(names ...string) Table
	RenameColumn(from, to string) Table
	ModifyColumn(name string) Column
	
	// Indexes
	Index(columns ...string) Index
	Unique(columns ...string) Index
	Primary(columns ...string) Index
	SpatialIndex(columns ...string) Index
	FullText(columns ...string) Index
	
	// Index operations
	DropIndex(name string) Table
	DropUnique(name string) Table
	DropPrimary() Table
	DropSpatialIndex(name string) Table
	DropFullText(name string) Table
	
	// Foreign keys
	Foreign(column string) ForeignKey
	DropForeign(name string) Table
	
	// Table options
	Engine(engine string) Table
	Charset(charset string) Table
	Collation(collation string) Table
	Comment(comment string) Table
	Temporary() Table
	
	// Raw SQL
	Raw(sql string) Table
	
	// Get table definition for SQL generation
	GetDefinition() *TableDefinition
}

// Column provides fluent interface for column definition
type Column interface {
	// Basic properties
	Length(length int) Column
	Precision(precision int) Column
	Scale(scale int) Column
	
	// Constraints
	Nullable() Column
	NotNull() Column
	Default(value interface{}) Column
	
	// Indexes
	Primary() Column
	Unique() Column
	Index() Column
	SpatialIndex() Column
	FullText() Column
	
	// Auto increment
	AutoIncrement() Column
	
	// Unsigned (for numeric types)
	Unsigned() Column
	
	// String specific
	Charset(charset string) Column
	Collation(collation string) Column
	
	// Comments
	Comment(comment string) Column
	
	// Placement (MySQL specific)
	After(column string) Column
	First() Column
	
	// Virtual/Generated columns
	VirtualAs(expression string) Column
	StoredAs(expression string) Column
	
	// Change/Add operations
	Change() Column
	
	// Foreign key shorthand
	References(column string) ForeignKey
	ConstrainedBy(table string) Column
	CascadeOnUpdate() Column
	CascadeOnDelete() Column
	NullOnDelete() Column
	RestrictOnDelete() Column
	NoActionOnDelete() Column
	
	// Get column definition for SQL generation
	GetDefinition() ColumnDefinition
}

// Index provides fluent interface for index definition
type Index interface {
	// Basic properties
	Name(name string) Index
	Algorithm(algorithm string) Index
	
	// Index types
	Unique() Index
	Spatial() Index
	FullText() Index
	
	// Conditions (PostgreSQL partial indexes)
	Where(condition string) Index
	
	// Get index definition for SQL generation
	GetDefinition() IndexDefinition
}

// ForeignKey provides fluent interface for foreign key definition
type ForeignKey interface {
	// Target definition
	References(column string) ForeignKey
	On(table string) ForeignKey
	
	// Actions
	OnDelete(action string) ForeignKey
	OnUpdate(action string) ForeignKey
	CascadeOnDelete() ForeignKey
	CascadeOnUpdate() ForeignKey
	NullOnDelete() ForeignKey
	RestrictOnDelete() ForeignKey
	RestrictOnUpdate() ForeignKey
	NoActionOnDelete() ForeignKey
	NoActionOnUpdate() ForeignKey
	
	// Naming
	Name(name string) ForeignKey
	
	// Get foreign key definition for SQL generation
	GetDefinition() ForeignKeyDefinition
}

// Supporting types for SQL generation

// ColumnDefinition holds the complete definition of a column
type ColumnDefinition struct {
	Name          string
	Type          string
	Length        *int
	Precision     *int
	Scale         *int
	Nullable      bool
	Default       interface{}
	AutoIncrement bool
	Unsigned      bool
	Charset       string
	Collation     string
	Comment       string
	After         string
	First         bool
	Virtual       string
	Stored        string
	Primary       bool
	Unique        bool
	Index         bool
	SpatialIndex  bool
	FullText      bool
	Change        bool
}

// IndexDefinition holds the complete definition of an index
type IndexDefinition struct {
	Name      string
	Columns   []string
	Type      string // '', 'unique', 'spatial', 'fulltext'
	Algorithm string
	Where     string
}

// ForeignKeyDefinition holds the complete definition of a foreign key
type ForeignKeyDefinition struct {
	Name            string
	Column          string
	ReferencedTable string
	ReferencedColumn string
	OnDelete        string
	OnUpdate        string
}

// TableDefinition holds the complete definition of a table
type TableDefinition struct {
	Name        string
	Columns     []ColumnDefinition
	Indexes     []IndexDefinition
	ForeignKeys []ForeignKeyDefinition
	Engine      string
	Charset     string
	Collation   string
	Comment     string
	Temporary   bool
	RawSQL      []string
}

// OperationType defines the type of schema operation
type OperationType string

const (
	OperationCreate OperationType = "create"
	OperationAlter  OperationType = "alter"
	OperationDrop   OperationType = "drop"
	OperationRename OperationType = "rename"
)

// SchemaOperation represents a schema operation to be executed
type SchemaOperation struct {
	Type       OperationType
	Table      string
	NewTable   string      // For rename operations
	Definition *TableDefinition
	SQL        string      // For raw SQL operations
	Bindings   []interface{}
}

// MigrationStatus represents the status of a migration
type MigrationStatus struct {
	Migration string
	Batch     int
	Executed  bool
	ExecutedAt *sql.NullTime
}

// RanMigration represents a migration that has been executed
type RanMigration struct {
	Migration string
	Batch     int
}

// Migrator provides migration management functionality
type Migrator interface {
	// Migration execution
	Run() error
	Rollback(steps int) error
	Reset() error
	Refresh() error
	Fresh() error
	Step(steps int) error
	
	// Migration status
	Status() ([]MigrationStatus, error)
	Pending() ([]string, error)
	Ran() ([]RanMigration, error)
	
	// Migration management
	Install() error
	Uninstall() error
	Installed() (bool, error)
	
	// File operations
	GetMigrationFiles() ([]string, error)
	GetMigrationName(filename string) string
	GetTimestamp(filename string) string
	
	// Configuration
	SetPath(path string)
	GetPath() string
	SetTable(table string)
	GetTable() string
	
	// Dependencies
	SetSchemaBuilder(builder SchemaBuilder)
	GetSchemaBuilder() SchemaBuilder
}

// MigrationRepository provides data access for migration tracking
type MigrationRepository interface {
	// Repository operations
	GetRan() ([]RanMigration, error)
	GetMigrations(steps int) ([]RanMigration, error)
	GetLast() ([]RanMigration, error)
	GetMigrationBatches() (map[string]int, error)
	
	// Tracking operations
	Log(migration string, batch int) error
	Delete(migration string) error
	GetNextBatchNumber() (int, error)
	
	// Repository management
	CreateRepository() error
	RepositoryExists() (bool, error)
	DeleteRepository() error
}

// SQLGenerator provides database-specific SQL generation
type SQLGenerator interface {
	// Table operations
	GenerateCreateTable(definition *TableDefinition) string
	GenerateAlterTable(tableName string, definition *TableDefinition) []string
	GenerateDropTable(tableName string, ifExists bool) string
	GenerateRenameTable(oldName, newName string) string
	
	// Column operations
	GenerateAddColumn(tableName string, column ColumnDefinition) string
	GenerateModifyColumn(tableName string, column ColumnDefinition) string
	GenerateDropColumn(tableName string, columnName string) string
	GenerateRenameColumn(tableName, oldName, newName string) string
	
	// Index operations
	GenerateCreateIndex(tableName string, index IndexDefinition) string
	GenerateDropIndex(tableName string, indexName string) string
	
	// Foreign key operations
	GenerateAddForeignKey(tableName string, foreignKey ForeignKeyDefinition) string
	GenerateDropForeignKey(tableName string, keyName string) string
	
	// Introspection queries
	GetTableExistsQuery(tableName string) string
	GetColumnExistsQuery(tableName, columnName string) string
	GetIndexExistsQuery(tableName, indexName string) string
	GetForeignKeyExistsQuery(tableName, keyName string) string
	GetColumnListingQuery(tableName string) string
	GetTableListingQuery() string
	
	// Type mapping
	MapColumnType(columnType string) string
	SupportsFeature(feature string) bool
}

// Common SQL generation features
const (
	FeatureTransactions        = "transactions"
	FeatureForeignKeys        = "foreign_keys"
	FeaturePartialIndexes     = "partial_indexes"
	FeatureJsonColumns        = "json_columns"
	FeatureGeneratedColumns   = "generated_columns"
	FeatureRenameColumns      = "rename_columns"
	FeatureRenameIndexes      = "rename_indexes"
	FeatureDropIndexes        = "drop_indexes"
	FeatureMultipleIndexes    = "multiple_indexes"
	FeatureIndexAlgorithms    = "index_algorithms"
	FeatureCheckConstraints   = "check_constraints"
	FeatureCommentOnColumns   = "comment_on_columns"
	FeatureCommentOnTables    = "comment_on_tables"
)