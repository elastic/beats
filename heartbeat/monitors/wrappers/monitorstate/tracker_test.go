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

package monitorstate

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
)

func TestTrackerRecord(t *testing.T) {
	mst := NewTracker(NilStateLoader, true)
	ms := mst.RecordStatus(TestSf, StatusUp, true)
	require.Equal(t, StatusUp, ms.Status)
	requireMSStatusCount(t, ms, StatusUp, 1)

	for i := 0; i < FlappingThreshold; i++ {
		_ = mst.RecordStatus(TestSf, StatusDown, true)
		ms = mst.RecordStatus(TestSf, StatusUp, true)
	}
	require.Equal(t, StatusFlapping, ms.Status)
	requireMSCounts(t, ms, FlappingThreshold+1, FlappingThreshold)

	// Restore stable state
	for i := 0; i < FlappingThreshold; i++ {
		_ = mst.RecordStatus(TestSf, StatusDown, true)
	}

	ms = mst.RecordStatus(TestSf, StatusDown, true)
	require.Equal(t, StatusDown, ms.Status)
	requireMSStatusCount(t, ms, StatusDown, FlappingThreshold-1)
}

func TestTrackerRecordFlappingDisabled(t *testing.T) {
	mst := NewTracker(NilStateLoader, false)
	ms := mst.RecordStatus(TestSf, StatusUp, true)
	require.Equal(t, StatusUp, ms.Status)
	requireMSStatusCount(t, ms, StatusUp, 1)

	for i := 0; i < FlappingThreshold; i++ {
		_ = mst.RecordStatus(TestSf, StatusDown, true)
		ms = mst.RecordStatus(TestSf, StatusUp, true)
	}

	// with flapping disabled it only shows as up
	require.Equal(t, StatusUp, ms.Status)
	requireMSCounts(t, ms, 1, 0)

	ms = mst.RecordStatus(TestSf, StatusDown, true)
	require.Equal(t, StatusDown, ms.Status)
	requireMSStatusCount(t, ms, StatusDown, 1)
}

func TestAtomicStateLoader(t *testing.T) {
	stateA := &State{ID: "A"}
	stateB := &State{ID: "B"}
	loaderA := func(stdfields.StdMonitorFields) (*State, error) {
		return stateA, nil
	}
	loaderB := func(stdfields.StdMonitorFields) (*State, error) {
		return stateB, nil
	}

	asl, replace := AtomicStateLoader(loaderA)
	resState, _ := asl(stdfields.StdMonitorFields{})
	require.Equal(t, stateA, resState)

	replace(loaderB)
	resState, _ = asl(stdfields.StdMonitorFields{})
	require.Equal(t, stateB, resState)

	replace(loaderA)
	resState, _ = asl(stdfields.StdMonitorFields{})
	require.Equal(t, stateA, resState)

}

func TestDeferredStateLoaderTimeout(t *testing.T) {
	stateA := &State{ID: "A"}
	loaderA := func(stdfields.StdMonitorFields) (*State, error) {
		return stateA, nil
	}

	dsl, _ := DeferredStateLoader(loaderA, 100*time.Millisecond)
	resState, _ := dsl(stdfields.StdMonitorFields{})
	require.Equal(t, stateA, resState)
}

func TestDeferredStateLoader(t *testing.T) {
	stateA := &State{ID: "A"}
	stateB := &State{ID: "B"}
	loaderA := func(stdfields.StdMonitorFields) (*State, error) {
		return stateA, nil
	}
	loaderB := func(stdfields.StdMonitorFields) (*State, error) {
		return stateB, nil
	}

	// Test deferred initialization, launch query while stateA and expect
	// updated stateB
	dsl, replace := DeferredStateLoader(loaderA, 10*time.Second)

	go func() {
		time.Sleep(1 * time.Second)

		replace(loaderB)
	}()

	resState, _ := dsl(stdfields.StdMonitorFields{})
	require.Equal(t, stateB, resState)

	replace(loaderA)
	resState, _ = dsl(stdfields.StdMonitorFields{})
	require.Equal(t, stateA, resState)
}

func TestStateLoaderRetry(t *testing.T) {
	// While testing the sleep time between retries should be negligible
	waitFn := func() time.Duration {
		return time.Microsecond
	}

	tests := []struct {
		name          string
		retryable     bool
		rc            RetryConfig
		expectedCalls int
	}{
		{
			"should retry 3 times when fails with retryable error",
			true,
			RetryConfig{waitFn: waitFn},
			3,
		},
		{
			"should not retry when fails with non-retryable error",
			false,
			RetryConfig{waitFn: waitFn},
			1,
		},
		{
			"should honour the configured number of attempts when fails with retryable error",
			true,
			RetryConfig{attempts: 5, waitFn: waitFn},
			5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			errorStateLoader := func(_ stdfields.StdMonitorFields) (*State, error) {
				calls += 1
				return nil, LoaderError{err: errors.New("test error"), Retry: tt.retryable}
			}

			mst := NewTracker(errorStateLoader, true)
			mst.GetCurrentState(stdfields.StdMonitorFields{}, tt.rc)

			require.Equal(t, calls, tt.expectedCalls)
		})
	}
}
