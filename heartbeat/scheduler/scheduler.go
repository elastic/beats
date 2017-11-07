package scheduler

import (
	"errors"
	"runtime/debug"
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/logp"
)

type Scheduler struct {
	limit   uint
	running bool

	location *time.Location

	jobs   []*job
	active uint // number of active entries

	add, rm  chan *job
	finished chan taskOverSignal

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
	name     string
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
	Next(time.Time) time.Time
}

var debugf = logp.MakeDebug("scheduler")

func New(limit uint) *Scheduler {
	return NewWithLocation(limit, time.Local)
}

func NewWithLocation(limit uint, location *time.Location) *Scheduler {
	return &Scheduler{
		limit:    limit,
		location: location,

		running: false,
		jobs:    nil,
		active:  0,

		add:      make(chan *job),
		rm:       make(chan *job),
		finished: make(chan taskOverSignal),

		done: make(chan struct{}),
		wg:   sync.WaitGroup{},
	}
}

func (s *Scheduler) Start() error {
	if s.running {
		return errors.New("scheduler already running")
	}

	s.running = true
	go s.run()
	return nil
}

func (s *Scheduler) Stop() error {
	if !s.running {
		return errors.New("scheduler already stopped")
	}

	s.running = false
	close(s.done)
	s.wg.Wait()
	return nil
}

func (s *Scheduler) Add(sched Schedule, name string, entrypoint TaskFunc) func() error {
	debugf("Add scheduler job '%v'.", name)

	j := &job{
		name:       name,
		fn:         entrypoint,
		schedule:   sched,
		registered: false,
		running:    0,
	}
	if !s.running {
		s.doAdd(j)
	} else {
		s.add <- j
	}

	return func() error { return s.remove(j) }
}

func (s *Scheduler) remove(j *job) error {
	debugf("Remove scheduler job '%v'", j.name)

	if !s.running {
		s.doRemove(j)
	} else {
		s.rm <- j
	}

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

		if (s.limit == 0 || s.active < s.limit) && len(s.jobs) > 0 {
			next := s.jobs[0].next
			debugf("Next wakeup time: %v", next)

			if timer != nil {
				timer.Stop()
			}
			timer = time.NewTimer(next.Sub(time.Now().In(s.location)))
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
					debugf("Scheduled job '%v' still active.", j.name)
					reschedActive(j, now)
					continue
				}

				if s.limit > 0 && s.active == s.limit {
					logp.Info("Scheduled job '%v' waiting.", j.name)
					timer = nil
					continue
				}

				s.startJob(j)
			}

		case sig := <-s.finished:
			s.active--
			j := sig.entry
			debugf("Job '%v' returned at %v (cont=%v).", j.name, time.Now(), len(sig.cont))

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

				debugf("Start waiting job: %v", waiting.name)
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

		case j := <-s.add:
			j.next = j.schedule.Next(time.Now().In(s.location))
			s.doAdd(j)

		case j := <-s.rm:
			s.doRemove(j)

		case <-s.done:
			debugf("done")
			return

		}
	}
}

func reschedActive(j *job, now time.Time) {
	logp.Info("Scheduled job '%v' already active.", j.name)
	if !now.Before(j.next) {
		j.next = j.schedule.Next(j.next)
	}
}

func (s *Scheduler) startJob(j *job) {
	j.running++
	j.next = j.schedule.Next(j.next)
	debugf("Start job '%v' at %v.", j.name, time.Now())

	s.runTask(task{j, j.fn})
}

func (s *Scheduler) runTask(t task) {
	j := t.job
	s.active++

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logp.Err("Panic in job '%v'. Recovering, but please report this: %s.",
					j.name, r)
				logp.Err("Stacktrace: %s", debug.Stack())
				s.signalFinished(j, nil)
			}
		}()

		cont := t.fn()
		s.signalFinished(j, cont)
	}()
}

func (s *Scheduler) doAdd(j *job) {
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
