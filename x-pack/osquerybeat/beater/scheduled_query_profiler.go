// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/osqdcli"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

const scheduledProfilePendingJobs = 64

type scheduledProfileRequest struct {
	name, query, namespace, scheduleID, responseID, spaceID string
	scheduleExecutionCount                                  int64
}

// scheduledProfileMetrics are the osquerybeat.scheduled_query_profiles
// counters of the collection worker. Profiles dropped after collection are
// counted under osquerybeat.publishing.scheduled.
type scheduledProfileMetrics struct {
	queueFull, stale, shutdown, failed *monitoring.Uint
	// collected counts profiles collected and handed to the scheduled
	// queue, not profiles delivered.
	collected *monitoring.Uint
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
	lastWarn atomic.Int64
	log      *logp.Logger
	profiler *queryProfiler
	executor queryExecutor
	config   *ConfigPlugin
	metrics  *scheduledProfileMetrics
	// publish hands a collected profile on for publication without waiting.
	publish func(scheduledProfileItem)
}

func newScheduledQueryProfiler(ctx context.Context, limit int, executor queryExecutor, plugin *ConfigPlugin, profiler *queryProfiler, metrics *scheduledProfileMetrics, publish func(scheduledProfileItem)) *scheduledQueryProfiler {
	ctx, cancel := context.WithCancel(ctx)
	w := &scheduledQueryProfiler{
		ctx:      ctx,
		cancel:   cancel,
		jobs:     make(chan scheduledProfileRequest, limit),
		done:     make(chan struct{}),
		log:      profiler.log,
		profiler: profiler,
		executor: executor,
		config:   plugin,
		metrics:  metrics,
		publish:  publish,
	}
	go w.run()
	return w
}

func (w *scheduledQueryProfiler) enqueue(request scheduledProfileRequest) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.ctx.Err() != nil {
		w.metrics.shutdown.Inc()
		return false
	}
	if !w.config.scheduledProfileMatches(request) {
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
	for {
		select {
		case <-w.ctx.Done():
			// Serialize with enqueue so callbacks admitted before cancellation
			// are drained before stop returns.
			w.mu.Lock()
			for len(w.jobs) != 0 {
				<-w.jobs
				w.metrics.shutdown.Inc()
			}
			w.mu.Unlock()
			close(w.done)
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
	if !w.config.scheduledProfileMatches(request) {
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
	w.metrics.collected.Inc()
	w.publish(scheduledProfileItem{
		name: request.name, namespace: request.namespace, query: request.query, scheduleID: request.scheduleID,
		responseID: request.responseID, spaceID: request.spaceID, executionCount: request.scheduleExecutionCount, profile: profile,
	})
}

// stop cancels the worker and waits for it to exit. Collection never waits
// for the output, so only the cancelled RPC can delay the exit.
func (w *scheduledQueryProfiler) stop() {
	w.cancel()
	<-w.done
}

// profilingQueryExecutor links diagnostic request deadlines to daemon teardown.
// Its client is isolated from result resolution, actions, and health monitoring.
type profilingQueryExecutor struct {
	runCtx   context.Context
	executor queryExecutor
}

func newProfilingQueryExecutor(ctx context.Context, client *osqdcli.Client) profilingQueryExecutor {
	return profilingQueryExecutor{runCtx: ctx, executor: client}
}

func (q profilingQueryExecutor) Query(ctx context.Context, sql string, timeout time.Duration) ([]map[string]any, error) {
	if err := q.runCtx.Err(); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(q.runCtx, cancel)
	defer func() { stop(); cancel() }()
	return q.executor.Query(ctx, sql, timeout)
}
