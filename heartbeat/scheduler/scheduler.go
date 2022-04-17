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

	"golang.org/x/sync/semaphore"

	"github.com/menderesk/beats/v7/heartbeat/config"
	"github.com/menderesk/beats/v7/heartbeat/scheduler/timerqueue"
	"github.com/menderesk/beats/v7/libbeat/logp"
	"github.com/menderesk/beats/v7/libbeat/monitoring"
)

var debugf = logp.MakeDebug("scheduler")

// ErrInvalidTransition is returned from start/stop when making an invalid state transition, say from preRunning to stopped
var ErrInvalidTransition = fmt.Errorf("invalid state transition")

// Scheduler represents our async timer based scheduler.
type Scheduler struct {
	limit       int64
	limitSem    *semaphore.Weighted
	location    *time.Location
	timerQueue  *timerqueue.TimerQueue
	ctx         context.Context
	cancelCtx   context.CancelFunc
	stats       schedulerStats
	jobLimitSem map[string]*semaphore.Weighted
	runOnce     bool
	runOnceWg   *sync.WaitGroup
}

type schedulerStats struct {
	activeJobs         *monitoring.Uint // gauge showing number of active jobs
	activeTasks        *monitoring.Uint // gauge showing number of active tasks
	waitingTasks       *monitoring.Uint // number of tasks waiting to run, but constrained by scheduler limit
	jobsMissedDeadline *monitoring.Uint // counter for number of jobs that missed start deadline
}

// TaskFunc represents a single task in a job. Optionally returns continuation of tasks to
// be executed within current job.
type TaskFunc func(ctx context.Context) []TaskFunc

// Schedule defines an interface for getting the next scheduled runtime for a job
type Schedule interface {
	// Next returns the next runAt a scheduled event occurs after the given runAt
	Next(now time.Time) (next time.Time)
	// Returns true if this schedule type should run once immediately before checking Next.
	// Cron tasks run at exact times so should set this to false.
	RunOnInit() bool
}

func getJobLimitSem(jobLimitByType map[string]config.JobLimit) map[string]*semaphore.Weighted {
	jobLimitSem := map[string]*semaphore.Weighted{}
	for jobType, jobLimit := range jobLimitByType {
		if jobLimit.Limit > 0 {
			jobLimitSem[jobType] = semaphore.NewWeighted(jobLimit.Limit)
		}
	}
	return jobLimitSem
}

// NewWithLocation creates a new Scheduler using the given runAt zone.
func Create(limit int64, registry *monitoring.Registry, location *time.Location, jobLimitByType map[string]config.JobLimit, runOnce bool) *Scheduler {
	ctx, cancelCtx := context.WithCancel(context.Background())

	if limit < 1 {
		limit = math.MaxInt64
	}

	jobsMissedDeadlineCounter := monitoring.NewUint(registry, "jobs.missed_deadline")
	activeJobsGauge := monitoring.NewUint(registry, "jobs.active")
	activeTasksGauge := monitoring.NewUint(registry, "tasks.active")
	waitingTasksGauge := monitoring.NewUint(registry, "tasks.waiting")

	sched := &Scheduler{
		limit:       limit,
		location:    location,
		ctx:         ctx,
		cancelCtx:   cancelCtx,
		limitSem:    semaphore.NewWeighted(limit),
		jobLimitSem: getJobLimitSem(jobLimitByType),
		timerQueue:  timerqueue.NewTimerQueue(ctx),
		runOnce:     runOnce,
		runOnceWg:   &sync.WaitGroup{},

		stats: schedulerStats{
			activeJobs:         activeJobsGauge,
			activeTasks:        activeTasksGauge,
			waitingTasks:       waitingTasksGauge,
			jobsMissedDeadline: jobsMissedDeadlineCounter,
		},
	}

	sched.timerQueue.Start()
	go sched.missedDeadlineReporter()

	return sched
}

func (s *Scheduler) missedDeadlineReporter() {
	interval := time.Second * 15

	t := time.NewTicker(interval)

	// Counter used to check if we're missing more checks now than before
	missedAtLastCheck := uint64(0)
	for {
		select {
		case <-s.ctx.Done():
			t.Stop()
			return
		case <-t.C:
			missingNow := s.stats.jobsMissedDeadline.Get()
			missedDelta := missingNow - missedAtLastCheck
			if missedDelta > 0 {
				logp.Warn("%d tasks have missed their schedule deadlines by more than 1 second in the last %s.", missedDelta, interval)
			}
			missedAtLastCheck = missingNow
		}
	}
}

// Stop all executing tasks in the scheduler. Cannot be restarted after Stop.
func (s *Scheduler) Stop() {
	s.cancelCtx()
}

// Wait until all tasks are done if run in runOnce mode. Will block forever
// if this scheduler does not have the runOnce option set.
// Adding new tasks after this method is invoked is not supported.
func (s *Scheduler) WaitForRunOnce() {
	s.runOnceWg.Wait()
	s.Stop()
}

// ErrAlreadyStopped is returned when an Add operation is attempted after the scheduler
// has already stopped.
var ErrAlreadyStopped = errors.New("attempted to add job to already stopped scheduler")

type AddTask func(sched Schedule, id string, entrypoint TaskFunc, jobType string, waitForPublish func()) (removeFn context.CancelFunc, err error)

// Add adds the given TaskFunc to the current scheduler. Will return an error if the scheduler
// is done.
func (s *Scheduler) Add(sched Schedule, id string, entrypoint TaskFunc, jobType string, waitForPublish func()) (removeFn context.CancelFunc, err error) {
	if errors.Is(s.ctx.Err(), context.Canceled) {
		return nil, ErrAlreadyStopped
	}

	jobCtx, jobCtxCancel := context.WithCancel(s.ctx)

	// lastRanAt stores the last runAt the task was invoked
	// The initial value is runAt.Now() because we use it to get the next runAt a job is scheduled to run
	lastRanAt := time.Now().In(s.location)

	var taskFn timerqueue.TimerTaskFn

	taskFn = func(_ time.Time) {
		select {
		case <-jobCtx.Done():
			debugf("Job '%v' canceled", id)
			return
		default:
		}
		s.stats.activeJobs.Inc()
		debugf("Job '%s' started", id)
		sj := newSchedJob(jobCtx, s, id, jobType, entrypoint)

		lastRanAt := sj.run()
		s.stats.activeJobs.Dec()

		if s.runOnce {
			waitForPublish()
			s.runOnceWg.Done()
		} else {
			// Schedule the next run
			s.runTaskOnce(sched.Next(lastRanAt), taskFn, true)
		}
		debugf("Job '%v' returned at %v", id, time.Now())
	}

	if s.runOnce {
		s.runOnceWg.Add(1)
	}

	// Run non-cron tasks immediately, or run all tasks immediately if we're
	// in RunOnce mode
	if s.runOnce || sched.RunOnInit() {
		s.runTaskOnce(time.Now(), taskFn, false)
	} else {
		s.runTaskOnce(sched.Next(lastRanAt), taskFn, true)
	}

	return func() {
		debugf("Remove scheduler job '%v'", id)
		jobCtxCancel()
	}, nil
}

// runTaskOnce runs the given task exactly once at the given time. Set deadlineCheck
// to false if this is the first invocation of this, otherwise the deadline checker
// will complain about a missed task
func (s *Scheduler) runTaskOnce(runAt time.Time, taskFn timerqueue.TimerTaskFn, deadlineCheck bool) {
	now := time.Now().In(s.location)
	// Check if the task is more than 1 second late
	if deadlineCheck && runAt.Sub(now) < time.Second {
		s.stats.jobsMissedDeadline.Inc()
	}

	// Schedule task to run sometime in the future. Wrap the task in a go-routine so it doesn't
	// blocks the timer thread.
	asyncTask := func(now time.Time) { go taskFn(now) }
	s.timerQueue.Push(runAt, asyncTask)
}
