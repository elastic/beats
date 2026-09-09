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

package pipeline

import (
	"sync"
	"time"
)

// reaperInterval is how often the reaper re-checks pending clients whose events
// have not drained yet. Finalization is cleanup, not latency-sensitive, so a
// coarse interval keeps the reaper cheap; a client whose events are already
// acked when it is handed over is finalized immediately on the notify wakeup.
const reaperInterval = 50 * time.Millisecond

// sharedReaper is the process-wide client reaper, shared across all pipelines.
// It is started on the first pipeline connect and stopped when the last
// pipeline disconnects. It must not be used during package init.
var sharedReaper = newClientReaper()

func newClientReaper() *clientReaper {
	r := &clientReaper{
		pending: make(map[*Pipeline]map[*client]struct{}),
		notify:  make(chan struct{}, 1),
	}
	r.noSweep = sync.NewCond(&r.mu)
	return r
}

// clientReaper finalizes Closed-but-not-yet-drained clients across all
// pipelines in a single goroutine rather than one per pipeline. Pending
// clients are grouped by pipeline so release can drop all of a disconnecting
// pipeline's entries atomically.
//
// Three design invariants, each addressing a specific correctness hazard:
//
//  1. done is captured once into a local variable inside the goroutine
//     closure (not re-read from r.done inside run). This prevents a data
//     race with a concurrent acquire() that overwrites r.done.
//
//  2. Each goroutine generation owns its own *sync.WaitGroup (runWG), created
//     fresh in acquire() before the goroutine starts. release() captures runWG
//     before releasing r.mu and waits on that specific WaitGroup — not a
//     shared r.wg field. This prevents concurrent Add/Wait violations when a
//     new acquire() races with a release() that is still in wg.Wait().
//
//  3. sweeping is set to true (under mu) before any c.disconnect() call
//     begins, and cleared to false (under mu, with a Broadcast on noSweep)
//     once all calls in the batch finish. release() waits on noSweep until
//     sweeping is false after deleting p from pending. Because both the
//     sweeping=true assignment and delete(r.pending, p) are serialised by mu,
//     a client from p can only be present in the current sweep's ready list
//     if sweeping was set to true before release() acquired mu — in which
//     case the Wait loop will block until Broadcast signals completion.
type clientReaper struct {
	mu       sync.Mutex
	noSweep  *sync.Cond // broadcast when sweeping → false; uses mu as its lock
	sweeping bool       // true while c.disconnect() calls are in progress
	pending  map[*Pipeline]map[*client]struct{}
	notify   chan struct{}

	refs  int
	done  chan struct{}
	runWG *sync.WaitGroup // per-generation WaitGroup; see invariant 2
}

// acquire registers pipeline p with the reaper, starting the reaper goroutine
// on the first call. Every acquire must be paired with a release.
func (r *clientReaper) acquire(p *Pipeline) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pending[p] = make(map[*client]struct{})
	if r.refs == 0 {
		r.done = make(chan struct{})
		wg := &sync.WaitGroup{}
		r.runWG = wg
		done := r.done // capture once; see invariant 1
		wg.Go(func() { r.run(done) })
	}
	r.refs++
}

// release removes all pending clients for pipeline p and decrements the
// reference count, stopping the reaper goroutine once the last pipeline
// releases.
//
// After release returns, no client from p will be touched by the reaper
// goroutine. The caller may then safely call disconnectClients().
func (r *clientReaper) release(p *Pipeline) {
	r.mu.Lock()
	delete(r.pending, p)
	r.refs--
	shutdown := r.refs == 0
	var wg *sync.WaitGroup
	if shutdown {
		close(r.done)
		wg = r.runWG
	}
	// Wait for any in-flight sweep to finish its c.disconnect() calls.
	// See invariant 3.
	for r.sweeping {
		r.noSweep.Wait()
	}
	r.mu.Unlock()

	if wg != nil {
		wg.Wait()
	}
}

// add hands a Closed client to the reaper for finalization once its events
// drain. If pipeline p has already been released, the call is a no-op; the
// pipeline's disconnectClients will handle the client instead.
func (r *clientReaper) add(p *Pipeline, c *client) {
	r.mu.Lock()
	if set, ok := r.pending[p]; ok {
		set[c] = struct{}{}
	}
	r.mu.Unlock()
	select {
	case r.notify <- struct{}{}:
	default:
	}
}

// run is the single reaper goroutine. It non-blockingly sweeps every
// pipeline's pending set each pass and finalizes clients whose ACKWaitChan
// has closed. When nothing is pending it blocks until a client is added or
// the reaper is stopped; otherwise it re-sweeps every reaperInterval.
//
// done is the channel that signals shutdown; it is passed in rather than read
// from r.done to avoid a data race with acquire() — see invariant 1.
func (r *clientReaper) run(done <-chan struct{}) {
	for {
		r.mu.Lock()
		var ready []*client
		total := 0
		for _, set := range r.pending {
			for c := range set {
				select {
				case <-c.producer.ACKWaitChan():
					ready = append(ready, c)
					delete(set, c)
				default:
					total++
				}
			}
		}
		if len(ready) > 0 {
			r.sweeping = true
		}
		r.mu.Unlock()

		for _, c := range ready {
			c.disconnect()
		}

		if len(ready) > 0 {
			r.mu.Lock()
			r.sweeping = false
			r.noSweep.Broadcast()
			r.mu.Unlock()
		}

		if total == 0 {
			select {
			case <-done:
				return
			case <-r.notify:
			}
		} else {
			select {
			case <-done:
				return
			case <-r.notify:
			case <-time.After(reaperInterval):
			}
		}
	}
}
