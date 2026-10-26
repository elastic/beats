// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gofrs/uuid/v5"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

type scheduledQueryClient interface {
	ResolveResult(context.Context, string, []map[string]string) ([]map[string]any, error)
}

type scheduledResult struct {
	ctx        context.Context
	client     scheduledQueryClient
	result     QueryResult
	namespace  string
	query      QueryInfo
	queuedAt   time.Time
	responseID string
}

// rows counts the result rows plus the response document.
func (r scheduledResult) rows() int {
	return len(r.result.Hits) + len(r.result.DiffResults.Added) + len(r.result.DiffResults.Removed) + 1
}

func (r scheduledResult) enqueued() time.Time { return r.queuedAt }

func (r scheduledResult) evictFirst() bool { return false }

// Each policy owns separate clients. Retirement never waits behind an output
// call; the one process-lifetime worker closes an active lease when it returns.
type scheduledResultLease struct {
	closeOnce sync.Once
	mu        sync.Mutex
	retired   atomic.Bool
	publisher osquerybeatPublisher
	view      *ConfigPlugin
	ctx       context.Context // accessed only while the worker holds mu
}

func (l *scheduledResultLease) allowed() bool {
	return !l.retired.Load() && l.ctx != nil && l.ctx.Err() == nil
}

// routes reports whether l still sends job's query to the namespace and SQL
// that job was captured with.
func (l *scheduledResultLease) routes(job scheduledResult) bool {
	namespace, query, ok := l.view.lookupResultMetadata(job.result.Name)
	return ok && namespace == job.namespace && query.Query == job.query.Query
}

// unlock ends the worker's use of l, closing l if it was retired meanwhile.
func (l *scheduledResultLease) unlock() {
	l.ctx = nil
	if l.retired.Load() {
		l.closeOnce.Do(l.publisher.Close)
	}
	l.mu.Unlock()
}

func (l *scheduledResultLease) retire() {
	l.retired.Store(true)
	if l.mu.TryLock() {
		l.closeOnce.Do(l.publisher.Close)
		l.mu.Unlock()
	}
}

type scheduledResultDispatcher struct {
	mu              sync.Mutex
	ctx             context.Context
	cancel          context.CancelFunc
	queue           *rowQueue[scheduledResult]
	done            chan struct{}
	started, closed bool
	lease           *scheduledResultLease
	maxAge          time.Duration
	log             *logp.Logger
	metrics         *scheduledResultMetrics
	newPublisher    func(func() bool, func()) osquerybeatPublisher
	profile         func(*scheduledQueryProfiler, string, QueryInfo, QueryResult, string)
	handle          func(context.Context, scheduledQueryClient, scheduledQueryPublisher, string, QueryInfo, QueryResult, string)
}

func newScheduledResultDispatcher(reg *monitoring.Registry, log *logp.Logger, maxEvents int, factory func(func() bool, func()) osquerybeatPublisher,
	handle func(context.Context, scheduledQueryClient, scheduledQueryPublisher, string, QueryInfo, QueryResult, string)) *scheduledResultDispatcher {
	if reg == nil {
		reg = monitoring.NewRegistry()
	}
	ctx, cancel := context.WithCancel(context.Background())
	metrics := newScheduledResultMetrics(reg, log)
	return &scheduledResultDispatcher{
		ctx: ctx, cancel: cancel, done: make(chan struct{}), log: log,
		queue: newRowQueue(maxEvents, func(r scheduledResult) {
			metrics.dropRows(metrics.overflow, metrics.overflowRows, r.rows())
		}),
		metrics: metrics, newPublisher: factory, handle: handle,
	}
}

// configure replaces the active policy and queue limits. Queued results are
// kept; the worker binds each to the policy that is active when it is popped.
func (d *scheduledResultDispatcher) configure(inputs []config.InputConfig, cfg config.ScheduledPublishingConfig) error {
	var next *scheduledResultLease
	if config.ResultInput(inputs) != nil {
		view := NewConfigPlugin(d.log)
		if err := view.Set(inputs); err != nil {
			return err
		}
		if _, err := view.GenerateConfig(d.ctx); err != nil {
			return err
		}
		next = &scheduledResultLease{view: view}
		// A blocking client drops only events published after it is closed.
		next.publisher = d.newPublisher(next.allowed, func() { d.metrics.drop(d.metrics.shutdown) })
		if err := next.publisher.Configure(inputs); err != nil {
			next.retire()
			return err
		}
	}
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		if next != nil {
			next.retire()
		}
		return context.Canceled
	}
	previous := d.lease
	if previous != nil {
		previous.retired.Store(true)
	}
	d.lease = next
	d.maxAge = cfg.MaxAge
	if next != nil && !d.started {
		d.started = true
		go d.run()
	}
	d.mu.Unlock()
	if previous != nil {
		previous.retire()
	}
	d.queue.setLimit(cfg.MaxEvents)
	return nil
}

func (d *scheduledResultDispatcher) enqueue(ctx context.Context, client scheduledQueryClient, plugin *ConfigPlugin, result QueryResult, profiler *scheduledQueryProfiler) {
	job := scheduledResult{ctx: ctx, client: client, result: result}
	namespace, query, ok := plugin.lookupResultMetadata(result.Name)
	if !ok {
		d.dropStale(job)
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed || ctx.Err() != nil {
		d.metrics.drop(d.metrics.shutdown)
		return
	}
	if d.lease == nil {
		d.dropStale(job)
		return
	}
	expectedNamespace, expectedQuery, ok := d.lease.view.lookupResultMetadata(result.Name)
	if !ok || namespace != expectedNamespace || !reflect.DeepEqual(query, expectedQuery) {
		d.dropStale(job)
		return
	}
	responseID := uuid.Must(uuid.NewV4()).String()
	job.namespace, job.query, job.queuedAt, job.responseID = namespace, query, time.Now(), responseID
	d.queue.push(job)
	// Profiling keeps its own collection worker. Its admission never waits for
	// this result's publication or shares the result worker.
	if d.profile != nil {
		d.profile(profiler, namespace, query, result, responseID)
	}
}

func (d *scheduledResultDispatcher) run() {
	defer close(d.done)
	for {
		job, ok := d.queue.pop(d.ctx)
		if !ok {
			return
		}
		d.process(job)
	}
}

func (d *scheduledResultDispatcher) process(job scheduledResult) {
	d.mu.Lock()
	lease, maxAge := d.lease, d.maxAge
	if lease != nil {
		// Lock while the lease is still active, so it cannot have been closed;
		// a later retire leaves closing it to unlock.
		lease.mu.Lock()
		defer lease.unlock()
	}
	d.mu.Unlock()
	switch {
	case maxAge > 0 && time.Since(job.queuedAt) > maxAge:
		d.metrics.dropRows(d.metrics.expired, d.metrics.expiredRows, job.rows())
	case lease == nil || !lease.routes(job):
		d.dropStale(job)
	default:
		d.publish(lease, job)
	}
}

// publish sends job through lease, whose mu the caller holds.
func (d *scheduledResultDispatcher) publish(lease *scheduledResultLease, job scheduledResult) {
	ctx, cancel := context.WithCancel(job.ctx)
	stop := context.AfterFunc(d.ctx, cancel)
	if d.ctx.Err() != nil {
		cancel()
	}
	defer func() { stop(); cancel() }()
	lease.ctx = ctx
	if !lease.allowed() {
		d.dropStale(job)
		return
	}
	d.handle(ctx, job.client, lease.publisher, job.namespace, job.query, job.result, job.responseID)
}

func (d *scheduledResultDispatcher) dropStale(job scheduledResult) {
	d.metrics.dropRows(d.metrics.stale, d.metrics.staleRows, job.rows())
}

// prune releases results of cancelled daemon runs even while the active
// publication is blocked.
func (d *scheduledResultDispatcher) prune() {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, job := range d.queue.drain() {
		if job.ctx.Err() != nil {
			d.dropStale(job)
			continue
		}
		d.queue.push(job)
	}
}

func (d *scheduledResultDispatcher) stop() {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return
	}
	d.closed = true
	d.cancel()
	lease := d.lease
	if lease != nil {
		lease.retired.Store(true)
	}
	d.lease = nil
	for range d.queue.drain() {
		d.metrics.drop(d.metrics.shutdown)
	}
	if !d.started {
		close(d.done)
	}
	d.mu.Unlock()
	if lease != nil {
		lease.retire()
	}
	// The existing shared pipeline closes after the beater returns. Do not
	// join a worker that it must release from beat.Client.Publish first.
}

type scheduledResultMetrics struct {
	overflow, stale, expired, shutdown   *monitoring.Uint
	overflowRows, staleRows, expiredRows *monitoring.Uint
	lastWarning                          atomic.Int64
	log                                  *logp.Logger
}

func newScheduledResultMetrics(reg *monitoring.Registry, log *logp.Logger) *scheduledResultMetrics {
	reg = reg.GetOrCreateRegistry("osquerybeat.scheduled_results")
	return &scheduledResultMetrics{
		overflow:     monitoring.NewUint(reg, "dropped.overflow"),
		overflowRows: monitoring.NewUint(reg, "dropped_rows.overflow"),
		stale:        monitoring.NewUint(reg, "dropped.stale"),
		staleRows:    monitoring.NewUint(reg, "dropped_rows.stale"),
		expired:      monitoring.NewUint(reg, "dropped.expired"),
		expiredRows:  monitoring.NewUint(reg, "dropped_rows.expired"),
		shutdown:     monitoring.NewUint(reg, "dropped.shutdown"),
		log:          log,
	}
}

// dropRows counts one dropped document in counter and its rows in rowCounter.
func (m *scheduledResultMetrics) dropRows(counter, rowCounter *monitoring.Uint, rows int) {
	rowCounter.Add(uint64(rows))
	m.drop(counter)
}

func (m *scheduledResultMetrics) drop(counter *monitoring.Uint) {
	counter.Inc()
	now := time.Now().UnixNano()
	last := m.lastWarning.Load()
	if now-last >= int64(time.Minute) && m.lastWarning.CompareAndSwap(last, now) {
		m.log.Warn("Scheduled osquery result dropped; see osquerybeat.scheduled_results.dropped monitoring counters")
	}
}
