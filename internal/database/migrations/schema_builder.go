package migrations

import (
	"database/sql"
	"fmt"
)

// schemaBuilder implements the SchemaBuilder interface
type schemaBuilder struct {
	db     *sql.DB
	driver string
	config *MigrationConfig
	sqlGen SQLGenerator
}

// NewSchemaBuilder creates a new schema builder instance
func NewSchemaBuilder(db *sql.DB, driver string, config *MigrationConfig) SchemaBuilder {
	sb := &schemaBuilder{
		db:     db,
		driver: driver,
		config: config,
	}
	
	// Set SQL generator
	if config != nil {
		sb.sqlGen = config.GetSQLGenerator()
	} else {
		sb.sqlGen = getDefaultSQLGenerator(driver)
	}
	
	return sb
}

// Create creates a new table with the given callback
func (sb *schemaBuilder) Create(tableName string, callback func(Table)) error {
	table := NewTableBuilder(tableName, "create", sb.driver, sb.sqlGen)
	callback(table)
	
	operation := &SchemaOperation{
		Type:       OperationCreate,
		Table:      tableName,
		Definition: table.GetDefinition(),
	}
	
	return sb.executeOperation(operation)
}

// Table is an alias for Alter for fluent interface
func (sb *schemaBuilder) Table(tableName string, callback func(Table)) error {
	return sb.Alter(tableName, callback)
}

// Alter alters an existing table with the given callback
func (sb *schemaBuilder) Alter(tableName string, callback func(Table)) error {
	table := NewTableBuilder(tableName, "alter", sb.driver, sb.sqlGen)
	callback(table)
	
	operation := &SchemaOperation{
		Type:       OperationAlter,
		Table:      tableName,
		Definition: table.GetDefinition(),
	}
	
	return sb.executeOperation(operation)
}

// Drop drops a table
func (sb *schemaBuilder) Drop(tableName string) error {
	operation := &SchemaOperation{
		Type:  OperationDrop,
		Table: tableName,
		SQL:   sb.sqlGen.GenerateDropTable(tableName, false),
	}
	
	return sb.executeOperation(operation)
}

// DropIfExists drops a table if it exists
func (sb *schemaBuilder) DropIfExists(tableName string) error {
	operation := &SchemaOperation{
		Type:  OperationDrop,
		Table: tableName,
		SQL:   sb.sqlGen.GenerateDropTable(tableName, true),
	}
	
	return sb.executeOperation(operation)
}

// Rename renames a table
func (sb *schemaBuilder) Rename(from, to string) error {
	operation := &SchemaOperation{
		Type:     OperationRename,
		Table:    from,
		NewTable: to,
		SQL:      sb.sqlGen.GenerateRenameTable(from, to),
	}
	
	return sb.executeOperation(operation)
}

// HasTable checks if a table exists
func (sb *schemaBuilder) HasTable(tableName string) (bool, error) {
	query := sb.sqlGen.GetTableExistsQuery(tableName)
	
	var count int
	err := sb.db.QueryRow(query, tableName).Scan(&count)
	if err != nil {
		return false, err
	}
	
	return count > 0, nil
}

// HasColumn checks if a column exists in a table
func (sb *schemaBuilder) HasColumn(tableName, columnName string) (bool, error) {
	query := sb.sqlGen.GetColumnExistsQuery(tableName, columnName)
	
	var count int
	err := sb.db.QueryRow(query, tableName, columnName).Scan(&count)
	if err != nil {
		return false, err
	}
	
	return count > 0, nil
}

// HasIndex checks if an index exists
func (sb *schemaBuilder) HasIndex(tableName, indexName string) (bool, error) {
	query := sb.sqlGen.GetIndexExistsQuery(tableName, indexName)
	
	var count int
	err := sb.db.QueryRow(query, tableName, indexName).Scan(&count)
	if err != nil {
		return false, err
	}
	
	return count > 0, nil
}

// HasForeignKey checks if a foreign key exists
func (sb *schemaBuilder) HasForeignKey(tableName, keyName string) (bool, error) {
	query := sb.sqlGen.GetForeignKeyExistsQuery(tableName, keyName)
	
	var count int
	err := sb.db.QueryRow(query, tableName, keyName).Scan(&count)
	if err != nil {
		return false, err
	}
	
	return count > 0, nil
}

// GetColumnListing returns a list of columns for a table
func (sb *schemaBuilder) GetColumnListing(tableName string) ([]string, error) {
	query := sb.sqlGen.GetColumnListingQuery(tableName)
	
	rows, err := sb.db.Query(query, tableName)
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

// GetColumnType returns the type of a column
func (sb *schemaBuilder) GetColumnType(tableName, columnName string) (string, error) {
	columns, err := sb.GetColumnListing(tableName)
	if err != nil {
		return "", err
	}
	
	// This is a simplified implementation
	// In a real implementation, you'd query the information schema
	for _, col := range columns {
		if col == columnName {
			return "varchar", nil // Default fallback
		}
	}
	
	return "", fmt.Errorf("column %s not found in table %s", columnName, tableName)
}

// GetTableListing returns a list of all tables
func (sb *schemaBuilder) GetTableListing() ([]string, error) {
	query := sb.sqlGen.GetTableListingQuery()
	
	rows, err := sb.db.Query(query)
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

// Raw executes raw SQL
func (sb *schemaBuilder) Raw(sql string, bindings ...interface{}) error {
	_, err := sb.db.Exec(sql, bindings...)
	return err
}

// GetDriverName returns the database driver name
func (sb *schemaBuilder) GetDriverName() string {
	return sb.driver
}

// GetConnection returns the database connection
func (sb *schemaBuilder) GetConnection() *sql.DB {
	return sb.db
}

// executeOperation executes a schema operation
func (sb *schemaBuilder) executeOperation(operation *SchemaOperation) error {
	var statements []string
	
	switch operation.Type {
	case OperationCreate:
		if operation.Definition != nil {
			statements = append(statements, sb.sqlGen.GenerateCreateTable(operation.Definition))
			
			// Add indexes
			for _, index := range operation.Definition.Indexes {
				statements = append(statements, sb.sqlGen.GenerateCreateIndex(operation.Table, index))
			}
			
			// Add foreign keys
			for _, fk := range operation.Definition.ForeignKeys {
				statements = append(statements, sb.sqlGen.GenerateAddForeignKey(operation.Table, fk))
			}
		}
		
	case OperationAlter:
		if operation.Definition != nil {
			statements = append(statements, sb.sqlGen.GenerateAlterTable(operation.Table, operation.Definition)...)
		}
		
	case OperationDrop, OperationRename:
		if operation.SQL != "" {
			statements = append(statements, operation.SQL)
		}
	}
	
	// Add any raw SQL
	if operation.SQL != "" && operation.Type != OperationDrop && operation.Type != OperationRename {
		statements = append(statements, operation.SQL)
	}
	
	// Execute statements
	if sb.config != nil && sb.config.UseTransactions {
		return sb.executeInTransaction(statements)
	}
	
	return sb.executeStatements(statements)
}

// executeInTransaction executes statements within a transaction
func (sb *schemaBuilder) executeInTransaction(statements []string) error {
	tx, err := sb.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	
	for _, stmt := range statements {
		if stmt == "" {
			continue
		}
		
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("failed to execute SQL: %s, error: %w", stmt, err)
		}
	}
	
	return tx.Commit()
}

// executeStatements executes statements without a transaction
func (sb *schemaBuilder) executeStatements(statements []string) error {
	for _, stmt := range statements {
		if stmt == "" {
			continue
		}
		
		if _, err := sb.db.Exec(stmt); err != nil {
			return fmt.Errorf("failed to execute SQL: %s, error: %w", stmt, err)
		}
	}
	
	return nil
}

// getDefaultSQLGenerator returns the default SQL generator for a driver
func getDefaultSQLGenerator(driver string) SQLGenerator {
	switch driver {
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

// Ensure schemaBuilder implements SchemaBuilder
var _ SchemaBuilder = (*schemaBuilder)(nil)