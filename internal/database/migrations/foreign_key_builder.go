package migrations

import "strings"

// foreignKeyBuilder implements the ForeignKey interface
type foreignKeyBuilder struct {
	name             string
	column           string
	referencedTable  string
	referencedColumn string
	onDelete         string
	onUpdate         string
}

// Target definition

// References sets the referenced column
func (fkb *foreignKeyBuilder) References(column string) ForeignKey {
	fkb.referencedColumn = column
	return fkb
}

// On sets the referenced table
func (fkb *foreignKeyBuilder) On(table string) ForeignKey {
	fkb.referencedTable = table
	return fkb
}

// Actions

// OnDelete sets the on delete action
func (fkb *foreignKeyBuilder) OnDelete(action string) ForeignKey {
	fkb.onDelete = action
	return fkb
}

// OnUpdate sets the on update action
func (fkb *foreignKeyBuilder) OnUpdate(action string) ForeignKey {
	fkb.onUpdate = action
	return fkb
}

// CascadeOnDelete sets CASCADE on delete
func (fkb *foreignKeyBuilder) CascadeOnDelete() ForeignKey {
	fkb.onDelete = ForeignKeyCascade
	return fkb
}

// CascadeOnUpdate sets CASCADE on update
func (fkb *foreignKeyBuilder) CascadeOnUpdate() ForeignKey {
	fkb.onUpdate = ForeignKeyCascade
	return fkb
}

// NullOnDelete sets SET NULL on delete
func (fkb *foreignKeyBuilder) NullOnDelete() ForeignKey {
	fkb.onDelete = ForeignKeySetNull
	return fkb
}

// RestrictOnDelete sets RESTRICT on delete
func (fkb *foreignKeyBuilder) RestrictOnDelete() ForeignKey {
	fkb.onDelete = ForeignKeyRestrict
	return fkb
}

// RestrictOnUpdate sets RESTRICT on update
func (fkb *foreignKeyBuilder) RestrictOnUpdate() ForeignKey {
	fkb.onUpdate = ForeignKeyRestrict
	return fkb
}

// NoActionOnDelete sets NO ACTION on delete
func (fkb *foreignKeyBuilder) NoActionOnDelete() ForeignKey {
	fkb.onDelete = ForeignKeyNoAction
	return fkb
}

// NoActionOnUpdate sets NO ACTION on update
func (fkb *foreignKeyBuilder) NoActionOnUpdate() ForeignKey {
	fkb.onUpdate = ForeignKeyNoAction
	return fkb
}

// Naming

// Name sets the foreign key constraint name
func (fkb *foreignKeyBuilder) Name(name string) ForeignKey {
	fkb.name = name
	return fkb
}

// GetDefinition returns the foreign key definition for SQL generation
func (fkb *foreignKeyBuilder) GetDefinition() ForeignKeyDefinition {
	// Auto-determine referenced table and column if not set
	if fkb.referencedTable == "" && strings.HasSuffix(fkb.column, "_id") {
		// Extract table name from column name (e.g., user_id -> users)
		tableName := strings.TrimSuffix(fkb.column, "_id")
		fkb.referencedTable = tableName + "s" // Simple pluralization
	}
	
	if fkb.referencedColumn == "" {
		fkb.referencedColumn = "id" // Default to 'id' column
	}
	
	return ForeignKeyDefinition{
		Name:            fkb.name,
		Column:          fkb.column,
		ReferencedTable: fkb.referencedTable,
		ReferencedColumn: fkb.referencedColumn,
		OnDelete:        fkb.onDelete,
		OnUpdate:        fkb.onUpdate,
	}
}

// Ensure foreignKeyBuilder implements ForeignKey
var _ ForeignKey = (*foreignKeyBuilder)(nil)