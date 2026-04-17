// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePackScheduleDefaults(t *testing.T) {
	t.Run("ok native only", func(t *testing.T) {
		err := ValidatePackScheduleDefaults(Pack{
			DefaultNativeSchedule: NativeSchedule{Interval: 60},
		})
		assert.NoError(t, err)
	})

	t.Run("ok rrule only", func(t *testing.T) {
		err := ValidatePackScheduleDefaults(Pack{
			DefaultRRuleSchedule: &RRuleScheduleConfig{RRule: "FREQ=DAILY", StartDate: "2024-01-01T00:00:00Z"},
		})
		assert.NoError(t, err)
	})

	t.Run("ok empty", func(t *testing.T) {
		assert.NoError(t, ValidatePackScheduleDefaults(Pack{}))
	})

	t.Run("reject both", func(t *testing.T) {
		err := ValidatePackScheduleDefaults(Pack{
			DefaultNativeSchedule: NativeSchedule{Interval: 60},
			DefaultRRuleSchedule:  &RRuleScheduleConfig{RRule: "FREQ=DAILY", StartDate: "2024-01-01T00:00:00Z"},
		})
		assert.ErrorIs(t, err, ErrPackConflictingScheduleDefaults)
	})

	t.Run("reject schedule_id without interval", func(t *testing.T) {
		err := ValidatePackScheduleDefaults(Pack{
			DefaultNativeSchedule: NativeSchedule{ScheduleID: "x"},
		})
		assert.ErrorIs(t, err, ErrPackNativeScheduleMetadataWithoutInterval)
	})

	t.Run("reject start_date without interval", func(t *testing.T) {
		err := ValidatePackScheduleDefaults(Pack{
			DefaultNativeSchedule: NativeSchedule{StartDate: "2024-01-01T00:00:00Z"},
		})
		assert.ErrorIs(t, err, ErrPackNativeScheduleMetadataWithoutInterval)
	})
}

func TestValidateQueryScheduleMode(t *testing.T) {
	t.Run("ok native only", func(t *testing.T) {
		assert.NoError(t, ValidateQueryScheduleMode(Query{
			Query: "select 1",
			NativeSchedule: NativeSchedule{
				Interval: 60,
			},
		}))
	})

	t.Run("ok rrule only", func(t *testing.T) {
		assert.NoError(t, ValidateQueryScheduleMode(Query{
			Query:         "select 1",
			RRuleSchedule: &RRuleScheduleConfig{RRule: "FREQ=DAILY", StartDate: "2024-01-01T00:00:00Z"},
		}))
	})

	t.Run("reject both", func(t *testing.T) {
		err := ValidateQueryScheduleMode(Query{
			Query: "select 1",
			NativeSchedule: NativeSchedule{
				Interval: 60,
			},
			RRuleSchedule: &RRuleScheduleConfig{RRule: "FREQ=DAILY", StartDate: "2024-01-01T00:00:00Z"},
		})
		assert.ErrorIs(t, err, ErrConflictingScheduleModes)
	})
}

func TestMergeQueryWithPackScheduleDefaults(t *testing.T) {
	packRRULE := &RRuleScheduleConfig{
		RRule:     "FREQ=DAILY",
		StartDate: "2024-01-01T00:00:00Z",
	}

	t.Run("inherits native from pack", func(t *testing.T) {
		pack := Pack{
			DefaultNativeSchedule: NativeSchedule{
				Interval:   120,
				ScheduleID: "pack-sched",
				StartDate:  "2024-06-01T00:00:00Z",
			},
		}
		q := Query{Query: "select 1"}
		got, err := MergeQueryWithPackScheduleDefaults(pack, q)
		require.NoError(t, err)
		assert.Equal(t, 120, got.Interval)
		assert.Equal(t, "pack-sched", got.ScheduleID)
		assert.Equal(t, "2024-06-01T00:00:00Z", got.StartDate)
	})

	t.Run("query overrides native fields", func(t *testing.T) {
		pack := Pack{
			DefaultNativeSchedule: NativeSchedule{
				Interval:   120,
				ScheduleID: "pack-sched",
				StartDate:  "2024-06-01T00:00:00Z",
			},
		}
		q := Query{
			Query: "select 1",
			NativeSchedule: NativeSchedule{
				Interval:   30,
				ScheduleID: "q-sched",
				StartDate:  "2025-01-01T00:00:00Z",
			},
		}
		got, err := MergeQueryWithPackScheduleDefaults(pack, q)
		require.NoError(t, err)
		assert.Equal(t, 30, got.Interval)
		assert.Equal(t, "q-sched", got.ScheduleID)
		assert.Equal(t, "2025-01-01T00:00:00Z", got.StartDate)
	})

	t.Run("inherits rrule from pack", func(t *testing.T) {
		pack := Pack{DefaultRRuleSchedule: packRRULE}
		q := Query{Query: "select 1"}
		got, err := MergeQueryWithPackScheduleDefaults(pack, q)
		require.NoError(t, err)
		require.NotNil(t, got.RRuleSchedule)
		assert.Equal(t, "FREQ=DAILY", got.RRuleSchedule.RRule)
		assert.Equal(t, "2024-01-01T00:00:00Z", got.RRuleSchedule.StartDate)
	})

	t.Run("query rrule skips pack native merge fields", func(t *testing.T) {
		pack := Pack{
			DefaultNativeSchedule: NativeSchedule{Interval: 300, ScheduleID: "p", StartDate: "2024-01-01T00:00:00Z"},
		}
		q := Query{
			Query:         "select 1",
			RRuleSchedule: &RRuleScheduleConfig{RRule: "FREQ=WEEKLY;BYDAY=MO", StartDate: "2024-01-01T00:00:00Z"},
		}
		got, err := MergeQueryWithPackScheduleDefaults(pack, q)
		require.NoError(t, err)
		assert.Equal(t, 0, got.Interval)
		assert.Equal(t, "", got.ScheduleID)
		assert.True(t, got.RRuleSchedule.IsEnabled())
		err = ValidatePackQueriesAfterMerge(Pack{
			DefaultNativeSchedule: pack.DefaultNativeSchedule,
			Queries:               map[string]Query{"q": got},
		})
		assert.ErrorIs(t, err, ErrPackQueryViolatesPackScheduleDefault, "single-query pack must still match pack native default")
	})

	t.Run("query interval skips pack rrule merge fields", func(t *testing.T) {
		pack := Pack{DefaultRRuleSchedule: packRRULE}
		q := Query{
			Query: "select 1",
			NativeSchedule: NativeSchedule{
				Interval: 60,
			},
		}
		got, err := MergeQueryWithPackScheduleDefaults(pack, q)
		require.NoError(t, err)
		assert.Equal(t, 60, got.Interval)
		assert.False(t, got.RRuleSchedule.IsEnabled())
		err = ValidatePackQueriesAfterMerge(Pack{
			DefaultRRuleSchedule: pack.DefaultRRuleSchedule,
			Queries:              map[string]Query{"q": got},
		})
		assert.ErrorIs(t, err, ErrPackQueryViolatesPackScheduleDefault, "single-query pack must still match pack rrule default")
	})

	t.Run("reject query that already mixes native and rrule", func(t *testing.T) {
		q := Query{
			Query: "select 1",
			NativeSchedule: NativeSchedule{
				Interval: 10,
			},
			RRuleSchedule: packRRULE,
		}
		_, err := MergeQueryWithPackScheduleDefaults(Pack{}, q)
		assert.ErrorIs(t, err, ErrConflictingScheduleModes)
	})

	t.Run("inherits pack default schedule_id and space_id", func(t *testing.T) {
		pack := Pack{
			DefaultScheduleID: "policy-sched",
			DefaultSpaceID:    "space-1",
		}
		q := Query{Query: "select 1", NativeSchedule: NativeSchedule{Interval: 60}}
		got, err := MergeQueryWithPackScheduleDefaults(pack, q)
		require.NoError(t, err)
		assert.Equal(t, "policy-sched", got.ScheduleID)
		assert.Equal(t, "space-1", got.SpaceID)
	})

	t.Run("query overrides pack default schedule_id and space_id", func(t *testing.T) {
		pack := Pack{
			DefaultScheduleID: "pack-sched",
			DefaultSpaceID:    "pack-space",
		}
		q := Query{
			Query:          "select 1",
			NativeSchedule: NativeSchedule{Interval: 60, ScheduleID: "q-sched"},
			SpaceID:        "q-space",
		}
		got, err := MergeQueryWithPackScheduleDefaults(pack, q)
		require.NoError(t, err)
		assert.Equal(t, "q-sched", got.ScheduleID)
		assert.Equal(t, "q-space", got.SpaceID)
	})

	t.Run("pack default schedule_id applies to inherited rrule query", func(t *testing.T) {
		pack := Pack{
			DefaultRRuleSchedule: packRRULE,
			DefaultScheduleID:    "rrule-policy-sched",
			DefaultSpaceID:       "s1",
		}
		q := Query{Query: "select 1"}
		got, err := MergeQueryWithPackScheduleDefaults(pack, q)
		require.NoError(t, err)
		assert.Equal(t, "rrule-policy-sched", got.ScheduleID)
		assert.Equal(t, "s1", got.SpaceID)
		require.NotNil(t, got.RRuleSchedule)
	})
}

func TestValidatePackQueriesAfterMerge(t *testing.T) {
	rrule := &RRuleScheduleConfig{RRule: "FREQ=DAILY", StartDate: "2024-01-01T00:00:00Z"}

	t.Run("ok uniform native without pack defaults", func(t *testing.T) {
		err := ValidatePackQueriesAfterMerge(Pack{
			Queries: map[string]Query{
				"a": {Query: "select 1", NativeSchedule: NativeSchedule{Interval: 60}},
				"b": {Query: "select 2", NativeSchedule: NativeSchedule{Interval: 120}},
			},
		})
		assert.NoError(t, err)
	})

	t.Run("ok uniform rrule without pack defaults", func(t *testing.T) {
		err := ValidatePackQueriesAfterMerge(Pack{
			Queries: map[string]Query{
				"a": {Query: "select 1", RRuleSchedule: rrule},
				"b": {Query: "select 2", RRuleSchedule: rrule},
			},
		})
		assert.NoError(t, err)
	})

	t.Run("ok uniform unscheduled without pack defaults", func(t *testing.T) {
		err := ValidatePackQueriesAfterMerge(Pack{
			Queries: map[string]Query{
				"a": {Query: "select 1"},
				"b": {Query: "select 2"},
			},
		})
		assert.NoError(t, err)
	})

	t.Run("reject mixed native and unscheduled with no pack defaults", func(t *testing.T) {
		err := ValidatePackQueriesAfterMerge(Pack{
			Queries: map[string]Query{
				"a": {Query: "select 1", NativeSchedule: NativeSchedule{Interval: 60}},
				"b": {Query: "select 2"},
			},
		})
		assert.ErrorIs(t, err, ErrPackMixedScheduleModes)
	})

	t.Run("reject mixed native and rrule with no pack defaults", func(t *testing.T) {
		err := ValidatePackQueriesAfterMerge(Pack{
			Queries: map[string]Query{
				"a": {Query: "select 1", NativeSchedule: NativeSchedule{Interval: 60}},
				"b": {Query: "select 2", RRuleSchedule: rrule},
			},
		})
		assert.ErrorIs(t, err, ErrPackMixedScheduleModes)
	})

	t.Run("ok all inherit native from pack default", func(t *testing.T) {
		p := Pack{
			DefaultNativeSchedule: NativeSchedule{Interval: 300, ScheduleID: "s"},
			Queries: map[string]Query{
				"a": {Query: "select 1"},
				"b": {Query: "select 2"},
			},
		}
		for n, q := range p.Queries {
			mq, err := MergeQueryWithPackScheduleDefaults(p, q)
			require.NoError(t, err)
			p.Queries[n] = mq
		}
		assert.NoError(t, ValidatePackQueriesAfterMerge(p))
	})

	t.Run("reject rrule query when pack has native default", func(t *testing.T) {
		p := Pack{
			DefaultNativeSchedule: NativeSchedule{Interval: 300},
			Queries: map[string]Query{
				"native": {Query: "select 1"},
				"rrule":  {Query: "select 2", RRuleSchedule: rrule},
			},
		}
		for n, q := range p.Queries {
			mq, err := MergeQueryWithPackScheduleDefaults(p, q)
			require.NoError(t, err)
			p.Queries[n] = mq
		}
		err := ValidatePackQueriesAfterMerge(p)
		assert.ErrorIs(t, err, ErrPackQueryViolatesPackScheduleDefault)
	})
}
