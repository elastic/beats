// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGcFind(t *testing.T) {
	now := time.Date(2021, time.July, 22, 18, 38, 00, 0, time.UTC)
	started := now.Add(-10 * time.Second)

	testCases := []struct {
		key      string
		stateFn  func() State
		expected bool
	}{
		{
			"stateBeforeStartedStored",
			func() State {
				state := NewState("bucket", "key", 100, started.Add(-15*time.Second))
				state.MarkAsStored()
				return state
			},
			false,
		},
		{
			"stateBeforeStartedNotStored",
			func() State {
				state := NewState("bucket", "key", 100, started.Add(-15*time.Second))
				return state
			},
			true,
		},
		{
			"stateAfterStartedStored",
			func() State {
				state := NewState("bucket", "key", 100, started.Add(5*time.Second))
				state.MarkAsStored()
				return state
			},
			false,
		},
		{
			"stateAfterStartedNotStored",
			func() State {
				state := NewState("bucket", "key", 100, started.Add(5*time.Second))
				return state
			},
			true,
		},
		{
			"stateAfterTTLStored",
			func() State {
				state := NewState("bucket", "key", 100, started.Add(25*60*time.Second))
				state.MarkAsStored()
				return state

			},
			false,
		},
		{
			"stateAfterTTLNotStored",
			func() State {
				state := NewState("bucket", "key", 100, started.Add(25*60*time.Second))
				return state
			},
			false,
		},
		{
			"stateBeforeTTLStored",
			func() State {
				state := NewState("bucket", "key", 100, started.Add(-25*60*time.Second))
				state.MarkAsStored()
				return state
			},
			true,
		},
		{
			"stateBeforeTTLNotStored",
			func() State {
				state := NewState("bucket", "key", 100, started.Add(-25*60*time.Second))
				return state
			},
			true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.key, func(t *testing.T) {
			states := NewStates()
			state := testCase.stateFn()
			states.Update(state)
			keys := gcFind(states, started, now)
			expected := map[string]struct{}{}
			if testCase.expected {
				expected[state.Id] = struct{}{}
			}

			assert.Equal(t, expected, keys)
		})
	}
}
