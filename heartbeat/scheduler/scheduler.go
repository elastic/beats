// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package scheduler

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/elastic/beats/heartbeat/scheduler/timerqueue"

	"golang.org/x/sync/semaphore"

	"github.com/elastic/beats/libbeat/common/atomic"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"
)

const (
	statePreRunning int = iota + 1
	stateRunning
	stateStopped
)

// InvalidTransitionError is returned from start/stop when making an invalid state transition, say from preRunning to stopped
var InvalidTransitionError = fmt.Errorf("invalid state transition")

type Scheduler struct {
	limit int64
	state atomic.Int

	location *time.Location

	activeJobs         *monitoring.Uint // gauge showing number of active jobs
	activeTasks        *monitoring.Uint // gauge showing number of active tasks
	waitingTasks       *monitoring.Uint // number of tasks waiting to run, but constrained by scheduler limit
	jobsPerSecond      *monitoring.Uint // rate of job processing computed over the past hour
	jobsMissedDeadline *monitoring.Uint // counter for number of jobs that missed start deadline

	timerQueue *timerqueue.TimerQueue

	jobsRun   *atomic.Uint64
	ctx       context.Context
	cancelCtx context.CancelFunc
	sem       *semaphore.Weighted
}

// TaskFunc represents a single task in a job. Optionally returns continuation of tasks to
// be executed within current job.
type TaskFunc func() []TaskFunc

// Schedule defines an interface for getting the next scheduled runtime for a job
type Schedule interface {
	// Next returns the next runAt a scheduled event occurs after the given runAt
	Next(now time.Time) (next time.Time)
	// Returns true if this schedule type should run once immediately before checking Next.
	// Cron tasks run at exact times so should set this to false.
	RunOnInit() bool
}

var debugf = logp.MakeDebug("scheduler")

// New creates a new Scheduler
func New(limit int64, registry *monitoring.Registry) *Scheduler {
	return NewWithLocation(limit, registry, time.Local)
}

// NewWithLocation creates a new Scheduler using the given runAt zone.
func NewWithLocation(limit int64, registry *monitoring.Registry, location *time.Location) *Scheduler {
	ctx, cancelCtx := context.WithCancel(context.Background())

	if limit < 1 {
		limit = math.MaxInt64
	}

	jobsMissedDeadlineCounter := monitoring.NewUint(registry, "jobs.missed_deadline")
	activeJobsGauge := monitoring.NewUint(registry, "jobs.active")
	activeTasksGauge := monitoring.NewUint(registry, "tasks.active")
	waitingTasksGauge := monitoring.NewUint(registry, "tasks.waiting")

	sched := &Scheduler{
		limit:     limit,
		location:  location,
		state:     atomic.MakeInt(statePreRunning),
		ctx:       ctx,
		cancelCtx: cancelCtx,
		sem:       semaphore.NewWeighted(limit),

		jobsRun:            atomic.NewUint64(0),
		activeJobs:         activeJobsGauge,
		activeTasks:        activeTasksGauge,
		waitingTasks:       waitingTasksGauge,
		jobsMissedDeadline: jobsMissedDeadlineCounter,

		timerQueue: timerqueue.NewTimerQueue(ctx),
	}

	return sched
}

// Start the scheduler. Starting a stopped scheduler returns an error.
func (s *Scheduler) Start() error {
	if s.state.Load() == stateStopped {
		return InvalidTransitionError
		//} else if !s.state.CAS(statePreRunning, stateRunning) {
		// We already were running, so just return nil and do nothing.
		return nil
	}
	if !s.state.CAS(statePreRunning, stateRunning) {
		return nil // we already running, just exit
	}

	s.timerQueue.Start()

	// Missed deadline reporter
	go s.missedDeadlineReporter()

	return nil
}

func (s *Scheduler) missedDeadlineReporter() {
	interval := time.Second

	t := time.NewTicker(interval)

	// Counter used to check if we're missing more checks now than before
	missedAtLastCheck := uint64(0)
	for {
		select {
		case <-s.ctx.Done():
			t.Stop()
			return
		case <-t.C:
			logp.Info("JOBS RUN %d", s.jobsRun.Load())
			missingNow := s.jobsMissedDeadline.Get()
			missedDelta := missingNow - missedAtLastCheck
			if missedDelta > 0 {
				logp.Warn("%d tasks have missed their schedule deadlines in the last %s.", missedDelta, interval)
			}
			missedAtLastCheck = missingNow
		}
	}
}

// Stop all executing tasks in the scheduler. Cannot be restarted after Stop.
func (s *Scheduler) Stop() error {
	if s.state.CAS(stateRunning, stateStopped) {
		s.cancelCtx()
		return nil
	} else if s.state.Load() == stateStopped {
		return nil
	}

	return InvalidTransitionError
}

// ErrAlreadyStopped is returned when an Add operation is attempted after the scheduler
// has already stopped.
var ErrAlreadyStopped = errors.New("attempted to add job to already stopped scheduler")

// Add adds the given TaskFunc to the current scheduler. Will return an error if the scheduler
// is done.
func (s *Scheduler) Add(sched Schedule, id string, entrypoint TaskFunc) (removeFn context.CancelFunc, err error) {
	if s.state.Load() == stateStopped {
		return nil, ErrAlreadyStopped
	}

	jobCtx, jobCtxCancel := context.WithCancel(s.ctx)

	// lastRanAt stores the last runAt the task was invoked
	// The initial value is runAt.Now() because we use it to get the next runAt a job is scheduled to run
	lastRanAt := time.Now().In(s.location)

	var taskFn timerqueue.TimerTaskFn

	schedNextRun := func() {
		next := sched.Next(lastRanAt)

		now := time.Now().In(s.location)
		if next.Before(now) {
			// Our last invocation went long!
			s.jobsMissedDeadline.Inc()
			taskFn(now)
		} else {
			// Schedule task to run sometime in the future. Wrap the task in a go-routine so it doesn't
			// block the timer thread.
			asyncTask := func(now time.Time) { go taskFn(now) }
			s.timerQueue.Push(timerqueue.NewTimerTask(next, asyncTask))
		}
	}

	taskFn = func(_ time.Time) {
		// Attempt to acquire a resource to see if we're blocked
		// We can't really tell if we're blocked because we need to acquire resources later
		// for the entrypoint + continuations, but this is good enough for our lastRanAt
		// accounting since we want that value set at the first plausible time where the
		// job could run
		s.sem.Acquire(s.ctx, 1)
		s.sem.Release(1)
		lastRanAt = time.Now()

		s.activeJobs.Inc()

		s.runRecursiveJob(jobCtx, entrypoint)

		s.jobsRun.Inc()
		s.activeJobs.Dec()

		schedNextRun()
	}

	if sched.RunOnInit() {
		go taskFn(time.Now())
	} else {
		schedNextRun()
	}

	return jobCtxCancel, nil
}

// runRecursiveJob runs the entry point for a job, blocking until all subtasks are completed.
// Subtasks are run in separate goroutines.
func (s *Scheduler) runRecursiveJob(jobCtx context.Context, task TaskFunc) {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	s.runRecursiveTask(jobCtx, task, wg)
	wg.Wait()
}

// runRecursiveTask runs an individual task and its continuations until none are left with as much parallelism as possible.
// Since task funcs can emit continuations recursively we need a function to execute
// recursively.
// The wait group passed into this function expects to already have its count incremented by one.
func (s *Scheduler) runRecursiveTask(jobCtx context.Context, task TaskFunc, wg *sync.WaitGroup) {
	defer wg.Done()

	// The accounting for waiting/active tasks is done using atomics.
	// Absolute accuracy is not critical here so the gap between modifying waitingTasks and activeJobs is acceptable.
	s.waitingTasks.Inc()

	// Acquire an execution slot in keeping with heartbeat.scheduler.limit
	s.sem.Acquire(s.ctx, 1)
	defer s.sem.Release(1)

	// Check if the scheduler has been shut down. If so, exit early
	select {
	case <-jobCtx.Done():
		return
	default:
		s.activeTasks.Inc()
		s.waitingTasks.Dec()

		continuations := task()
		s.activeTasks.Dec()

		for _, cont := range continuations {
			go s.runRecursiveTask(jobCtx, cont, wg) // Run continuations in parallel, note that these each will acquire their own slots
		}
	}

}
