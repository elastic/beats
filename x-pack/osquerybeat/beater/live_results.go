// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"errors"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/ecs"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

var (
	errLivePublishTimeout = errors.New("timed out publishing live query results")
	errLiveShutdown       = errors.New("live query results not published: osquerybeat is stopping")
)

// liveJob is one live query's results and, if requested, its profile.
type liveJob struct {
	index, actionID, responseID, spaceID string
	hits                                 []map[string]any
	ecsm                                 ecs.Mapping
	reqData                              any
	profileIndex                         string
	profile                              map[string]any // nil when not requested
}

// liveItem is a queued live job, or an action response when pub is nil.
type liveItem struct {
	job       liveJob
	pub       actionQueryPublisher
	handed    atomic.Int64  // rows of job handed to Publish so far
	cancelled atomic.Bool   // set when the waiting action gives up
	done      chan struct{} // closed when the worker is finished with job

	resPub   actionResultPublisher
	req, res map[string]any
}

// liveResultDispatcher publishes live results, profiles and action responses
// in order on one worker for the lifetime of the beat. An action waits for its
// rows at most for the timeout, so a full output cannot hold it, or the
// resetableActionHandler lock, indefinitely.
type liveResultDispatcher struct {
	log      *logp.Logger
	metrics  *scheduledResultMetrics
	timeout  atomic.Int64 // time.Duration
	start    sync.Once
	wake     chan struct{}
	stopping chan struct{}

	mu     sync.Mutex
	queue  []*liveItem
	closed bool
}

func newLiveResultDispatcher(reg *monitoring.Registry, log *logp.Logger, timeout time.Duration) *liveResultDispatcher {
	if reg == nil {
		reg = monitoring.NewRegistry()
	}
	d := &liveResultDispatcher{
		log:      log,
		metrics:  newLiveResultMetrics(reg, log),
		wake:     make(chan struct{}, 1),
		stopping: make(chan struct{}),
	}
	d.setTimeout(timeout)
	return d
}

// setTimeout sets how long an action waits for its rows to be handed to the
// output, counted from when the query returns.
func (d *liveResultDispatcher) setTimeout(timeout time.Duration) {
	d.timeout.Store(int64(timeout))
}

// publishResults hands job's profile and hits to the worker and waits until
// all hits are handed to Publish, ctx is done, the timeout elapses or the
// dispatcher stops. It returns the number of hits handed over. Hits not
// handed over by then are skipped; one already inside a blocked Publish cannot
// be pulled back and may be delivered late.
func (d *liveResultDispatcher) publishResults(ctx context.Context, pub actionQueryPublisher, job liveJob) (int, error) {
	it := &liveItem{job: job, pub: pub, done: make(chan struct{})}
	if !d.push(it) {
		d.metrics.drop(d.metrics.shutdown)
		return 0, errLiveShutdown
	}

	t := time.NewTimer(time.Duration(d.timeout.Load()))
	defer t.Stop()
	var err error
	select {
	case <-it.done:
	case <-t.C:
		err = errLivePublishTimeout
	case <-ctx.Done():
		err = ctx.Err()
	case <-d.stopping:
	}
	d.abandon(it)

	n := int(it.handed.Load())
	switch {
	case n == len(job.hits):
		return n, nil
	case err == nil:
		// The worker gave up, or has not started, because the dispatcher stopped.
		d.metrics.drop(d.metrics.shutdown)
		return n, errLiveShutdown
	default:
		d.metrics.dropRows(d.metrics.timeout, d.metrics.timeoutRows, len(job.hits)-n)
		return n, err
	}
}

// enqueueResponse queues an action response after the action's rows. It never
// waits. Only stop drops a response.
func (d *liveResultDispatcher) enqueueResponse(pub actionResultPublisher, req, res map[string]any) {
	if !d.push(&liveItem{resPub: pub, req: req, res: res}) {
		d.metrics.drop(d.metrics.shutdown)
	}
}

func (d *liveResultDispatcher) push(it *liveItem) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return false
	}
	d.start.Do(func() { go d.run() })
	d.queue = append(d.queue, it)
	select {
	case d.wake <- struct{}{}:
	default:
	}
	return true
}

// abandon tells the worker to skip its remaining rows, and drops it if the
// worker has not reached it.
func (d *liveResultDispatcher) abandon(it *liveItem) {
	it.cancelled.Store(true)
	d.mu.Lock()
	defer d.mu.Unlock()
	d.queue = slices.DeleteFunc(d.queue, func(q *liveItem) bool { return q == it })
}

func (d *liveResultDispatcher) run() {
	for {
		it, ok := d.next()
		if !ok {
			return
		}
		d.publish(it)
	}
}

func (d *liveResultDispatcher) next() (*liveItem, bool) {
	for {
		d.mu.Lock()
		if d.closed {
			d.mu.Unlock()
			return nil, false
		}
		if len(d.queue) > 0 {
			it := d.queue[0]
			d.queue[0] = nil
			d.queue = d.queue[1:]
			d.mu.Unlock()
			return it, true
		}
		d.mu.Unlock()
		select {
		case <-d.wake:
		case <-d.stopping:
		}
	}
}

func (d *liveResultDispatcher) publish(it *liveItem) {
	if it.pub == nil {
		it.resPub.PublishActionResult(it.req, it.res)
		return
	}
	defer close(it.done)
	j := it.job
	if j.profile != nil && !d.abandoned(it) {
		it.pub.PublishQueryProfile(j.profileIndex, "", j.actionID, j.responseID, j.spaceID, j.profile, j.reqData)
	}
	// stop runs before each hit, so its call count is the number of hits
	// already handed to Publish; the waiting action reads it on timeout.
	handed := 0
	stop := func() bool {
		it.handed.Store(int64(handed))
		handed++
		return d.abandoned(it)
	}
	n := it.pub.PublishEach(j.index, j.actionID, "action_id", j.responseID, j.spaceID, "", "", "", nil, j.hits, j.ecsm, j.reqData, stop)
	it.handed.Store(int64(n))
}

func (d *liveResultDispatcher) abandoned(it *liveItem) bool {
	if it.cancelled.Load() {
		return true
	}
	select {
	case <-d.stopping:
		return true
	default:
		return false
	}
}

// stop drops queued action responses and releases waiting actions, which
// count their own results. It does not join a worker blocked in Publish; the
// pipeline closing its queue releases it.
func (d *liveResultDispatcher) stop() {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return
	}
	d.closed = true
	close(d.stopping)
	queued := d.queue
	d.queue = nil
	d.mu.Unlock()
	for _, it := range queued {
		if it.pub == nil {
			d.metrics.drop(d.metrics.shutdown)
		}
	}
}

func newLiveResultMetrics(reg *monitoring.Registry, log *logp.Logger) *scheduledResultMetrics {
	reg = reg.GetOrCreateRegistry("osquerybeat.live_results")
	return &scheduledResultMetrics{
		timeout:     monitoring.NewUint(reg, "dropped.timeout"),
		timeoutRows: monitoring.NewUint(reg, "dropped_rows.timeout"),
		shutdown:    monitoring.NewUint(reg, "dropped.shutdown"),
		log:         log,
		warning:     "Live osquery result dropped; see osquerybeat.live_results.dropped monitoring counters",
	}
}

// liveQueryPublisher publishes live results, profiles and action responses.
type liveQueryPublisher interface {
	actionQueryPublisher
	actionResultPublisher
	Configure(inputs []config.InputConfig) error
	Close()
}

// liveClients gives the live worker clients of its own. A policy change swaps
// in new clients; the old ones are closed once no publish is in flight,
// because beat.Client.Close waits for an in-flight Publish.
type liveClients struct {
	newPublisher func() liveQueryPublisher

	mu      sync.Mutex
	current *liveLease
}

var _ liveQueryPublisher = (*liveClients)(nil)

// Configure connects clients for inputs and retires the previous ones. With
// no inputs it keeps the current clients, as pub.Publisher.Configure does.
func (c *liveClients) Configure(inputs []config.InputConfig) error {
	if len(inputs) == 0 {
		return nil
	}
	p := c.newPublisher()
	if err := p.Configure(inputs); err != nil {
		p.Close()
		return err
	}
	c.swap(&liveLease{pub: p})
	return nil
}

// Close retires the current clients without waiting for a stuck publish.
func (c *liveClients) Close() {
	c.swap(nil)
}

func (c *liveClients) swap(next *liveLease) {
	c.mu.Lock()
	previous := c.current
	c.current = next
	c.mu.Unlock()
	if previous != nil {
		previous.retire()
	}
}

func (c *liveClients) PublishEach(index, idValue, idFieldKey, responseID, spaceID, packID, packName, queryName string, meta map[string]any, hits []map[string]any, ecsm ecs.Mapping, reqData any, stop func() bool) int {
	l := c.acquire()
	if l == nil {
		return 0
	}
	defer l.release()
	return l.pub.PublishEach(index, idValue, idFieldKey, responseID, spaceID, packID, packName, queryName, meta, hits, ecsm, reqData, stop)
}

func (c *liveClients) PublishQueryProfile(index, queryName, actionID, responseID, spaceID string, profile map[string]any, reqData any) {
	l := c.acquire()
	if l == nil {
		return
	}
	defer l.release()
	l.pub.PublishQueryProfile(index, queryName, actionID, responseID, spaceID, profile, reqData)
}

func (c *liveClients) PublishActionResult(req, res map[string]any) {
	l := c.acquire()
	if l == nil {
		return
	}
	defer l.release()
	l.pub.PublishActionResult(req, res)
}

// acquire returns the current lease, locked for publishing, or nil when no
// clients are configured.
func (c *liveClients) acquire() *liveLease {
	c.mu.Lock()
	defer c.mu.Unlock()
	l := c.current
	if l != nil {
		// Lock while the lease is still current, so it cannot have been
		// closed; a later retire leaves closing it to release.
		l.mu.Lock()
	}
	return l
}

// liveLease is one generation of live clients.
type liveLease struct {
	mu        sync.Mutex // held while publishing
	retired   atomic.Bool
	closeOnce sync.Once
	pub       liveQueryPublisher
}

func (l *liveLease) release() {
	if l.retired.Load() {
		l.closeOnce.Do(l.pub.Close)
	}
	l.mu.Unlock()
}

func (l *liveLease) retire() {
	l.retired.Store(true)
	if l.mu.TryLock() {
		l.closeOnce.Do(l.pub.Close)
		l.mu.Unlock()
	}
}
