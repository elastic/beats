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
	"errors"
	"sync"
	"time"

	"github.com/elastic/beats/heartbeat/scheduler/throttler"

	"github.com/elastic/beats/libbeat/common/atomic"
	"github.com/elastic/beats/libbeat/logp"
)

const (
	statePreRunning int = iota + 1
	stateRunning
	stateDone
)

type Scheduler struct {
	limit uint
	state atomic.Int
	done  chan int

	location *time.Location

	jobs    []*job
	active  atomic.Uint // number of active entries
	waiting atomic.Uint // number of jobs waiting to run, but constrained by scheduler limit

	throttler *throttler.Throttler
}

type Canceller func() error

// A job is a re-schedulable entry point in a set of tasks. Each task can return
// a new set of tasks being executed (subject to active task limits). Only after
// all tasks of a job have been finished, the job is marked as done and subject
// to be re-scheduled.
type job struct {
	id       string
	next     time.Time
	schedule Schedule
	fn       TaskFunc

	registered bool
	running    uint32 // count number of active task for job
}

// A single task in an active job.
type task struct {
	job *job
	fn  TaskFunc
}

// Single task in an active job. Optionally returns continuation of tasks to
// be executed within current job.
type TaskFunc func() []TaskFunc

type taskOverSignal struct {
	entry *job
	cont  []task // continuation tasks to be executed by concurrently for job at hand
}

// Schedule defines an interface for getting the next scheduled runtime for a job
type Schedule interface {
	Next(now time.Time) (next time.Time)
}

var debugf = logp.MakeDebug("scheduler")

// New creates a new Scheduler
func New(limit uint) *Scheduler {
	return NewWithLocation(limit, time.Local)
}

// NewWithLocation creates a new Scheduler using the given time zone.
func NewWithLocation(limit uint, location *time.Location) *Scheduler {
	stateInitial := statePreRunning
	return &Scheduler{
		limit:    limit,
		location: location,

		state: atomic.MakeInt(stateInitial),
		done:  make(chan int),

		throttler: throttler.NewThrottler(limit),
	}
}

// Start the scheduler.
func (s *Scheduler) Start() error {
	s.state.Store(stateRunning)
	s.throttler.Start()

	// Stats reporter
	go func() {
		t := time.NewTicker(time.Second * 10)

		for {
			select {
			case <-s.done:
				t.Stop()
				return
			case <-t.C:
				logp.Info("Scheduler Active/Waiting (limit): %d/%d (%d)", s.active.Load(), s.waiting.Load(), s.limit)
			}
		}
	}()

	return nil
}

// Stop all executing tasks in the scheduler. Cannot be restarted after Stop.
func (s *Scheduler) Stop() error {
	s.state.Store(stateDone)
	close(s.done)
	s.throttler.Stop()

	return nil
}

// ErrAlreadyStopped is returned when an Add operation is attempted after the scheduler
// has already stopped.
var ErrAlreadyStopped = errors.New("attempted to add job to already stopped scheduler")

// Add adds the given TaskFunc to the current scheduler. Will return an error if the scheduler
// is done.
func (s *Scheduler) Add(sched Schedule, id string, entrypoint TaskFunc) (removeFn func() error, err error) {
	if s.state.Load() == stateDone {
		return nil, ErrAlreadyStopped
	}

	removeCh := make(chan bool)
	removeFn = func() error {
		removeCh <- true
		return nil
	}

	var timer *time.Timer
	go func() {
		for {
			now := time.Now()
			next := sched.Next(now)
			if timer == nil {
				timer = time.NewTimer(next.Sub(now))
			} else {
				timer.Reset(next.Sub(now))
			}

			select {
			case <-timer.C:
				s.runOnce(id, entrypoint)
			case <-removeCh:
				return
			}
		}
	}()

	return removeFn, nil
}

// runOnce runs a TaskFunc and its continuations once.
func (s *Scheduler) runOnce(id string, entrypoint TaskFunc) {
	// Since we run all continuations asynchronously we use a wait group to block until we're done.
	wg := sync.WaitGroup{}

	// Since task funcs can emit continuations recursively we need a function to execute
	// recursively. We declare the function variable before definition to allow for recursion.
	var runRecursive func(task TaskFunc)
	runRecursive = func(task TaskFunc) {
		wg.Add(1)
		defer wg.Done()

		// The accounting for waiting/active is done using atomics. Absolute accuracy is not critical here so the gap
		// between modifying waiting and active is acceptable.
		s.waiting.Inc()

		// Acquire an execution slot in keeping with heartbeat.scheduler.limit
		acquired, releaseSlot := s.throttler.AcquireSlot()
		defer releaseSlot()

		s.active.Inc()
		s.waiting.Dec()

		// The only situation in which we can't acquire a slot is during shutdown. In that case we can just return
		// without worrying about any of the counters since we're going away soon.
		if !acquired {
			return
		}
		continuations := task()
		s.active.Dec()

		for _, cont := range continuations {
			go runRecursive(cont) // Run continuations in parallel, note that these each will acquire their own slots
		}
	}

	runRecursive(entrypoint)

	wg.Wait()
}
