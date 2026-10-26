// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/osquery/osquery-go/plugin/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/pub"
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

type profileTestSink struct {
	retired   atomic.Bool
	published chan collectedProfile
	// drop makes Publish report a full output queue. retireOnDrop also
	// retires the lease, as a policy change during publication does.
	drop, retireOnDrop bool
}

func (s *profileTestSink) Valid() bool { return !s.retired.Load() }
func (s *profileTestSink) Publish(name, scheduleID, responseID, spaceID string, scheduleExecutionCount int64, profile map[string]any) bool {
	if !s.Valid() {
		return false
	}
	if s.drop {
		if s.retireOnDrop {
			s.retired.Store(true)
		}
		return false
	}
	s.published <- collectedProfile{name, scheduleID, responseID, spaceID, scheduleExecutionCount, profile}
	return true
}

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

func profileTestRequest(sink *profileTestSink) scheduledProfileRequest {
	return scheduledProfileRequest{
		name:                   "scheduled",
		query:                  "SELECT 1",
		namespace:              "default",
		scheduleID:             "scheduled",
		responseID:             "response",
		spaceID:                "space",
		scheduleExecutionCount: 1,
		sink:                   sink,
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
	worker := newScheduledQueryProfiler(t.Context(), 2, query, profileTestPlugin(t), profiler, metrics)
	t.Cleanup(func() {
		// Release the fake RPC even if a regression blocks admission or shutdown.
		close(release)
		worker.stop()
	})
	sink := &profileTestSink{published: make(chan collectedProfile, 4)}
	request := profileTestRequest(sink)

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

	// After shutdown nothing may remain queued, be admitted, or be published.
	assert.Empty(t, worker.jobs, "shutdown should release queued metadata references")
	assert.Equal(t, uint64(3), metrics.shutdown.Get(), "shutdown should count the active and queued profiles it discarded")
	assert.False(t, worker.enqueue(request), "the stopped worker must reject later callbacks")
	assert.Empty(t, sink.published, "cancelled collection must not publish a profile")
}

// stuckProfileSink is a live lease whose Publish waits until release is closed,
// as a memory queue publish waits for a slot.
type stuckProfileSink struct{ entered, release chan struct{} }

func (*stuckProfileSink) Valid() bool { return true }
func (s *stuckProfileSink) Publish(string, string, string, string, int64, map[string]any) bool {
	close(s.entered)
	<-s.release
	return false
}

func TestScheduledProfilerStopDoesNotJoinStuckPublish(t *testing.T) {
	// Run a worker whose only profile publish gets stuck.
	query := profileQueryFunc(func(context.Context, string, time.Duration) ([]map[string]any, error) {
		return profileTestRow("SELECT 1"), nil
	})
	metrics := newScheduledProfileMetrics(monitoring.NewRegistry())
	profiler := newQueryProfiler(logptest.NewTestingLogger(t, "profile"))
	worker := newScheduledQueryProfiler(t.Context(), 2, query, profileTestPlugin(t), profiler, metrics)
	sink := &stuckProfileSink{entered: make(chan struct{}), release: make(chan struct{})}
	t.Cleanup(func() {
		// Release the publish even if a regression makes stop wait for it.
		close(sink.release)
		worker.stop()
	})
	request := profileTestRequest(nil)
	request.sink = sink
	require.True(t, worker.enqueue(request), "enqueue a profile whose publish will get stuck")
	select {
	case <-sink.entered:
	case <-time.After(time.Second):
		t.Fatal("the worker did not reach Publish")
	}

	// Stopping must not wait for the stuck publish.
	done := make(chan struct{})
	go func() {
		worker.stop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("stop waited for a collector stuck in Publish")
	}
}

func TestScheduledProfilerPreservesProfileCorrelation(t *testing.T) {
	// Return metrics from a later execution than the result that queued this profile.
	query := profileQueryFunc(func(context.Context, string, time.Duration) ([]map[string]any, error) {
		return profileTestRow("SELECT 1"), nil
	})

	// Run a worker against that query and capture what it publishes.
	metrics := newScheduledProfileMetrics(monitoring.NewRegistry())
	profiler := newQueryProfiler(logptest.NewTestingLogger(t, "profile"))
	worker := newScheduledQueryProfiler(t.Context(), 2, query, profileTestPlugin(t), profiler, metrics)
	t.Cleanup(worker.stop)
	sink := &profileTestSink{published: make(chan collectedProfile, 1)}

	// The worker must preserve captured result metadata while collecting current metrics.
	require.True(t, worker.enqueue(profileTestRequest(sink)), "default profiling should enqueue a scheduled profile")
	select {
	case published := <-sink.published:
		assert.Equal(t, "scheduled", published.name, "preserve the native query name")
		assert.Equal(t, "scheduled", published.scheduleID, "preserve the schedule identity")
		assert.Equal(t, "response", published.responseID, "preserve correlation with the result and scheduled response")
		assert.Equal(t, int64(1), published.scheduleExecutionCount, "preserve the originating schedule execution")
		assert.Equal(t, "space", published.spaceID, "preserve the captured policy space")
		assert.Equal(t, "scheduled", published.profile["source"], "preserve the scheduled profile schema")
		assert.Equal(t, int64(2), published.profile["executions"], "preserve the native schedule's execution metrics")
	case <-time.After(time.Second):
		t.Fatal("the worker did not publish its collected profile")
	}

	// Collection is counted by the worker, separately from output admission.
	require.Eventually(t, func() bool { return metrics.collected.Get() == 1 }, time.Second, time.Millisecond, "completed collection should be counted separately from output admission")
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
			worker := newScheduledQueryProfiler(t.Context(), 2, query, plugin, profiler, metrics)
			t.Cleanup(worker.stop)
			sink := &profileTestSink{published: make(chan collectedProfile, 1)}

			// Enqueue a profile and wait until collection is in flight.
			require.True(t, worker.enqueue(profileTestRequest(sink)), "enqueue the original policy's profile")
			select {
			case <-entered:
			case <-time.After(time.Second):
				t.Fatal("profile collection did not begin")
			}

			// A pending policy must invalidate old work even before native config refresh.
			require.NoError(t, plugin.Set(change.inputs), "install the pending policy before native config refresh")
			close(release)

			require.Eventually(t, func() bool { return metrics.stale.Get() == 1 }, time.Second, time.Millisecond, "collection must reject a policy change before publication")
			assert.Empty(t, sink.published, "old profile metadata must not use the new policy destination")
		})
	}
}

func TestScheduledProfilerCountsRejectedPublish(t *testing.T) {
	for _, tc := range []struct {
		name      string
		retire    bool
		wantStale uint64
	}{
		// The sink's output_full counter records this drop, not the worker.
		{"output full", false, 0},
		{"lease retired during publish", true, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Run a worker whose first profile is rejected by its sink.
			query := profileQueryFunc(func(context.Context, string, time.Duration) ([]map[string]any, error) {
				return profileTestRow("SELECT 1"), nil
			})
			metrics := newScheduledProfileMetrics(monitoring.NewRegistry())
			profiler := newQueryProfiler(logptest.NewTestingLogger(t, "profile"))
			worker := newScheduledQueryProfiler(t.Context(), 2, query, profileTestPlugin(t), profiler, metrics)
			t.Cleanup(worker.stop)
			rejecting := &profileTestSink{published: make(chan collectedProfile, 1), drop: true, retireOnDrop: tc.retire}
			require.True(t, worker.enqueue(profileTestRequest(rejecting)), "enqueue a profile its sink will reject")

			// The worker is serial, so a later publication shows the first collection finished.
			next := &profileTestSink{published: make(chan collectedProfile, 1)}
			require.True(t, worker.enqueue(profileTestRequest(next)), "enqueue a profile that publishes")
			select {
			case <-next.published:
			case <-time.After(time.Second):
				t.Fatal("the worker did not publish the later profile")
			}
			require.Eventually(t, func() bool { return metrics.collected.Get() == 1 }, time.Second, time.Millisecond, "only the published profile counts as collected")
			assert.Equal(t, tc.wantStale, metrics.stale.Get(), "a rejected publish is stale only when the lease retired")
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
	worker := newScheduledQueryProfiler(t.Context(), 2, query, profileTestPlugin(t), profiler, metrics)
	t.Cleanup(worker.stop)
	sink := &profileTestSink{published: make(chan collectedProfile, 1)}
	require.True(t, worker.enqueue(profileTestRequest(sink)), "enqueue the captured query identity")

	// Keep the replacement's metrics out of the original result's profile.
	require.Eventually(t, func() bool { return metrics.stale.Get() == 1 }, time.Second, time.Millisecond, "the schedule row must still describe the captured SQL")
	assert.Empty(t, sink.published, "replacement query metrics must not be attached to the old result response ID")
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
	executionCount         int64
}

type profileCallbackPublisher struct {
	mockBeatPublisher
	sink      pub.ScheduledProfileSink
	responses chan scheduledProfileResponse
}

func (p *profileCallbackPublisher) CaptureScheduledProfileSink(string) pub.ScheduledProfileSink {
	return p.sink
}

func (p *profileCallbackPublisher) PublishScheduledResponse(scheduleID, _, _, _, _, responseID string, _, _, _ time.Time, _ int, executionCount int64) {
	p.responses <- scheduledProfileResponse{scheduleID: scheduleID, responseID: responseID, executionCount: executionCount}
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

	// Wire the real logger callback to a worker and fake publishers.
	profiler := newQueryProfiler(log)
	metrics := newScheduledProfileMetrics(monitoring.NewRegistry())
	worker := newScheduledQueryProfiler(t.Context(), 2, query, plugin, profiler, metrics)
	t.Cleanup(worker.stop)
	sink := &profileTestSink{published: make(chan collectedProfile, 1)}
	publisher := &profileCallbackPublisher{sink: sink, responses: make(chan scheduledProfileResponse, 1)}
	bt := &osquerybeat{pub: publisher, qp: profiler, log: log}
	dispatcher := newScheduledResultDispatcher(monitoring.NewRegistry(), log, config.DefaultScheduledMaxEvents,
		func(func() bool, func()) osquerybeatPublisher { return publisher }, bt.handleScheduledQueryResult)
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
	assert.Empty(t, sink.published, "the scheduled response should be emitted while profiling is still pending")

	// Release the RPC and check the profile carries the result's correlation.
	close(release)
	select {
	case profile := <-sink.published:
		assert.Equal(t, response.responseID, profile.responseID, "asynchronous profiling must retain the completed result's response ID")
		assert.Equal(t, response.scheduleID, profile.scheduleID, "retain the originating policy schedule identity")
		assert.Equal(t, response.executionCount, profile.scheduleExecutionCount, "retain execution attribution captured by the logger callback")
		assert.Equal(t, int64(5), profile.profile["executions"], "native metrics remain a latest snapshot, independent of the originating execution slot")
	case <-time.After(time.Second):
		t.Fatal("the released profile RPC did not publish its profile")
	}
}
