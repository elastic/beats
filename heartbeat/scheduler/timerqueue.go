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
	"container/heap"
	"context"
	"sync"
	"time"
)

type TimerTask struct {
	fn    TimerTaskFn
	id    string
	runAt time.Time
}

type TimerTaskFn func(now time.Time)

type TimerQueue struct {
	th              *TimerHeap
	mtx             *sync.Mutex
	nextRunAt       *time.Time
	nextRunAtChange chan time.Time
}

// NewTimerQueue creates a new instance.
func NewTimerQueue() *TimerQueue {
	mtx := &sync.Mutex{}
	tq := &TimerQueue{
		th:              &TimerHeap{},
		mtx:             mtx,
		nextRunAtChange: make(chan time.Time),
	}
	heap.Init(tq.th)

	return tq
}

// Push adds a task to the queue
func (tq *TimerQueue) Push(tt *TimerTask) {
	tq.mtx.Lock()

	heap.Push(tq.th, tt)

	if tq.nextRunAt == nil || tt.runAt.Before(*tq.nextRunAt) {
		tq.nextRunAt = &tt.runAt
		tq.mtx.Unlock()
		tq.nextRunAtChange <- tt.runAt
	} else {
		tq.mtx.Unlock()
	}
}

// PopRunnable pops as many runnable tasks from the queue as possible
func (tq *TimerQueue) PopRunnable() (res []*TimerTask, newNext *time.Time) {
	tq.mtx.Lock()
	defer tq.mtx.Unlock()

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

	if tq.th.Len() == 0 {
		tq.nextRunAt = nil
	}

	// make a copy of the nextRunAt pointer for threadsafety
	return res, &(*tq.nextRunAt)
}

// Start runs a goroutine within the given context that processes items in the queue, spawning a new goroutine
// for each.
func (tq *TimerQueue) Start(ctx context.Context) {
	// Timer runs every 10ms
	timer := time.NewTimer(1)
	go func() {
		for {
			var newNext *time.Time
			select {
			case <-ctx.Done():
				// Stop the timerqueue
				return
			case <-timer.C:
				newNext = tq.RunRunnableTasks()
			case changed := <-tq.nextRunAtChange:
				newNext = &changed
				timer.Stop()
			}

			if newNext != nil {
				nextIn := newNext.Sub(time.Now())
				if nextIn < 1 {
					timer.Reset(1)
				} else {
					timer.Reset(nextIn)
				}
			}
		}
	}()
}

// RunRunnableTasks runs all tasks that are currently runnable
func (tq *TimerQueue) RunRunnableTasks() *time.Time {
	runnable, newNext := tq.PopRunnable()
	now := time.Now()
	for _, tt := range runnable {
		go tt.fn(now)
	}
	return newNext
}
