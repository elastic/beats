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

package unison

import (
	"context"
	"sync"

	"github.com/elastic/go-concert/ctxtool"
	"github.com/urso/sderr"
)

// Group interface, that can be used to start tasks. The tasks started will
// spawn go-routines, and will get a shutdown signal by the provided Canceler.
type Group interface {
	// Go method returns an error if the task can not be started. The error
	// returned by the task itself is not supposed to be returned, as the error is
	// assumed to be generated asynchronously.
	Go(fn func(context.Context) error) error
}

// Canceler interface, that can be used to pass along some shutdown signal to
// child goroutines.
type Canceler interface {
	Done() <-chan struct{}
	Err() error
}

type closedGroup struct {
	err error
}

// ClosedGroup creates a Group that always fails to start a go-routine.
// Go will return reportedError on each attempt to create a go routine.
// If reportedError is nil, ErrGroupClosed will be used.
func ClosedGroup(reportedError error) Group {
	if reportedError == nil {
		reportedError = ErrGroupClosed
	}
	return &closedGroup{err: reportedError}
}

func (c closedGroup) Go(_ func(context.Context) error) error {
	return c.err
}

// TaskGroup implements the Group interface. Once the group is shutting down,
// no more goroutines can be created via Go.
// The Stop method of TaskGroup will block until all sub-tasks have returned.
// Errors from sub-tasks are collected. The Stop method collects all errors and returns
// a single error summarizing all errors encountered.
//
// By default sub-tasks continue running if any task did encounter an error.
// This behavior can be modified by setting StopOnError.
//
// The zero value of TaskGroup is fully functional. StopOnError must not be set after
// the first go-routine has been spawned.
type TaskGroup struct {
	// OnQuit  configures the behavior when a sub-task returned. If not set
	// all other tasks will continue to run. If the function return true, a
	// shutdown signal is passed, and Go will fail on attempts to start new
	// tasks.
	// Next to the action, does the OnQuit error also return the error value
	// to be recorded. The context.Cancel error will never be recorded.
	//
	// Common OnError handlers are given by ContinueOnErrors, StopOnError,
	// StopOnErrorOrCancel.
	// By default StopOnError will be used.
	OnQuit TaskGroupQuitHandler

	// MaxErrors configures the maximum amount of errors the TaskGroup will record.
	// Older errors will be replaced once the limit is exceeded.
	// If MaxErrors is set to a value < 0, all errors will be recorded.
	MaxErrors int

	mu   sync.Mutex
	errs []error
	wg   SafeWaitGroup

	initOnce sync.Once
	closer   context.Context
	cancel   context.CancelFunc
}

type TaskGroupQuitHandler func(error) (TaskGroupStopAction, error)

// TaskGroupStopAction signals the action to take when a go-routine owned by a
// TaskGroup did quit.
type TaskGroupStopAction uint

const (
	// TaskGroupStopActionContinue notifies the TaskGroup that other managed go-routines
	// should not be signalled to shutdown.
	TaskGroupStopActionContinue TaskGroupStopAction = iota

	// TaskGroupStopActionShutdown notifies the TaskGroup that shutdown should be signaled
	// to all maanaged go-routines.
	TaskGroupStopActionShutdown

	// TaskGroupStopActionRestart signals the TaskGroup that the managed go-routine that has
	// just been returned should be restarted.
	TaskGroupStopActionRestart
)

var _ Group = (*TaskGroup)(nil)

// init initializes internal state the first time the group is actively used.
func (t *TaskGroup) init(parent Canceler) {
	t.initOnce.Do(func() {
		t.closer, t.cancel = context.WithCancel(ctxtool.FromCanceller(parent))
		if t.OnQuit == nil {
			t.OnQuit = StopOnError
		}
		if t.MaxErrors == 0 {
			t.MaxErrors = 10
		}
	})
}

// TaskGroupWithCancel creates a TaskGroup that gets stopped when the parent context
// signals shutdown or the Stop method is called.
//
// Although the managed go-routines are signalled to stop when the parent context is done,
// one still might want to call Stop in order to wait for the managed go-routines to stop.
//
// Associated resources are cleaned when the parent context is cancelled, or Stop is called.
func TaskGroupWithCancel(canceler Canceler) *TaskGroup {
	t := &TaskGroup{}
	t.init(canceler)
	return t
}

// Go starts a new go-routine and passes a Canceler to signal group shutdown.
// Errors returned by the function are collected and finally returned on Stop.
// If the group was stopped before calling Go, then Go will return the
// ErrGroupClosed error.
func (t *TaskGroup) Go(fn func(context.Context) error) error {
	t.init(context.Background())

	if err := t.wg.Add(1); err != nil {
		return err
	}

	go func() {
		defer t.wg.Done()

		for t.closer.Err() == nil {
			err := fn(t.closer)
			action, err := t.OnQuit(err)

			if err != nil && err != context.Canceled {
				t.mu.Lock()
				t.errs = append(t.errs, err)
				if t.MaxErrors > 0 && len(t.errs) > t.MaxErrors {
					t.errs = t.errs[1:]
				}
				t.mu.Unlock()
			}

			switch action {
			case TaskGroupStopActionContinue:
				return // finish managed go-routine, but keep other routines alive.
			case TaskGroupStopActionShutdown:
				t.signalStop()
				return
			case TaskGroupStopActionRestart:
				// continue with loop
			}
		}
	}()

	return nil
}

// Context returns the task groups internal context.
// The internal context will be cancelled if the groups parent context gets
// cancelled, or Stop has been called.
func (t *TaskGroup) Context() context.Context {
	t.init(context.Background())
	return t.closer
}

// Wait blocks until all owned child routines have been stopped.
func (t *TaskGroup) Wait() error {
	errs := t.waitErrors()
	if len(errs) > 0 {
		return sderr.WrapAll(errs, "task failures")
	}
	return nil
}

func (t *TaskGroup) waitErrors() []error {
	t.wg.Wait()
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.errs
}

// Stop sends a shutdown signal to all tasks, and waits for them to finish.
// It returns an error that contains all errors encountered.
func (t *TaskGroup) Stop() error {
	t.init(context.Background())
	t.signalStop()
	return t.Wait()
}

// signalStop will cancel the internal context, signaling existing go-routines
// to shutdown AND invalidate the TaskGroup, such that no new go-routines can
// be started anymore.
func (t *TaskGroup) signalStop() {
	t.wg.Close()
	t.cancel()
}

// ContinueOnErrors provides a TaskGroup.OnQuit handler, that will ignore
// any errors. Other go-routines owned by the TaskGroup will continue to run.
func ContinueOnErrors(err error) (TaskGroupStopAction, error) {
	return TaskGroupStopActionContinue, err
}

// RestartOnError provides a TaskGroup.OnQuit handler, that will restart a
// go-routine if the routine failed with an error.
func RestartOnError(err error) (TaskGroupStopAction, error) {
	if err != nil && err != context.Canceled {
		return TaskGroupStopActionRestart, err
	}
	return TaskGroupStopActionContinue, err
}

// StopAll provides a Taskgroup.OnError handler, that will signal
// the TaskGroup to shutdown once an owned go-routine returns.
// The TaskGroup is supposed to stop even on successful return.
func StopAll(err error) (TaskGroupStopAction, error) {
	return TaskGroupStopActionShutdown, err
}

// StopOnError provides a TaskGroup.OnError handler, that will signal the Taskgroup
// to stop all owned go-routines.
// The context.Canceled error value will be ignored.
func StopOnError(err error) (TaskGroupStopAction, error) {
	if err != nil && err != context.Canceled {
		return TaskGroupStopActionShutdown, err
	}
	return TaskGroupStopActionContinue, err
}

// StopOnErrorOrCancel provides a TaskGroup.OnError handler, that will signal the Taskgroup
// to stop all owned go-routines.
func StopOnErrorOrCancel(err error) (TaskGroupStopAction, error) {
	if err != nil {
		return TaskGroupStopActionShutdown, err
	}
	return TaskGroupStopActionContinue, err
}
