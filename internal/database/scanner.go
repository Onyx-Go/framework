package database

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"strings"
	"time"
)

// scanner implements the Scanner interface
type scanner struct{}

// NewScanner creates a new database scanner
func NewScanner() Scanner {
	return &scanner{}
}

// ScanRows scans multiple rows into a slice
func (s *scanner) ScanRows(rows *sql.Rows, dest interface{}) error {
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
		
		if err := s.ScanIntoStruct(rows, element.Addr().Interface()); err != nil {
			return err
		}
		
		destValue.Set(reflect.Append(destValue, element))
	}
	
	return rows.Err()
}

// ScanRow scans a single row
func (s *scanner) ScanRow(row *sql.Row, dest interface{}) error {
	return s.scanIntoStruct(row, dest)
}

// ScanIntoStruct scans into a struct from either *sql.Rows or *sql.Row
func (s *scanner) ScanIntoStruct(scanner interface{}, dest interface{}) error {
	return s.scanIntoStruct(scanner, dest)
}

// scanIntoStruct handles the actual scanning logic
func (s *scanner) scanIntoStruct(scanner interface{}, dest interface{}) error {
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr {
		return fmt.Errorf("dest must be a pointer")
	}
	
	destValue = destValue.Elem()
	if destValue.Kind() != reflect.Struct {
		return fmt.Errorf("dest must be a pointer to a struct")
	}
	
	destType := destValue.Type()
	
	switch sc := scanner.(type) {
	case *sql.Rows:
		return s.scanRowsIntoStruct(sc, destValue, destType)
	case *sql.Row:
		return s.scanSingleRowIntoStruct(sc, destValue, destType)
	default:
		return fmt.Errorf("unsupported scanner type")
	}
}

// scanRowsIntoStruct handles scanning from sql.Rows (multiple rows)
func (s *scanner) scanRowsIntoStruct(rows *sql.Rows, destValue reflect.Value, destType reflect.Type) error {
	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	
	scanArgs := make([]interface{}, len(columns))
	
	// Map columns to struct fields using 'db' tags
	for i, column := range columns {
		if field := s.findFieldValueByColumn(destValue, column); field.IsValid() {
			if field.CanSet() {
				scanArgs[i] = s.createScanDestination(field)
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
func (s *scanner) scanSingleRowIntoStruct(row *sql.Row, destValue reflect.Value, destType reflect.Type) error {
	// For sql.Row, we'll scan all available fields based on the struct
	columns := s.getStructColumns(destType)
	scanArgs := make([]interface{}, len(columns))
	
	// Map columns to struct fields
	for i, column := range columns {
		if field := s.findFieldValueByColumn(destValue, column); field.IsValid() {
			if field.CanSet() {
				scanArgs[i] = s.createScanDestination(field)
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

// findFieldValueByColumn finds a struct field by database column name
func (s *scanner) findFieldValueByColumn(structValue reflect.Value, columnName string) reflect.Value {
	structType := structValue.Type()
	
	for i := 0; i < structValue.NumField(); i++ {
		field := structType.Field(i)
		fieldValue := structValue.Field(i)
		
		// Check 'db' tag first
		if dbTag := field.Tag.Get("db"); dbTag != "" {
			if dbTag == columnName || dbTag == "-" {
				if dbTag == "-" {
					return reflect.Value{} // Skip this field
				}
				return fieldValue
			}
		}
		
		// Check 'json' tag as fallback
		if jsonTag := field.Tag.Get("json"); jsonTag != "" {
			tagName := strings.Split(jsonTag, ",")[0]
			if tagName == columnName {
				return fieldValue
			}
		}
		
		// Check field name (snake_case conversion)
		if s.toSnakeCase(field.Name) == columnName {
			return fieldValue
		}
		
		// Check exact field name match
		if strings.ToLower(field.Name) == strings.ToLower(columnName) {
			return fieldValue
		}
	}
	
	return reflect.Value{}
}

// createScanDestination creates a scan destination for a field
func (s *scanner) createScanDestination(fieldValue reflect.Value) interface{} {
	fieldType := fieldValue.Type()
	
	// Handle pointers
	if fieldType.Kind() == reflect.Ptr {
		// Create a new instance for pointer fields
		newValue := reflect.New(fieldType.Elem())
		fieldValue.Set(newValue)
		return newValue.Interface()
	}
	
	// Handle sql.NullTime for time fields
	if fieldType == reflect.TypeOf(time.Time{}) {
		nullTime := &sql.NullTime{}
		// We'll need to handle the conversion after scanning
		return &nullTimeScanner{
			NullTime: nullTime,
			target:   fieldValue.Addr().Interface().(*time.Time),
		}
	}
	
	// Handle sql.NullString
	if fieldType.Kind() == reflect.String {
		nullString := &sql.NullString{}
		return &nullStringScanner{
			NullString: nullString,
			target:     fieldValue.Addr().Interface().(*string),
		}
	}
	
	// Handle sql.NullInt64
	if fieldType.Kind() == reflect.Int || fieldType.Kind() == reflect.Int64 {
		nullInt := &sql.NullInt64{}
		return &nullIntScanner{
			NullInt64: nullInt,
			target:    fieldValue.Addr().Interface(),
		}
	}
	
	// Handle sql.NullFloat64
	if fieldType.Kind() == reflect.Float64 {
		nullFloat := &sql.NullFloat64{}
		return &nullFloatScanner{
			NullFloat64: nullFloat,
			target:      fieldValue.Addr().Interface().(*float64),
		}
	}
	
	// Handle sql.NullBool
	if fieldType.Kind() == reflect.Bool {
		nullBool := &sql.NullBool{}
		return &nullBoolScanner{
			NullBool: nullBool,
			target:   fieldValue.Addr().Interface().(*bool),
		}
	}
	
	// Default: return pointer to the field
	return fieldValue.Addr().Interface()
}

// getStructColumns extracts column names from struct tags and field names
func (s *scanner) getStructColumns(structType reflect.Type) []string {
	columns := make([]string, 0)
	
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		
		// Skip unexported fields
		if !field.IsExported() {
			continue
		}
		
		// Check 'db' tag
		if dbTag := field.Tag.Get("db"); dbTag != "" {
			if dbTag != "-" {
				columns = append(columns, dbTag)
			}
			continue
		}
		
		// Check 'json' tag as fallback
		if jsonTag := field.Tag.Get("json"); jsonTag != "" {
			tagName := strings.Split(jsonTag, ",")[0]
			if tagName != "-" {
				columns = append(columns, tagName)
			}
			continue
		}
		
		// Use snake_case field name
		columns = append(columns, s.toSnakeCase(field.Name))
	}
	
	return columns
}

// toSnakeCase converts CamelCase to snake_case
func (s *scanner) toSnakeCase(name string) string {
	var result strings.Builder
	for i, r := range name {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// Null scanners for handling NULL values

type nullTimeScanner struct {
	*sql.NullTime
	target *time.Time
}

func (n *nullTimeScanner) Scan(value interface{}) error {
	err := n.NullTime.Scan(value)
	if err != nil {
		return err
	}
	if n.NullTime.Valid {
		*n.target = n.NullTime.Time
	}
	return nil
}

type nullStringScanner struct {
	*sql.NullString
	target *string
}

func (n *nullStringScanner) Scan(value interface{}) error {
	err := n.NullString.Scan(value)
	if err != nil {
		return err
	}
	if n.NullString.Valid {
		*n.target = n.NullString.String
	}
	return nil
}

type nullIntScanner struct {
	*sql.NullInt64
	target interface{}
}

func (n *nullIntScanner) Scan(value interface{}) error {
	err := n.NullInt64.Scan(value)
	if err != nil {
		return err
	}
	if n.NullInt64.Valid {
		switch t := n.target.(type) {
		case *int:
			*t = int(n.NullInt64.Int64)
		case *int64:
			*t = n.NullInt64.Int64
		}
	}
	return nil
}

type nullFloatScanner struct {
	*sql.NullFloat64
	target *float64
}

func (n *nullFloatScanner) Scan(value interface{}) error {
	err := n.NullFloat64.Scan(value)
	if err != nil {
		return err
	}
	if n.NullFloat64.Valid {
		*n.target = n.NullFloat64.Float64
	}
	return nil
}

type nullBoolScanner struct {
	*sql.NullBool
	target *bool
}

func (n *nullBoolScanner) Scan(value interface{}) error {
	err := n.NullBool.Scan(value)
	if err != nil {
		return err
	}
	if n.NullBool.Valid {
		*n.target = n.NullBool.Bool
	}
	return nil
}

// Custom time scanner that implements the sql/driver.Valuer interface
type nullTimeScanner2 struct {
	Time  time.Time
	Valid bool
}

func (nt *nullTimeScanner2) Scan(value interface{}) error {
	if value == nil {
		nt.Time, nt.Valid = time.Time{}, false
		return nil
	}
	nt.Valid = true
	
	switch v := value.(type) {
	case time.Time:
		nt.Time = v
	case []byte:
		var err error
		nt.Time, err = time.Parse("2006-01-02 15:04:05", string(v))
		return err
	case string:
		var err error
		nt.Time, err = time.Parse("2006-01-02 15:04:05", v)
		return err
	default:
		return fmt.Errorf("cannot scan %T into nullTimeScanner", value)
	}
	
	return nil
}

func (nt nullTimeScanner2) Value() (driver.Value, error) {
	if !nt.Valid {
		return nil, nil
	}
	return nt.Time, nil
}

// Ensure scanner implements Scanner interface
var _ Scanner = (*scanner)(nil)