package onyx

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"
)

// QueryOptimizer provides query optimization features
type QueryOptimizer struct {
	db                *DB
	enablePreparedStmts bool
	enableQueryCache    bool
	enableIndexHints    bool
	preparedStmts       map[string]*sql.Stmt
	queryCache          map[string]CachedQuery
	cacheMutex          sync.RWMutex
	stmtMutex           sync.RWMutex
	maxCacheSize        int
	cacheHits           int64
	cacheMisses         int64
}

// CachedQuery represents a cached query result
type CachedQuery struct {
	Result     interface{}
	ExpiresAt  time.Time
	QueryHash  string
	Parameters []interface{}
}

// QueryOptimizationConfig configures query optimization features
type QueryOptimizationConfig struct {
	EnablePreparedStatements bool          `json:"enable_prepared_statements"`
	EnableQueryCache         bool          `json:"enable_query_cache"`
	EnableIndexHints         bool          `json:"enable_index_hints"`
	QueryCacheTTL           time.Duration `json:"query_cache_ttl"`
	MaxCacheSize            int           `json:"max_cache_size"`
	PreparedStatementsLimit int           `json:"prepared_statements_limit"`
}

// DefaultQueryOptimizationConfig returns sensible defaults
func DefaultQueryOptimizationConfig() QueryOptimizationConfig {
	return QueryOptimizationConfig{
		EnablePreparedStatements: true,
		EnableQueryCache:         true,
		EnableIndexHints:         false, // Can be database-specific
		QueryCacheTTL:           5 * time.Minute,
		MaxCacheSize:            1000,
		PreparedStatementsLimit: 100,
	}
}

// NewQueryOptimizer creates a new query optimizer
func NewQueryOptimizer(db *DB, config QueryOptimizationConfig) *QueryOptimizer {
	return &QueryOptimizer{
		db:                  db,
		enablePreparedStmts: config.EnablePreparedStatements,
		enableQueryCache:    config.EnableQueryCache,
		enableIndexHints:    config.EnableIndexHints,
		preparedStmts:       make(map[string]*sql.Stmt),
		queryCache:          make(map[string]CachedQuery),
		maxCacheSize:        config.MaxCacheSize,
	}
}

// OptimizedQueryBuilder extends QueryBuilder with optimization features
type OptimizedQueryBuilder struct {
	*QueryBuilder
	optimizer     *QueryOptimizer
	enableCache   bool
	cacheTTL      time.Duration
	forceIndex    string
	ignoreIndex   string
	useIndex      string
	queryHints    []string
}

// Optimize creates an optimized query builder
func (qb *QueryBuilder) Optimize(optimizer *QueryOptimizer) *OptimizedQueryBuilder {
	return &OptimizedQueryBuilder{
		QueryBuilder: qb,
		optimizer:    optimizer,
		enableCache:  true,
		cacheTTL:     5 * time.Minute,
	}
}

// WithCache enables or disables caching for this query
func (oqb *OptimizedQueryBuilder) WithCache(enable bool, ttl time.Duration) *OptimizedQueryBuilder {
	oqb.enableCache = enable
	oqb.cacheTTL = ttl
	return oqb
}

// ForceIndex adds a FORCE INDEX hint (MySQL specific)
func (oqb *OptimizedQueryBuilder) ForceIndex(indexName string) *OptimizedQueryBuilder {
	oqb.forceIndex = indexName
	return oqb
}

// IgnoreIndex adds an IGNORE INDEX hint (MySQL specific)
func (oqb *OptimizedQueryBuilder) IgnoreIndex(indexName string) *OptimizedQueryBuilder {
	oqb.ignoreIndex = indexName
	return oqb
}

// UseIndex adds a USE INDEX hint (MySQL specific)
func (oqb *OptimizedQueryBuilder) UseIndex(indexName string) *OptimizedQueryBuilder {
	oqb.useIndex = indexName
	return oqb
}

// Hint adds a custom query hint
func (oqb *OptimizedQueryBuilder) Hint(hint string) *OptimizedQueryBuilder {
	oqb.queryHints = append(oqb.queryHints, hint)
	return oqb
}

// buildOptimizedQuery creates an optimized SQL query with hints
func (oqb *OptimizedQueryBuilder) buildOptimizedQuery() (string, []interface{}) {
	query, args := oqb.QueryBuilder.buildSelectQuery()
	
	// Add index hints for MySQL
	if oqb.optimizer.enableIndexHints && oqb.optimizer.db.driver == "mysql" {
		tablePart := oqb.table
		
		if oqb.forceIndex != "" {
			tablePart += fmt.Sprintf(" FORCE INDEX (%s)", oqb.forceIndex)
		} else if oqb.useIndex != "" {
			tablePart += fmt.Sprintf(" USE INDEX (%s)", oqb.useIndex)
		} else if oqb.ignoreIndex != "" {
			tablePart += fmt.Sprintf(" IGNORE INDEX (%s)", oqb.ignoreIndex)
		}
		
		// Replace table name with table + index hint
		query = strings.Replace(query, "FROM "+oqb.table, "FROM "+tablePart, 1)
	}
	
	// Add custom query hints
	if len(oqb.queryHints) > 0 {
		hintString := "/*+ " + strings.Join(oqb.queryHints, ", ") + " */"
		query = hintString + " " + query
	}
	
	return query, args
}

// GetOptimized executes an optimized SELECT query
func (oqb *OptimizedQueryBuilder) GetOptimized() ([]map[string]interface{}, error) {
	query, args := oqb.buildOptimizedQuery()
	
	// Check cache first
	if oqb.enableCache && oqb.optimizer.enableQueryCache {
		if cached := oqb.optimizer.getCachedQuery(query, args); cached != nil {
			oqb.optimizer.cacheMutex.RLock()
			oqb.optimizer.cacheHits++
			oqb.optimizer.cacheMutex.RUnlock()
			return cached.Result.([]map[string]interface{}), nil
		}
		oqb.optimizer.cacheMutex.RLock()
		oqb.optimizer.cacheMisses++
		oqb.optimizer.cacheMutex.RUnlock()
	}
	
	// Execute query
	var rows *sql.Rows
	var err error
	
	if oqb.optimizer.enablePreparedStmts {
		// Use prepared statement
		stmt := oqb.optimizer.getPreparedStatement(query)
		if stmt != nil {
			rows, err = stmt.Query(args...)
		} else {
			rows, err = oqb.db.Query(query, args...)
		}
	} else {
		rows, err = oqb.db.Query(query, args...)
	}
	
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	result := oqb.scanRowsToMaps(rows)
	
	// Cache result
	if oqb.enableCache && oqb.optimizer.enableQueryCache {
		oqb.optimizer.cacheQuery(query, args, result, oqb.cacheTTL)
	}
	
	return result, nil
}

// getPreparedStatement retrieves or creates a prepared statement
func (qo *QueryOptimizer) getPreparedStatement(query string) *sql.Stmt {
	qo.stmtMutex.RLock()
	stmt, exists := qo.preparedStmts[query]
	qo.stmtMutex.RUnlock()
	
	if exists {
		return stmt
	}
	
	// Create new prepared statement
	stmt, err := qo.db.Prepare(query)
	if err != nil {
		return nil // Fall back to regular query
	}
	
	qo.stmtMutex.Lock()
	qo.preparedStmts[query] = stmt
	qo.stmtMutex.Unlock()
	
	return stmt
}

// getCachedQuery retrieves a cached query result
func (qo *QueryOptimizer) getCachedQuery(query string, args []interface{}) *CachedQuery {
	qo.cacheMutex.RLock()
	defer qo.cacheMutex.RUnlock()
	
	key := qo.generateCacheKey(query, args)
	cached, exists := qo.queryCache[key]
	
	if !exists || time.Now().After(cached.ExpiresAt) {
		return nil
	}
	
	return &cached
}

// cacheQuery stores a query result in cache
func (qo *QueryOptimizer) cacheQuery(query string, args []interface{}, result interface{}, ttl time.Duration) {
	qo.cacheMutex.Lock()
	defer qo.cacheMutex.Unlock()
	
	// Check if we need to evict before adding
	if len(qo.queryCache) >= qo.maxCacheSize {
		qo.evictOldestCacheEntryUnsafe()
	}
	
	key := qo.generateCacheKey(query, args)
	cached := CachedQuery{
		Result:     result,
		ExpiresAt:  time.Now().Add(ttl),
		QueryHash:  key,
		Parameters: args,
	}
	
	qo.queryCache[key] = cached
}

// generateCacheKey creates a unique key for caching
func (qo *QueryOptimizer) generateCacheKey(query string, args []interface{}) string {
	key := query
	for _, arg := range args {
		key += fmt.Sprintf(":%v", arg)
	}
	return key
}

// evictOldestCacheEntry removes the oldest cache entry
func (qo *QueryOptimizer) evictOldestCacheEntry() {
	qo.cacheMutex.Lock()
	defer qo.cacheMutex.Unlock()
	qo.evictOldestCacheEntryUnsafe()
}

// evictOldestCacheEntryUnsafe removes the oldest cache entry (caller must hold lock)
func (qo *QueryOptimizer) evictOldestCacheEntryUnsafe() {
	if len(qo.queryCache) == 0 {
		return
	}
	
	var oldestKey string
	var oldestTime time.Time
	first := true
	
	for key, cached := range qo.queryCache {
		if first || cached.ExpiresAt.Before(oldestTime) {
			oldestTime = cached.ExpiresAt
			oldestKey = key
			first = false
		}
	}
	
	if oldestKey != "" {
		delete(qo.queryCache, oldestKey)
	}
}

// ClearCache clears all cached queries
func (qo *QueryOptimizer) ClearCache() {
	qo.cacheMutex.Lock()
	qo.queryCache = make(map[string]CachedQuery)
	qo.cacheMutex.Unlock()
}

// ClearPreparedStatements closes and clears all prepared statements
func (qo *QueryOptimizer) ClearPreparedStatements() {
	qo.stmtMutex.Lock()
	defer qo.stmtMutex.Unlock()
	
	for _, stmt := range qo.preparedStmts {
		stmt.Close()
	}
	qo.preparedStmts = make(map[string]*sql.Stmt)
}

// GetCacheStats returns cache performance statistics
func (qo *QueryOptimizer) GetCacheStats() map[string]interface{} {
	qo.cacheMutex.RLock()
	defer qo.cacheMutex.RUnlock()
	
	totalRequests := qo.cacheHits + qo.cacheMisses
	hitRate := float64(0)
	if totalRequests > 0 {
		hitRate = float64(qo.cacheHits) / float64(totalRequests)
	}
	
	return map[string]interface{}{
		"cache_hits":     qo.cacheHits,
		"cache_misses":   qo.cacheMisses,
		"hit_rate":       hitRate,
		"cached_queries": len(qo.queryCache),
		"max_cache_size": qo.maxCacheSize,
	}
}

// GetPreparedStatementStats returns prepared statement statistics
func (qo *QueryOptimizer) GetPreparedStatementStats() map[string]interface{} {
	qo.stmtMutex.RLock()
	defer qo.stmtMutex.RUnlock()
	
	return map[string]interface{}{
		"prepared_statements": len(qo.preparedStmts),
	}
}

// scanRowsToMaps converts sql.Rows to a slice of maps
func (oqb *OptimizedQueryBuilder) scanRowsToMaps(rows *sql.Rows) []map[string]interface{} {
	columns, _ := rows.Columns()
	columnTypes, _ := rows.ColumnTypes()
	
	var results []map[string]interface{}
	
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		
		rows.Scan(valuePtrs...)
		
		entry := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			
			if val == nil {
				entry[col] = nil
				continue
			}
			
			// Handle different column types
			switch columnTypes[i].DatabaseTypeName() {
			case "TEXT", "VARCHAR", "CHAR":
				if b, ok := val.([]byte); ok {
					entry[col] = string(b)
				} else {
					entry[col] = val
				}
			case "INTEGER", "INT", "BIGINT":
				entry[col] = val
			case "REAL", "FLOAT", "DOUBLE":
				entry[col] = val
			case "DATETIME", "TIMESTAMP":
				entry[col] = val
			default:
				entry[col] = val
			}
		}
		
		results = append(results, entry)
	}
	
	return results
}

// QueryProfiler provides query performance profiling
type QueryProfiler struct {
	enabled        bool
	queries        []QueryProfile
	mutex          sync.RWMutex
	maxQueries     int
	slowQueryThreshold time.Duration
}

// QueryProfile contains query execution statistics
type QueryProfile struct {
	Query         string        `json:"query"`
	Duration      time.Duration `json:"duration"`
	Timestamp     time.Time     `json:"timestamp"`
	RowsAffected  int64         `json:"rows_affected"`
	Error         string        `json:"error,omitempty"`
	Parameters    []interface{} `json:"parameters,omitempty"`
}

// NewQueryProfiler creates a new query profiler
func NewQueryProfiler(enabled bool, maxQueries int, slowQueryThreshold time.Duration) *QueryProfiler {
	return &QueryProfiler{
		enabled:           enabled,
		queries:           make([]QueryProfile, 0, maxQueries),
		maxQueries:        maxQueries,
		slowQueryThreshold: slowQueryThreshold,
	}
}

// ProfileQuery wraps query execution with profiling
func (qp *QueryProfiler) ProfileQuery(query string, params []interface{}, fn func() (interface{}, error)) (interface{}, error) {
	if !qp.enabled {
		return fn()
	}
	
	start := time.Now()
	result, err := fn()
	duration := time.Since(start)
	
	profile := QueryProfile{
		Query:     query,
		Duration:  duration,
		Timestamp: start,
		Parameters: params,
	}
	
	if err != nil {
		profile.Error = err.Error()
	}
	
	// Try to get rows affected if result supports it
	if r, ok := result.(sql.Result); ok {
		if rowsAffected, err := r.RowsAffected(); err == nil {
			profile.RowsAffected = rowsAffected
		}
	}
	
	qp.addProfile(profile)
	
	return result, err
}

// addProfile adds a query profile to the collection
func (qp *QueryProfiler) addProfile(profile QueryProfile) {
	qp.mutex.Lock()
	defer qp.mutex.Unlock()
	
	if len(qp.queries) >= qp.maxQueries {
		// Remove oldest query
		qp.queries = qp.queries[1:]
	}
	
	qp.queries = append(qp.queries, profile)
}

// GetSlowQueries returns queries that exceeded the slow query threshold
func (qp *QueryProfiler) GetSlowQueries() []QueryProfile {
	qp.mutex.RLock()
	defer qp.mutex.RUnlock()
	
	var slowQueries []QueryProfile
	for _, profile := range qp.queries {
		if profile.Duration > qp.slowQueryThreshold {
			slowQueries = append(slowQueries, profile)
		}
	}
	
	return slowQueries
}

// GetQueryStats returns aggregate query statistics
func (qp *QueryProfiler) GetQueryStats() map[string]interface{} {
	qp.mutex.RLock()
	defer qp.mutex.RUnlock()
	
	if len(qp.queries) == 0 {
		return map[string]interface{}{
			"total_queries": 0,
			"avg_duration": 0,
			"slow_queries": 0,
		}
	}
	
	var totalDuration time.Duration
	var slowQueries int
	var errorCount int
	
	for _, profile := range qp.queries {
		totalDuration += profile.Duration
		if profile.Duration > qp.slowQueryThreshold {
			slowQueries++
		}
		if profile.Error != "" {
			errorCount++
		}
	}
	
	avgDuration := totalDuration / time.Duration(len(qp.queries))
	
	return map[string]interface{}{
		"total_queries":    len(qp.queries),
		"avg_duration":     avgDuration,
		"slow_queries":     slowQueries,
		"error_count":      errorCount,
		"slow_query_threshold": qp.slowQueryThreshold,
	}
}

// Close cleans up resources
func (qo *QueryOptimizer) Close() error {
	qo.ClearPreparedStatements()
	qo.ClearCache()
	return nil
}