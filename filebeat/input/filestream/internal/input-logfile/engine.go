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
	"sync"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
)

// wakerIdleWait caps how long the waker sleeps when nothing is due sooner, so a
// newly parked source is picked up even if its wake-up signal were lost. It
// matches DefaultBackoffConfig().Max, the longest a source is parked by default.
const wakerIdleWait = 10 * time.Second

// engine is the scheduler shared by every running filestream input in the
// process: one waker goroutine and one parked heap serve them all, rather than
// one of each per input.
//
// It holds no registry, pipeline or configuration state. Each input keeps its
// own store, pipeline connector, source identifier, metrics and open-file budget
// on its harvesterRunner and registers with the engine only to be scheduled, so
// a source's events still reach its own input's pipeline client and the engine
// needs no keying by registry path.
//
// Lifetime is reference counted: the first input to start creates it, later
// inputs join it, and it is torn down when the last input stops.
type engine struct {
	log *logp.Logger

	// mu guards all scheduling state: the parked heap below, and every
	// registered runner's states map, counters and per-source status. One lock
	// keeps a source in exactly one structure, in exactly one status, at a time
	// without a lock ordering between a runner and the engine.
	//
	// Contention is process-wide rather than per input. Most critical sections
	// are a heap push/pop and a status flip; the exceptions are Migrate, which
	// holds it across a registry write, and currentResource, across a store
	// lookup. Both are on source setup and take-over, not the read path.
	mu sync.Mutex
	// parked is a min-heap of parked sources across all registered runners, keyed
	// by nextCheck, so the waker sees only the sources actually due.
	parked sourceHeap

	wakerCh chan struct{}
	done    chan struct{}
	wg      sync.WaitGroup

	// start and stop are idempotent: acquireEngine starts the shared engine and
	// every registered runner starts it again, so a privately constructed engine
	// (in tests) comes up with the first runner that starts.
	startOnce sync.Once
	stopOnce  sync.Once
}

// sharedEngine is the process-global engine and its reference count. Callers go
// through acquireEngine/releaseEngine, never touching these directly.
var sharedEngine = struct {
	sync.Mutex
	engine *engine
	refs   int
}{}

// acquireEngine returns the process-global engine, starting it on the first
// call, together with a release function that is safe to call more than once.
// log is used for the engine's own lifecycle messages; the first acquirer's
// logger is kept for the engine's lifetime, which outlives that input.
func acquireEngine(log *logp.Logger) (*engine, func()) {
	sharedEngine.Lock()
	defer sharedEngine.Unlock()

	if sharedEngine.engine == nil {
		sharedEngine.engine = newEngine(log)
		sharedEngine.engine.start()
	}
	sharedEngine.refs++

	var once sync.Once
	return sharedEngine.engine, func() { once.Do(releaseEngine) }
}

// releaseEngine drops one reference and stops the engine once the last input
// has left.
//
// The engine is detached from the shared slot before it is stopped, so an input
// starting concurrently either joins a still-referenced engine or builds a fresh
// one, and can never be handed the engine being stopped — which would leave it
// holding an engine whose waker had exited, quietly not reading its files.
// TestEngineHandOver pins that ordering.
//
// Stopping outside the lock keeps a starting input from waiting on the departing
// engine's waker, which can take a poll's grace period to wind down.
func releaseEngine() {
	// Synchronous, so a stopped input leaves no waker running behind it.
	if stopping := releaseEngineRef(); stopping != nil {
		stopping.stop()
	}
}

// releaseEngineRef drops one engine reference. If this was the last reference,
// it detaches the engine from the shared slot and returns it to the caller to
// stop; otherwise returns nil.
func releaseEngineRef() *engine {
	sharedEngine.Lock()
	defer sharedEngine.Unlock()

	if sharedEngine.engine == nil {
		return nil
	}
	sharedEngine.refs--
	if sharedEngine.refs > 0 {
		return nil
	}

	stopping := sharedEngine.engine
	sharedEngine.engine = nil
	sharedEngine.refs = 0

	return stopping
}

// newEngine builds an engine without registering it as the process-global one.
// Tests use this to get an isolated scheduler.
func newEngine(log *logp.Logger) *engine {
	return &engine{
		log:     log,
		wakerCh: make(chan struct{}, 1),
		done:    make(chan struct{}),
	}
}

// start launches the waker goroutine. It is safe to call more than once.
func (e *engine) start() {
	e.startOnce.Do(func() {
		e.log.Debug("starting the shared filestream harvester scheduler")
		e.wg.Go(e.waker)
	})
}

// stop shuts the waker down and waits for it to exit. It is safe to call more
// than once. Sources are not torn down here: each input tears down its own
// sources through StopHarvesters before it releases the engine.
func (e *engine) stop() {
	e.stopOnce.Do(func() {
		close(e.done)
		e.wg.Wait()
		e.log.Debug("stopped the shared filestream harvester scheduler")
	})
}

// waker hands parked sources from every registered input to their own input's
// poll chain as they come due, which resumes those with new data (by spawning a
// reader), re-parks those still idle, and tears down those that hit a close
// condition. It pops only due sources and sleeps until the next one is due, so
// an idle fleet of inputs costs one sleeping goroutine.
func (e *engine) waker() {
	for {
		e.mu.Lock()
		due := e.popDue(time.Now())
		var next time.Time
		hasNext := e.parked.Len() > 0
		if hasNext {
			next = e.parked[0].due
		}
		e.mu.Unlock()

		// Hand each due source to its own input's poll chain and go back to
		// sleep. Polls run in turn within an input and concurrently across
		// inputs, so the fan-out is one chain per input with due work rather than
		// a goroutine per due file, and a source stuck in Poll (a stat() on an
		// unresponsive mount) delays only the input that owns it. A re-parked
		// source signals the waker, so nothing is missed by not waiting here.
		for _, state := range due {
			state.runner.enqueuePoll(state)
		}

		wait := wakerIdleWait
		if hasNext {
			wait = min(wait, time.Until(next))
		}
		wait = max(0, wait)
		select {
		case <-e.done:
			return
		case <-time.After(wait):
		case <-e.wakerCh:
		}
	}
}

// popDue removes and returns the parked sources whose nextCheck is due, claiming
// each (statusPolling) so nothing else touches it. Stale heap entries — a source
// re-parked or torn down since it was pushed — are skipped. Caller holds e.mu.
func (e *engine) popDue(now time.Time) []*sourceState {
	var due []*sourceState
	for e.parked.Len() > 0 {
		if e.parked[0].due.After(now) {
			break
		}
		entry, _ := heap.Pop(&e.parked).(*parkedEntry)
		state := entry.state
		// A source whose input is shutting down is dropped rather than claimed;
		// finishRemaining tears it down. Claiming it would flip it to
		// statusPolling, counting a reader that will never exist.
		if state.runner.closed {
			continue
		}
		// Skip stale heap entries: a source re-parked or torn down since it was
		// pushed gets a fresh entry with a new due time, leaving the old entry's
		// due no longer matching state.nextCheck.
		if state.status == statusParked && state.nextCheck.Equal(entry.due) {
			state.runner.setStatus(state, statusPolling)
			due = append(due, state)
		}
	}
	return due
}

// signalWaker makes the waker re-evaluate the parked heap now, rather than at
// the deadline it is currently sleeping on.
func (e *engine) signalWaker() {
	select {
	case e.wakerCh <- struct{}{}:
	default:
	}
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
