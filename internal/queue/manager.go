package queue

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DefaultManager implements the Manager interface
type DefaultManager struct {
	connections    map[string]Queue
	defaultConn    string
	mutex          sync.RWMutex
	workerPool     *WorkerPool
	closed         bool
}

// NewManager creates a new queue manager
func NewManager() *DefaultManager {
	return &DefaultManager{
		connections: make(map[string]Queue),
		defaultConn: "memory",
		workerPool:  NewWorkerPool(),
		closed:      false,
	}
}

// Connection returns a queue connection
func (m *DefaultManager) Connection(name ...string) Queue {
	connName := m.defaultConn
	if len(name) > 0 && name[0] != "" {
		connName = name[0]
	}

	m.mutex.RLock()
	if conn, exists := m.connections[connName]; exists && !m.closed {
		m.mutex.RUnlock()
		return conn
	}
	m.mutex.RUnlock()

	// Create connection if it doesn't exist
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.closed {
		return nil
	}

	// Double-check after acquiring write lock
	if conn, exists := m.connections[connName]; exists {
		return conn
	}

	// Create default connection
	conn := m.createDefaultConnection(connName)
	m.connections[connName] = conn
	return conn
}

// Push adds a job using the default connection
func (m *DefaultManager) Push(ctx context.Context, job Job) error {
	if m.closed {
		return fmt.Errorf("manager is closed")
	}
	
	conn := m.Connection()
	if conn == nil {
		return fmt.Errorf("no connection available")
	}
	return conn.Push(ctx, job)
}

// PushOn adds a job to a specific queue using the default connection
func (m *DefaultManager) PushOn(ctx context.Context, queue string, job Job) error {
	if m.closed {
		return fmt.Errorf("manager is closed")
	}
	
	conn := m.Connection()
	if conn == nil {
		return fmt.Errorf("no connection available")
	}
	return conn.PushOn(ctx, queue, job)
}

// Later adds a delayed job using the default connection
func (m *DefaultManager) Later(ctx context.Context, delay time.Duration, job Job) error {
	if m.closed {
		return fmt.Errorf("manager is closed")
	}
	
	conn := m.Connection()
	if conn == nil {
		return fmt.Errorf("no connection available")
	}
	return conn.Later(ctx, delay, job)
}

// LaterOn adds a delayed job to a specific queue using the default connection
func (m *DefaultManager) LaterOn(ctx context.Context, queue string, delay time.Duration, job Job) error {
	if m.closed {
		return fmt.Errorf("manager is closed")
	}
	
	conn := m.Connection()
	if conn == nil {
		return fmt.Errorf("no connection available")
	}
	return conn.LaterOn(ctx, queue, delay, job)
}

// RegisterConnection registers a queue connection
func (m *DefaultManager) RegisterConnection(name string, queue Queue) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if !m.closed {
		m.connections[name] = queue
	}
}

// SetDefaultConnection sets the default connection name
func (m *DefaultManager) SetDefaultConnection(name string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.defaultConn = name
}

// GetDefaultConnection returns the default connection name
func (m *DefaultManager) GetDefaultConnection() string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.defaultConn
}

// Close closes all connections
func (m *DefaultManager) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.closed {
		return nil
	}

	m.closed = true

	// Stop all workers
	if err := m.workerPool.StopAll(context.Background()); err != nil {
		// Log error but continue closing connections
	}

	// Close all connections
	var lastErr error
	for name, conn := range m.connections {
		if err := conn.Close(); err != nil {
			lastErr = fmt.Errorf("error closing connection %s: %w", name, err)
		}
	}

	// Clear connections
	m.connections = make(map[string]Queue)

	return lastErr
}

// createDefaultConnection creates a default connection
func (m *DefaultManager) createDefaultConnection(name string) Queue {
	switch name {
	case "memory":
		return NewMemoryQueue()
	case "priority":
		return NewPriorityQueue()
	default:
		return NewMemoryQueue()
	}
}

// GetConnections returns all connection names
func (m *DefaultManager) GetConnections() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	connections := make([]string, 0, len(m.connections))
	for name := range m.connections {
		connections = append(connections, name)
	}
	return connections
}

// GetWorkerPool returns the worker pool
func (m *DefaultManager) GetWorkerPool() *WorkerPool {
	return m.workerPool
}

// StartWorker starts a new worker
func (m *DefaultManager) StartWorker(ctx context.Context, id string, connectionName string, options WorkerOptions) error {
	if m.closed {
		return fmt.Errorf("manager is closed")
	}

	conn := m.Connection(connectionName)
	if conn == nil {
		return fmt.Errorf("connection %s not found", connectionName)
	}

	return m.workerPool.StartWorker(ctx, id, conn, options)
}

// StopWorker stops a worker
func (m *DefaultManager) StopWorker(ctx context.Context, id string) error {
	return m.workerPool.StopWorker(ctx, id)
}

// GetWorkerStats returns worker statistics
func (m *DefaultManager) GetWorkerStats() []*WorkerStats {
	return m.workerPool.GetStats()
}

// DefaultDispatcher implements the Dispatcher interface
type DefaultDispatcher struct {
	manager Manager
}

// NewDispatcher creates a new dispatcher
func NewDispatcher(manager Manager) *DefaultDispatcher {
	return &DefaultDispatcher{manager: manager}
}

// Dispatch adds a job to the default queue
func (d *DefaultDispatcher) Dispatch(ctx context.Context, job Job) error {
	return d.manager.Push(ctx, job)
}

// DispatchOn adds a job to a specific queue
func (d *DefaultDispatcher) DispatchOn(ctx context.Context, queue string, job Job) error {
	return d.manager.PushOn(ctx, queue, job)
}

// DispatchLater adds a delayed job
func (d *DefaultDispatcher) DispatchLater(ctx context.Context, delay time.Duration, job Job) error {
	return d.manager.Later(ctx, delay, job)
}

// DispatchSync processes a job synchronously
func (d *DefaultDispatcher) DispatchSync(ctx context.Context, job Job) error {
	return job.Handle(ctx)
}

// Batch dispatches multiple jobs as a batch
func (d *DefaultDispatcher) Batch(ctx context.Context, jobs []Job) (*BatchResult, error) {
	result := &BatchResult{
		ID:         GenerateJobID(),
		TotalJobs:  len(jobs),
		CreatedAt:  time.Now(),
	}

	for _, job := range jobs {
		if err := d.Dispatch(ctx, job); err != nil {
			result.Failed++
			if queueJob, ok := job.(*QueueJob); ok {
				result.FailedJobs = append(result.FailedJobs, queueJob.GetID())
			}
		} else {
			result.Successful++
		}
	}

	now := time.Now()
	result.CompletedAt = &now

	return result, nil
}

// DefaultRepository implements the Repository interface
type DefaultRepository struct {
	manager *DefaultManager
	configs map[string]Config
	mutex   sync.RWMutex
}

// NewRepository creates a new repository
func NewRepository() *DefaultRepository {
	return &DefaultRepository{
		manager: NewManager(),
		configs: make(map[string]Config),
	}
}

// GetManager returns the queue manager
func (r *DefaultRepository) GetManager() Manager {
	return r.manager
}

// CreateConnection creates a new queue connection
func (r *DefaultRepository) CreateConnection(name string, config Config) (Queue, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var queue Queue
	var err error

	switch config.Driver {
	case "memory":
		queue = NewMemoryQueue()
	case "priority":
		queue = NewPriorityQueue()
	default:
		return nil, fmt.Errorf("unsupported queue driver: %s", config.Driver)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create %s queue: %w", config.Driver, err)
	}

	r.manager.RegisterConnection(name, queue)
	r.configs[name] = config

	return queue, nil
}

// GetConnection returns an existing connection
func (r *DefaultRepository) GetConnection(name string) (Queue, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	conn, exists := r.manager.connections[name]
	return conn, exists
}

// SetupQueues initializes queues from configuration
func (r *DefaultRepository) SetupQueues(configs map[string]Config) error {
	for name, config := range configs {
		if _, err := r.CreateConnection(name, config); err != nil {
			return fmt.Errorf("failed to setup queue %s: %w", name, err)
		}
	}
	return nil
}

// Close closes all resources
func (r *DefaultRepository) Close() error {
	return r.manager.Close()
}

// GetConfigs returns all configurations
func (r *DefaultRepository) GetConfigs() map[string]Config {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	result := make(map[string]Config)
	for name, config := range r.configs {
		result[name] = config
	}
	return result
}

// Global repository instance
var globalRepository *DefaultRepository

// SetupQueues initializes the global queue repository
func SetupQueues(configs map[string]Config) error {
	globalRepository = NewRepository()
	return globalRepository.SetupQueues(configs)
}

// GetRepository returns the global queue repository
func GetRepository() Repository {
	if globalRepository == nil {
		globalRepository = NewRepository()
	}
	return globalRepository
}

// Global queue functions

// GetManager returns the manager from the global repository
func GetManager() Manager {
	return GetRepository().GetManager()
}

// GetConnection returns a connection from the global repository
func GetConnection(name ...string) Queue {
	return GetManager().Connection(name...)
}

// Push adds a job using the global manager
func Push(ctx context.Context, job Job) error {
	return GetManager().Push(ctx, job)
}

// PushOn adds a job to a specific queue using the global manager
func PushOn(ctx context.Context, queue string, job Job) error {
	return GetManager().PushOn(ctx, queue, job)
}

// Later adds a delayed job using the global manager
func Later(ctx context.Context, delay time.Duration, job Job) error {
	return GetManager().Later(ctx, delay, job)
}

// LaterOn adds a delayed job to a specific queue using the global manager
func LaterOn(ctx context.Context, queue string, delay time.Duration, job Job) error {
	return GetManager().LaterOn(ctx, queue, delay, job)
}

// RegisterConnection registers a connection with the global manager
func RegisterConnection(name string, queue Queue) {
	GetManager().RegisterConnection(name, queue)
}

// SetDefaultConnection sets the default connection in the global manager
func SetDefaultConnection(name string) {
	GetManager().SetDefaultConnection(name)
}