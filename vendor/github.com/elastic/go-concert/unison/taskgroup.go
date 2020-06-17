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
	"sync"

	"github.com/elastic/go-concert/chorus"
	"github.com/urso/sderr"
)

// Group interface, that can be used to start tasks. The tasks started will
// spawn go-routines, and will get a shutdown signal by the provided Canceler.
type Group interface {
	// Go method returns an error if the task can not be started. The error
	// returned by the task itself is not supposed to be returned, as the error is
	// assumed to be generated asynchronously.
	Go(fn func(Canceler) error) error
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

func (c closedGroup) Go(_ func(Canceler) error) error {
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
	// StopOnError  configures the behavior when a sub-task failed. If not set
	// all other tasks will continue to run. If the function return true, a
	// shutdown signal is passed, and Go will fail on attempts to start new
	// tasks.
	StopOnError func(err error) bool

	mu   sync.Mutex
	errs []error
	wg   SafeWaitGroup

	initOnce sync.Once
	closer   *chorus.Closer
}

var _ Group = (*TaskGroup)(nil)

// init initializes internal state the first time the group is actively used.
func (t *TaskGroup) init() {
	t.initOnce.Do(func() {
		t.closer = chorus.NewCloser(nil)
	})
}

// Go starts a new go-routine and passes a Canceler to signal group shutdown.
// Errors returned by the function are collected and finally returned on Stop.
// If the group was stopped before calling Go, then Go will return the
// ErrGroupClosed error.
func (t *TaskGroup) Go(fn func(Canceler) error) error {
	t.init()

	if err := t.wg.Add(1); err != nil {
		return err
	}

	go func() {
		defer t.wg.Done()
		err := fn(t.closer)
		if err != nil {
			t.mu.Lock()
			t.errs = append(t.errs, err)
			t.mu.Unlock()

			if t.StopOnError != nil && t.StopOnError(err) {
				t.wg.Close()
				t.closer.Close()
			}
		}
	}()

	return nil
}

func (t *TaskGroup) wait() []error {
	t.wg.Wait()
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.errs
}

// Stop sends a shutdown signal to all tasks, and waits for them to finish.
// It returns an error that contains all errors encountered.
func (t *TaskGroup) Stop() error {
	t.init()
	t.closer.Close()
	errs := t.wait()
	if len(errs) > 0 {
		return sderr.WrapAll(errs, "task failures")
	}
	return nil
}
