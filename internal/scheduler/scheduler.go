package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/itsmrshow/bulwark/internal/logging"
	"github.com/robfig/cron/v3"
)

// Job represents a scheduled job
type Job interface {
	// Execute runs the job
	Execute(ctx context.Context) error

	// Name returns the job name
	Name() string
}

// Scheduler manages scheduled jobs with cron expressions
type Scheduler struct {
	cron   *cron.Cron
	jobs   map[string]Job
	logger *logging.Logger
	mu     sync.RWMutex
}

// NewScheduler creates a new scheduler
func NewScheduler(logger *logging.Logger) *Scheduler {
	return &Scheduler{
		cron:   cron.New(),
		jobs:   make(map[string]Job),
		logger: logger.WithComponent("scheduler"),
	}
}

// AddJob adds a job with a cron expression
func (s *Scheduler) AddJob(cronExpr string, job Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Info().
		Str("job", job.Name()).
		Str("schedule", cronExpr).
		Msg("Adding scheduled job")

	// Parse and validate cron expression
	_, err := cron.ParseStandard(cronExpr)
	if err != nil {
		return fmt.Errorf("invalid cron expression %q: %w", cronExpr, err)
	}

	// Add job to cron
	_, err = s.cron.AddFunc(cronExpr, func() {
		s.executeJob(job)
	})

	if err != nil {
		return fmt.Errorf("failed to add job: %w", err)
	}

	s.jobs[job.Name()] = job

	s.logger.Info().
		Str("job", job.Name()).
		Msg("Job added successfully")

	return nil
}

// Start starts the scheduler
func (s *Scheduler) Start() {
	s.mu.RLock()
	jobCount := len(s.jobs)
	s.mu.RUnlock()

	s.logger.Info().
		Int("jobs", jobCount).
		Msg("Starting scheduler")

	s.cron.Start()
}

// Stop stops the scheduler gracefully
func (s *Scheduler) Stop() {
	s.logger.Info().Msg("Stopping scheduler")
	ctx := s.cron.Stop()
	<-ctx.Done()
	s.logger.Info().Msg("Scheduler stopped")
}

// executeJob runs a job with logging and error handling
func (s *Scheduler) executeJob(job Job) {
	s.logger.Info().
		Str("job", job.Name()).
		Msg("Executing scheduled job")

	start := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	err := job.Execute(ctx)
	duration := time.Since(start)

	if err != nil {
		s.logger.Error().
			Err(err).
			Str("job", job.Name()).
			Dur("duration", duration).
			Msg("Scheduled job failed")
	} else {
		s.logger.Info().
			Str("job", job.Name()).
			Dur("duration", duration).
			Msg("Scheduled job completed successfully")
	}
}

// GetJobs returns the list of registered jobs
func (s *Scheduler) GetJobs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobs := make([]string, 0, len(s.jobs))
	for name := range s.jobs {
		jobs = append(jobs, name)
	}
	return jobs
}
