// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package input_logfile

import (
	"container/heap"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	inputv2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

// permanentHarvesterError marks a harvester setup failure that should degrade
// the input's status rather than being silently retried on the next scan.
type permanentHarvesterError struct {
	err error
}

func (e permanentHarvesterError) Error() string { return e.err.Error() }
func (e permanentHarvesterError) Unwrap() error { return e.err }

func isPermanentHarvesterError(err error) bool {
	var permanentErr permanentHarvesterError
	return errors.As(err, &permanentErr)
}

const (
	minWakerBackoff = 1 * time.Second
	maxWakerBackoff = 10 * time.Second
)

// sourceStatus is the scheduling state of a source. A source is in exactly one
// status at a time (guarded by harvesterRunner.mu), which guarantees a
// single reader goroutine operates a source's session at any moment.
type sourceStatus int

const (
	statusUnset   sourceStatus = iota // zero value: created, not yet scheduled
	statusNew                         // admitted with an open slot, no reader yet
	statusRunning                     // a reader goroutine is reading it
	statusParked                      // caught up to EOF; watched by the waker
	statusPolling                     // the waker is evaluating it
	statusWaiting                     // admitted but queued: no slot, no fd, no reader
	statusClosing                     // being torn down
)

// sourceState is the passive per-source state: the open reading session plus the
// registry resource, pipeline client, cursor and cancellation. A source's file
// handle (in the session) stays open while it is being harvested, but it only
// has a goroutine while it is actively reading.
type sourceState struct {
	srcID   string
	src     Source
	session HarvesterSession

	resource  *resource
	client    beat.Client
	cursor    Cursor
	publisher *cursorPublisher
	ctx       inputv2.Context
	cancel    context.CancelFunc

	// Runner bookkeeping, guarded by harvesterRunner.mu.
	status    sourceStatus
	holdsSlot bool // occupies one of the harvesterLimit open slots
	setUp     bool // resources (lock/client/session) acquired
	isGZIP    bool // source reads a GZIP file; for the GZIP lifecycle metrics
	backoff   time.Duration
	nextCheck time.Time
	finished  bool
	done      chan struct{} // closed by finish; lets Restart wait for teardown
}

// harvesterRunner implements HarvesterGroup by spawning one short-lived
// reader goroutine per source that has data to read. A reader reads its source
// until the read would block (then parks it for the waker) or a terminal
// condition is reached (then tears it down), and exits. The waker re-spawns a
// reader when a parked source has new data. So the number of goroutines tracks
// the amount of active work, not the number of files: idle/tailing files cost no
// goroutine, while a burst of ready files gets full read concurrency.
//
// harvester_limit, when > 0, is a hard cap on the number of simultaneously open
// files: sources holding a file descriptor, whether actively reading or parked.
// A source that cannot get an open slot is queued and promoted (FIFO) when an
// open file closes, so the fd count never exceeds the limit; 0 (the default) is
// unbounded.
type harvesterRunner struct {
	pipeline     beat.PipelineConnector
	harvester    Harvester
	cleanTimeout time.Duration
	store        *store
	ackCH        *updateChan
	identifier   *SourceIdentifier
	metrics      *Metrics
	inputID      string

	// harvesterLimit, when > 0, is a hard cap on the number of simultaneously
	// open files. Sources that cannot get an open slot are queued (see waiting).
	harvesterLimit uint64

	ctx        inputv2.Context // input lifetime context, for the waker
	notifyChan chan HarvesterStatus

	// readUntilEOF, when enabled, makes StopHarvesters drain every source to EOF
	// (bounded by its Timeout) before tearing it down, instead of cancelling
	// in-flight reads. See StopHarvesters.
	readUntilEOF ReadUntilEOFConfig

	// Observability gauges (nil when no metrics registry is available).
	mActive  *monitoring.Uint // sources with a reader goroutine
	mParked  *monitoring.Uint // sources parked, watched by the waker
	mWaiting *monitoring.Uint // sources queued, waiting for an open slot

	mu     sync.Mutex
	states map[string]*sourceState
	closed bool
	// draining is set during a read_until_eof shutdown: while it is true a reader
	// that catches up to EOF tears its source down instead of parking it.
	draining bool
	// parked is a min-heap of parked sources keyed by nextCheck, so the waker
	// processes only the sources that are actually due instead of scanning every
	// source each tick. nActive/nParked track the gauge counts incrementally to
	// avoid recounting under the lock. All guarded by mu.
	parked  sourceHeap
	nActive int
	nParked int
	// nOpen counts sources holding an open slot (an open fd, reading or parked).
	// waiting is the FIFO of sources admitted but not yet opened because the
	// limit was reached; they are promoted as slots free. nWaiting mirrors the
	// live waiting count for the gauge. All guarded by mu.
	nOpen    uint64
	waiting  []*sourceState
	nWaiting int

	wakerCh chan struct{}
	wg      sync.WaitGroup
}

// parkedEntry is one scheduled poll of a source. Entries are immutable: when a
// source is re-parked it gets a fresh entry, and an entry whose due time no
// longer matches the source's nextCheck (re-parked or torn down) is stale and
// skipped when popped.
type parkedEntry struct {
	state *sourceState
	due   time.Time
}

type sourceHeap []*parkedEntry

func (h sourceHeap) Len() int           { return len(h) }
func (h sourceHeap) Less(i, j int) bool { return h[i].due.Before(h[j].due) }
func (h sourceHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *sourceHeap) Push(x any) {
	e, _ := x.(*parkedEntry)
	*h = append(*h, e)
}
func (h *sourceHeap) Pop() any {
	old := *h
	n := len(old)
	e := old[n-1]
	old[n-1] = nil
	*h = old[:n-1]
	return e
}

func newHarvesterRunner(
	ctx inputv2.Context,
	harvesterLimit uint64,
	pipeline beat.PipelineConnector,
	harvester Harvester,
	cleanTimeout time.Duration,
	store *store,
	ackCH *updateChan,
	identifier *SourceIdentifier,
	metrics *Metrics,
	inputID string,
	readUntilEOF ReadUntilEOFConfig,
) *harvesterRunner {
	g := &harvesterRunner{
		pipeline:       pipeline,
		harvester:      harvester,
		cleanTimeout:   cleanTimeout,
		store:          store,
		ackCH:          ackCH,
		identifier:     identifier,
		metrics:        metrics,
		inputID:        inputID,
		readUntilEOF:   readUntilEOF,
		harvesterLimit: harvesterLimit,
		ctx:            ctx,
		states:         map[string]*sourceState{},
		wakerCh:        make(chan struct{}, 1),
	}

	if reg := ctx.MetricsRegistry; reg != nil {
		g.mActive = monitoring.NewUint(reg, "harvester_active_readers")
		g.mParked = monitoring.NewUint(reg, "harvester_parked")
		g.mWaiting = monitoring.NewUint(reg, "harvester_waiting")
	}

	return g
}

// start launches the waker goroutine. Reader goroutines are created on demand as
// sources become ready.
func (g *harvesterRunner) start() {
	if g.harvesterLimit > 0 {
		g.ctx.Logger.Infof("starting filestream harvester (max %d open files)", g.harvesterLimit)
	} else {
		g.ctx.Logger.Info("starting filestream harvester (unlimited open files)")
	}
	g.wg.Add(1)
	go g.waker()
}

func (g *harvesterRunner) SetObserver(c chan HarvesterStatus) { g.notifyChan = c }

// run spawns a reader goroutine for state unless one is already reading it (or it is
// being torn down, or the group is shutting down). While draining (read_until_eof
// shutdown) a closed runner still spawns readers for parked sources.
func (g *harvesterRunner) run(state *sourceState) {
	g.mu.Lock()
	switch {
	// Shutting down without draining: finishRemaining sweeps remaining sources.
	case g.closed && !g.draining:
		g.mu.Unlock()
		return
	// A reader goroutine already owns state and will tear it down.
	case state.status == statusRunning:
		g.mu.Unlock()
		return
	// PollResume hand-off holder: Stop/stopAndWait may set statusClosing on a
	// polling source without finishing (see Stop); tear down here. finish() is
	// idempotent.
	case state.finished || state.status == statusClosing:
		g.mu.Unlock()
		g.finish(state)
		return
	}
	g.setStatus(state, statusRunning)
	g.wg.Add(1)
	g.mu.Unlock()

	go g.readOnce(state)
}

// readOnce reads a source until the read would block or a terminal condition is
// reached, then parks or tears it down and exits.
func (g *harvesterRunner) readOnce(state *sourceState) {
	defer g.wg.Done()

	g.mu.Lock()
	if state.status == statusClosing || state.finished {
		g.mu.Unlock()
		g.finish(state)
		return
	}
	needSetup := !state.setUp
	g.mu.Unlock()

	// First read of a source acquires its lock, client and session. Subsequent
	// reads (after a park/resume) reuse them, so a tailing file keeps its fd open.
	if needSetup {
		state.ctx.Logger.Debug("Starting harvester for file")
		if err := g.setup(state); err != nil {
			state.ctx.Logger.Errorf("could not set up harvester: %v", err)
			// Report permanent setup failures as a degraded state for the input.
			if isPermanentHarvesterError(err) {
				state.ctx.UpdateStatus(
					status.Degraded,
					fmt.Sprintf("Harvester for Filestream input %q failed: %s", g.inputID, err),
				)
			}
			g.finish(state)
			return
		}
		g.mu.Lock()
		state.setUp = true
		closing := state.status == statusClosing || state.finished
		g.mu.Unlock()
		if closing {
			g.finish(state)
			return
		}
	}

	before := state.session.Offset()
	verdict, err := state.session.ReadSlice(state.ctx, state.publisher)
	after := state.session.Offset()

	g.mu.Lock()
	if state.status == statusClosing || state.finished {
		g.mu.Unlock()
		g.finish(state)
		return
	}

	if err != nil || verdict == SliceDone {
		g.mu.Unlock()
		if err != nil {
			state.ctx.Logger.Debugf("Harvester stopped with error: %v", err)
		}
		g.finish(state)
		return
	}

	// SliceYield: caught up to EOF. During a read_until_eof shutdown the source
	// has now been drained, so tear it down instead of parking it.
	if g.draining {
		g.mu.Unlock()
		g.finish(state)
		return
	}

	// Park for the waker and exit the goroutine. A slice that made progress resets
	// the backoff; one that read nothing grows it.
	if after > before {
		g.park(state, minWakerBackoff)
	} else {
		g.park(state, growBackoff(state.backoff))
	}
	g.mu.Unlock()
	g.signalWaker()
}

// currentResource atomically returns state's current registration key and the
// store resource for it, serialized against Migrate so a reader never pairs a
// stale key with a resource that already moved under it (see Migrate).
func (g *harvesterRunner) currentResource(state *sourceState) (string, *resource) {
	g.mu.Lock()
	defer g.mu.Unlock()

	return state.srcID, g.store.Get(state.srcID)
}

// setup acquires the per-source resources: registry lock, pipeline client,
// cursor/publisher and the reading session, populating state. On error it releases
// whatever it acquired and leaves state's resource fields nil.
func (g *harvesterRunner) setup(state *sourceState) error {
	id, resource := g.currentResource(state)
	if err := lockResource(state.ctx, resource, id); err != nil {
		return err
	}

	client, err := g.pipeline.ConnectWith(beat.ClientConfig{
		EventListener: newInputACKHandler(g.ackCH),
	})
	if err != nil {
		releaseResource(resource)
		// A pipeline connection failure is not transient; surface it as a
		// degraded input status rather than silently retrying each scan.
		return permanentHarvesterError{
			err: fmt.Errorf("error while connecting to output with pipeline: %w", err),
		}
	}

	g.store.UpdateTTL(resource, g.cleanTimeout)

	state.resource = resource
	state.client = client
	state.cursor = makeCursor(resource)
	state.publisher = &cursorPublisher{canceler: state.ctx.Cancelation, client: client, cursor: &state.cursor}

	session, err := g.harvester.OpenSession(state.ctx, state.src, state.cursor, g.metrics)
	if err != nil {
		_ = client.Close()
		releaseResource(resource)
		state.resource = nil
		state.client = nil
		state.publisher = nil
		return err
	}
	state.session = session
	state.isGZIP = session.IsGZIP()

	g.metrics.FilesActive.Inc()
	g.metrics.HarvesterRunning.Inc()
	g.metrics.FilesOpened.Inc()
	g.metrics.HarvesterOpenFiles.Inc()
	g.metrics.HarvesterStarted.Inc()
	if state.isGZIP {
		g.metrics.FilesGZIPActive.Inc()
		g.metrics.HarvesterGZIPRunning.Inc()
		g.metrics.FilesGZIPOpened.Inc()
		g.metrics.HarvesterOpenGZIPFiles.Inc()
		g.metrics.HarvesterGZIPStarted.Inc()
	}

	return nil
}

// Start registers a newly discovered source and starts reading it.
func (g *harvesterRunner) Start(ctx inputv2.Context, src Source) {
	g.enqueue(ctx, src)
}

// Restart stops a possibly-running harvester for the source and starts a fresh
// one. It does not block.
func (g *harvesterRunner) Restart(ctx inputv2.Context, src Source) {
	g.spawn(func() {
		srcID := g.identifier.ID(src)
		g.mu.Lock()
		state := g.states[srcID]
		g.mu.Unlock()
		if state != nil {
			g.stopAndWait(state)
		}
		g.enqueue(ctx, src)
	})
}

// enqueue registers a source and spawns a reader for it. Repeated calls for an
// already-known source are ignored; a parked source is resumed by the waker, not
// here.
func (g *harvesterRunner) enqueue(ctx inputv2.Context, src Source) {
	srcID := g.identifier.ID(src)

	g.mu.Lock()
	if g.closed {
		g.mu.Unlock()
		return
	}
	if _, exists := g.states[srcID]; exists {
		g.mu.Unlock()
		ctx.Logger.Debugf("Harvester already running for %s", srcID)
		return
	}

	// Standalone cancel context, deliberately NOT a child of the input context:
	// making 10k+ sources children of one parent serialises them on the parent's
	// mutex (context.removeChild on every create/cancel). Input shutdown cancels
	// every source explicitly via StopHarvesters (deferred in the prospector's
	// Run), and Stop cancels individual sources, so the parent link is not needed
	// for propagation.
	hctx, cancel := context.WithCancel(context.Background())
	ctx.Cancelation = hctx
	ctx.Logger = ctx.Logger.With("source_file", srcID)

	state := &sourceState{
		srcID:  srcID,
		src:    src,
		ctx:    ctx,
		cancel: cancel,
		done:   make(chan struct{}),
	}
	g.states[srcID] = state

	// Hard open-files limit: if every slot is taken, queue the source instead of
	// opening it. It holds no fd and no goroutine until finish() promotes it when
	// an open file closes.
	if g.harvesterLimit > 0 && g.nOpen >= g.harvesterLimit {
		g.setStatus(state, statusWaiting)
		g.waiting = append(g.waiting, state)
		g.mu.Unlock()
		ctx.Logger.Debugf("harvester_limit (%d) reached, queueing %s", g.harvesterLimit, srcID)
		return
	}

	state.holdsSlot = true
	g.nOpen++
	g.setStatus(state, statusNew)
	g.mu.Unlock()

	g.run(state)
}

// promoteLocked moves the next live waiting source into an open slot and returns
// it so the caller can spawn a reader (outside g.mu). Stale entries — sources
// stopped while queued — are skipped. Returns nil if nothing is waiting. Caller
// holds g.mu.
func (g *harvesterRunner) promoteLocked() *sourceState {
	for len(g.waiting) > 0 {
		next := g.waiting[0]
		g.waiting[0] = nil
		g.waiting = g.waiting[1:]
		if len(g.waiting) == 0 {
			g.waiting = nil
		}
		if next.finished || next.status != statusWaiting {
			continue // stopped or already claimed while queued
		}
		next.holdsSlot = true
		g.nOpen++
		g.setStatus(next, statusNew)
		return next
	}
	return nil
}

// waker evaluates parked sources as they come due: it resumes those with new
// data (by spawning a reader), re-parks those still idle, and tears down those
// that hit a close condition. It pops only due sources from the parked min-heap
// (rather than scanning every source) and sleeps until the next one is due. It
// centralises the read backoff and close-condition checks that the per-file
// harvester ran per file.
func (g *harvesterRunner) waker() {
	defer g.wg.Done()

	for {
		g.mu.Lock()
		if g.closed {
			g.mu.Unlock()
			return
		}
		due := g.popDue(time.Now())
		g.publishGauges()
		var next time.Time
		hasNext := g.parked.Len() > 0
		if hasNext {
			next = g.parked[0].due
		}
		g.mu.Unlock()

		for _, state := range due {
			g.pollParked(state)
		}

		// If we processed any, loop immediately to pick up sources that became
		// due during the polls before sleeping.
		if len(due) > 0 {
			continue
		}

		wait := maxWakerBackoff
		if hasNext {
			if d := time.Until(next); d < wait {
				wait = d
			}
		}
		if wait < 0 {
			wait = 0
		}
		t := time.NewTimer(wait)
		select {
		case <-g.ctx.Cancelation.Done():
			t.Stop()
			return
		case <-t.C:
		case <-g.wakerCh:
			t.Stop()
		}
	}
}

func statusIsActive(s sourceStatus) bool {
	return s == statusNew || s == statusRunning || s == statusPolling
}

// setStatus transitions state to ns and keeps the active/parked gauge counters in
// sync. statusClosing counts as neither. Caller holds g.mu.
func (g *harvesterRunner) setStatus(state *sourceState, ns sourceStatus) {
	old := state.status
	if old == ns {
		return
	}
	switch {
	case old == statusParked:
		g.nParked--
	case old == statusWaiting:
		g.nWaiting--
	case statusIsActive(old):
		g.nActive--
	}
	switch {
	case ns == statusParked:
		g.nParked++
	case ns == statusWaiting:
		g.nWaiting++
	case statusIsActive(ns):
		g.nActive++
	}
	state.status = ns
}

// park schedules state to be polled by the waker after backoff, recording it on the
// parked min-heap. Caller holds g.mu.
func (g *harvesterRunner) park(state *sourceState, backoff time.Duration) {
	state.backoff = backoff
	state.nextCheck = time.Now().Add(backoff)
	g.setStatus(state, statusParked)
	heap.Push(&g.parked, &parkedEntry{state: state, due: state.nextCheck})
}

// popDue removes and returns the parked sources whose nextCheck is due, claiming
// each (statusPolling) so nothing else touches it. Stale heap entries — a source
// re-parked or torn down since it was pushed — are skipped. Caller holds g.mu.
func (g *harvesterRunner) popDue(now time.Time) []*sourceState {
	var due []*sourceState
	for g.parked.Len() > 0 {
		if g.parked[0].due.After(now) {
			break
		}
		e, _ := heap.Pop(&g.parked).(*parkedEntry)
		state := e.state
		if state.status == statusParked && state.nextCheck.Equal(e.due) {
			g.setStatus(state, statusPolling)
			due = append(due, state)
		}
	}
	return due
}

// pollParked polls one due source and acts on the result: resume (spawn a
// reader), close (tear down) or re-park. Must be called without holding g.mu.
func (g *harvesterRunner) pollParked(state *sourceState) {
	result := state.session.Poll()

	g.mu.Lock()
	if state.status == statusClosing || state.finished {
		g.mu.Unlock()
		g.finish(state)
		return
	}
	if state.status != statusPolling {
		// Claimed by another actor (e.g. a drain reader during shutdown) while we
		// were polling; leave it to that actor.
		g.mu.Unlock()
		return
	}
	switch result {
	case PollResume:
		g.mu.Unlock()
		g.run(state) // new data: spawn a reader
	case PollClose:
		g.mu.Unlock()
		g.finish(state)
	default: // PollPark
		g.park(state, growBackoff(state.backoff))
		g.mu.Unlock()
	}
}

// publishGauges updates the observability gauges from the maintained counters.
// Caller holds g.mu.
func (g *harvesterRunner) publishGauges() {
	if g.mActive == nil {
		return
	}
	g.mActive.Set(uint64(g.nActive))   //nolint:gosec // counters are non-negative
	g.mParked.Set(uint64(g.nParked))   //nolint:gosec // counters are non-negative
	g.mWaiting.Set(uint64(g.nWaiting)) //nolint:gosec // counters are non-negative
}

// Stop stops a running harvester for a source. It does not block on an active
// read: it cancels the source and lets the reading goroutine tear it down.
func (g *harvesterRunner) Stop(src Source) {
	srcID := g.identifier.ID(src)

	g.mu.Lock()
	state := g.states[srcID]
	if state == nil {
		g.mu.Unlock()
		return
	}
	prev := state.status
	g.setStatus(state, statusClosing)
	cancel := state.cancel
	g.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	// If the source was parked, queued or still new, no goroutine holds it: a
	// parked source is skipped by the waker, a queued source has no reader at
	// all, and a new source's run() will finish it without spawning a reader.
	// For running/polling, the reader or waker that holds it finishes it after
	// the cancel.
	if prev == statusParked || prev == statusNew || prev == statusWaiting {
		g.finish(state)
	}
}

// stopAndWait stops a source and blocks until it has been fully torn down. Used
// by Restart so the new harvester does not race the old one's resource lock.
func (g *harvesterRunner) stopAndWait(state *sourceState) {
	g.mu.Lock()
	if state.finished {
		// A finish is already in flight; wait for it to fully tear down (done is
		// closed last, after the source is removed) so a following enqueue does
		// not see the dying source and skip the restart.
		done := state.done
		g.mu.Unlock()
		<-done
		return
	}
	prev := state.status
	g.setStatus(state, statusClosing)
	cancel := state.cancel
	done := state.done
	g.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	// A parked, queued or still-new source has no goroutine that will finish it
	// (see Stop), so tear it down here; otherwise wait for the holder to do it.
	if prev == statusParked || prev == statusNew || prev == statusWaiting {
		g.finish(state)
		return
	}
	<-done
}

// finish tears down a source's resources exactly once and removes it from the
// runner. It must be called without holding g.mu and never while a goroutine
// is inside the source's ReadSlice (the status/cancel protocol guarantees this).
func (g *harvesterRunner) finish(state *sourceState) {
	g.mu.Lock()
	if state.finished {
		g.mu.Unlock()
		return
	}
	g.setStatus(state, statusClosing) // adjust gauge counters from whatever it was
	state.finished = true
	wasSetUp := state.setUp
	g.mu.Unlock()

	g.notifyObserver(state)

	log := state.ctx.Logger
	if state.session != nil {
		log.Debug("Closing reader of filestream")
		_ = state.session.Close()
	}
	if state.client != nil {
		_ = state.client.Close()
	}
	if state.resource != nil {
		releaseResource(state.resource)
	}
	if state.cancel != nil {
		state.cancel()
	}

	// Only balance the metrics that setup incremented; a source torn down before
	// it was set up never incremented them.
	if wasSetUp {
		g.metrics.FilesActive.Dec()
		g.metrics.HarvesterRunning.Dec()
		g.metrics.FilesClosed.Inc()
		g.metrics.HarvesterOpenFiles.Dec()
		g.metrics.HarvesterClosed.Inc()
		if state.isGZIP {
			g.metrics.FilesGZIPActive.Dec()
			g.metrics.HarvesterGZIPRunning.Dec()
			g.metrics.FilesGZIPClosed.Inc()
			g.metrics.HarvesterOpenGZIPFiles.Dec()
			g.metrics.HarvesterGZIPClosed.Inc()
		}
		log.Debug("Stopped harvester for file")
	}

	// Remove the source from the runner only after its resources are released,
	// so anything that observes the removal (Restart's <-done wait, hasID) sees a
	// fully torn-down source rather than one still closing its session. Releasing
	// the slot here (after the fd is closed) and promoting the next queued source
	// keeps the open-files count at or below harvesterLimit at all times.
	g.mu.Lock()
	delete(g.states, state.srcID)
	var promoted *sourceState
	if state.holdsSlot {
		state.holdsSlot = false
		g.nOpen--
		if !g.closed {
			promoted = g.promoteLocked()
		}
	}
	g.mu.Unlock()

	close(state.done)

	if promoted != nil {
		g.run(promoted)
	}
}

func (g *harvesterRunner) notifyObserver(state *sourceState) {
	if g.notifyChan == nil || state.session == nil {
		return
	}
	offset := state.session.Offset()
	select {
	case g.notifyChan <- HarvesterStatus{ID: state.srcID, Size: offset}:
	case <-g.ctx.Cancelation.Done():
	}
}

// Continue starts a new harvester carrying over the state of a previous source.
func (g *harvesterRunner) Continue(ctx inputv2.Context, previous, next Source) {
	g.spawn(func() {
		prevID := g.identifier.ID(previous)
		nextID := g.identifier.ID(next)

		previousResource, err := lock(ctx, g.store, prevID)
		if err != nil {
			ctx.Logger.Errorf("Continue: cannot lock previous resource %s: %v", prevID, err)
			return
		}
		_ = g.store.remove(prevID)

		nextResource, err := lock(ctx, g.store, nextID)
		if err != nil {
			releaseResource(previousResource)
			ctx.Logger.Errorf("Continue: cannot lock next resource %s: %v", nextID, err)
			return
		}
		g.store.UpdateTTL(nextResource, g.cleanTimeout)
		previousResource.copyInto(nextResource)
		releaseResource(previousResource)
		releaseResource(nextResource)

		g.enqueue(ctx, next)
	})
}

// Migrate re-keys a running (or pending) source from oldID to next's identity
// without stopping it, so a later Start(next) finds it already registered and
// no-ops instead of spawning a duplicate. The registry re-key (updateStore) and
// the in-memory re-key happen in the same critical section: a Start racing with
// Migrate either lands before it (Migrate then refuses the now-occupied target)
// or after it (Start sees the migrated registration and no-ops), never both
// spawning a harvester for the same source. It is also safe to call when
// nothing is registered under oldID (or the registration is already tearing
// down): only the store is updated then.
func (g *harvesterRunner) Migrate(oldID string, next Source, updateStore func(newID string) error) error {
	newID := g.identifier.ID(next)

	g.mu.Lock()
	defer g.mu.Unlock()

	if _, exists := g.states[newID]; exists {
		// Target occupied — don't clobber an existing registration, and leave
		// the store alone.
		return fmt.Errorf("a harvester is already registered for %q", newID)
	}

	if err := updateStore(newID); err != nil {
		return err
	}

	state := g.states[oldID]
	if state == nil || state.finished {
		// Nothing to re-key in memory: either no source is registered under
		// oldID, or it is already tearing down (finish deletes it under its
		// original key once torn down).
		return nil
	}

	delete(g.states, oldID)
	state.srcID = newID
	state.src = next
	g.states[newID] = state
	return nil
}

// StopHarvesters stops all harvesters and the waker goroutine. With
// read_until_eof enabled it first drains every source to EOF (bounded by the
// configured Timeout) so data is not left unread on shutdown.
func (g *harvesterRunner) StopHarvesters() error {
	g.mu.Lock()
	if g.closed {
		g.mu.Unlock()
		return nil
	}
	if g.readUntilEOF.Enabled {
		return g.drainAndStop() // holds and releases g.mu
	}
	return g.stopNow() // holds and releases g.mu
}

// stopNow cancels every source and the waker and tears the sources down without
// draining. The caller holds g.mu; stopNow releases it.
func (g *harvesterRunner) stopNow() error {
	g.closed = true
	for _, state := range g.states {
		if state != nil {
			g.setStatus(state, statusClosing)
			if state.cancel != nil {
				state.cancel()
			}
		}
	}
	g.mu.Unlock()

	g.signalWaker()
	g.wg.Wait()
	g.finishRemaining()
	return nil
}

// drainAndStop implements the read_until_eof shutdown: it lets in-flight readers
// finish and spawns drain readers for idle sources so every source is read to
// EOF before teardown, instead of cancelling mid-read. A Timeout bounds the
// drain so a stuck read or blocked output cannot hang shutdown. The caller holds
// g.mu; drainAndStop releases it.
func (g *harvesterRunner) drainAndStop() error {
	g.closed = true
	g.draining = true
	// Running sources are already draining themselves; spawn a reader for the rest
	// so they read any remaining data to EOF. Sources the waker is currently
	// polling are handled by that poll (pollParked re-spawns a reader on resume).
	toDrain := make([]*sourceState, 0, len(g.states))
	for _, state := range g.states {
		// Queued sources were never opened; shutdown should not open new files to
		// drain them. finishRemaining tears them down.
		if state != nil && state.status != statusRunning && state.status != statusPolling && state.status != statusWaiting {
			toDrain = append(toDrain, state)
		}
	}
	// Only an actual in-flight drain is worth announcing: if every source already
	// reached EOF and tore itself down, there is nothing to wait for.
	draining := len(g.states) > 0
	g.mu.Unlock()

	if draining {
		g.ctx.Logger.Infof(
			"input closing, read_until_eof enabled, waiting EOF or %s timeout, whichever happens first",
			g.readUntilEOF.Timeout)
	}

	g.signalWaker() // let the waker observe closed and exit
	for _, state := range toDrain {
		g.run(state) // draining: read to EOF, then finish (not park)
	}

	// Bound the drain: cancel every source after Timeout so a stuck read or a
	// blocked output unblocks and tears down.
	timer := time.AfterFunc(g.readUntilEOF.Timeout, func() {
		g.mu.Lock()
		for _, state := range g.states {
			if state != nil && state.cancel != nil {
				state.cancel()
			}
		}
		g.mu.Unlock()
	})

	g.wg.Wait()
	// Stop reports true only if it cancelled the timer before it fired, i.e. the
	// drain reached EOF on its own; false means the Timeout elapsed first.
	reachedEOF := timer.Stop()
	g.finishRemaining()

	if draining {
		if reachedEOF {
			g.ctx.Logger.Info("read_until_eof enabled, EOF reached. closing input")
		} else {
			g.ctx.Logger.Infof(
				"read_until_eof enabled, %s timeout reached. closing input",
				g.readUntilEOF.Timeout)
		}
	}
	return nil
}

// finishRemaining tears down any sources still registered after the readers and
// waker have stopped (e.g. parked sources nothing drained).
func (g *harvesterRunner) finishRemaining() {
	g.mu.Lock()
	remaining := make([]*sourceState, 0, len(g.states))
	for _, state := range g.states {
		if state != nil {
			remaining = append(remaining, state)
		}
	}
	g.mu.Unlock()
	for _, state := range remaining {
		g.finish(state)
	}
}

func (g *harvesterRunner) spawn(fn func()) {
	g.mu.Lock()
	if g.closed {
		g.mu.Unlock()
		return
	}
	g.wg.Add(1)
	g.mu.Unlock()
	go func() {
		defer g.wg.Done()
		fn()
	}()
}

func (g *harvesterRunner) signalWaker() {
	select {
	case g.wakerCh <- struct{}{}:
	default:
	}
}

func growBackoff(cur time.Duration) time.Duration {
	if cur <= 0 {
		return minWakerBackoff
	}
	return min(cur*2, maxWakerBackoff)
}
