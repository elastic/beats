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
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/heartbeat/config"
	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
)

func TestRecordingAndFlapping(t *testing.T) {
	ms := newMonitorState(TestSf, StatusUp, 0, true)
	recordFlappingSeries(TestSf, ms)
	require.Equal(t, StatusFlapping, ms.Status)
	require.Equal(t, FlappingThreshold+1, ms.Checks)
	require.Equal(t, ms.Up+ms.Down, ms.Checks)

	recordStableSeries(TestSf, ms, FlappingThreshold*2, StatusDown)
	require.Equal(t, StatusDown, ms.Status)
	// The count should be FlappingThreshold+1 since we used double the threshold before
	// This is because we have one full threshold of stable checks, as well as the final check that
	// flipped us out of the threshold, which goes toward the new state.
	requireMSCounts(t, ms, 0, FlappingThreshold+1)
	require.Nil(t, ms.Ends, "expected nil ends after a stable series")

	// Since we're now in a stable state a single up check should create a new state from a stable one
	ms.recordCheck(TestSf, StatusUp)
	require.Equal(t, StatusUp, ms.Status)
	requireMSCounts(t, ms, 1, 0)
}

func TestDuration(t *testing.T) {
	ms := newMonitorState(TestSf, StatusUp, 0, true)
	ms.recordCheck(TestSf, StatusUp)
	time.Sleep(time.Millisecond * 10)
	ms.recordCheck(TestSf, StatusUp)
	// Pretty forgiving upper bound to account for flaky CI
	require.True(t, ms.DurationMs > 9 && ms.DurationMs < 900, "Expected duration to be ~10ms, got %d", ms.DurationMs)
}

// recordFlappingSeries is a helper that should always put the monitor into a flapping state.
func recordFlappingSeries(TestSf stdfields.StdMonitorFields, ms *State) {
	for i := 0; i < FlappingThreshold; i++ {
		if i%2 == 0 {
			ms.recordCheck(TestSf, StatusUp)
		} else {
			ms.recordCheck(TestSf, StatusDown)
		}
	}
}

// recordStableSeries is a test helper for repeatedly recording one status
func recordStableSeries(TestSf stdfields.StdMonitorFields, ms *State, count int, s StateStatus) {
	for i := 0; i < count; i++ {
		ms.recordCheck(TestSf, s)
	}
}

func TestTransitionTo(t *testing.T) {
	s := newMonitorState(TestSf, StatusUp, 0, true)
	first := *s
	s.transitionTo(TestSf, StatusDown)
	second := *s
	s.transitionTo(TestSf, StatusUp)

	require.NotEqual(t, s.ID, second.ID)
	require.NotEqual(t, s.ID, first)

	// Ensure ends is set
	require.Equal(t, second.ID, s.Ends.ID)
	require.Equal(t, second.DurationMs, s.Ends.DurationMs)
	require.Equal(t, second.StartedAt, s.Ends.StartedAt)
	require.Equal(t, second.Checks, s.Ends.Checks)
	require.Equal(t, second.Up, s.Ends.Up)
	require.Equal(t, second.Down, s.Ends.Down)
	// Ensure No infinite storage of states
	require.Nil(t, s.Ends.Ends)
}

func TestLoaderDBKey(t *testing.T) {
	tests := []struct {
		name      string
		runFromID string
		at        time.Time
		ctr       int
		expected  string
	}{
		{
			"simple - no rfid",
			"",
			time.Unix(0, 0),
			0,
			"default-0-0",
		},
		{
			"simple - other time / count",
			"",
			time.Unix(12345, 0),
			98765,
			fmt.Sprintf("default-%x-%x", 12345000, 98765),
		},
		{
			"Service location, weird chars",
			"Asia/Pacific - Japan",
			time.Unix(0, 0),
			0,
			"Asia_Pacific_-_Japan-0-0",
		},
	}

	for _, tt := range tests {
		sf := stdfields.StdMonitorFields{}
		if tt.runFromID != "" {
			sf.RunFrom = &config.LocationWithID{
				ID: tt.runFromID,
			}
		}

		key := LoaderDBKey(sf, tt.at, tt.ctr)
		require.Equal(t, tt.expected, key)
	}
}
