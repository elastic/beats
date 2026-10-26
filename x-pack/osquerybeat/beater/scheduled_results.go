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
	lease      *scheduledResultLease
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

func (d *scheduledResultDispatcher) configure(inputs []config.InputConfig) error {
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
		next.publisher = d.newPublisher(next.allowed, func() { d.metrics.drop(d.metrics.outputFull) })
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
	if next != nil && !d.started {
		d.started = true
		go d.run()
	}
	d.mu.Unlock()
	if previous != nil {
		previous.retire()
	}
	return nil
}

func (d *scheduledResultDispatcher) enqueue(ctx context.Context, client scheduledQueryClient, plugin *ConfigPlugin, result QueryResult, profiler *scheduledQueryProfiler) {
	namespace, query, ok := plugin.lookupResultMetadata(result.Name)
	if !ok {
		d.metrics.drop(d.metrics.stale)
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed || ctx.Err() != nil {
		d.metrics.drop(d.metrics.shutdown)
		return
	}
	if d.lease == nil {
		d.metrics.drop(d.metrics.stale)
		return
	}
	expectedNamespace, expectedQuery, ok := d.lease.view.lookupResultMetadata(result.Name)
	if !ok || namespace != expectedNamespace || !reflect.DeepEqual(query, expectedQuery) {
		d.metrics.drop(d.metrics.stale)
		return
	}
	responseID := uuid.Must(uuid.NewV4()).String()
	d.queue.push(scheduledResult{ctx: ctx, client: client, result: result, namespace: namespace, query: query, lease: d.lease, queuedAt: time.Now(), responseID: responseID})
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
	lease := job.lease
	lease.mu.Lock()
	defer func() {
		lease.ctx = nil
		if lease.retired.Load() {
			lease.closeOnce.Do(lease.publisher.Close)
		}
		lease.mu.Unlock()
	}()
	ctx, cancel := context.WithCancel(job.ctx)
	stop := context.AfterFunc(d.ctx, cancel)
	if d.ctx.Err() != nil {
		cancel()
	}
	defer func() { stop(); cancel() }()
	lease.ctx = ctx
	if !lease.allowed() {
		d.metrics.drop(d.metrics.stale)
		return
	}
	d.handle(ctx, job.client, lease.publisher, job.namespace, job.query, job.result, job.responseID)
}

// prune releases cancelled runs and retired policies even while the active
// publication is blocked.
func (d *scheduledResultDispatcher) prune() {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, job := range d.queue.drain() {
		if job.ctx.Err() != nil || job.lease.retired.Load() {
			d.metrics.drop(d.metrics.stale)
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
	overflow, stale, shutdown, outputFull *monitoring.Uint
	overflowRows                          *monitoring.Uint
	lastWarning                           atomic.Int64
	log                                   *logp.Logger
}

func newScheduledResultMetrics(reg *monitoring.Registry, log *logp.Logger) *scheduledResultMetrics {
	reg = reg.GetOrCreateRegistry("osquerybeat.scheduled_results")
	return &scheduledResultMetrics{
		overflow:     monitoring.NewUint(reg, "dropped.overflow"),
		overflowRows: monitoring.NewUint(reg, "dropped_rows.overflow"),
		outputFull:   monitoring.NewUint(reg, "dropped.output_full"),
		stale:        monitoring.NewUint(reg, "dropped.stale"),
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
