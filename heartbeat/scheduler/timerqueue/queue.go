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
	"fmt"
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
	th        *timerHeap
	ctx       context.Context
	nextRunAt *time.Time
	pushCh    chan *TimerTask
	timer     *time.Timer
}

// NewTimerQueue creates a new instance.
func NewTimerQueue(ctx context.Context) *TimerQueue {
	tq := &TimerQueue{
		th:     &timerHeap{},
		ctx:    ctx,
		pushCh: make(chan *TimerTask, 4096),
		timer:  time.NewTimer(0),
	}
	heap.Init(tq.th)

	return tq
}

// Push adds a task to the queue. Returns true if successful
// false if failed (due to cancelled context)
func (tq *TimerQueue) Push(tt *TimerTask) bool {
	// Block until push succeeds or shutdown
	select {
	case tq.pushCh <- tt:
		fmt.Printf("PUSH %s\n", tt.runAt)
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
					nr := (*tq.th)[0].runAt
					tq.nextRunAt = &nr
					tq.timer.Reset(nr.Sub(time.Now()))
				} else {
					tq.timer.Reset(10 * time.Millisecond)
					tq.nextRunAt = nil
				}
			case tt := <-tq.pushCh:
				tq.pushInternal(tt)
			}
		}
	}()
}

func (tq *TimerQueue) pushInternal(tt *TimerTask) {
	heap.Push(tq.th, tt)

	if tq.nextRunAt == nil || tq.nextRunAt.After(tt.runAt) {
		tq.timer.Stop()
		tq.nextRunAt = &tt.runAt
		tq.timer.Reset(tt.runAt.Sub(time.Now()))
	}
}

func (tq *TimerQueue) popRunnable(now time.Time) (res []*TimerTask) {
	for i := 0; tq.th.Len() > 0; i++ {
		// the zeroth element of the heap is the same as a peek
		peeked := (*tq.th)[0]
		if peeked.runAt.Before(now) {
			popped := heap.Pop(tq.th).(*TimerTask)
			res = append(res, popped)
		} else {
			break
		}
	}

	return res
}
