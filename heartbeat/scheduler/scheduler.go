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

	jobs   []*job
	active uint // number of active entries

	throttler *Throttler
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

type Schedule interface {
	Next(now time.Time) (next time.Time)
}

var debugf = logp.MakeDebug("scheduler")

func New(limit uint) *Scheduler {
	return NewWithLocation(limit, time.Local)
}

func NewWithLocation(limit uint, location *time.Location) *Scheduler {
	stateInitial := statePreRunning
	return &Scheduler{
		limit:    limit,
		location: location,

		state: atomic.MakeInt(stateInitial),
		done:  make(chan int),

		throttler: NewThrottler(limit),
	}
}

func (s *Scheduler) Start() error {
	s.state.Store(stateRunning)
	s.throttler.start()
	return nil
}

func (s *Scheduler) Stop() error {
	s.state.Store(stateDone)
	close(s.done)
	s.throttler.stop()

	return nil
}

// ErrAlreadyStopped is returned when an Add operation is attempted after the scheduler
// has already stopped.
var ErrAlreadyStopped = errors.New("attempted to add job to already stopped scheduler")

// Add adds the given TaskFunc to the current scheduler. Will return an error if the scheduler
// is done.
func (s *Scheduler) Add(sched Schedule, id string, entrypoint TaskFunc) (removeFn func() error, err error) {
	removeCh := make(chan bool)
	removeFn = func() error {
		removeCh <- true
		return nil
	}

	go func() {
		for {
			now := time.Now()
			next := sched.Next(now)
			timer := time.NewTimer(next.Sub(now))

			logp.Info("TIMER LOOP %s", next.Sub(now))

			select {
			case <-timer.C:
				logp.Info("TIMER RUN")
				s.runOnce(id, entrypoint)
			case <-removeCh:
				logp.Info("TIMER REMOVE")
				return
			}
		}
	}()

	return removeFn, nil
}

func (s *Scheduler) runOnce(id string, entrypoint TaskFunc) {
	wg := sync.WaitGroup{}

	// Declare this to allow us to use it recursively
	var runRecursive func(task TaskFunc)
	runRecursive = func(task TaskFunc) {
		logp.Info("INCR")
		wg.Add(1) // we start with one task
		defer wg.Done()

		acquired, releaseSlot := s.throttler.acquireSlot()
		defer releaseSlot()
		logp.Info("ACQ %v", acquired)
		if !acquired {
			debugf("Could not acquire slot for task, likely due to stop")
			return
		}
		continuations := task()
		logp.Info("RAN TASK, CONTS %v", continuations)

		for _, cont := range continuations {
			go runRecursive(cont) // Run continuations in parallel
		}

		logp.Info("DECR")
	}

	runRecursive(entrypoint)

	wg.Wait()
}
