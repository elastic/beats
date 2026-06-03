// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

func TestRecurrenceQueryHandlerCreateScheduledQueryOptions(t *testing.T) {
	log := logptest.NewTestingLogger(t, "")
	handler := newRecurrenceQueryHandler(log, nil, nil, nil, nil, "5.12.0")

	snapshot := false
	removed := false
	sq, err := handler.createScheduledQuery("q", config.Query{
		Query:    "select 1",
		Platform: "all",
		Version:  "5.11.0",
		Snapshot: &snapshot,
		Removed:  &removed,
		CommonScheduleConfig: config.CommonScheduleConfig{
			ScheduleID: "schedule-1",
		},
		RRuleSchedule: &config.RRuleScheduleConfig{
			RRule:     "FREQ=DAILY",
			StartDate: "2024-01-01T00:00:00Z",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, sq)
	assert.Equal(t, "select 1", sq.SQL())
	assert.Equal(t, "schedule-1", sq.ScheduleID())
	assert.False(t, sq.Snapshot())
	assert.False(t, sq.Removed())
	assert.Equal(t, "all", sq.Config.Platform)
	assert.Equal(t, "5.11.0", sq.Config.Version)
}

func TestRecurrenceQueryHandlerCreateScheduledQuerySkipsIneligibleQueries(t *testing.T) {
	log := logptest.NewTestingLogger(t, "")
	handler := newRecurrenceQueryHandler(log, nil, nil, nil, nil, "5.12.0")
	rrule := &config.RRuleScheduleConfig{
		RRule:     "FREQ=DAILY",
		StartDate: "2024-01-01T00:00:00Z",
	}

	sq, err := handler.createScheduledQuery("platform", config.Query{
		Query:         "select 1",
		Platform:      "definitely-not-this-platform",
		RRuleSchedule: rrule,
	})
	require.NoError(t, err)
	assert.Nil(t, sq)

	sq, err = handler.createScheduledQuery("version", config.Query{
		Query:         "select 1",
		Version:       "99.0.0",
		RRuleSchedule: rrule,
	})
	require.NoError(t, err)
	assert.Nil(t, sq)
}

func TestRRuleDiffRows(t *testing.T) {
	previous := rowsByKey([]map[string]interface{}{
		{"id": int64(1), "name": "old"},
		{"id": int64(2), "name": "same"},
	})
	current := rowsByKey([]map[string]interface{}{
		{"id": int64(2), "name": "same"},
		{"id": int64(3), "name": "new"},
	})

	added, removed := diffRows(previous, current, true)
	assert.ElementsMatch(t, []map[string]interface{}{{"id": int64(3), "name": "new"}}, added)
	assert.ElementsMatch(t, []map[string]interface{}{{"id": int64(1), "name": "old"}}, removed)

	added, removed = diffRows(previous, current, false)
	assert.ElementsMatch(t, []map[string]interface{}{{"id": int64(3), "name": "new"}}, added)
	assert.Empty(t, removed)
}

func TestPlatformAndVersionMatches(t *testing.T) {
	assert.True(t, platformMatches("", "linux"))
	assert.True(t, platformMatches("all", "linux"))
	assert.True(t, platformMatches("posix", "darwin"))
	assert.True(t, platformMatches("linux,darwin", "linux"))
	assert.False(t, platformMatches("windows", "linux"))

	assert.True(t, versionMatches("", "5.12.0"))
	assert.True(t, versionMatches("5.11.0", "5.12.0"))
	assert.True(t, versionMatches("5.12.0", "5.12.0"))
	assert.False(t, versionMatches("5.13.0", "5.12.0"))
}
