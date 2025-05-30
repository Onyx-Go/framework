package migrations

// columnBuilder implements the Column interface
type columnBuilder struct {
	name                string
	columnType          string
	length              *int
	precision           *int
	scale               *int
	nullable            bool
	defaultValue        interface{}
	autoIncrement       bool
	unsigned            bool
	charset             string
	collation           string
	comment             string
	after               string
	first               bool
	virtual             string
	stored              string
	primary             bool
	unique              bool
	index               bool
	spatialIndex        bool
	fullText            bool
	change              bool
	enumValues          []string
	useCurrentOnUpdate  bool
}

// Basic properties

// Length sets the column length
func (cb *columnBuilder) Length(length int) Column {
	cb.length = &length
	return cb
}

// Precision sets the column precision
func (cb *columnBuilder) Precision(precision int) Column {
	cb.precision = &precision
	return cb
}

// Scale sets the column scale
func (cb *columnBuilder) Scale(scale int) Column {
	cb.scale = &scale
	return cb
}

// Constraints

// Nullable marks the column as nullable
func (cb *columnBuilder) Nullable() Column {
	cb.nullable = true
	return cb
}

// NotNull marks the column as not nullable
func (cb *columnBuilder) NotNull() Column {
	cb.nullable = false
	return cb
}

// Default sets the default value
func (cb *columnBuilder) Default(value interface{}) Column {
	cb.defaultValue = value
	return cb
}

// Indexes

// Primary marks the column as primary key
func (cb *columnBuilder) Primary() Column {
	cb.primary = true
	cb.nullable = false
	return cb
}

// Unique marks the column as unique
func (cb *columnBuilder) Unique() Column {
	cb.unique = true
	return cb
}

// Index marks the column to have an index
func (cb *columnBuilder) Index() Column {
	cb.index = true
	return cb
}

// SpatialIndex marks the column to have a spatial index
func (cb *columnBuilder) SpatialIndex() Column {
	cb.spatialIndex = true
	return cb
}

// FullText marks the column to have a full-text index
func (cb *columnBuilder) FullText() Column {
	cb.fullText = true
	return cb
}

// Auto increment

// AutoIncrement marks the column as auto-incrementing
func (cb *columnBuilder) AutoIncrement() Column {
	cb.autoIncrement = true
	cb.nullable = false
	return cb
}

// Unsigned (for numeric types)

// Unsigned marks the column as unsigned
func (cb *columnBuilder) Unsigned() Column {
	cb.unsigned = true
	return cb
}

// String specific

// Charset sets the column charset
func (cb *columnBuilder) Charset(charset string) Column {
	cb.charset = charset
	return cb
}

// Collation sets the column collation
func (cb *columnBuilder) Collation(collation string) Column {
	cb.collation = collation
	return cb
}

// Comments

// Comment sets the column comment
func (cb *columnBuilder) Comment(comment string) Column {
	cb.comment = comment
	return cb
}

// Placement (MySQL specific)

// After places the column after another column
func (cb *columnBuilder) After(column string) Column {
	cb.after = column
	return cb
}

// First places the column first in the table
func (cb *columnBuilder) First() Column {
	cb.first = true
	return cb
}

// Virtual/Generated columns

// VirtualAs creates a virtual generated column
func (cb *columnBuilder) VirtualAs(expression string) Column {
	cb.virtual = expression
	return cb
}

// StoredAs creates a stored generated column
func (cb *columnBuilder) StoredAs(expression string) Column {
	cb.stored = expression
	return cb
}

// Change/Add operations

// Change marks the column for modification
func (cb *columnBuilder) Change() Column {
	cb.change = true
	return cb
}

// Foreign key shorthand

// References creates a foreign key reference to another column
func (cb *columnBuilder) References(column string) ForeignKey {
	foreignKey := &foreignKeyBuilder{
		name:             generateForeignKeyName("", cb.name),
		column:           cb.name,
		referencedColumn: column,
	}
	
	return foreignKey
}

// ConstrainedBy sets up a foreign key constraint to a table
func (cb *columnBuilder) ConstrainedBy(table string) Column {
	// This would typically be handled by the table builder
	// For now, just return the column
	return cb
}

// CascadeOnUpdate sets cascade on update for foreign key
func (cb *columnBuilder) CascadeOnUpdate() Column {
	// This would be handled in conjunction with foreign key creation
	return cb
}

// CascadeOnDelete sets cascade on delete for foreign key
func (cb *columnBuilder) CascadeOnDelete() Column {
	// This would be handled in conjunction with foreign key creation
	return cb
}

// NullOnDelete sets null on delete for foreign key
func (cb *columnBuilder) NullOnDelete() Column {
	// This would be handled in conjunction with foreign key creation
	return cb
}

// RestrictOnDelete sets restrict on delete for foreign key
func (cb *columnBuilder) RestrictOnDelete() Column {
	// This would be handled in conjunction with foreign key creation
	return cb
}

// NoActionOnDelete sets no action on delete for foreign key
func (cb *columnBuilder) NoActionOnDelete() Column {
	// This would be handled in conjunction with foreign key creation
	return cb
}

// GetDefinition returns the column definition for SQL generation
func (cb *columnBuilder) GetDefinition() ColumnDefinition {
	return ColumnDefinition{
		Name:          cb.name,
		Type:          cb.columnType,
		Length:        cb.length,
		Precision:     cb.precision,
		Scale:         cb.scale,
		Nullable:      cb.nullable,
		Default:       cb.defaultValue,
		AutoIncrement: cb.autoIncrement,
		Unsigned:      cb.unsigned,
		Charset:       cb.charset,
		Collation:     cb.collation,
		Comment:       cb.comment,
		After:         cb.after,
		First:         cb.first,
		Virtual:       cb.virtual,
		Stored:        cb.stored,
		Primary:       cb.primary,
		Unique:        cb.unique,
		Index:         cb.index,
		SpatialIndex:  cb.spatialIndex,
		FullText:      cb.fullText,
		Change:        cb.change,
	}
}

// Ensure columnBuilder implements Column
var _ Column = (*columnBuilder)(nil)