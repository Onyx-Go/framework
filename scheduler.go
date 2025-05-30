package onyx

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// SchedulableTask represents a task that can be scheduled
type SchedulableTask interface {
	Handle() error
	GetName() string
	GetDescription() string
}

// ScheduledJob wraps a task with scheduling information
type ScheduledJob struct {
	task        SchedulableTask
	expression  string
	timezone    *time.Location
	mutex       sync.RWMutex
	lastRun     time.Time
	nextRun     time.Time
	runCount    int64
	failCount   int64
	enabled     bool
	singleServer bool
	conditions  []ConditionFunc
	beforeHooks []HookFunc
	afterHooks  []HookFunc
	schedule    *Schedule
}

type ConditionFunc func() bool
type HookFunc func(job *ScheduledJob, err error)

// Schedule represents the main scheduler
type Schedule struct {
	cron        *cron.Cron
	jobs        map[string]*ScheduledJob
	mutex       sync.RWMutex
	logger      Logger
	queueManager QueueManager
	running     bool
	ctx         context.Context
	cancel      context.CancelFunc
	entryIds    map[string]cron.EntryID
}

// Scheduler interface for Laravel-style task scheduling
type Scheduler interface {
	// Task registration
	Call(fn func() error) *ScheduledJob
	Command(command string, args ...string) *ScheduledJob
	Job(job Job) *ScheduledJob
	Exec(command string, args ...string) *ScheduledJob

	// Lifecycle
	Start() error
	Stop() error
	IsRunning() bool

	// Management
	GetJobs() map[string]*ScheduledJob
	GetJob(name string) (*ScheduledJob, bool)
	RemoveJob(name string) error
	RunDue() error

	// Event hooks
	OnJobStart(hook HookFunc)
	OnJobComplete(hook HookFunc)
	OnJobError(hook HookFunc)
}

// NewSchedule creates a new scheduler instance
func NewSchedule(logger Logger, queueManager QueueManager) *Schedule {
	ctx, cancel := context.WithCancel(context.Background())

	schedule := &Schedule{
		jobs:         make(map[string]*ScheduledJob),
		queueManager: queueManager,
		logger:       logger,
		ctx:          ctx,
		cancel:       cancel,
		entryIds:     make(map[string]cron.EntryID),
	}

	// Configure cron with options
	schedule.cron = cron.New(
		cron.WithSeconds(),                          // Support seconds field
		cron.WithLocation(time.UTC),                 // Default to UTC
		cron.WithChain(                              // Add middleware
			cron.Recover(cron.DefaultLogger),        // Panic recovery
			cron.DelayIfStillRunning(cron.DefaultLogger), // Prevent overlapping
		),
	)

	return schedule
}

// Laravel-style fluent scheduling methods
func (s *Schedule) Call(fn func() error) *ScheduledJob {
	task := &ClosureTask{
		name: fmt.Sprintf("closure_%d", time.Now().UnixNano()),
		fn:   fn,
	}

	job := &ScheduledJob{
		task:        task,
		enabled:     true,
		timezone:    time.UTC,
		conditions:  make([]ConditionFunc, 0),
		beforeHooks: make([]HookFunc, 0),
		afterHooks:  make([]HookFunc, 0),
		schedule:    s,
	}

	s.mutex.Lock()
	s.jobs[task.GetName()] = job
	s.mutex.Unlock()

	return job
}

func (s *Schedule) Command(command string, args ...string) *ScheduledJob {
	task := &CommandTask{
		name:    fmt.Sprintf("command_%s", command),
		command: command,
		args:    args,
	}

	return s.registerTask(task)
}

func (s *Schedule) Job(job Job) *ScheduledJob {
	task := &QueueJobTask{
		name:         fmt.Sprintf("job_%T", job),
		job:          job,
		queueManager: s.queueManager,
	}

	return s.registerTask(task)
}

func (s *Schedule) Exec(command string, args ...string) *ScheduledJob {
	task := &ExecTask{
		name:    fmt.Sprintf("exec_%s", command),
		command: command,
		args:    args,
	}

	return s.registerTask(task)
}

func (s *Schedule) registerTask(task SchedulableTask) *ScheduledJob {
	job := &ScheduledJob{
		task:        task,
		enabled:     true,
		timezone:    time.UTC,
		conditions:  make([]ConditionFunc, 0),
		beforeHooks: make([]HookFunc, 0),
		afterHooks:  make([]HookFunc, 0),
		schedule:    s,
	}

	s.mutex.Lock()
	s.jobs[task.GetName()] = job
	s.mutex.Unlock()

	return job
}

// Start the scheduler
func (s *Schedule) Start() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.running {
		return fmt.Errorf("scheduler is already running")
	}

	// Add all jobs to cron
	for _, job := range s.jobs {
		if job.enabled && job.expression != "" {
			entryID, err := s.cron.AddFunc(job.expression, s.wrapJobExecution(job))
			if err != nil {
				s.logger.Error("Failed to add job to cron", map[string]interface{}{
					"job":   job.task.GetName(),
					"error": err,
				})
				continue
			}
			s.entryIds[job.task.GetName()] = entryID
		}
	}

	s.cron.Start()
	s.running = true

	s.logger.Info("Task scheduler started", map[string]interface{}{
		"jobs": len(s.jobs),
	})

	return nil
}

// Stop the scheduler
func (s *Schedule) Stop() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.running {
		return nil
	}

	ctx := s.cron.Stop()
	select {
	case <-ctx.Done():
		s.logger.Info("Task scheduler stopped gracefully", nil)
	case <-time.After(30 * time.Second):
		s.logger.Warn("Task scheduler stop timeout, forcing shutdown", nil)
	}

	s.running = false
	s.cancel()

	return nil
}

// IsRunning returns whether the scheduler is running
func (s *Schedule) IsRunning() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.running
}

// GetJobs returns all scheduled jobs
func (s *Schedule) GetJobs() map[string]*ScheduledJob {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	jobs := make(map[string]*ScheduledJob)
	for name, job := range s.jobs {
		jobs[name] = job
	}
	return jobs
}

// GetJob returns a specific job by name
func (s *Schedule) GetJob(name string) (*ScheduledJob, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	job, exists := s.jobs[name]
	return job, exists
}

// RemoveJob removes a job from the scheduler
func (s *Schedule) RemoveJob(name string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if entryID, exists := s.entryIds[name]; exists {
		s.cron.Remove(entryID)
		delete(s.entryIds, name)
	}

	delete(s.jobs, name)
	return nil
}

// RunDue runs all jobs that are due (for command-line execution)
func (s *Schedule) RunDue() error {
	s.mutex.RLock()
	jobs := make([]*ScheduledJob, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, job)
	}
	s.mutex.RUnlock()

	for _, job := range jobs {
		if job.enabled && job.shouldRun() {
			go s.executeJob(job)
		}
	}

	return nil
}

// OnJobStart, OnJobComplete, OnJobError add global hooks
func (s *Schedule) OnJobStart(hook HookFunc) {
	// Implementation would add global hooks to all jobs
}

func (s *Schedule) OnJobComplete(hook HookFunc) {
	// Implementation would add global hooks to all jobs
}

func (s *Schedule) OnJobError(hook HookFunc) {
	// Implementation would add global hooks to all jobs
}

// wrapJobExecution wraps job execution with logging and error handling
func (s *Schedule) wrapJobExecution(job *ScheduledJob) func() {
	return func() {
		s.executeJob(job)
	}
}

// executeJob executes a job with all hooks and error handling
func (s *Schedule) executeJob(job *ScheduledJob) {
	if !job.shouldRun() {
		return
	}

	job.mutex.Lock()
	job.lastRun = time.Now()
	job.runCount++
	job.mutex.Unlock()

	// Execute before hooks
	for _, hook := range job.beforeHooks {
		hook(job, nil)
	}

	// Log job start
	s.logger.Info("Starting scheduled job", map[string]interface{}{
		"job":         job.task.GetName(),
		"description": job.task.GetDescription(),
		"run_count":   job.runCount,
	})

	// Execute the task
	var err error
	start := time.Now()

	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic in scheduled job: %v", r)
			}
		}()
		err = job.task.Handle()
	}()

	duration := time.Since(start)

	// Update statistics
	job.mutex.Lock()
	if err != nil {
		job.failCount++
	}
	job.mutex.Unlock()

	// Log job completion
	if err != nil {
		s.logger.Error("Scheduled job failed", map[string]interface{}{
			"job":         job.task.GetName(),
			"error":       err,
			"duration_ms": duration.Milliseconds(),
			"fail_count":  job.failCount,
		})
	} else {
		s.logger.Info("Scheduled job completed", map[string]interface{}{
			"job":         job.task.GetName(),
			"duration_ms": duration.Milliseconds(),
			"run_count":   job.runCount,
		})
	}

	// Execute after hooks
	for _, hook := range job.afterHooks {
		hook(job, err)
	}
}

// shouldRun checks if a job should run based on conditions
func (sj *ScheduledJob) shouldRun() bool {
	if !sj.enabled {
		return false
	}

	// Check all conditions
	for _, condition := range sj.conditions {
		if !condition() {
			return false
		}
	}

	return true
}

// Frequency methods for ScheduledJob (Laravel-style)
func (sj *ScheduledJob) EveryMinute() *ScheduledJob {
	sj.expression = "0 * * * * *"
	return sj
}

func (sj *ScheduledJob) EveryTwoMinutes() *ScheduledJob {
	sj.expression = "0 */2 * * * *"
	return sj
}

func (sj *ScheduledJob) EveryThreeMinutes() *ScheduledJob {
	sj.expression = "0 */3 * * * *"
	return sj
}

func (sj *ScheduledJob) EveryFourMinutes() *ScheduledJob {
	sj.expression = "0 */4 * * * *"
	return sj
}

func (sj *ScheduledJob) EveryFiveMinutes() *ScheduledJob {
	sj.expression = "0 */5 * * * *"
	return sj
}

func (sj *ScheduledJob) EveryTenMinutes() *ScheduledJob {
	sj.expression = "0 */10 * * * *"
	return sj
}

func (sj *ScheduledJob) EveryFifteenMinutes() *ScheduledJob {
	sj.expression = "0 */15 * * * *"
	return sj
}

func (sj *ScheduledJob) EveryThirtyMinutes() *ScheduledJob {
	sj.expression = "0 */30 * * * *"
	return sj
}

func (sj *ScheduledJob) Hourly() *ScheduledJob {
	sj.expression = "0 0 * * * *"
	return sj
}

func (sj *ScheduledJob) HourlyAt(minute int) *ScheduledJob {
	sj.expression = fmt.Sprintf("0 %d * * * *", minute)
	return sj
}

func (sj *ScheduledJob) EveryTwoHours() *ScheduledJob {
	sj.expression = "0 0 */2 * * *"
	return sj
}

func (sj *ScheduledJob) EveryThreeHours() *ScheduledJob {
	sj.expression = "0 0 */3 * * *"
	return sj
}

func (sj *ScheduledJob) EveryFourHours() *ScheduledJob {
	sj.expression = "0 0 */4 * * *"
	return sj
}

func (sj *ScheduledJob) EverySixHours() *ScheduledJob {
	sj.expression = "0 0 */6 * * *"
	return sj
}

func (sj *ScheduledJob) EveryTwelveHours() *ScheduledJob {
	sj.expression = "0 0 */12 * * *"
	return sj
}

func (sj *ScheduledJob) Daily() *ScheduledJob {
	sj.expression = "0 0 0 * * *"
	return sj
}

func (sj *ScheduledJob) DailyAt(timeStr string) *ScheduledJob {
	// Parse time string like "14:30" or "2:30"
	var hour, minute int
	if _, err := fmt.Sscanf(timeStr, "%d:%d", &hour, &minute); err == nil {
		sj.expression = fmt.Sprintf("0 %d %d * * *", minute, hour)
	}
	return sj
}

func (sj *ScheduledJob) Twicedaily(first, second int) *ScheduledJob {
	sj.expression = fmt.Sprintf("0 0 %d,%d * * *", first, second)
	return sj
}

func (sj *ScheduledJob) Weekly() *ScheduledJob {
	sj.expression = "0 0 0 * * 0" // Sunday at midnight
	return sj
}

func (sj *ScheduledJob) WeeklyOn(day int, timeStr string) *ScheduledJob {
	var hour, minute int
	if _, err := fmt.Sscanf(timeStr, "%d:%d", &hour, &minute); err == nil {
		sj.expression = fmt.Sprintf("0 %d %d * * %d", minute, hour, day)
	}
	return sj
}

func (sj *ScheduledJob) Monthly() *ScheduledJob {
	sj.expression = "0 0 0 1 * *" // First day of month
	return sj
}

func (sj *ScheduledJob) MonthlyOn(day int, timeStr string) *ScheduledJob {
	var hour, minute int
	if _, err := fmt.Sscanf(timeStr, "%d:%d", &hour, &minute); err == nil {
		sj.expression = fmt.Sprintf("0 %d %d %d * *", minute, hour, day)
	}
	return sj
}

func (sj *ScheduledJob) Quarterly() *ScheduledJob {
	sj.expression = "0 0 0 1 */3 *" // First day of quarter
	return sj
}

func (sj *ScheduledJob) Yearly() *ScheduledJob {
	sj.expression = "0 0 0 1 1 *" // January 1st
	return sj
}

func (sj *ScheduledJob) YearlyOn(month, day int, timeStr string) *ScheduledJob {
	var hour, minute int
	if _, err := fmt.Sscanf(timeStr, "%d:%d", &hour, &minute); err == nil {
		sj.expression = fmt.Sprintf("0 %d %d %d %d *", minute, hour, day, month)
	}
	return sj
}

func (sj *ScheduledJob) Cron(expression string) *ScheduledJob {
	sj.expression = expression
	return sj
}

// Constraint methods
func (sj *ScheduledJob) Timezone(tz string) *ScheduledJob {
	if location, err := time.LoadLocation(tz); err == nil {
		sj.timezone = location
	}
	return sj
}

func (sj *ScheduledJob) When(condition ConditionFunc) *ScheduledJob {
	sj.conditions = append(sj.conditions, condition)
	return sj
}

func (sj *ScheduledJob) Skip(condition ConditionFunc) *ScheduledJob {
	sj.conditions = append(sj.conditions, func() bool {
		return !condition()
	})
	return sj
}

func (sj *ScheduledJob) OnOneServer() *ScheduledJob {
	sj.singleServer = true
	return sj
}

func (sj *ScheduledJob) Environments(envs ...string) *ScheduledJob {
	return sj.When(func() bool {
		currentEnv := os.Getenv("APP_ENV")
		if currentEnv == "" {
			currentEnv = "development"
		}
		for _, env := range envs {
			if env == currentEnv {
				return true
			}
		}
		return false
	})
}

func (sj *ScheduledJob) Between(start, end string) *ScheduledJob {
	return sj.When(func() bool {
		now := time.Now()
		currentTime := fmt.Sprintf("%02d:%02d", now.Hour(), now.Minute())
		return currentTime >= start && currentTime <= end
	})
}

func (sj *ScheduledJob) Unlessbetween(start, end string) *ScheduledJob {
	return sj.When(func() bool {
		now := time.Now()
		currentTime := fmt.Sprintf("%02d:%02d", now.Hour(), now.Minute())
		return !(currentTime >= start && currentTime <= end)
	})
}

// Day constraints
func (sj *ScheduledJob) Weekdays() *ScheduledJob {
	return sj.When(func() bool {
		weekday := time.Now().Weekday()
		return weekday >= time.Monday && weekday <= time.Friday
	})
}

func (sj *ScheduledJob) Weekends() *ScheduledJob {
	return sj.When(func() bool {
		weekday := time.Now().Weekday()
		return weekday == time.Saturday || weekday == time.Sunday
	})
}

func (sj *ScheduledJob) Sundays() *ScheduledJob {
	return sj.When(func() bool {
		return time.Now().Weekday() == time.Sunday
	})
}

func (sj *ScheduledJob) Mondays() *ScheduledJob {
	return sj.When(func() bool {
		return time.Now().Weekday() == time.Monday
	})
}

func (sj *ScheduledJob) Tuesdays() *ScheduledJob {
	return sj.When(func() bool {
		return time.Now().Weekday() == time.Tuesday
	})
}

func (sj *ScheduledJob) Wednesdays() *ScheduledJob {
	return sj.When(func() bool {
		return time.Now().Weekday() == time.Wednesday
	})
}

func (sj *ScheduledJob) Thursdays() *ScheduledJob {
	return sj.When(func() bool {
		return time.Now().Weekday() == time.Thursday
	})
}

func (sj *ScheduledJob) Fridays() *ScheduledJob {
	return sj.When(func() bool {
		return time.Now().Weekday() == time.Friday
	})
}

func (sj *ScheduledJob) Saturdays() *ScheduledJob {
	return sj.When(func() bool {
		return time.Now().Weekday() == time.Saturday
	})
}

// Hook methods
func (sj *ScheduledJob) Before(hook HookFunc) *ScheduledJob {
	sj.beforeHooks = append(sj.beforeHooks, hook)
	return sj
}

func (sj *ScheduledJob) After(hook HookFunc) *ScheduledJob {
	sj.afterHooks = append(sj.afterHooks, hook)
	return sj
}

func (sj *ScheduledJob) OnSuccess(hook HookFunc) *ScheduledJob {
	return sj.After(func(job *ScheduledJob, err error) {
		if err == nil {
			hook(job, err)
		}
	})
}

func (sj *ScheduledJob) OnFailure(hook HookFunc) *ScheduledJob {
	return sj.After(func(job *ScheduledJob, err error) {
		if err != nil {
			hook(job, err)
		}
	})
}

// State methods
func (sj *ScheduledJob) Enable() *ScheduledJob {
	sj.enabled = true
	return sj
}

func (sj *ScheduledJob) Disable() *ScheduledJob {
	sj.enabled = false
	return sj
}

func (sj *ScheduledJob) IsEnabled() bool {
	return sj.enabled
}

func (sj *ScheduledJob) GetRunCount() int64 {
	sj.mutex.RLock()
	defer sj.mutex.RUnlock()
	return sj.runCount
}

func (sj *ScheduledJob) GetFailCount() int64 {
	sj.mutex.RLock()
	defer sj.mutex.RUnlock()
	return sj.failCount
}

func (sj *ScheduledJob) GetLastRun() time.Time {
	sj.mutex.RLock()
	defer sj.mutex.RUnlock()
	return sj.lastRun
}

func (sj *ScheduledJob) GetNextRun() time.Time {
	sj.mutex.RLock()
	defer sj.mutex.RUnlock()
	return sj.nextRun
}

func (sj *ScheduledJob) GetExpression() string {
	return sj.expression
}

// Task implementation types

// ClosureTask executes a closure function
type ClosureTask struct {
	name string
	fn   func() error
}

func (ct *ClosureTask) Handle() error {
	return ct.fn()
}

func (ct *ClosureTask) GetName() string {
	return ct.name
}

func (ct *ClosureTask) GetDescription() string {
	return fmt.Sprintf("Closure task: %s", ct.name)
}

// CommandTask executes system commands
type CommandTask struct {
	name    string
	command string
	args    []string
}

func (ct *CommandTask) Handle() error {
	cmd := exec.Command(ct.command, ct.args...)
	return cmd.Run()
}

func (ct *CommandTask) GetName() string {
	return ct.name
}

func (ct *CommandTask) GetDescription() string {
	return fmt.Sprintf("Command: %s %v", ct.command, ct.args)
}

// QueueJobTask dispatches jobs to the queue
type QueueJobTask struct {
	name         string
	job          Job
	queueManager QueueManager
}

func (qjt *QueueJobTask) Handle() error {
	if qjt.queueManager == nil {
		return fmt.Errorf("queue manager not configured")
	}
	return qjt.queueManager.Push(qjt.job)
}

func (qjt *QueueJobTask) GetName() string {
	return qjt.name
}

func (qjt *QueueJobTask) GetDescription() string {
	return fmt.Sprintf("Queue job: %T", qjt.job)
}

// ExecTask executes shell commands
type ExecTask struct {
	name    string
	command string
	args    []string
}

func (et *ExecTask) Handle() error {
	cmd := exec.Command("sh", "-c", et.command+" "+strings.Join(et.args, " "))
	return cmd.Run()
}

func (et *ExecTask) GetName() string {
	return et.name
}

func (et *ExecTask) GetDescription() string {
	return fmt.Sprintf("Exec: %s %v", et.command, et.args)
}