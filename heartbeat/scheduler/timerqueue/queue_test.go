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
	"context"
	"math/rand"
	"os"
	"runtime/pprof"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRunsInOrder(t *testing.T) {
	testQueueRunsInOrderOnce(t)
}

// TestStress tries to figure out if we have any deadlocks that show up under concurrency
func TestStress(t *testing.T) {
	for i := 0; i < 120000; i++ {
		failed := make(chan bool)
		succeeded := make(chan bool)

		watchdog := time.AfterFunc(time.Second*5, func() {
			failed <- true
		})

		go func() {
			testQueueRunsInOrderOnce(t)
			succeeded <- true
		}()

		select {
		case <-failed:
			pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
			require.FailNow(t, "Scheduler test iteration timed out, deadlock issue?")
		case <-succeeded:
			watchdog.Stop()
		}
	}
}

func testQueueRunsInOrderOnce(t *testing.T) {
	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()
	tq := NewTimerQueue(ctx)

	// Number of items to test with
	numItems := 10

	// Make a buffered queue for taskResCh so we can easily write to it within this thread.
	taskResCh := make(chan int, numItems)

	// Make a bunch of tasks past their deadline
	var tasks []*timerTask
	// Start from 1 so we can use the zero value when closing the channel
	for i := 1; i <= numItems; i++ {
		func(i int) {
			schedFor := time.Unix(0, 0).Add(time.Duration(i))
			tasks = append(tasks, &timerTask{runAt: schedFor, fn: func(now time.Time) {
				taskResCh <- i
				if i == numItems {
					close(taskResCh)
				}
			}})
		}(i)
	}
	// shuffle them so they're out of order
	rand.Shuffle(len(tasks), func(i, j int) { tasks[i], tasks[j] = tasks[j], tasks[i] })

	// insert the randomly ordered events into the queue
	// we use the internal push because pushing and running are in the same threads, so
	// using Push() may result in tasks being executed before all are inserted.
	// This private method is not threadsafe, so is kept private.
	for _, tt := range tasks {
		tq.pushInternal(tt)
	}

	tq.Start()

	var taskResults []int
Reader:
	for {
		select {
		case res := <-taskResCh:
			if res == 0 { // chan closed
				break Reader
			}
			taskResults = append(taskResults, res)
		}
	}

	require.Len(t, taskResults, numItems)
	require.True(t, sort.IntsAreSorted(taskResults), "Results not in order! %v", taskResults)
}

func TestQueueRunsTasksAddedAfterStart(t *testing.T) {
	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()
	tq := NewTimerQueue(ctx)

	tq.Start()

	resCh := make(chan int)
	tq.Push(time.Now(), func(now time.Time) {
		resCh <- 1
	})

	select {
	case r := <-resCh:
		require.Equal(t, 1, r)
	}
}
