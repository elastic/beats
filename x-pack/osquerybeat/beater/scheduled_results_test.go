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

type scheduledResultClient struct{}

func (scheduledResultClient) ResolveResult(_ context.Context, _ string, hits []map[string]string) ([]map[string]any, error) {
	result := make([]map[string]any, len(hits))
	for i, hit := range hits {
		result[i] = make(map[string]any)
		for key, value := range hit {
			result[i][key] = value
		}
	}
	return result, nil
}

func (scheduledResultClient) Query(context.Context, string, time.Duration) ([]map[string]any, error) {
	return nil, nil
}

type stalledResponsePublisher struct {
	mockBeatPublisher
	entered chan struct{}
	release chan struct{}
}

func (p *stalledResponsePublisher) PublishScheduledResponse(string, string, string, string, string, string, time.Time, time.Time, time.Time, int, int64) {
	close(p.entered)
	<-p.release
}

func scheduledTestConfig(t *testing.T) *ConfigPlugin {
	t.Helper()
	p := NewConfigPlugin(logptest.NewTestingLogger(t, "scheduled_results"))
	disabled := false
	require.NoError(t, p.Set([]config.InputConfig{{Osquery: &config.OsqueryConfig{
		Schedule: map[string]config.Query{"test": {Query: "SELECT 1", Profiling: &disabled}},
	}}}), "test schedule must be valid")
	_, err := p.GenerateConfig(t.Context())
	require.NoError(t, err, "test schedule metadata must be applied")
	return p
}

func TestScheduledLoggerReturnsWhileOutputBlocked(t *testing.T) {
	for _, payload := range []string{
		`{"name":"test","action":"snapshot","snapshot":[],"unixTime":1704067200}`,
		`{"name":"test","action":"snapshot","snapshot":[{"value":"1"}],"unixTime":1704067200}`,
	} {
		t.Run(payload, func(t *testing.T) {
			p := &stalledResponsePublisher{entered: make(chan struct{}), release: make(chan struct{})}
			bt := &osquerybeat{pub: p, log: logptest.NewTestingLogger(t, "scheduled_results")}
			plugin, stop := bt.newScheduledLogger(t.Context(), scheduledResultClient{}, scheduledTestConfig(t))
			defer stop()
			defer close(p.release)
			done := make(chan error, 1)
			go func() { done <- plugin.Log(t.Context(), logger.LogTypeSnapshot, payload) }()
			select {
			case <-p.entered:
			case <-time.After(time.Second):
				t.Fatal("scheduled response was not attempted, including for zero hits")
			}
			select {
			case err := <-done:
				assert.NoError(t, err, "output backpressure must not fail the osquery logger")
			case <-time.After(200 * time.Millisecond):
				t.Error("logger callback waited for the unavailable output")
			}
		})
	}
}

func TestScheduledResultWorkerBoundsAndShutdown(t *testing.T) {
	for _, tc := range []struct {
		name                     string
		maxCount, maxBytes, size int
	}{
		{"count", 2, 1000, 1}, {"bytes", 10, 10, 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			entered := make(chan struct{})
			metrics := newScheduledResultMetrics(monitoring.NewRegistry(), logptest.NewTestingLogger(t, "worker"))
			worker := newScheduledResultWorker(ctx, tc.maxCount, tc.maxBytes, metrics, func(ctx context.Context, _ scheduledResult) {
				close(entered)
				<-ctx.Done()
			})
			require.True(t, worker.enqueue(scheduledResult{bytes: tc.size}), "first result must be accepted")
			<-entered
			require.True(t, worker.enqueue(scheduledResult{bytes: tc.size}), "second result must fit")
			assert.False(t, worker.enqueue(scheduledResult{bytes: tc.size}), "in-flight result must count against the bound")
			assert.Equal(t, uint64(1), metrics.overflow.Get(), "overflow must be observable")
			done := make(chan struct{})
			go func() { worker.stop(); close(done) }()
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("shutdown waited for output or queued work")
			}
			assert.Equal(t, uint64(1), metrics.shutdown.Get(), "queued work must be discarded on shutdown")
			assert.False(t, worker.enqueue(scheduledResult{bytes: tc.size}), "stopped worker must reject work")
		})
	}
}

func TestScheduledMetadataRetainsAcceptedMapping(t *testing.T) {
	p := scheduledTestConfig(t)
	ns, old, generation, ok := p.lookupResultMetadata("test")
	require.True(t, ok, "scheduled result metadata must exist")
	disabled := false
	require.NoError(t, p.setGeneration([]config.InputConfig{{Datastream: config.DatastreamConfig{Namespace: "new"}, Osquery: &config.OsqueryConfig{
		Schedule: map[string]config.Query{"test": {Query: "SELECT 2", Profiling: &disabled}},
	}}}, generation+1), "new policy must be accepted")
	_, err := p.GenerateConfig(t.Context())
	require.NoError(t, err, "new query metadata must be applied")
	assert.Equal(t, "default", ns, "accepted result must retain its original namespace")
	assert.Equal(t, "SELECT 1", old.Query, "accepted result must retain its original SQL for type resolution")
	newNS, newInfo, newGeneration, ok := p.lookupResultMetadata("test")
	require.True(t, ok, "replacement query must exist")
	assert.Equal(t, "new", newNS, "new results must use the new namespace")
	assert.Equal(t, "SELECT 2", newInfo.Query, "new results must use the new query")
	assert.Equal(t, generation+1, newGeneration, "policy generation must accompany routing metadata")
}

func TestScheduledResultWorkerPreservesFIFO(t *testing.T) {
	metrics := newScheduledResultMetrics(monitoring.NewRegistry(), logptest.NewTestingLogger(t, "worker"))
	received := make(chan int64, 3)
	worker := newScheduledResultWorker(t.Context(), 3, 1000, metrics, func(_ context.Context, res scheduledResult) { received <- res.result.Counter })
	defer worker.stop()
	for i := int64(1); i <= 3; i++ {
		require.True(t, worker.enqueue(scheduledResult{result: QueryResult{Counter: i}, bytes: 1}), "ordered result must fit")
	}
	for i := int64(1); i <= 3; i++ {
		select {
		case actual := <-received:
			assert.Equal(t, i, actual, "scheduled results must retain their order")
		case <-time.After(time.Second):
			t.Fatal("worker did not process accepted result")
		}
	}
}

type pendingProfileClient struct {
	scheduledResultClient
	entered chan struct{}
}

func (c pendingProfileClient) Query(ctx context.Context, _ string, _ time.Duration) ([]map[string]any, error) {
	close(c.entered)
	<-ctx.Done()
	return nil, ctx.Err()
}

type notifiedResponsePublisher struct {
	mockBeatPublisher
	response chan struct{}
}

func (p *notifiedResponsePublisher) PublishScheduledResponse(string, string, string, string, string, string, time.Time, time.Time, time.Time, int, int64) {
	p.response <- struct{}{}
}

func TestScheduledLoggerProfilingDoesNotHoldResponseOrShutdown(t *testing.T) {
	p := scheduledTestConfig(t)
	qi := p.queryInfoMap["test"]
	qi.Profile = true
	p.queryInfoMap["test"] = qi
	cli := pendingProfileClient{entered: make(chan struct{})}
	publisher := &notifiedResponsePublisher{response: make(chan struct{}, 1)}
	bt := &osquerybeat{pub: publisher, log: logptest.NewTestingLogger(t, "profile")}
	bt.qp = newQueryProfiler(bt.log)
	plugin, stop := bt.newScheduledLogger(t.Context(), cli, p)
	defer stop()
	require.NoError(t, plugin.Log(t.Context(), logger.LogTypeSnapshot, `{"name":"test","action":"snapshot","snapshot":[]}`), "logger must return while scheduled profiling is pending")
	select {
	case <-cli.entered:
	case <-time.After(time.Second):
		t.Fatal("scheduled profile was not queried")
	}
	select {
	case <-publisher.response:
	default:
		t.Error("scheduled response waited for the profiling query")
	}
	done := make(chan struct{})
	go func() { stop(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("shutdown did not cancel the pending profile")
	}
}
