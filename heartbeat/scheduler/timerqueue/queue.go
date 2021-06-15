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
	"sync"
	"time"
)

// timerTask represents a task run by the TimerQueue.
type timerTask struct {
	fn    TimerTaskFn
	runAt time.Time
}

// TimerTaskFn is the function invoked by a timerTask.
type TimerTaskFn func(now time.Time)

// TimerQueue represents a priority queue of timers.
type TimerQueue struct {
	doneWg sync.WaitGroup
	th     timerHeap
	ctx    context.Context
	ticker *time.Ticker
	pushCh chan *timerTask
	runNow chan time.Time
}

// NewTimerQueue creates a new instance.
func NewTimerQueue(ctx context.Context) *TimerQueue {
	tq := &TimerQueue{
		th:     timerHeap{},
		ctx:    ctx,
		pushCh: make(chan *timerTask, 4096),
		runNow: make(chan time.Time),
	}
	heap.Init(&tq.th)

	return tq
}

// Push adds a task to the queue. Returns true if successful
// false if failed (due to cancelled context)
func (tq *TimerQueue) Push(runAt time.Time, fn TimerTaskFn) bool {
	// Block until push succeeds or shutdown
	select {
	case tq.pushCh <- &timerTask{runAt: runAt, fn: fn}:
		return true
	case <-tq.ctx.Done():
		return false
	}
}

// Start runs a goroutine within the given context that processes items in the queue, spawning a new goroutine
// for each.
func (tq *TimerQueue) Start() {
	tq.doneWg.Add(1)
	tq.ticker = time.NewTicker(time.Millisecond * 10)
	go func() {
		defer tq.doneWg.Done()
		for {
			select {
			case <-tq.ctx.Done():
				tq.ticker.Stop()
				return
			case now := <-tq.ticker.C:
				tq.runTasksInternal(now)
			case tt := <-tq.pushCh:
				heap.Push(&tq.th, tt)
				// If some items were scheduled to run right now, do it quickly!
				tq.runTasksInternal(time.Now())
			}

		}
	}()
}

func (tq *TimerQueue) runTasksInternal(now time.Time) {
	// Look ahead 5ms and grab soonish tasks
	tasks := tq.popRunnable(now.Add(time.Millisecond * 10))

	// Run the tasks in a separate goroutine so we can unblock the thread here for pushes etc.
	go func() {
		for _, tt := range tasks {
			tt.fn(now)
		}
	}()
}

func (tq *TimerQueue) popRunnable(now time.Time) (res []*timerTask) {
	for i := 0; tq.th.Len() > 0; i++ {
		// the zeroth element of the heap is the same as a peek
		peeked := tq.th[0]
		if peeked.runAt.Before(now) {
			popped := heap.Pop(&tq.th).(*timerTask)
			res = append(res, popped)
		} else {
			break
		}
	}

	return res
}
