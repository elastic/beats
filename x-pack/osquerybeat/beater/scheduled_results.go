// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"slices"
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

// scheduledItem is a native result, a resolved RRULE result or a profile
// waiting in the scheduled queue.
type scheduledItem interface {
	queuedItem
	// target returns the query name, namespace and SQL captured at admission.
	target() (name, namespace, sql string)
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

func (r scheduledResult) target() (string, string, string) {
	return r.result.Name, r.namespace, r.query.Query
}

// resolvedScheduledResult is an RRULE run. Its rows were resolved when
// osquerybeat ran the query.
type resolvedScheduledResult struct {
	name, namespace, scheduleID, responseID string
	query                                   QueryInfo
	batches                                 []resolvedRows
	startedAt, completedAt, plannedTime     time.Time
	executionCount                          int64
	profile                                 map[string]any // nil when profiling is off
	enqueuedAt                              time.Time
}

// resolvedRows are rows published with one osquery_meta.
type resolvedRows struct {
	meta map[string]any
	hits []map[string]any
}

// rows counts the result rows, the response document and the profile, if any.
func (r resolvedScheduledResult) rows() int {
	n := 1
	for _, b := range r.batches {
		n += len(b.hits)
	}
	if r.profile != nil {
		n++
	}
	return n
}

func (r resolvedScheduledResult) enqueued() time.Time { return r.enqueuedAt }

func (r resolvedScheduledResult) evictFirst() bool { return false }

func (r resolvedScheduledResult) target() (string, string, string) {
	return r.name, r.namespace, r.query.Query
}

// scheduledProfileItem is the profile of one scheduled query run, queued
// apart from the run's results.
type scheduledProfileItem struct {
	name, namespace, query, scheduleID, responseID, spaceID string
	executionCount                                          int64
	profile                                                 map[string]any
	enqueuedAt                                              time.Time
}

func (p scheduledProfileItem) rows() int { return 1 }

func (p scheduledProfileItem) enqueued() time.Time { return p.enqueuedAt }

// evictFirst reports true because profiles are best effort and worth less
// than results.
func (p scheduledProfileItem) evictFirst() bool { return true }

func (p scheduledProfileItem) target() (string, string, string) {
	return p.name, p.namespace, p.query
}

// Each policy owns separate clients. Retirement never waits behind an output
// call; the one process-lifetime worker closes an active lease when it returns.
type scheduledResultLease struct {
	closeOnce sync.Once
	mu        sync.Mutex
	retired   atomic.Bool
	publisher osquerybeatPublisher
	view      *ConfigPlugin
	profiles  bool            // whether the policy has a query profile stream
	ctx       context.Context // accessed only while the worker holds mu
	item      scheduledItem   // the item being published; accessed only while the worker holds mu
	cut       int             // rows the publisher skipped after retirement; accessed only while the worker holds mu
}

// allowed reports whether the worker may publish the rest of l.item through
// l: the dispatcher has not stopped, and l is active or current, the lease
// that replaced it, still routes the item.
func (l *scheduledResultLease) allowed(current *scheduledResultLease) bool {
	if l.ctx == nil || l.ctx.Err() != nil {
		return false
	}
	return !l.retired.Load() || current != nil && current.routes(l.item)
}

// routes reports whether l sends item's query to the item's namespace with
// the same SQL and, for a profile, has a profile stream. Items are published
// with the metadata captured at admission, so other metadata may differ.
func (l *scheduledResultLease) routes(item scheduledItem) bool {
	if _, ok := item.(scheduledProfileItem); ok && !l.profiles {
		return false
	}
	name, namespace, sql := item.target()
	ns, q, ok := l.view.lookupResultMetadata(name)
	return ok && ns == namespace && q.Query == sql
}

// unlock ends the worker's use of l, closing l if it was retired meanwhile.
func (l *scheduledResultLease) unlock() {
	l.ctx, l.item = nil, nil
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
	queue           *rowQueue[scheduledItem]
	done            chan struct{}
	started, closed bool
	lease           *scheduledResultLease
	// current mirrors lease for publishers' allowed checks, which run
	// without mu.
	current atomic.Pointer[scheduledResultLease]
	// runCtx and client belong to the running osqueryd, if any; client is
	// nil between runs. clientSet is closed while client is set.
	runCtx         context.Context
	client         scheduledQueryClient
	clientSet      chan struct{}
	maxAge         time.Duration
	log            *logp.Logger
	metrics        *publishingMetrics
	profileMetrics *scheduledProfileMetrics
	newPublisher   func(allowed func() bool, dropped func(), cut func(rows int)) osquerybeatPublisher
	profile        func(*scheduledQueryProfiler, string, QueryInfo, QueryResult, string)
	// handle resolves and publishes a native result and returns the rows it
	// could not publish because their types could not be resolved.
	handle func(context.Context, scheduledQueryClient, scheduledQueryPublisher, string, QueryInfo, QueryResult, string) int
}

func newScheduledResultDispatcher(reg *monitoring.Registry, log *logp.Logger, maxEvents int, factory func(allowed func() bool, dropped func(), cut func(rows int)) osquerybeatPublisher,
	handle func(context.Context, scheduledQueryClient, scheduledQueryPublisher, string, QueryInfo, QueryResult, string) int) *scheduledResultDispatcher {
	if reg == nil {
		reg = monitoring.NewRegistry()
	}
	ctx, cancel := context.WithCancel(context.Background())
	metrics := newPublishingMetrics(reg, "scheduled", log)
	profileMetrics := newScheduledProfileMetrics(reg)
	queue := newRowQueue(maxEvents, func(item scheduledItem) {
		metrics.drop(dropOverflow, item.rows())
		countProfileDrop(item, profileMetrics.outputFull)
	})
	queue.observe = metrics.setQueue
	return &scheduledResultDispatcher{
		ctx: ctx, cancel: cancel, done: make(chan struct{}), clientSet: make(chan struct{}), log: log,
		queue: queue, metrics: metrics, profileMetrics: profileMetrics, newPublisher: factory, handle: handle,
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
		next = &scheduledResultLease{view: view, profiles: slices.ContainsFunc(inputs, func(in config.InputConfig) bool {
			return in.Datastream.Dataset == config.DefaultQueryProfileDataset
		})}
		// A blocking client drops only events published after it is closed,
		// which happens when the pipeline closes on shutdown.
		next.publisher = d.newPublisher(func() bool { return next.allowed(d.current.Load()) },
			func() { d.metrics.dropRows(dropShutdown, 1) },
			func(rows int) { next.cut += rows })
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
	d.current.Store(next)
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
	job.namespace, job.query = namespace, query
	d.mu.Lock()
	defer d.mu.Unlock()
	// A result of an ended run is kept: any run's client resolves its rows.
	if d.closed {
		d.dropShutdown(job)
		return
	}
	if d.lease == nil || !d.lease.routes(job) {
		d.dropStale(job)
		return
	}
	responseID := uuid.Must(uuid.NewV4()).String()
	job.queuedAt, job.responseID = time.Now(), responseID
	d.queue.push(job)
	// Profiling keeps its own collection worker. Its admission never waits for
	// this result's publication or shares the result worker. Without a profile
	// stream, skip the profile RPC.
	if d.profile != nil && d.lease.profiles {
		d.profile(profiler, namespace, query, result, responseID)
	}
}

// enqueueResolved queues an RRULE run. It never waits.
func (d *scheduledResultDispatcher) enqueueResolved(item resolvedScheduledResult) {
	item.enqueuedAt = time.Now()
	d.admit(item)
}

// enqueueProfile queues a collected profile. It never waits.
func (d *scheduledResultDispatcher) enqueueProfile(item scheduledProfileItem) {
	item.enqueuedAt = time.Now()
	d.admit(item)
}

func (d *scheduledResultDispatcher) admit(item scheduledItem) {
	d.mu.Lock()
	defer d.mu.Unlock()
	switch {
	case d.closed:
		d.dropShutdown(item)
	case d.lease == nil || !d.lease.routes(item):
		d.dropStale(item)
	default:
		d.queue.push(item)
	}
}

func (d *scheduledResultDispatcher) run() {
	defer close(d.done)
	for {
		item, ok := d.queue.pop(d.ctx)
		if !ok {
			return
		}
		d.process(item)
	}
}

func (d *scheduledResultDispatcher) process(item scheduledItem) {
	d.mu.Lock()
	lease, maxAge := d.lease, d.maxAge
	expired := maxAge > 0 && time.Since(item.enqueued()) > maxAge
	routed := lease != nil && lease.routes(item)
	runCtx, client, ok := d.runFor(item)
	if !expired && routed && !ok {
		// Keep item, as the oldest, until a run can resolve its rows. A
		// routing lease means stop has not drained the queue yet.
		set := d.clientSet
		d.queue.requeue(item)
		d.mu.Unlock()
		select {
		case <-set:
		case <-d.ctx.Done():
		}
		return
	}
	if lease != nil {
		// Lock while the lease is still active, so it cannot have been closed;
		// a later retire leaves closing it to unlock.
		lease.mu.Lock()
		defer lease.unlock()
	}
	d.mu.Unlock()
	switch {
	case expired:
		d.metrics.drop(dropExpired, item.rows())
		countProfileDrop(item, d.profileMetrics.expired)
	case !routed:
		d.dropStale(item)
	default:
		d.publish(lease, item, runCtx, client)
	}
}

// runFor returns the osqueryd run whose client resolves item's rows: the run
// that produced item while it lasts, else the current run. Column types depend
// only on the SQL, so any run's client resolves them. ok is false when item
// needs a client and no run is current. The caller holds d.mu.
func (d *scheduledResultDispatcher) runFor(item scheduledItem) (ctx context.Context, client scheduledQueryClient, ok bool) {
	r, native := item.(scheduledResult)
	switch {
	case !native:
		return context.Background(), nil, true
	case r.ctx.Err() == nil:
		return r.ctx, r.client, true
	case d.client != nil:
		return d.runCtx, d.client, true
	default:
		return nil, nil, false
	}
}

// publish sends item through lease, whose mu the caller holds. runCtx and
// client resolve a native result's rows.
func (d *scheduledResultDispatcher) publish(lease *scheduledResultLease, item scheduledItem, runCtx context.Context, client scheduledQueryClient) {
	// Only resolution depends on the run; rows already resolved keep
	// publishing after it ends.
	ctx, cancel := context.WithCancel(runCtx)
	stop := context.AfterFunc(d.ctx, cancel)
	if d.ctx.Err() != nil {
		cancel()
	}
	defer func() { stop(); cancel() }()
	lease.ctx, lease.item = d.ctx, item
	if !lease.allowed(d.current.Load()) {
		d.dropCut(item, item.rows())
		return
	}
	unresolved := 0
	switch item := item.(type) {
	case scheduledResult:
		unresolved = d.handle(ctx, client, lease.publisher, item.namespace, item.query, item.result, item.responseID)
	case resolvedScheduledResult:
		publishResolvedResult(lease.publisher, item)
	case scheduledProfileItem:
		lease.publisher.PublishScheduledProfile(config.QueryProfileDatastream(item.namespace), item.name, item.scheduleID, item.responseID, item.spaceID, item.executionCount, item.profile)
	}
	if rows := lease.cut + unresolved; rows > 0 {
		d.dropCut(item, rows)
	}
	lease.cut = 0
}

// dropCut counts the rows of item that were not published because its lease
// was retired, the dispatcher stopped, or its rows could not be resolved while
// the item was in flight.
func (d *scheduledResultDispatcher) dropCut(item scheduledItem, rows int) {
	if d.ctx.Err() != nil {
		d.metrics.drop(dropShutdown, rows)
		countProfileDrop(item, d.profileMetrics.shutdown)
		return
	}
	d.metrics.drop(dropStale, rows)
	countProfileDrop(item, d.profileMetrics.stale)
}

// publishResolvedResult publishes an RRULE run's profile, rows and response.
func publishResolvedResult(p scheduledQueryPublisher, r resolvedScheduledResult) {
	qi := r.query
	if r.profile != nil {
		p.PublishScheduledProfile(config.QueryProfileDatastream(r.namespace), r.name, r.scheduleID, r.responseID, qi.SpaceID, r.executionCount, r.profile)
	}
	count := 0
	for _, b := range r.batches {
		count += len(b.hits)
		p.Publish(config.Datastream(r.namespace), r.scheduleID, "schedule_id", r.responseID, qi.SpaceID, qi.PackID, qi.PackName, qi.QueryName, b.meta, b.hits, qi.ECSMapping, nil)
	}
	p.PublishScheduledResponse(r.scheduleID, qi.PackID, qi.PackName, qi.QueryName, qi.SpaceID, r.responseID, r.startedAt, r.completedAt, r.plannedTime, count, r.executionCount)
}

func (d *scheduledResultDispatcher) dropStale(item scheduledItem) {
	d.metrics.drop(dropStale, item.rows())
	countProfileDrop(item, d.profileMetrics.stale)
}

func (d *scheduledResultDispatcher) dropShutdown(item scheduledItem) {
	d.metrics.drop(dropShutdown, item.rows())
	countProfileDrop(item, d.profileMetrics.shutdown)
}

// countProfileDrop counts a dropped profile item in counter too. Its
// osquerybeat.scheduled_query_profiles registry is read by existing dashboards.
func countProfileDrop(item scheduledItem, counter *monitoring.Uint) {
	if _, ok := item.(scheduledProfileItem); ok {
		counter.Inc()
	}
}

// prune releases queued items the active policy no longer routes, even while
// the active publication is blocked. Items that still route are kept for the
// next osqueryd run.
func (d *scheduledResultDispatcher) prune() {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, item := range d.queue.drain() {
		if d.lease == nil || !d.lease.routes(item) {
			d.dropStale(item)
			continue
		}
		d.queue.push(item)
	}
}

// setClient makes client, of the osqueryd run with context ctx, resolve the
// rows of results whose own run has ended, and releases the worker if it
// waits for a run.
func (d *scheduledResultDispatcher) setClient(ctx context.Context, client scheduledQueryClient) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.runCtx, d.client = ctx, client
	select {
	case <-d.clientSet:
	default:
		close(d.clientSet)
	}
}

// clearClient ends the use of client set by setClient once its run ends.
// Results then wait in the queue for the next run.
func (d *scheduledResultDispatcher) clearClient(client scheduledQueryClient) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.client != client {
		return
	}
	d.runCtx, d.client = nil, nil
	d.clientSet = make(chan struct{})
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
	d.current.Store(nil)
	for _, item := range d.queue.drain() {
		d.dropShutdown(item)
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
