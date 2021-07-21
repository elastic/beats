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

// timerTask represents a task run by the TimerQueue.
type timerTask struct {
	fn    TimerTaskFn
	runAt time.Time
}

// TimerTaskFn is the function invoked by a timerTask.
type TimerTaskFn func(now time.Time)

// TimerQueue represents a priority queue of timers.
type TimerQueue struct {
	th        timerHeap
	ctx       context.Context
	nextRunAt *time.Time
	pushCh    chan *timerTask
	timer     *time.Timer
}

// NewTimerQueue creates a new instance.
func NewTimerQueue(ctx context.Context) *TimerQueue {
	tq := &TimerQueue{
		th:     timerHeap{},
		ctx:    ctx,
		pushCh: make(chan *timerTask, 4096),
		timer:  time.NewTimer(time.Hour * 86400),
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
	go func() {
		for {
			select {
			case <-tq.ctx.Done():
				// Stop the timerqueue
				return
			case now := <-tq.timer.C:
				tasks := tq.popRunnable(now)

				// Run the tasks in a separate goroutine so we can unblock the thread here for pushes etc.
				go func() {
					for _, tt := range tasks {
						tt.fn(now)
					}
				}()

				if tq.th.Len() > 0 {
					nr := tq.th[0].runAt
					tq.nextRunAt = &nr
					tq.timer = time.NewTimer(time.Until(nr))
				} else {
					tq.timer.Stop()
					tq.nextRunAt = nil
				}
			case tt := <-tq.pushCh:
				tq.pushInternal(tt)
			}
		}
	}()
}

func (tq *TimerQueue) pushInternal(tt *timerTask) {
	heap.Push(&tq.th, tt)

	if tq.nextRunAt == nil || tq.nextRunAt.After(tt.runAt) {
		if tq.nextRunAt != nil && !tq.timer.Stop() {
			<-tq.timer.C
		}
		// Originally the line below this comment was
		//
		//  tq.timer.Reset(time.Until(tt.runAt))
		//
		// however this broke in go1.16rc1, specifically on the commit b4b014465216790e01aa66f9120d03230e4aff46
		//, specifically on this line:
		// https://github.com/golang/go/commit/b4b014465216790e01aa66f9120d03230e4aff46#diff-73699b6edfe5dbb3f6824e66bb3566bce9405e9a8c810cac55c8199459f0ac19R652
		// where some nice new optimizations don't actually work reliably
		// This can be worked around by instantiating a new timer rather than resetting the timer.
		// since that internally calls deltimer in runtime/timer.go rather than modtimer,
		// I suspect that the problem is in modtimer's setting of &pp.timerModifiedEarliest
		tq.timer = time.NewTimer(time.Until(tt.runAt))
		tq.nextRunAt = &tt.runAt
	}
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
