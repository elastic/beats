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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/monitoring"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestNew(t *testing.T) {
	scheduler := New(123, monitoring.NewRegistry())
	assert.Equal(t, int64(123), scheduler.limit)
	assert.Equal(t, time.Local, scheduler.location)
}

func TestNewWithLocation(t *testing.T) {
	scheduler := NewWithLocation(123, monitoring.NewRegistry(), tarawaTime())
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

func TestScheduler_Start(t *testing.T) {
	// We use tarawa runAt because it could expose some weird runAt math if by accident some code
	// relied on the local TZ.
	s := NewWithLocation(10, monitoring.NewRegistry(), tarawaTime())
	defer s.Stop()

	executed := make(chan string)

	preAddEvents := uint32(10)
	s.Add(testSchedule{0}, "preAdd", testTaskTimes(preAddEvents, func(_ context.Context) []TaskFunc {
		executed <- "preAdd"
		cont := func(_ context.Context) []TaskFunc {
			executed <- "preAddCont"
			return nil
		}
		return []TaskFunc{cont}
	}))

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
	remove, err := s.Add(testSchedule{}, "removed", testTaskTimes(removedEvents+1, testFn))
	require.NoError(t, err)
	require.NotNil(t, remove)
	removeMtx.Unlock()

	s.Start()

	postAddEvents := uint32(10)
	s.Add(testSchedule{}, "postAdd", testTaskTimes(postAddEvents, func(_ context.Context) []TaskFunc {
		executed <- "postAdd"
		cont := func(_ context.Context) []TaskFunc {
			executed <- "postAddCont"
			return nil
		}
		return []TaskFunc{cont}
	}))

	received := make([]string, 0)
	// We test for a good number of events in this loop because we want to ensure that the remove() took effect
	// Otherwise, we might only do 1 preAdd and 1 postAdd event
	// We double the number of pre/post add events to account for their continuations
	totalExpected := preAddEvents*2 + removedEvents + postAddEvents*2
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
	counts := map[string]uint32{"preAdd": 0, "postAdd": 0, "preAddCont": 0, "postAddcont": 0, "removed": 0}
	for _, s := range received {
		counts[s]++
	}

	// convert with int() because the printed output is nicer than hex
	assert.Equal(t, int(preAddEvents), int(counts["preAdd"]))
	assert.Equal(t, int(preAddEvents), int(counts["preAddCont"]))
	assert.Equal(t, int(postAddEvents), int(counts["postAdd"]))
	assert.Equal(t, int(postAddEvents), int(counts["postAddCont"]))
	assert.Equal(t, int(removedEvents), int(counts["removed"]))
}

func TestScheduler_Stop(t *testing.T) {
	s := NewWithLocation(10, monitoring.NewRegistry(), tarawaTime())

	executed := make(chan struct{})

	require.NoError(t, s.Start())
	require.NoError(t, s.Stop())

	_, err := s.Add(testSchedule{}, "testPostStop", testTaskTimes(1, func(_ context.Context) []TaskFunc {
		executed <- struct{}{}
		return nil
	}))

	assert.Equal(t, ErrAlreadyStopped, err)
}

func BenchmarkScheduler(b *testing.B) {
	s := NewWithLocation(0, monitoring.NewRegistry(), tarawaTime())

	sched := testSchedule{0}

	executed := make(chan struct{})
	for i := 0; i < 1024; i++ {
		_, err := s.Add(sched, "testPostStop", func(_ context.Context) []TaskFunc {
			executed <- struct{}{}
			return nil
		})
		assert.NoError(b, err)
	}

	err := s.Start()
	defer s.Stop()
	assert.NoError(b, err)

	count := 0
	for count < b.N {
		select {
		case <-executed:
			count++
		}
	}
}
