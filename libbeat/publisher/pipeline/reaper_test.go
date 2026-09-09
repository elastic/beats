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
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common/acker"
	"github.com/elastic/beats/v7/libbeat/tests/resources"
	"github.com/elastic/elastic-agent-libs/logp"
)

// newTestReaper returns a fresh, isolated clientReaper for unit tests.
// Tests must use this rather than sharedReaper to avoid cross-test
// interference and satisfy the goroutine-leak checker.
func newTestReaper() *clientReaper {
	return &clientReaper{
		pending: make(map[*Pipeline]map[*client]struct{}),
		notify:  make(chan struct{}, 1),
	}
}

// newDrainedClient returns a client whose events are already acknowledged
// (ACKWaitChan is pre-closed), so the reaper will finalize it on the next
// sweep without waiting.
func newDrainedClient(onRemove func()) *client {
	return &client{
		logger:         logp.NewNopLogger(),
		observer:       nilObserver,
		eventListener:  acker.Nil(),
		clientListener: &mockClientListener{},
		// nil ackWait → testProducer.ACKWaitChan returns closedChan
		producer: &testProducer{},
		onRemove: onRemove,
	}
}

// newHeldClient returns a client whose events have not yet drained.
// The caller closes the returned chan to simulate event acknowledgment,
// causing the reaper to finalize the client on its next sweep.
func newHeldClient(onRemove func()) (*client, chan struct{}) {
	ackWait := make(chan struct{})
	c := &client{
		logger:         logp.NewNopLogger(),
		observer:       nilObserver,
		eventListener:  acker.Nil(),
		clientListener: &mockClientListener{},
		producer:       &testProducer{ackWait: ackWait},
		onRemove:       onRemove,
	}
	return c, ackWait
}

// TestReaperFinalizesDrainedClient verifies that a client whose events are
// already acknowledged is finalized promptly after being handed to the reaper.
func TestReaperFinalizesDrainedClient(t *testing.T) {
	routinesChecker := resources.NewGoroutinesChecker()
	defer routinesChecker.Check(t)

	r := newTestReaper()
	p := new(Pipeline)
	r.acquire(p)
	defer r.release(p)

	var disconnected atomic.Int32
	c := newDrainedClient(func() { disconnected.Add(1) })
	r.add(p, c)

	require.Eventually(t, func() bool {
		return disconnected.Load() == 1
	}, 5*time.Second, 5*time.Millisecond,
		"reaper must finalize a drained client without a pipeline disconnect")
}

// TestReaperHoldsUndrainedClient verifies that the reaper does not finalize
// a client whose events have not drained, and finalizes it once they do.
// This exercises the reaperInterval re-poll path.
func TestReaperHoldsUndrainedClient(t *testing.T) {
	routinesChecker := resources.NewGoroutinesChecker()
	defer routinesChecker.Check(t)

	r := newTestReaper()
	p := new(Pipeline)
	r.acquire(p)
	defer r.release(p)

	var disconnected atomic.Int32
	c, ackWait := newHeldClient(func() { disconnected.Add(1) })
	r.add(p, c)

	// The client must stay pending across multiple reaper ticks.
	require.Never(t, func() bool {
		return disconnected.Load() != 0
	}, 3*reaperInterval, reaperInterval/2,
		"reaper must not finalize a client whose events have not drained")

	// Simulate acknowledgment.
	close(ackWait)

	require.Eventually(t, func() bool {
		return disconnected.Load() == 1
	}, 5*time.Second, 5*time.Millisecond,
		"reaper must finalize the client once its events drain")
}

// TestReaperReleaseDropsOnlyTargetPipeline verifies that releasing one
// pipeline silently drops only its own pending clients while a second
// pipeline's clients continue to be finalized normally.
func TestReaperReleaseDropsOnlyTargetPipeline(t *testing.T) {
	routinesChecker := resources.NewGoroutinesChecker()
	defer routinesChecker.Check(t)

	r := newTestReaper()
	p1 := new(Pipeline)
	p2 := new(Pipeline)
	r.acquire(p1)
	r.acquire(p2)

	// p1 gets a client that is not yet drained.
	var p1Disconnected atomic.Int32
	c1, _ := newHeldClient(func() { p1Disconnected.Add(1) })
	r.add(p1, c1)

	// p2 gets a client that is already drained.
	var p2Disconnected atomic.Int32
	c2 := newDrainedClient(func() { p2Disconnected.Add(1) })
	r.add(p2, c2)

	// p2's client must be finalized promptly.
	require.Eventually(t, func() bool {
		return p2Disconnected.Load() == 1
	}, 5*time.Second, 5*time.Millisecond,
		"p2's drained client must be finalized")

	// Releasing p1 must drop its pending client without finalizing it.
	r.release(p1)
	assert.Equal(t, int32(0), p1Disconnected.Load(),
		"release must not call disconnect on a still-pending client")

	r.release(p2)
}

// TestReaperMultipleClientsFinalized verifies that a burst of drained clients
// all handed to the reaper at once are all finalized.
func TestReaperMultipleClientsFinalized(t *testing.T) {
	routinesChecker := resources.NewGoroutinesChecker()
	defer routinesChecker.Check(t)

	r := newTestReaper()
	p := new(Pipeline)
	r.acquire(p)
	defer r.release(p)

	const N = 20
	var disconnected atomic.Int32
	for range N {
		c := newDrainedClient(func() { disconnected.Add(1) })
		r.add(p, c)
	}

	require.Eventually(t, func() bool {
		return disconnected.Load() == N
	}, 5*time.Second, 5*time.Millisecond,
		"all drained clients must be finalized")
}

// TestReaperNotifyWakesOnAdd verifies that add() wakes the reaper promptly
// rather than requiring a full reaperInterval poll.
func TestReaperNotifyWakesOnAdd(t *testing.T) {
	routinesChecker := resources.NewGoroutinesChecker()
	defer routinesChecker.Check(t)

	r := newTestReaper()
	p := new(Pipeline)
	r.acquire(p)
	defer r.release(p)

	var disconnected atomic.Int32
	c := newDrainedClient(func() { disconnected.Add(1) })

	start := time.Now()
	r.add(p, c)

	require.Eventually(t, func() bool {
		return disconnected.Load() == 1
	}, 5*time.Second, time.Millisecond)

	// Notification must arrive well within one polling interval.
	assert.Less(t, time.Since(start), reaperInterval,
		"reaper must wake on notify, not wait for the poll interval")
}

// TestReaperConcurrentAcquireRelease stress-tests rapid concurrent
// acquire/release cycles under the race detector.
//
// Without the fixes this exercises two hazards:
//   - Invariant 1: run() reads r.done without a lock while acquire() writes it.
//   - Invariant 2: a new acquire() calls wg.Add() on the per-generation
//     WaitGroup while a concurrent release() is still in wg.Wait().
func TestReaperConcurrentAcquireRelease(t *testing.T) {
	routinesChecker := resources.NewGoroutinesChecker()
	defer routinesChecker.Check(t)

	r := newTestReaper()

	const N = 200
	var wg sync.WaitGroup
	for range N {
		wg.Go(func() {
			p := new(Pipeline)
			r.acquire(p)
			r.release(p)
		})
	}
	wg.Wait()
}

// TestReaperSweepRaceWithRelease verifies that releasing a pipeline while
// the reaper goroutine is in the middle of calling c.disconnect() for that
// pipeline's clients does not produce a data race (invariant 3: sweepMu).
//
// The test uses a blocking onRemove to hold the reaper inside c.disconnect()
// while concurrently calling release(), then asserts that release() blocks
// until the in-flight disconnect() completes.
func TestReaperSweepRaceWithRelease(t *testing.T) {
	routinesChecker := resources.NewGoroutinesChecker()
	defer routinesChecker.Check(t)

	r := newTestReaper()
	p := new(Pipeline)
	r.acquire(p)

	// gate holds the reaper inside onRemove, which is called while
	// sweepMu is held. release() must not return until gate opens.
	gate := make(chan struct{})
	var disconnected atomic.Int32
	c := newDrainedClient(func() {
		<-gate
		disconnected.Add(1)
	})
	r.add(p, c)

	// Wait until the reaper has moved the client out of r.pending (it deleted
	// it from the set before acquiring sweepMu to call disconnect).
	require.Eventually(t, func() bool {
		r.mu.Lock()
		defer r.mu.Unlock()
		set, ok := r.pending[p]
		return !ok || len(set) == 0
	}, 5*time.Second, time.Millisecond,
		"reaper must have started sweeping the client")

	// release() must block because the reaper holds sweepMu inside onRemove.
	releaseDone := make(chan struct{})
	go func() {
		defer close(releaseDone)
		r.release(p)
	}()

	select {
	case <-releaseDone:
		t.Fatal("release() returned while the reaper's disconnect() was still running")
	case <-time.After(100 * time.Millisecond):
		// Good — release is serialized behind sweepMu.
	}

	// Unblock the reaper: disconnect() completes, sweepMu is released,
	// and release() can now proceed.
	close(gate)

	select {
	case <-releaseDone:
	case <-time.After(5 * time.Second):
		t.Fatal("release() did not return after the reaper's disconnect() completed")
	}

	assert.Equal(t, int32(1), disconnected.Load(),
		"disconnect must be called exactly once")
}
