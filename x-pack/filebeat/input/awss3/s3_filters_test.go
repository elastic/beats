// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"testing"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/stretchr/testify/assert"
)

func Test_filterProvider(t *testing.T) {
	t.Run("Configuration check", func(t *testing.T) {
		cfg := config{
			StartTimestamp: "2024-11-26T21:00:00Z",
			IgnoreOlder:    10 * time.Minute,
		}

		fProvider := newFilterProvider(&cfg)

		assert.Equal(t, 1, len(fProvider.staticFilters))
		assert.Equal(t, filterStartTime, fProvider.staticFilters[0].getID())
	})

	logger := logp.NewLogger("test-logger")

	tests := []struct {
		name                string
		cfg                 *config
		inputState          state
		runFilterForCount   int
		expectFilterResults []bool
	}{
		{
			name: "Simple run - all valid result",
			cfg: &config{
				StartTimestamp: "2024-11-26T21:00:00Z",
				IgnoreOlder:    10 * time.Minute,
			},
			inputState:          newState("bucket", "key", "eTag", time.Now()),
			runFilterForCount:   1,
			expectFilterResults: []bool{true},
		},
		{
			name: "Simple run - all invalid result",
			cfg: &config{
				StartTimestamp: "2024-11-26T21:00:00Z",
			},
			inputState:          newState("bucket", "key", "eTag", time.Unix(0, 0)),
			runFilterForCount:   1,
			expectFilterResults: []bool{false},
		},
		{
			name:                "Simple run - no filters hence valid result",
			cfg:                 &config{},
			inputState:          newState("bucket", "key", "eTag", time.Now()),
			runFilterForCount:   1,
			expectFilterResults: []bool{true},
		},
		{
			name: "Single filter - ignore old invalid result",
			cfg: &config{
				IgnoreOlder: 1 * time.Minute,
			},
			inputState:          newState("bucket", "key", "eTag", time.Unix(time.Now().Add(-2*time.Minute).Unix(), 0)),
			runFilterForCount:   1,
			expectFilterResults: []bool{false},
		},
		{
			name: "Combined filters - ignore old won't affect first run if timestamp is given but will affect thereafter",
			cfg: &config{
				StartTimestamp: "2024-11-26T21:00:00Z",
				IgnoreOlder:    10 * time.Minute,
			},
			inputState:          newState("bucket", "key", "eTag", time.Unix(1732654860, 0)), // 2024-11-26T21:01:00Z in epoch
			runFilterForCount:   3,
			expectFilterResults: []bool{true, false, false},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			provider := newFilterProvider(test.cfg)
			results := make([]bool, 0, test.runFilterForCount)

			for i := 0; i < test.runFilterForCount; i++ {
				applierFunc := provider.getApplierFunc()
				results = append(results, applierFunc(logger, test.inputState))
			}

			assert.Equal(t, test.expectFilterResults, results)
		})
	}
}

func Test_startTimestampFilter(t *testing.T) {
	t.Run("Configuration check", func(t *testing.T) {
		entry := newState("bucket", "key", "eTag", time.Now())

		oldTimeFilter := newStartTimestampFilter(time.Now().Add(-2 * time.Minute))

		assert.Equal(t, filterStartTime, oldTimeFilter.getID())
		assert.True(t, oldTimeFilter.isValid(entry))
	})

	tests := []struct {
		name      string
		timeStamp time.Time
		input     state
		result    bool
	}{
		{
			name:      "State valid",
			timeStamp: time.Now().Add(-10 * time.Minute),
			input:     newState("bucket", "key", "eTag", time.Now()),
			result:    true,
		},

		{
			name:      "State invalid",
			timeStamp: time.Now(),
			input:     newState("bucket", "key", "eTag", time.Now().Add(-10*time.Minute)),
			result:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			timeFilter := newStartTimestampFilter(test.timeStamp)
			assert.Equal(t, test.result, timeFilter.isValid(test.input))
		})
	}

}

func Test_oldestTimeFilter(t *testing.T) {
	t.Run("configuration check", func(t *testing.T) {
		duration := time.Duration(1) * time.Second
		entry := newState("bucket", "key", "eTag", time.Now())

		oldTimeFilter := newOldestTimeFilter(duration, time.Now())

		assert.Equal(t, filterOldestTime, oldTimeFilter.getID())
		assert.True(t, oldTimeFilter.isValid(entry))
	})

	tests := []struct {
		name     string
		duration time.Duration
		input    state
		result   bool
	}{
		{
			name:     "State valid",
			duration: time.Duration(1) * time.Minute,
			input:    newState("bucket", "key", "eTag", time.Now()),
			result:   true,
		},

		{
			name:     "State invalid",
			duration: time.Duration(1) * time.Minute,
			input:    newState("bucket", "key", "eTag", time.Now().Add(-10*time.Minute)),
			result:   false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			timeFilter := newOldestTimeFilter(test.duration, time.Now())
			assert.Equal(t, test.result, timeFilter.isValid(test.input))
		})
	}

}
