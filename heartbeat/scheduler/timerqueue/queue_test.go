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
	"fmt"
	"math/rand"
	"os"
	"runtime/pprof"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestQueueRunsInOrder(t *testing.T) {
	// Bugs can show up only occasionally
	var min time.Duration
	var max time.Duration
	var slow int
	var failures int
	for i := 0; i < 1000000; i++ {
		start := time.Now()

		sd := time.AfterFunc(time.Second*5, func() {
			t.Logf("Stall detected!")
			pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
		})

		testQueueRunsInOrderOnce(t)

		sd.Stop()

		res := false
		//res := timerReset()
		if res {
			failures++
		}

		duration := time.Now().Sub(start)
		if duration < min || min == 0 {
			min = duration
		}
		if duration > max {
			max = duration
		}
		if duration > time.Millisecond*5000 {
			slow++
		}

		if i%1000 == 0 && i > 0 {
			t.Logf("count: %07d | min/max: %s/%s\t| slow: %d\t| fail: %d\n", i, min, max, slow, failures)
		}
	}
}

func timerReset() bool {
	t := time.NewTimer(time.Millisecond)

	ran := []time.Time{}
	var expired bool

	ctx, _ := context.WithTimeout(context.TODO(), time.Second*5)

	for len(ran) < 10 {
		select {
		case now := <-t.C:
			ran = append(ran, now)
			if false && !t.Stop() {
				fmt.Printf("BLOCK\n")
				<-t.C
				fmt.Printf("UNB\n")
			}
			t.Reset(time.Nanosecond * -2)
		case <-ctx.Done():
			expired = true
		}
	}

	return expired
}

func testQueueRunsInOrderOnce(t *testing.T) {
	ctx, ctxCancel := context.WithTimeout(context.Background(), time.Second*20)
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
