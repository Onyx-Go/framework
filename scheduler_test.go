package onyx

import (
	"fmt"
	"os"
	"sync"
	"testing"
	"time"
)

// Test task implementation
type TestTask struct {
	name        string
	executed    bool
	executionCount int
	shouldFail  bool
	mutex       sync.RWMutex
}

func (tt *TestTask) Handle() error {
	tt.mutex.Lock()
	defer tt.mutex.Unlock()
	
	tt.executed = true
	tt.executionCount++
	
	if tt.shouldFail {
		return fmt.Errorf("test task failed")
	}
	
	return nil
}

func (tt *TestTask) GetName() string {
	return tt.name
}

func (tt *TestTask) GetDescription() string {
	return fmt.Sprintf("Test task: %s", tt.name)
}

func (tt *TestTask) WasExecuted() bool {
	tt.mutex.RLock()
	defer tt.mutex.RUnlock()
	return tt.executed
}

func (tt *TestTask) GetExecutionCount() int {
	tt.mutex.RLock()
	defer tt.mutex.RUnlock()
	return tt.executionCount
}

func (tt *TestTask) Reset() {
	tt.mutex.Lock()
	defer tt.mutex.Unlock()
	tt.executed = false
	tt.executionCount = 0
}

func setupTestScheduler(t *testing.T) *Schedule {
	logManager := NewLogManager()
	console := NewConsoleDriver(true)
	logManager.AddChannel("console", console, InfoLevel)
	logger := logManager.Channel("console")
	
	schedule := NewSchedule(logger, nil)
	return schedule
}

func TestScheduleCreation(t *testing.T) {
	schedule := setupTestScheduler(t)
	
	if schedule == nil {
		t.Fatal("Schedule should not be nil")
	}
	
	if schedule.IsRunning() {
		t.Fatal("Schedule should not be running initially")
	}
	
	jobs := schedule.GetJobs()
	if len(jobs) != 0 {
		t.Fatalf("Expected 0 jobs, got %d", len(jobs))
	}
}

func TestScheduleCall(t *testing.T) {
	schedule := setupTestScheduler(t)
	
	job := schedule.Call(func() error {
		return nil
	}).EveryMinute()
	
	if job == nil {
		t.Fatal("Job should not be nil")
	}
	
	if job.GetExpression() != "0 * * * * *" {
		t.Fatalf("Expected expression '0 * * * * *', got '%s'", job.GetExpression())
	}
	
	jobs := schedule.GetJobs()
	if len(jobs) != 1 {
		t.Fatalf("Expected 1 job, got %d", len(jobs))
	}
}

func TestScheduleFrequencies(t *testing.T) {
	schedule := setupTestScheduler(t)
	
	tests := []struct {
		name       string
		method     func(*ScheduledJob) *ScheduledJob
		expression string
	}{
		{"EveryMinute", (*ScheduledJob).EveryMinute, "0 * * * * *"},
		{"EveryFiveMinutes", (*ScheduledJob).EveryFiveMinutes, "0 */5 * * * *"},
		{"Hourly", (*ScheduledJob).Hourly, "0 0 * * * *"},
		{"Daily", (*ScheduledJob).Daily, "0 0 0 * * *"},
		{"Weekly", (*ScheduledJob).Weekly, "0 0 0 * * 0"},
		{"Monthly", (*ScheduledJob).Monthly, "0 0 0 1 * *"},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			job := schedule.Call(func() error { return nil })
			test.method(job)
			
			if job.GetExpression() != test.expression {
				t.Fatalf("Expected expression '%s', got '%s'", test.expression, job.GetExpression())
			}
		})
	}
}

func TestScheduleDailyAt(t *testing.T) {
	schedule := setupTestScheduler(t)
	
	job := schedule.Call(func() error { return nil }).DailyAt("14:30")
	
	expected := "0 30 14 * * *"
	if job.GetExpression() != expected {
		t.Fatalf("Expected expression '%s', got '%s'", expected, job.GetExpression())
	}
}

func TestScheduleWeeklyOn(t *testing.T) {
	schedule := setupTestScheduler(t)
	
	job := schedule.Call(func() error { return nil }).WeeklyOn(1, "09:00") // Monday at 9 AM
	
	expected := "0 0 9 * * 1"
	if job.GetExpression() != expected {
		t.Fatalf("Expected expression '%s', got '%s'", expected, job.GetExpression())
	}
}

func TestScheduleMonthlyOn(t *testing.T) {
	schedule := setupTestScheduler(t)
	
	job := schedule.Call(func() error { return nil }).MonthlyOn(15, "12:00") // 15th at noon
	
	expected := "0 0 12 15 * *"
	if job.GetExpression() != expected {
		t.Fatalf("Expected expression '%s', got '%s'", expected, job.GetExpression())
	}
}

func TestScheduleCustomCron(t *testing.T) {
	schedule := setupTestScheduler(t)
	
	cronExpr := "0 30 2 * * *" // 2:30 AM daily
	job := schedule.Call(func() error { return nil }).Cron(cronExpr)
	
	if job.GetExpression() != cronExpr {
		t.Fatalf("Expected expression '%s', got '%s'", cronExpr, job.GetExpression())
	}
}

func TestScheduleConditions(t *testing.T) {
	schedule := setupTestScheduler(t)
	
	conditionMet := true
	job := schedule.Call(func() error { return nil }).
		EveryMinute().
		When(func() bool { return conditionMet })
	
	// Should run when condition is true
	if !job.shouldRun() {
		t.Fatal("Job should run when condition is true")
	}
	
	// Should not run when condition is false
	conditionMet = false
	if job.shouldRun() {
		t.Fatal("Job should not run when condition is false")
	}
}

func TestScheduleSkip(t *testing.T) {
	schedule := setupTestScheduler(t)
	
	skipCondition := false
	job := schedule.Call(func() error { return nil }).
		EveryMinute().
		Skip(func() bool { return skipCondition })
	
	// Should run when skip condition is false
	if !job.shouldRun() {
		t.Fatal("Job should run when skip condition is false")
	}
	
	// Should not run when skip condition is true
	skipCondition = true
	if job.shouldRun() {
		t.Fatal("Job should not run when skip condition is true")
	}
}

func TestScheduleEnvironments(t *testing.T) {
	schedule := setupTestScheduler(t)
	
	// Set environment
	originalEnv := os.Getenv("APP_ENV")
	defer os.Setenv("APP_ENV", originalEnv)
	
	os.Setenv("APP_ENV", "testing")
	
	job := schedule.Call(func() error { return nil }).
		EveryMinute().
		Environments("testing", "staging")
	
	// Should run in testing environment
	if !job.shouldRun() {
		t.Fatal("Job should run in testing environment")
	}
	
	// Set to production environment
	os.Setenv("APP_ENV", "production")
	
	// Should not run in production environment
	if job.shouldRun() {
		t.Fatal("Job should not run in production environment")
	}
}

func TestScheduleTimeConstraints(t *testing.T) {
	schedule := setupTestScheduler(t)
	
	// Test weekdays constraint
	job := schedule.Call(func() error { return nil }).
		EveryMinute().
		Weekdays()
	
	// Mock time to Tuesday (weekday)
	// Note: In a real test, you'd want to mock time.Now()
	// For this test, we'll just check the constraint exists
	if len(job.conditions) == 0 {
		t.Fatal("Weekdays constraint should add a condition")
	}
}

func TestScheduleHooks(t *testing.T) {
	schedule := setupTestScheduler(t)
	
	beforeCalled := false
	afterCalled := false
	successCalled := false
	failureCalled := false
	
	task := &TestTask{name: "hook_test", shouldFail: false}
	
	job := schedule.Call(task.Handle).
		EveryMinute().
		Before(func(job *ScheduledJob, err error) {
			beforeCalled = true
		}).
		After(func(job *ScheduledJob, err error) {
			afterCalled = true
		}).
		OnSuccess(func(job *ScheduledJob, err error) {
			successCalled = true
		}).
		OnFailure(func(job *ScheduledJob, err error) {
			failureCalled = true
		})
	
	// Execute the job manually
	schedule.executeJob(job)
	
	if !beforeCalled {
		t.Fatal("Before hook should be called")
	}
	
	if !afterCalled {
		t.Fatal("After hook should be called")
	}
	
	if !successCalled {
		t.Fatal("OnSuccess hook should be called for successful job")
	}
	
	if failureCalled {
		t.Fatal("OnFailure hook should not be called for successful job")
	}
}

func TestScheduleHooksOnFailure(t *testing.T) {
	schedule := setupTestScheduler(t)
	
	successCalled := false
	failureCalled := false
	
	task := &TestTask{name: "hook_test_fail", shouldFail: true}
	
	job := schedule.Call(task.Handle).
		EveryMinute().
		OnSuccess(func(job *ScheduledJob, err error) {
			successCalled = true
		}).
		OnFailure(func(job *ScheduledJob, err error) {
			failureCalled = true
		})
	
	// Execute the job manually
	schedule.executeJob(job)
	
	if successCalled {
		t.Fatal("OnSuccess hook should not be called for failed job")
	}
	
	if !failureCalled {
		t.Fatal("OnFailure hook should be called for failed job")
	}
}

func TestScheduleJobManagement(t *testing.T) {
	schedule := setupTestScheduler(t)
	
	task := &TestTask{name: "test_job"}
	job := schedule.Call(task.Handle).EveryMinute()
	
	// The closure task will have a generated name, so get it from the job
	jobName := job.task.GetName()
	
	// Get job
	retrievedJob, exists := schedule.GetJob(jobName)
	if !exists {
		t.Fatal("Job should exist")
	}
	
	if retrievedJob != job {
		t.Fatal("Retrieved job should be the same as created job")
	}
	
	// Remove job
	err := schedule.RemoveJob(jobName)
	if err != nil {
		t.Fatalf("Failed to remove job: %v", err)
	}
	
	// Job should no longer exist
	_, exists = schedule.GetJob(jobName)
	if exists {
		t.Fatal("Job should not exist after removal")
	}
}

func TestScheduleJobState(t *testing.T) {
	schedule := setupTestScheduler(t)
	
	task := &TestTask{name: "state_test"}
	job := schedule.Call(task.Handle).EveryMinute()
	
	// Check initial state
	if !job.IsEnabled() {
		t.Fatal("Job should be enabled by default")
	}
	
	if job.GetRunCount() != 0 {
		t.Fatal("Initial run count should be 0")
	}
	
	if job.GetFailCount() != 0 {
		t.Fatal("Initial fail count should be 0")
	}
	
	// Disable job
	job.Disable()
	if job.IsEnabled() {
		t.Fatal("Job should be disabled")
	}
	
	// Re-enable job
	job.Enable()
	if !job.IsEnabled() {
		t.Fatal("Job should be enabled")
	}
}

func TestScheduleStartStop(t *testing.T) {
	schedule := setupTestScheduler(t)
	
	// Add a job
	schedule.Call(func() error { return nil }).EveryMinute()
	
	// Start scheduler
	err := schedule.Start()
	if err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}
	
	if !schedule.IsRunning() {
		t.Fatal("Schedule should be running after start")
	}
	
	// Stop scheduler
	err = schedule.Stop()
	if err != nil {
		t.Fatalf("Failed to stop scheduler: %v", err)
	}
	
	if schedule.IsRunning() {
		t.Fatal("Schedule should not be running after stop")
	}
}

func TestScheduleDoubleStart(t *testing.T) {
	schedule := setupTestScheduler(t)
	
	// Start scheduler
	err := schedule.Start()
	if err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}
	defer schedule.Stop()
	
	// Try to start again
	err = schedule.Start()
	if err == nil {
		t.Fatal("Starting scheduler twice should return an error")
	}
}

func TestScheduleTaskTypes(t *testing.T) {
	schedule := setupTestScheduler(t)
	
	// Test Command task
	cmdJob := schedule.Command("echo", "hello")
	if cmdJob == nil {
		t.Fatal("Command job should not be nil")
	}
	
	// Test Exec task
	execJob := schedule.Exec("echo hello")
	if execJob == nil {
		t.Fatal("Exec job should not be nil")
	}
	
	// Test Job task (without queue manager)
	jobTask := &TestJob{data: "test"}
	queueJob := schedule.Job(jobTask)
	if queueJob == nil {
		t.Fatal("Queue job should not be nil")
	}
}

func TestScheduleRunDue(t *testing.T) {
	schedule := setupTestScheduler(t)
	
	task := &TestTask{name: "run_due_test"}
	job := schedule.Call(task.Handle).EveryMinute()
	
	// Mock that the job should run
	job.enabled = true
	
	// Run due jobs
	err := schedule.RunDue()
	if err != nil {
		t.Fatalf("RunDue should not return error: %v", err)
	}
	
	// Wait a moment for async execution
	time.Sleep(100 * time.Millisecond)
	
	// Check if task was executed
	if !task.WasExecuted() {
		t.Fatal("Task should have been executed")
	}
}

func TestApplicationSchedulerIntegration(t *testing.T) {
	app := New()
	
	schedule := app.Schedule()
	if schedule == nil {
		t.Fatal("Application schedule should not be nil")
	}
	
	// Add a test job
	schedule.Call(func() error {
		return nil
	}).EveryMinute()
	
	// Start scheduler
	err := app.StartScheduler()
	if err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}
	defer app.StopScheduler()
	
	if !schedule.IsRunning() {
		t.Fatal("Scheduler should be running")
	}
}

// Test job for queue integration
type TestJob struct {
	data string
}

func (tj *TestJob) Handle() error {
	// Simulate job processing
	return nil
}

func (tj *TestJob) Failed(err error) {
	// Handle job failure
}

func (tj *TestJob) GetPayload() map[string]interface{} {
	return map[string]interface{}{
		"data": tj.data,
	}
}

func (tj *TestJob) GetQueue() string {
	return "default"
}

func (tj *TestJob) GetDelay() time.Duration {
	return 0
}

func (tj *TestJob) GetMaxTries() int {
	return 3
}

func (tj *TestJob) GetTimeout() time.Duration {
	return 30 * time.Second
}

func TestScheduleWithQueue(t *testing.T) {
	// Create a simple queue manager
	queueManager := NewQueueManager()
	logManager := NewLogManager()
	console := NewConsoleDriver(true)
	logManager.AddChannel("console", console, InfoLevel)
	logger := logManager.Channel("console")
	
	schedule := NewSchedule(logger, queueManager)
	
	job := &TestJob{data: "test"}
	scheduledJob := schedule.Job(job).EveryMinute()
	
	if scheduledJob == nil {
		t.Fatal("Scheduled job should not be nil")
	}
	
	// Test that the job task references the queue manager
	queueTask, ok := scheduledJob.task.(*QueueJobTask)
	if !ok {
		t.Fatal("Task should be a QueueJobTask")
	}
	
	if queueTask.queueManager != queueManager {
		t.Fatal("Queue task should reference the queue manager")
	}
}