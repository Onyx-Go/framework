package database

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// DB represents a database connection with driver information
type DB struct {
	*sql.DB
	driver string
}

// NewDB creates a new database connection
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

// Driver returns the database driver name
func (db *DB) Driver() string {
	return db.driver
}

// Table creates a new query builder for the specified table
func (db *DB) Table(tableName string) QueryBuilder {
	return &queryBuilder{
		db:              db,
		table:           tableName,
		selects:         []string{},
		wheres:          []whereClause{},
		orders:          []string{},
		joins:           []string{},
		groupBy:         []string{},
		having:          []whereClause{},
		bindings:        []interface{}{},
		eagerLoad:       make(map[string]interface{}),
		includeDeleted:  false,
	}
}

// Model creates a new query builder for the specified model
func (db *DB) Model(model Model) QueryBuilder {
	return db.Table(model.TableName())
}

// Raw creates a query builder with raw SQL
func (db *DB) Raw(query string, args ...interface{}) QueryBuilder {
	qb := db.Table("")
	if qbImpl, ok := qb.(*queryBuilder); ok {
		qbImpl.rawQuery = query
		qbImpl.bindings = args
	}
	return qb
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.DB.Close()
}

// Ping verifies the database connection is still alive
func (db *DB) Ping() error {
	return db.DB.Ping()
}

// Exec executes a query that doesn't return rows
func (db *DB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return db.DB.Exec(query, args...)
}

// Query executes a query that returns rows
func (db *DB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return db.DB.Query(query, args...)
}

// QueryRow executes a query that is expected to return at most one row
func (db *DB) QueryRow(query string, args ...interface{}) *sql.Row {
	return db.DB.QueryRow(query, args...)
}

// Begin starts a transaction
func (db *DB) Begin() (*sql.Tx, error) {
	return db.DB.Begin()
}

// Ensure DB implements Database interface
var _ Database = (*DB)(nil)