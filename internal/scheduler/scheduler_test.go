package scheduler

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/itsmrshow/bulwark/internal/logging"
)

// fakeJob implements Job for testing.
type fakeJob struct {
	name      string
	execFn    func(ctx context.Context) error
	execCount atomic.Int32
}

func (j *fakeJob) Name() string { return j.name }

func (j *fakeJob) Execute(ctx context.Context) error {
	j.execCount.Add(1)
	if j.execFn != nil {
		return j.execFn(ctx)
	}
	return nil
}

func newFakeJob(name string) *fakeJob {
	return &fakeJob{name: name}
}

func testLogger() *logging.Logger {
	return logging.New(logging.Config{Level: "error"})
}

func TestNewScheduler(t *testing.T) {
	s := NewScheduler(testLogger())
	if s == nil {
		t.Fatal("expected non-nil scheduler")
	}
	if len(s.jobs) != 0 {
		t.Errorf("expected 0 jobs, got %d", len(s.jobs))
	}
}

func TestAddJob_Valid(t *testing.T) {
	s := NewScheduler(testLogger())
	job := newFakeJob("test-job")

	err := s.AddJob("*/5 * * * *", job)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	jobs := s.GetJobs()
	if len(jobs) != 1 {
		t.Errorf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0] != "test-job" {
		t.Errorf("expected job name test-job, got %s", jobs[0])
	}
}

func TestAddJob_InvalidCron(t *testing.T) {
	s := NewScheduler(testLogger())
	job := newFakeJob("bad-cron")

	err := s.AddJob("not-a-cron", job)
	if err == nil {
		t.Error("expected error for invalid cron expression")
	}
}

func TestAddJob_MultipleJobs(t *testing.T) {
	s := NewScheduler(testLogger())

	err := s.AddJob("*/5 * * * *", newFakeJob("job-a"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = s.AddJob("0 9 * * *", newFakeJob("job-b"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	jobs := s.GetJobs()
	if len(jobs) != 2 {
		t.Errorf("expected 2 jobs, got %d", len(jobs))
	}
}

func TestStartStop(t *testing.T) {
	s := NewScheduler(testLogger())
	_ = s.AddJob("*/5 * * * *", newFakeJob("test"))

	s.Start()
	// Should not panic on stop
	s.Stop()
}

func TestGetJobs_Empty(t *testing.T) {
	s := NewScheduler(testLogger())
	jobs := s.GetJobs()
	if len(jobs) != 0 {
		t.Errorf("expected 0 jobs, got %d", len(jobs))
	}
}

func TestExecuteJob_Success(t *testing.T) {
	s := NewScheduler(testLogger())
	job := newFakeJob("ok-job")

	s.executeJob(job)

	if job.execCount.Load() != 1 {
		t.Errorf("expected 1 execution, got %d", job.execCount.Load())
	}
}

func TestExecuteJob_Error(t *testing.T) {
	s := NewScheduler(testLogger())
	job := &fakeJob{
		name: "failing-job",
		execFn: func(ctx context.Context) error {
			return fmt.Errorf("job failed")
		},
	}

	// Should not panic even when job fails
	s.executeJob(job)
	if job.execCount.Load() != 1 {
		t.Errorf("expected 1 execution, got %d", job.execCount.Load())
	}
}

func TestExecuteJob_RespectsContext(t *testing.T) {
	s := NewScheduler(testLogger())
	var receivedCtx context.Context
	job := &fakeJob{
		name: "ctx-job",
		execFn: func(ctx context.Context) error {
			receivedCtx = ctx
			return nil
		},
	}

	s.executeJob(job)

	if receivedCtx == nil {
		t.Fatal("expected non-nil context")
	}
	// executeJob creates a context with 30-minute timeout
	deadline, ok := receivedCtx.Deadline()
	if !ok {
		t.Fatal("expected context with deadline")
	}
	if time.Until(deadline) > 31*time.Minute {
		t.Error("expected deadline within ~30 minutes")
	}
}
