// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package scheduler

import (
	"context"
	"crypto/rand"
	"math/big"
	"sync"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
)

// QueryFunc is the function type for executing a query.
// scheduleID is the policy-defined schedule id (may be empty, caller can use name).
// executionIndex is the 1-based schedule execution count for this run.
type QueryFunc func(ctx context.Context, name, query string, timeout time.Duration, scheduleID string, executionIndex int) error

// ScheduledQuery represents a query with its recurrence schedule
type ScheduledQuery struct {
	Name     string
	Query    string
	Timeout  time.Duration
	Schedule *RecurrenceSchedule
	// ScheduleID is the policy-defined schedule id for this scheduled query (optional)
	ScheduleID string
}

// Scheduler manages cron-based query scheduling
type Scheduler struct {
	log       *logp.Logger
	queries   map[string]*scheduledJob
	queryFunc QueryFunc
	mx        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

type scheduledJob struct {
	query  *ScheduledQuery
	cancel context.CancelFunc
}

// New creates a new recurrence scheduler
func New(log *logp.Logger, queryFunc QueryFunc) *Scheduler {
	return &Scheduler{
		log:       log.With("component", "recurrence-scheduler"),
		queries:   make(map[string]*scheduledJob),
		queryFunc: queryFunc,
	}
}

// Start starts the scheduler
func (s *Scheduler) Start(ctx context.Context) {
	s.mx.Lock()
	defer s.mx.Unlock()

	s.ctx, s.cancel = context.WithCancel(ctx)
	s.log.Info("Recurrence scheduler started")
}

// Stop stops the scheduler and all scheduled jobs
func (s *Scheduler) Stop() {
	s.mx.Lock()
	defer s.mx.Unlock()

	if s.cancel != nil {
		s.cancel()
	}

	// Cancel all running jobs
	for _, job := range s.queries {
		if job.cancel != nil {
			job.cancel()
		}
	}

	// Wait for all jobs to finish
	s.wg.Wait()

	s.queries = make(map[string]*scheduledJob)
	s.log.Info("Recurrence scheduler stopped")
}

// AddQuery adds or updates a scheduled query
func (s *Scheduler) AddQuery(sq *ScheduledQuery) error {
	s.mx.Lock()
	defer s.mx.Unlock()

	if s.ctx == nil {
		return ErrOutsideScheduleWindow
	}

	// Remove existing job if present
	if existing, ok := s.queries[sq.Name]; ok {
		if existing.cancel != nil {
			existing.cancel()
		}
	}

	// Validate schedule
	if sq.Schedule == nil || !sq.Schedule.IsActive() {
		delete(s.queries, sq.Name)
		return nil
	}

	// Create job context
	jobCtx, jobCancel := context.WithCancel(s.ctx)

	job := &scheduledJob{
		query:  sq,
		cancel: jobCancel,
	}
	s.queries[sq.Name] = job

	// Start the scheduling goroutine
	s.wg.Add(1)
	go s.runJob(jobCtx, job)

	s.log.Infof("Added scheduled query '%s' with rrule '%s'", sq.Name, sq.Schedule.RRule)
	return nil
}

// RemoveQuery removes a scheduled query
func (s *Scheduler) RemoveQuery(name string) {
	s.mx.Lock()
	defer s.mx.Unlock()

	if job, ok := s.queries[name]; ok {
		if job.cancel != nil {
			job.cancel()
		}
		delete(s.queries, name)
		s.log.Infof("Removed scheduled query '%s'", name)
	}
}

// UpdateQueries updates all scheduled queries
func (s *Scheduler) UpdateQueries(queries []*ScheduledQuery) error {
	s.mx.Lock()
	defer s.mx.Unlock()

	if s.ctx == nil {
		return ErrOutsideScheduleWindow
	}

	// Track which queries to keep
	newQueryNames := make(map[string]bool)
	for _, sq := range queries {
		newQueryNames[sq.Name] = true
	}

	// Remove queries that are no longer present
	for name, job := range s.queries {
		if !newQueryNames[name] {
			if job.cancel != nil {
				job.cancel()
			}
			delete(s.queries, name)
			s.log.Infof("Removed scheduled query '%s'", name)
		}
	}

	// Add or update queries
	for _, sq := range queries {
		if sq.Schedule == nil || !sq.Schedule.IsActive() {
			continue
		}

		// Check if query needs updating
		if existing, ok := s.queries[sq.Name]; ok {
			// If schedule hasn't changed, skip
			if existing.query.Schedule.RRule == sq.Schedule.RRule &&
				existing.query.Query == sq.Query {
				continue
			}
			// Cancel existing job
			if existing.cancel != nil {
				existing.cancel()
			}
		}

		// Create new job
		jobCtx, jobCancel := context.WithCancel(s.ctx)
		job := &scheduledJob{
			query:  sq,
			cancel: jobCancel,
		}
		s.queries[sq.Name] = job

		// Start the scheduling goroutine
		s.wg.Add(1)
		go s.runJob(jobCtx, job)

		s.log.Infof("Scheduled query '%s' with rrule '%s'", sq.Name, sq.Schedule.RRule)
	}

	return nil
}

// runJob runs a scheduled job according to its cron schedule
func (s *Scheduler) runJob(ctx context.Context, job *scheduledJob) {
	defer s.wg.Done()

	sq := job.query
	schedule := sq.Schedule

	// Validate splay (should already be validated during config parsing, but check again)
	if err := schedule.ValidateSplay(); err != nil {
		s.log.Warnf("Query '%s': splay validation error: %v", sq.Name, err)
	}

	for {
		now := time.Now()

		// Check if we're within the schedule window
		if !schedule.IsWithinWindow(now) {
			// If we haven't reached the start date yet, wait until then
			if schedule.StartDate != nil && now.Before(*schedule.StartDate) {
				waitDuration := schedule.StartDate.Sub(now)
				s.log.Debugf("Query '%s' waiting until start date: %v", sq.Name, schedule.StartDate)

				select {
				case <-ctx.Done():
					return
				case <-time.After(waitDuration):
					continue
				}
			}
			// If we're past the end date, stop
			s.log.Infof("Query '%s' has passed its end date, stopping", sq.Name)
			return
		}

		// Calculate next execution time
		nextRun := schedule.Next(now)
		if nextRun.IsZero() {
			s.log.Infof("Query '%s' has no more scheduled runs, stopping", sq.Name)
			return
		}

		// Calculate splay
		maxSplay := schedule.GetMaxSplayDuration()
		splayDuration := s.randomSplay(maxSplay)

		scheduledTime := nextRun.Add(splayDuration)

		waitDuration := time.Until(scheduledTime)
		if waitDuration < 0 {
			waitDuration = 0
		}

		s.log.Debugf("Query '%s' scheduled for %v (splay: %v)", sq.Name, scheduledTime, splayDuration)

		select {
		case <-ctx.Done():
			return
		case <-time.After(waitDuration):
			// Execution time (after splay); use for execution index
			runTime := time.Now()
			executionIndex := 0
			if n, ok := schedule.ExecutionIndex(runTime); ok {
				executionIndex = n
			}
			s.executeQuery(ctx, sq, runTime, executionIndex)
		}
	}
}

// executeQuery executes a scheduled query
func (s *Scheduler) executeQuery(ctx context.Context, sq *ScheduledQuery, runTime time.Time, executionIndex int) {
	s.log.Debugf("Executing scheduled query '%s' (execution #%d)", sq.Name, executionIndex)

	// Create a timeout context if specified
	execCtx := ctx
	var cancel context.CancelFunc
	if sq.Timeout > 0 {
		execCtx, cancel = context.WithTimeout(ctx, sq.Timeout)
		defer cancel()
	}

	scheduleID := sq.ScheduleID
	if scheduleID == "" {
		scheduleID = sq.Name
	}
	err := s.queryFunc(execCtx, sq.Name, sq.Query, sq.Timeout, scheduleID, executionIndex)
	if err != nil {
		s.log.Errorf("Error executing scheduled query '%s': %v", sq.Name, err)
	} else {
		s.log.Debugf("Completed scheduled query '%s'", sq.Name)
	}
}

// randomSplay returns a random duration between 0 and maxSplay
func (s *Scheduler) randomSplay(maxSplay time.Duration) time.Duration {
	if maxSplay <= 0 {
		return 0
	}

	n, err := rand.Int(rand.Reader, big.NewInt(int64(maxSplay)))
	if err != nil {
		// Fall back to no splay on error
		s.log.Warnf("Failed to generate random splay: %v", err)
		return 0
	}

	return time.Duration(n.Int64())
}

// Count returns the number of scheduled queries
func (s *Scheduler) Count() int {
	s.mx.RLock()
	defer s.mx.RUnlock()
	return len(s.queries)
}

// QueryNames returns the names of all scheduled queries
func (s *Scheduler) QueryNames() []string {
	s.mx.RLock()
	defer s.mx.RUnlock()

	names := make([]string, 0, len(s.queries))
	for name := range s.queries {
		names = append(names, name)
	}
	return names
}
