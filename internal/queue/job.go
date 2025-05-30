package queue

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// BaseJob provides a basic implementation of the Job interface
type BaseJob struct {
	queue     string
	delay     time.Duration
	maxTries  int
	timeout   time.Duration
	priority  Priority
	payload   map[string]interface{}
	metadata  map[string]interface{}
	mutex     sync.RWMutex
}

// NewBaseJob creates a new BaseJob with defaults
func NewBaseJob() *BaseJob {
	return &BaseJob{
		queue:    "default",
		maxTries: 3,
		timeout:  60 * time.Second,
		priority: PriorityNormal,
		payload:  make(map[string]interface{}),
		metadata: make(map[string]interface{}),
	}
}

// Handle provides a default implementation that must be overridden
func (bj *BaseJob) Handle(ctx context.Context) error {
	return fmt.Errorf("handle method must be implemented")
}

// Failed provides a default error handler
func (bj *BaseJob) Failed(ctx context.Context, err error) error {
	// Default implementation - log the error
	if logger, ok := bj.metadata["logger"].(func(string, ...interface{})); ok {
		logger("Job failed: %v", err)
	}
	return nil
}

// GetQueue returns the queue name
func (bj *BaseJob) GetQueue() string {
	bj.mutex.RLock()
	defer bj.mutex.RUnlock()
	return bj.queue
}

// GetDelay returns the delay before processing
func (bj *BaseJob) GetDelay() time.Duration {
	bj.mutex.RLock()
	defer bj.mutex.RUnlock()
	return bj.delay
}

// GetMaxTries returns the maximum number of attempts
func (bj *BaseJob) GetMaxTries() int {
	bj.mutex.RLock()
	defer bj.mutex.RUnlock()
	return bj.maxTries
}

// GetTimeout returns the maximum execution time
func (bj *BaseJob) GetTimeout() time.Duration {
	bj.mutex.RLock()
	defer bj.mutex.RUnlock()
	return bj.timeout
}

// GetPriority returns the job priority
func (bj *BaseJob) GetPriority() Priority {
	bj.mutex.RLock()
	defer bj.mutex.RUnlock()
	return bj.priority
}

// GetPayload returns the job's data payload
func (bj *BaseJob) GetPayload() map[string]interface{} {
	bj.mutex.RLock()
	defer bj.mutex.RUnlock()
	
	// Return a copy to prevent race conditions
	result := make(map[string]interface{})
	for k, v := range bj.payload {
		result[k] = v
	}
	return result
}

// GetMetadata returns job metadata
func (bj *BaseJob) GetMetadata() map[string]interface{} {
	bj.mutex.RLock()
	defer bj.mutex.RUnlock()
	
	// Return a copy to prevent race conditions
	result := make(map[string]interface{})
	for k, v := range bj.metadata {
		result[k] = v
	}
	return result
}

// OnQueue sets the queue name
func (bj *BaseJob) OnQueue(queue string) QueueableJob {
	bj.mutex.Lock()
	defer bj.mutex.Unlock()
	bj.queue = queue
	return bj
}

// Delay sets the delay before processing
func (bj *BaseJob) Delay(delay time.Duration) QueueableJob {
	bj.mutex.Lock()
	defer bj.mutex.Unlock()
	bj.delay = delay
	return bj
}

// OnConnection sets the connection (not used in BaseJob, but required by interface)
func (bj *BaseJob) OnConnection(connection string) QueueableJob {
	bj.mutex.Lock()
	defer bj.mutex.Unlock()
	bj.metadata["connection"] = connection
	return bj
}

// WithPriority sets the job priority
func (bj *BaseJob) WithPriority(priority Priority) QueueableJob {
	bj.mutex.Lock()
	defer bj.mutex.Unlock()
	bj.priority = priority
	return bj
}

// WithTimeout sets the job timeout
func (bj *BaseJob) WithTimeout(timeout time.Duration) QueueableJob {
	bj.mutex.Lock()
	defer bj.mutex.Unlock()
	bj.timeout = timeout
	return bj
}

// WithMaxTries sets the maximum number of attempts
func (bj *BaseJob) WithMaxTries(maxTries int) QueueableJob {
	bj.mutex.Lock()
	defer bj.mutex.Unlock()
	bj.maxTries = maxTries
	return bj
}

// WithMetadata sets a metadata value
func (bj *BaseJob) WithMetadata(key string, value interface{}) QueueableJob {
	bj.mutex.Lock()
	defer bj.mutex.Unlock()
	bj.metadata[key] = value
	return bj
}

// SetPayload sets the job payload data
func (bj *BaseJob) SetPayload(payload map[string]interface{}) {
	bj.mutex.Lock()
	defer bj.mutex.Unlock()
	bj.payload = payload
}

// SetPayloadValue sets a specific payload value
func (bj *BaseJob) SetPayloadValue(key string, value interface{}) {
	bj.mutex.Lock()
	defer bj.mutex.Unlock()
	bj.payload[key] = value
}

// GetPayloadValue gets a specific payload value
func (bj *BaseJob) GetPayloadValue(key string) (interface{}, bool) {
	bj.mutex.RLock()
	defer bj.mutex.RUnlock()
	value, exists := bj.payload[key]
	return value, exists
}

// GetMetadataValue gets a specific metadata value
func (bj *BaseJob) GetMetadataValue(key string) (interface{}, bool) {
	bj.mutex.RLock()
	defer bj.mutex.RUnlock()
	value, exists := bj.metadata[key]
	return value, exists
}

// Clone creates a copy of the job
func (bj *BaseJob) Clone() *BaseJob {
	bj.mutex.RLock()
	defer bj.mutex.RUnlock()
	
	clone := &BaseJob{
		queue:    bj.queue,
		delay:    bj.delay,
		maxTries: bj.maxTries,
		timeout:  bj.timeout,
		priority: bj.priority,
		payload:  make(map[string]interface{}),
		metadata: make(map[string]interface{}),
	}
	
	// Copy payload
	for k, v := range bj.payload {
		clone.payload[k] = v
	}
	
	// Copy metadata
	for k, v := range bj.metadata {
		clone.metadata[k] = v
	}
	
	return clone
}

// QueueJob wraps a job with additional queue-specific information
type QueueJob struct {
	*BaseJob
	id       string
	attempts int
	status   JobStatus
	lastError string
	createdAt time.Time
	processedAt *time.Time
	failedAt    *time.Time
}

// NewQueueJob creates a new QueueJob
func NewQueueJob(baseJob *BaseJob, id string) *QueueJob {
	return &QueueJob{
		BaseJob:   baseJob,
		id:        id,
		attempts:  0,
		status:    JobStatusPending,
		createdAt: time.Now(),
	}
}

// GetID returns the job ID
func (qj *QueueJob) GetID() string {
	return qj.id
}

// GetAttempts returns the number of attempts
func (qj *QueueJob) GetAttempts() int {
	return qj.attempts
}

// GetStatus returns the current job status
func (qj *QueueJob) GetStatus() JobStatus {
	return qj.status
}

// GetLastError returns the last error message
func (qj *QueueJob) GetLastError() string {
	return qj.lastError
}

// GetCreatedAt returns when the job was created
func (qj *QueueJob) GetCreatedAt() time.Time {
	return qj.createdAt
}

// GetProcessedAt returns when the job was last processed
func (qj *QueueJob) GetProcessedAt() *time.Time {
	return qj.processedAt
}

// GetFailedAt returns when the job failed
func (qj *QueueJob) GetFailedAt() *time.Time {
	return qj.failedAt
}

// MarkAsStarted marks the job as being processed
func (qj *QueueJob) MarkAsStarted() {
	qj.status = JobStatusProcessing
	now := time.Now()
	qj.processedAt = &now
}

// MarkAsCompleted marks the job as successfully completed
func (qj *QueueJob) MarkAsCompleted() {
	qj.status = JobStatusCompleted
}

// MarkAsFailed marks the job as failed
func (qj *QueueJob) MarkAsFailed(err error) {
	qj.status = JobStatusFailed
	qj.lastError = err.Error()
	now := time.Now()
	qj.failedAt = &now
	qj.attempts++
}

// MarkAsRetrying marks the job for retry
func (qj *QueueJob) MarkAsRetrying() {
	qj.status = JobStatusRetrying
	qj.attempts++
}

// ShouldRetry determines if the job should be retried
func (qj *QueueJob) ShouldRetry() bool {
	return qj.attempts < qj.GetMaxTries()
}

// ToPayload converts the job to a JobPayload for serialization
func (qj *QueueJob) ToPayload() *JobPayload {
	return &JobPayload{
		ID:          qj.id,
		DisplayName: fmt.Sprintf("%T", qj),
		Job:         fmt.Sprintf("%T", qj),
		MaxTries:    qj.GetMaxTries(),
		Timeout:     int(qj.GetTimeout().Seconds()),
		Priority:    qj.GetPriority(),
		Data:        qj.GetPayload(),
		Metadata:    qj.GetMetadata(),
		Queue:       qj.GetQueue(),
		Attempts:    qj.attempts,
		CreatedAt:   qj.createdAt,
		AvailableAt: time.Now().Add(qj.GetDelay()),
		ProcessedAt: qj.processedAt,
		FailedAt:    qj.failedAt,
		LastError:   qj.lastError,
	}
}

// Common job implementations

// CallbackJob executes a callback function
type CallbackJob struct {
	*BaseJob
	callback func(ctx context.Context) error
	name     string
}

// NewCallbackJob creates a new callback job
func NewCallbackJob(name string, callback func(ctx context.Context) error) *CallbackJob {
	baseJob := NewBaseJob()
	baseJob.SetPayloadValue("name", name)
	return &CallbackJob{
		BaseJob:  baseJob,
		callback: callback,
		name:     name,
	}
}

// Handle executes the callback
func (cj *CallbackJob) Handle(ctx context.Context) error {
	if cj.callback == nil {
		return fmt.Errorf("no callback function provided")
	}
	return cj.callback(ctx)
}

// String returns a string representation of the job
func (cj *CallbackJob) String() string {
	return fmt.Sprintf("CallbackJob[%s]", cj.name)
}

// CommandJob executes a system command
type CommandJob struct {
	*BaseJob
	Command string
	Args    []string
}

// NewCommandJob creates a new command job
func NewCommandJob(command string, args ...string) *CommandJob {
	job := &CommandJob{
		BaseJob: NewBaseJob(),
		Command: command,
		Args:    args,
	}
	job.SetPayloadValue("command", command)
	job.SetPayloadValue("args", args)
	return job
}

// Handle executes the command
func (cj *CommandJob) Handle(ctx context.Context) error {
	// This would typically use exec.CommandContext
	return fmt.Errorf("command execution not implemented in base job")
}

// String returns a string representation of the job
func (cj *CommandJob) String() string {
	return fmt.Sprintf("CommandJob[%s %v]", cj.Command, cj.Args)
}

// JobBuilder provides a fluent interface for building jobs
type JobBuilder struct {
	job *BaseJob
}

// NewJobBuilder creates a new job builder
func NewJobBuilder() *JobBuilder {
	return &JobBuilder{
		job: NewBaseJob(),
	}
}

// OnQueue sets the queue name
func (jb *JobBuilder) OnQueue(queue string) *JobBuilder {
	jb.job.OnQueue(queue)
	return jb
}

// WithDelay sets the processing delay
func (jb *JobBuilder) WithDelay(delay time.Duration) *JobBuilder {
	jb.job.Delay(delay)
	return jb
}

// WithPriority sets the job priority
func (jb *JobBuilder) WithPriority(priority Priority) *JobBuilder {
	jb.job.WithPriority(priority)
	return jb
}

// WithTimeout sets the job timeout
func (jb *JobBuilder) WithTimeout(timeout time.Duration) *JobBuilder {
	jb.job.WithTimeout(timeout)
	return jb
}

// WithMaxTries sets the maximum attempts
func (jb *JobBuilder) WithMaxTries(maxTries int) *JobBuilder {
	jb.job.WithMaxTries(maxTries)
	return jb
}

// WithPayload sets the job payload
func (jb *JobBuilder) WithPayload(payload map[string]interface{}) *JobBuilder {
	jb.job.SetPayload(payload)
	return jb
}

// WithMetadata sets job metadata
func (jb *JobBuilder) WithMetadata(key string, value interface{}) *JobBuilder {
	jb.job.WithMetadata(key, value)
	return jb
}

// Build returns the constructed job
func (jb *JobBuilder) Build() *BaseJob {
	return jb.job
}

// GenerateJobID generates a unique job ID
func GenerateJobID() string {
	return fmt.Sprintf("job_%d_%d", time.Now().Unix(), time.Now().Nanosecond())
}