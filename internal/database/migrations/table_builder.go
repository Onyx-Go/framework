package migrations

import (
	"strings"
)

// tableBuilder implements the Table interface
type tableBuilder struct {
	name        string
	action      string // "create" or "alter"
	driver      string
	sqlGen      SQLGenerator
	columns     []*columnBuilder
	indexes     []*indexBuilder
	foreignKeys []*foreignKeyBuilder
	commands    []string
	engine      string
	charset     string
	collation   string
	comment     string
	temporary   bool
	rawSQL      []string
}

// NewTableBuilder creates a new table builder
func NewTableBuilder(name, action, driver string, sqlGen SQLGenerator) Table {
	return &tableBuilder{
		name:        name,
		action:      action,
		driver:      driver,
		sqlGen:      sqlGen,
		columns:     make([]*columnBuilder, 0),
		indexes:     make([]*indexBuilder, 0),
		foreignKeys: make([]*foreignKeyBuilder, 0),
		commands:    make([]string, 0),
		rawSQL:      make([]string, 0),
	}
}

// Primary key columns

// ID creates an auto-incrementing ID column
func (tb *tableBuilder) ID() Column {
	column := &columnBuilder{
		name:          "id",
		columnType:    ColumnTypeBigInteger,
		autoIncrement: true,
		nullable:      false,
		primary:       true,
		unsigned:      true,
	}
	
	tb.columns = append(tb.columns, column)
	return column
}

// BigID creates a big auto-incrementing ID column
func (tb *tableBuilder) BigID() Column {
	column := &columnBuilder{
		name:          "id",
		columnType:    ColumnTypeBigInteger,
		autoIncrement: true,
		nullable:      false,
		primary:       true,
		unsigned:      true,
	}
	
	tb.columns = append(tb.columns, column)
	return column
}

// UUIID creates a UUID primary key column
func (tb *tableBuilder) UUIID() Column {
	column := &columnBuilder{
		name:       "id",
		columnType: ColumnTypeUUID,
		nullable:   false,
		primary:    true,
	}
	
	tb.columns = append(tb.columns, column)
	return column
}

// String column types

// String creates a VARCHAR column
func (tb *tableBuilder) String(name string, length ...int) Column {
	columnLength := 255
	if len(length) > 0 {
		columnLength = length[0]
	}
	
	column := &columnBuilder{
		name:       name,
		columnType: ColumnTypeString,
		length:     &columnLength,
		nullable:   true,
	}
	
	tb.columns = append(tb.columns, column)
	return column
}

// Text creates a TEXT column
func (tb *tableBuilder) Text(name string) Column {
	column := &columnBuilder{
		name:       name,
		columnType: ColumnTypeText,
		nullable:   true,
	}
	
	tb.columns = append(tb.columns, column)
	return column
}

// LongText creates a LONGTEXT column
func (tb *tableBuilder) LongText(name string) Column {
	column := &columnBuilder{
		name:       name,
		columnType: ColumnTypeLongText,
		nullable:   true,
	}
	
	tb.columns = append(tb.columns, column)
	return column
}

// MediumText creates a MEDIUMTEXT column
func (tb *tableBuilder) MediumText(name string) Column {
	column := &columnBuilder{
		name:       name,
		columnType: ColumnTypeMediumText,
		nullable:   true,
	}
	
	tb.columns = append(tb.columns, column)
	return column
}

// TinyText creates a TINYTEXT column
func (tb *tableBuilder) TinyText(name string) Column {
	column := &columnBuilder{
		name:       name,
		columnType: ColumnTypeTinyText,
		nullable:   true,
	}
	
	tb.columns = append(tb.columns, column)
	return column
}

// Char creates a CHAR column
func (tb *tableBuilder) Char(name string, length int) Column {
	column := &columnBuilder{
		name:       name,
		columnType: ColumnTypeChar,
		length:     &length,
		nullable:   true,
	}
	
	tb.columns = append(tb.columns, column)
	return column
}

// Numeric types

// Integer creates an INT column
func (tb *tableBuilder) Integer(name string) Column {
	column := &columnBuilder{
		name:       name,
		columnType: ColumnTypeInteger,
		nullable:   true,
	}
	
	tb.columns = append(tb.columns, column)
	return column
}

// BigInteger creates a BIGINT column
func (tb *tableBuilder) BigInteger(name string) Column {
	column := &columnBuilder{
		name:       name,
		columnType: ColumnTypeBigInteger,
		nullable:   true,
	}
	
	tb.columns = append(tb.columns, column)
	return column
}

// SmallInteger creates a SMALLINT column
func (tb *tableBuilder) SmallInteger(name string) Column {
	column := &columnBuilder{
		name:       name,
		columnType: ColumnTypeSmallInteger,
		nullable:   true,
	}
	
	tb.columns = append(tb.columns, column)
	return column
}

// TinyInteger creates a TINYINT column
func (tb *tableBuilder) TinyInteger(name string) Column {
	column := &columnBuilder{
		name:       name,
		columnType: ColumnTypeTinyInteger,
		nullable:   true,
	}
	
	tb.columns = append(tb.columns, column)
	return column
}

// UnsignedInteger creates an unsigned INT column
func (tb *tableBuilder) UnsignedInteger(name string) Column {
	column := &columnBuilder{
		name:       name,
		columnType: ColumnTypeInteger,
		nullable:   true,
		unsigned:   true,
	}
	
	tb.columns = append(tb.columns, column)
	return column
}

// UnsignedBigInteger creates an unsigned BIGINT column
func (tb *tableBuilder) UnsignedBigInteger(name string) Column {
	column := &columnBuilder{
		name:       name,
		columnType: ColumnTypeBigInteger,
		nullable:   true,
		unsigned:   true,
	}
	
	tb.columns = append(tb.columns, column)
	return column
}

// UnsignedSmallInteger creates an unsigned SMALLINT column
func (tb *tableBuilder) UnsignedSmallInteger(name string) Column {
	column := &columnBuilder{
		name:       name,
		columnType: ColumnTypeSmallInteger,
		nullable:   true,
		unsigned:   true,
	}
	
	tb.columns = append(tb.columns, column)
	return column
}

// UnsignedTinyInteger creates an unsigned TINYINT column
func (tb *tableBuilder) UnsignedTinyInteger(name string) Column {
	column := &columnBuilder{
		name:       name,
		columnType: ColumnTypeTinyInteger,
		nullable:   true,
		unsigned:   true,
	}
	
	tb.columns = append(tb.columns, column)
	return column
}

// Decimal types

// Float creates a FLOAT column
func (tb *tableBuilder) Float(name string, precision ...int) Column {
	column := &columnBuilder{
		name:       name,
		columnType: ColumnTypeFloat,
		nullable:   true,
	}
	
	if len(precision) > 0 {
		column.precision = &precision[0]
	}
	
	tb.columns = append(tb.columns, column)
	return column
}

// Double creates a DOUBLE column
func (tb *tableBuilder) Double(name string, precision ...int) Column {
	column := &columnBuilder{
		name:       name,
		columnType: ColumnTypeDouble,
		nullable:   true,
	}
	
	if len(precision) > 0 {
		column.precision = &precision[0]
	}
	
	tb.columns = append(tb.columns, column)
	return column
}

// Decimal creates a DECIMAL column
func (tb *tableBuilder) Decimal(name string, precision, scale int) Column {
	column := &columnBuilder{
		name:       name,
		columnType: ColumnTypeDecimal,
		precision:  &precision,
		scale:      &scale,
		nullable:   true,
	}
	
	tb.columns = append(tb.columns, column)
	return column
}

// UnsignedFloat creates an unsigned FLOAT column
func (tb *tableBuilder) UnsignedFloat(name string, precision ...int) Column {
	column := &columnBuilder{
		name:       name,
		columnType: ColumnTypeFloat,
		nullable:   true,
		unsigned:   true,
	}
	
	if len(precision) > 0 {
		column.precision = &precision[0]
	}
	
	tb.columns = append(tb.columns, column)
	return column
}

// UnsignedDouble creates an unsigned DOUBLE column
func (tb *tableBuilder) UnsignedDouble(name string, precision ...int) Column {
	column := &columnBuilder{
		name:       name,
		columnType: ColumnTypeDouble,
		nullable:   true,
		unsigned:   true,
	}
	
	if len(precision) > 0 {
		column.precision = &precision[0]
	}
	
	tb.columns = append(tb.columns, column)
	return column
}

// UnsignedDecimal creates an unsigned DECIMAL column
func (tb *tableBuilder) UnsignedDecimal(name string, precision, scale int) Column {
	column := &columnBuilder{
		name:       name,
		columnType: ColumnTypeDecimal,
		precision:  &precision,
		scale:      &scale,
		nullable:   true,
		unsigned:   true,
	}
	
	tb.columns = append(tb.columns, column)
	return column
}

// Date and time types

// Date creates a DATE column
func (tb *tableBuilder) Date(name string) Column {
	column := &columnBuilder{
		name:       name,
		columnType: ColumnTypeDate,
		nullable:   true,
	}
	
	tb.columns = append(tb.columns, column)
	return column
}

// DateTime creates a DATETIME column
func (tb *tableBuilder) DateTime(name string) Column {
	column := &columnBuilder{
		name:       name,
		columnType: ColumnTypeDateTime,
		nullable:   true,
	}
	
	tb.columns = append(tb.columns, column)
	return column
}

// DateTimeTz creates a DATETIME with timezone column
func (tb *tableBuilder) DateTimeTz(name string) Column {
	column := &columnBuilder{
		name:       name,
		columnType: ColumnTypeDateTimeTz,
		nullable:   true,
	}
	
	tb.columns = append(tb.columns, column)
	return column
}

// Time creates a TIME column
func (tb *tableBuilder) Time(name string) Column {
	column := &columnBuilder{
		name:       name,
		columnType: ColumnTypeTime,
		nullable:   true,
	}
	
	tb.columns = append(tb.columns, column)
	return column
}

// TimeTz creates a TIME with timezone column
func (tb *tableBuilder) TimeTz(name string) Column {
	column := &columnBuilder{
		name:       name,
		columnType: ColumnTypeTimeTz,
		nullable:   true,
	}
	
	tb.columns = append(tb.columns, column)
	return column
}

// Timestamp creates a TIMESTAMP column
func (tb *tableBuilder) Timestamp(name string) Column {
	column := &columnBuilder{
		name:       name,
		columnType: ColumnTypeTimestamp,
		nullable:   true,
	}
	
	tb.columns = append(tb.columns, column)
	return column
}

// TimestampTz creates a TIMESTAMP with timezone column
func (tb *tableBuilder) TimestampTz(name string) Column {
	column := &columnBuilder{
		name:       name,
		columnType: ColumnTypeTimestampTz,
		nullable:   true,
	}
	
	tb.columns = append(tb.columns, column)
	return column
}

// Year creates a YEAR column
func (tb *tableBuilder) Year(name string) Column {
	column := &columnBuilder{
		name:       name,
		columnType: ColumnTypeYear,
		nullable:   true,
	}
	
	tb.columns = append(tb.columns, column)
	return column
}

// Boolean and binary types

// Boolean creates a BOOLEAN column
func (tb *tableBuilder) Boolean(name string) Column {
	column := &columnBuilder{
		name:       name,
		columnType: ColumnTypeBoolean,
		nullable:   true,
	}
	
	tb.columns = append(tb.columns, column)
	return column
}

// Binary creates a BINARY column
func (tb *tableBuilder) Binary(name string) Column {
	column := &columnBuilder{
		name:       name,
		columnType: ColumnTypeBinary,
		nullable:   true,
	}
	
	tb.columns = append(tb.columns, column)
	return column
}

// JSON types

// JSON creates a JSON column
func (tb *tableBuilder) JSON(name string) Column {
	column := &columnBuilder{
		name:       name,
		columnType: ColumnTypeJSON,
		nullable:   true,
	}
	
	tb.columns = append(tb.columns, column)
	return column
}

// JSONB creates a JSONB column
func (tb *tableBuilder) JSONB(name string) Column {
	column := &columnBuilder{
		name:       name,
		columnType: ColumnTypeJSONB,
		nullable:   true,
	}
	
	tb.columns = append(tb.columns, column)
	return column
}

// UUID type

// UUID creates a UUID column
func (tb *tableBuilder) UUID(name string) Column {
	column := &columnBuilder{
		name:       name,
		columnType: ColumnTypeUUID,
		nullable:   true,
	}
	
	tb.columns = append(tb.columns, column)
	return column
}

// Enum type

// Enum creates an ENUM column
func (tb *tableBuilder) Enum(name string, values []string) Column {
	column := &columnBuilder{
		name:       name,
		columnType: ColumnTypeEnum,
		enumValues: values,
		nullable:   true,
	}
	
	tb.columns = append(tb.columns, column)
	return column
}

// Special columns

// Timestamps adds created_at and updated_at columns
func (tb *tableBuilder) Timestamps() Table {
	createdAt := &columnBuilder{
		name:         "created_at",
		columnType:   ColumnTypeTimestamp,
		nullable:     true,
		defaultValue: "CURRENT_TIMESTAMP",
	}
	
	updatedAt := &columnBuilder{
		name:                "updated_at",
		columnType:          ColumnTypeTimestamp,
		nullable:            true,
		defaultValue:        "CURRENT_TIMESTAMP",
		useCurrentOnUpdate:  true,
	}
	
	tb.columns = append(tb.columns, createdAt, updatedAt)
	return tb
}

// SoftDeletes adds deleted_at column
func (tb *tableBuilder) SoftDeletes() Table {
	deletedAt := &columnBuilder{
		name:       "deleted_at",
		columnType: ColumnTypeTimestamp,
		nullable:   true,
	}
	
	tb.columns = append(tb.columns, deletedAt)
	return tb
}

// SoftDeletesTz adds deleted_at column with timezone
func (tb *tableBuilder) SoftDeletesTz() Table {
	deletedAt := &columnBuilder{
		name:       "deleted_at",
		columnType: ColumnTypeTimestampTz,
		nullable:   true,
	}
	
	tb.columns = append(tb.columns, deletedAt)
	return tb
}

// RememberToken adds remember_token column
func (tb *tableBuilder) RememberToken() Table {
	length := 100
	rememberToken := &columnBuilder{
		name:       "remember_token",
		columnType: ColumnTypeString,
		length:     &length,
		nullable:   true,
	}
	
	tb.columns = append(tb.columns, rememberToken)
	return tb
}

// Column modifications

// DropColumn drops columns
func (tb *tableBuilder) DropColumn(names ...string) Table {
	for _, name := range names {
		tb.commands = append(tb.commands, "DROP COLUMN "+name)
	}
	return tb
}

// RenameColumn renames a column
func (tb *tableBuilder) RenameColumn(from, to string) Table {
	tb.commands = append(tb.commands, "RENAME COLUMN "+from+" TO "+to)
	return tb
}

// ModifyColumn modifies an existing column
func (tb *tableBuilder) ModifyColumn(name string) Column {
	column := &columnBuilder{
		name:   name,
		change: true,
	}
	
	tb.columns = append(tb.columns, column)
	return column
}

// Indexes

// Index creates an index
func (tb *tableBuilder) Index(columns ...string) Index {
	name := generateIndexName(tb.name, "idx", columns)
	
	index := &indexBuilder{
		name:    name,
		columns: columns,
		indexType: "",
	}
	
	tb.indexes = append(tb.indexes, index)
	return index
}

// Unique creates a unique index
func (tb *tableBuilder) Unique(columns ...string) Index {
	name := generateIndexName(tb.name, "unq", columns)
	
	index := &indexBuilder{
		name:    name,
		columns: columns,
		indexType: "unique",
	}
	
	tb.indexes = append(tb.indexes, index)
	return index
}

// Primary creates a primary key
func (tb *tableBuilder) Primary(columns ...string) Index {
	index := &indexBuilder{
		name:    "PRIMARY",
		columns: columns,
		indexType: "primary",
	}
	
	tb.indexes = append(tb.indexes, index)
	return index
}

// SpatialIndex creates a spatial index
func (tb *tableBuilder) SpatialIndex(columns ...string) Index {
	name := generateIndexName(tb.name, "spatial", columns)
	
	index := &indexBuilder{
		name:    name,
		columns: columns,
		indexType: "spatial",
	}
	
	tb.indexes = append(tb.indexes, index)
	return index
}

// FullText creates a full-text index
func (tb *tableBuilder) FullText(columns ...string) Index {
	name := generateIndexName(tb.name, "fulltext", columns)
	
	index := &indexBuilder{
		name:    name,
		columns: columns,
		indexType: "fulltext",
	}
	
	tb.indexes = append(tb.indexes, index)
	return index
}

// Index operations

// DropIndex drops an index
func (tb *tableBuilder) DropIndex(name string) Table {
	tb.commands = append(tb.commands, "DROP INDEX "+name)
	return tb
}

// DropUnique drops a unique constraint
func (tb *tableBuilder) DropUnique(name string) Table {
	tb.commands = append(tb.commands, "DROP INDEX "+name)
	return tb
}

// DropPrimary drops the primary key
func (tb *tableBuilder) DropPrimary() Table {
	tb.commands = append(tb.commands, "DROP PRIMARY KEY")
	return tb
}

// DropSpatialIndex drops a spatial index
func (tb *tableBuilder) DropSpatialIndex(name string) Table {
	tb.commands = append(tb.commands, "DROP INDEX "+name)
	return tb
}

// DropFullText drops a full-text index
func (tb *tableBuilder) DropFullText(name string) Table {
	tb.commands = append(tb.commands, "DROP INDEX "+name)
	return tb
}

// Foreign keys

// Foreign creates a foreign key constraint
func (tb *tableBuilder) Foreign(column string) ForeignKey {
	foreignKey := &foreignKeyBuilder{
		name:   generateForeignKeyName(tb.name, column),
		column: column,
	}
	
	tb.foreignKeys = append(tb.foreignKeys, foreignKey)
	return foreignKey
}

// DropForeign drops a foreign key constraint
func (tb *tableBuilder) DropForeign(name string) Table {
	tb.commands = append(tb.commands, "DROP FOREIGN KEY "+name)
	return tb
}

// Table options

// Engine sets the table engine (MySQL specific)
func (tb *tableBuilder) Engine(engine string) Table {
	tb.engine = engine
	return tb
}

// Charset sets the table charset
func (tb *tableBuilder) Charset(charset string) Table {
	tb.charset = charset
	return tb
}

// Collation sets the table collation
func (tb *tableBuilder) Collation(collation string) Table {
	tb.collation = collation
	return tb
}

// Comment sets the table comment
func (tb *tableBuilder) Comment(comment string) Table {
	tb.comment = comment
	return tb
}

// Temporary marks the table as temporary
func (tb *tableBuilder) Temporary() Table {
	tb.temporary = true
	return tb
}

// Raw adds raw SQL to the table definition
func (tb *tableBuilder) Raw(sql string) Table {
	tb.rawSQL = append(tb.rawSQL, sql)
	return tb
}

// GetDefinition returns the table definition for SQL generation
func (tb *tableBuilder) GetDefinition() *TableDefinition {
	definition := &TableDefinition{
		Name:        tb.name,
		Engine:      tb.engine,
		Charset:     tb.charset,
		Collation:   tb.collation,
		Comment:     tb.comment,
		Temporary:   tb.temporary,
		RawSQL:      tb.rawSQL,
		Columns:     make([]ColumnDefinition, 0),
		Indexes:     make([]IndexDefinition, 0),
		ForeignKeys: make([]ForeignKeyDefinition, 0),
	}
	
	// Convert columns
	for _, col := range tb.columns {
		definition.Columns = append(definition.Columns, col.GetDefinition())
	}
	
	// Convert indexes
	for _, idx := range tb.indexes {
		definition.Indexes = append(definition.Indexes, idx.GetDefinition())
	}
	
	// Convert foreign keys
	for _, fk := range tb.foreignKeys {
		definition.ForeignKeys = append(definition.ForeignKeys, fk.GetDefinition())
	}
	
	return definition
}

// Helper functions

// generateIndexName generates an index name
func generateIndexName(tableName, prefix string, columns []string) string {
	return prefix + "_" + tableName + "_" + strings.Join(columns, "_")
}

// generateForeignKeyName generates a foreign key name
func generateForeignKeyName(tableName, column string) string {
	return "fk_" + tableName + "_" + column
}

// Ensure tableBuilder implements Table
var _ Table = (*tableBuilder)(nil)