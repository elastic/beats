// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/scheduler"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

// rruleQueryClient answers the runtime profiling snapshot, and returns hits
// and err for any other query.
func rruleQueryClient(hits []map[string]any, err error) profileQueryFunc {
	return func(_ context.Context, sql string, _ time.Duration) ([]map[string]any, error) {
		if strings.Contains(sql, "osquery_info") {
			return []map[string]any{{"pid": int64(1), "resident_size": int64(100), "user_time": int64(1), "system_time": int64(1)}}, nil
		}
		return hits, err
	}
}

func TestRecurrenceResultsUseScheduledQueue(t *testing.T) {
	// Hold the scheduled output on an earlier native result.
	d, plugin, blocked, release := outageDispatcher(t, outageLimits)
	d.enqueue(t.Context(), nil, plugin, QueryResult{Name: "scheduled", Action: "diff"}, nil)
	waitForResultPublication(t, blocked)

	// One RRULE execution must queue its rows and response without waiting.
	// Routing needs only the query's name, namespace and SQL, so the native
	// schedule entry of the outage policy serves.
	hits := []map[string]any{{"value": "a"}, {"value": "b"}, {"value": "c"}}
	h := newRecurrenceQueryHandler(logptest.NewTestingLogger(t, "rrule"), rruleQueryClient(hits, nil), plugin, d, nil, "5.12.0")
	sq := scheduler.ScheduledQuery{Name: "scheduled", Config: config.Query{Query: "SELECT 1"}, Timeout: time.Second}
	done := make(chan error, 1)
	go func() { done <- h.executeQuery(t.Context(), sq, 1, time.Now()) }()
	select {
	case err := <-done:
		require.NoError(t, err, "run the RRULE query")
	case <-time.After(50 * time.Millisecond):
		t.Fatal("the RRULE execution waited for the blocked output")
	}
	items, events := d.queue.stats()
	assert.Equal(t, 1, items, "queue the run as one item")
	assert.Equal(t, len(hits)+1, events, "count the run's rows and its response")

	// The worker publishes the run once the output recovers.
	release()
	assert.Eventually(t, func() bool { return blocked.calls.Load() == 2 }, time.Second, time.Millisecond,
		"publish the queued run's response")
}

func TestRecurrenceProfilesUseScheduledQueue(t *testing.T) {
	tests := []struct {
		name     string
		queryErr error
	}{
		{name: "with result"},
		{name: "failed query", queryErr: errors.New("query failed")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Apply a profiled policy with an explicit schedule ID.
			log := logptest.NewTestingLogger(t, "rrule")
			inputs := profileTestInputs("default", "SELECT 1", nil)
			query := inputs[0].Osquery.Schedule["scheduled"]
			query.ScheduleID = "policy-schedule"
			inputs[0].Osquery.Schedule["scheduled"] = query
			plugin := NewConfigPlugin(log)
			require.NoError(t, plugin.Set(inputs), "stage the profiled policy")
			_, err := plugin.GenerateConfig(t.Context())
			require.NoError(t, err, "apply the profiled policy")
			publisher := &profileCallbackPublisher{responses: make(chan scheduledProfileResponse, 1), profiles: make(chan collectedProfile, 1)}
			d := newScheduledResultDispatcher(monitoring.NewRegistry(), log, config.DefaultScheduledMaxEvents,
				func(func() bool, func()) osquerybeatPublisher { return publisher }, (&osquerybeat{log: log}).handleScheduledQueryResult)
			require.NoError(t, d.configure(inputs, config.ScheduledPublishingConfig{MaxEvents: config.DefaultScheduledMaxEvents}), "configure the scheduled clients")
			stopOnCleanup(t, d)

			// Run execution 7 of the query.
			h := newRecurrenceQueryHandler(log, rruleQueryClient([]map[string]any{{"value": "a"}}, test.queryErr), plugin, d, nil, "5.12.0")
			sq := scheduler.ScheduledQuery{Name: "scheduled", Config: query, Timeout: time.Second}
			assert.ErrorIs(t, h.executeQuery(t.Context(), sq, 7, time.Now()), test.queryErr, "report the query's own outcome")

			// The worker publishes the profile with the run's correlation.
			var profile collectedProfile
			select {
			case profile = <-publisher.profiles:
			case <-time.After(time.Second):
				t.Fatal("the RRULE profile was not published")
			}
			assert.Equal(t, "policy-schedule", profile.scheduleID, "RRULE profiles carry the schedule ID")
			assert.Equal(t, int64(7), profile.scheduleExecutionCount, "RRULE profiles carry the execution count")
			assert.Equal(t, "space", profile.spaceID, "RRULE profiles carry the policy space")
			assert.Equal(t, "rrule", profile.profile["source"], "keep the RRULE profile schema")
			if test.queryErr != nil {
				assert.Equal(t, int64(1), profile.profile["exit"], "the profile of a failed run records the failure")
				return
			}
			select {
			case response := <-publisher.responses:
				assert.Equal(t, response.responseID, profile.responseID, "the profile and response share the run's response ID")
			case <-time.After(time.Second):
				t.Fatal("the RRULE response was not published")
			}
		})
	}
}

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
	previous := rowsByKey([]map[string]any{
		{"id": int64(1), "name": "old"},
		{"id": int64(2), "name": "same"},
	})
	current := rowsByKey([]map[string]any{
		{"id": int64(2), "name": "same"},
		{"id": int64(3), "name": "new"},
	})

	added, removed := diffRows(previous, current, true)
	assert.ElementsMatch(t, []map[string]any{{"id": int64(3), "name": "new"}}, added)
	assert.ElementsMatch(t, []map[string]any{{"id": int64(1), "name": "old"}}, removed)

	added, removed = diffRows(previous, current, false)
	assert.ElementsMatch(t, []map[string]any{{"id": int64(3), "name": "new"}}, added)
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
	assert.False(t, versionMatches("not-semver", "5.12.0"))
	assert.False(t, versionMatches("5.12.0", "not-semver"))
	assert.False(t, versionMatches("5.12.0", ""))
}
