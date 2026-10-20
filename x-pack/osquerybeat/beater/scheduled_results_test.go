// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"strings"
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

func outageDispatcher(t *testing.T) (*scheduledResultDispatcher, *ConfigPlugin, *stalledResultPublisher, func()) {
	t.Helper()
	log := logptest.NewTestingLogger(t, "outage")
	bt := &osquerybeat{log: log}
	first := &stalledResultPublisher{entered: make(chan struct{}), release: make(chan struct{})}
	created := false
	d := newScheduledResultDispatcher(monitoring.NewRegistry(), log, func(func() bool, func()) osquerybeatPublisher {
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
		d.enqueue(t.Context(), nil, plugin, res)
	})
	payload := `{"name":"scheduled","action":"diff","unixTime":123}`
	require.NoError(t, loggerPlugin.Log(t.Context(), logger.LogTypeString, payload), "zero-hit logger call should return before its response is published")
	waitForResultPublication(t, publisher)

	// Keep the first response blocked, fill the bounded backlog, and prove
	// logger admission and shutdown both finish before the output recovers.
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		for range scheduledResultLimit {
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
	assert.Equal(t, uint64(1), d.metrics.overflow.Get(), "the active result must count against the document limit")
	assert.Equal(t, uint64(scheduledResultLimit-1), d.metrics.shutdown.Get(), "shutdown should discard all pending results")
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

func TestScheduledResultsBoundsBytesIncludingActiveDocument(t *testing.T) {
	d, plugin, publisher, _ := outageDispatcher(t)
	result := QueryResult{Name: "scheduled", Action: "diff", Hits: []map[string]string{{"value": strings.Repeat("x", 3<<20)}}}
	d.enqueue(t.Context(), nil, plugin, result)
	waitForResultPublication(t, publisher)
	d.enqueue(t.Context(), nil, plugin, result)
	d.enqueue(t.Context(), nil, plugin, result)
	assert.Equal(t, uint64(1), d.metrics.overflow.Get(), "third document must exceed the retained-byte budget")
	d.mu.Lock()
	assert.Equal(t, 2, d.count, "count the active and pending documents")
	assert.LessOrEqual(t, d.bytes, scheduledResultBytesLimit, "retained estimates must stay bounded")
	d.mu.Unlock()
}

func TestScheduledResultsPolicyRetirementDoesNotWaitForOutput(t *testing.T) {
	d, plugin, publisher, _ := outageDispatcher(t)
	result := QueryResult{Name: "scheduled", Action: "diff"}
	d.enqueue(t.Context(), nil, plugin, result)
	waitForResultPublication(t, publisher)
	d.enqueue(t.Context(), nil, plugin, result)
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
	d.enqueue(t.Context(), nil, plugin, result)
	assert.Equal(t, uint64(2), d.metrics.stale.Get(), "old namespace must not enter replacement clients")
	d.mu.Lock()
	assert.Equal(t, 1, d.count, "only the one original publication may remain")
	d.mu.Unlock()
}

func TestScheduledResultsReuseWorkerAcrossDaemonRestarts(t *testing.T) {
	d, plugin, publisher, _ := outageDispatcher(t)
	result := QueryResult{Name: "scheduled", Action: "diff"}
	ctx, cancel := context.WithCancel(t.Context())
	d.enqueue(ctx, nil, plugin, result)
	waitForResultPublication(t, publisher)
	cancel()
	for range 20 {
		ctx, cancel := context.WithCancel(t.Context())
		d.enqueue(ctx, nil, plugin, result)
		cancel()
		d.prune()
	}
	d.mu.Lock()
	assert.Equal(t, 1, d.count, "cancelled runs must not accumulate behind the stuck publication")
	d.mu.Unlock()
	assert.Equal(t, int32(1), publisher.calls.Load(), "restarts must reuse the single publishing worker")
	assert.Equal(t, uint64(20), d.metrics.stale.Get(), "prune each cancelled run from the backlog")
}

func TestScheduledLoggerDropsOversizeBeforeParsing(t *testing.T) {
	var dropped atomic.Int32
	p := NewLoggerPlugin(logptest.NewTestingLogger(t, "oversized"), func(QueryResult) { t.Error("oversized input reached result handling") })
	p.maxResultBytes = 8
	p.dropOversized = func() { dropped.Add(1) }
	assert.NoError(t, p.Log(t.Context(), logger.LogTypeString, strings.Repeat("invalid JSON", 2)), "oversized payload must drop before JSON parsing")
	assert.Equal(t, int32(1), dropped.Load(), "count the dropped document")
}
