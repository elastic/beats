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
	"context"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v8/heartbeat/config"
	"github.com/elastic/beats/v8/libbeat/monitoring"
)

// The runAt in the island of tarawa 🏝. Good test TZ because it's pretty rare for a local box
// to be state in this TZ, and it has a weird offset +0125+17300.
func tarawaTime() *time.Location {
	loc, err := time.LoadLocation("Pacific/Tarawa")
	if err != nil {
		panic("this computer doesn't know about tarawa runAt " + err.Error())
	}

	return loc
}

func TestNewWithLocation(t *testing.T) {
	scheduler := Create(123, monitoring.NewRegistry(), tarawaTime(), nil, false)
	assert.Equal(t, int64(123), scheduler.limit)
	assert.Equal(t, tarawaTime(), scheduler.location)
}

// Runs tasks as fast as possible. Good for keeping tests snappy.
type testSchedule struct {
	delay time.Duration
}

func (testSchedule) RunOnInit() bool {
	return true
}

func (t testSchedule) Next(now time.Time) time.Time {
	return now.Add(t.delay)
}

// Test task that will only actually invoke the fn the given number of times
// this lets us test around timing / scheduling weirdness more accurately, since
// we can in tests expect an exact number of invocations
func testTaskTimes(limit uint32, fn TaskFunc) TaskFunc {
	invoked := new(uint32)
	return func(ctx context.Context) (conts []TaskFunc) {
		if atomic.LoadUint32(invoked) < limit {
			conts = fn(ctx)
		}
		atomic.AddUint32(invoked, 1)
		return conts
	}
}

func TestSchedulerRun(t *testing.T) {
	// We use tarawa runAt because it could expose some weird runAt math if by accident some code
	// relied on the local TZ.
	s := Create(10, monitoring.NewRegistry(), tarawaTime(), nil, false)
	defer s.Stop()

	executed := make(chan string)

	initialEvents := uint32(10)
	s.Add(testSchedule{0}, "add", testTaskTimes(initialEvents, func(_ context.Context) []TaskFunc {
		executed <- "initial"
		cont := func(_ context.Context) []TaskFunc {
			executed <- "initialCont"
			return nil
		}
		return []TaskFunc{cont}
	}), "http", nil)

	removedEvents := uint32(1)
	// This function will be removed after being invoked once
	removeMtx := sync.Mutex{}
	var remove context.CancelFunc
	var testFn TaskFunc = func(_ context.Context) []TaskFunc {
		executed <- "removed"
		removeMtx.Lock()
		remove()
		removeMtx.Unlock()
		return nil
	}
	// Attempt to execute this twice to see if remove() had any effect
	removeMtx.Lock()
	remove, err := s.Add(testSchedule{}, "removed", testTaskTimes(removedEvents+1, testFn), "http", nil)
	require.NoError(t, err)
	require.NotNil(t, remove)
	removeMtx.Unlock()

	postRemoveEvents := uint32(10)
	s.Add(testSchedule{}, "postRemove", testTaskTimes(postRemoveEvents, func(_ context.Context) []TaskFunc {
		executed <- "postRemove"
		cont := func(_ context.Context) []TaskFunc {
			executed <- "postRemoveCont"
			return nil
		}
		return []TaskFunc{cont}
	}), "http", nil)

	received := make([]string, 0)
	// We test for a good number of events in this loop because we want to ensure that the remove() took effect
	// Otherwise, we might only do 1 preAdd and 1 postRemove event
	// We double the number of pre/post add events to account for their continuations
	totalExpected := initialEvents*2 + removedEvents + postRemoveEvents*2
	for uint32(len(received)) < totalExpected {
		select {
		case got := <-executed:
			received = append(received, got)
		case <-time.After(5 * time.Second):
			require.Fail(t, fmt.Sprintf("Timed out waitingTasks for schedule job to execute, got %d of %d: %v",
				len(received), totalExpected, received))
		}
	}

	// The removed callback should only have been executed once
	counts := map[string]uint32{"initial": 0, "initialCont": 0, "removed": 0, "postRemove": 0, "postRemoveCont": 0}
	for _, s := range received {
		counts[s]++
	}

	// convert with int() because the printed output is nicer than hex
	assert.Equal(t, int(initialEvents), int(counts["initial"]))
	assert.Equal(t, int(initialEvents), int(counts["initialCont"]))
	assert.Equal(t, int(removedEvents), int(counts["removed"]))
	assert.Equal(t, int(postRemoveEvents), int(counts["postRemove"]))
	assert.Equal(t, int(postRemoveEvents), int(counts["postRemoveCont"]))
}

func TestScheduler_WaitForRunOnce(t *testing.T) {
	s := Create(10, monitoring.NewRegistry(), tarawaTime(), nil, true)

	defer s.Stop()

	executed := new(uint32)
	waits := new(uint32)

	s.Add(testSchedule{0}, "runOnce", func(_ context.Context) []TaskFunc {
		cont := func(_ context.Context) []TaskFunc {
			// Make sure we actually wait for the task!
			time.Sleep(time.Millisecond * 250)
			atomic.AddUint32(executed, 1)
			return nil
		}
		return []TaskFunc{cont}
	}, "http", func() { atomic.AddUint32(waits, 1) })

	s.WaitForRunOnce()
	require.Equal(t, uint32(1), atomic.LoadUint32(executed))
	require.Equal(t, uint32(1), atomic.LoadUint32(waits))
}

func TestScheduler_Stop(t *testing.T) {
	s := Create(10, monitoring.NewRegistry(), tarawaTime(), nil, false)

	executed := make(chan struct{})

	s.Stop()

	_, err := s.Add(testSchedule{}, "testPostStop", testTaskTimes(1, func(_ context.Context) []TaskFunc {
		executed <- struct{}{}
		return nil
	}), "http", nil)

	assert.Equal(t, ErrAlreadyStopped, err)
}

func makeTasks(num int, callback func()) TaskFunc {
	return func(ctx context.Context) []TaskFunc {
		callback()
		if num < 1 {
			return nil
		}
		return []TaskFunc{makeTasks(num-1, callback)}
	}
}

func TestSchedTaskLimits(t *testing.T) {
	tests := []struct {
		name    string
		numJobs int
		limit   int64
		expect  func(events []int)
	}{
		{
			name:    "runs more than 1 with limit of 1",
			numJobs: 2,
			limit:   1,
			expect: func(events []int) {
				mid := len(events) / 2
				firstHalf := events[0:mid]
				lastHalf := events[mid:]
				for _, ele := range firstHalf {
					assert.Equal(t, firstHalf[0], ele)
				}
				for _, ele := range lastHalf {
					assert.Equal(t, lastHalf[0], ele)
				}
			},
		},
		{
			name:    "runs 50 interleaved without limit",
			numJobs: 50,
			limit:   math.MaxInt64,
			expect: func(events []int) {
				require.GreaterOrEqual(t, len(events), 50)
			},
		},
		{
			name:    "runs 100 with limit not configured",
			numJobs: 100,
			limit:   0,
			expect: func(events []int) {
				require.GreaterOrEqual(t, len(events), 100)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var jobConfigByType = map[string]config.JobLimit{}
			jobType := "http"
			if tt.limit > 0 {
				jobConfigByType = map[string]config.JobLimit{
					jobType: {Limit: tt.limit},
				}
			}
			s := Create(math.MaxInt64, monitoring.NewRegistry(), tarawaTime(), jobConfigByType, false)
			var taskArr []int
			wg := sync.WaitGroup{}
			wg.Add(tt.numJobs)
			for i := 0; i < tt.numJobs; i++ {
				num := i
				tf := makeTasks(4, func() {
					taskArr = append(taskArr, num)
				})
				go func(tff TaskFunc) {
					sj := newSchedJob(context.Background(), s, "myid", jobType, tff)
					sj.run()
					wg.Done()
				}(tf)
			}
			wg.Wait()
			tt.expect(taskArr)
		})
	}
}

func BenchmarkScheduler(b *testing.B) {
	s := Create(0, monitoring.NewRegistry(), tarawaTime(), nil, false)

	sched := testSchedule{0}

	executed := make(chan struct{})
	for i := 0; i < 1024; i++ {
		_, err := s.Add(sched, "testPostStop", func(_ context.Context) []TaskFunc {
			executed <- struct{}{}
			return nil
		}, "http", nil)
		assert.NoError(b, err)
	}

	defer s.Stop()

	count := 0
	for count < b.N {
		<-executed
		count++
	}
}
