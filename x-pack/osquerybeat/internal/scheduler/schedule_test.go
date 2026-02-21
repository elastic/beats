// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package scheduler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecurrenceSchedule_Parse(t *testing.T) {
	tests := []struct {
		name    string
		rrule   string
		wantErr bool
		errType error
	}{
		{
			name:    "valid daily",
			rrule:   "FREQ=DAILY",
			wantErr: false,
		},
		{
			name:    "valid weekly on Monday and Wednesday",
			rrule:   "FREQ=WEEKLY;BYDAY=MO,WE",
			wantErr: false,
		},
		{
			name:    "valid every 2 weeks on Monday and Wednesday",
			rrule:   "FREQ=WEEKLY;INTERVAL=2;BYDAY=MO,WE",
			wantErr: false,
		},
		{
			name:    "valid monthly on first day",
			rrule:   "FREQ=MONTHLY;BYMONTHDAY=1",
			wantErr: false,
		},
		{
			name:    "valid yearly",
			rrule:   "FREQ=YEARLY;BYMONTH=1;BYMONTHDAY=1",
			wantErr: false,
		},
		{
			name:    "valid weekdays only (Mon-Fri)",
			rrule:   "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR",
			wantErr: false,
		},
		{
			name:    "empty rrule",
			rrule:   "",
			wantErr: true,
			errType: ErrInvalidRRule,
		},
		{
			name:    "invalid rrule syntax",
			rrule:   "not a valid rrule",
			wantErr: true,
			errType: ErrInvalidRRule,
		},
		{
			name:    "interval too short - hourly",
			rrule:   "FREQ=HOURLY",
			wantErr: true,
			errType: ErrIntervalTooShort,
		},
		{
			name:    "interval too short - every 5 hours",
			rrule:   "FREQ=HOURLY;INTERVAL=5",
			wantErr: true,
			errType: ErrIntervalTooShort,
		},
		{
			name:    "interval too short - minutely",
			rrule:   "FREQ=MINUTELY",
			wantErr: true,
			errType: ErrIntervalTooShort,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
			s := &RecurrenceSchedule{
				RRule:     tt.rrule,
				StartDate: &startDate,
			}
			err := s.Parse()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
			} else {
				assert.NoError(t, err)
				assert.True(t, s.IsActive())
			}
		})
	}
}

func TestRecurrenceSchedule_Next(t *testing.T) {
	// Every day
	s := &RecurrenceSchedule{RRule: "FREQ=DAILY"}
	startDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	s.StartDate = &startDate
	require.NoError(t, s.Parse())

	// Reference time: Jan 15, 10:30
	ref := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	next := s.Next(ref)

	// Should be Jan 16 at midnight (same time as DTSTART)
	expected := time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, expected, next)
}

func TestRecurrenceSchedule_NextWithStartDate(t *testing.T) {
	s := &RecurrenceSchedule{RRule: "FREQ=DAILY"}
	startDate := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	s.StartDate = &startDate
	require.NoError(t, s.Parse())

	// Reference time before start date
	ref := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)
	next := s.Next(ref)

	// Should be at or after start date
	assert.True(t, next.Equal(startDate) || next.After(startDate))
}

func TestRecurrenceSchedule_NextWithEndDate(t *testing.T) {
	s := &RecurrenceSchedule{RRule: "FREQ=DAILY"}
	startDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 20, 12, 0, 0, 0, time.UTC)
	s.StartDate = &startDate
	s.EndDate = &endDate
	require.NoError(t, s.Parse())

	// Reference time before end date - should get next run
	ref := time.Date(2024, 1, 18, 10, 0, 0, 0, time.UTC)
	next := s.Next(ref)
	assert.False(t, next.IsZero())
	assert.True(t, next.Before(endDate) || next.Equal(endDate))

	// Reference time close to end date - should get zero time for next after end
	ref = time.Date(2024, 1, 20, 13, 0, 0, 0, time.UTC)
	next = s.Next(ref)
	assert.True(t, next.IsZero())
}

func TestRecurrenceSchedule_NextWithWindow(t *testing.T) {
	s := &RecurrenceSchedule{RRule: "FREQ=DAILY"}
	startDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 25, 14, 0, 0, 0, time.UTC)
	s.StartDate = &startDate
	s.EndDate = &endDate
	require.NoError(t, s.Parse())

	// Reference time within window
	ref := time.Date(2024, 1, 17, 11, 30, 0, 0, time.UTC)
	next := s.Next(ref)

	// Should be Jan 18 at midnight
	expected := time.Date(2024, 1, 18, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, expected, next)
}

func TestRecurrenceSchedule_StartDateAfterEndDate(t *testing.T) {
	s := &RecurrenceSchedule{RRule: "FREQ=DAILY"}
	startDate := time.Date(2024, 1, 25, 14, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	s.StartDate = &startDate
	s.EndDate = &endDate

	err := s.Parse()
	assert.ErrorIs(t, err, ErrStartDateAfterEndDate)
}

func TestRecurrenceSchedule_IsWithinWindow(t *testing.T) {
	s := &RecurrenceSchedule{RRule: "FREQ=DAILY"}
	startDate := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 25, 14, 0, 0, 0, time.UTC)
	s.StartDate = &startDate
	s.EndDate = &endDate
	require.NoError(t, s.Parse())

	tests := []struct {
		name string
		t    time.Time
		want bool
	}{
		{
			name: "before window",
			t:    time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC),
			want: false,
		},
		{
			name: "at start",
			t:    startDate,
			want: true,
		},
		{
			name: "within window",
			t:    time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
			want: true,
		},
		{
			name: "at end",
			t:    endDate,
			want: true,
		},
		{
			name: "after window",
			t:    time.Date(2024, 1, 26, 15, 0, 0, 0, time.UTC),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.IsWithinWindow(tt.t)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestRecurrenceSchedule_ExecutionIndex(t *testing.T) {
	// FREQ=DAILY with start Jan 15, 2024 09:00 UTC
	startDate := time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC)
	s := &RecurrenceSchedule{
		RRule:     "FREQ=DAILY",
		StartDate: &startDate,
	}
	require.NoError(t, s.Parse())

	tests := []struct {
		name string
		t    time.Time
		n    int
		ok   bool
	}{
		{
			name: "before first occurrence",
			t:    time.Date(2024, 1, 14, 12, 0, 0, 0, time.UTC),
			n:    0,
			ok:   false,
		},
		{
			name: "first occurrence (Jan 15)",
			t:    time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC),
			n:    1,
			ok:   true,
		},
		{
			name: "during first day",
			t:    time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC),
			n:    1,
			ok:   true,
		},
		{
			name: "second occurrence (Jan 16)",
			t:    time.Date(2024, 1, 16, 9, 0, 0, 0, time.UTC),
			n:    2,
			ok:   true,
		},
		{
			name: "third occurrence (Jan 17)",
			t:    time.Date(2024, 1, 17, 9, 0, 0, 0, time.UTC),
			n:    3,
			ok:   true,
		},
		{
			name: "tenth occurrence",
			t:    time.Date(2024, 1, 24, 9, 0, 0, 0, time.UTC),
			n:    10,
			ok:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, ok := s.ExecutionIndex(tt.t)
			assert.Equal(t, tt.n, n, "execution index")
			assert.Equal(t, tt.ok, ok, "ok")
		})
	}
}

func TestRecurrenceSchedule_ExecutionIndex_Weekly(t *testing.T) {
	// Every Monday and Wednesday, starting Monday Jan 15, 2024
	s := &RecurrenceSchedule{
		RRule:     "FREQ=WEEKLY;BYDAY=MO,WE",
		StartDate: ptr(time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC)),
	}
	require.NoError(t, s.Parse())

	// Jan 15 = 1st, Jan 17 = 2nd, Jan 22 = 3rd, Jan 24 = 4th
	n, ok := s.ExecutionIndex(time.Date(2024, 1, 24, 10, 0, 0, 0, time.UTC))
	assert.True(t, ok)
	assert.Equal(t, 4, n)
}

func ptr(t time.Time) *time.Time { return &t }

func TestRecurrenceSchedule_WeeklyPattern(t *testing.T) {
	// Every Monday and Wednesday
	s := &RecurrenceSchedule{RRule: "FREQ=WEEKLY;BYDAY=MO,WE"}
	startDate := time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC) // Monday Jan 15, 2024
	s.StartDate = &startDate
	require.NoError(t, s.Parse())

	// Reference time is Monday 10:00
	ref := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	next := s.Next(ref)

	// Next should be Wednesday Jan 17 at 9:00
	expected := time.Date(2024, 1, 17, 9, 0, 0, 0, time.UTC)
	assert.Equal(t, expected, next)

	// From Wednesday, next should be Monday Jan 22
	ref = time.Date(2024, 1, 17, 10, 0, 0, 0, time.UTC)
	next = s.Next(ref)
	expected = time.Date(2024, 1, 22, 9, 0, 0, 0, time.UTC)
	assert.Equal(t, expected, next)
}

func TestRecurrenceSchedule_BiweeklyPattern(t *testing.T) {
	// Every 2 weeks on Monday and Wednesday
	s := &RecurrenceSchedule{RRule: "FREQ=WEEKLY;INTERVAL=2;BYDAY=MO,WE"}
	startDate := time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC) // Monday Jan 15, 2024
	s.StartDate = &startDate
	require.NoError(t, s.Parse())

	// Reference time is Monday Jan 15 10:00
	ref := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	next := s.Next(ref)

	// Next should be Wednesday Jan 17 at 9:00 (same week)
	expected := time.Date(2024, 1, 17, 9, 0, 0, 0, time.UTC)
	assert.Equal(t, expected, next)

	// From Jan 17, next should be Monday Jan 29 (skip one week)
	ref = time.Date(2024, 1, 17, 10, 0, 0, 0, time.UTC)
	next = s.Next(ref)
	expected = time.Date(2024, 1, 29, 9, 0, 0, 0, time.UTC)
	assert.Equal(t, expected, next)
}

func TestRecurrenceSchedule_MonthlyPattern(t *testing.T) {
	// First day of every month
	s := &RecurrenceSchedule{RRule: "FREQ=MONTHLY;BYMONTHDAY=1"}
	startDate := time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC)
	s.StartDate = &startDate
	require.NoError(t, s.Parse())

	// Reference time is Jan 15
	ref := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	next := s.Next(ref)

	// Next should be Feb 1
	expected := time.Date(2024, 2, 1, 9, 0, 0, 0, time.UTC)
	assert.Equal(t, expected, next)
}

func TestRecurrenceSchedule_Splay(t *testing.T) {
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	s := &RecurrenceSchedule{
		RRule:     "FREQ=DAILY",
		StartDate: &startDate,
		Splay:     30 * time.Minute,
	}
	require.NoError(t, s.Parse())

	assert.Equal(t, 30*time.Minute, s.GetMaxSplayDuration())
}

func TestRecurrenceSchedule_ValidateSplay(t *testing.T) {
	tests := []struct {
		name    string
		splay   time.Duration
		wantErr bool
	}{
		{
			name:    "zero splay",
			splay:   0,
			wantErr: false,
		},
		{
			name:    "small splay",
			splay:   5 * time.Minute,
			wantErr: false,
		},
		{
			name:    "max splay",
			splay:   MaxSplay,
			wantErr: false,
		},
		{
			name:    "over max splay",
			splay:   MaxSplay + time.Hour,
			wantErr: true,
		},
		{
			name:    "negative splay",
			splay:   -1 * time.Minute,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &RecurrenceSchedule{
				RRule: "FREQ=DAILY",
				Splay: tt.splay,
			}
			err := s.ValidateSplay()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRecurrenceSchedule_Unpack(t *testing.T) {
	cfg := map[string]interface{}{
		"rrule":      "FREQ=WEEKLY;BYDAY=MO,WE,FR",
		"start_date": "2024-01-15T09:00:00Z",
		"end_date":   "2024-12-31T23:59:59Z",
		"splay":      "15m",
	}

	s := &RecurrenceSchedule{}
	err := s.Unpack(cfg)
	require.NoError(t, err)

	assert.Equal(t, "FREQ=WEEKLY;BYDAY=MO,WE,FR", s.RRule)
	assert.NotNil(t, s.StartDate)
	assert.NotNil(t, s.EndDate)
	assert.Equal(t, 15*time.Minute, s.Splay)
	assert.True(t, s.IsActive())
}

func TestRecurrenceSchedule_UnpackDefaults(t *testing.T) {
	cfg := map[string]interface{}{
		"rrule": "FREQ=DAILY",
	}

	s := &RecurrenceSchedule{}
	err := s.Unpack(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "start_date is required")
}

func TestRecurrenceSchedule_UnpackInvalidSplay(t *testing.T) {
	tests := []struct {
		name    string
		cfg     map[string]interface{}
		wantErr bool
	}{
		{
			name: "invalid splay format",
			cfg: map[string]interface{}{
				"rrule":      "FREQ=DAILY",
				"start_date": "2024-01-15T09:00:00Z",
				"splay":      "not a duration",
			},
			wantErr: true,
		},
		{
			name: "negative splay",
			cfg: map[string]interface{}{
				"rrule":      "FREQ=DAILY",
				"start_date": "2024-01-15T09:00:00Z",
				"splay":      "-5m",
			},
			wantErr: true,
		},
		{
			name: "splay exceeds max",
			cfg: map[string]interface{}{
				"rrule":      "FREQ=DAILY",
				"start_date": "2024-01-15T09:00:00Z",
				"splay":      "13h",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &RecurrenceSchedule{}
			err := s.Unpack(tt.cfg)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRecurrenceSchedule_String(t *testing.T) {
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	s := &RecurrenceSchedule{
		RRule:     "FREQ=WEEKLY;INTERVAL=2;BYDAY=MO,WE",
		StartDate: &startDate,
	}
	require.NoError(t, s.Parse())

	assert.Equal(t, "FREQ=WEEKLY;INTERVAL=2;BYDAY=MO,WE", s.String())
}

func TestRecurrenceSchedule_IsActive(t *testing.T) {
	// Active schedule
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	s := &RecurrenceSchedule{RRule: "FREQ=DAILY", StartDate: &startDate}
	require.NoError(t, s.Parse())
	assert.True(t, s.IsActive())

	// Inactive - empty rrule
	s2 := &RecurrenceSchedule{}
	assert.False(t, s2.IsActive())

	// Inactive - not parsed
	s3 := &RecurrenceSchedule{RRule: "FREQ=DAILY", StartDate: &startDate}
	assert.False(t, s3.IsActive())
}
