package migrations

import (
	"fmt"
	"strings"
)

// NewMySQLGenerator creates a new MySQL SQL generator
func NewMySQLGenerator() SQLGenerator {
	return &mysqlGenerator{}
}

// NewPostgreSQLGenerator creates a new PostgreSQL SQL generator
func NewPostgreSQLGenerator() SQLGenerator {
	return &postgresqlGenerator{}
}

// NewSQLiteGenerator creates a new SQLite SQL generator
func NewSQLiteGenerator() SQLGenerator {
	return &sqliteGenerator{}
}

// mysqlGenerator implements SQLGenerator for MySQL
type mysqlGenerator struct{}

// Table operations
func (mg *mysqlGenerator) GenerateCreateTable(definition *TableDefinition) string {
	var parts []string
	
	// Add columns
	for _, column := range definition.Columns {
		parts = append(parts, "  "+mg.generateColumnSQL(column))
	}
	
	// Add primary key if not already handled by column
	var primaryColumns []string
	for _, column := range definition.Columns {
		if column.Primary {
			primaryColumns = append(primaryColumns, column.Name)
		}
	}
	if len(primaryColumns) > 0 {
		parts = append(parts, fmt.Sprintf("  PRIMARY KEY (%s)", strings.Join(primaryColumns, ", ")))
	}
	
	// Add other indexes during table creation if needed
	for _, index := range definition.Indexes {
		if index.Type != "" && index.Type != "primary" {
			indexSQL := mg.generateIndexInTable(index)
			if indexSQL != "" {
				parts = append(parts, "  "+indexSQL)
			}
		}
	}
	
	tableSQL := fmt.Sprintf("CREATE TABLE %s (\n%s\n)", definition.Name, strings.Join(parts, ",\n"))
	
	// Add table options
	var options []string
	if definition.Engine != "" {
		options = append(options, "ENGINE="+definition.Engine)
	}
	if definition.Charset != "" {
		options = append(options, "DEFAULT CHARSET="+definition.Charset)
	}
	if definition.Collation != "" {
		options = append(options, "COLLATE="+definition.Collation)
	}
	if definition.Comment != "" {
		options = append(options, fmt.Sprintf("COMMENT='%s'", definition.Comment))
	}
	
	if len(options) > 0 {
		tableSQL += " " + strings.Join(options, " ")
	}
	
	return tableSQL
}

func (mg *mysqlGenerator) GenerateAlterTable(tableName string, definition *TableDefinition) []string {
	var statements []string
	
	// Add columns
	for _, column := range definition.Columns {
		if column.Change {
			statements = append(statements, fmt.Sprintf("ALTER TABLE %s MODIFY COLUMN %s", tableName, mg.generateColumnSQL(column)))
		} else {
			statements = append(statements, fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", tableName, mg.generateColumnSQL(column)))
		}
	}
	
	return statements
}

func (mg *mysqlGenerator) GenerateDropTable(tableName string, ifExists bool) string {
	if ifExists {
		return fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName)
	}
	return fmt.Sprintf("DROP TABLE %s", tableName)
}

func (mg *mysqlGenerator) GenerateRenameTable(oldName, newName string) string {
	return fmt.Sprintf("RENAME TABLE %s TO %s", oldName, newName)
}

// Column operations
func (mg *mysqlGenerator) GenerateAddColumn(tableName string, column ColumnDefinition) string {
	return fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", tableName, mg.generateColumnSQL(column))
}

func (mg *mysqlGenerator) GenerateModifyColumn(tableName string, column ColumnDefinition) string {
	return fmt.Sprintf("ALTER TABLE %s MODIFY COLUMN %s", tableName, mg.generateColumnSQL(column))
}

func (mg *mysqlGenerator) GenerateDropColumn(tableName string, columnName string) string {
	return fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", tableName, columnName)
}

func (mg *mysqlGenerator) GenerateRenameColumn(tableName, oldName, newName string) string {
	return fmt.Sprintf("ALTER TABLE %s RENAME COLUMN %s TO %s", tableName, oldName, newName)
}

// Index operations
func (mg *mysqlGenerator) GenerateCreateIndex(tableName string, index IndexDefinition) string {
	indexType := ""
	switch index.Type {
	case "unique":
		indexType = "UNIQUE "
	case "spatial":
		indexType = "SPATIAL "
	case "fulltext":
		indexType = "FULLTEXT "
	}
	
	sql := fmt.Sprintf("CREATE %sINDEX %s ON %s (%s)", indexType, index.Name, tableName, strings.Join(index.Columns, ", "))
	
	if index.Algorithm != "" {
		sql += " USING " + index.Algorithm
	}
	
	return sql
}

func (mg *mysqlGenerator) GenerateDropIndex(tableName string, indexName string) string {
	return fmt.Sprintf("DROP INDEX %s ON %s", indexName, tableName)
}

// Foreign key operations
func (mg *mysqlGenerator) GenerateAddForeignKey(tableName string, foreignKey ForeignKeyDefinition) string {
	sql := fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s (%s)",
		tableName, foreignKey.Name, foreignKey.Column, foreignKey.ReferencedTable, foreignKey.ReferencedColumn)
	
	if foreignKey.OnDelete != "" {
		sql += " ON DELETE " + foreignKey.OnDelete
	}
	if foreignKey.OnUpdate != "" {
		sql += " ON UPDATE " + foreignKey.OnUpdate
	}
	
	return sql
}

func (mg *mysqlGenerator) GenerateDropForeignKey(tableName string, keyName string) string {
	return fmt.Sprintf("ALTER TABLE %s DROP FOREIGN KEY %s", tableName, keyName)
}

// Introspection queries
func (mg *mysqlGenerator) GetTableExistsQuery(tableName string) string {
	return "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?"
}

func (mg *mysqlGenerator) GetColumnExistsQuery(tableName, columnName string) string {
	return "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?"
}

func (mg *mysqlGenerator) GetIndexExistsQuery(tableName, indexName string) string {
	return "SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = ? AND index_name = ?"
}

func (mg *mysqlGenerator) GetForeignKeyExistsQuery(tableName, keyName string) string {
	return "SELECT COUNT(*) FROM information_schema.key_column_usage WHERE table_schema = DATABASE() AND table_name = ? AND constraint_name = ?"
}

func (mg *mysqlGenerator) GetColumnListingQuery(tableName string) string {
	return "SELECT column_name FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = ? ORDER BY ordinal_position"
}

func (mg *mysqlGenerator) GetTableListingQuery() string {
	return "SELECT table_name FROM information_schema.tables WHERE table_schema = DATABASE()"
}

// Type mapping
func (mg *mysqlGenerator) MapColumnType(columnType string) string {
	switch columnType {
	case ColumnTypeString:
		return "VARCHAR"
	case ColumnTypeText:
		return "TEXT"
	case ColumnTypeLongText:
		return "LONGTEXT"
	case ColumnTypeMediumText:
		return "MEDIUMTEXT"
	case ColumnTypeTinyText:
		return "TINYTEXT"
	case ColumnTypeChar:
		return "CHAR"
	case ColumnTypeInteger:
		return "INT"
	case ColumnTypeBigInteger:
		return "BIGINT"
	case ColumnTypeSmallInteger:
		return "SMALLINT"
	case ColumnTypeTinyInteger:
		return "TINYINT"
	case ColumnTypeFloat:
		return "FLOAT"
	case ColumnTypeDouble:
		return "DOUBLE"
	case ColumnTypeDecimal:
		return "DECIMAL"
	case ColumnTypeBoolean:
		return "BOOLEAN"
	case ColumnTypeDate:
		return "DATE"
	case ColumnTypeDateTime:
		return "DATETIME"
	case ColumnTypeDateTimeTz:
		return "DATETIME"
	case ColumnTypeTime:
		return "TIME"
	case ColumnTypeTimeTz:
		return "TIME"
	case ColumnTypeTimestamp:
		return "TIMESTAMP"
	case ColumnTypeTimestampTz:
		return "TIMESTAMP"
	case ColumnTypeYear:
		return "YEAR"
	case ColumnTypeJSON:
		return "JSON"
	case ColumnTypeJSONB:
		return "JSON"
	case ColumnTypeUUID:
		return "CHAR(36)"
	case ColumnTypeBinary:
		return "BLOB"
	case ColumnTypeEnum:
		return "ENUM"
	default:
		return columnType
	}
}

func (mg *mysqlGenerator) SupportsFeature(feature string) bool {
	switch feature {
	case FeatureTransactions, FeatureForeignKeys, FeatureJsonColumns, FeatureGeneratedColumns,
		 FeatureRenameColumns, FeatureRenameIndexes, FeatureDropIndexes, FeatureMultipleIndexes,
		 FeatureIndexAlgorithms, FeatureCommentOnColumns, FeatureCommentOnTables:
		return true
	case FeaturePartialIndexes, FeatureCheckConstraints:
		return false
	default:
		return false
	}
}

// Helper methods
func (mg *mysqlGenerator) generateColumnSQL(column ColumnDefinition) string {
	sql := column.Name + " " + mg.getColumnTypeSQL(column)
	
	if column.Unsigned {
		sql += " UNSIGNED"
	}
	
	if !column.Nullable {
		sql += " NOT NULL"
	}
	
	if column.AutoIncrement {
		sql += " AUTO_INCREMENT"
	}
	
	if column.Default != nil {
		switch v := column.Default.(type) {
		case string:
			if v == "CURRENT_TIMESTAMP" {
				sql += " DEFAULT CURRENT_TIMESTAMP"
			} else {
				sql += fmt.Sprintf(" DEFAULT '%s'", v)
			}
		default:
			sql += fmt.Sprintf(" DEFAULT %v", v)
		}
	}
	
	if column.Comment != "" {
		sql += fmt.Sprintf(" COMMENT '%s'", column.Comment)
	}
	
	if column.After != "" {
		sql += " AFTER " + column.After
	}
	
	if column.First {
		sql += " FIRST"
	}
	
	return sql
}

func (mg *mysqlGenerator) getColumnTypeSQL(column ColumnDefinition) string {
	baseType := mg.MapColumnType(column.Type)
	
	switch column.Type {
	case ColumnTypeString:
		length := 255
		if column.Length != nil {
			length = *column.Length
		}
		return fmt.Sprintf("VARCHAR(%d)", length)
	case ColumnTypeChar:
		length := 255
		if column.Length != nil {
			length = *column.Length
		}
		return fmt.Sprintf("CHAR(%d)", length)
	case ColumnTypeFloat, ColumnTypeDouble:
		if column.Precision != nil {
			if column.Scale != nil {
				return fmt.Sprintf("%s(%d,%d)", baseType, *column.Precision, *column.Scale)
			}
			return fmt.Sprintf("%s(%d)", baseType, *column.Precision)
		}
		return baseType
	case ColumnTypeDecimal:
		precision := 8
		scale := 2
		if column.Precision != nil {
			precision = *column.Precision
		}
		if column.Scale != nil {
			scale = *column.Scale
		}
		return fmt.Sprintf("DECIMAL(%d,%d)", precision, scale)
	default:
		return baseType
	}
}

func (mg *mysqlGenerator) generateIndexInTable(index IndexDefinition) string {
	switch index.Type {
	case "unique":
		return fmt.Sprintf("UNIQUE KEY %s (%s)", index.Name, strings.Join(index.Columns, ", "))
	case "spatial":
		return fmt.Sprintf("SPATIAL KEY %s (%s)", index.Name, strings.Join(index.Columns, ", "))
	case "fulltext":
		return fmt.Sprintf("FULLTEXT KEY %s (%s)", index.Name, strings.Join(index.Columns, ", "))
	default:
		return fmt.Sprintf("KEY %s (%s)", index.Name, strings.Join(index.Columns, ", "))
	}
}

// PostgreSQL and SQLite generators (simplified implementations)
type postgresqlGenerator struct {
	*mysqlGenerator // Embed for basic functionality, override as needed
}

type sqliteGenerator struct {
	*mysqlGenerator // Embed for basic functionality, override as needed
}

// Override specific methods for PostgreSQL
func (pg *postgresqlGenerator) GetTableExistsQuery(tableName string) string {
	return "SELECT COUNT(*) FROM information_schema.tables WHERE table_name = $1"
}

func (pg *postgresqlGenerator) GetColumnExistsQuery(tableName, columnName string) string {
	return "SELECT COUNT(*) FROM information_schema.columns WHERE table_name = $1 AND column_name = $2"
}

// Override specific methods for SQLite
func (sg *sqliteGenerator) GetTableExistsQuery(tableName string) string {
	return "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name = ?"
}

func (sg *sqliteGenerator) GetColumnExistsQuery(tableName, columnName string) string {
	return "SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?"
}

func (sg *sqliteGenerator) SupportsFeature(feature string) bool {
	switch feature {
	case FeatureTransactions:
		return true
	case FeatureForeignKeys, FeatureJsonColumns, FeatureGeneratedColumns,
		 FeatureRenameColumns, FeatureRenameIndexes, FeatureDropIndexes, FeatureMultipleIndexes,
		 FeatureIndexAlgorithms, FeatureCommentOnColumns, FeatureCommentOnTables,
		 FeaturePartialIndexes, FeatureCheckConstraints:
		return false
	default:
		return false
	}
}

// Ensure generators implement SQLGenerator
var _ SQLGenerator = (*mysqlGenerator)(nil)
var _ SQLGenerator = (*postgresqlGenerator)(nil)
var _ SQLGenerator = (*sqliteGenerator)(nil)