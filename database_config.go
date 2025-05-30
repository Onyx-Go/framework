package onyx

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// DatabaseConfig holds database connection configuration
type DatabaseConfig struct {
	Driver          string        `json:"driver"`
	DSN             string        `json:"dsn"`
	MaxOpenConns    int           `json:"max_open_conns"`    // Maximum number of open connections
	MaxIdleConns    int           `json:"max_idle_conns"`    // Maximum number of idle connections
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime"` // Maximum connection lifetime
	ConnMaxIdleTime time.Duration `json:"conn_max_idle_time"` // Maximum idle time for connections
}

// DefaultDatabaseConfig returns sensible defaults for database connections
func DefaultDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		Driver:          "sqlite3",
		DSN:             ":memory:",
		MaxOpenConns:    25,                 // Maximum 25 open connections
		MaxIdleConns:    10,                 // Keep up to 10 idle connections
		ConnMaxLifetime: 30 * time.Minute,   // Close connections after 30 minutes
		ConnMaxIdleTime: 15 * time.Minute,   // Close idle connections after 15 minutes
	}
}

// MySQLConfig returns optimized configuration for MySQL
func MySQLConfig(dsn string) DatabaseConfig {
	return DatabaseConfig{
		Driver:          "mysql",
		DSN:             dsn,
		MaxOpenConns:    50,                 // Higher limit for MySQL
		MaxIdleConns:    20,                 // More idle connections for busy systems
		ConnMaxLifetime: 60 * time.Minute,   // MySQL handles longer connections well
		ConnMaxIdleTime: 30 * time.Minute,   // Keep connections warm longer
	}
}

// PostgreSQLConfig returns optimized configuration for PostgreSQL
func PostgreSQLConfig(dsn string) DatabaseConfig {
	return DatabaseConfig{
		Driver:          "postgres",
		DSN:             dsn,
		MaxOpenConns:    40,                 // PostgreSQL handles many connections well
		MaxIdleConns:    15,                 // Reasonable idle connection pool
		ConnMaxLifetime: 45 * time.Minute,   // Good balance for connection reuse
		ConnMaxIdleTime: 20 * time.Minute,   // Keep connections available
	}
}

// SQLiteConfig returns optimized configuration for SQLite
func SQLiteConfig(dsn string) DatabaseConfig {
	return DatabaseConfig{
		Driver:          "sqlite3",
		DSN:             dsn,
		MaxOpenConns:    1,                  // SQLite works best with single connection
		MaxIdleConns:    1,                  // Keep one connection open
		ConnMaxLifetime: 24 * time.Hour,     // SQLite connections can live long
		ConnMaxIdleTime: 2 * time.Hour,      // Keep connection warm for local usage
	}
}

// NewDBWithConfig creates a new database connection with optimized pooling configuration
func NewDBWithConfig(config DatabaseConfig) (*DB, error) {
	sqlDB, err := sql.Open(config.Driver, config.DSN)
	if err != nil {
		return nil, err
	}
	
	// Configure connection pooling
	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(config.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(config.ConnMaxIdleTime)
	
	// Test the connection
	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return nil, err
	}
	
	return &DB{
		DB:     sqlDB,
		driver: config.Driver,
	}, nil
}

// ConfigureExistingDB applies connection pooling configuration to an existing database
func (db *DB) ConfigurePooling(config DatabaseConfig) {
	if db.DB != nil {
		db.DB.SetMaxOpenConns(config.MaxOpenConns)
		db.DB.SetMaxIdleConns(config.MaxIdleConns)
		db.DB.SetConnMaxLifetime(config.ConnMaxLifetime)
		db.DB.SetConnMaxIdleTime(config.ConnMaxIdleTime)
	}
}

// GetPoolStats returns connection pool statistics
func (db *DB) GetPoolStats() sql.DBStats {
	if db.DB != nil {
		return db.DB.Stats()
	}
	return sql.DBStats{}
}

// ConnectionPoolMetrics provides detailed pool metrics
type ConnectionPoolMetrics struct {
	MaxOpenConnections int           `json:"max_open_connections"`
	OpenConnections    int           `json:"open_connections"`
	InUse              int           `json:"in_use"`
	Idle               int           `json:"idle"`
	WaitCount          int64         `json:"wait_count"`
	WaitDuration       time.Duration `json:"wait_duration"`
	MaxIdleClosed      int64         `json:"max_idle_closed"`
	MaxIdleTimeClosed  int64         `json:"max_idle_time_closed"`
	MaxLifetimeClosed  int64         `json:"max_lifetime_closed"`
}

// GetDetailedPoolMetrics returns comprehensive connection pool metrics
func (db *DB) GetDetailedPoolMetrics() ConnectionPoolMetrics {
	stats := db.GetPoolStats()
	return ConnectionPoolMetrics{
		MaxOpenConnections: stats.MaxOpenConnections,
		OpenConnections:    stats.OpenConnections,
		InUse:              stats.InUse,
		Idle:               stats.Idle,
		WaitCount:          stats.WaitCount,
		WaitDuration:       stats.WaitDuration,
		MaxIdleClosed:      stats.MaxIdleClosed,
		MaxIdleTimeClosed:  stats.MaxIdleTimeClosed,
		MaxLifetimeClosed:  stats.MaxLifetimeClosed,
	}
}

// IsHealthy checks if the database connection pool is healthy
func (db *DB) IsHealthy() bool {
	if db.DB == nil {
		return false
	}
	
	// Test connection
	if err := db.DB.Ping(); err != nil {
		return false
	}
	
	// Check pool stats for issues
	stats := db.GetPoolStats()
	
	// If wait count is high compared to total queries, there might be contention
	if stats.WaitCount > 0 && stats.WaitDuration > 5*time.Second {
		return false
	}
	
	// If we have max connections but many are idle, configuration might be inefficient
	if stats.OpenConnections == stats.MaxOpenConnections && 
	   stats.Idle > stats.MaxOpenConnections/2 {
		// This isn't necessarily unhealthy, but might indicate over-provisioning
	}
	
	return true
}

// OptimizeForWorkload adjusts connection pool settings based on usage patterns
func (db *DB) OptimizeForWorkload(workloadType string) {
	var config DatabaseConfig
	
	switch workloadType {
	case "high_read":
		// Many concurrent reads
		config = DatabaseConfig{
			MaxOpenConns:    100,
			MaxIdleConns:    50,
			ConnMaxLifetime: 2 * time.Hour,
			ConnMaxIdleTime: 1 * time.Hour,
		}
	case "high_write":
		// Many concurrent writes
		config = DatabaseConfig{
			MaxOpenConns:    30,  // Fewer connections to reduce lock contention
			MaxIdleConns:    15,
			ConnMaxLifetime: 30 * time.Minute,
			ConnMaxIdleTime: 10 * time.Minute,
		}
	case "batch_processing":
		// Long-running batch jobs
		config = DatabaseConfig{
			MaxOpenConns:    10,  // Fewer long-lived connections
			MaxIdleConns:    5,
			ConnMaxLifetime: 8 * time.Hour,
			ConnMaxIdleTime: 4 * time.Hour,
		}
	case "low_latency":
		// Low latency requirements
		config = DatabaseConfig{
			MaxOpenConns:    20,
			MaxIdleConns:    20,  // Keep all connections warm
			ConnMaxLifetime: 24 * time.Hour,
			ConnMaxIdleTime: 12 * time.Hour,
		}
	default:
		// Default balanced configuration
		config = DefaultDatabaseConfig()
	}
	
	db.ConfigurePooling(config)
}

// WarmupConnections pre-establishes idle connections for better performance
func (db *DB) WarmupConnections() error {
	if db.DB == nil {
		return fmt.Errorf("database not initialized")
	}
	
	stats := db.GetPoolStats()
	maxIdle := stats.MaxOpenConnections
	if maxIdle > 10 {
		maxIdle = 10 // Don't warm up too many at once
	}
	
	// Create connections by opening and immediately closing them
	for i := 0; i < maxIdle; i++ {
		conn, err := db.DB.Conn(context.Background())
		if err != nil {
			return fmt.Errorf("failed to warm up connection %d: %v", i, err)
		}
		// Close immediately to return to idle pool
		conn.Close()
	}
	
	return nil
}

// ConnectionPoolDiagnostics provides detailed diagnostics about the connection pool
type ConnectionPoolDiagnostics struct {
	Config      DatabaseConfig        `json:"config"`
	Metrics     ConnectionPoolMetrics `json:"metrics"`
	Healthy     bool                  `json:"healthy"`
	Efficiency  float64               `json:"efficiency"`  // Ratio of in-use to total connections
	Utilization float64               `json:"utilization"` // Ratio of open to max connections
	Suggestions []string              `json:"suggestions"`
}

// DiagnoseConnectionPool provides comprehensive analysis of connection pool performance
func (db *DB) DiagnoseConnectionPool() ConnectionPoolDiagnostics {
	metrics := db.GetDetailedPoolMetrics()
	healthy := db.IsHealthy()
	
	var efficiency, utilization float64
	var suggestions []string
	
	// Calculate efficiency (how many connections are actively used)
	if metrics.OpenConnections > 0 {
		efficiency = float64(metrics.InUse) / float64(metrics.OpenConnections)
	}
	
	// Calculate utilization (how much of max capacity is used)
	if metrics.MaxOpenConnections > 0 {
		utilization = float64(metrics.OpenConnections) / float64(metrics.MaxOpenConnections)
	}
	
	// Generate suggestions based on metrics
	if efficiency < 0.3 && metrics.OpenConnections > 5 {
		suggestions = append(suggestions, "Consider reducing MaxIdleConns - many connections are idle")
	}
	
	if utilization > 0.9 && metrics.WaitCount > 100 {
		suggestions = append(suggestions, "Consider increasing MaxOpenConns - high contention detected")
	}
	
	if metrics.WaitDuration > 10*time.Second {
		suggestions = append(suggestions, "High wait times detected - check query performance or increase connection pool")
	}
	
	if metrics.MaxLifetimeClosed > metrics.MaxIdleClosed && metrics.MaxLifetimeClosed > 1000 {
		suggestions = append(suggestions, "Many connections closed due to lifetime - consider increasing ConnMaxLifetime")
	}
	
	if efficiency > 0.8 && utilization < 0.5 {
		suggestions = append(suggestions, "Good efficiency but low utilization - pool size is well-tuned")
	}
	
	return ConnectionPoolDiagnostics{
		Config: DatabaseConfig{
			MaxOpenConns:    metrics.MaxOpenConnections,
			MaxIdleConns:    metrics.MaxOpenConnections - metrics.InUse, // Estimate
			ConnMaxLifetime: 30 * time.Minute,                           // Default estimate
			ConnMaxIdleTime: 15 * time.Minute,                           // Default estimate
		},
		Metrics:     metrics,
		Healthy:     healthy,
		Efficiency:  efficiency,
		Utilization: utilization,
		Suggestions: suggestions,
	}
}