package onyx

import (
	"fmt"
	"sync"
	"time"
)

type Job interface {
	Handle() error
	Failed(error)
	GetPayload() map[string]interface{}
	GetQueue() string
	GetDelay() time.Duration
	GetMaxTries() int
	GetTimeout() time.Duration
}

type QueueableJob interface {
	Job
	OnQueue(queue string) QueueableJob
	Delay(delay time.Duration) QueueableJob
	OnConnection(connection string) QueueableJob
}

type Queue interface {
	Push(job Job) error
	PushOn(queue string, job Job) error
	Later(delay time.Duration, job Job) error
	LaterOn(queue string, delay time.Duration, job Job) error
	Pop(queue ...string) (Job, error)
	Size(queue ...string) int
	Clear(queue string) error
}

type QueueManager interface {
	Connection(name ...string) Queue
	Push(job Job) error
	PushOn(queue string, job Job) error
	Later(delay time.Duration, job Job) error
	LaterOn(queue string, delay time.Duration, job Job) error
}

type Worker interface {
	Work(queue Queue, options WorkerOptions)
	Stop()
	IsRunning() bool
}

type WorkerOptions struct {
	Queue      string
	MaxJobs    int
	MaxTime    time.Duration
	Memory     int
	Sleep      time.Duration
	Timeout    time.Duration
	Tries      int
	StopWhenEmpty bool
}

type JobPayload struct {
	ID          string                 `json:"id"`
	DisplayName string                 `json:"displayName"`
	Job         string                 `json:"job"`
	MaxTries    int                    `json:"maxTries"`
	Timeout     int                    `json:"timeout"`
	Data        map[string]interface{} `json:"data"`
	Queue       string                 `json:"queue"`
	Attempts    int                    `json:"attempts"`
	CreatedAt   time.Time              `json:"created_at"`
	AvailableAt time.Time              `json:"available_at"`
}

type BaseJob struct {
	queue     string
	delay     time.Duration
	maxTries  int
	timeout   time.Duration
	payload   map[string]interface{}
}

func NewBaseJob() *BaseJob {
	return &BaseJob{
		queue:    "default",
		maxTries: 3,
		timeout:  60 * time.Second,
		payload:  make(map[string]interface{}),
	}
}

func (bj *BaseJob) GetQueue() string {
	return bj.queue
}

func (bj *BaseJob) GetDelay() time.Duration {
	return bj.delay
}

func (bj *BaseJob) GetMaxTries() int {
	return bj.maxTries
}

func (bj *BaseJob) GetTimeout() time.Duration {
	return bj.timeout
}

func (bj *BaseJob) GetPayload() map[string]interface{} {
	return bj.payload
}

func (bj *BaseJob) OnQueue(queue string) QueueableJob {
	bj.queue = queue
	return bj
}

func (bj *BaseJob) Delay(delay time.Duration) QueueableJob {
	bj.delay = delay
	return bj
}

func (bj *BaseJob) OnConnection(connection string) QueueableJob {
	return bj
}

func (bj *BaseJob) Handle() error {
	return fmt.Errorf("handle method must be implemented")
}

func (bj *BaseJob) Failed(err error) {
	fmt.Printf("Job failed: %v\n", err)
}

type MemoryQueue struct {
	queues map[string][]*JobPayload
	mutex  sync.RWMutex
}

func NewMemoryQueue() *MemoryQueue {
	return &MemoryQueue{
		queues: make(map[string][]*JobPayload),
	}
}

func (mq *MemoryQueue) Push(job Job) error {
	return mq.PushOn(job.GetQueue(), job)
}

func (mq *MemoryQueue) PushOn(queue string, job Job) error {
	return mq.LaterOn(queue, 0, job)
}

func (mq *MemoryQueue) Later(delay time.Duration, job Job) error {
	return mq.LaterOn(job.GetQueue(), delay, job)
}

func (mq *MemoryQueue) LaterOn(queue string, delay time.Duration, job Job) error {
	payload := &JobPayload{
		ID:          generateJobID(),
		DisplayName: fmt.Sprintf("%T", job),
		Job:         fmt.Sprintf("%T", job),
		MaxTries:    job.GetMaxTries(),
		Timeout:     int(job.GetTimeout().Seconds()),
		Data:        job.GetPayload(),
		Queue:       queue,
		Attempts:    0,
		CreatedAt:   time.Now(),
		AvailableAt: time.Now().Add(delay),
	}

	mq.mutex.Lock()
	defer mq.mutex.Unlock()

	if _, exists := mq.queues[queue]; !exists {
		mq.queues[queue] = make([]*JobPayload, 0)
	}

	mq.queues[queue] = append(mq.queues[queue], payload)
	return nil
}

func (mq *MemoryQueue) Pop(queue ...string) (Job, error) {
	queueName := "default"
	if len(queue) > 0 {
		queueName = queue[0]
	}

	mq.mutex.Lock()
	defer mq.mutex.Unlock()

	jobs, exists := mq.queues[queueName]
	if !exists || len(jobs) == 0 {
		return nil, fmt.Errorf("no jobs available")
	}

	now := time.Now()
	for i, payload := range jobs {
		if payload.AvailableAt.Before(now) || payload.AvailableAt.Equal(now) {
			mq.queues[queueName] = append(jobs[:i], jobs[i+1:]...)
			return mq.createJobFromPayload(payload), nil
		}
	}

	return nil, fmt.Errorf("no jobs available")
}

func (mq *MemoryQueue) Size(queue ...string) int {
	queueName := "default"
	if len(queue) > 0 {
		queueName = queue[0]
	}

	mq.mutex.RLock()
	defer mq.mutex.RUnlock()

	if jobs, exists := mq.queues[queueName]; exists {
		return len(jobs)
	}
	return 0
}

func (mq *MemoryQueue) Clear(queue string) error {
	mq.mutex.Lock()
	defer mq.mutex.Unlock()

	delete(mq.queues, queue)
	return nil
}

func (mq *MemoryQueue) createJobFromPayload(payload *JobPayload) Job {
	job := &QueueJob{
		BaseJob: &BaseJob{
			queue:    payload.Queue,
			maxTries: payload.MaxTries,
			timeout:  time.Duration(payload.Timeout) * time.Second,
			payload:  payload.Data,
		},
		id:       payload.ID,
		attempts: payload.Attempts,
	}
	return job
}

type QueueJob struct {
	*BaseJob
	id       string
	attempts int
}

func (qj *QueueJob) Handle() error {
	if handler, exists := qj.payload["handler"]; exists {
		if handlerFunc, ok := handler.(func() error); ok {
			return handlerFunc()
		}
	}
	return fmt.Errorf("no handler found for job")
}

type QueueWorker struct {
	queue     Queue
	options   WorkerOptions
	running   bool
	stopCh    chan struct{}
	mutex     sync.RWMutex
}

func NewQueueWorker() *QueueWorker {
	return &QueueWorker{
		stopCh: make(chan struct{}),
	}
}

func (qw *QueueWorker) Work(queue Queue, options WorkerOptions) {
	qw.mutex.Lock()
	qw.queue = queue
	qw.options = options
	qw.running = true
	qw.mutex.Unlock()

	go qw.runWorker()
}

func (qw *QueueWorker) Stop() {
	qw.mutex.Lock()
	defer qw.mutex.Unlock()

	if qw.running {
		qw.running = false
		close(qw.stopCh)
	}
}

func (qw *QueueWorker) IsRunning() bool {
	qw.mutex.RLock()
	defer qw.mutex.RUnlock()
	return qw.running
}

func (qw *QueueWorker) runWorker() {
	ticker := time.NewTicker(qw.options.Sleep)
	defer ticker.Stop()

	processedJobs := 0
	startTime := time.Now()

	for {
		select {
		case <-qw.stopCh:
			return
		case <-ticker.C:
			if qw.shouldStop(processedJobs, startTime) {
				qw.Stop()
				return
			}

			job, err := qw.queue.Pop(qw.options.Queue)
			if err != nil {
				if qw.options.StopWhenEmpty {
					qw.Stop()
					return
				}
				continue
			}

			if qw.processJob(job) {
				processedJobs++
			}
		}
	}
}

func (qw *QueueWorker) shouldStop(processedJobs int, startTime time.Time) bool {
	if qw.options.MaxJobs > 0 && processedJobs >= qw.options.MaxJobs {
		return true
	}

	if qw.options.MaxTime > 0 && time.Since(startTime) >= qw.options.MaxTime {
		return true
	}

	return false
}

func (qw *QueueWorker) processJob(job Job) bool {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Job panicked: %v\n", r)
			job.Failed(fmt.Errorf("job panicked: %v", r))
		}
	}()

	timeout := job.GetTimeout()
	if timeout == 0 {
		timeout = qw.options.Timeout
	}

	done := make(chan error, 1)

	go func() {
		done <- job.Handle()
	}()

	select {
	case err := <-done:
		if err != nil {
			fmt.Printf("Job failed: %v\n", err)
			job.Failed(err)
			return false
		}
		return true
	case <-time.After(timeout):
		fmt.Printf("Job timed out after %v\n", timeout)
		job.Failed(fmt.Errorf("job timed out"))
		return false
	}
}

type DefaultQueueManager struct {
	connections map[string]Queue
	defaultConn string
}

func NewQueueManager() *DefaultQueueManager {
	return &DefaultQueueManager{
		connections: make(map[string]Queue),
		defaultConn: "memory",
	}
}

func (qm *DefaultQueueManager) Connection(name ...string) Queue {
	connName := qm.defaultConn
	if len(name) > 0 {
		connName = name[0]
	}

	if conn, exists := qm.connections[connName]; exists {
		return conn
	}

	return qm.createConnection(connName)
}

func (qm *DefaultQueueManager) createConnection(name string) Queue {
	var queue Queue

	switch name {
	case "memory":
		queue = NewMemoryQueue()
	default:
		queue = NewMemoryQueue()
	}

	qm.connections[name] = queue
	return queue
}

func (qm *DefaultQueueManager) Push(job Job) error {
	return qm.Connection().Push(job)
}

func (qm *DefaultQueueManager) PushOn(queue string, job Job) error {
	return qm.Connection().PushOn(queue, job)
}

func (qm *DefaultQueueManager) Later(delay time.Duration, job Job) error {
	return qm.Connection().Later(delay, job)
}

func (qm *DefaultQueueManager) LaterOn(queue string, delay time.Duration, job Job) error {
	return qm.Connection().LaterOn(queue, delay, job)
}

func (qm *DefaultQueueManager) RegisterConnection(name string, queue Queue) {
	qm.connections[name] = queue
}

func generateJobID() string {
	return fmt.Sprintf("job_%d", time.Now().UnixNano())
}

type SendEmailJob struct {
	*BaseJob
	To      string
	Subject string
	Body    string
}

func NewSendEmailJob(to, subject, body string) *SendEmailJob {
	return &SendEmailJob{
		BaseJob: NewBaseJob(),
		To:      to,
		Subject: subject,
		Body:    body,
	}
}

func (sej *SendEmailJob) Handle() error {
	fmt.Printf("Sending email to %s: %s\n", sej.To, sej.Subject)
	time.Sleep(100 * time.Millisecond)
	return nil
}

func (sej *SendEmailJob) Failed(err error) {
	fmt.Printf("Failed to send email to %s: %v\n", sej.To, err)
}

type ProcessImageJob struct {
	*BaseJob
	ImagePath string
}

func NewProcessImageJob(imagePath string) *ProcessImageJob {
	return &ProcessImageJob{
		BaseJob:   NewBaseJob(),
		ImagePath: imagePath,
	}
}

func (pij *ProcessImageJob) Handle() error {
	fmt.Printf("Processing image: %s\n", pij.ImagePath)
	time.Sleep(500 * time.Millisecond)
	return nil
}

func (pij *ProcessImageJob) Failed(err error) {
	fmt.Printf("Failed to process image %s: %v\n", pij.ImagePath, err)
}

func DispatchJob(job Job) error {
	queueManager, _ := globalApp.Container().Make("queue")
	if qm, ok := queueManager.(QueueManager); ok {
		return qm.Push(job)
	}
	return fmt.Errorf("queue manager not configured")
}

func DispatchJobOn(queue string, job Job) error {
	queueManager, _ := globalApp.Container().Make("queue")
	if qm, ok := queueManager.(QueueManager); ok {
		return qm.PushOn(queue, job)
	}
	return fmt.Errorf("queue manager not configured")
}

func DispatchJobLater(delay time.Duration, job Job) error {
	queueManager, _ := globalApp.Container().Make("queue")
	if qm, ok := queueManager.(QueueManager); ok {
		return qm.Later(delay, job)
	}
	return fmt.Errorf("queue manager not configured")
}

var globalApp *Application

func (app *Application) SetGlobal() {
	globalApp = app
}

func (c *Context) Dispatch(job Job) error {
	queueManager, _ := c.app.Container().Make("queue")
	if qm, ok := queueManager.(QueueManager); ok {
		return qm.Push(job)
	}
	return fmt.Errorf("queue manager not configured")
}

func (c *Context) DispatchOn(queue string, job Job) error {
	queueManager, _ := c.app.Container().Make("queue")
	if qm, ok := queueManager.(QueueManager); ok {
		return qm.PushOn(queue, job)
	}
	return fmt.Errorf("queue manager not configured")
}

func (c *Context) DispatchLater(delay time.Duration, job Job) error {
	queueManager, _ := c.app.Container().Make("queue")
	if qm, ok := queueManager.(QueueManager); ok {
		return qm.Later(delay, job)
	}
	return fmt.Errorf("queue manager not configured")
}