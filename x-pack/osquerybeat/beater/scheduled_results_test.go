// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/osquery/osquery-go/plugin/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/ecs"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

type stalledResultPublisher struct {
	mockBeatPublisher
	entered  chan struct{}
	release  chan struct{}
	calls    atomic.Int32
	profiles atomic.Int32
	closed   atomic.Int32
	once     sync.Once
}

func (p *stalledResultPublisher) PublishScheduledResponse(string, string, string, string, string, string, time.Time, time.Time, time.Time, int, int64) {
	p.calls.Add(1)
	p.once.Do(func() { close(p.entered) })
	<-p.release
}
func (p *stalledResultPublisher) PublishScheduledProfile(string, string, string, string, string, int64, map[string]any) {
	p.profiles.Add(1)
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

// outageProfileInputs adds a query profile stream to outageInputs.
func outageProfileInputs(namespace string) []config.InputConfig {
	return append(outageInputs(namespace), config.InputConfig{
		Datastream: config.DatastreamConfig{Dataset: config.DefaultQueryProfileDataset, Namespace: namespace},
	})
}

// outageMaxEvents is the outage dispatcher's queue limit in rows.
const outageMaxEvents = 8

var outageLimits = config.ScheduledPublishingConfig{MaxEvents: outageMaxEvents}

// outageDispatcher returns a dispatcher whose first policy's publisher blocks
// until release is called. Publishers for later policies do not block.
func outageDispatcher(t *testing.T, limits config.ScheduledPublishingConfig) (*scheduledResultDispatcher, *ConfigPlugin, *stalledResultPublisher, func()) {
	t.Helper()
	return outageDispatcherFor(t, outageInputs("old"), limits)
}

// outageDispatcherFor is outageDispatcher with inputs as the first policy.
func outageDispatcherFor(t *testing.T, inputs []config.InputConfig, limits config.ScheduledPublishingConfig) (*scheduledResultDispatcher, *ConfigPlugin, *stalledResultPublisher, func()) {
	t.Helper()
	log := logptest.NewTestingLogger(t, "outage")
	bt := &osquerybeat{log: log}
	first := &stalledResultPublisher{entered: make(chan struct{}), release: make(chan struct{})}
	created := false
	d := newScheduledResultDispatcher(monitoring.NewRegistry(), log, outageMaxEvents, func(func() bool, func(), func(int)) osquerybeatPublisher {
		if !created {
			created = true
			return first
		}
		next := &stalledResultPublisher{entered: make(chan struct{}), release: make(chan struct{})}
		close(next.release)
		return next
	}, bt.handleScheduledQueryResult)
	plugin := NewConfigPlugin(log)
	require.NoError(t, plugin.Set(inputs), "validate the initial policy")
	_, err := plugin.GenerateConfig(t.Context())
	require.NoError(t, err, "apply the initial native policy")
	require.NoError(t, d.configure(inputs, limits), "configure independent scheduled clients")
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

// stopOnCleanup stops d when the test ends and waits for its worker, which
// must not log once the test has finished. d's publisher must not block.
func stopOnCleanup(t *testing.T, d *scheduledResultDispatcher) {
	t.Helper()
	t.Cleanup(func() {
		d.stop()
		select {
		case <-d.done:
		case <-time.After(time.Second):
			t.Error("scheduled result worker did not stop")
		}
	})
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
	assert.Equal(t, uint64(1), d.metrics.dropped[dropOverflow].Get(), "the oldest queued result must be dropped once the row limit is reached")
	assert.Equal(t, uint64(1), d.metrics.droppedRows[dropOverflow].Get(), "count the dropped result's rows")
	assert.Equal(t, uint64(outageMaxEvents), d.metrics.dropped[dropShutdown].Get(), "shutdown should discard all pending results")
	assert.Equal(t, uint64(outageMaxEvents), d.metrics.droppedRows[dropShutdown].Get(), "count the rows of the discarded results")
	assert.Zero(t, d.metrics.items.Get(), "the queue.items gauge must be zero after stop discards the backlog")
	assert.Zero(t, d.metrics.events.Get(), "the queue.events gauge must be zero after stop discards the backlog")
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

func TestScheduledResultsOverflowCountsRows(t *testing.T) {
	d, plugin, blocked, _ := outageDispatcher(t, outageLimits)
	d.enqueue(t.Context(), nil, plugin, QueryResult{Name: "scheduled", Action: "diff"}, nil)
	waitForResultPublication(t, blocked)

	// Each result is two rows plus its response, so the third overflows the
	// eight-row limit.
	result := QueryResult{Name: "scheduled", Action: "snapshot", Hits: make([]map[string]string, 2)}
	for range 3 {
		d.enqueue(t.Context(), outageQueryClient{}, plugin, result, nil)
	}
	assert.Equal(t, uint64(1), d.metrics.dropped[dropOverflow].Get(), "the oldest queued result must be dropped")
	assert.Equal(t, uint64(3), d.metrics.droppedRows[dropOverflow].Get(), "count every row of the dropped result")
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
	assert.Equal(t, uint64(1), d.metrics.dropped[dropStale].Get(), "old namespace must not enter replacement clients")
	assert.Equal(t, uint64(1), d.metrics.droppedRows[dropStale].Get(), "count the stale result's rows")
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
	next := activePublisher(t, d)
	release()
	assert.Eventually(t, func() bool { return next.calls.Load() == 3 }, time.Second, time.Millisecond,
		"queued results must publish through the new policy's clients")
	assert.Equal(t, int32(1), blocked.calls.Load(), "only the in-flight result may use the old clients")
	assert.Zero(t, d.metrics.dropped[dropStale].Get(), "an unchanged query must not be dropped as stale")
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
	next := activePublisher(t, d)
	release()
	assert.Eventually(t, func() bool { return d.metrics.dropped[dropStale].Get() == 3 }, time.Second, time.Millisecond,
		"results for a moved query must be dropped as stale")
	assert.Equal(t, uint64(3), d.metrics.droppedRows[dropStale].Get(), "count the stale results' rows")
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
	assert.Eventually(t, func() bool { return d.metrics.dropped[dropExpired].Get() == 2 }, time.Second, time.Millisecond,
		"results queued longer than max_age must be dropped as expired")
	assert.Equal(t, uint64(2), d.metrics.droppedRows[dropExpired].Get(), "count the expired results' rows")
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
	next := activePublisher(t, d)
	d.enqueue(t.Context(), nil, plugin, result, nil)
	release()
	assert.Eventually(t, func() bool { return d.metrics.dropped[dropStale].Get() == 3 }, time.Second, time.Millisecond,
		"results captured for the old SQL must be dropped as stale at admission and when dequeued")
	assert.Equal(t, uint64(3), d.metrics.droppedRows[dropStale].Get(), "count the stale results' rows")
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
	next := activePublisher(t, d)
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
	assert.Zero(t, d.metrics.dropped[dropStale].Get(), "an ECS mapping change must not make results stale")
}

func TestScheduledProfilesQueuedAndEvictedFirst(t *testing.T) {
	// Each result is four rows plus its response. The first one holds the output.
	d, plugin, blocked, release := outageDispatcherFor(t, outageProfileInputs("old"), config.ScheduledPublishingConfig{MaxEvents: 10})
	result := QueryResult{Name: "scheduled", Action: "snapshot", Hits: make([]map[string]string, 4)}
	d.enqueue(t.Context(), outageQueryClient{}, plugin, result, nil)
	waitForResultPublication(t, blocked)

	// Queue a result and then a profile behind the in-flight result, then
	// overflow the limit. The profile goes although the result is older.
	d.enqueue(t.Context(), outageQueryClient{}, plugin, result, nil)
	d.enqueueProfile(scheduledProfileItem{name: "scheduled", namespace: "old", query: "SELECT 1", profile: map[string]any{}})
	d.enqueue(t.Context(), outageQueryClient{}, plugin, result, nil)
	assert.Equal(t, uint64(1), d.metrics.dropped[dropOverflow].Get(), "only the profile should be evicted")
	assert.Equal(t, uint64(1), d.metrics.droppedRows[dropOverflow].Get(), "a profile is one row")
	items, events := d.queue.stats()
	assert.Equal(t, 2, items, "both queued results should stay queued")
	assert.Equal(t, 10, events, "the queued results should fill the queue")
	assert.Equal(t, int64(2), d.metrics.items.Get(), "the queue.items gauge must report the queued results")
	assert.Equal(t, int64(10), d.metrics.events.Get(), "the queue.events gauge must report the queued rows")

	// Both results are published once the output recovers; the profile is not.
	release()
	assert.Eventually(t, func() bool { return blocked.calls.Load() == 3 }, time.Second, time.Millisecond,
		"the queued results must be published")
	assert.Zero(t, blocked.profiles.Load(), "the evicted profile must not be published")
	assert.Eventually(t, func() bool { return d.metrics.items.Get() == 0 && d.metrics.events.Get() == 0 }, time.Second, time.Millisecond,
		"the queue gauges must drop to zero once the worker takes the results")
}

func TestScheduledProfilesExpire(t *testing.T) {
	d, plugin, blocked, release := outageDispatcherFor(t, outageProfileInputs("old"),
		config.ScheduledPublishingConfig{MaxEvents: outageMaxEvents, MaxAge: 50 * time.Millisecond})
	d.enqueue(t.Context(), nil, plugin, QueryResult{Name: "scheduled", Action: "diff"}, nil)
	waitForResultPublication(t, blocked)
	d.enqueueProfile(scheduledProfileItem{name: "scheduled", namespace: "old", query: "SELECT 1", profile: map[string]any{}})
	time.Sleep(100 * time.Millisecond)
	release()
	assert.Eventually(t, func() bool { return d.metrics.dropped[dropExpired].Get() == 1 }, time.Second, time.Millisecond,
		"count the expired profile in the queue's counters")
	assert.Equal(t, uint64(1), d.metrics.droppedRows[dropExpired].Get(), "an expired profile is one row")
	assert.Zero(t, blocked.profiles.Load(), "an expired profile must not be published")
}

// TestScheduledPublishDroppedOnCloseCountsRows checks that an event the
// pipeline drops while closing counts as a row, not as a queue item.
func TestScheduledPublishDroppedOnCloseCountsRows(t *testing.T) {
	log := logptest.NewTestingLogger(t, "dropped-on-close")
	var dropped func()
	d := newScheduledResultDispatcher(monitoring.NewRegistry(), log, outageMaxEvents, func(_ func() bool, onDrop func(), _ func(int)) osquerybeatPublisher {
		dropped = onDrop
		return &mockBeatPublisher{}
	}, (&osquerybeat{log: log}).handleScheduledQueryResult)
	require.NoError(t, d.configure(outageInputs("old"), outageLimits), "configure the scheduled clients")
	stopOnCleanup(t, d)

	dropped()
	assert.Zero(t, d.metrics.dropped[dropShutdown].Get(), "a dropped event is not a dropped queue item")
	assert.Equal(t, uint64(1), d.metrics.droppedRows[dropShutdown].Get(), "count the dropped event as a shutdown row")
}

// unresolvedQueryClient fails ResolveResult from its failFrom-th call on,
// counting from 0.
type unresolvedQueryClient struct {
	failFrom int
	calls    *int // only the worker calls it
}

func (c unresolvedQueryClient) ResolveResult(_ context.Context, _ string, rows []map[string]string) ([]map[string]any, error) {
	call := *c.calls
	*c.calls++
	if call >= c.failFrom {
		return nil, errors.New("resolve failed")
	}
	return make([]map[string]any, len(rows)), nil
}

func TestScheduledResultsUnresolvedRowsCountStale(t *testing.T) {
	rows := func(n int) []map[string]string { return make([]map[string]string, n) }
	diff := QueryResult{Name: "scheduled", Action: "diff"}
	diff.DiffResults.Added, diff.DiffResults.Removed = rows(2), rows(3)
	tests := []struct {
		name     string
		result   QueryResult
		failFrom int
		want     uint64 // stale rows
	}{
		{name: "snapshot", result: QueryResult{Name: "scheduled", Action: "snapshot", Hits: rows(2)}, failFrom: 0, want: 2 + 1},
		{
			name:     "diff removed",
			result:   diff,
			failFrom: 1,
			want:     3 + 1,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			log := logptest.NewTestingLogger(t, "unresolved")
			d := newScheduledResultDispatcher(monitoring.NewRegistry(), log, outageMaxEvents,
				func(func() bool, func(), func(int)) osquerybeatPublisher { return &mockBeatPublisher{} }, (&osquerybeat{log: log}).handleScheduledQueryResult)
			inputs := outageInputs("old")
			plugin := NewConfigPlugin(log)
			require.NoError(t, plugin.Set(inputs), "validate the policy")
			_, err := plugin.GenerateConfig(t.Context())
			require.NoError(t, err, "apply the native policy")
			require.NoError(t, d.configure(inputs, outageLimits), "configure the scheduled clients")
			stopOnCleanup(t, d)

			d.enqueue(t.Context(), unresolvedQueryClient{failFrom: test.failFrom, calls: new(int)}, plugin, test.result, nil)

			assert.Eventually(t, func() bool { return d.metrics.dropped[dropStale].Get() == 1 }, time.Second, time.Millisecond,
				"a result whose rows could not be resolved must count as one stale item")
			assert.Equal(t, test.want, d.metrics.droppedRows[dropStale].Get(), "count the unresolved rows and the unpublished response")
		})
	}
}

// cuttingResultPublisher stops publishing rows once its lease is retired and
// reports the rows it skipped, like pub.Publisher. Its first row waits for
// release; done is closed when it publishes a response.
type cuttingResultPublisher struct {
	mockBeatPublisher
	allowed   func() bool
	cut       func(int)
	entered   chan struct{}
	release   chan struct{}
	done      chan struct{}
	once      sync.Once
	published atomic.Int32
}

func (p *cuttingResultPublisher) Publish(_, _, _, _, _, _, _, _ string, _ map[string]any, hits []map[string]any, _ ecs.Mapping, _ any) {
	p.once.Do(func() {
		close(p.entered)
		<-p.release
	})
	for i := range hits {
		if !p.allowed() {
			p.cut(len(hits) - i)
			return
		}
		p.published.Add(1)
	}
}

func (p *cuttingResultPublisher) PublishScheduledResponse(string, string, string, string, string, string, time.Time, time.Time, time.Time, int, int64) {
	if !p.allowed() {
		p.cut(1)
		return
	}
	p.published.Add(1)
	close(p.done)
}

func TestScheduledResultsRetiredLeaseCutsUnroutedRows(t *testing.T) {
	rows := []map[string]string{{"value": "a"}, {"value": "b"}, {"value": "c"}}
	tests := []struct {
		name          string
		namespace     string // of the replacement policy
		stop          bool   // stop the dispatcher instead of replacing the policy
		wantPublished int    // rows and response published by the retired clients
		reason        string // drop reason of the cut rows
		wantCut       uint64 // rows cut
	}{
		{name: "still routed", namespace: "old", wantPublished: len(rows) + 1, reason: dropStale},
		{name: "moved namespace", namespace: "other", reason: dropStale, wantCut: uint64(len(rows) + 1)},
		{name: "stopped", stop: true, reason: dropShutdown, wantCut: uint64(len(rows) + 1)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			log := logptest.NewTestingLogger(t, "cut")
			var first *cuttingResultPublisher
			d := newScheduledResultDispatcher(monitoring.NewRegistry(), log, outageMaxEvents, func(allowed func() bool, _ func(), cut func(int)) osquerybeatPublisher {
				p := &cuttingResultPublisher{allowed: allowed, cut: cut, entered: make(chan struct{}), release: make(chan struct{}), done: make(chan struct{})}
				if first == nil {
					first = p
				} else {
					close(p.release)
				}
				return p
			}, (&osquerybeat{log: log}).handleScheduledQueryResult)
			inputs := outageInputs("old")
			plugin := NewConfigPlugin(log)
			require.NoError(t, plugin.Set(inputs), "validate the policy")
			_, err := plugin.GenerateConfig(t.Context())
			require.NoError(t, err, "apply the native policy")
			require.NoError(t, d.configure(inputs, outageLimits), "configure the first scheduled clients")
			stopOnCleanup(t, d)

			d.enqueue(t.Context(), outageQueryClient{}, plugin, QueryResult{Name: "scheduled", Action: "snapshot", Hits: rows}, nil)
			select {
			case <-first.entered:
			case <-time.After(time.Second):
				t.Fatal("the result did not reach the blocked output")
			}
			if test.stop {
				d.stop()
			} else {
				require.NoError(t, d.configure(outageInputs(test.namespace), outageLimits), "retire the clients publishing the result")
			}
			close(first.release)

			if test.wantCut == 0 {
				select {
				case <-first.done:
				case <-time.After(time.Second):
					t.Fatal("the in-flight result's response was not published")
				}
			} else {
				assert.Eventually(t, func() bool { return d.metrics.dropped[test.reason].Get() == 1 }, time.Second, time.Millisecond,
					"a result cut short by retirement must count as one %s item", test.reason)
			}
			assert.Equal(t, test.wantPublished, int(first.published.Load()), "rows and response published through the retired clients")
			assert.Equal(t, test.wantCut, d.metrics.droppedRows[test.reason].Get(), "rows cut by retirement")
		})
	}
}

func TestScheduledProfilesNeedProfileStream(t *testing.T) {
	// Queue a profile while the output is blocked.
	d, plugin, blocked, release := outageDispatcherFor(t, outageProfileInputs("old"), outageLimits)
	d.enqueue(t.Context(), nil, plugin, QueryResult{Name: "scheduled", Action: "diff"}, nil)
	waitForResultPublication(t, blocked)
	profile := scheduledProfileItem{name: "scheduled", namespace: "old", query: "SELECT 1", profile: map[string]any{}}
	d.enqueueProfile(profile)

	// Remove the profile stream; the queued and a new profile must both be stale.
	require.NoError(t, d.configure(outageInputs("old"), outageLimits), "remove the profile stream")
	next := activePublisher(t, d)
	d.enqueueProfile(profile)
	release()
	assert.Eventually(t, func() bool { return d.metrics.dropped[dropStale].Get() == 2 }, time.Second, time.Millisecond,
		"profiles must be dropped as stale without a profile stream, when admitted and when dequeued")
	assert.Zero(t, blocked.profiles.Load()+next.profiles.Load(), "no profile may be published without a profile stream")
}

func TestScheduledRunProfileNeedsProfileStream(t *testing.T) {
	d, _, publisher, release := outageDispatcher(t, outageLimits)
	release()
	d.enqueueResolved(resolvedScheduledResult{name: "scheduled", namespace: "old", query: QueryInfo{Query: "SELECT 1"}, profile: map[string]any{}})
	assert.Eventually(t, func() bool { return publisher.calls.Load() == 1 }, time.Second, time.Millisecond,
		"the run's response must be published without a profile stream")
	assert.Zero(t, publisher.profiles.Load(), "the run's profile must not be published without a profile stream")
	assert.Equal(t, uint64(1), d.metrics.droppedRows[dropStale].Get(), "count the run's profile as a stale row")
	assert.Zero(t, d.metrics.dropped[dropStale].Get(), "the run itself was published")
}

func TestScheduledItemsDroppedAfterStop(t *testing.T) {
	tests := []struct {
		name    string
		enqueue func(*scheduledResultDispatcher)
	}{
		{
			name: "resolved run",
			enqueue: func(d *scheduledResultDispatcher) {
				d.enqueueResolved(resolvedScheduledResult{name: "scheduled", namespace: "old", query: QueryInfo{Query: "SELECT 1"}})
			},
		},
		{
			name: "profile",
			enqueue: func(d *scheduledResultDispatcher) {
				d.enqueueProfile(scheduledProfileItem{name: "scheduled", namespace: "old", query: "SELECT 1"})
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			log := logptest.NewTestingLogger(t, "stopped")
			d := newScheduledResultDispatcher(monitoring.NewRegistry(), log, outageMaxEvents,
				func(func() bool, func(), func(int)) osquerybeatPublisher { return &mockBeatPublisher{} }, (&osquerybeat{log: log}).handleScheduledQueryResult)
			require.NoError(t, d.configure(outageProfileInputs("old"), outageLimits), "configure a policy that routes the item")
			stopOnCleanup(t, d)
			d.stop()

			test.enqueue(d)
			assert.Equal(t, uint64(1), d.metrics.dropped[dropShutdown].Get(), "an item admitted after stop must be dropped as shutdown")
			assert.Zero(t, d.metrics.dropped[dropStale].Get(), "an item admitted after stop is not stale")
			items, _ := d.queue.stats()
			assert.Zero(t, items, "a stopped dispatcher must not queue the item")
		})
	}
}

// countingQueryClient resolves every row and counts its calls.
type countingQueryClient struct{ calls atomic.Int32 }

func (c *countingQueryClient) ResolveResult(_ context.Context, _ string, rows []map[string]string) ([]map[string]any, error) {
	c.calls.Add(1)
	return make([]map[string]any, len(rows)), nil
}

// endRun ends an osqueryd run as runOsquery does when it returns.
func endRun(d *scheduledResultDispatcher, cancel context.CancelFunc, client scheduledQueryClient) {
	cancel()
	d.clearClient(client)
	d.prune()
}

func TestScheduledResultsSurviveDaemonRestart(t *testing.T) {
	d, plugin, blocked, release := outageDispatcher(t, outageLimits)
	first, second := &countingQueryClient{}, &countingQueryClient{}
	run, cancel := context.WithCancel(t.Context())
	d.setClient(run, first)
	d.enqueue(run, first, plugin, QueryResult{Name: "scheduled", Action: "diff"}, nil)
	waitForResultPublication(t, blocked)
	rows := []map[string]string{{"value": "a"}, {"value": "b"}}
	d.enqueue(run, first, plugin, QueryResult{Name: "scheduled", Action: "snapshot", Hits: rows}, nil)

	endRun(d, cancel, first)
	d.setClient(t.Context(), second)
	release()
	assert.Eventually(t, func() bool { return blocked.calls.Load() == 2 }, time.Second, time.Millisecond,
		"a result queued by an ended run must be published")
	assert.Zero(t, first.calls.Load(), "the ended run's client must not resolve rows")
	assert.Equal(t, int32(1), second.calls.Load(), "the current run's client must resolve the rows")
	assert.Zero(t, d.metrics.dropped[dropStale].Get(), "a result that still routes must not be dropped as stale")
}

func TestScheduledResultsWaitForDaemonRun(t *testing.T) {
	d, plugin, blocked, release := outageDispatcher(t, outageLimits)
	client := &countingQueryClient{}
	run, cancel := context.WithCancel(t.Context())
	d.setClient(run, client)
	d.enqueue(run, client, plugin, QueryResult{Name: "scheduled", Action: "diff"}, nil)
	waitForResultPublication(t, blocked)
	rows := []map[string]string{{"value": "a"}, {"value": "b"}}
	d.enqueue(run, client, plugin, QueryResult{Name: "scheduled", Action: "snapshot", Hits: rows}, nil)

	endRun(d, cancel, client)
	release()
	assert.Never(t, func() bool { return d.metrics.dropped[dropStale].Get() > 0 || blocked.calls.Load() > 1 }, 100*time.Millisecond, time.Millisecond,
		"a result must be neither dropped nor published while no osqueryd runs")
	items, events := d.queue.stats()
	assert.Equal(t, 1, items, "the result must wait in the queue for the next run")
	assert.Equal(t, len(rows)+1, events, "the waiting result's rows must count against the limit")

	next := &countingQueryClient{}
	d.setClient(t.Context(), next)
	assert.Eventually(t, func() bool { return blocked.calls.Load() == 2 }, time.Second, time.Millisecond,
		"the waiting result must be published once a run starts")
	assert.Equal(t, int32(1), next.calls.Load(), "the new run's client must resolve the rows")
	assert.Zero(t, client.calls.Load(), "the ended run's client must not resolve rows")
}

// ctxQueryClient resolves rows only while ctx is not done, as osqdcli.Client
// does, and counts its calls.
type ctxQueryClient struct{ calls atomic.Int32 }

func (c *ctxQueryClient) ResolveResult(ctx context.Context, _ string, rows []map[string]string) ([]map[string]any, error) {
	c.calls.Add(1)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return make([]map[string]any, len(rows)), nil
}

func TestScheduledResultsWaitWhileDaemonStops(t *testing.T) {
	d, plugin, blocked, release := outageDispatcher(t, outageLimits)
	client := &ctxQueryClient{}
	run, cancel := context.WithCancel(t.Context())
	d.setClient(run, client)
	d.enqueue(run, client, plugin, QueryResult{Name: "scheduled", Action: "diff"}, nil)
	waitForResultPublication(t, blocked)
	rows := []map[string]string{{"value": "a"}, {"value": "b"}}
	d.enqueue(run, client, plugin, QueryResult{Name: "scheduled", Action: "snapshot", Hits: rows}, nil)

	// The run has ended, but runOsquery has not cleared its client yet.
	cancel()
	release()
	assert.Never(t, func() bool { return d.metrics.dropped[dropStale].Get() > 0 || blocked.calls.Load() > 1 }, 100*time.Millisecond, time.Millisecond,
		"a result must be neither dropped nor published with the client of an ended run")
	d.mu.Lock()
	current := d.client
	d.mu.Unlock()
	assert.Nil(t, current, "the waiting worker must clear the ended run's client")

	next := &ctxQueryClient{}
	d.setClient(t.Context(), next)
	assert.Eventually(t, func() bool { return blocked.calls.Load() == 2 }, time.Second, time.Millisecond,
		"the waiting result must be published once a run starts")
	assert.Equal(t, int32(1), next.calls.Load(), "the new run's client must resolve the rows")
	assert.Zero(t, client.calls.Load(), "the ended run's client must not resolve rows")
	assert.Zero(t, d.metrics.dropped[dropStale].Get(), "a result that still routes must not be dropped as stale")
}

func TestScheduledResultsUnroutedAtDaemonRestartAreStale(t *testing.T) {
	d, plugin, blocked, _ := outageDispatcher(t, outageLimits)
	client := &countingQueryClient{}
	run, cancel := context.WithCancel(t.Context())
	d.setClient(run, client)
	d.enqueue(run, client, plugin, QueryResult{Name: "scheduled", Action: "diff"}, nil)
	waitForResultPublication(t, blocked)
	d.enqueue(run, client, plugin, QueryResult{Name: "scheduled", Action: "diff"}, nil)
	require.NoError(t, d.configure(outageInputs("other"), outageLimits), "move the query to another namespace")

	endRun(d, cancel, client)
	assert.Equal(t, uint64(1), d.metrics.dropped[dropStale].Get(), "a result the policy no longer routes must be pruned as stale")
	items, _ := d.queue.stats()
	assert.Zero(t, items, "the pruned result must leave the queue")
}

func TestScheduledResultsConfigureNamesInvalidPolicy(t *testing.T) {
	log := logptest.NewTestingLogger(t, "invalid-mapping")
	d := newScheduledResultDispatcher(monitoring.NewRegistry(), log, outageMaxEvents,
		func(func() bool, func(), func(int)) osquerybeatPublisher { return &mockBeatPublisher{} }, (&osquerybeat{log: log}).handleScheduledQueryResult)
	t.Cleanup(d.stop)
	inputs := outageInputs("old")
	query := inputs[0].Osquery.Schedule["scheduled"]
	query.ECSMapping = map[string]any{"user": map[string]any{"id": map[string]any{"field": ""}}}
	inputs[0].Osquery.Schedule["scheduled"] = query

	err := d.configure(inputs, outageLimits)
	assert.ErrorIs(t, err, ErrECSMappingIsInvalid, "the policy's invalid ECS mapping must be rejected")
	assert.ErrorContains(t, err, "invalid osquery configuration", "the error must name the configuration as the cause")
}

func TestScheduledResultsInvalidPolicyKeepsLimits(t *testing.T) {
	log := logptest.NewTestingLogger(t, "invalid-policy")
	d := newScheduledResultDispatcher(monitoring.NewRegistry(), log, outageMaxEvents, func(func() bool, func(), func(int)) osquerybeatPublisher {
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

func TestScheduledResultsRemovedInputsKeepPolicy(t *testing.T) {
	limits := config.ScheduledPublishingConfig{MaxEvents: outageMaxEvents, MaxAge: time.Hour}
	d, plugin, blocked, release := outageDispatcher(t, limits)
	result := QueryResult{Name: "scheduled", Action: "diff"}
	d.enqueue(t.Context(), nil, plugin, result, nil)
	waitForResultPublication(t, blocked)
	for range 3 {
		d.enqueue(t.Context(), nil, plugin, result, nil)
	}

	// configurePublishers passes the default settings when the inputs are removed.
	publishing, err := config.GetPublishingConfig(nil)
	require.NoError(t, err, "read the settings of no inputs")
	require.NoError(t, d.configure(nil, publishing.Scheduled), "remove the inputs")
	assert.Equal(t, outageMaxEvents, queueLimit(d), "removing the inputs must keep the queue limit")
	d.mu.Lock()
	maxAge := d.maxAge
	d.mu.Unlock()
	assert.Equal(t, time.Hour, maxAge, "removing the inputs must keep max_age")
	release()
	assert.Eventually(t, func() bool { return blocked.calls.Load() == 4 }, time.Second, time.Millisecond,
		"queued results must keep publishing through the current policy's clients")
	assert.Zero(t, d.metrics.dropped[dropStale].Get(), "removing the inputs must not make queued results stale")

	require.NoError(t, d.configure(outageInputs("old"), config.ScheduledPublishingConfig{MaxEvents: 5}), "apply a later policy")
	assert.Equal(t, 5, queueLimit(d), "a later policy must apply its queue limit")
	assert.NotSame(t, blocked, activePublisher(t, d), "a later policy must replace the clients")
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
	d.handle = func(ctx context.Context, cli scheduledQueryClient, publisher scheduledQueryPublisher, ns string, qi QueryInfo, res QueryResult, responseID string) int {
		calls <- handledResult{publisher: publisher, query: qi}
		return handle(ctx, cli, publisher, ns, qi, res, responseID)
	}
	return calls
}

// activePublisher returns the publisher of d's current policy.
func activePublisher(t *testing.T, d *scheduledResultDispatcher) *stalledResultPublisher {
	d.mu.Lock()
	defer d.mu.Unlock()
	publisher, ok := d.lease.publisher.(*stalledResultPublisher)
	require.True(t, ok, "the active publisher should be a stalled result publisher")
	return publisher
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
		client := &countingQueryClient{}
		d.setClient(ctx, client)
		d.enqueue(ctx, client, plugin, result, nil)
		endRun(d, cancel, client)
	}
	items, _ := d.queue.stats()
	assert.Equal(t, outageMaxEvents, items, "the backlog of ended runs must stay bounded by the queue limit")
	assert.Equal(t, uint64(20-outageMaxEvents), d.metrics.dropped[dropOverflow].Get(), "results beyond the limit must be dropped as overflow")
	assert.Zero(t, d.metrics.dropped[dropStale].Get(), "results that still route must survive restarts")
	assert.Equal(t, int32(1), publisher.calls.Load(), "restarts must reuse the single publishing worker")
}

func TestScheduledResultsStopReleasesIdleWorker(t *testing.T) {
	log := logptest.NewTestingLogger(t, "idle")
	d := newScheduledResultDispatcher(monitoring.NewRegistry(), log, outageMaxEvents, func(func() bool, func(), func(int)) osquerybeatPublisher {
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

// Profiles are published by the scheduled worker, but collected by their own
// worker. Capturing correlation when admitting the logger document must not
// make collection wait for the output.
type correlatedStalledResultPublisher struct {
	stalledResultPublisher
	response chan scheduledProfileResponse
	profile  chan collectedProfile
}

func (p *correlatedStalledResultPublisher) PublishScheduledResponse(scheduleID, packID, packName, queryName, spaceID, responseID string, startedAt, completedAt, plannedAt time.Time, count int, executionCount int64) {
	p.response <- scheduledProfileResponse{scheduleID: scheduleID, responseID: responseID, executionCount: executionCount}
	p.stalledResultPublisher.PublishScheduledResponse(scheduleID, packID, packName, queryName, spaceID, responseID, startedAt, completedAt, plannedAt, count, executionCount)
}

func (p *correlatedStalledResultPublisher) PublishScheduledProfile(_, name, scheduleID, responseID, spaceID string, executionCount int64, profile map[string]any) {
	p.profile <- collectedProfile{name, scheduleID, responseID, spaceID, executionCount, profile}
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

	bt := &osquerybeat{log: log}
	publisher := &correlatedStalledResultPublisher{
		stalledResultPublisher: stalledResultPublisher{entered: make(chan struct{}), release: make(chan struct{})},
		response:               make(chan scheduledProfileResponse, 1),
		profile:                make(chan collectedProfile, 1),
	}
	release := sync.OnceFunc(func() { close(publisher.release) })
	d := newScheduledResultDispatcher(monitoring.NewRegistry(), log, config.DefaultScheduledMaxEvents,
		func(func() bool, func(), func(int)) osquerybeatPublisher { return publisher }, bt.handleScheduledQueryResult)
	d.profile = bt.enqueueScheduledProfile
	require.NoError(t, d.configure(inputs, config.ScheduledPublishingConfig{MaxEvents: config.DefaultScheduledMaxEvents}), "configure the scheduled clients")
	t.Cleanup(func() {
		release()
		d.stop()
		select {
		case <-d.done:
		case <-time.After(time.Second):
			t.Error("released result publisher did not stop")
		}
	})
	executor := profileQueryFunc(func(context.Context, string, time.Duration) ([]map[string]any, error) {
		return profileTestRow("SELECT 1"), nil
	})
	metrics := newScheduledProfileMetrics(monitoring.NewRegistry())
	worker := newScheduledQueryProfiler(t.Context(), 2, executor, plugin, newQueryProfiler(log), metrics, d.enqueueProfile)
	t.Cleanup(worker.stop)

	// Collection must finish and queue the profile while the result output is stalled.
	d.enqueue(t.Context(), nil, plugin, QueryResult{Name: "scheduled", Action: "diff", UnixTime: 123}, worker)
	waitForResultPublication(t, &publisher.stalledResultPublisher)
	assert.Eventually(t, func() bool { items, _ := d.queue.stats(); return items == 1 }, time.Second, time.Millisecond,
		"profile collection waited for the blocked result publisher")
	assert.Equal(t, uint64(1), metrics.collected.Get(), "count the collected profile")

	// After the output recovers, the profile keeps the result's correlation.
	release()
	var response scheduledProfileResponse
	select {
	case response = <-publisher.response:
	case <-time.After(time.Second):
		t.Fatal("the stalled result's response was not published")
	}
	select {
	case profile := <-publisher.profile:
		assert.Equal(t, response.responseID, profile.responseID, "independent workers must retain the same response identity")
		assert.Equal(t, "policy-schedule", profile.scheduleID, "capture policy identity before deferred work")
		assert.Equal(t, response.executionCount, profile.scheduleExecutionCount, "retain the originating execution slot")
	case <-time.After(time.Second):
		t.Fatal("the queued profile was not published after the output recovered")
	}
}
