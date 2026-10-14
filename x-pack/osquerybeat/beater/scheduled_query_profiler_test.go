// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/sync/semaphore"

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
	name, responseID, spaceID string
	profile                   map[string]any
}

type profileTestSink struct {
	retired   atomic.Bool
	published chan collectedProfile
}

func (s *profileTestSink) Valid() bool { return !s.retired.Load() }
func (s *profileTestSink) Publish(name, responseID, spaceID string, profile map[string]any) bool {
	if !s.Valid() {
		return false
	}
	s.published <- collectedProfile{name, responseID, spaceID, profile}
	return true
}

func profileTestInputs(namespace, sql string, profiling *bool) []config.InputConfig {
	return []config.InputConfig{{
		Datastream: config.DatastreamConfig{Dataset: config.DefaultDataset, Namespace: namespace},
		Osquery: &config.OsqueryConfig{Schedule: map[string]config.Query{
			"scheduled": {Query: sql, CommonScheduleConfig: config.CommonScheduleConfig{SpaceID: "space"}, NativeSchedule: config.NativeSchedule{Interval: 1}, Profiling: profiling},
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
	return scheduledProfileRequest{name: "scheduled", query: "SELECT 1", namespace: "default", responseID: "response", spaceID: "space", sink: sink}
}

func profileTestRow(sql string) []map[string]any {
	return []map[string]any{{"query": sql, "executions": int64(2), "last_wall_time_ms": int64(10), "last_user_time": int64(2), "last_system_time": int64(3), "last_memory": int64(100)}}
}

func TestScheduledProfilerBoundsJobsAndJoinsCancellation(t *testing.T) {
	entered := make(chan struct{})
	query := profileQueryFunc(func(ctx context.Context, _ string, _ time.Duration) ([]map[string]any, error) {
		close(entered)
		<-ctx.Done()
		return nil, ctx.Err()
	})
	metrics := newScheduledProfileMetrics(monitoring.NewRegistry())
	worker := newScheduledQueryProfiler(t.Context(), 2, query, profileTestPlugin(t), newQueryProfiler(logptest.NewTestingLogger(t, "profile")), metrics)
	t.Cleanup(worker.stop)
	sink := &profileTestSink{published: make(chan collectedProfile, 3)}
	request := profileTestRequest(sink)
	require.True(t, worker.enqueue(request), "the first profile should reach the owned worker")
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("the profile worker did not start its RPC")
	}
	for range 2 {
		assert.True(t, worker.enqueue(request), "the bounded pending queue should accept its configured jobs")
	}
	assert.False(t, worker.enqueue(request), "a full profiling queue must reject work without blocking the logger")
	assert.Equal(t, uint64(1), metrics.queueFull.Get(), "queue overflow should be observable")
	done := make(chan struct{})
	go func() { worker.stop(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("shutdown did not cancel and join the profiling RPC")
	}
	assert.Empty(t, worker.jobs, "shutdown should release queued metadata references")
	assert.Equal(t, uint64(3), metrics.shutdown.Get(), "shutdown should count the active and queued profiles it discarded")
	assert.False(t, worker.enqueue(request), "the stopped worker must reject later callbacks")
	assert.Empty(t, sink.published, "cancelled collection must not publish a profile")
}

func TestScheduledProfilerPreservesProfileCorrelation(t *testing.T) {
	query := profileQueryFunc(func(context.Context, string, time.Duration) ([]map[string]any, error) {
		return profileTestRow("SELECT 1"), nil
	})
	metrics := newScheduledProfileMetrics(monitoring.NewRegistry())
	worker := newScheduledQueryProfiler(t.Context(), 2, query, profileTestPlugin(t), newQueryProfiler(logptest.NewTestingLogger(t, "profile")), metrics)
	t.Cleanup(worker.stop)
	sink := &profileTestSink{published: make(chan collectedProfile, 1)}
	require.True(t, worker.enqueue(profileTestRequest(sink)), "default profiling should enqueue a scheduled profile")
	select {
	case published := <-sink.published:
		assert.Equal(t, "scheduled", published.name, "preserve the native query name")
		assert.Equal(t, "response", published.responseID, "preserve correlation with the result and scheduled response")
		assert.Equal(t, "space", published.spaceID, "preserve the captured policy space")
		assert.Equal(t, "scheduled", published.profile["source"], "preserve the scheduled profile schema")
		assert.Equal(t, int64(2), published.profile["executions"], "preserve the native schedule's execution metrics")
	case <-time.After(time.Second):
		t.Fatal("the worker did not publish its collected profile")
	}
	require.Eventually(t, func() bool { return metrics.collected.Get() == 1 }, time.Second, time.Millisecond, "completed collection should be counted separately from output admission")
}

func TestScheduledProfilerRejectsPolicyChangesDuringCollection(t *testing.T) {
	disabled := false
	for _, change := range []struct {
		name   string
		inputs []config.InputConfig
	}{
		{"namespace changed", profileTestInputs("new", "SELECT 1", nil)},
		{"query replaced", profileTestInputs("default", "SELECT 2", nil)},
		{"profiling disabled", profileTestInputs("default", "SELECT 1", &disabled)},
		{"inputs removed", nil},
	} {
		t.Run(change.name, func(t *testing.T) {
			entered, release := make(chan struct{}), make(chan struct{})
			query := profileQueryFunc(func(ctx context.Context, _ string, _ time.Duration) ([]map[string]any, error) {
				close(entered)
				select {
				case <-release:
					return profileTestRow("SELECT 1"), nil
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			})
			plugin := profileTestPlugin(t)
			metrics := newScheduledProfileMetrics(monitoring.NewRegistry())
			worker := newScheduledQueryProfiler(t.Context(), 2, query, plugin, newQueryProfiler(logptest.NewTestingLogger(t, "profile")), metrics)
			t.Cleanup(worker.stop)
			sink := &profileTestSink{published: make(chan collectedProfile, 1)}
			require.True(t, worker.enqueue(profileTestRequest(sink)), "enqueue the original policy's profile")
			select {
			case <-entered:
			case <-time.After(time.Second):
				t.Fatal("profile collection did not begin")
			}
			require.NoError(t, plugin.Set(change.inputs), "install the pending policy before native config refresh")
			close(release)
			require.Eventually(t, func() bool { return metrics.stale.Get() == 1 }, time.Second, time.Millisecond, "collection must reject a policy change before publication")
			assert.Empty(t, sink.published, "old profile metadata must not use the new policy destination")
		})
	}
}

func TestScheduledProfilerRejectsReplacementNativeMetrics(t *testing.T) {
	query := profileQueryFunc(func(context.Context, string, time.Duration) ([]map[string]any, error) {
		return profileTestRow("SELECT 2"), nil
	})
	metrics := newScheduledProfileMetrics(monitoring.NewRegistry())
	worker := newScheduledQueryProfiler(t.Context(), 2, query, profileTestPlugin(t), newQueryProfiler(logptest.NewTestingLogger(t, "profile")), metrics)
	t.Cleanup(worker.stop)
	sink := &profileTestSink{published: make(chan collectedProfile, 1)}
	require.True(t, worker.enqueue(profileTestRequest(sink)), "enqueue the captured query identity")
	require.Eventually(t, func() bool { return metrics.stale.Get() == 1 }, time.Second, time.Millisecond, "the schedule row must still describe the captured SQL")
	assert.Empty(t, sink.published, "replacement query metrics must not be attached to the old result response ID")
}

func TestProfilingDiagnosticsRespectRunCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	entered := make(chan struct{})
	query := profilingQueryExecutor{runCtx: ctx, executor: profileQueryFunc(func(ctx context.Context, _ string, _ time.Duration) ([]map[string]any, error) {
		close(entered)
		<-ctx.Done()
		return nil, ctx.Err()
	})}
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
	cancel()
	select {
	case err := <-done:
		assert.ErrorIs(t, err, context.Canceled, "run teardown should cancel a diagnostic regardless of its request deadline")
	case <-time.After(time.Second):
		t.Fatal("the diagnostic held the isolated client alive after run cancellation")
	}
}

type profileCallbackPublisher struct {
	mockBeatPublisher
	sink      pub.ScheduledProfileSink
	responses chan string
}

func (p *profileCallbackPublisher) CaptureScheduledProfileSink(string) pub.ScheduledProfileSink {
	return p.sink
}

func (p *profileCallbackPublisher) PublishScheduledResponse(_, _, _, _, _, responseID string, _, _, _ time.Time, _ int, _ int64) {
	p.responses <- responseID
}

func TestLoggerReturnsWhileScheduledProfileRPCWaits(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	query := profileQueryFunc(func(ctx context.Context, _ string, _ time.Duration) ([]map[string]any, error) {
		close(entered)
		select {
		case <-release:
			return profileTestRow("SELECT 1"), nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	})
	plugin := profileTestPlugin(t)
	log := logptest.NewTestingLogger(t, "callback")
	worker := newScheduledQueryProfiler(t.Context(), 2, query, plugin, newQueryProfiler(log), newScheduledProfileMetrics(monitoring.NewRegistry()))
	t.Cleanup(worker.stop)
	sink := &profileTestSink{published: make(chan collectedProfile, 1)}
	publisher := &profileCallbackPublisher{sink: sink, responses: make(chan string, 1)}
	bt := &osquerybeat{pub: publisher, qp: newQueryProfiler(log), log: log}
	loggerPlugin := NewLoggerPlugin(log, func(result QueryResult) {
		// A zero-hit diff needs no column resolution, so nil cli catches any
		// accidental reintroduction of profiling RPCs through the main client.
		bt.handleQueryResult(t.Context(), nil, plugin, worker, result)
	})
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
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("deferred profile collection did not start")
	}
	responseID := <-publisher.responses
	assert.Empty(t, sink.published, "the scheduled response should be emitted while profiling is still pending")
	close(release)
	select {
	case profile := <-sink.published:
		assert.Equal(t, responseID, profile.responseID, "asynchronous profiling must retain the completed result's response ID")
	case <-time.After(time.Second):
		t.Fatal("the released profile RPC did not publish its profile")
	}
}

func TestProfilingConnectionIsLazyAndRetriesFailure(t *testing.T) {
	connectFailure := errors.New("profile socket unavailable")
	var attempts int
	connection := &profilingConnection{gate: semaphore.NewWeighted(1), connect: func(context.Context) error {
		attempts++
		if attempts == 1 {
			return connectFailure
		}
		return nil
	}}
	var queries int
	executor := profilingQueryExecutor{runCtx: t.Context(), connection: connection, executor: profileQueryFunc(func(context.Context, string, time.Duration) ([]map[string]any, error) {
		queries++
		return profileTestRow("SELECT 1"), nil
	})}
	assert.Zero(t, attempts, "disabled profiling or a missing stream must not open a socket during setup")
	_, err := executor.Query(t.Context(), "SELECT 1", time.Second)
	assert.ErrorIs(t, err, connectFailure, "a connection failure belongs to this optional profile request")
	assert.Zero(t, queries, "a failed connection must not start its query")
	_, err = executor.Query(t.Context(), "SELECT 1", time.Second)
	require.NoError(t, err, "a later profile should retry a failed connection")
	_, err = executor.Query(t.Context(), "SELECT 1", time.Second)
	require.NoError(t, err, "the connected executor should serve later diagnostics and profile requests")
	assert.Equal(t, 2, attempts, "successful connection should be shared for the remainder of this daemon run")
	assert.Equal(t, 2, queries, "both requests after connection should reach the isolated executor")
}

func TestProfilingConnectionWaitCanBeCancelled(t *testing.T) {
	entered := make(chan struct{})
	var attempts atomic.Int32
	connection := &profilingConnection{gate: semaphore.NewWeighted(1), connect: func(ctx context.Context) error {
		attempts.Add(1)
		close(entered)
		<-ctx.Done()
		return ctx.Err()
	}}
	activeCtx, cancelActive := context.WithCancel(t.Context())
	defer cancelActive()
	activeDone := make(chan error, 1)
	go func() { activeDone <- connection.ensure(activeCtx) }()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("the first lazy connection did not start")
	}
	waitingCtx, cancelWaiting := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancelWaiting()
	waitingDone := make(chan error, 1)
	go func() { waitingDone <- connection.ensure(waitingCtx) }()
	select {
	case err := <-waitingDone:
		assert.ErrorIs(t, err, context.DeadlineExceeded, "queued diagnostics must not wait past their own deadline")
	case <-time.After(time.Second):
		t.Fatal("the queued connection ignored cancellation")
	}
	assert.Equal(t, int32(1), attempts.Load(), "cancelled waiters must not start a competing connection")
	cancelActive()
	select {
	case err := <-activeDone:
		assert.ErrorIs(t, err, context.Canceled, "daemon cancellation must reach the active connection")
	case <-time.After(time.Second):
		t.Fatal("active profile connection did not exit after cancellation")
	}
	assert.False(t, connection.connected.Load(), "cancelled initialization must remain retryable")
}
