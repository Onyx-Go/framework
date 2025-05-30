package onyx

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	*sql.DB
	driver string
}

type QueryBuilder struct {
	db              *DB
	table           string
	selects         []string
	wheres          []whereClause
	orders          []string
	limit           int
	offset          int
	joins           []string
	groupBy         []string
	having          []whereClause
	bindings        []interface{}
	eagerLoad       map[string]interface{}
	eagerLoadEngine *EagerLoadingEngine
	includeDeleted  bool // Whether to include soft-deleted records
}

type whereClause struct {
	column   string
	operator string
	value    interface{}
	boolean  string
}

type Model interface {
	TableName() string
}

func NewDB(driver, dsn string) (*DB, error) {
	sqlDB, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}
	
	if err := sqlDB.Ping(); err != nil {
		return nil, err
	}
	
	return &DB{
		DB:     sqlDB,
		driver: driver,
	}, nil
}

func (db *DB) Table(tableName string) *QueryBuilder {
	return &QueryBuilder{
		db:              db,
		table:           tableName,
		selects:         []string{"*"},
		wheres:          []whereClause{},
		orders:          []string{},
		joins:           []string{},
		groupBy:         []string{},
		having:          []whereClause{},
		bindings:        []interface{}{},
		eagerLoad:       make(map[string]interface{}),
		eagerLoadEngine: nil,
		includeDeleted:  false,
	}
}

func (db *DB) Model(model Model) *QueryBuilder {
	return db.Table(model.TableName())
}

func NewQueryBuilder(db *DB) *QueryBuilder {
	return &QueryBuilder{
		db:              db,
		selects:         []string{"*"},
		wheres:          []whereClause{},
		orders:          []string{},
		joins:           []string{},
		groupBy:         []string{},
		having:          []whereClause{},
		bindings:        []interface{}{},
		eagerLoad:       make(map[string]interface{}),
		eagerLoadEngine: nil,
		includeDeleted:  false,
	}
}

func (qb *QueryBuilder) Select(columns ...string) *QueryBuilder {
	qb.selects = columns
	return qb
}

func (qb *QueryBuilder) Where(column, operator string, value interface{}) *QueryBuilder {
	qb.wheres = append(qb.wheres, whereClause{
		column:   column,
		operator: operator,
		value:    value,
		boolean:  "AND",
	})
	return qb
}

func (qb *QueryBuilder) OrWhere(column, operator string, value interface{}) *QueryBuilder {
	qb.wheres = append(qb.wheres, whereClause{
		column:   column,
		operator: operator,
		value:    value,
		boolean:  "OR",
	})
	return qb
}


func (qb *QueryBuilder) OrderBy(column, direction string) *QueryBuilder {
	qb.orders = append(qb.orders, fmt.Sprintf("%s %s", column, strings.ToUpper(direction)))
	return qb
}

func (qb *QueryBuilder) Limit(limit int) *QueryBuilder {
	qb.limit = limit
	return qb
}

func (qb *QueryBuilder) Offset(offset int) *QueryBuilder {
	qb.offset = offset
	return qb
}

func (qb *QueryBuilder) Join(table, first, operator, second string) *QueryBuilder {
	qb.joins = append(qb.joins, fmt.Sprintf("JOIN %s ON %s %s %s", table, first, operator, second))
	return qb
}

func (qb *QueryBuilder) LeftJoin(table, first, operator, second string) *QueryBuilder {
	qb.joins = append(qb.joins, fmt.Sprintf("LEFT JOIN %s ON %s %s %s", table, first, operator, second))
	return qb
}

func (qb *QueryBuilder) GroupBy(columns ...string) *QueryBuilder {
	qb.groupBy = append(qb.groupBy, columns...)
	return qb
}

func (qb *QueryBuilder) Having(column, operator string, value interface{}) *QueryBuilder {
	qb.having = append(qb.having, whereClause{
		column:   column,
		operator: operator,
		value:    value,
		boolean:  "AND",
	})
	return qb
}

func (qb *QueryBuilder) Get(dest interface{}) error {
	query, args := qb.buildSelectQuery()
	
	rows, err := qb.db.Query(query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	
	return qb.scanRows(rows, dest)
}

func (qb *QueryBuilder) First(dest interface{}) error {
	qb.Limit(1)
	query, args := qb.buildSelectQuery()
	
	row := qb.db.QueryRow(query, args...)
	return qb.scanRow(row, dest)
}

func (qb *QueryBuilder) Insert(data map[string]interface{}) (int64, error) {
	columns := make([]string, 0, len(data))
	placeholders := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))
	
	for column, value := range data {
		columns = append(columns, column)
		placeholders = append(placeholders, "?")
		values = append(values, value)
	}
	
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		qb.table,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)
	
	result, err := qb.db.Exec(query, values...)
	if err != nil {
		return 0, err
	}
	
	return result.LastInsertId()
}

func (qb *QueryBuilder) Update(data map[string]interface{}) (int64, error) {
	setParts := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))
	
	for column, value := range data {
		setParts = append(setParts, fmt.Sprintf("%s = ?", column))
		values = append(values, value)
	}
	
	query := fmt.Sprintf("UPDATE %s SET %s", qb.table, strings.Join(setParts, ", "))
	
	if len(qb.wheres) > 0 {
		whereClause, whereArgs := qb.buildWhereClause(qb.wheres)
		query += " WHERE " + whereClause
		values = append(values, whereArgs...)
	}
	
	result, err := qb.db.Exec(query, values...)
	if err != nil {
		return 0, err
	}
	
	return result.RowsAffected()
}

func (qb *QueryBuilder) Delete() (int64, error) {
	// Perform soft delete by default for tables with deleted_at column
	now := time.Now()
	return qb.Update(map[string]interface{}{
		"deleted_at": now,
		"updated_at": now,
	})
}

// ForceDelete performs a hard delete, permanently removing records
func (qb *QueryBuilder) ForceDelete() (int64, error) {
	query := fmt.Sprintf("DELETE FROM %s", qb.table)
	var args []interface{}
	
	if len(qb.wheres) > 0 {
		whereClause, whereArgs := qb.buildWhereClause(qb.wheres)
		query += " WHERE " + whereClause
		args = whereArgs
	}
	
	result, err := qb.db.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	
	return result.RowsAffected()
}

// Restore restores soft-deleted records
func (qb *QueryBuilder) Restore() (int64, error) {
	return qb.Update(map[string]interface{}{
		"deleted_at": nil,
		"updated_at": time.Now(),
	})
}

func (qb *QueryBuilder) buildSelectQuery() (string, []interface{}) {
	// Apply soft delete filter before building query
	qb.applySoftDeleteFilter()
	
	query := fmt.Sprintf("SELECT %s FROM %s", strings.Join(qb.selects, ", "), qb.table)
	var args []interface{}
	
	if len(qb.joins) > 0 {
		query += " " + strings.Join(qb.joins, " ")
	}
	
	if len(qb.wheres) > 0 {
		whereClause, whereArgs := qb.buildWhereClause(qb.wheres)
		query += " WHERE " + whereClause
		args = append(args, whereArgs...)
	}
	
	if len(qb.groupBy) > 0 {
		query += " GROUP BY " + strings.Join(qb.groupBy, ", ")
	}
	
	if len(qb.having) > 0 {
		havingClause, havingArgs := qb.buildWhereClause(qb.having)
		query += " HAVING " + havingClause
		args = append(args, havingArgs...)
	}
	
	if len(qb.orders) > 0 {
		query += " ORDER BY " + strings.Join(qb.orders, ", ")
	}
	
	if qb.limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", qb.limit)
	}
	
	if qb.offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", qb.offset)
	}
	
	args = append(args, qb.bindings...)
	return query, args
}

func (qb *QueryBuilder) buildWhereClause(wheres []whereClause) (string, []interface{}) {
	var parts []string
	var args []interface{}
	
	for i, where := range wheres {
		var part string
		
		if i > 0 {
			part += fmt.Sprintf(" %s ", where.boolean)
		}
		
		if where.operator == "IN" {
			part += fmt.Sprintf("%s %s %s", where.column, where.operator, where.value)
		} else if where.operator == "IS NULL" || where.operator == "IS NOT NULL" {
			part += fmt.Sprintf("%s %s", where.column, where.operator)
		} else {
			part += fmt.Sprintf("%s %s ?", where.column, where.operator)
			args = append(args, where.value)
		}
		
		parts = append(parts, part)
	}
	
	return strings.Join(parts, ""), args
}

func (qb *QueryBuilder) scanRows(rows *sql.Rows, dest interface{}) error {
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr {
		return fmt.Errorf("dest must be a pointer")
	}
	
	destValue = destValue.Elem()
	if destValue.Kind() != reflect.Slice {
		return fmt.Errorf("dest must be a pointer to a slice")
	}
	
	elementType := destValue.Type().Elem()
	
	for rows.Next() {
		element := reflect.New(elementType).Elem()
		
		if err := qb.scanIntoStruct(rows, element.Addr().Interface()); err != nil {
			return err
		}
		
		destValue.Set(reflect.Append(destValue, element))
	}
	
	return rows.Err()
}

func (qb *QueryBuilder) scanRow(row *sql.Row, dest interface{}) error {
	return qb.scanIntoStruct(row, dest)
}

func (qb *QueryBuilder) scanIntoStruct(scanner interface{}, dest interface{}) error {
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr {
		return fmt.Errorf("dest must be a pointer")
	}
	
	destValue = destValue.Elem()
	if destValue.Kind() != reflect.Struct {
		return fmt.Errorf("dest must be a pointer to a struct")
	}
	
	destType := destValue.Type()
	
	switch s := scanner.(type) {
	case *sql.Rows:
		return qb.scanRowsIntoStruct(s, destValue, destType)
	case *sql.Row:
		return qb.scanSingleRowIntoStruct(s, destValue, destType)
	default:
		return fmt.Errorf("unsupported scanner type")
	}
}

// scanRowsIntoStruct handles scanning from sql.Rows (multiple rows)
func (qb *QueryBuilder) scanRowsIntoStruct(rows *sql.Rows, destValue reflect.Value, destType reflect.Type) error {
	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	
	scanArgs := make([]interface{}, len(columns))
	
	// Map columns to struct fields using 'db' tags
	for i, column := range columns {
		if field := qb.findFieldValueByColumn(destValue, column); field.IsValid() {
			if field.CanSet() {
				scanArgs[i] = qb.createScanDestination(field)
			} else {
				var dummy interface{}
				scanArgs[i] = &dummy
			}
		} else {
			// Column doesn't match any field, scan into dummy variable
			var dummy interface{}
			scanArgs[i] = &dummy
		}
	}
	
	return rows.Scan(scanArgs...)
}

// scanSingleRowIntoStruct handles scanning from sql.Row (single row)
func (qb *QueryBuilder) scanSingleRowIntoStruct(row *sql.Row, destValue reflect.Value, destType reflect.Type) error {
	// For sql.Row, we need to know which columns are being selected
	// We'll use the select clause from the query builder
	columns := qb.getSelectedColumns(destType)
	scanArgs := make([]interface{}, len(columns))
	
	// Map columns to struct fields
	for i, column := range columns {
		if field := qb.findFieldValueByColumn(destValue, column); field.IsValid() {
			if field.CanSet() {
				scanArgs[i] = qb.createScanDestination(field)
			} else {
				var dummy interface{}
				scanArgs[i] = &dummy
			}
		} else {
			var dummy interface{}
			scanArgs[i] = &dummy
		}
	}
	
	return row.Scan(scanArgs...)
}

// findFieldByColumn finds struct field by column name using db tags
func (qb *QueryBuilder) findFieldByColumn(structType reflect.Type, column string) reflect.StructField {
	field, _ := qb.findFieldByColumnRecursive(structType, column)
	return field
}

// findFieldByColumnRecursive recursively searches for field by column, handling embedded structs
func (qb *QueryBuilder) findFieldByColumnRecursive(structType reflect.Type, column string) (reflect.StructField, bool) {
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		
		// Handle embedded structs
		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			if foundField, found := qb.findFieldByColumnRecursive(field.Type, column); found {
				return foundField, true
			}
			continue
		}
		
		dbTag := field.Tag.Get("db")
		if dbTag == "" || dbTag == "-" {
			continue
		}
		
		if dbTag == column {
			return field, true
		}
	}
	return reflect.StructField{}, false
}

// findFieldValueByColumn finds the reflect.Value of field by column name, handling embedded structs
func (qb *QueryBuilder) findFieldValueByColumn(structValue reflect.Value, column string) reflect.Value {
	return qb.findFieldValueByColumnRecursive(structValue, column)
}

// findFieldValueByColumnRecursive recursively searches for field value by column
func (qb *QueryBuilder) findFieldValueByColumnRecursive(structValue reflect.Value, column string) reflect.Value {
	structType := structValue.Type()
	
	for i := 0; i < structValue.NumField(); i++ {
		field := structType.Field(i)
		fieldValue := structValue.Field(i)
		
		// Handle embedded structs
		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			if found := qb.findFieldValueByColumnRecursive(fieldValue, column); found.IsValid() {
				return found
			}
			continue
		}
		
		dbTag := field.Tag.Get("db")
		if dbTag == "" || dbTag == "-" {
			continue
		}
		
		if dbTag == column {
			return fieldValue
		}
	}
	return reflect.Value{}
}

// getSelectedColumns returns the columns to be selected based on the select clause
func (qb *QueryBuilder) getSelectedColumns(structType reflect.Type) []string {
	// If "*" is selected, get all columns from struct
	if len(qb.selects) == 1 && qb.selects[0] == "*" {
		return qb.getColumnsFromStruct(structType)
	}
	
	// Otherwise return the explicitly selected columns
	return qb.selects
}

// createScanDestination creates appropriate scan destination for different field types
func (qb *QueryBuilder) createScanDestination(field reflect.Value) interface{} {
	fieldType := field.Type()
	
	// Handle pointer types (for nullable fields like *time.Time)
	if fieldType.Kind() == reflect.Ptr {
		// For *time.Time and similar nullable types, create a custom scanner
		if fieldType.Elem().String() == "time.Time" {
			return &nullTimeScanner{field: field}
		}
		
		// For other pointer types, create a new instance
		elemType := fieldType.Elem()
		newVal := reflect.New(elemType)
		field.Set(newVal)
		return newVal.Interface()
	}
	
	// Handle sql.NullString, sql.NullInt64, etc.
	if fieldType.PkgPath() == "database/sql" {
		return field.Addr().Interface()
	}
	
	// For primitive types, we need to handle potential NULL values
	switch fieldType.Kind() {
	case reflect.String:
		return field.Addr().Interface()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return field.Addr().Interface()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return field.Addr().Interface()
	case reflect.Float32, reflect.Float64:
		return field.Addr().Interface()
	case reflect.Bool:
		return field.Addr().Interface()
	case reflect.Struct:
		// Handle time.Time and other structs
		if fieldType == reflect.TypeOf(time.Time{}) {
			return field.Addr().Interface()
		}
		return field.Addr().Interface()
	default:
		return field.Addr().Interface()
	}
}

// nullTimeScanner handles scanning NULL values into *time.Time fields
type nullTimeScanner struct {
	field reflect.Value
}

func (nts *nullTimeScanner) Scan(value interface{}) error {
	if value == nil {
		nts.field.Set(reflect.Zero(nts.field.Type()))
		return nil
	}
	
	// Create a new time.Time value and scan into it
	timeVal := time.Time{}
	err := convertAssign(&timeVal, value)
	if err != nil {
		return err
	}
	
	// Set the field to point to the scanned time
	timePtr := &timeVal
	nts.field.Set(reflect.ValueOf(timePtr))
	return nil
}

// convertAssign is a simplified version of driver.DefaultParameterConverter.ConvertValue
func convertAssign(dest interface{}, src interface{}) error {
	switch d := dest.(type) {
	case *time.Time:
		switch s := src.(type) {
		case time.Time:
			*d = s
			return nil
		case string:
			parsed, err := time.Parse("2006-01-02 15:04:05", s)
			if err != nil {
				return err
			}
			*d = parsed
			return nil
		default:
			return fmt.Errorf("cannot convert %T to time.Time", src)
		}
	default:
		return fmt.Errorf("unsupported destination type %T", dest)
	}
}

// getColumnsFromStruct extracts column names from struct tags for sql.Row scanning
func (qb *QueryBuilder) getColumnsFromStruct(structType reflect.Type) []string {
	var columns []string
	qb.getColumnsFromStructRecursive(structType, &columns)
	return columns
}

// getColumnsFromStructRecursive recursively extracts columns from struct, handling embedded structs
func (qb *QueryBuilder) getColumnsFromStructRecursive(structType reflect.Type, columns *[]string) {
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		
		// Handle embedded structs
		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			qb.getColumnsFromStructRecursive(field.Type, columns)
			continue
		}
		
		dbTag := field.Tag.Get("db")
		if dbTag != "" && dbTag != "-" {
			*columns = append(*columns, dbTag)
		}
	}
}

type BaseModel struct {
	ID        uint       `db:"id" json:"id"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
	DeletedAt *time.Time `db:"deleted_at" json:"deleted_at,omitempty"`
	
	// Internal state for tracking changes
	original    map[string]interface{} // Original field values
	dirty       map[string]interface{} // Changed field values
	exists      bool                   // Whether record exists in database
}

// GetModelName returns the model name for event dispatching
func (bm *BaseModel) GetModelName() string {
	// This will be overridden by embedding models
	return "BaseModel"
}

// GetEventContext returns the event context for this model
func (bm *BaseModel) GetEventContext() *ModelEventContext {
	return &ModelEventContext{
		Model:     bm,
		ModelName: bm.GetModelName(),
		Fields:    bm.dirty,
		Original:  bm.original,
	}
}

// InitializeModel initializes the model for change tracking
func (bm *BaseModel) InitializeModel() {
	if bm.original == nil {
		bm.original = make(map[string]interface{})
	}
	if bm.dirty == nil {
		bm.dirty = make(map[string]interface{})
	}
}

// MarkAsExisting marks the model as existing in the database
func (bm *BaseModel) MarkAsExisting() {
	bm.exists = true
	bm.InitializeModel()
	// Store original values
	bm.syncOriginal()
}

// MarkAsDirty marks a field as dirty (changed)
func (bm *BaseModel) MarkAsDirty(field string, value interface{}) {
	bm.InitializeModel()
	bm.dirty[field] = value
}

// IsDirty checks if the model has any dirty fields
func (bm *BaseModel) IsDirty() bool {
	return len(bm.dirty) > 0
}

// GetDirtyFields returns the dirty fields
func (bm *BaseModel) GetDirtyFields() map[string]interface{} {
	if bm.dirty == nil {
		return make(map[string]interface{})
	}
	return bm.dirty
}

// syncOriginal syncs the current state to original
func (bm *BaseModel) syncOriginal() {
	if bm.original == nil {
		bm.original = make(map[string]interface{})
	}
	
	// Use reflection to get current field values
	v := reflect.ValueOf(bm).Elem()
	t := v.Type()
	
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		if dbTag := field.Tag.Get("db"); dbTag != "" && dbTag != "-" {
			bm.original[dbTag] = v.Field(i).Interface()
		}
	}
	
	// Clear dirty fields
	bm.dirty = make(map[string]interface{})
}

// Exists returns whether the model exists in the database
func (bm *BaseModel) Exists() bool {
	return bm.exists
}

// SetID sets the model ID and marks as existing
func (bm *BaseModel) SetID(id uint) {
	bm.ID = id
	if id > 0 {
		bm.MarkAsExisting()
	}
}

// IsSoftDeleted checks if the model has been soft deleted
func (bm *BaseModel) IsSoftDeleted() bool {
	return bm.DeletedAt != nil
}

// SoftDelete performs a soft delete by setting DeletedAt timestamp
func (bm *BaseModel) SoftDelete() {
	now := time.Now()
	bm.DeletedAt = &now
	bm.MarkAsDirty("deleted_at", now)
}

// Restore restores a soft-deleted model by clearing DeletedAt
func (bm *BaseModel) Restore() {
	bm.DeletedAt = nil
	bm.MarkAsDirty("deleted_at", nil)
}

// WithTrashed includes soft-deleted records in query results
func (qb *QueryBuilder) WithTrashed() *QueryBuilder {
	qb.includeDeleted = true
	return qb
}

// OnlyTrashed returns only soft-deleted records
func (qb *QueryBuilder) OnlyTrashed() *QueryBuilder {
	qb.includeDeleted = true
	qb.WhereNotNull("deleted_at")
	return qb
}

// WhereNotNull adds a WHERE column IS NOT NULL condition
func (qb *QueryBuilder) WhereNotNull(column string) *QueryBuilder {
	qb.wheres = append(qb.wheres, whereClause{
		column:   column,
		operator: "IS NOT NULL",
		value:    nil,
		boolean:  "AND",
	})
	return qb
}

// shouldApplySoftDeleteFilter determines if soft delete filter should be applied
func (qb *QueryBuilder) shouldApplySoftDeleteFilter() bool {
	// Only apply if not including deleted records and table has deleted_at column
	// For now, we'll assume BaseModel tables have deleted_at column
	return !qb.includeDeleted
}

// applySoftDeleteFilter adds the deleted_at IS NULL condition
func (qb *QueryBuilder) applySoftDeleteFilter() {
	if qb.shouldApplySoftDeleteFilter() {
		// Check if we already have a deleted_at condition to avoid duplicates
		hasDeletedAtCondition := false
		for _, where := range qb.wheres {
			if where.column == "deleted_at" {
				hasDeletedAtCondition = true
				break
			}
		}
		
		if !hasDeletedAtCondition {
			qb.wheres = append(qb.wheres, whereClause{
				column:   "deleted_at",
				operator: "IS NULL",
				value:    nil,
				boolean:  "AND",
			})
		}
	}
}

// Additional QueryBuilder methods for relationships

// selectRaw adds a raw select clause
func (qb *QueryBuilder) selectRaw(expression string) *QueryBuilder {
	qb.selects = append(qb.selects, expression)
	return qb
}

// whereRaw adds a raw where clause
func (qb *QueryBuilder) whereRaw(expression string) *QueryBuilder {
	qb.wheres = append(qb.wheres, whereClause{
		column:   expression,
		operator: "RAW",
		value:    nil,
		boolean:  "AND",
	})
	return qb
}

// toSQL converts the query to SQL string
func (qb *QueryBuilder) toSQL() string {
	// Apply soft delete filter before building SQL
	qb.applySoftDeleteFilter()
	
	var query strings.Builder
	
	// SELECT clause
	query.WriteString("SELECT ")
	if len(qb.selects) > 0 {
		query.WriteString(strings.Join(qb.selects, ", "))
	} else {
		query.WriteString("*")
	}
	
	// FROM clause
	if qb.table != "" {
		query.WriteString(" FROM ")
		query.WriteString(qb.table)
	}
	
	// JOIN clauses
	for _, join := range qb.joins {
		query.WriteString(" ")
		query.WriteString(join)
	}
	
	// WHERE clauses
	if len(qb.wheres) > 0 {
		query.WriteString(" WHERE ")
		for i, where := range qb.wheres {
			if i > 0 {
				query.WriteString(" ")
				query.WriteString(where.boolean)
				query.WriteString(" ")
			}
			
			if where.operator == "RAW" {
				query.WriteString(where.column)
			} else if where.operator == "IS NULL" || where.operator == "IS NOT NULL" {
				query.WriteString(where.column)
				query.WriteString(" ")
				query.WriteString(where.operator)
			} else {
				query.WriteString(where.column)
				query.WriteString(" ")
				query.WriteString(where.operator)
				query.WriteString(" ?")
			}
		}
	}
	
	// GROUP BY clause
	if len(qb.groupBy) > 0 {
		query.WriteString(" GROUP BY ")
		query.WriteString(strings.Join(qb.groupBy, ", "))
	}
	
	// HAVING clauses
	if len(qb.having) > 0 {
		query.WriteString(" HAVING ")
		for i, having := range qb.having {
			if i > 0 {
				query.WriteString(" ")
				query.WriteString(having.boolean)
				query.WriteString(" ")
			}
			query.WriteString(having.column)
			query.WriteString(" ")
			query.WriteString(having.operator)
			query.WriteString(" ?")
		}
	}
	
	// ORDER BY clause
	if len(qb.orders) > 0 {
		query.WriteString(" ORDER BY ")
		query.WriteString(strings.Join(qb.orders, ", "))
	}
	
	// LIMIT clause
	if qb.limit > 0 {
		query.WriteString(fmt.Sprintf(" LIMIT %d", qb.limit))
	}
	
	// OFFSET clause
	if qb.offset > 0 {
		query.WriteString(fmt.Sprintf(" OFFSET %d", qb.offset))
	}
	
	return query.String()
}

// WhereIn adds a WHERE IN clause
func (qb *QueryBuilder) WhereIn(column string, values []interface{}) *QueryBuilder {
	if len(values) == 0 {
		return qb
	}
	
	placeholders := make([]string, len(values))
	for i := range values {
		placeholders[i] = "?"
		qb.bindings = append(qb.bindings, values[i])
	}
	
	qb.wheres = append(qb.wheres, whereClause{
		column:   column,
		operator: "IN",
		value:    fmt.Sprintf("(%s)", strings.Join(placeholders, ", ")),
		boolean:  "AND",
	})
	
	return qb
}

// Table sets the table for the query
func (qb *QueryBuilder) Table(table string) *QueryBuilder {
	qb.table = table
	return qb
}

// Model CRUD Operations with Event Integration

// ModelSaver interface for models that can be saved
type ModelSaver interface {
	EventableModel
	Save(ctx context.Context, db *DB) error
}

// ModelCreator interface for models that can be created
type ModelCreator interface {
	EventableModel
	Create(ctx context.Context, db *DB) error
}

// ModelUpdater interface for models that can be updated
type ModelUpdater interface {
	EventableModel
	Update(ctx context.Context, db *DB) error
}

// ModelDeleter interface for models that can be deleted
type ModelDeleter interface {
	EventableModel
	Delete(ctx context.Context, db *DB) error
}

// SaveModel saves a model to the database with event handling
func SaveModel(ctx context.Context, db *DB, model EventableModel) error {
	// Check if model has BaseModel embedded to check existence
	if baseModel := getBaseModel(model); baseModel != nil && baseModel.Exists() {
		return UpdateModel(ctx, db, model)
	}
	return CreateModel(ctx, db, model)
}

// getBaseModel extracts BaseModel from an EventableModel using reflection
func getBaseModel(model EventableModel) *BaseModel {
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	
	// Look for BaseModel field
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if field.Type() == reflect.TypeOf(BaseModel{}) {
			if field.CanAddr() {
				return field.Addr().Interface().(*BaseModel)
			}
		}
	}
	return nil
}

// CreateModel creates a new model in the database with event handling
func CreateModel(ctx context.Context, db *DB, model EventableModel) error {
	dispatcher := GetModelEventDispatcher()
	
	// Initialize model if needed
	if baseModel := getBaseModel(model); baseModel != nil {
		baseModel.InitializeModel()
		baseModel.CreatedAt = time.Now()
		baseModel.UpdatedAt = time.Now()
	}
	
	// Dispatch saving event
	if err := dispatcher.DispatchEvent(ctx, EventSaving, model); err != nil {
		return &ModelEventError{Event: EventSaving, ModelName: model.GetModelName(), Err: err}
	}
	
	// Dispatch creating event
	if err := dispatcher.DispatchEvent(ctx, EventCreating, model); err != nil {
		return &ModelEventError{Event: EventCreating, ModelName: model.GetModelName(), Err: err}
	}
	
	// Perform the actual database insert
	tableName := model.TableName()
	fields, values := extractModelFields(model)
	
	// Build INSERT query
	placeholders := make([]string, len(values))
	for i := range placeholders {
		placeholders[i] = "?"
	}
	
	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		tableName,
		strings.Join(fields, ", "),
		strings.Join(placeholders, ", "),
	)
	
	// Execute the insert
	result, err := db.Exec(query, values...)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", model.GetModelName(), err)
	}
	
	// Get the inserted ID
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get inserted ID for %s: %w", model.GetModelName(), err)
	}
	
	// Update model with new ID
	if baseModel := getBaseModel(model); baseModel != nil {
		baseModel.SetID(uint(id))
		baseModel.MarkAsExisting()
	}
	
	// Dispatch created event
	if err := dispatcher.DispatchEvent(ctx, EventCreated, model); err != nil {
		return &ModelEventError{Event: EventCreated, ModelName: model.GetModelName(), Err: err}
	}
	
	// Dispatch saved event
	if err := dispatcher.DispatchEvent(ctx, EventSaved, model); err != nil {
		return &ModelEventError{Event: EventSaved, ModelName: model.GetModelName(), Err: err}
	}
	
	return nil
}

// UpdateModel updates an existing model in the database with event handling
func UpdateModel(ctx context.Context, db *DB, model EventableModel) error {
	dispatcher := GetModelEventDispatcher()
	
	// Get base model for dirty field tracking
	baseModel := getBaseModel(model)
	if baseModel == nil {
		return fmt.Errorf("model must embed BaseModel for update operations")
	}
	
	// Check if there are changes to save
	if !baseModel.IsDirty() {
		return nil // No changes to save
	}
	
	// Update timestamp
	baseModel.UpdatedAt = time.Now()
	baseModel.MarkAsDirty("updated_at", baseModel.UpdatedAt)
	
	// Dispatch saving event
	if err := dispatcher.DispatchEvent(ctx, EventSaving, model); err != nil {
		return &ModelEventError{Event: EventSaving, ModelName: model.GetModelName(), Err: err}
	}
	
	// Dispatch updating event
	if err := dispatcher.DispatchEvent(ctx, EventUpdating, model); err != nil {
		return &ModelEventError{Event: EventUpdating, ModelName: model.GetModelName(), Err: err}
	}
	
	// Build UPDATE query for dirty fields only
	dirtyFields := baseModel.GetDirtyFields()
	fields := make([]string, 0, len(dirtyFields))
	values := make([]interface{}, 0, len(dirtyFields))
	
	for field, value := range dirtyFields {
		fields = append(fields, fmt.Sprintf("%s = ?", field))
		values = append(values, value)
	}
	
	// Add ID to WHERE clause
	values = append(values, baseModel.ID)
	
	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE id = ?",
		model.TableName(),
		strings.Join(fields, ", "),
	)
	
	// Execute the update
	result, err := db.Exec(query, values...)
	if err != nil {
		return fmt.Errorf("failed to update %s: %w", model.GetModelName(), err)
	}
	
	// Check if any rows were affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected for %s update: %w", model.GetModelName(), err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("no rows affected when updating %s with ID %d", model.GetModelName(), baseModel.ID)
	}
	
	// Sync the changes as original values
	baseModel.syncOriginal()
	
	// Dispatch updated event
	if err := dispatcher.DispatchEvent(ctx, EventUpdated, model); err != nil {
		return &ModelEventError{Event: EventUpdated, ModelName: model.GetModelName(), Err: err}
	}
	
	// Dispatch saved event
	if err := dispatcher.DispatchEvent(ctx, EventSaved, model); err != nil {
		return &ModelEventError{Event: EventSaved, ModelName: model.GetModelName(), Err: err}
	}
	
	return nil
}

// DeleteModel deletes a model from the database with event handling
func DeleteModel(ctx context.Context, db *DB, model EventableModel) error {
	dispatcher := GetModelEventDispatcher()
	
	// Get base model for ID
	baseModel := getBaseModel(model)
	if baseModel == nil {
		return fmt.Errorf("model must embed BaseModel for delete operations")
	}
	
	if baseModel.ID == 0 {
		return fmt.Errorf("cannot delete %s: no ID specified", model.GetModelName())
	}
	
	// Dispatch deleting event
	if err := dispatcher.DispatchEvent(ctx, EventDeleting, model); err != nil {
		return &ModelEventError{Event: EventDeleting, ModelName: model.GetModelName(), Err: err}
	}
	
	// Perform soft delete by updating deleted_at
	now := time.Now()
	baseModel.DeletedAt = &now
	baseModel.UpdatedAt = now
	baseModel.MarkAsDirty("deleted_at", now)
	baseModel.MarkAsDirty("updated_at", now)
	
	// Execute the soft delete
	query := fmt.Sprintf("UPDATE %s SET deleted_at = ?, updated_at = ? WHERE id = ?", model.TableName())
	result, err := db.Exec(query, now, now, baseModel.ID)
	if err != nil {
		return fmt.Errorf("failed to delete %s: %w", model.GetModelName(), err)
	}
	
	// Check if any rows were affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected for %s delete: %w", model.GetModelName(), err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("no rows affected when deleting %s with ID %d", model.GetModelName(), baseModel.ID)
	}
	
	// Mark as not existing
	baseModel.exists = false
	
	// Dispatch deleted event
	if err := dispatcher.DispatchEvent(ctx, EventDeleted, model); err != nil {
		return &ModelEventError{Event: EventDeleted, ModelName: model.GetModelName(), Err: err}
	}
	
	return nil
}

// ForceDeleteModel permanently deletes a model from the database with event handling
func ForceDeleteModel(ctx context.Context, db *DB, model EventableModel) error {
	dispatcher := GetModelEventDispatcher()
	
	// Get base model for ID
	baseModel := getBaseModel(model)
	if baseModel == nil {
		return fmt.Errorf("model must embed BaseModel for delete operations")
	}
	
	if baseModel.ID == 0 {
		return fmt.Errorf("cannot force delete %s: no ID specified", model.GetModelName())
	}
	
	// Dispatch deleting event
	if err := dispatcher.DispatchEvent(ctx, EventDeleting, model); err != nil {
		return &ModelEventError{Event: EventDeleting, ModelName: model.GetModelName(), Err: err}
	}
	
	// Execute the hard delete
	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", model.TableName())
	result, err := db.Exec(query, baseModel.ID)
	if err != nil {
		return fmt.Errorf("failed to force delete %s: %w", model.GetModelName(), err)
	}
	
	// Check if any rows were affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected for %s force delete: %w", model.GetModelName(), err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("no rows affected when force deleting %s with ID %d", model.GetModelName(), baseModel.ID)
	}
	
	// Mark as not existing
	baseModel.exists = false
	
	// Dispatch deleted event
	if err := dispatcher.DispatchEvent(ctx, EventDeleted, model); err != nil {
		return &ModelEventError{Event: EventDeleted, ModelName: model.GetModelName(), Err: err}
	}
	
	return nil
}

// RestoreModel restores a soft-deleted model
func RestoreModel(ctx context.Context, db *DB, model EventableModel) error {
	// Get base model for ID
	baseModel := getBaseModel(model)
	if baseModel == nil {
		return fmt.Errorf("model must embed BaseModel for restore operations")
	}
	
	if baseModel.ID == 0 {
		return fmt.Errorf("cannot restore %s: no ID specified", model.GetModelName())
	}
	
	// Restore by clearing deleted_at
	now := time.Now()
	baseModel.DeletedAt = nil
	baseModel.UpdatedAt = now
	baseModel.MarkAsDirty("deleted_at", nil)
	baseModel.MarkAsDirty("updated_at", now)
	
	// Execute the restore
	query := fmt.Sprintf("UPDATE %s SET deleted_at = NULL, updated_at = ? WHERE id = ?", model.TableName())
	result, err := db.Exec(query, now, baseModel.ID)
	if err != nil {
		return fmt.Errorf("failed to restore %s: %w", model.GetModelName(), err)
	}
	
	// Check if any rows were affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected for %s restore: %w", model.GetModelName(), err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("no rows affected when restoring %s with ID %d", model.GetModelName(), baseModel.ID)
	}
	
	// Mark as existing
	baseModel.exists = true
	
	return nil
}

// extractModelFields extracts field names and values from a model for database operations
func extractModelFields(model interface{}) ([]string, []interface{}) {
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	
	fields := make([]string, 0)
	values := make([]interface{}, 0)
	
	extractFromStruct(v, &fields, &values)
	
	return fields, values
}

// extractFromStruct recursively extracts fields from struct, handling embedded structs
func extractFromStruct(v reflect.Value, fields *[]string, values *[]interface{}) {
	t := v.Type()
	
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)
		
		// Skip unexported fields
		if !fieldValue.CanInterface() {
			continue
		}
		
		// Handle embedded structs
		if field.Anonymous {
			if fieldValue.Kind() == reflect.Struct {
				extractFromStruct(fieldValue, fields, values)
			}
			continue
		}
		
		// Get database field name from tag
		dbTag := field.Tag.Get("db")
		if dbTag == "" || dbTag == "-" {
			continue
		}
		
		// Skip ID field for inserts (auto-increment)
		if dbTag == "id" {
			continue
		}
		
		// Skip internal BaseModel fields that aren't database columns
		if dbTag == "original" || dbTag == "dirty" || dbTag == "exists" {
			continue
		}
		
		*fields = append(*fields, dbTag)
		*values = append(*values, fieldValue.Interface())
	}
}