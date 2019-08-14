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
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The time in the island of tarawa 🏝. Good test TZ because it's pretty rare for a local box
// to be state in this TZ, and it has a weird offset +0125+17300.
func tarawaTime() *time.Location {
	loc, err := time.LoadLocation("Pacific/Tarawa")
	if err != nil {
		panic("this computer doesn't know about tarawa time " + err.Error())
	}

	return loc
}

func TestNew(t *testing.T) {
	scheduler := New(123)
	assert.Equal(t, uint(123), scheduler.limit)
	assert.Equal(t, time.Local, scheduler.location)
}

func TestNewWithLocation(t *testing.T) {
	scheduler := NewWithLocation(123, tarawaTime())
	assert.Equal(t, uint(123), scheduler.limit)
	assert.Equal(t, tarawaTime(), scheduler.location)
}

// Runs tasks as fast as possible. Good for keeping tests snappy.
type instantSchedule struct{}

func (instantSchedule) Next(now time.Time) time.Time {
	return now
}

// Test task that will only actually invoke the fn the given number of times
// this lets us test around timing / scheduling weirdness more accurately, since
// we can in tests expect an exact number of invocations
func testTaskTimes(limit uint32, fn func()) func() []TaskFunc {
	invoked := new(uint32)
	return func() []TaskFunc {
		if atomic.LoadUint32(invoked) < limit {
			fn()
		}
		atomic.AddUint32(invoked, 1)
		return nil
	}
}

func TestScheduler_Start(t *testing.T) {
	// We use tarawa time because it could expose some weird time math if by accident some code
	// relied on the local TZ.
	s := NewWithLocation(10, tarawaTime())
	defer s.Stop()

	executed := make(chan string)

	preAddEvents := uint32(10)
	s.Add(instantSchedule{}, "preAdd", testTaskTimes(preAddEvents, func() {
		executed <- "preAdd"
	}))

	removedEvents := uint32(1)
	// This function will be removed after being invoked once
	var remove func() error
	// Attempt to execute this twice to see if remove() had any effect
	remove, err := s.Add(instantSchedule{}, "removed", testTaskTimes(removedEvents+1, func() {
		executed <- "removed"
		remove()
	}))
	require.NoError(t, err)

	s.Start()

	postAddEvents := uint32(10)
	s.Add(instantSchedule{}, "postAdd", testTaskTimes(postAddEvents, func() {
		executed <- "postAdd"
	}))

	received := make([]string, 0)
	// We test for a good number of events in this loop because we want to ensure that the remove() took effect
	// Otherwise, we might only do 1 preAdd and 1 postAdd event
	for uint32(len(received)) < preAddEvents+removedEvents+postAddEvents {
		select {
		case got := <-executed:
			received = append(received, got)
		case <-time.After(10 * time.Second):
			require.Fail(t, "Timed out waiting for schedule job to execute")
		}
	}

	// The removed callback should only have been executed once
	counts := map[string]uint32{"preAdd": 0, "postAdd": 0, "removed": 0}
	for _, s := range received {
		counts[s]++
	}
	assert.Equal(t, preAddEvents, counts["preAdd"])
	assert.Equal(t, postAddEvents, counts["postAdd"])
	assert.Equal(t, removedEvents, counts["removed"])
}

func TestScheduler_Stop(t *testing.T) {
	s := NewWithLocation(10, tarawaTime())

	executed := make(chan struct{})

	s.Start()
	s.Stop()

	_, err := s.Add(instantSchedule{}, "testPostStop", testTaskTimes(1, func() {
		executed <- struct{}{}
	}))

	assert.Equal(t, ErrAlreadyStopped, err)
}
