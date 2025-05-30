package onyx

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Migration interface defines the contract for database migrations
type Migration interface {
	Up() error
	Down() error
	GetName() string
	GetBatch() int
	SetBatch(batch int)
	GetTimestamp() string
}

// SchemaBuilder interface for database schema operations
type SchemaBuilder interface {
	Create(tableName string, callback func(table Table)) error
	Table(tableName string, callback func(table Table)) error
	Drop(tableName string) error
	DropIfExists(tableName string) error
	Rename(from, to string) error
	HasTable(tableName string) (bool, error)
	HasColumn(tableName, columnName string) (bool, error)
	GetColumnListing(tableName string) ([]string, error)
	GetConnection() *sql.DB
	SetConnection(db *sql.DB)
}

// Table interface for table definition
type Table interface {
	// Primary key columns
	ID(name ...string) Column
	Increments(name string) Column
	BigIncrements(name string) Column
	
	// String columns
	String(name string, length ...int) Column
	Char(name string, length int) Column
	Text(name string) Column
	MediumText(name string) Column
	LongText(name string) Column
	TinyText(name string) Column
	
	// Numeric columns
	Integer(name string) Column
	BigInteger(name string) Column
	SmallInteger(name string) Column
	TinyInteger(name string) Column
	UnsignedInteger(name string) Column
	UnsignedBigInteger(name string) Column
	
	// Decimal columns
	Float(name string, precision, scale int) Column
	Double(name string, precision, scale int) Column
	Decimal(name string, precision, scale int) Column
	
	// Date/Time columns
	Date(name string) Column
	DateTime(name string) Column
	Timestamp(name string) Column
	Time(name string) Column
	Year(name string) Column
	Timestamps()
	
	// Special columns
	Boolean(name string) Column
	Binary(name string) Column
	JSON(name string) Column
	UUID(name string) Column
	ULID(name string) Column
	Enum(name string, values []string) Column
	Set(name string, values []string) Column
	
	// Foreign key columns
	ForeignID(name string) Column
	ForeignIDFor(model string) Column
	
	// Indexes
	Index(columns []string, name ...string) Index
	Unique(columns []string, name ...string) Index
	Primary(columns []string, name ...string) Index
	Foreign(column string) ForeignKey
	
	// Drop operations
	DropColumn(columns ...string)
	DropIndex(name string)
	DropUnique(name string)
	DropPrimary(name ...string)
	DropForeign(name string)
	
	// Modify operations
	RenameColumn(from, to string)
	ChangeColumn(name string) Column
	
	// Special helpers
	SoftDeletes()
	RememberToken()
	Morphs(name string)
	
	// Get SQL
	ToSQL(driver string) []string
}

// Column interface for column definition
type Column interface {
	// Nullability
	Nullable() Column
	NotNull() Column
	
	// Defaults
	Default(value interface{}) Column
	UseCurrent() Column
	UseCurrentOnUpdate() Column
	
	// Constraints
	Unique() Column
	Index() Column
	Primary() Column
	AutoIncrement() Column
	Unsigned() Column
	
	// Metadata
	Comment(comment string) Column
	Charset(charset string) Column
	Collation(collation string) Column
	
	// Positioning (MySQL specific)
	After(column string) Column
	First() Column
	
	// Change operations
	Change() Column
	
	// Get properties
	GetName() string
	GetType() string
	ToSQL(driver string) string
}

// Index interface for index definition
type Index interface {
	Algorithm(algorithm string) Index
	Comment(comment string) Index
	Length(lengths map[string]int) Index
	Where(condition string) Index
	ToSQL(tableName, driver string) string
}

// ForeignKey interface for foreign key constraints
type ForeignKey interface {
	References(column string) ForeignKey
	On(table string) ForeignKey
	OnDelete(action string) ForeignKey
	OnUpdate(action string) ForeignKey
	Constrained(table ...string) ForeignKey
	CascadeOnDelete() ForeignKey
	CascadeOnUpdate() ForeignKey
	ToSQL(tableName, driver string) string
}

// BaseMigration provides base functionality for migrations
type BaseMigration struct {
	name      string
	timestamp string
	batch     int
}

func NewBaseMigration(name string) *BaseMigration {
	return &BaseMigration{
		name:      name,
		timestamp: extractTimestamp(name),
	}
}

func (bm *BaseMigration) GetName() string {
	return bm.name
}

func (bm *BaseMigration) GetTimestamp() string {
	return bm.timestamp
}

func (bm *BaseMigration) GetBatch() int {
	return bm.batch
}

func (bm *BaseMigration) SetBatch(batch int) {
	bm.batch = batch
}

func (bm *BaseMigration) Up() error {
	return fmt.Errorf("up method must be implemented")
}

func (bm *BaseMigration) Down() error {
	return fmt.Errorf("down method must be implemented")
}

// DefaultSchemaBuilder implements SchemaBuilder interface
type DefaultSchemaBuilder struct {
	db     *sql.DB
	driver string
}

func NewSchemaBuilder(db *sql.DB, driver string) *DefaultSchemaBuilder {
	return &DefaultSchemaBuilder{
		db:     db,
		driver: driver,
	}
}

func (dsb *DefaultSchemaBuilder) GetConnection() *sql.DB {
	return dsb.db
}

func (dsb *DefaultSchemaBuilder) SetConnection(db *sql.DB) {
	dsb.db = db
}

func (dsb *DefaultSchemaBuilder) Create(tableName string, callback func(table Table)) error {
	table := NewTableBuilder(tableName, "create")
	callback(&table)
	
	sqlStatements := table.ToSQL(dsb.driver)
	for _, sql := range sqlStatements {
		if _, err := dsb.db.Exec(sql); err != nil {
			return fmt.Errorf("failed to execute SQL: %s, error: %w", sql, err)
		}
	}
	
	return nil
}

func (dsb *DefaultSchemaBuilder) Table(tableName string, callback func(table Table)) error {
	table := NewTableBuilder(tableName, "alter")
	callback(&table)
	
	sqlStatements := table.ToSQL(dsb.driver)
	for _, sql := range sqlStatements {
		if _, err := dsb.db.Exec(sql); err != nil {
			return fmt.Errorf("failed to execute SQL: %s, error: %w", sql, err)
		}
	}
	
	return nil
}

func (dsb *DefaultSchemaBuilder) Drop(tableName string) error {
	sql := fmt.Sprintf("DROP TABLE %s", tableName)
	_, err := dsb.db.Exec(sql)
	return err
}

func (dsb *DefaultSchemaBuilder) DropIfExists(tableName string) error {
	var sql string
	switch dsb.driver {
	case "mysql":
		sql = fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName)
	case "postgres":
		sql = fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", tableName)
	case "sqlite3":
		sql = fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName)
	default:
		sql = fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName)
	}
	
	_, err := dsb.db.Exec(sql)
	return err
}

func (dsb *DefaultSchemaBuilder) Rename(from, to string) error {
	var sql string
	switch dsb.driver {
	case "mysql":
		sql = fmt.Sprintf("RENAME TABLE %s TO %s", from, to)
	case "postgres":
		sql = fmt.Sprintf("ALTER TABLE %s RENAME TO %s", from, to)
	case "sqlite3":
		sql = fmt.Sprintf("ALTER TABLE %s RENAME TO %s", from, to)
	default:
		sql = fmt.Sprintf("ALTER TABLE %s RENAME TO %s", from, to)
	}
	
	_, err := dsb.db.Exec(sql)
	return err
}

func (dsb *DefaultSchemaBuilder) HasTable(tableName string) (bool, error) {
	var query string
	var args []interface{}
	
	switch dsb.driver {
	case "mysql":
		query = "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?"
		args = []interface{}{tableName}
	case "postgres":
		query = "SELECT COUNT(*) FROM information_schema.tables WHERE table_name = $1"
		args = []interface{}{tableName}
	case "sqlite3":
		query = "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name = ?"
		args = []interface{}{tableName}
	default:
		return false, fmt.Errorf("unsupported driver: %s", dsb.driver)
	}
	
	var count int
	err := dsb.db.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return false, err
	}
	
	return count > 0, nil
}

func (dsb *DefaultSchemaBuilder) HasColumn(tableName, columnName string) (bool, error) {
	var query string
	var args []interface{}
	
	switch dsb.driver {
	case "mysql":
		query = "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?"
		args = []interface{}{tableName, columnName}
	case "postgres":
		query = "SELECT COUNT(*) FROM information_schema.columns WHERE table_name = $1 AND column_name = $2"
		args = []interface{}{tableName, columnName}
	case "sqlite3":
		query = "SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?"
		args = []interface{}{tableName, columnName}
	default:
		return false, fmt.Errorf("unsupported driver: %s", dsb.driver)
	}
	
	var count int
	err := dsb.db.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return false, err
	}
	
	return count > 0, nil
}

func (dsb *DefaultSchemaBuilder) GetColumnListing(tableName string) ([]string, error) {
	var query string
	var args []interface{}
	
	switch dsb.driver {
	case "mysql":
		query = "SELECT column_name FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = ? ORDER BY ordinal_position"
		args = []interface{}{tableName}
	case "postgres":
		query = "SELECT column_name FROM information_schema.columns WHERE table_name = $1 ORDER BY ordinal_position"
		args = []interface{}{tableName}
	case "sqlite3":
		query = "SELECT name FROM pragma_table_info(?)"
		args = []interface{}{tableName}
	default:
		return nil, fmt.Errorf("unsupported driver: %s", dsb.driver)
	}
	
	rows, err := dsb.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var columns []string
	for rows.Next() {
		var column string
		if err := rows.Scan(&column); err != nil {
			return nil, err
		}
		columns = append(columns, column)
	}
	
	return columns, rows.Err()
}

// TableBuilder implements Table interface
type TableBuilder struct {
	name         string
	action       string
	columns      []*ColumnBuilder
	indexes      []*IndexBuilder
	foreignKeys  []*ForeignKeyBuilder
	commands     []string
}

func NewTableBuilder(name, action string) TableBuilder {
	return TableBuilder{
		name:        name,
		action:      action,
		columns:     make([]*ColumnBuilder, 0),
		indexes:     make([]*IndexBuilder, 0),
		foreignKeys: make([]*ForeignKeyBuilder, 0),
		commands:    make([]string, 0),
	}
}

// Primary key methods
func (t *TableBuilder) ID(name ...string) Column {
	columnName := "id"
	if len(name) > 0 {
		columnName = name[0]
	}
	
	column := &ColumnBuilder{
		name:          columnName,
		dataType:      "BIGINT",
		autoIncrement: true,
		primary:       true,
		unsigned:      true,
		nullable:      false,
	}
	
	t.columns = append(t.columns, column)
	return column
}

func (t *TableBuilder) Increments(name string) Column {
	column := &ColumnBuilder{
		name:          name,
		dataType:      "INT",
		autoIncrement: true,
		primary:       true,
		unsigned:      true,
		nullable:      false,
	}
	
	t.columns = append(t.columns, column)
	return column
}

func (t *TableBuilder) BigIncrements(name string) Column {
	column := &ColumnBuilder{
		name:          name,
		dataType:      "BIGINT",
		autoIncrement: true,
		primary:       true,
		unsigned:      true,
		nullable:      false,
	}
	
	t.columns = append(t.columns, column)
	return column
}

// String methods
func (t *TableBuilder) String(name string, length ...int) Column {
	columnLength := 255
	if len(length) > 0 {
		columnLength = length[0]
	}
	
	column := &ColumnBuilder{
		name:     name,
		dataType: "VARCHAR",
		length:   columnLength,
		nullable: true,
	}
	
	t.columns = append(t.columns, column)
	return column
}

func (t *TableBuilder) Char(name string, length int) Column {
	column := &ColumnBuilder{
		name:     name,
		dataType: "CHAR",
		length:   length,
		nullable: true,
	}
	
	t.columns = append(t.columns, column)
	return column
}

func (t *TableBuilder) Text(name string) Column {
	column := &ColumnBuilder{
		name:     name,
		dataType: "TEXT",
		nullable: true,
	}
	
	t.columns = append(t.columns, column)
	return column
}

func (t *TableBuilder) MediumText(name string) Column {
	column := &ColumnBuilder{
		name:     name,
		dataType: "MEDIUMTEXT",
		nullable: true,
	}
	
	t.columns = append(t.columns, column)
	return column
}

func (t *TableBuilder) LongText(name string) Column {
	column := &ColumnBuilder{
		name:     name,
		dataType: "LONGTEXT",
		nullable: true,
	}
	
	t.columns = append(t.columns, column)
	return column
}

func (t *TableBuilder) TinyText(name string) Column {
	column := &ColumnBuilder{
		name:     name,
		dataType: "TINYTEXT",
		nullable: true,
	}
	
	t.columns = append(t.columns, column)
	return column
}

// Numeric methods
func (t *TableBuilder) Integer(name string) Column {
	column := &ColumnBuilder{
		name:     name,
		dataType: "INT",
		nullable: true,
	}
	
	t.columns = append(t.columns, column)
	return column
}

func (t *TableBuilder) BigInteger(name string) Column {
	column := &ColumnBuilder{
		name:     name,
		dataType: "BIGINT",
		nullable: true,
	}
	
	t.columns = append(t.columns, column)
	return column
}

func (t *TableBuilder) SmallInteger(name string) Column {
	column := &ColumnBuilder{
		name:     name,
		dataType: "SMALLINT",
		nullable: true,
	}
	
	t.columns = append(t.columns, column)
	return column
}

func (t *TableBuilder) TinyInteger(name string) Column {
	column := &ColumnBuilder{
		name:     name,
		dataType: "TINYINT",
		nullable: true,
	}
	
	t.columns = append(t.columns, column)
	return column
}

func (t *TableBuilder) UnsignedInteger(name string) Column {
	column := &ColumnBuilder{
		name:     name,
		dataType: "INT",
		unsigned: true,
		nullable: true,
	}
	
	t.columns = append(t.columns, column)
	return column
}

func (t *TableBuilder) UnsignedBigInteger(name string) Column {
	column := &ColumnBuilder{
		name:     name,
		dataType: "BIGINT",
		unsigned: true,
		nullable: true,
	}
	
	t.columns = append(t.columns, column)
	return column
}

// Decimal methods
func (t *TableBuilder) Float(name string, precision, scale int) Column {
	column := &ColumnBuilder{
		name:      name,
		dataType:  "FLOAT",
		precision: precision,
		scale:     scale,
		nullable:  true,
	}
	
	t.columns = append(t.columns, column)
	return column
}

func (t *TableBuilder) Double(name string, precision, scale int) Column {
	column := &ColumnBuilder{
		name:      name,
		dataType:  "DOUBLE",
		precision: precision,
		scale:     scale,
		nullable:  true,
	}
	
	t.columns = append(t.columns, column)
	return column
}

func (t *TableBuilder) Decimal(name string, precision, scale int) Column {
	column := &ColumnBuilder{
		name:      name,
		dataType:  "DECIMAL",
		precision: precision,
		scale:     scale,
		nullable:  true,
	}
	
	t.columns = append(t.columns, column)
	return column
}

// Date/Time methods
func (t *TableBuilder) Date(name string) Column {
	column := &ColumnBuilder{
		name:     name,
		dataType: "DATE",
		nullable: true,
	}
	
	t.columns = append(t.columns, column)
	return column
}

func (t *TableBuilder) DateTime(name string) Column {
	column := &ColumnBuilder{
		name:     name,
		dataType: "DATETIME",
		nullable: true,
	}
	
	t.columns = append(t.columns, column)
	return column
}

func (t *TableBuilder) Timestamp(name string) Column {
	column := &ColumnBuilder{
		name:     name,
		dataType: "TIMESTAMP",
		nullable: true,
	}
	
	t.columns = append(t.columns, column)
	return column
}

func (t *TableBuilder) Time(name string) Column {
	column := &ColumnBuilder{
		name:     name,
		dataType: "TIME",
		nullable: true,
	}
	
	t.columns = append(t.columns, column)
	return column
}

func (t *TableBuilder) Year(name string) Column {
	column := &ColumnBuilder{
		name:     name,
		dataType: "YEAR",
		nullable: true,
	}
	
	t.columns = append(t.columns, column)
	return column
}

func (t *TableBuilder) Timestamps() {
	createdAt := &ColumnBuilder{
		name:         "created_at",
		dataType:     "TIMESTAMP",
		nullable:     true,
		useCurrent:   true,
	}
	
	updatedAt := &ColumnBuilder{
		name:                "updated_at",
		dataType:            "TIMESTAMP",
		nullable:            true,
		useCurrent:          true,
		useCurrentOnUpdate:  true,
	}
	
	t.columns = append(t.columns, createdAt, updatedAt)
}

// Special columns
func (t *TableBuilder) Boolean(name string) Column {
	column := &ColumnBuilder{
		name:     name,
		dataType: "BOOLEAN",
		nullable: true,
	}
	
	t.columns = append(t.columns, column)
	return column
}

func (t *TableBuilder) Binary(name string) Column {
	column := &ColumnBuilder{
		name:     name,
		dataType: "BLOB",
		nullable: true,
	}
	
	t.columns = append(t.columns, column)
	return column
}

func (t *TableBuilder) JSON(name string) Column {
	column := &ColumnBuilder{
		name:     name,
		dataType: "JSON",
		nullable: true,
	}
	
	t.columns = append(t.columns, column)
	return column
}

func (t *TableBuilder) UUID(name string) Column {
	column := &ColumnBuilder{
		name:     name,
		dataType: "CHAR",
		length:   36,
		nullable: true,
	}
	
	t.columns = append(t.columns, column)
	return column
}

func (t *TableBuilder) ULID(name string) Column {
	column := &ColumnBuilder{
		name:     name,
		dataType: "CHAR",
		length:   26,
		nullable: true,
	}
	
	t.columns = append(t.columns, column)
	return column
}

func (t *TableBuilder) Enum(name string, values []string) Column {
	column := &ColumnBuilder{
		name:       name,
		dataType:   "ENUM",
		enumValues: values,
		nullable:   true,
	}
	
	t.columns = append(t.columns, column)
	return column
}

func (t *TableBuilder) Set(name string, values []string) Column {
	column := &ColumnBuilder{
		name:       name,
		dataType:   "SET",
		enumValues: values,
		nullable:   true,
	}
	
	t.columns = append(t.columns, column)
	return column
}

// Foreign key columns
func (t *TableBuilder) ForeignID(name string) Column {
	column := &ColumnBuilder{
		name:     name,
		dataType: "BIGINT",
		unsigned: true,
		nullable: true,
	}
	
	t.columns = append(t.columns, column)
	return column
}

func (t *TableBuilder) ForeignIDFor(model string) Column {
	name := strings.ToLower(model) + "_id"
	return t.ForeignID(name)
}

// Index methods
func (t *TableBuilder) Index(columns []string, name ...string) Index {
	indexName := generateIndexName(t.name, "idx", columns)
	if len(name) > 0 {
		indexName = name[0]
	}
	
	index := &IndexBuilder{
		name:    indexName,
		columns: columns,
		type_:   "INDEX",
	}
	
	t.indexes = append(t.indexes, index)
	return index
}

func (t *TableBuilder) Unique(columns []string, name ...string) Index {
	indexName := generateIndexName(t.name, "uniq", columns)
	if len(name) > 0 {
		indexName = name[0]
	}
	
	index := &IndexBuilder{
		name:    indexName,
		columns: columns,
		type_:   "UNIQUE",
	}
	
	t.indexes = append(t.indexes, index)
	return index
}

func (t *TableBuilder) Primary(columns []string, name ...string) Index {
	indexName := "PRIMARY"
	if len(name) > 0 {
		indexName = name[0]
	}
	
	index := &IndexBuilder{
		name:    indexName,
		columns: columns,
		type_:   "PRIMARY KEY",
	}
	
	t.indexes = append(t.indexes, index)
	return index
}

func (t *TableBuilder) Foreign(column string) ForeignKey {
	foreignKey := &ForeignKeyBuilder{
		column: column,
		name:   generateForeignKeyName(t.name, column),
	}
	
	t.foreignKeys = append(t.foreignKeys, foreignKey)
	return foreignKey
}

// Drop operations
func (t *TableBuilder) DropColumn(columns ...string) {
	for _, column := range columns {
		t.commands = append(t.commands, fmt.Sprintf("DROP COLUMN %s", column))
	}
}

func (t *TableBuilder) DropIndex(name string) {
	t.commands = append(t.commands, fmt.Sprintf("DROP INDEX %s", name))
}

func (t *TableBuilder) DropUnique(name string) {
	t.commands = append(t.commands, fmt.Sprintf("DROP INDEX %s", name))
}

func (t *TableBuilder) DropPrimary(name ...string) {
	t.commands = append(t.commands, "DROP PRIMARY KEY")
}

func (t *TableBuilder) DropForeign(name string) {
	t.commands = append(t.commands, fmt.Sprintf("DROP FOREIGN KEY %s", name))
}

// Modify operations
func (t *TableBuilder) RenameColumn(from, to string) {
	t.commands = append(t.commands, fmt.Sprintf("RENAME COLUMN %s TO %s", from, to))
}

func (t *TableBuilder) ChangeColumn(name string) Column {
	column := &ColumnBuilder{
		name:   name,
		change: true,
	}
	
	t.columns = append(t.columns, column)
	return column
}

// Special helpers
func (t *TableBuilder) SoftDeletes() {
	deletedAt := &ColumnBuilder{
		name:     "deleted_at",
		dataType: "TIMESTAMP",
		nullable: true,
	}
	
	t.columns = append(t.columns, deletedAt)
}

func (t *TableBuilder) RememberToken() {
	rememberToken := &ColumnBuilder{
		name:     "remember_token",
		dataType: "VARCHAR",
		length:   100,
		nullable: true,
	}
	
	t.columns = append(t.columns, rememberToken)
}

func (t *TableBuilder) Morphs(name string) {
	morphType := &ColumnBuilder{
		name:     name + "_type",
		dataType: "VARCHAR",
		length:   255,
		nullable: false,
	}
	
	morphID := &ColumnBuilder{
		name:     name + "_id",
		dataType: "BIGINT",
		unsigned: true,
		nullable: false,
	}
	
	t.columns = append(t.columns, morphType, morphID)
}

// ToSQL generates SQL statements for the table
func (t *TableBuilder) ToSQL(driver string) []string {
	if t.action == "create" {
		return t.buildCreateTableSQL(driver)
	} else {
		return t.buildAlterTableSQL(driver)
	}
}

func (t *TableBuilder) buildCreateTableSQL(driver string) []string {
	var statements []string
	var parts []string
	
	// Add columns
	for _, column := range t.columns {
		parts = append(parts, "  "+column.ToSQL(driver))
	}
	
	// Add primary key if not already added
	for _, index := range t.indexes {
		if index.type_ == "PRIMARY KEY" {
			parts = append(parts, fmt.Sprintf("  PRIMARY KEY (%s)", strings.Join(index.columns, ", ")))
		}
	}
	
	// Add foreign keys inline for SQLite (during table creation)
	if driver == "sqlite3" {
		for _, fk := range t.foreignKeys {
			if fk.referencedTable != "" && fk.referencedColumn != "" {
				fkSQL := fmt.Sprintf("  FOREIGN KEY (%s) REFERENCES %s (%s)", fk.column, fk.referencedTable, fk.referencedColumn)
				if fk.onDelete != "" {
					fkSQL += fmt.Sprintf(" ON DELETE %s", fk.onDelete)
				}
				if fk.onUpdate != "" {
					fkSQL += fmt.Sprintf(" ON UPDATE %s", fk.onUpdate)
				}
				parts = append(parts, fkSQL)
			}
		}
	}
	
	// Create table statement
	sql := fmt.Sprintf("CREATE TABLE %s (\n%s\n)", t.name, strings.Join(parts, ",\n"))
	statements = append(statements, sql)
	
	// Add indexes
	for _, index := range t.indexes {
		if index.type_ != "PRIMARY KEY" {
			statements = append(statements, index.ToSQL(t.name, driver))
		}
	}
	
	// Add foreign keys for non-SQLite databases
	if driver != "sqlite3" {
		for _, fk := range t.foreignKeys {
			statements = append(statements, fk.ToSQL(t.name, driver))
		}
	}
	
	return statements
}

func (t *TableBuilder) buildAlterTableSQL(driver string) []string {
	var statements []string
	
	// Handle SQLite's limitation: it doesn't support multiple ADD COLUMN in one statement
	if driver == "sqlite3" {
		// Add new columns one by one for SQLite
		for _, column := range t.columns {
			if column.change {
				// SQLite doesn't support MODIFY COLUMN, would need recreate table
				statements = append(statements, fmt.Sprintf("-- WARNING: SQLite doesn't support MODIFY COLUMN for %s", column.name))
			} else {
				sql := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", t.name, column.ToSQL(driver))
				statements = append(statements, sql)
			}
		}
		
		// Add commands one by one for SQLite
		for _, command := range t.commands {
			sql := fmt.Sprintf("ALTER TABLE %s %s", t.name, command)
			statements = append(statements, sql)
		}
	} else {
		// For other databases, group alterations
		var alterations []string
		
		// Add new columns
		for _, column := range t.columns {
			if column.change {
				alterations = append(alterations, "MODIFY COLUMN "+column.ToSQL(driver))
			} else {
				alterations = append(alterations, "ADD COLUMN "+column.ToSQL(driver))
			}
		}
		
		// Add commands
		for _, command := range t.commands {
			alterations = append(alterations, command)
		}
		
		if len(alterations) > 0 {
			sql := fmt.Sprintf("ALTER TABLE %s %s", t.name, strings.Join(alterations, ", "))
			statements = append(statements, sql)
		}
	}
	
	// Add indexes
	for _, index := range t.indexes {
		statements = append(statements, index.ToSQL(t.name, driver))
	}
	
	// Add foreign keys
	for _, fk := range t.foreignKeys {
		statements = append(statements, fk.ToSQL(t.name, driver))
	}
	
	return statements
}

// ColumnBuilder implements Column interface
type ColumnBuilder struct {
	name                string
	dataType            string
	length              int
	precision           int
	scale               int
	nullable            bool
	defaultValue        interface{}
	useCurrent          bool
	useCurrentOnUpdate  bool
	unique              bool
	index               bool
	primary             bool
	autoIncrement       bool
	unsigned            bool
	comment             string
	charset             string
	collation           string
	after               string
	first               bool
	change              bool
	enumValues          []string
}

// Nullability methods
func (c *ColumnBuilder) Nullable() Column {
	c.nullable = true
	return c
}

func (c *ColumnBuilder) NotNull() Column {
	c.nullable = false
	return c
}

// Default methods
func (c *ColumnBuilder) Default(value interface{}) Column {
	c.defaultValue = value
	return c
}

func (c *ColumnBuilder) UseCurrent() Column {
	c.useCurrent = true
	return c
}

func (c *ColumnBuilder) UseCurrentOnUpdate() Column {
	c.useCurrentOnUpdate = true
	return c
}

// Constraint methods
func (c *ColumnBuilder) Unique() Column {
	c.unique = true
	return c
}

func (c *ColumnBuilder) Index() Column {
	c.index = true
	return c
}

func (c *ColumnBuilder) Primary() Column {
	c.primary = true
	return c
}

func (c *ColumnBuilder) AutoIncrement() Column {
	c.autoIncrement = true
	return c
}

func (c *ColumnBuilder) Unsigned() Column {
	c.unsigned = true
	return c
}

// Metadata methods
func (c *ColumnBuilder) Comment(comment string) Column {
	c.comment = comment
	return c
}

func (c *ColumnBuilder) Charset(charset string) Column {
	c.charset = charset
	return c
}

func (c *ColumnBuilder) Collation(collation string) Column {
	c.collation = collation
	return c
}

// Positioning methods
func (c *ColumnBuilder) After(column string) Column {
	c.after = column
	return c
}

func (c *ColumnBuilder) First() Column {
	c.first = true
	return c
}

// Change methods
func (c *ColumnBuilder) Change() Column {
	c.change = true
	return c
}

// Getter methods
func (c *ColumnBuilder) GetName() string {
	return c.name
}

func (c *ColumnBuilder) GetType() string {
	return c.dataType
}

// ToSQL generates SQL for the column based on driver
func (c *ColumnBuilder) ToSQL(driver string) string {
	// Special handling for SQLite auto-increment primary keys
	if driver == "sqlite3" && c.autoIncrement && c.primary {
		sql := c.name + " INTEGER PRIMARY KEY AUTOINCREMENT"
		
		if c.defaultValue != nil {
			switch v := c.defaultValue.(type) {
			case string:
				sql += fmt.Sprintf(" DEFAULT '%s'", v)
			default:
				sql += fmt.Sprintf(" DEFAULT %v", v)
			}
		}
		
		if c.useCurrent {
			sql += " DEFAULT CURRENT_TIMESTAMP"
		}
		
		if c.useCurrentOnUpdate && driver == "mysql" {
			sql += " ON UPDATE CURRENT_TIMESTAMP"
		}
		
		if c.comment != "" && driver == "mysql" {
			sql += fmt.Sprintf(" COMMENT '%s'", c.comment)
		}
		
		return sql
	}
	
	// Handle ENUM types for SQLite
	if driver == "sqlite3" && c.dataType == "ENUM" && len(c.enumValues) > 0 {
		sql := c.name + " TEXT"
		
		if !c.nullable {
			sql += " NOT NULL"
		}
		
		// Add CHECK constraint for ENUM values
		quotedValues := make([]string, len(c.enumValues))
		for i, value := range c.enumValues {
			quotedValues[i] = fmt.Sprintf("'%s'", value)
		}
		checkConstraint := fmt.Sprintf(" CHECK (%s IN (%s))", c.name, strings.Join(quotedValues, ", "))
		sql += checkConstraint
		
		if c.defaultValue != nil {
			switch v := c.defaultValue.(type) {
			case string:
				sql += fmt.Sprintf(" DEFAULT '%s'", v)
			default:
				sql += fmt.Sprintf(" DEFAULT %v", v)
			}
		}
		
		return sql
	}
	
	sql := c.name + " " + c.getDataTypeSQL(driver)
	
	if c.unsigned && (driver == "mysql") {
		sql += " UNSIGNED"
	}
	
	if !c.nullable {
		sql += " NOT NULL"
	}
	
	if c.autoIncrement {
		switch driver {
		case "mysql":
			sql += " AUTO_INCREMENT"
		case "postgres":
			// PostgreSQL uses SERIAL or BIGSERIAL
			if c.dataType == "BIGINT" {
				sql = c.name + " BIGSERIAL"
			} else {
				sql = c.name + " SERIAL"
			}
		case "sqlite3":
			// This case is handled above for primary keys
			// For non-primary auto-increment columns, SQLite doesn't support AUTOINCREMENT
			// without PRIMARY KEY, so we'll ignore it here
		}
	}
	
	if c.defaultValue != nil {
		switch v := c.defaultValue.(type) {
		case string:
			sql += fmt.Sprintf(" DEFAULT '%s'", v)
		default:
			sql += fmt.Sprintf(" DEFAULT %v", v)
		}
	}
	
	if c.useCurrent {
		switch driver {
		case "mysql":
			sql += " DEFAULT CURRENT_TIMESTAMP"
		case "postgres":
			sql += " DEFAULT CURRENT_TIMESTAMP"
		case "sqlite3":
			sql += " DEFAULT CURRENT_TIMESTAMP"
		}
	}
	
	if c.useCurrentOnUpdate && driver == "mysql" {
		sql += " ON UPDATE CURRENT_TIMESTAMP"
	}
	
	if c.comment != "" && driver == "mysql" {
		sql += fmt.Sprintf(" COMMENT '%s'", c.comment)
	}
	
	return sql
}

func (c *ColumnBuilder) getDataTypeSQL(driver string) string {
	switch c.dataType {
	case "VARCHAR":
		return fmt.Sprintf("VARCHAR(%d)", c.length)
	case "CHAR":
		return fmt.Sprintf("CHAR(%d)", c.length)
	case "FLOAT":
		if c.precision > 0 && c.scale > 0 {
			return fmt.Sprintf("FLOAT(%d,%d)", c.precision, c.scale)
		}
		return "FLOAT"
	case "DOUBLE":
		if c.precision > 0 && c.scale > 0 {
			return fmt.Sprintf("DOUBLE(%d,%d)", c.precision, c.scale)
		}
		return "DOUBLE"
	case "DECIMAL":
		if c.precision > 0 && c.scale > 0 {
			return fmt.Sprintf("DECIMAL(%d,%d)", c.precision, c.scale)
		}
		return "DECIMAL(8,2)"
	case "ENUM":
		if driver == "sqlite3" {
			// SQLite doesn't support ENUM, use TEXT with CHECK constraint
			return "TEXT"
		} else if len(c.enumValues) > 0 {
			quotedValues := make([]string, len(c.enumValues))
			for i, value := range c.enumValues {
				quotedValues[i] = fmt.Sprintf("'%s'", value)
			}
			return fmt.Sprintf("ENUM(%s)", strings.Join(quotedValues, ","))
		}
		return "ENUM('')"
	case "SET":
		if len(c.enumValues) > 0 {
			quotedValues := make([]string, len(c.enumValues))
			for i, value := range c.enumValues {
				quotedValues[i] = fmt.Sprintf("'%s'", value)
			}
			return fmt.Sprintf("SET(%s)", strings.Join(quotedValues, ","))
		}
		return "SET('')"
	case "JSON":
		switch driver {
		case "mysql":
			return "JSON"
		case "postgres":
			return "JSONB"
		case "sqlite3":
			return "TEXT"
		default:
			return "TEXT"
		}
	case "BOOLEAN":
		switch driver {
		case "mysql":
			return "BOOLEAN"
		case "postgres":
			return "BOOLEAN"
		case "sqlite3":
			return "BOOLEAN"
		default:
			return "BOOLEAN"
		}
	default:
		return c.dataType
	}
}

// IndexBuilder implements Index interface
type IndexBuilder struct {
	name      string
	columns   []string
	type_     string
	algorithm string
	comment   string
	lengths   map[string]int
	where     string
}

func (i *IndexBuilder) Algorithm(algorithm string) Index {
	i.algorithm = algorithm
	return i
}

func (i *IndexBuilder) Comment(comment string) Index {
	i.comment = comment
	return i
}

func (i *IndexBuilder) Length(lengths map[string]int) Index {
	i.lengths = lengths
	return i
}

func (i *IndexBuilder) Where(condition string) Index {
	i.where = condition
	return i
}

func (i *IndexBuilder) ToSQL(tableName, driver string) string {
	if i.type_ == "PRIMARY KEY" {
		return fmt.Sprintf("ALTER TABLE %s ADD PRIMARY KEY (%s)", tableName, strings.Join(i.columns, ", "))
	}
	
	indexType := ""
	if i.type_ == "UNIQUE" {
		indexType = "UNIQUE "
	}
	
	sql := fmt.Sprintf("CREATE %sINDEX %s ON %s (%s)", indexType, i.name, tableName, strings.Join(i.columns, ", "))
	
	if i.algorithm != "" && driver == "mysql" {
		sql += fmt.Sprintf(" USING %s", i.algorithm)
	}
	
	if i.where != "" && driver == "postgres" {
		sql += fmt.Sprintf(" WHERE %s", i.where)
	}
	
	return sql
}

// ForeignKeyBuilder implements ForeignKey interface
type ForeignKeyBuilder struct {
	name             string
	column           string
	referencedTable  string
	referencedColumn string
	onDelete         string
	onUpdate         string
}

func (fk *ForeignKeyBuilder) References(column string) ForeignKey {
	fk.referencedColumn = column
	return fk
}

func (fk *ForeignKeyBuilder) On(table string) ForeignKey {
	fk.referencedTable = table
	return fk
}

func (fk *ForeignKeyBuilder) OnDelete(action string) ForeignKey {
	fk.onDelete = action
	return fk
}

func (fk *ForeignKeyBuilder) OnUpdate(action string) ForeignKey {
	fk.onUpdate = action
	return fk
}

func (fk *ForeignKeyBuilder) Constrained(table ...string) ForeignKey {
	if len(table) > 0 {
		fk.referencedTable = table[0]
	} else {
		// Auto-determine table name from column name
		if strings.HasSuffix(fk.column, "_id") {
			fk.referencedTable = strings.TrimSuffix(fk.column, "_id") + "s"
		}
	}
	
	fk.referencedColumn = "id"
	return fk
}

func (fk *ForeignKeyBuilder) CascadeOnDelete() ForeignKey {
	fk.onDelete = "CASCADE"
	return fk
}

func (fk *ForeignKeyBuilder) CascadeOnUpdate() ForeignKey {
	fk.onUpdate = "CASCADE"
	return fk
}

func (fk *ForeignKeyBuilder) ToSQL(tableName, driver string) string {
	if driver == "sqlite3" {
		// SQLite foreign keys need to be defined during table creation
		// or require enabling foreign key constraints
		// For now, we'll create a comment indicating the foreign key relationship
		return fmt.Sprintf("-- Foreign key: %s.%s -> %s.%s (SQLite foreign keys need special handling)",
			tableName, fk.column, fk.referencedTable, fk.referencedColumn)
	}
	
	sql := fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s (%s)",
		tableName, fk.name, fk.column, fk.referencedTable, fk.referencedColumn)
	
	if fk.onDelete != "" {
		sql += fmt.Sprintf(" ON DELETE %s", fk.onDelete)
	}
	
	if fk.onUpdate != "" {
		sql += fmt.Sprintf(" ON UPDATE %s", fk.onUpdate)
	}
	
	return sql
}

// Migrator manages database migrations
type Migrator struct {
	db         *sql.DB
	driver     string
	schema     SchemaBuilder
	migrations map[string]Migration
	tableName  string
}

func NewMigrator(db *sql.DB, driver string) *Migrator {
	return &Migrator{
		db:         db,
		driver:     driver,
		schema:     NewSchemaBuilder(db, driver),
		migrations: make(map[string]Migration),
		tableName:  "migrations",
	}
}

func (m *Migrator) SetMigrationsTable(tableName string) {
	m.tableName = tableName
}

func (m *Migrator) Register(migration Migration) {
	m.migrations[migration.GetName()] = migration
}

func (m *Migrator) RegisterFromDirectory(directory string) error {
	files, err := os.ReadDir(directory)
	if err != nil {
		return fmt.Errorf("failed to read migration directory: %w", err)
	}
	
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".go") {
			// This would require reflection or code generation to load Go migration files
			// For now, we'll skip this and rely on manual registration
		}
	}
	
	return nil
}

func (m *Migrator) Run() error {
	if err := m.createMigrationsTable(); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}
	
	pendingMigrations, err := m.getPendingMigrations()
	if err != nil {
		return fmt.Errorf("failed to get pending migrations: %w", err)
	}
	
	if len(pendingMigrations) == 0 {
		Info("Nothing to migrate")
		return nil
	}
	
	batch, err := m.getNextBatchNumber()
	if err != nil {
		return fmt.Errorf("failed to get next batch number: %w", err)
	}
	
	for _, migration := range pendingMigrations {
		Info(fmt.Sprintf("Migrating: %s", migration.GetName()))
		
		if err := migration.Up(); err != nil {
			return fmt.Errorf("migration %s failed: %w", migration.GetName(), err)
		}
		
		migration.SetBatch(batch)
		if err := m.logMigration(migration); err != nil {
			return fmt.Errorf("failed to log migration: %w", err)
		}
		
		Info(fmt.Sprintf("Migrated: %s", migration.GetName()))
	}
	
	return nil
}

func (m *Migrator) Rollback(steps int) error {
	if steps <= 0 {
		steps = 1
	}
	
	batches, err := m.getLastBatches(steps)
	if err != nil {
		return fmt.Errorf("failed to get last batches: %w", err)
	}
	
	for _, batch := range batches {
		migrations, err := m.getMigrationsFromBatch(batch)
		if err != nil {
			return fmt.Errorf("failed to get migrations from batch %d: %w", batch, err)
		}
		
		// Rollback in reverse order
		for i := len(migrations) - 1; i >= 0; i-- {
			migration := migrations[i]
			Info(fmt.Sprintf("Rolling back: %s", migration.GetName()))
			
			if err := migration.Down(); err != nil {
				return fmt.Errorf("rollback %s failed: %w", migration.GetName(), err)
			}
			
			if err := m.removeMigrationLog(migration); err != nil {
				return fmt.Errorf("failed to remove migration log: %w", err)
			}
			
			Info(fmt.Sprintf("Rolled back: %s", migration.GetName()))
		}
	}
	
	return nil
}

func (m *Migrator) Reset() error {
	batches, err := m.getAllBatches()
	if err != nil {
		return fmt.Errorf("failed to get all batches: %w", err)
	}
	
	return m.Rollback(len(batches))
}

func (m *Migrator) Fresh() error {
	// Drop all tables and rerun migrations
	tables, err := m.getAllTables()
	if err != nil {
		return fmt.Errorf("failed to get all tables: %w", err)
	}
	
	for _, table := range tables {
		if err := m.schema.DropIfExists(table); err != nil {
			return fmt.Errorf("failed to drop table %s: %w", table, err)
		}
	}
	
	return m.Run()
}

func (m *Migrator) Status() ([]MigrationStatus, error) {
	ranMigrations, err := m.getRanMigrations()
	if err != nil {
		return nil, fmt.Errorf("failed to get ran migrations: %w", err)
	}
	
	ranMap := make(map[string]int)
	for _, rm := range ranMigrations {
		ranMap[rm.Migration] = rm.Batch
	}
	
	var statuses []MigrationStatus
	for name, migration := range m.migrations {
		status := MigrationStatus{
			Name:      name,
			Timestamp: migration.GetTimestamp(),
			Ran:       false,
			Batch:     0,
		}
		
		if batch, exists := ranMap[name]; exists {
			status.Ran = true
			status.Batch = batch
		}
		
		statuses = append(statuses, status)
	}
	
	// Sort by timestamp
	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].Timestamp < statuses[j].Timestamp
	})
	
	return statuses, nil
}

func (m *Migrator) createMigrationsTable() error {
	exists, err := m.schema.HasTable(m.tableName)
	if err != nil {
		return err
	}
	
	if exists {
		return nil
	}
	
	return m.schema.Create(m.tableName, func(table Table) {
		table.ID()
		table.String("migration")
		table.Integer("batch")
	})
}

func (m *Migrator) getPendingMigrations() ([]Migration, error) {
	ranMigrations, err := m.getRanMigrations()
	if err != nil {
		return nil, err
	}
	
	ranMap := make(map[string]bool)
	for _, rm := range ranMigrations {
		ranMap[rm.Migration] = true
	}
	
	var pending []Migration
	var migrationNames []string
	
	for name := range m.migrations {
		migrationNames = append(migrationNames, name)
	}
	
	sort.Strings(migrationNames)
	
	for _, name := range migrationNames {
		if !ranMap[name] {
			pending = append(pending, m.migrations[name])
		}
	}
	
	return pending, nil
}

func (m *Migrator) getRanMigrations() ([]RanMigration, error) {
	var ranMigrations []RanMigration
	
	query := fmt.Sprintf("SELECT migration, batch FROM %s ORDER BY batch, migration", m.tableName)
	rows, err := m.db.Query(query)
	if err != nil {
		return ranMigrations, err
	}
	defer rows.Close()
	
	for rows.Next() {
		var rm RanMigration
		if err := rows.Scan(&rm.Migration, &rm.Batch); err != nil {
			return nil, err
		}
		ranMigrations = append(ranMigrations, rm)
	}
	
	return ranMigrations, rows.Err()
}

func (m *Migrator) getNextBatchNumber() (int, error) {
	query := fmt.Sprintf("SELECT COALESCE(MAX(batch), 0) + 1 FROM %s", m.tableName)
	var batch int
	err := m.db.QueryRow(query).Scan(&batch)
	return batch, err
}

func (m *Migrator) getLastBatches(count int) ([]int, error) {
	query := fmt.Sprintf("SELECT DISTINCT batch FROM %s ORDER BY batch DESC LIMIT %d", m.tableName, count)
	rows, err := m.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var batches []int
	for rows.Next() {
		var batch int
		if err := rows.Scan(&batch); err != nil {
			return nil, err
		}
		batches = append(batches, batch)
	}
	
	return batches, rows.Err()
}

func (m *Migrator) getAllBatches() ([]int, error) {
	query := fmt.Sprintf("SELECT DISTINCT batch FROM %s ORDER BY batch DESC", m.tableName)
	rows, err := m.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var batches []int
	for rows.Next() {
		var batch int
		if err := rows.Scan(&batch); err != nil {
			return nil, err
		}
		batches = append(batches, batch)
	}
	
	return batches, rows.Err()
}

func (m *Migrator) getMigrationsFromBatch(batch int) ([]Migration, error) {
	query := fmt.Sprintf("SELECT migration FROM %s WHERE batch = ? ORDER BY migration", m.tableName)
	var placeholder string
	switch m.driver {
	case "postgres":
		placeholder = "$1"
	default:
		placeholder = "?"
	}
	query = strings.Replace(query, "?", placeholder, -1)
	
	rows, err := m.db.Query(query, batch)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var migrations []Migration
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		
		if migration, exists := m.migrations[name]; exists {
			migration.SetBatch(batch)
			migrations = append(migrations, migration)
		}
	}
	
	return migrations, rows.Err()
}

func (m *Migrator) getAllTables() ([]string, error) {
	var query string
	switch m.driver {
	case "mysql":
		query = "SHOW TABLES"
	case "postgres":
		query = "SELECT tablename FROM pg_tables WHERE schemaname = 'public'"
	case "sqlite3":
		query = "SELECT name FROM sqlite_master WHERE type='table'"
	default:
		return nil, fmt.Errorf("unsupported driver: %s", m.driver)
	}
	
	rows, err := m.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}
	
	return tables, rows.Err()
}

func (m *Migrator) logMigration(migration Migration) error {
	query := fmt.Sprintf("INSERT INTO %s (migration, batch) VALUES (?, ?)", m.tableName)
	var placeholders []string
	switch m.driver {
	case "postgres":
		placeholders = []string{"$1", "$2"}
	default:
		placeholders = []string{"?", "?"}
	}
	query = strings.Replace(query, "?", placeholders[0], 1)
	query = strings.Replace(query, "?", placeholders[1], 1)
	
	_, err := m.db.Exec(query, migration.GetName(), migration.GetBatch())
	return err
}

func (m *Migrator) removeMigrationLog(migration Migration) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE migration = ?", m.tableName)
	var placeholder string
	switch m.driver {
	case "postgres":
		placeholder = "$1"
	default:
		placeholder = "?"
	}
	query = strings.Replace(query, "?", placeholder, -1)
	
	_, err := m.db.Exec(query, migration.GetName())
	return err
}

// Helper types and functions
type RanMigration struct {
	Migration string
	Batch     int
}

type MigrationStatus struct {
	Name      string
	Timestamp string
	Ran       bool
	Batch     int
}

func extractTimestamp(migrationName string) string {
	parts := strings.Split(migrationName, "_")
	if len(parts) >= 4 {
		return strings.Join(parts[:4], "_")
	}
	return ""
}

func generateIndexName(tableName, prefix string, columns []string) string {
	return fmt.Sprintf("%s_%s_%s", prefix, tableName, strings.Join(columns, "_"))
}

func generateForeignKeyName(tableName, column string) string {
	return fmt.Sprintf("fk_%s_%s", tableName, column)
}

// Global schema builder function
func Schema() SchemaBuilder {
	if globalApp != nil && globalApp.Container() != nil {
		if schema, err := globalApp.Container().Make("schema"); err == nil {
			if s, ok := schema.(SchemaBuilder); ok {
				return s
			}
		}
	}
	return nil
}

// Context schema helper function
func GetSchemaFromContext(c Context) SchemaBuilder {
	// For now, use the global schema since Application interface doesn't expose Container
	// TODO: Extend Application interface to provide access to Container/SchemaBuilder
	return Schema()
}

// Example migration implementation
type CreateUsersTable struct {
	*BaseMigration
	schema SchemaBuilder
}

func NewCreateUsersTable(schema SchemaBuilder) *CreateUsersTable {
	return &CreateUsersTable{
		BaseMigration: NewBaseMigration("2024_01_01_000001_create_users_table"),
		schema:        schema,
	}
}

func (cut *CreateUsersTable) Up() error {
	return cut.schema.Create("users", func(table Table) {
		table.ID()
		table.String("name")
		table.String("email").Unique()
		table.String("password")
		table.RememberToken()
		table.Timestamps()
	})
}

func (cut *CreateUsersTable) Down() error {
	return cut.schema.Drop("users")
}

// MakeMigration creates a new migration file
func MakeMigration(name string, directory string) error {
	timestamp := time.Now().Format("2006_01_02_150405")
	filename := fmt.Sprintf("%s_%s.go", timestamp, name)
	filepath := filepath.Join(directory, filename)
	
	template := fmt.Sprintf(`package migrations

import "github.com/onyx-go/framework"

type %s struct {
	*framework.BaseMigration
	schema framework.SchemaBuilder
}

func New%s(schema framework.SchemaBuilder) *%s {
	return &%s{
		BaseMigration: framework.NewBaseMigration("%s_%s"),
		schema:        schema,
	}
}

func (m *%s) Up() error {
	return m.schema.Create("table_name", func(table framework.Table) {
		table.ID()
		table.Timestamps()
	})
}

func (m *%s) Down() error {
	return m.schema.Drop("table_name")
}
`, toCamelCase(name), toCamelCase(name), toCamelCase(name), toCamelCase(name), timestamp, name, toCamelCase(name), toCamelCase(name))
	
	return os.WriteFile(filepath, []byte(template), 0644)
}

func toCamelCase(s string) string {
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}