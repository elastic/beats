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
	"runtime/debug"
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

	location *time.Location

	jobs   []*job
	active uint // number of active entries

	addCh, rmCh chan *job
	finished    chan taskOverSignal

	// list of active tasks waiting to be executed
	tasks []task

	done chan struct{}
	wg   sync.WaitGroup
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

		state:  atomic.MakeInt(stateInitial),
		jobs:   nil,
		active: 0,

		addCh:    make(chan *job),
		rmCh:     make(chan *job),
		finished: make(chan taskOverSignal),

		done: make(chan struct{}),
		wg:   sync.WaitGroup{},
	}
}

func (s *Scheduler) Start() error {
	if !s.transitionRunning() {
		return errors.New("scheduler can only be stopped from a running state")
	}

	go s.run()
	return nil
}

func (s *Scheduler) Stop() error {
	if !s.isRunning() {
		return errors.New("scheduler can only be started from an initialized state")
	}

	close(s.done)
	s.wg.Wait()
	s.transitionStopped()
	return nil
}

// ErrAlreadyStopped is returned when an Add operation is attempted after the scheduler
// has already stopped.
var ErrAlreadyStopped = errors.New("attempted to add job to already stopped scheduler")

// Add adds the given TaskFunc to the current scheduler. Will return an error if the scheduler
// is done.
func (s *Scheduler) Add(sched Schedule, id string, entrypoint TaskFunc) (removeFn func() error, err error) {
	debugf("Add scheduler job '%v'.", id)

	j := &job{
		id:         id,
		fn:         entrypoint,
		schedule:   sched,
		registered: false,
		running:    0,
	}
	if s.isPreRunning() {
		s.addSync(j)
	} else if s.isRunning() {
		s.addCh <- j
	} else {
		return nil, ErrAlreadyStopped
	}

	return func() error { return s.remove(j) }, nil
}

func (s *Scheduler) remove(j *job) error {
	debugf("Remove scheduler job '%v'", j.id)

	if s.isPreRunning() {
		s.doRemove(j)
	} else if s.isRunning() {
		s.rmCh <- j
	}
	// There is no need to handle the isDone case
	// because removing the job accomplishes nothing if
	// the scheduler is stopped

	return nil
}

func (s *Scheduler) run() {
	defer func() {
		// drain finished queue for active jobs to not leak
		// go-routines on exit
		for i := uint(0); i < s.active; i++ {
			<-s.finished
		}
	}()

	debugf("Start scheduler.")
	defer debugf("Scheduler stopped.")

	now := time.Now().In(s.location)
	for _, j := range s.jobs {
		j.next = j.schedule.Next(now)
	}

	resched := true

	var timer *time.Timer
	for {
		if resched {
			sortEntries(s.jobs)
		}
		resched = true

		unlimited := s.limit == 0
		if (unlimited || s.active < s.limit) && len(s.jobs) > 0 {
			next := s.jobs[0].next
			debugf("Next wakeup time: %v", next)

			if timer != nil {
				timer.Stop()
			}

			// Calculate the amount of time between now and the next execution
			// since the timers operation on durations, not exact amounts of time
			nextExecIn := next.Sub(time.Now().In(s.location))
			timer = time.NewTimer(nextExecIn)
		}

		var timeSignal <-chan time.Time
		if timer != nil {
			timeSignal = timer.C
		}

		select {
		case now = <-timeSignal:
			for _, j := range s.jobs {
				if now.Before(j.next) {
					break
				}

				if j.running > 0 {
					debugf("Scheduled job '%v' still active.", j.id)
					reschedActive(j, now)
					continue
				}

				if s.limit > 0 && s.active == s.limit {
					logp.Info("Scheduled job '%v' waiting.", j.id)
					timer = nil
					continue
				}

				s.startJob(j)
			}

		case sig := <-s.finished:
			s.active--
			j := sig.entry
			debugf("Job '%v' returned at %v (cont=%v).", j.id, time.Now(), len(sig.cont))

			// add number of job continuation tasks returned to current job task
			// counter and remove count for task just being finished
			j.running += uint32(len(sig.cont)) - 1

			count := 0 // number of rescheduled waiting jobs

			// try to start waiting jobs
			for _, waiting := range s.jobs {
				if now.Before(waiting.next) {
					break
				}

				if waiting.running > 0 {
					count++
					reschedActive(waiting, now)
					continue
				}

				debugf("Start waiting job: %v", waiting.id)
				s.startJob(waiting)
				break
			}

			// Try to start waiting tasks of already running jobs.
			// The s.tasks waiting list will only have any entries if `s.limit > 0`.
			if s.limit > 0 && (s.active < s.limit) {
				if T := uint(len(s.tasks)); T > 0 {
					N := s.limit - s.active
					debugf("start up to %v waiting tasks (%v)", N, T)
					if N > T {
						N = T
					}

					tasks := s.tasks[:N]
					s.tasks = s.tasks[N:]
					for _, t := range tasks {
						s.runTask(t)
					}
				}
			}

			// try to start returned tasks for current job and put left-over tasks into
			// waiting list.
			if N := len(sig.cont); N > 0 {
				if s.limit > 0 {
					limit := int(s.limit - s.active)
					if N > limit {
						N = limit
					}
				}

				if N > 0 {
					debugf("start returned tasks")
					tasks := sig.cont[:N]
					sig.cont = sig.cont[N:]
					for _, t := range tasks {
						s.runTask(t)
					}
				}
			}
			if len(sig.cont) > 0 {
				s.tasks = append(s.tasks, sig.cont...)
			}

			// reschedule (sort) list of tasks, if any task to be run next is
			// still active.
			resched = count > 0

		case j := <-s.addCh:
			j.next = j.schedule.Next(time.Now().In(s.location))
			s.addSync(j)

		case j := <-s.rmCh:
			s.doRemove(j)

		case <-s.done:
			debugf("done")
			return

		}
	}
}

func reschedActive(j *job, now time.Time) {
	logp.Info("Scheduled job '%v' already active.", j.id)
	if !now.Before(j.next) {
		j.next = j.schedule.Next(j.next)
	}
}

func (s *Scheduler) startJob(j *job) {
	j.running++
	j.next = j.schedule.Next(j.next)
	debugf("Start job '%v' at %v.", j.id, time.Now())

	s.runTask(task{j, j.fn})
}

func (s *Scheduler) runTask(t task) {
	j := t.job
	s.active++

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logp.Err("Panic in job '%v'. Recovering, but please report this: %s.",
					j.id, r)
				logp.Err("Stacktrace: %s", debug.Stack())
				s.signalFinished(j, nil)
			}
		}()

		cont := t.fn()
		s.signalFinished(j, cont)
	}()
}

func (s *Scheduler) addSync(j *job) {
	j.registered = true
	s.jobs = append(s.jobs, j)
}

func (s *Scheduler) doRemove(j *job) {
	// find entry
	idx := -1
	for i, other := range s.jobs {
		if j == other {
			idx = i
			break
		}
	}
	if idx == -1 {
		return
	}

	// delete entry, not preserving order
	s.jobs[idx] = s.jobs[len(s.jobs)-1]
	s.jobs = s.jobs[:len(s.jobs)-1]

	// mark entry as unregistered
	j.registered = false
}

func (s *Scheduler) signalFinished(j *job, cont []TaskFunc) {
	var tasks []task
	if len(cont) > 0 {
		tasks = make([]task, len(cont))
		for i, f := range cont {
			tasks[i] = task{j, f}
		}
	}

	s.finished <- taskOverSignal{j, tasks}
}

func (s *Scheduler) transitionRunning() bool {
	return s.state.CAS(statePreRunning, stateRunning)
}

func (s *Scheduler) transitionStopped() bool {
	return s.state.CAS(stateRunning, stateDone)
}

func (s *Scheduler) isPreRunning() bool {
	return s.state.Load() == statePreRunning
}

func (s *Scheduler) isRunning() bool {
	return s.state.Load() == stateRunning
}
