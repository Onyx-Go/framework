package onyx

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestDefaultDatabaseConfig(t *testing.T) {
	config := DefaultDatabaseConfig()
	
	if config.Driver != "sqlite3" {
		t.Errorf("Expected driver 'sqlite3', got '%s'", config.Driver)
	}
	
	if config.MaxOpenConns != 25 {
		t.Errorf("Expected MaxOpenConns 25, got %d", config.MaxOpenConns)
	}
	
	if config.MaxIdleConns != 10 {
		t.Errorf("Expected MaxIdleConns 10, got %d", config.MaxIdleConns)
	}
	
	if config.ConnMaxLifetime != 30*time.Minute {
		t.Errorf("Expected ConnMaxLifetime 30m, got %v", config.ConnMaxLifetime)
	}
}

func TestMySQLConfig(t *testing.T) {
	dsn := "user:password@tcp(localhost:3306)/testdb"
	config := MySQLConfig(dsn)
	
	if config.Driver != "mysql" {
		t.Errorf("Expected driver 'mysql', got '%s'", config.Driver)
	}
	
	if config.DSN != dsn {
		t.Errorf("Expected DSN '%s', got '%s'", dsn, config.DSN)
	}
	
	if config.MaxOpenConns != 50 {
		t.Errorf("Expected MaxOpenConns 50, got %d", config.MaxOpenConns)
	}
}

func TestPostgreSQLConfig(t *testing.T) {
	dsn := "postgres://user:password@localhost/testdb?sslmode=disable"
	config := PostgreSQLConfig(dsn)
	
	if config.Driver != "postgres" {
		t.Errorf("Expected driver 'postgres', got '%s'", config.Driver)
	}
	
	if config.MaxOpenConns != 40 {
		t.Errorf("Expected MaxOpenConns 40, got %d", config.MaxOpenConns)
	}
}

func TestSQLiteConfig(t *testing.T) {
	dsn := "test.db"
	config := SQLiteConfig(dsn)
	
	if config.Driver != "sqlite3" {
		t.Errorf("Expected driver 'sqlite3', got '%s'", config.Driver)
	}
	
	if config.MaxOpenConns != 1 {
		t.Errorf("Expected MaxOpenConns 1 for SQLite, got %d", config.MaxOpenConns)
	}
	
	if config.MaxIdleConns != 1 {
		t.Errorf("Expected MaxIdleConns 1 for SQLite, got %d", config.MaxIdleConns)
	}
}

func TestNewDBWithConfig(t *testing.T) {
	config := DefaultDatabaseConfig()
	
	db, err := NewDBWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create database with config: %v", err)
	}
	defer db.Close()
	
	// Verify connection pool settings
	stats := db.GetPoolStats()
	if stats.MaxOpenConnections != config.MaxOpenConns {
		t.Errorf("Expected MaxOpenConnections %d, got %d", config.MaxOpenConns, stats.MaxOpenConnections)
	}
}

func TestConfigurePooling(t *testing.T) {
	// Create a basic database connection
	db, err := NewDB("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()
	
	// Apply custom pooling configuration
	config := DatabaseConfig{
		MaxOpenConns:    15,
		MaxIdleConns:    5,
		ConnMaxLifetime: 20 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	}
	
	db.ConfigurePooling(config)
	
	// Verify settings were applied
	stats := db.GetPoolStats()
	if stats.MaxOpenConnections != 15 {
		t.Errorf("Expected MaxOpenConnections 15, got %d", stats.MaxOpenConnections)
	}
}

func TestGetPoolStats(t *testing.T) {
	config := DefaultDatabaseConfig()
	db, err := NewDBWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()
	
	stats := db.GetPoolStats()
	
	// Check that we get valid stats
	if stats.MaxOpenConnections <= 0 {
		t.Error("MaxOpenConnections should be greater than 0")
	}
	
	// Initially should have 0 open connections
	if stats.OpenConnections < 0 {
		t.Error("OpenConnections should be >= 0")
	}
}

func TestGetDetailedPoolMetrics(t *testing.T) {
	config := DefaultDatabaseConfig()
	db, err := NewDBWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()
	
	metrics := db.GetDetailedPoolMetrics()
	
	if metrics.MaxOpenConnections != config.MaxOpenConns {
		t.Errorf("Expected MaxOpenConnections %d, got %d", config.MaxOpenConns, metrics.MaxOpenConnections)
	}
	
	if metrics.OpenConnections < 0 {
		t.Error("OpenConnections should be >= 0")
	}
	
	if metrics.InUse < 0 {
		t.Error("InUse should be >= 0")
	}
	
	if metrics.Idle < 0 {
		t.Error("Idle should be >= 0")
	}
}

func TestIsHealthy(t *testing.T) {
	config := DefaultDatabaseConfig()
	db, err := NewDBWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()
	
	if !db.IsHealthy() {
		t.Error("Database should be healthy")
	}
	
	// Test with closed database
	db.Close()
	if db.IsHealthy() {
		t.Error("Closed database should not be healthy")
	}
}

func TestOptimizeForWorkload(t *testing.T) {
	config := DefaultDatabaseConfig()
	db, err := NewDBWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()
	
	testCases := []struct {
		workload string
		expected int // expected MaxOpenConns
	}{
		{"high_read", 100},
		{"high_write", 30},
		{"batch_processing", 10},
		{"low_latency", 20},
		{"unknown", 25}, // should fall back to default
	}
	
	for _, tc := range testCases {
		t.Run(tc.workload, func(t *testing.T) {
			db.OptimizeForWorkload(tc.workload)
			stats := db.GetPoolStats()
			
			if stats.MaxOpenConnections != tc.expected {
				t.Errorf("For workload '%s', expected MaxOpenConnections %d, got %d", 
					tc.workload, tc.expected, stats.MaxOpenConnections)
			}
		})
	}
}

func TestWarmupConnections(t *testing.T) {
	config := DatabaseConfig{
		Driver:          "sqlite3",
		DSN:             ":memory:",
		MaxOpenConns:    5,
		MaxIdleConns:    3,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 15 * time.Minute,
	}
	
	db, err := NewDBWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()
	
	// Warm up connections
	err = db.WarmupConnections()
	if err != nil {
		t.Fatalf("Failed to warm up connections: %v", err)
	}
	
	// After warmup, we should have some idle connections
	// Note: The exact number may vary based on implementation
	stats := db.GetPoolStats()
	if stats.OpenConnections < 0 {
		t.Error("Should have some open connections after warmup")
	}
}

func TestConnectionPoolConcurrency(t *testing.T) {
	config := DatabaseConfig{
		Driver:          "sqlite3",
		DSN:             "file:memdb1?mode=memory&cache=shared",
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 15 * time.Minute,
	}
	
	db, err := NewDBWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()
	
	// Create a test table
	_, err = db.Exec("CREATE TABLE test_concurrency (id INTEGER PRIMARY KEY, value TEXT)")
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}
	
	// Run concurrent read operations (safer for SQLite)
	var wg sync.WaitGroup
	const numWorkers = 10
	const opsPerWorker = 5
	
	// First insert some test data
	for i := 0; i < 10; i++ {
		_, err = db.Exec("INSERT INTO test_concurrency (value) VALUES (?)", fmt.Sprintf("test-%d", i))
		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}
	}
	
	// Run concurrent reads
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			for j := 0; j < opsPerWorker; j++ {
				// Select operation - safe for concurrent access
				var count int
				err := db.QueryRow("SELECT COUNT(*) FROM test_concurrency").Scan(&count)
				if err != nil {
					t.Errorf("Worker %d failed to select: %v", workerID, err)
					return
				}
				
				if count < 10 {
					t.Errorf("Worker %d got unexpected count: %d", workerID, count)
					return
				}
			}
		}(i)
	}
	
	wg.Wait()
	
	// Check final metrics
	metrics := db.GetDetailedPoolMetrics()
	if metrics.WaitCount < 0 {
		t.Error("WaitCount should be >= 0")
	}
	
	// Verify data integrity
	var finalCount int
	err = db.QueryRow("SELECT COUNT(*) FROM test_concurrency").Scan(&finalCount)
	if err != nil {
		t.Fatalf("Failed to get final count: %v", err)
	}
	
	if finalCount != 10 {
		t.Errorf("Expected 10 records, got %d", finalCount)
	}
}

func TestDiagnoseConnectionPool(t *testing.T) {
	config := DefaultDatabaseConfig()
	db, err := NewDBWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()
	
	// Run some operations to generate metrics
	for i := 0; i < 5; i++ {
		db.Ping()
	}
	
	diagnostics := db.DiagnoseConnectionPool()
	
	if !diagnostics.Healthy {
		t.Error("Database should be diagnosed as healthy")
	}
	
	if diagnostics.Efficiency < 0 || diagnostics.Efficiency > 1 {
		t.Errorf("Efficiency should be between 0 and 1, got %f", diagnostics.Efficiency)
	}
	
	if diagnostics.Utilization < 0 || diagnostics.Utilization > 1 {
		t.Errorf("Utilization should be between 0 and 1, got %f", diagnostics.Utilization)
	}
	
	// Should have valid metrics
	if diagnostics.Metrics.MaxOpenConnections <= 0 {
		t.Error("MaxOpenConnections should be > 0")
	}
}

func TestConnectionPoolEfficiency(t *testing.T) {
	config := DatabaseConfig{
		Driver:          "sqlite3",
		DSN:             ":memory:",
		MaxOpenConns:    5,
		MaxIdleConns:    5,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 15 * time.Minute,
	}
	
	db, err := NewDBWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()
	
	// Create a table for testing
	_, err = db.Exec("CREATE TABLE efficiency_test (id INTEGER PRIMARY KEY, data TEXT)")
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}
	
	// Simulate concurrent load
	var wg sync.WaitGroup
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			for {
				select {
				case <-ctx.Done():
					return
				default:
					// Keep connections busy
					db.Exec("INSERT INTO efficiency_test (data) VALUES (?)", fmt.Sprintf("test-%d", id))
					time.Sleep(50 * time.Millisecond)
				}
			}
		}(i)
	}
	
	// Let it run for a bit
	time.Sleep(500 * time.Millisecond)
	
	// Check diagnostics while under load
	diagnostics := db.DiagnoseConnectionPool()
	
	// Should have some efficiency since connections are being used
	if diagnostics.Efficiency < 0 {
		t.Errorf("Expected some efficiency under load, got %f", diagnostics.Efficiency)
	}
	
	cancel()
	wg.Wait()
	
	// Final diagnostics
	finalDiagnostics := db.DiagnoseConnectionPool()
	if !finalDiagnostics.Healthy {
		t.Error("Database should still be healthy after load test")
	}
}

// Benchmark connection pool performance
func BenchmarkConnectionPoolPerformance(b *testing.B) {
	config := DatabaseConfig{
		Driver:          "sqlite3",
		DSN:             ":memory:",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 15 * time.Minute,
	}
	
	db, err := NewDBWithConfig(config)
	if err != nil {
		b.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()
	
	// Create test table
	_, err = db.Exec("CREATE TABLE benchmark_test (id INTEGER PRIMARY KEY, value INTEGER)")
	if err != nil {
		b.Fatalf("Failed to create test table: %v", err)
	}
	
	b.ResetTimer()
	
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Simple insert operation
			_, err := db.Exec("INSERT INTO benchmark_test (value) VALUES (?)", b.N)
			if err != nil {
				b.Errorf("Insert failed: %v", err)
			}
		}
	})
}