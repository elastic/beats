// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"testing"
	"time"

	"github.com/osquery/osquery-go/plugin/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

type profileQueryFunc func(context.Context, string, time.Duration) ([]map[string]any, error)

func (f profileQueryFunc) Query(ctx context.Context, sql string, timeout time.Duration) ([]map[string]any, error) {
	return f(ctx, sql, timeout)
}

type collectedProfile struct {
	name, scheduleID, responseID, spaceID string
	scheduleExecutionCount                int64
	profile                               map[string]any
}

// profileTestQueue receives the profiles a worker hands to the scheduled queue.
type profileTestQueue chan scheduledProfileItem

func (q profileTestQueue) enqueue(item scheduledProfileItem) { q <- item }

func profileTestInputs(namespace, sql string, profiling *bool) []config.InputConfig {
	return []config.InputConfig{{
		Datastream: config.DatastreamConfig{Dataset: config.DefaultDataset, Namespace: namespace},
		Osquery: &config.OsqueryConfig{Schedule: map[string]config.Query{
			"scheduled": {
				Query:                sql,
				CommonScheduleConfig: config.CommonScheduleConfig{SpaceID: "space"},
				NativeSchedule:       config.NativeSchedule{Interval: 1},
				Profiling:            profiling,
			},
		}},
	}, {
		Datastream: config.DatastreamConfig{Dataset: config.DefaultQueryProfileDataset, Namespace: namespace},
	}}
}

func profileTestPlugin(t *testing.T) *ConfigPlugin {
	t.Helper()
	p := NewConfigPlugin(logptest.NewTestingLogger(t, "scheduled_profile"))
	require.NoError(t, p.Set(profileTestInputs("default", "SELECT 1", nil)), "configure profiling with its default enabled behavior")
	_, err := p.GenerateConfig(t.Context())
	require.NoError(t, err, "apply the initial native schedule")
	return p
}

func profileTestRequest() scheduledProfileRequest {
	return scheduledProfileRequest{
		name:                   "scheduled",
		query:                  "SELECT 1",
		namespace:              "default",
		scheduleID:             "scheduled",
		responseID:             "response",
		spaceID:                "space",
		scheduleExecutionCount: 1,
	}
}

func profileTestRow(sql string) []map[string]any {
	return []map[string]any{{
		"query":             sql,
		"executions":        int64(2),
		"last_wall_time_ms": int64(10),
		"last_user_time":    int64(2),
		"last_system_time":  int64(3),
		"last_memory":       int64(100),
	}}
}

func TestScheduledProfilerBoundsJobsAndJoinsCancellation(t *testing.T) {
	// Hold collection in its RPC so the pending queue cannot drain during admission.
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	query := profileQueryFunc(func(ctx context.Context, _ string, _ time.Duration) ([]map[string]any, error) {
		select {
		case entered <- struct{}{}:
		default:
		}
		select {
		case <-release:
			return profileTestRow("SELECT 1"), nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	})

	// Build a worker with two pending slots whose cleanup always releases the RPC.
	metrics := newScheduledProfileMetrics(monitoring.NewRegistry())
	profiler := newQueryProfiler(logptest.NewTestingLogger(t, "profile"))
	queue := make(profileTestQueue, 4)
	worker := newScheduledQueryProfiler(t.Context(), 2, query, profileTestPlugin(t), profiler, metrics, queue.enqueue)
	t.Cleanup(func() {
		// Release the fake RPC even if a regression blocks admission or shutdown.
		close(release)
		worker.stop()
	})
	request := profileTestRequest()

	// Start one collection and fill both pending slots behind it.
	require.True(t, worker.enqueue(request), "the first profile should reach the owned worker")
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("the profile worker did not start its RPC")
	}

	for range 2 {
		assert.True(t, worker.enqueue(request), "the bounded pending queue should accept its configured jobs")
	}

	// Bound the admission call itself: a blocking send must fail this test promptly.
	enqueued := make(chan bool, 1)
	go func() { enqueued <- worker.enqueue(request) }()
	select {
	case accepted := <-enqueued:
		assert.False(t, accepted, "a full profiling queue must reject work without blocking the logger")
	case <-time.After(time.Second):
		t.Fatal("full-queue admission blocked while the profiling RPC was still waiting")
	}
	assert.Equal(t, uint64(1), metrics.queueFull.Get(), "queue overflow should be observable")

	// Cancellation must join the active RPC and account for all queued metadata.
	done := make(chan struct{})
	go func() {
		worker.stop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("shutdown did not cancel and join the profiling RPC")
	}

	// After shutdown nothing may remain queued, be admitted, or be handed on.
	assert.Empty(t, worker.jobs, "shutdown should release queued metadata references")
	assert.Equal(t, uint64(3), metrics.shutdown.Get(), "shutdown should count the active and queued profiles it discarded")
	assert.False(t, worker.enqueue(request), "the stopped worker must reject later callbacks")
	assert.Empty(t, queue, "cancelled collection must not queue a profile")
}

func TestScheduledProfilerPreservesProfileCorrelation(t *testing.T) {
	// Return metrics from a later execution than the result that queued this profile.
	query := profileQueryFunc(func(context.Context, string, time.Duration) ([]map[string]any, error) {
		return profileTestRow("SELECT 1"), nil
	})

	// Run a worker against that query and capture what it queues.
	metrics := newScheduledProfileMetrics(monitoring.NewRegistry())
	profiler := newQueryProfiler(logptest.NewTestingLogger(t, "profile"))
	queue := make(profileTestQueue, 1)
	worker := newScheduledQueryProfiler(t.Context(), 2, query, profileTestPlugin(t), profiler, metrics, queue.enqueue)
	t.Cleanup(worker.stop)

	// The worker must preserve captured result metadata while collecting current metrics.
	require.True(t, worker.enqueue(profileTestRequest()), "default profiling should enqueue a scheduled profile")
	select {
	case item := <-queue:
		assert.Equal(t, "scheduled", item.name, "preserve the native query name")
		assert.Equal(t, "default", item.namespace, "preserve the result namespace")
		assert.Equal(t, "SELECT 1", item.query, "preserve the SQL the profile describes")
		assert.Equal(t, "scheduled", item.scheduleID, "preserve the schedule identity")
		assert.Equal(t, "response", item.responseID, "preserve correlation with the result and scheduled response")
		assert.Equal(t, int64(1), item.executionCount, "preserve the originating schedule execution")
		assert.Equal(t, "space", item.spaceID, "preserve the captured policy space")
		assert.Equal(t, "scheduled", item.profile["source"], "preserve the scheduled profile schema")
		assert.Equal(t, int64(2), item.profile["executions"], "preserve the native schedule's execution metrics")
	case <-time.After(time.Second):
		t.Fatal("the worker did not queue its collected profile")
	}

	// Collection is counted by the worker, separately from publication.
	require.Eventually(t, func() bool { return metrics.collected.Get() == 1 }, time.Second, time.Millisecond, "completed collection should be counted")
}

func TestScheduledProfilerRejectsPolicyChangesDuringCollection(t *testing.T) {
	// Describe each policy change that must invalidate an in-flight profile.
	disabled := false
	scheduleIDChanged := profileTestInputs("default", "SELECT 1", nil)
	query := scheduleIDChanged[0].Osquery.Schedule["scheduled"]
	query.ScheduleID = "new-schedule"
	scheduleIDChanged[0].Osquery.Schedule["scheduled"] = query
	spaceIDChanged := profileTestInputs("default", "SELECT 1", nil)
	query = spaceIDChanged[0].Osquery.Schedule["scheduled"]
	query.SpaceID = "new-space"
	spaceIDChanged[0].Osquery.Schedule["scheduled"] = query

	for _, change := range []struct {
		name   string
		inputs []config.InputConfig
	}{
		{"namespace changed", profileTestInputs("new", "SELECT 1", nil)},
		{"query replaced", profileTestInputs("default", "SELECT 2", nil)},
		{"schedule id changed", scheduleIDChanged},
		{"space id changed", spaceIDChanged},
		{"profiling disabled", profileTestInputs("default", "SELECT 1", &disabled)},
		{"inputs removed", nil},
	} {
		t.Run(change.name, func(t *testing.T) {
			// Pause collection so the policy changes after admission but before publication.
			entered := make(chan struct{})
			release := make(chan struct{})
			query := profileQueryFunc(func(ctx context.Context, _ string, _ time.Duration) ([]map[string]any, error) {
				close(entered)
				select {
				case <-release:
					return profileTestRow("SELECT 1"), nil
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			})

			// Build a worker bound to the original policy.
			plugin := profileTestPlugin(t)
			metrics := newScheduledProfileMetrics(monitoring.NewRegistry())
			profiler := newQueryProfiler(logptest.NewTestingLogger(t, "profile"))
			queue := make(profileTestQueue, 1)
			worker := newScheduledQueryProfiler(t.Context(), 2, query, plugin, profiler, metrics, queue.enqueue)
			t.Cleanup(worker.stop)

			// Enqueue a profile and wait until collection is in flight.
			require.True(t, worker.enqueue(profileTestRequest()), "enqueue the original policy's profile")
			select {
			case <-entered:
			case <-time.After(time.Second):
				t.Fatal("profile collection did not begin")
			}

			// A pending policy must invalidate old work even before native config refresh.
			require.NoError(t, plugin.Set(change.inputs), "install the pending policy before native config refresh")
			close(release)

			require.Eventually(t, func() bool { return metrics.stale.Get() == 1 }, time.Second, time.Millisecond, "collection must reject a policy change before publication")
			assert.Empty(t, queue, "old profile metadata must not use the new policy destination")
		})
	}
}

func TestScheduledProfilerRejectsReplacementNativeMetrics(t *testing.T) {
	// The native schedule now associates this name with different SQL.
	query := profileQueryFunc(func(context.Context, string, time.Duration) ([]map[string]any, error) {
		return profileTestRow("SELECT 2"), nil
	})

	// Enqueue a profile that captured the original SQL.
	metrics := newScheduledProfileMetrics(monitoring.NewRegistry())
	profiler := newQueryProfiler(logptest.NewTestingLogger(t, "profile"))
	queue := make(profileTestQueue, 1)
	worker := newScheduledQueryProfiler(t.Context(), 2, query, profileTestPlugin(t), profiler, metrics, queue.enqueue)
	t.Cleanup(worker.stop)
	require.True(t, worker.enqueue(profileTestRequest()), "enqueue the captured query identity")

	// Keep the replacement's metrics out of the original result's profile.
	require.Eventually(t, func() bool { return metrics.stale.Get() == 1 }, time.Second, time.Millisecond, "the schedule row must still describe the captured SQL")
	assert.Empty(t, queue, "replacement query metrics must not be attached to the old result response ID")
}

func TestProfilingDiagnosticsRespectRunCancellation(t *testing.T) {
	// The request outlives the daemon run; only run cancellation can unblock it.
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	entered := make(chan struct{})
	query := profilingQueryExecutor{runCtx: ctx, executor: profileQueryFunc(func(ctx context.Context, _ string, _ time.Duration) ([]map[string]any, error) {
		close(entered)
		<-ctx.Done()
		return nil, ctx.Err()
	})}

	// Start a diagnostic query with a long deadline and wait until it is in flight.
	done := make(chan error, 1)
	go func() {
		_, err := query.Query(context.Background(), osqueryScheduleProfilesDiagnosticsQuery, time.Minute)
		done <- err
	}()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("diagnostic query did not begin")
	}

	// Daemon teardown must propagate into the already-running diagnostic request.
	cancel()
	select {
	case err := <-done:
		assert.ErrorIs(t, err, context.Canceled, "run teardown should cancel a diagnostic regardless of its request deadline")
	case <-time.After(time.Second):
		t.Fatal("the diagnostic held the isolated client alive after run cancellation")
	}
}

type scheduledProfileResponse struct {
	scheduleID, responseID string
	count                  int
	executionCount         int64
}

// profileCallbackPublisher records the responses and profiles the scheduled
// worker publishes.
type profileCallbackPublisher struct {
	mockBeatPublisher
	responses chan scheduledProfileResponse
	profiles  chan collectedProfile
}

func (p *profileCallbackPublisher) PublishScheduledResponse(scheduleID, _, _, _, _, responseID string, _, _, _ time.Time, count int, executionCount int64) {
	p.responses <- scheduledProfileResponse{scheduleID: scheduleID, responseID: responseID, count: count, executionCount: executionCount}
}

func (p *profileCallbackPublisher) PublishScheduledProfile(_, name, scheduleID, responseID, spaceID string, executionCount int64, profile map[string]any) {
	p.profiles <- collectedProfile{name, scheduleID, responseID, spaceID, executionCount, profile}
}

func TestScheduledProfileAdmission(t *testing.T) {
	disabled, enabled := false, true
	tests := []struct {
		name             string
		applied, pending []config.InputConfig // pending is nil when the applied policy is current
		wantProfile      bool
	}{
		{
			name:        "pending policy enables profiling",
			applied:     profileTestInputs("default", "SELECT 1", &disabled),
			pending:     profileTestInputs("default", "SELECT 1", &enabled),
			wantProfile: true,
		},
		{
			// Without a profile stream the profile RPC would be wasted.
			name:    "no profile stream",
			applied: profileTestInputs("default", "SELECT 1", &enabled)[:1],
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Apply the policy, then stage the pending one before osqueryd refreshes.
			log := logptest.NewTestingLogger(t, "profile-admission")
			plugin := NewConfigPlugin(log)
			require.NoError(t, plugin.Set(test.applied), "stage the applied policy")
			_, err := plugin.GenerateConfig(t.Context())
			require.NoError(t, err, "apply the native schedule")
			policy := test.applied
			if test.pending != nil {
				require.NoError(t, plugin.Set(test.pending), "stage the pending policy")
				policy = test.pending
			}

			// Wire the logger path to a collector that reports its profile RPC.
			publisher := &profileCallbackPublisher{responses: make(chan scheduledProfileResponse, 1), profiles: make(chan collectedProfile, 1)}
			bt := &osquerybeat{log: log}
			d := newScheduledResultDispatcher(monitoring.NewRegistry(), log, config.DefaultScheduledMaxEvents,
				func(func() bool, func(), func(int)) osquerybeatPublisher { return publisher }, bt.handleScheduledQueryResult)
			d.profile = bt.enqueueScheduledProfile
			require.NoError(t, d.configure(policy, config.ScheduledPublishingConfig{MaxEvents: config.DefaultScheduledMaxEvents}), "configure the scheduled clients")
			stopOnCleanup(t, d)
			collecting := make(chan struct{}, 1)
			query := profileQueryFunc(func(context.Context, string, time.Duration) ([]map[string]any, error) {
				collecting <- struct{}{}
				return profileTestRow("SELECT 1"), nil
			})
			metrics := newScheduledProfileMetrics(monitoring.NewRegistry())
			worker := newScheduledQueryProfiler(t.Context(), 2, query, plugin, newQueryProfiler(log), metrics, d.enqueueProfile)

			// Stopping the worker either collects or discards an admitted profile.
			d.enqueue(t.Context(), nil, plugin, QueryResult{Name: "scheduled", Action: "diff", UnixTime: 123}, worker)
			worker.stop()
			admitted := len(collecting) == 1 || metrics.shutdown.Get() == 1
			assert.Equal(t, test.wantProfile, admitted, "whether the result's profile is collected")
		})
	}
}

func TestLoggerReturnsWhileScheduledProfileRPCWaits(t *testing.T) {
	// Keep the profile RPC blocked until the result callback has returned.
	entered := make(chan struct{})
	release := make(chan struct{})
	query := profileQueryFunc(func(ctx context.Context, _ string, _ time.Duration) ([]map[string]any, error) {
		close(entered)
		select {
		case <-release:
			rows := profileTestRow("SELECT 1")
			rows[0]["executions"] = int64(5)
			return rows, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	})

	// Configure a native schedule with an explicit policy identity and start time.
	log := logptest.NewTestingLogger(t, "callback")
	plugin := NewConfigPlugin(log)
	inputs := profileTestInputs("default", "SELECT 1", nil)
	scheduled := inputs[0].Osquery.Schedule["scheduled"]
	scheduled.ScheduleID = "policy-schedule"
	scheduled.StartDate = time.Unix(120, 0).UTC().Format(time.RFC3339)
	inputs[0].Osquery.Schedule["scheduled"] = scheduled
	require.NoError(t, plugin.Set(inputs), "configure a schedule with explicit execution timing and identity")
	_, err := plugin.GenerateConfig(t.Context())
	require.NoError(t, err, "apply the timed native schedule")

	// Wire the real logger callback to a worker and a fake publisher.
	publisher := &profileCallbackPublisher{responses: make(chan scheduledProfileResponse, 1), profiles: make(chan collectedProfile, 1)}
	bt := &osquerybeat{log: log}
	dispatcher := newScheduledResultDispatcher(monitoring.NewRegistry(), log, config.DefaultScheduledMaxEvents,
		func(func() bool, func(), func(int)) osquerybeatPublisher { return publisher }, bt.handleScheduledQueryResult)
	dispatcher.profile = bt.enqueueScheduledProfile
	require.NoError(t, dispatcher.configure(inputs, config.ScheduledPublishingConfig{MaxEvents: config.DefaultScheduledMaxEvents}), "configure the scheduled result handoff")
	t.Cleanup(func() {
		dispatcher.stop()
		select {
		case <-dispatcher.done:
		case <-time.After(time.Second):
			t.Error("scheduled result worker did not stop")
		}
	})
	worker := newScheduledQueryProfiler(t.Context(), 2, query, plugin, newQueryProfiler(log), newScheduledProfileMetrics(monitoring.NewRegistry()), dispatcher.enqueueProfile)
	t.Cleanup(worker.stop)
	loggerPlugin := NewLoggerPlugin(log, func(result QueryResult) {
		// A zero-hit diff needs no column resolution, so nil cli catches any
		// accidental reintroduction of profiling RPCs through the main client.
		dispatcher.enqueue(t.Context(), nil, plugin, result, worker)
	})

	// Exercise the real logger callback while the profile RPC stays blocked.
	done := make(chan error, 1)
	go func() {
		done <- loggerPlugin.Log(t.Context(), logger.LogTypeString, `{"name":"scheduled","action":"diff","unixTime":123}`)
	}()
	select {
	case err := <-done:
		require.NoError(t, err, "the logger callback should complete without waiting for profile collection")
	case <-time.After(time.Second):
		t.Fatal("the logger callback waited for its profile RPC")
	}

	// Profile collection must start only after the callback returned.
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("deferred profile collection did not start")
	}

	// Result completion must not wait for profiling, and both retain the same response ID.
	var response scheduledProfileResponse
	select {
	case response = <-publisher.responses:
	case <-time.After(time.Second):
		t.Fatal("the result callback did not publish its scheduled response")
	}
	assert.Equal(t, "policy-schedule", response.scheduleID, "the scheduled response must use policy identity")
	assert.Equal(t, int64(4), response.executionCount, "capture the result's execution slot before delayed collection")
	assert.Empty(t, publisher.profiles, "the scheduled response should be emitted while profiling is still pending")

	// Release the RPC and check the profile carries the result's correlation.
	close(release)
	select {
	case profile := <-publisher.profiles:
		assert.Equal(t, response.responseID, profile.responseID, "asynchronous profiling must retain the completed result's response ID")
		assert.Equal(t, response.scheduleID, profile.scheduleID, "retain the originating policy schedule identity")
		assert.Equal(t, response.executionCount, profile.scheduleExecutionCount, "retain execution attribution captured by the logger callback")
		assert.Equal(t, int64(5), profile.profile["executions"], "native metrics remain a latest snapshot, independent of the originating execution slot")
	case <-time.After(time.Second):
		t.Fatal("the released profile RPC did not publish its profile")
	}
}
