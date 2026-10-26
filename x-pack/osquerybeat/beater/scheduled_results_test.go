// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/osquery/osquery-go/plugin/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

type stalledResultPublisher struct {
	mockBeatPublisher
	entered chan struct{}
	release chan struct{}
	calls   atomic.Int32
	closed  atomic.Int32
	once    sync.Once
}

func (p *stalledResultPublisher) PublishScheduledResponse(string, string, string, string, string, string, time.Time, time.Time, time.Time, int, int64) {
	p.calls.Add(1)
	p.once.Do(func() { close(p.entered) })
	<-p.release
}
func (p *stalledResultPublisher) Close() { p.closed.Add(1) }

func outageInputs(namespace string) []config.InputConfig {
	profile := false
	return []config.InputConfig{{
		Datastream: config.DatastreamConfig{Dataset: config.DefaultDataset, Namespace: namespace},
		Osquery: &config.OsqueryConfig{Schedule: map[string]config.Query{
			"scheduled": {Query: "SELECT 1", NativeSchedule: config.NativeSchedule{Interval: 1}, Profiling: &profile},
		}},
	}}
}

// outageMaxEvents is the outage dispatcher's queue limit in rows.
const outageMaxEvents = 8

var outageLimits = config.ScheduledPublishingConfig{MaxEvents: outageMaxEvents}

// outageDispatcher returns a dispatcher whose first policy's publisher blocks
// until release is called. Publishers for later policies do not block.
func outageDispatcher(t *testing.T, limits config.ScheduledPublishingConfig) (*scheduledResultDispatcher, *ConfigPlugin, *stalledResultPublisher, func()) {
	t.Helper()
	log := logptest.NewTestingLogger(t, "outage")
	bt := &osquerybeat{log: log}
	first := &stalledResultPublisher{entered: make(chan struct{}), release: make(chan struct{})}
	created := false
	d := newScheduledResultDispatcher(monitoring.NewRegistry(), log, outageMaxEvents, func(func() bool, func()) osquerybeatPublisher {
		if !created {
			created = true
			return first
		}
		next := &stalledResultPublisher{entered: make(chan struct{}), release: make(chan struct{})}
		close(next.release)
		return next
	}, bt.handleScheduledQueryResult)
	plugin := NewConfigPlugin(log)
	require.NoError(t, plugin.Set(outageInputs("old")), "validate the initial policy")
	_, err := plugin.GenerateConfig(t.Context())
	require.NoError(t, err, "apply the initial native policy")
	require.NoError(t, d.configure(outageInputs("old"), limits), "configure independent scheduled clients")
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(first.release) }) }
	t.Cleanup(func() {
		release()
		d.stop()
		select {
		case <-d.done:
		case <-time.After(time.Second):
			t.Error("released dispatcher did not finish")
		}
	})
	return d, plugin, first, release
}

func waitForResultPublication(t *testing.T, p *stalledResultPublisher) {
	t.Helper()
	select {
	case <-p.entered:
	case <-time.After(time.Second):
		t.Fatal("scheduled response did not reach the blocked output")
	}
}

func TestScheduledResultsBoundLoggerAdmissionAndShutdown(t *testing.T) {
	d, plugin, publisher, release := outageDispatcher(t, outageLimits)
	loggerPlugin := NewLoggerPlugin(logptest.NewTestingLogger(t, "logger"), func(res QueryResult) {
		d.enqueue(t.Context(), nil, plugin, res, nil)
	})
	payload := `{"name":"scheduled","action":"diff","unixTime":123}`
	require.NoError(t, loggerPlugin.Log(t.Context(), logger.LogTypeString, payload), "zero-hit logger call should return before its response is published")
	waitForResultPublication(t, publisher)

	// Keep the first response blocked, overfill the bounded backlog, and prove
	// logger admission and shutdown both finish before the output recovers.
	// Each zero-hit result is one row.
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		for range outageMaxEvents + 1 {
			if err := loggerPlugin.Log(t.Context(), logger.LogTypeString, payload); err != nil {
				t.Errorf("logger admission: %v", err)
				return
			}
		}
		d.stop()
	}()
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("logger or shutdown waited for blocked publication")
	}
	assert.Equal(t, uint64(1), d.metrics.overflow.Get(), "the oldest queued result must be dropped once the row limit is reached")
	assert.Equal(t, uint64(1), d.metrics.overflowRows.Get(), "count the dropped result's rows")
	assert.Equal(t, uint64(outageMaxEvents), d.metrics.shutdown.Get(), "shutdown should discard all pending results")
	assert.Zero(t, publisher.closed.Load(), "the active worker owns deferred client cleanup")
	release()
	select {
	case <-d.done:
	case <-time.After(time.Second):
		t.Fatal("worker did not finish after output recovery")
	}
	assert.Equal(t, int32(1), publisher.closed.Load(), "retired publishing clients must close exactly once")
	assert.Equal(t, int32(1), publisher.calls.Load(), "shutdown must not publish pending documents")
}

func TestScheduledResultsPolicyRetirementDoesNotWaitForOutput(t *testing.T) {
	d, plugin, publisher, _ := outageDispatcher(t, outageLimits)
	result := QueryResult{Name: "scheduled", Action: "diff"}
	d.enqueue(t.Context(), nil, plugin, result, nil)
	waitForResultPublication(t, publisher)
	d.enqueue(t.Context(), nil, plugin, result, nil)
	configured := make(chan error, 1)
	go func() { configured <- d.configure(outageInputs("new"), outageLimits) }()
	select {
	case err := <-configured:
		require.NoError(t, err, "replace policy while old publication waits")
	case <-time.After(time.Second):
		t.Fatal("policy replacement waited on old output")
	}
	d.enqueue(t.Context(), nil, plugin, result, nil)
	assert.Equal(t, uint64(1), d.metrics.stale.Get(), "old namespace must not enter replacement clients")
	assert.Equal(t, uint64(1), d.metrics.staleRows.Get(), "count the stale result's rows")
	items, _ := d.queue.stats()
	assert.Equal(t, 1, items, "policy replacement must keep the queued result for the worker to rebind or drop")
}

func TestScheduledResultsPolicyChangeKeepsBacklog(t *testing.T) {
	d, plugin, blocked, release := outageDispatcher(t, outageLimits)
	result := QueryResult{Name: "scheduled", Action: "diff"}
	d.enqueue(t.Context(), nil, plugin, result, nil)
	waitForResultPublication(t, blocked)
	for range 3 {
		d.enqueue(t.Context(), nil, plugin, result, nil)
	}
	require.NoError(t, d.configure(outageInputs("old"), outageLimits), "replace the policy with one keeping the query")
	next := activePublisher(d)
	release()
	assert.Eventually(t, func() bool { return next.calls.Load() == 3 }, time.Second, time.Millisecond,
		"queued results must publish through the new policy's clients")
	assert.Equal(t, int32(1), blocked.calls.Load(), "only the in-flight result may use the old clients")
	assert.Zero(t, d.metrics.stale.Get(), "an unchanged query must not be dropped as stale")
}

func TestScheduledResultsPolicyChangeDropsMovedQuery(t *testing.T) {
	d, plugin, blocked, release := outageDispatcher(t, outageLimits)
	result := QueryResult{Name: "scheduled", Action: "diff"}
	d.enqueue(t.Context(), nil, plugin, result, nil)
	waitForResultPublication(t, blocked)
	for range 3 {
		d.enqueue(t.Context(), nil, plugin, result, nil)
	}
	require.NoError(t, d.configure(outageInputs("other"), outageLimits), "move the query to another namespace")
	next := activePublisher(d)
	release()
	assert.Eventually(t, func() bool { return d.metrics.stale.Get() == 3 }, time.Second, time.Millisecond,
		"results for a moved query must be dropped as stale")
	assert.Equal(t, uint64(3), d.metrics.staleRows.Get(), "count the stale results' rows")
	assert.Zero(t, next.calls.Load(), "results captured for the old namespace must not use the new clients")
	assert.Equal(t, int32(1), blocked.calls.Load(), "only the in-flight result may use the old clients")
}

func TestScheduledResultsMaxAgeExpires(t *testing.T) {
	d, plugin, blocked, release := outageDispatcher(t, config.ScheduledPublishingConfig{MaxEvents: outageMaxEvents, MaxAge: 50 * time.Millisecond})
	result := QueryResult{Name: "scheduled", Action: "diff"}
	d.enqueue(t.Context(), nil, plugin, result, nil)
	waitForResultPublication(t, blocked)
	d.enqueue(t.Context(), nil, plugin, result, nil)
	d.enqueue(t.Context(), nil, plugin, result, nil)
	time.Sleep(100 * time.Millisecond)
	release()
	assert.Eventually(t, func() bool { return d.metrics.expired.Get() == 2 }, time.Second, time.Millisecond,
		"results queued longer than max_age must be dropped as expired")
	assert.Equal(t, uint64(2), d.metrics.expiredRows.Get(), "count the expired results' rows")
	assert.Equal(t, int32(1), blocked.calls.Load(), "expired results must not be published")
}

func TestScheduledResultsPolicyChangeDropsChangedSQL(t *testing.T) {
	d, plugin, blocked, release := outageDispatcher(t, outageLimits)
	result := QueryResult{Name: "scheduled", Action: "diff"}
	d.enqueue(t.Context(), nil, plugin, result, nil)
	waitForResultPublication(t, blocked)
	for range 2 {
		d.enqueue(t.Context(), nil, plugin, result, nil)
	}
	changed := outageInputs("old")
	query := changed[0].Osquery.Schedule["scheduled"]
	query.Query = "SELECT 2"
	changed[0].Osquery.Schedule["scheduled"] = query
	require.NoError(t, d.configure(changed, outageLimits), "change the query's SQL in the same namespace")
	next := activePublisher(d)
	d.enqueue(t.Context(), nil, plugin, result, nil)
	release()
	assert.Eventually(t, func() bool { return d.metrics.stale.Get() == 3 }, time.Second, time.Millisecond,
		"results captured for the old SQL must be dropped as stale at admission and when dequeued")
	assert.Equal(t, uint64(3), d.metrics.staleRows.Get(), "count the stale results' rows")
	assert.Zero(t, next.calls.Load(), "results for the old SQL must not use the new clients")
}

func TestScheduledResultsPolicyChangeKeepsCapturedQueryInfo(t *testing.T) {
	d, plugin, blocked, release := outageDispatcher(t, outageLimits)
	calls := recordHandledResults(d)
	result := QueryResult{Name: "scheduled", Action: "diff"}
	d.enqueue(t.Context(), nil, plugin, result, nil)
	waitForResultPublication(t, blocked)
	d.enqueue(t.Context(), nil, plugin, result, nil)
	mapped := outageInputs("old")
	query := mapped[0].Osquery.Schedule["scheduled"]
	query.ECSMapping = map[string]any{"user": map[string]any{"id": map[string]any{"field": "uid"}}}
	mapped[0].Osquery.Schedule["scheduled"] = query
	require.NoError(t, d.configure(mapped, outageLimits), "change only the query's ECS mapping")
	next := activePublisher(d)
	// The live plugin still has the old mapping; admission must accept it.
	d.enqueue(t.Context(), nil, plugin, result, nil)
	release()
	<-calls // The in-flight result on the old clients.
	for i := range 2 {
		select {
		case call := <-calls:
			assert.Same(t, next, call.publisher, "result %d must publish through the new policy's clients", i)
			assert.Empty(t, call.query.ECSMapping, "result %d must publish with the QueryInfo captured at admission", i)
		case <-time.After(time.Second):
			t.Fatalf("result %d was not published", i)
		}
	}
	assert.Zero(t, d.metrics.stale.Get(), "an ECS mapping change must not make results stale")
}

func TestScheduledResultsCancelledRunDroppedBeforeHandling(t *testing.T) {
	d, plugin, blocked, release := outageDispatcher(t, outageLimits)
	calls := recordHandledResults(d)
	rows := []map[string]string{{"value": "a"}, {"value": "b"}}
	d.enqueue(t.Context(), nil, plugin, QueryResult{Name: "scheduled", Action: "diff"}, nil)
	waitForResultPublication(t, blocked)
	ctx, cancel := context.WithCancel(t.Context())
	d.enqueue(ctx, nil, plugin, QueryResult{Name: "scheduled", Action: "snapshot", Hits: rows}, nil)
	cancel()
	release()
	assert.Eventually(t, func() bool { return d.metrics.stale.Get() == 1 }, time.Second, time.Millisecond,
		"a result of a cancelled daemon run must be dropped as stale")
	assert.Equal(t, uint64(len(rows)+1), d.metrics.staleRows.Get(), "count the cancelled result's rows and response")
	<-calls // The in-flight result.
	select {
	case <-calls:
		t.Error("a result of a cancelled daemon run must not reach the handler")
	default:
	}
}

func TestScheduledResultsInvalidPolicyKeepsLimits(t *testing.T) {
	log := logptest.NewTestingLogger(t, "invalid-policy")
	d := newScheduledResultDispatcher(monitoring.NewRegistry(), log, outageMaxEvents, func(func() bool, func()) osquerybeatPublisher {
		return &stalledResultPublisher{entered: make(chan struct{}), release: make(chan struct{})}
	}, (&osquerybeat{log: log}).handleScheduledQueryResult)
	t.Cleanup(d.stop)
	require.NoError(t, d.configure(outageInputs("old"), outageLimits), "apply the initial policy")

	// Feed policy updates to the dispatcher as the beater's main loop does.
	registry := reload.NewRegistry()
	updates := config.WatchInputs(t.Context(), log, registry)
	configured := make(chan error, 1)
	go func() {
		for {
			select {
			case <-t.Context().Done():
				return
			case inputs := <-updates:
				publishing, err := config.GetPublishingConfig(inputs)
				if err == nil {
					err = d.configure(inputs, publishing.Scheduled)
				}
				configured <- err
			}
		}
	}()
	policy := func(maxEvents int) []*reload.ConfigWithMeta {
		return []*reload.ConfigWithMeta{{InputUnitID: "osquery-unit", Config: conf.MustNewConfigFrom(map[string]any{
			"data_stream": map[string]any{"dataset": config.DefaultDataset, "namespace": "old"},
			"osquery": map[string]any{
				"schedule":        map[string]any{"scheduled": map[string]any{"query": "SELECT 1", "interval": 1}},
				"elastic_options": map[string]any{"publishing": map[string]any{"scheduled": map[string]any{"max_events": maxEvents}}},
			},
		})}}
	}

	assert.Error(t, registry.GetInputList().Reload(policy(-1)), "the policy with a negative max_events must be rejected")
	select {
	case err := <-configured:
		t.Errorf("a rejected policy reached the dispatcher (configure error: %v)", err)
	case <-time.After(50 * time.Millisecond):
	}
	assert.Equal(t, outageMaxEvents, queueLimit(d), "a rejected policy must leave the queue limit unchanged")

	require.NoError(t, registry.GetInputList().Reload(policy(5)), "a valid policy must be accepted")
	require.NoError(t, <-configured, "apply the valid policy")
	assert.Equal(t, 5, queueLimit(d), "a valid policy must apply its queue limit")
}

func queueLimit(d *scheduledResultDispatcher) int {
	d.queue.mu.Lock()
	defer d.queue.mu.Unlock()
	return d.queue.limit
}

type handledResult struct {
	publisher scheduledQueryPublisher
	query     QueryInfo
}

// recordHandledResults wraps d's handler to report each call. Call it before
// enqueueing; the queue orders the change before the worker reads it.
func recordHandledResults(d *scheduledResultDispatcher) <-chan handledResult {
	calls := make(chan handledResult, 8)
	handle := d.handle
	d.handle = func(ctx context.Context, cli scheduledQueryClient, publisher scheduledQueryPublisher, ns string, qi QueryInfo, res QueryResult, responseID string) {
		calls <- handledResult{publisher: publisher, query: qi}
		handle(ctx, cli, publisher, ns, qi, res, responseID)
	}
	return calls
}

// activePublisher returns the publisher of d's current policy.
func activePublisher(d *scheduledResultDispatcher) *stalledResultPublisher {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.lease.publisher.(*stalledResultPublisher)
}

func TestScheduledResultsReuseWorkerAcrossDaemonRestarts(t *testing.T) {
	d, plugin, publisher, _ := outageDispatcher(t, outageLimits)
	result := QueryResult{Name: "scheduled", Action: "diff"}
	ctx, cancel := context.WithCancel(t.Context())
	d.enqueue(ctx, nil, plugin, result, nil)
	waitForResultPublication(t, publisher)
	cancel()
	for range 20 {
		ctx, cancel := context.WithCancel(t.Context())
		d.enqueue(ctx, nil, plugin, result, nil)
		cancel()
		d.prune()
	}
	items, _ := d.queue.stats()
	assert.Zero(t, items, "cancelled runs must not accumulate behind the stuck publication")
	assert.Equal(t, int32(1), publisher.calls.Load(), "restarts must reuse the single publishing worker")
	assert.Equal(t, uint64(20), d.metrics.stale.Get(), "prune each cancelled run from the backlog")
}

func TestScheduledResultsStopReleasesIdleWorker(t *testing.T) {
	log := logptest.NewTestingLogger(t, "idle")
	d := newScheduledResultDispatcher(monitoring.NewRegistry(), log, outageMaxEvents, func(func() bool, func()) osquerybeatPublisher {
		return &stalledResultPublisher{entered: make(chan struct{}), release: make(chan struct{})}
	}, (&osquerybeat{log: log}).handleScheduledQueryResult)
	require.NoError(t, d.configure(outageInputs("idle"), outageLimits), "start the result worker")
	time.Sleep(10 * time.Millisecond) // Let the worker block on the empty queue.
	d.stop()
	select {
	case <-d.done:
	case <-time.After(time.Second):
		t.Fatal("stop did not release the worker waiting on an empty queue")
	}
}

// Result and profile publication remain separate. Capturing correlation when
// admitting the logger document must not make collection wait for its output.
type correlatedStalledResultPublisher struct {
	stalledResultPublisher
	response chan scheduledProfileResponse
}

func (p *correlatedStalledResultPublisher) PublishScheduledResponse(scheduleID, packID, packName, queryName, spaceID, responseID string, startedAt, completedAt, plannedAt time.Time, count int, executionCount int64) {
	p.response <- scheduledProfileResponse{scheduleID: scheduleID, responseID: responseID, executionCount: executionCount}
	p.stalledResultPublisher.PublishScheduledResponse(scheduleID, packID, packName, queryName, spaceID, responseID, startedAt, completedAt, plannedAt, count, executionCount)
}

func TestScheduledResultsStalledOutputDoesNotBlockProfileCollection(t *testing.T) {
	log := logptest.NewTestingLogger(t, "independent-workers")
	inputs := profileTestInputs("default", "SELECT 1", nil)
	query := inputs[0].Osquery.Schedule["scheduled"]
	query.ScheduleID = "policy-schedule"
	query.StartDate = time.Unix(120, 0).UTC().Format(time.RFC3339)
	inputs[0].Osquery.Schedule["scheduled"] = query
	plugin := NewConfigPlugin(log)
	require.NoError(t, plugin.Set(inputs), "stage the profiled policy")
	_, err := plugin.GenerateConfig(t.Context())
	require.NoError(t, err, "apply the profiled policy")

	profiler := newQueryProfiler(log)
	executor := profileQueryFunc(func(context.Context, string, time.Duration) ([]map[string]any, error) {
		return profileTestRow("SELECT 1"), nil
	})
	worker := newScheduledQueryProfiler(t.Context(), 2, executor, plugin, profiler, newScheduledProfileMetrics(monitoring.NewRegistry()))
	t.Cleanup(worker.stop)
	sink := &profileTestSink{published: make(chan collectedProfile, 1)}
	bt := &osquerybeat{pub: &profileCallbackPublisher{sink: sink}, qp: profiler, log: log}
	publisher := &correlatedStalledResultPublisher{
		stalledResultPublisher: stalledResultPublisher{entered: make(chan struct{}), release: make(chan struct{})},
		response:               make(chan scheduledProfileResponse, 1),
	}
	d := newScheduledResultDispatcher(monitoring.NewRegistry(), log, config.DefaultScheduledMaxEvents,
		func(func() bool, func()) osquerybeatPublisher { return publisher }, bt.handleScheduledQueryResult)
	d.profile = bt.enqueueScheduledProfile
	require.NoError(t, d.configure(inputs, config.ScheduledPublishingConfig{MaxEvents: config.DefaultScheduledMaxEvents}), "configure the independent result publisher")
	t.Cleanup(func() {
		close(publisher.release)
		d.stop()
		select {
		case <-d.done:
		case <-time.After(time.Second):
			t.Error("released result publisher did not stop")
		}
	})
	d.enqueue(t.Context(), nil, plugin, QueryResult{Name: "scheduled", Action: "diff", UnixTime: 123}, worker)
	waitForResultPublication(t, &publisher.stalledResultPublisher)
	var profile collectedProfile
	select {
	case profile = <-sink.published:
	case <-time.After(time.Second):
		t.Fatal("profile collection waited for the blocked result publisher")
	}
	response := <-publisher.response
	assert.Equal(t, response.responseID, profile.responseID, "independent workers must retain the same response identity")
	assert.Equal(t, "policy-schedule", profile.scheduleID, "capture policy identity before deferred work")
	assert.Equal(t, response.executionCount, profile.scheduleExecutionCount, "retain the originating execution slot")
}
