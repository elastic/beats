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

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

const (
	scheduledResultLimit      = 128
	scheduledResultBytesLimit = 8 << 20
)

type scheduledQueryClient interface {
	queryExecutor
	ResolveResult(context.Context, string, []map[string]string) ([]map[string]any, error)
}

type scheduledResult struct {
	ctx       context.Context
	client    scheduledQueryClient
	result    QueryResult
	namespace string
	query     QueryInfo
	lease     *scheduledResultLease
	bytes     int
}

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
	pending         chan scheduledResult
	wake            chan struct{}
	done            chan struct{}
	started, closed bool
	count, bytes    int
	lease           *scheduledResultLease
	log             *logp.Logger
	metrics         *scheduledResultMetrics
	newPublisher    func(func() bool, func()) osquerybeatPublisher
	handle          func(context.Context, scheduledQueryClient, scheduledQueryPublisher, string, QueryInfo, QueryResult)
}

func newScheduledResultDispatcher(reg *monitoring.Registry, log *logp.Logger, factory func(func() bool, func()) osquerybeatPublisher,
	handle func(context.Context, scheduledQueryClient, scheduledQueryPublisher, string, QueryInfo, QueryResult)) *scheduledResultDispatcher {
	if reg == nil {
		reg = monitoring.NewRegistry()
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &scheduledResultDispatcher{
		ctx: ctx, cancel: cancel, pending: make(chan scheduledResult, scheduledResultLimit),
		wake: make(chan struct{}, 1), done: make(chan struct{}), log: log,
		metrics: newScheduledResultMetrics(reg, log), newPublisher: factory, handle: handle,
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

func (d *scheduledResultDispatcher) enqueue(ctx context.Context, client scheduledQueryClient, plugin *ConfigPlugin, result QueryResult) {
	namespace, query, ok := plugin.lookupResultMetadata(result.Name)
	if !ok {
		d.metrics.drop(d.metrics.stale)
		return
	}
	bytes := resultSize(result)
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
	if d.count >= scheduledResultLimit || bytes > scheduledResultBytesLimit-d.bytes {
		d.metrics.drop(d.metrics.overflow)
		return
	}
	d.count++
	d.bytes += bytes
	d.pending <- scheduledResult{ctx: ctx, client: client, result: result, namespace: namespace, query: query, lease: d.lease, bytes: bytes}
	select {
	case d.wake <- struct{}{}:
	default:
	}
}

func (d *scheduledResultDispatcher) run() {
	defer close(d.done)
	for {
		select {
		case <-d.ctx.Done():
			return
		case <-d.wake:
		}
		for {
			d.mu.Lock()
			var job scheduledResult
			select {
			case job = <-d.pending:
			default:
				d.mu.Unlock()
				goto drained
			}
			d.mu.Unlock()
			d.process(job)
			d.mu.Lock()
			d.count--
			d.bytes -= job.bytes
			d.mu.Unlock()
		}
	drained:
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
	d.handle(ctx, job.client, lease.publisher, job.namespace, job.query, job.result)
}

// prune releases cancelled runs and retired policies even while the active
// publication is blocked. Both the outstanding count and byte bound include it.
func (d *scheduledResultDispatcher) prune() {
	d.mu.Lock()
	defer d.mu.Unlock()
	for range len(d.pending) {
		job := <-d.pending
		if job.ctx.Err() != nil || job.lease.retired.Load() {
			d.count--
			d.bytes -= job.bytes
			d.metrics.drop(d.metrics.stale)
		} else {
			d.pending <- job
		}
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
	for len(d.pending) > 0 {
		job := <-d.pending
		d.count--
		d.bytes -= job.bytes
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
	lastWarning                           atomic.Int64
	log                                   *logp.Logger
}

func newScheduledResultMetrics(reg *monitoring.Registry, log *logp.Logger) *scheduledResultMetrics {
	reg = reg.GetOrCreateRegistry("osquerybeat.scheduled_results")
	return &scheduledResultMetrics{
		overflow:   monitoring.NewUint(reg, "dropped.overflow"),
		outputFull: monitoring.NewUint(reg, "dropped.output_full"),
		stale:      monitoring.NewUint(reg, "dropped.stale"),
		shutdown:   monitoring.NewUint(reg, "dropped.shutdown"),
		log:        log,
	}
}

func (m *scheduledResultMetrics) drop(counter *monitoring.Uint) {
	counter.Inc()
	now := time.Now().UnixNano()
	last := m.lastWarning.Load()
	if now-last >= int64(time.Minute) && m.lastWarning.CompareAndSwap(last, now) {
		m.log.Warn("Scheduled osquery result dropped; see osquerybeat.scheduled_results.dropped monitoring counters")
	}
}

// resultSize estimates retained decoded storage, including map and row overhead,
// rather than treating a result document with millions of tiny fields as cheap.
func resultSize(res QueryResult) int {
	size := 256 + len(res.Name) + len(res.Action) + len(res.CalendarTime)
	for _, rows := range [][]map[string]string{res.Hits, res.DiffResults.Added, res.DiffResults.Removed} {
		size += 24 * len(rows)
		for _, row := range rows {
			size += 128
			for key, value := range row {
				size += 128 + len(key) + len(value)
			}
		}
	}
	return size
}
