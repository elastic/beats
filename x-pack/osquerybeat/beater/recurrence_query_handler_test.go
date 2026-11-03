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
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/ecs"
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
	case <-time.After(time.Second):
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

// rrulePublishedRows is one Publish call for an RRULE run.
type rrulePublishedRows struct {
	index, idValue, idKey, responseID string
	meta                              map[string]any
	hits                              []map[string]any
}

// rruleRowsPublisher records the rows and responses the scheduled worker publishes.
type rruleRowsPublisher struct {
	profileCallbackPublisher
	rows chan rrulePublishedRows
}

func (p *rruleRowsPublisher) Publish(index, idValue, idFieldKey, responseID, _, _, _, _ string, meta map[string]any, hits []map[string]any, _ ecs.Mapping, _ any) {
	p.rows <- rrulePublishedRows{index: index, idValue: idValue, idKey: idFieldKey, responseID: responseID, meta: meta, hits: hits}
}

func TestRecurrenceRowsPublished(t *testing.T) {
	a, b, c := map[string]any{"value": "a"}, map[string]any{"value": "b"}, map[string]any{"value": "c"}
	type batch struct {
		typ, action string
		hits        []map[string]any
	}
	tests := []struct {
		name     string
		snapshot bool
		runs     [][]map[string]any // Rows each execution returns; the last one is checked.
		want     []batch
	}{
		{
			name:     "snapshot",
			snapshot: true,
			runs:     [][]map[string]any{{a, b}},
			want:     []batch{{typ: "snapshot", hits: []map[string]any{a, b}}},
		},
		{
			name: "diff",
			runs: [][]map[string]any{{a, b}, {b, c}},
			want: []batch{{typ: "diff", action: "added", hits: []map[string]any{c}}, {typ: "diff", action: "removed", hits: []map[string]any{a}}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Apply an unprofiled policy with an explicit schedule ID.
			log := logptest.NewTestingLogger(t, "rrule")
			disabled, removed := false, true
			inputs := profileTestInputs("default", "SELECT 1", &disabled)
			query := inputs[0].Osquery.Schedule["scheduled"]
			query.ScheduleID = "policy-schedule"
			query.Snapshot = &test.snapshot
			query.Removed = &removed
			inputs[0].Osquery.Schedule["scheduled"] = query
			plugin := NewConfigPlugin(log)
			require.NoError(t, plugin.Set(inputs), "stage the policy")
			_, err := plugin.GenerateConfig(t.Context())
			require.NoError(t, err, "apply the policy")
			publisher := &rruleRowsPublisher{
				profileCallbackPublisher: profileCallbackPublisher{responses: make(chan scheduledProfileResponse, 1)},
				rows:                     make(chan rrulePublishedRows, 4),
			}
			d := newScheduledResultDispatcher(monitoring.NewRegistry(), log, config.DefaultScheduledMaxEvents,
				func(func() bool, func(), func(int)) osquerybeatPublisher { return publisher }, (&osquerybeat{log: log}).handleScheduledQueryResult)
			require.NoError(t, d.configure(inputs, config.ScheduledPublishingConfig{MaxEvents: config.DefaultScheduledMaxEvents}), "configure the scheduled clients")
			stopOnCleanup(t, d)

			// Run each execution and collect what the worker publishes for it.
			run := 0
			cli := profileQueryFunc(func(context.Context, string, time.Duration) ([]map[string]any, error) {
				hits := test.runs[run]
				run++
				return hits, nil
			})
			h := newRecurrenceQueryHandler(log, cli, plugin, d, nil, "5.12.0")
			sq := scheduler.ScheduledQuery{Name: "scheduled", Config: query, Timeout: time.Second}
			var response scheduledProfileResponse
			var got []rrulePublishedRows
			for i := range test.runs {
				require.NoError(t, h.executeQuery(t.Context(), sq, i+1, time.Now()), "run execution %d", i+1)
				select {
				case response = <-publisher.responses:
				case <-time.After(time.Second):
					t.Fatalf("the response of execution %d was not published", i+1)
				}
				// The worker publishes a run's rows before its response.
				got = got[:0]
				for len(publisher.rows) > 0 {
					got = append(got, <-publisher.rows)
				}
			}

			// The last execution's rows keep their routing, correlation and metadata.
			require.Len(t, got, len(test.want), "publish each non-empty row group once")
			total := 0
			for i, want := range test.want {
				assert.Equal(t, config.Datastream("default"), got[i].index, "row group %d index", i)
				assert.Equal(t, "schedule_id", got[i].idKey, "row group %d ID field", i)
				assert.Equal(t, "policy-schedule", got[i].idValue, "row group %d schedule ID", i)
				assert.Equal(t, response.responseID, got[i].responseID, "row group %d must share the run's response ID", i)
				assert.Equal(t, want.typ, got[i].meta["type"], "row group %d osquery_meta.type", i)
				if want.action == "" {
					assert.NotContains(t, got[i].meta, "action", "row group %d must have no osquery_meta.action", i)
				} else {
					assert.Equal(t, want.action, got[i].meta["action"], "row group %d osquery_meta.action", i)
				}
				assert.Equal(t, len(test.runs), got[i].meta["schedule_execution_count"], "row group %d execution count", i)
				assert.Equal(t, want.hits, got[i].hits, "row group %d rows", i)
				total += len(want.hits)
			}
			assert.Equal(t, total, response.count, "the response must count the run's published rows")
			assert.Equal(t, int64(len(test.runs)), response.executionCount, "the response must carry the execution count")
		})
	}
}

func TestRecurrenceRunOfOtherSQLIsStale(t *testing.T) {
	// The policy routes "scheduled" with SELECT 1; the run executes SELECT 2.
	d, plugin, publisher, release := outageDispatcher(t, outageLimits)
	release()
	h := newRecurrenceQueryHandler(logptest.NewTestingLogger(t, "rrule"), rruleQueryClient([]map[string]any{{"value": "a"}}, nil), plugin, d, nil, "5.12.0")
	sq := scheduler.ScheduledQuery{Name: "scheduled", Config: config.Query{Query: "SELECT 2"}, Timeout: time.Second}
	require.NoError(t, h.executeQuery(t.Context(), sq, 1, time.Now()), "run the RRULE query")
	assert.Equal(t, uint64(1), d.metrics.dropped[dropStale].Get(), "a run of SQL the policy does not route must be dropped as stale")
	items, _ := d.queue.stats()
	assert.Zero(t, items, "the stale run must not be queued")
	assert.Zero(t, publisher.calls.Load(), "the stale run must not be published")
}

func TestRecurrenceSkipsQueryWithoutNamespace(t *testing.T) {
	log := logptest.NewTestingLogger(t, "rrule")
	d := newScheduledResultDispatcher(monitoring.NewRegistry(), log, outageMaxEvents,
		func(func() bool, func(), func(int)) osquerybeatPublisher { return &mockBeatPublisher{} }, (&osquerybeat{log: log}).handleScheduledQueryResult)
	require.NoError(t, d.configure(outageInputs("old"), outageLimits), "configure the scheduled clients")
	stopOnCleanup(t, d)
	plugin := NewConfigPlugin(log)
	require.NoError(t, plugin.Set(outageInputs("old")), "stage the policy")
	_, err := plugin.GenerateConfig(t.Context())
	require.NoError(t, err, "apply the policy")

	// A query the policy does not route must not run, and counts as stale.
	cli := profileQueryFunc(func(context.Context, string, time.Duration) ([]map[string]any, error) {
		t.Error("a query without a namespace must not run")
		return nil, nil
	})
	h := newRecurrenceQueryHandler(log, cli, plugin, d, nil, "5.12.0")
	sq := scheduler.ScheduledQuery{Name: "unknown", Config: config.Query{Query: "SELECT 1"}, Timeout: time.Second}
	assert.NoError(t, h.executeQuery(t.Context(), sq, 1, time.Now()), "skipping is not a query failure")
	assert.Equal(t, uint64(1), d.metrics.dropped[dropStale].Get(), "count the skipped run as stale")
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
				func(func() bool, func(), func(int)) osquerybeatPublisher { return publisher }, (&osquerybeat{log: log}).handleScheduledQueryResult)
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
