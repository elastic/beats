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

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
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

func outageDispatcher(t *testing.T) (*scheduledResultDispatcher, *ConfigPlugin, *stalledResultPublisher, func()) {
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
		return &stalledResultPublisher{entered: make(chan struct{}), release: make(chan struct{})}
	}, bt.handleScheduledQueryResult)
	plugin := NewConfigPlugin(log)
	require.NoError(t, plugin.Set(outageInputs("old")), "validate the initial policy")
	_, err := plugin.GenerateConfig(t.Context())
	require.NoError(t, err, "apply the initial native policy")
	require.NoError(t, d.configure(outageInputs("old")), "configure independent scheduled clients")
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
	d, plugin, publisher, release := outageDispatcher(t)
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
	d, plugin, publisher, _ := outageDispatcher(t)
	result := QueryResult{Name: "scheduled", Action: "diff"}
	d.enqueue(t.Context(), nil, plugin, result, nil)
	waitForResultPublication(t, publisher)
	d.enqueue(t.Context(), nil, plugin, result, nil)
	configured := make(chan error, 1)
	go func() { configured <- d.configure(outageInputs("new")) }()
	select {
	case err := <-configured:
		require.NoError(t, err, "replace policy while old publication waits")
	case <-time.After(time.Second):
		t.Fatal("policy replacement waited on old output")
	}
	d.prune()
	assert.Equal(t, uint64(1), d.metrics.stale.Get(), "queued old-policy work should retire immediately")
	d.enqueue(t.Context(), nil, plugin, result, nil)
	assert.Equal(t, uint64(2), d.metrics.stale.Get(), "old namespace must not enter replacement clients")
	items, _ := d.queue.stats()
	assert.Zero(t, items, "only the one original, active publication may remain")
}

func TestScheduledResultsReuseWorkerAcrossDaemonRestarts(t *testing.T) {
	d, plugin, publisher, _ := outageDispatcher(t)
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
	require.NoError(t, d.configure(outageInputs("idle")), "start the result worker")
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
	require.NoError(t, d.configure(inputs), "configure the independent result publisher")
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
