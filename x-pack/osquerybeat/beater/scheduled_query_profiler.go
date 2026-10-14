// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/semaphore"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/osqdcli"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/pub"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

const scheduledProfilePendingJobs = 64

type scheduledProfileRequest struct {
	name, query, namespace, responseID, spaceID string
	sink                                        pub.ScheduledProfileSink
}

type scheduledProfileMetrics struct {
	queueFull, stale, shutdown, failed, collected *monitoring.Uint
}

func newScheduledProfileMetrics(reg *monitoring.Registry) *scheduledProfileMetrics {
	if reg == nil {
		reg = monitoring.NewRegistry()
	}
	reg = reg.GetOrCreateRegistry("osquerybeat.scheduled_query_profiles")
	return &scheduledProfileMetrics{
		queueFull: monitoring.NewUint(reg, "dropped.queue_full"),
		stale:     monitoring.NewUint(reg, "dropped.stale"),
		shutdown:  monitoring.NewUint(reg, "dropped.shutdown"),
		failed:    monitoring.NewUint(reg, "failed"),
		collected: monitoring.NewUint(reg, "collected"),
	}
}

// scheduledQueryProfiler owns one worker and a bounded queue of metadata for
// one osquery run. It never retains query result rows or starts per-job workers.
type scheduledQueryProfiler struct {
	ctx      context.Context
	cancel   context.CancelFunc
	jobs     chan scheduledProfileRequest
	done     chan struct{}
	mu       sync.Mutex
	stopped  bool
	lastWarn atomic.Int64
	log      *logp.Logger
	profiler *queryProfiler
	executor queryExecutor
	config   *ConfigPlugin
	metrics  *scheduledProfileMetrics
}

func newScheduledQueryProfiler(ctx context.Context, limit int, executor queryExecutor, plugin *ConfigPlugin, profiler *queryProfiler, metrics *scheduledProfileMetrics) *scheduledQueryProfiler {
	ctx, cancel := context.WithCancel(ctx)
	w := &scheduledQueryProfiler{
		ctx: ctx, cancel: cancel, jobs: make(chan scheduledProfileRequest, limit), done: make(chan struct{}),
		log: profiler.log, profiler: profiler, executor: executor, config: plugin, metrics: metrics,
	}
	go w.run()
	return w
}

func (w *scheduledQueryProfiler) enqueue(request scheduledProfileRequest) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.stopped || w.ctx.Err() != nil {
		w.metrics.shutdown.Inc()
		return false
	}
	if request.sink == nil || !request.sink.Valid() || !w.config.scheduledProfileMatches(request) {
		w.metrics.stale.Inc()
		return false
	}
	select {
	case w.jobs <- request:
		return true
	default:
		w.metrics.queueFull.Inc()
		now := time.Now().UnixNano()
		previous := w.lastWarn.Load()
		if now-previous >= int64(time.Minute) && w.lastWarn.CompareAndSwap(previous, now) {
			w.log.Warn("Scheduled query profile dropped because the profiling queue is full; see osquerybeat.scheduled_query_profiles.dropped.queue_full")
		}
		return false
	}
}

func (w *scheduledQueryProfiler) run() {
	defer close(w.done)
	defer func() {
		w.mu.Lock()
		defer w.mu.Unlock()
		w.stopped = true
		for {
			select {
			case <-w.jobs:
				w.metrics.shutdown.Inc()
			default:
				return
			}
		}
	}()
	for {
		select {
		case <-w.ctx.Done():
			return
		case request := <-w.jobs:
			w.collect(request)
		}
	}
}

func (w *scheduledQueryProfiler) collect(request scheduledProfileRequest) {
	if w.ctx.Err() != nil {
		w.metrics.shutdown.Inc()
		return
	}
	if !request.sink.Valid() || !w.config.scheduledProfileMatches(request) {
		w.metrics.stale.Inc()
		return
	}
	profile, err := w.profiler.profileScheduledQuery(w.ctx, w.executor, request.name)
	if w.ctx.Err() != nil {
		w.metrics.shutdown.Inc()
		return
	}
	if err != nil {
		w.metrics.failed.Inc()
		w.log.Debugf("failed to collect scheduled query profile for %s: %v", request.name, err)
		return
	}
	// The native schedule can replace a name between queueing and collection.
	// Never correlate the replacement query's metrics with the old response.
	if profile["query"] != request.query || !w.config.scheduledProfileMatches(request) {
		w.metrics.stale.Inc()
		return
	}
	if !request.sink.Publish(request.name, request.responseID, request.spaceID, profile) {
		w.metrics.stale.Inc()
		return
	}
	// This counts collection, not output delivery. The sink's output_full
	// counter records failed best-effort queue admission separately.
	w.metrics.collected.Inc()
}

func (w *scheduledQueryProfiler) stop() {
	w.mu.Lock()
	w.stopped = true
	w.cancel()
	w.mu.Unlock()
	// Cancelling RPCs cannot interrupt beat.Client.Publish. Bounding this join
	// also requires cancellable publisher admission.
	<-w.done
}

// profilingQueryExecutor links diagnostic request deadlines to daemon teardown.
// Its client is isolated from result resolution, actions, and health monitoring.
type profilingQueryExecutor struct {
	runCtx     context.Context
	executor   queryExecutor
	connection *profilingConnection
}

// Connecting lazily keeps optional profiling failures out of daemon startup.
// A failed or cancelled attempt must allow a later request to try again.
type profilingConnection struct {
	connect   func(context.Context) error
	gate      *semaphore.Weighted
	connected atomic.Bool
}

func newProfilingQueryExecutor(ctx context.Context, client *osqdcli.Client) profilingQueryExecutor {
	return profilingQueryExecutor{
		runCtx: ctx, executor: client,
		connection: &profilingConnection{connect: client.Connect, gate: semaphore.NewWeighted(1)},
	}
}

func (c *profilingConnection) ensure(ctx context.Context) error {
	if c.connected.Load() {
		return ctx.Err()
	}
	if err := c.gate.Acquire(ctx, 1); err != nil {
		return err
	}
	defer c.gate.Release(1)
	if err := ctx.Err(); err != nil {
		return err
	}
	if !c.connected.Load() {
		if err := c.connect(ctx); err != nil {
			return err
		}
		c.connected.Store(true)
	}
	return nil
}

func (q profilingQueryExecutor) Query(ctx context.Context, sql string, timeout time.Duration) ([]map[string]any, error) {
	if err := q.runCtx.Err(); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(q.runCtx, cancel)
	defer func() { stop(); cancel() }()
	if q.connection != nil {
		if err := q.connection.ensure(ctx); err != nil {
			return nil, err
		}
	}
	return q.executor.Query(ctx, sql, timeout)
}
