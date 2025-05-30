package queue

import (
	"context"
	"time"
)

// Job represents a unit of work that can be queued and processed
type Job interface {
	// Handle processes the job with context support
	Handle(ctx context.Context) error
	
	// Failed is called when the job fails
	Failed(ctx context.Context, err error) error
	
	// GetPayload returns the job's data payload
	GetPayload() map[string]interface{}
	
	// GetQueue returns the queue name for this job
	GetQueue() string
	
	// GetDelay returns the delay before the job should be processed
	GetDelay() time.Duration
	
	// GetMaxTries returns the maximum number of attempts
	GetMaxTries() int
	
	// GetTimeout returns the maximum execution time
	GetTimeout() time.Duration
	
	// GetPriority returns the job priority
	GetPriority() Priority
	
	// GetMetadata returns job metadata
	GetMetadata() map[string]interface{}
}

// QueueableJob extends Job with fluent configuration methods
type QueueableJob interface {
	Job
	OnQueue(queue string) QueueableJob
	Delay(delay time.Duration) QueueableJob
	OnConnection(connection string) QueueableJob
	WithPriority(priority Priority) QueueableJob
	WithTimeout(timeout time.Duration) QueueableJob
	WithMaxTries(maxTries int) QueueableJob
	WithMetadata(key string, value interface{}) QueueableJob
}

// Queue represents a job queue with context support
type Queue interface {
	// Push adds a job to the queue
	Push(ctx context.Context, job Job) error
	
	// PushOn adds a job to a specific queue
	PushOn(ctx context.Context, queue string, job Job) error
	
	// Later adds a job to be processed after a delay
	Later(ctx context.Context, delay time.Duration, job Job) error
	
	// LaterOn adds a delayed job to a specific queue
	LaterOn(ctx context.Context, queue string, delay time.Duration, job Job) error
	
	// Pop retrieves the next job from the queue
	Pop(ctx context.Context, queue ...string) (Job, error)
	
	// Size returns the number of jobs in the queue
	Size(ctx context.Context, queue ...string) (int, error)
	
	// Clear removes all jobs from a queue
	Clear(ctx context.Context, queue string) error
	
	// Peek looks at the next job without removing it
	Peek(ctx context.Context, queue ...string) (Job, error)
	
	// Stats returns queue statistics
	Stats(ctx context.Context, queue ...string) (*QueueStats, error)
	
	// Close closes the queue connection
	Close() error
}

// Manager manages multiple queue connections
type Manager interface {
	// Connection returns a queue connection
	Connection(name ...string) Queue
	
	// Push adds a job using the default connection
	Push(ctx context.Context, job Job) error
	
	// PushOn adds a job to a specific queue using the default connection
	PushOn(ctx context.Context, queue string, job Job) error
	
	// Later adds a delayed job using the default connection
	Later(ctx context.Context, delay time.Duration, job Job) error
	
	// LaterOn adds a delayed job to a specific queue using the default connection
	LaterOn(ctx context.Context, queue string, delay time.Duration, job Job) error
	
	// RegisterConnection registers a queue connection
	RegisterConnection(name string, queue Queue)
	
	// SetDefaultConnection sets the default connection name
	SetDefaultConnection(name string)
	
	// GetDefaultConnection returns the default connection name
	GetDefaultConnection() string
	
	// Close closes all connections
	Close() error
}

// Worker processes jobs from a queue
type Worker interface {
	// Start begins processing jobs
	Start(ctx context.Context, queue Queue, options WorkerOptions) error
	
	// Stop gracefully stops the worker
	Stop(ctx context.Context) error
	
	// IsRunning returns true if the worker is active
	IsRunning() bool
	
	// Stats returns worker statistics
	Stats() *WorkerStats
}

// Dispatcher dispatches jobs to queues
type Dispatcher interface {
	// Dispatch adds a job to the default queue
	Dispatch(ctx context.Context, job Job) error
	
	// DispatchOn adds a job to a specific queue
	DispatchOn(ctx context.Context, queue string, job Job) error
	
	// DispatchLater adds a delayed job
	DispatchLater(ctx context.Context, delay time.Duration, job Job) error
	
	// DispatchSync processes a job synchronously
	DispatchSync(ctx context.Context, job Job) error
	
	// Batch dispatches multiple jobs as a batch
	Batch(ctx context.Context, jobs []Job) (*BatchResult, error)
}

// Repository manages queue configurations and connections
type Repository interface {
	// GetManager returns the queue manager
	GetManager() Manager
	
	// CreateConnection creates a new queue connection
	CreateConnection(name string, config Config) (Queue, error)
	
	// GetConnection returns an existing connection
	GetConnection(name string) (Queue, bool)
	
	// SetupQueues initializes queues from configuration
	SetupQueues(config map[string]Config) error
	
	// Close closes all resources
	Close() error
}

// Serializer handles job serialization
type Serializer interface {
	// Serialize converts a job to bytes
	Serialize(job Job) ([]byte, error)
	
	// Unserialize converts bytes back to a job
	Unserialize(data []byte) (Job, error)
	
	// GetContentType returns the serializer content type
	GetContentType() string
}

// Store provides persistent storage for jobs
type Store interface {
	// Save stores a job payload
	Save(ctx context.Context, payload *JobPayload) error
	
	// Find retrieves a job payload by ID
	Find(ctx context.Context, id string) (*JobPayload, error)
	
	// Delete removes a job payload
	Delete(ctx context.Context, id string) error
	
	// List returns job payloads for a queue
	List(ctx context.Context, queue string, limit int) ([]*JobPayload, error)
	
	// Count returns the number of jobs in a queue
	Count(ctx context.Context, queue string) (int, error)
	
	// Clear removes all jobs from a queue
	Clear(ctx context.Context, queue string) error
	
	// Close closes the store connection
	Close() error
}

// Monitor provides queue monitoring capabilities
type Monitor interface {
	// GetQueueStats returns statistics for a queue
	GetQueueStats(ctx context.Context, queue string) (*QueueStats, error)
	
	// GetWorkerStats returns worker statistics
	GetWorkerStats() ([]*WorkerStats, error)
	
	// GetFailedJobs returns failed jobs
	GetFailedJobs(ctx context.Context, limit int) ([]*JobPayload, error)
	
	// RetryJob retries a failed job
	RetryJob(ctx context.Context, jobID string) error
	
	// PurgeQueue removes all jobs from a queue
	PurgeQueue(ctx context.Context, queue string) error
}

// Middleware for job processing
type Middleware interface {
	// Handle processes a job with middleware
	Handle(ctx context.Context, job Job, next func(ctx context.Context, job Job) error) error
}

// MiddlewareFunc is a function that implements Middleware
type MiddlewareFunc func(ctx context.Context, job Job, next func(ctx context.Context, job Job) error) error

// Handle implements the Middleware interface
func (mf MiddlewareFunc) Handle(ctx context.Context, job Job, next func(ctx context.Context, job Job) error) error {
	return mf(ctx, job, next)
}

// Priority represents job priority levels
type Priority int

const (
	PriorityLow Priority = iota
	PriorityNormal
	PriorityHigh
	PriorityCritical
)

// String returns the string representation of priority
func (p Priority) String() string {
	switch p {
	case PriorityLow:
		return "low"
	case PriorityNormal:
		return "normal"
	case PriorityHigh:
		return "high"
	case PriorityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// JobPayload represents the serialized job data
type JobPayload struct {
	ID          string                 `json:"id"`
	DisplayName string                 `json:"displayName"`
	Job         string                 `json:"job"`
	MaxTries    int                    `json:"maxTries"`
	Timeout     int                    `json:"timeout"`
	Priority    Priority               `json:"priority"`
	Data        map[string]interface{} `json:"data"`
	Metadata    map[string]interface{} `json:"metadata"`
	Queue       string                 `json:"queue"`
	Attempts    int                    `json:"attempts"`
	CreatedAt   time.Time              `json:"created_at"`
	AvailableAt time.Time              `json:"available_at"`
	ProcessedAt *time.Time             `json:"processed_at,omitempty"`
	FailedAt    *time.Time             `json:"failed_at,omitempty"`
	LastError   string                 `json:"last_error,omitempty"`
}

// WorkerOptions configures worker behavior
type WorkerOptions struct {
	Queue         string             `json:"queue"`
	MaxJobs       int                `json:"max_jobs"`
	MaxTime       time.Duration      `json:"max_time"`
	Memory        int                `json:"memory"`
	Sleep         time.Duration      `json:"sleep"`
	Timeout       time.Duration      `json:"timeout"`
	Tries         int                `json:"tries"`
	StopWhenEmpty bool               `json:"stop_when_empty"`
	Middleware    []Middleware       `json:"-"`
	Logger        func(string, ...interface{}) `json:"-"`
}

// WorkerStats contains worker statistics
type WorkerStats struct {
	ID              string        `json:"id"`
	Queue           string        `json:"queue"`
	ProcessedJobs   int64         `json:"processed_jobs"`
	FailedJobs      int64         `json:"failed_jobs"`
	LastJobAt       *time.Time    `json:"last_job_at,omitempty"`
	AverageTime     time.Duration `json:"average_time"`
	TotalTime       time.Duration `json:"total_time"`
	StartedAt       time.Time     `json:"started_at"`
	IsRunning       bool          `json:"is_running"`
	CurrentJob      *JobPayload   `json:"current_job,omitempty"`
}

// QueueStats contains queue statistics
type QueueStats struct {
	Name        string    `json:"name"`
	Size        int       `json:"size"`
	Processing  int       `json:"processing"`
	Processed   int64     `json:"processed"`
	Failed      int64     `json:"failed"`
	LastJobAt   *time.Time `json:"last_job_at,omitempty"`
	TotalJobs   int64     `json:"total_jobs"`
	AverageTime time.Duration `json:"average_time"`
}

// BatchResult contains the result of a batch operation
type BatchResult struct {
	ID          string    `json:"id"`
	TotalJobs   int       `json:"total_jobs"`
	Successful  int       `json:"successful"`
	Failed      int       `json:"failed"`
	CreatedAt   time.Time `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	FailedJobs  []string  `json:"failed_jobs,omitempty"`
}

// Config represents queue configuration
type Config struct {
	Driver     string                 `json:"driver"`
	Connection string                 `json:"connection"`
	Queue      string                 `json:"queue"`
	Retry      int                    `json:"retry"`
	Timeout    time.Duration          `json:"timeout"`
	Options    map[string]interface{} `json:"options"`
	
	// Store-specific configuration
	Memory MemoryConfig `json:"memory"`
	Redis  RedisConfig  `json:"redis"`
	DB     DBConfig     `json:"database"`
}

// MemoryConfig represents memory queue configuration
type MemoryConfig struct {
	MaxSize int `json:"max_size"`
}

// RedisConfig represents Redis queue configuration
type RedisConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Password string `json:"password"`
	Database int    `json:"database"`
	Prefix   string `json:"prefix"`
}

// DBConfig represents database queue configuration
type DBConfig struct {
	Connection string `json:"connection"`
	Table      string `json:"table"`
}

// JobStatus represents the status of a job
type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
	JobStatusRetrying   JobStatus = "retrying"
)

// Event types for queue events
const (
	EventJobQueued    = "job.queued"
	EventJobStarted   = "job.started"
	EventJobCompleted = "job.completed"
	EventJobFailed    = "job.failed"
	EventJobRetrying  = "job.retrying"
	EventWorkerStarted = "worker.started"
	EventWorkerStopped = "worker.stopped"
	EventQueueCleared  = "queue.cleared"
)

// ErrorHandler handles job processing errors
type ErrorHandler interface {
	Handle(ctx context.Context, job Job, err error) error
}

// RetryPolicy determines when and how to retry failed jobs
type RetryPolicy interface {
	ShouldRetry(attempts int, maxTries int, err error) bool
	GetDelay(attempts int) time.Duration
}

// JobFactory creates jobs from payloads
type JobFactory interface {
	Create(payload *JobPayload) (Job, error)
	Register(jobType string, factory func(*JobPayload) (Job, error))
}