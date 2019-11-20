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

package timerqueue

import (
	"container/heap"
	"context"
	"time"
)

// TimerTask represents a task run by the TimerQueue.
type TimerTask struct {
	fn    TimerTaskFn
	runAt time.Time
}

// NewTimerTask creates a new TimerTask struct.
func NewTimerTask(runAt time.Time, fn TimerTaskFn) *TimerTask {
	return &TimerTask{runAt: runAt, fn: fn}
}

// TimerTaskFn is the function invoked by a TimerTask.
type TimerTaskFn func(now time.Time)

// TimerQueue represents a priority queue of timers.
type TimerQueue struct {
	th            *timerHeap
	ctx           context.Context
	nextRunAt     *time.Time
	popRunnableCh chan chan []*TimerTask
	pushCh        chan *TimerTask
	timer         *time.Timer
}

// NewTimerQueue creates a new instance.
func NewTimerQueue(ctx context.Context) *TimerQueue {
	tq := &TimerQueue{
		th:            &timerHeap{},
		ctx:           ctx,
		popRunnableCh: make(chan chan []*TimerTask),
		pushCh:        make(chan *TimerTask),
		timer:         time.NewTimer(0),
	}
	heap.Init(tq.th)

	return tq
}

// Push adds a task to the queue
func (tq *TimerQueue) Push(tt *TimerTask) {
	// Block until push succeeds or shutdown
	select {
	case tq.pushCh <- tt:
	case <-tq.ctx.Done():
	}
}

// PopRunnable pops as many runnable tasks from the queue as possible
func (tq *TimerQueue) PopRunnable() (res []*TimerTask) {
	popDone := make(chan []*TimerTask)
	tq.popRunnableCh <- popDone

	select {
	case res = <-popDone:
		return res
	case <-tq.ctx.Done():
		return
	}
}

// RunRunnableTasks runs all tasks that are currently runnable. Tasks are run in serial blocking manner in the
// current go-routine
func (tq *TimerQueue) RunRunnableTasks() {
	runnable := tq.PopRunnable()
	now := time.Now()
	for _, tt := range runnable {
		tt.fn(now)
	}
}

// Start runs a goroutine within the given context that processes items in the queue, spawning a new goroutine
// for each.
func (tq *TimerQueue) Start() {
	go func() {
		for {
			// flag controlling whether we should update tq.nextRunAt after the select block
			updateNextRunAt := false

			select {
			case <-tq.ctx.Done():
				// Stop the timerqueue
				return
			case <-tq.timer.C:
				tq.RunRunnableTasks()
				updateNextRunAt = true
			case retCh := <-tq.popRunnableCh:
				res := tq.popRunnableUnsafe()
				updateNextRunAt = true
				retCh <- res
			case tt := <-tq.pushCh:
				tq.pushUnsafe(tt)
			}

			if updateNextRunAt {
				tq.updateTimer()
			}
		}
	}()
}

func (tq *TimerQueue) updateTimer() {
	if tq.th.Len() == 0 {
		tq.nextRunAt = nil
		tq.timer.Stop()
	} else {
		now := time.Now()

		peeked := (*tq.th)[0]
		nextIn := peeked.runAt.Sub(now)
		tq.nextRunAt = &peeked.runAt
		tq.timer.Reset(nextIn)
	}
}

func (tq *TimerQueue) popRunnableUnsafe() (res []*TimerTask) {
	now := time.Now()
	for i := 0; i < tq.th.Len(); i++ {
		// the zeroth element of the heap is the same as a peek
		peeked := (*tq.th)[0]
		if peeked.runAt.Before(now) {
			popped := heap.Pop(tq.th).(*TimerTask)
			res = append(res, popped)
		} else {
			tq.nextRunAt = &peeked.runAt
			break
		}
	}

	return res
}

func (tq *TimerQueue) pushUnsafe(tt *TimerTask) {
	heap.Push(tq.th, tt)

	if tq.nextRunAt == nil || tt.runAt.Before(*tq.nextRunAt) {
		tq.nextRunAt = &tt.runAt
	}
}
