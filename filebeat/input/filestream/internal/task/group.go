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

type Goer interface {
	Go(fn func(context.Context) error) error
}

type Logger interface {
	Errorf(format string, args ...interface{})
}

func NewGroup(limit uint64, stopTimeout time.Duration, log Logger, errFormat string) *Group {
	ctx, cancel := context.WithCancel(context.Background())

	var logErr = func(error) {}
	if log != nil {
		logErr = func(err error) {
			log.Errorf(errFormat, err)
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

// Go starts fn on a goroutine when a worker becomes available.
// If the worker pool was already closed, Go returns a context.Canceled error.
// If there are no workers available and Group.Stop() is called, fn is discarded.
// Go does not block.
func (g *Group) Go(fn func(context.Context) error) error {
	if err := g.ctx.Err(); err != nil {
		return fmt.Errorf("task group is closed: %w", err)
	}

	g.wg.Add(1)
	go func() {
		defer g.wg.Done()

		if g.sem != nil {
			err := g.sem.Acquire(g.ctx, 1)
			defer g.sem.Release(1)
			if err != nil {
				g.logErr(fmt.Errorf(
					"task.Group: semaphore acquire failed, was the task group closed? err: %v",
					err))
				return
			}
		}

		err := fn(g.ctx)
		if err != nil {
			g.logErr(err)
		}
		return
	}()

	return nil
}

// Stop stops the task group accepting new goroutines to launch and waits until
// either all running tasks finishes or the stop timeout is reached, whatever
// happens first. It returns an error if the timout is reached or nil otherwise.
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
