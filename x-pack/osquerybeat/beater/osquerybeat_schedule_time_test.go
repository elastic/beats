// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNativePlannedScheduleTime(t *testing.T) {
	tests := []struct {
		name      string
		startDate string
		interval  int
		runTime   time.Time
		expected  time.Time
	}{
		{
			name:      "returns schedule slot for in-between runtime",
			startDate: "2024-01-01T00:00:00Z",
			interval:  3600,
			runTime:   time.Date(2024, 1, 1, 2, 35, 0, 0, time.UTC),
			expected:  time.Date(2024, 1, 1, 2, 0, 0, 0, time.UTC),
		},
		{
			name:      "returns exact slot when runtime on boundary",
			startDate: "2024-01-01T00:00:00Z",
			interval:  3600,
			runTime:   time.Date(2024, 1, 1, 3, 0, 0, 0, time.UTC),
			expected:  time.Date(2024, 1, 1, 3, 0, 0, 0, time.UTC),
		},
		{
			name:      "returns runtime when before start",
			startDate: "2024-01-01T05:00:00Z",
			interval:  3600,
			runTime:   time.Date(2024, 1, 1, 4, 30, 0, 0, time.UTC),
			expected:  time.Date(2024, 1, 1, 4, 30, 0, 0, time.UTC),
		},
		{
			name:      "returns runtime when start date is empty",
			startDate: "",
			interval:  3600,
			runTime:   time.Date(2024, 1, 1, 1, 23, 0, 0, time.UTC),
			expected:  time.Date(2024, 1, 1, 1, 23, 0, 0, time.UTC),
		},
		{
			name:      "returns runtime when start date is invalid",
			startDate: "not-a-date",
			interval:  3600,
			runTime:   time.Date(2024, 1, 1, 1, 23, 0, 0, time.UTC),
			expected:  time.Date(2024, 1, 1, 1, 23, 0, 0, time.UTC),
		},
		{
			name:      "returns runtime when interval is invalid",
			startDate: "2024-01-01T00:00:00Z",
			interval:  0,
			runTime:   time.Date(2024, 1, 1, 1, 23, 0, 0, time.UTC),
			expected:  time.Date(2024, 1, 1, 1, 23, 0, 0, time.UTC),
		},
		{
			name:      "handles non-utc start date offsets",
			startDate: "2024-01-01T09:00:00+02:00",
			interval:  3600,
			runTime:   time.Date(2024, 1, 1, 8, 10, 0, 0, time.UTC),
			expected:  time.Date(2024, 1, 1, 8, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nativePlannedScheduleTime(tt.startDate, tt.interval, tt.runTime.Unix())
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestQueryResultMeta_PlannedScheduleTime(t *testing.T) {
	planned := time.Date(2024, 2, 3, 4, 5, 6, 700, time.UTC)
	res := QueryResult{
		CalendarTime: "Mon Jan  1 00:00:00 2024 UTC",
		UnixTime:     1704067200,
		Epoch:        10,
		Counter:      42,
	}

	meta := queryResultMeta("snapshot", "", res, 7, planned)

	assert.Equal(t, planned.Format(time.RFC3339Nano), meta["planned_schedule_time"])
	assert.Equal(t, int64(7), meta["schedule_execution_count"])
}

