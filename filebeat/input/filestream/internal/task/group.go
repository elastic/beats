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

package task

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
)

type Group struct {
	sem *semaphore.Weighted
	wg  *sync.WaitGroup

	stopTimeout time.Duration
	logErr      func(error)

	ctx       context.Context
	cancelCtx context.CancelFunc
}

type Logger interface {
	Errorf(format string, args ...interface{})
}

// NewGroup returns a new task group which will run tasks on a goroutine. See
// Group.Go for details.
//
// The number of concurrent running tasks is limited by limit, if limit is zero,
// the group is unlimited.
//
// The group can be stopped by calling Group.Stop which will close the group,
// it'll not accept new tasks, and Group.Stop will wait for all running tasks to
// complete or stopTimeout to elapse, what ever comes first.
//
// log is used to log any error returned by the tasks or internal errors. The
// log message will be prefixed by errPrefix.
func NewGroup(limit uint64, stopTimeout time.Duration, log Logger, errPrefix string) *Group {
	ctx, cancel := context.WithCancel(context.Background())

	var logErr = func(error) {}
	if log != nil {
		var format string
		if errPrefix == "" {
			format = "%v"
		} else {
			format = errPrefix + ": %v"
		}

		logErr = func(err error) {
			log.Errorf(format, err)
		}
	}

	var sem *semaphore.Weighted
	if limit > 0 {
		sem = semaphore.NewWeighted(int64(limit))
	}

	return &Group{
		cancelCtx:   cancel,
		ctx:         ctx,
		logErr:      logErr,
		sem:         sem,
		stopTimeout: stopTimeout,
		wg:          &sync.WaitGroup{},
	}
}

// Go starts fn on a goroutine. If the limit of concurrent tasks has been reached,
// fn will wait until it can run. Go won't block while fn waits to run.
// If the group was already closed, Go returns a context.Canceled error.
// If the limit of concurrent tasks has been reached and Group.Stop() is called,
// fn is discarded and an error is logged.
func (g *Group) Go(fn func(context.Context) error) error {
	if err := g.ctx.Err(); err != nil {
		return fmt.Errorf("task group is closed: %w", err)
	}

	g.wg.Add(1)
	go func() {
		defer g.wg.Done()

		if g.sem != nil {
			err := g.sem.Acquire(g.ctx, 1)
			if err != nil {
				//nolint:errorlint // it's intentional
				g.logErr(fmt.Errorf(
					"task.Group: semaphore acquire failed, was the task group closed? err: %v",
					err))
				return
			}
			defer g.sem.Release(1)
		}

		err := fn(g.ctx)
		if err != nil {
			g.logErr(err)
		}
	}()

	return nil
}

// Stop stops the task group accepting new goroutines and waits until all
// running tasks to finish or the stop timeout to elapse, whatever
// happens first. It returns an error if the timout is reached, nil otherwise.
func (g *Group) Stop() error {
	g.cancelCtx()

	done := make(chan struct{})
	go func() {
		g.wg.Wait()
		done <- struct{}{}
	}()

	timeout, cancel := context.WithTimeout(context.Background(), g.stopTimeout)
	defer cancel()

	select {
	case <-timeout.Done():
		return fmt.Errorf("task group Stop timeout: %w", timeout.Err())
	case <-done:
		return nil
	}
}
