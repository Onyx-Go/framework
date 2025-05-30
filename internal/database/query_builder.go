package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// queryBuilder implements the QueryBuilder interface
type queryBuilder struct {
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
	eagerLoadEngine interface{} // Will be properly typed when we refactor eager loading
	includeDeleted  bool
	rawQuery        string
}

// NewQueryBuilder creates a new query builder instance
func NewQueryBuilder(db *DB) QueryBuilder {
	return &queryBuilder{
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

// Select specifies the columns to select
func (qb *queryBuilder) Select(columns ...string) QueryBuilder {
	qb.selects = columns
	return qb
}

// Table sets the table for the query
func (qb *queryBuilder) Table(tableName string) QueryBuilder {
	qb.table = tableName
	return qb
}

// Where adds a WHERE clause
func (qb *queryBuilder) Where(column, operator string, value interface{}) QueryBuilder {
	qb.wheres = append(qb.wheres, whereClause{
		Column:   column,
		Operator: operator,
		Value:    value,
		Boolean:  "AND",
	})
	return qb
}

// WhereIn adds a WHERE IN clause
func (qb *queryBuilder) WhereIn(column string, values []interface{}) QueryBuilder {
	return qb.Where(column, "IN", values)
}

// WhereNotIn adds a WHERE NOT IN clause
func (qb *queryBuilder) WhereNotIn(column string, values []interface{}) QueryBuilder {
	return qb.Where(column, "NOT IN", values)
}

// WhereNull adds a WHERE IS NULL clause
func (qb *queryBuilder) WhereNull(column string) QueryBuilder {
	return qb.Where(column, "IS", nil)
}

// WhereNotNull adds a WHERE IS NOT NULL clause
func (qb *queryBuilder) WhereNotNull(column string) QueryBuilder {
	return qb.Where(column, "IS NOT", nil)
}

// WhereBetween adds a WHERE BETWEEN clause
func (qb *queryBuilder) WhereBetween(column string, start, end interface{}) QueryBuilder {
	return qb.Where(column, "BETWEEN", []interface{}{start, end})
}

// WhereNotBetween adds a WHERE NOT BETWEEN clause
func (qb *queryBuilder) WhereNotBetween(column string, start, end interface{}) QueryBuilder {
	return qb.Where(column, "NOT BETWEEN", []interface{}{start, end})
}

// WhereLike adds a WHERE LIKE clause
func (qb *queryBuilder) WhereLike(column string, value string) QueryBuilder {
	return qb.Where(column, "LIKE", value)
}

// WhereNotLike adds a WHERE NOT LIKE clause
func (qb *queryBuilder) WhereNotLike(column string, value string) QueryBuilder {
	return qb.Where(column, "NOT LIKE", value)
}

// OrWhere adds an OR WHERE clause
func (qb *queryBuilder) OrWhere(column, operator string, value interface{}) QueryBuilder {
	qb.wheres = append(qb.wheres, whereClause{
		Column:   column,
		Operator: operator,
		Value:    value,
		Boolean:  "OR",
	})
	return qb
}

// OrWhereIn adds an OR WHERE IN clause
func (qb *queryBuilder) OrWhereIn(column string, values []interface{}) QueryBuilder {
	return qb.OrWhere(column, "IN", values)
}

// OrWhereNotIn adds an OR WHERE NOT IN clause
func (qb *queryBuilder) OrWhereNotIn(column string, values []interface{}) QueryBuilder {
	return qb.OrWhere(column, "NOT IN", values)
}

// OrWhereNull adds an OR WHERE IS NULL clause
func (qb *queryBuilder) OrWhereNull(column string) QueryBuilder {
	return qb.OrWhere(column, "IS", nil)
}

// OrWhereNotNull adds an OR WHERE IS NOT NULL clause
func (qb *queryBuilder) OrWhereNotNull(column string) QueryBuilder {
	return qb.OrWhere(column, "IS NOT", nil)
}

// Join adds an INNER JOIN clause
func (qb *queryBuilder) Join(table, first, operator, second string) QueryBuilder {
	qb.joins = append(qb.joins, fmt.Sprintf("JOIN %s ON %s %s %s", table, first, operator, second))
	return qb
}

// LeftJoin adds a LEFT JOIN clause
func (qb *queryBuilder) LeftJoin(table, first, operator, second string) QueryBuilder {
	qb.joins = append(qb.joins, fmt.Sprintf("LEFT JOIN %s ON %s %s %s", table, first, operator, second))
	return qb
}

// RightJoin adds a RIGHT JOIN clause
func (qb *queryBuilder) RightJoin(table, first, operator, second string) QueryBuilder {
	qb.joins = append(qb.joins, fmt.Sprintf("RIGHT JOIN %s ON %s %s %s", table, first, operator, second))
	return qb
}

// InnerJoin adds an INNER JOIN clause
func (qb *queryBuilder) InnerJoin(table, first, operator, second string) QueryBuilder {
	return qb.Join(table, first, operator, second)
}

// OrderBy adds an ORDER BY clause
func (qb *queryBuilder) OrderBy(column string, direction ...string) QueryBuilder {
	dir := "ASC"
	if len(direction) > 0 && strings.ToUpper(direction[0]) == "DESC" {
		dir = "DESC"
	}
	qb.orders = append(qb.orders, fmt.Sprintf("%s %s", column, dir))
	return qb
}

// GroupBy adds a GROUP BY clause
func (qb *queryBuilder) GroupBy(columns ...string) QueryBuilder {
	qb.groupBy = append(qb.groupBy, columns...)
	return qb
}

// Having adds a HAVING clause
func (qb *queryBuilder) Having(column, operator string, value interface{}) QueryBuilder {
	qb.having = append(qb.having, whereClause{
		Column:   column,
		Operator: operator,
		Value:    value,
		Boolean:  "AND",
	})
	return qb
}

// Limit sets the LIMIT clause
func (qb *queryBuilder) Limit(limit int) QueryBuilder {
	qb.limit = limit
	return qb
}

// Offset sets the OFFSET clause
func (qb *queryBuilder) Offset(offset int) QueryBuilder {
	qb.offset = offset
	return qb
}

// With adds eager loading relationships
func (qb *queryBuilder) With(relations ...string) QueryBuilder {
	for _, relation := range relations {
		qb.eagerLoad[relation] = nil
	}
	return qb
}

// WithCount adds eager loading relationship counts
func (qb *queryBuilder) WithCount(relations ...string) QueryBuilder {
	for _, relation := range relations {
		qb.eagerLoad[relation+"_count"] = "count"
	}
	return qb
}

// WithTrashed includes soft-deleted records
func (qb *queryBuilder) WithTrashed() QueryBuilder {
	qb.includeDeleted = true
	return qb
}

// OnlyTrashed only returns soft-deleted records
func (qb *queryBuilder) OnlyTrashed() QueryBuilder {
	qb.includeDeleted = true
	return qb.WhereNotNull("deleted_at")
}

// Get executes the query and returns all results
func (qb *queryBuilder) Get(dest interface{}) error {
	query, args, err := qb.buildSelectQuery()
	if err != nil {
		return err
	}

	rows, err := qb.db.Query(query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	// Use scanner to populate dest
	scanner := NewScanner()
	return scanner.ScanRows(rows, dest)
}

// First executes the query and returns the first result
func (qb *queryBuilder) First(dest interface{}) error {
	qb.limit = 1
	query, args, err := qb.buildSelectQuery()
	if err != nil {
		return err
	}

	row := qb.db.QueryRow(query, args...)
	
	// Use scanner to populate dest
	scanner := NewScanner()
	return scanner.ScanRow(row, dest)
}

// Find finds a record by ID
func (qb *queryBuilder) Find(dest interface{}, id interface{}) error {
	return qb.Where("id", "=", id).First(dest)
}

// Exists checks if any records exist
func (qb *queryBuilder) Exists() (bool, error) {
	count, err := qb.Count()
	return count > 0, err
}

// Count returns the count of records
func (qb *queryBuilder) Count() (int64, error) {
	originalSelects := qb.selects
	qb.selects = []string{"COUNT(*) as count"}
	
	query, args, err := qb.buildSelectQuery()
	if err != nil {
		qb.selects = originalSelects
		return 0, err
	}

	var count int64
	err = qb.db.QueryRow(query, args...).Scan(&count)
	qb.selects = originalSelects
	return count, err
}

// Sum returns the sum of a column
func (qb *queryBuilder) Sum(column string) (float64, error) {
	originalSelects := qb.selects
	qb.selects = []string{fmt.Sprintf("SUM(%s) as sum", column)}
	
	query, args, err := qb.buildSelectQuery()
	if err != nil {
		qb.selects = originalSelects
		return 0, err
	}

	var sum sql.NullFloat64
	err = qb.db.QueryRow(query, args...).Scan(&sum)
	qb.selects = originalSelects
	
	if !sum.Valid {
		return 0, nil
	}
	return sum.Float64, err
}

// Avg returns the average of a column
func (qb *queryBuilder) Avg(column string) (float64, error) {
	originalSelects := qb.selects
	qb.selects = []string{fmt.Sprintf("AVG(%s) as avg", column)}
	
	query, args, err := qb.buildSelectQuery()
	if err != nil {
		qb.selects = originalSelects
		return 0, err
	}

	var avg sql.NullFloat64
	err = qb.db.QueryRow(query, args...).Scan(&avg)
	qb.selects = originalSelects
	
	if !avg.Valid {
		return 0, nil
	}
	return avg.Float64, err
}

// Min returns the minimum value of a column
func (qb *queryBuilder) Min(column string) (interface{}, error) {
	originalSelects := qb.selects
	qb.selects = []string{fmt.Sprintf("MIN(%s) as min", column)}
	
	query, args, err := qb.buildSelectQuery()
	if err != nil {
		qb.selects = originalSelects
		return nil, err
	}

	var min interface{}
	err = qb.db.QueryRow(query, args...).Scan(&min)
	qb.selects = originalSelects
	return min, err
}

// Max returns the maximum value of a column
func (qb *queryBuilder) Max(column string) (interface{}, error) {
	originalSelects := qb.selects
	qb.selects = []string{fmt.Sprintf("MAX(%s) as max", column)}
	
	query, args, err := qb.buildSelectQuery()
	if err != nil {
		qb.selects = originalSelects
		return nil, err
	}

	var max interface{}
	err = qb.db.QueryRow(query, args...).Scan(&max)
	qb.selects = originalSelects
	return max, err
}

// Insert inserts data into the table
func (qb *queryBuilder) Insert(data interface{}) (sql.Result, error) {
	// This will be implemented with reflection to handle both maps and structs
	// For now, let's handle map[string]interface{}
	if dataMap, ok := data.(map[string]interface{}); ok {
		return qb.insertMap(dataMap)
	}
	
	// TODO: Handle struct insertion with reflection
	return nil, fmt.Errorf("unsupported data type for insertion")
}

// Update updates records in the table
func (qb *queryBuilder) Update(data interface{}) (sql.Result, error) {
	// Handle map[string]interface{}
	if dataMap, ok := data.(map[string]interface{}); ok {
		return qb.updateMap(dataMap)
	}
	
	// TODO: Handle struct updates with reflection
	return nil, fmt.Errorf("unsupported data type for update")
}

// Delete performs a soft delete
func (qb *queryBuilder) Delete() (sql.Result, error) {
	now := time.Now()
	updateData := map[string]interface{}{
		"deleted_at": now,
		"updated_at": now,
	}
	return qb.updateMap(updateData)
}

// ForceDelete performs a hard delete
func (qb *queryBuilder) ForceDelete() (sql.Result, error) {
	query := fmt.Sprintf("DELETE FROM %s", qb.table)
	
	if len(qb.wheres) > 0 {
		whereClause, whereArgs := qb.buildWhereClause(qb.wheres)
		query += " WHERE " + whereClause
		return qb.db.Exec(query, whereArgs...)
	}
	
	return qb.db.Exec(query)
}

// Restore restores soft-deleted records
func (qb *queryBuilder) Restore() (sql.Result, error) {
	updateData := map[string]interface{}{
		"deleted_at": nil,
		"updated_at": time.Now(),
	}
	return qb.updateMap(updateData)
}

// ToSQL returns the SQL query and arguments
func (qb *queryBuilder) ToSQL() (string, []interface{}, error) {
	return qb.buildSelectQuery()
}

// Helper methods

func (qb *queryBuilder) insertMap(data map[string]interface{}) (sql.Result, error) {
	columns := make([]string, 0, len(data))
	placeholders := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))
	
	for column, value := range data {
		columns = append(columns, column)
		placeholders = append(placeholders, "?")
		values = append(values, value)
	}
	
	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		qb.table,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)
	
	return qb.db.Exec(query, values...)
}

func (qb *queryBuilder) updateMap(data map[string]interface{}) (sql.Result, error) {
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
	
	return qb.db.Exec(query, values...)
}

func (qb *queryBuilder) buildSelectQuery() (string, []interface{}, error) {
	if qb.rawQuery != "" {
		return qb.rawQuery, qb.bindings, nil
	}

	query := fmt.Sprintf("SELECT %s FROM %s", strings.Join(qb.selects, ", "), qb.table)
	args := []interface{}{}
	
	// Add joins
	if len(qb.joins) > 0 {
		query += " " + strings.Join(qb.joins, " ")
	}
	
	// Add where clauses
	if len(qb.wheres) > 0 {
		whereClause, whereArgs := qb.buildWhereClause(qb.wheres)
		query += " WHERE " + whereClause
		args = append(args, whereArgs...)
	}
	
	// Add soft delete filtering
	if !qb.includeDeleted {
		if len(qb.wheres) > 0 {
			query += " AND deleted_at IS NULL"
		} else {
			query += " WHERE deleted_at IS NULL"
		}
	}
	
	// Add group by
	if len(qb.groupBy) > 0 {
		query += " GROUP BY " + strings.Join(qb.groupBy, ", ")
	}
	
	// Add having
	if len(qb.having) > 0 {
		havingClause, havingArgs := qb.buildWhereClause(qb.having)
		query += " HAVING " + havingClause
		args = append(args, havingArgs...)
	}
	
	// Add order by
	if len(qb.orders) > 0 {
		query += " ORDER BY " + strings.Join(qb.orders, ", ")
	}
	
	// Add limit and offset
	if qb.limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", qb.limit)
	}
	if qb.offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", qb.offset)
	}
	
	return query, args, nil
}

func (qb *queryBuilder) buildWhereClause(wheres []whereClause) (string, []interface{}) {
	if len(wheres) == 0 {
		return "", []interface{}{}
	}
	
	parts := make([]string, len(wheres))
	args := make([]interface{}, 0)
	
	for i, where := range wheres {
		boolean := ""
		if i > 0 {
			boolean = where.Boolean + " "
		}
		
		switch where.Operator {
		case "IN", "NOT IN":
			if values, ok := where.Value.([]interface{}); ok {
				placeholders := make([]string, len(values))
				for j := range values {
					placeholders[j] = "?"
				}
				parts[i] = fmt.Sprintf("%s%s %s (%s)", boolean, where.Column, where.Operator, strings.Join(placeholders, ", "))
				args = append(args, values...)
			}
		case "BETWEEN", "NOT BETWEEN":
			if values, ok := where.Value.([]interface{}); ok && len(values) == 2 {
				parts[i] = fmt.Sprintf("%s%s %s ? AND ?", boolean, where.Column, where.Operator)
				args = append(args, values...)
			}
		case "IS", "IS NOT":
			if where.Value == nil {
				parts[i] = fmt.Sprintf("%s%s %s NULL", boolean, where.Column, where.Operator)
			} else {
				parts[i] = fmt.Sprintf("%s%s %s ?", boolean, where.Column, where.Operator)
				args = append(args, where.Value)
			}
		default:
			parts[i] = fmt.Sprintf("%s%s %s ?", boolean, where.Column, where.Operator)
			args = append(args, where.Value)
		}
	}
	
	return strings.Join(parts, " "), args
}

// Ensure queryBuilder implements QueryBuilder interface
var _ QueryBuilder = (*queryBuilder)(nil)