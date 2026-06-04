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

package slabqueue

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/publisher/queue"
)

// TestRecycleAvoidsAllocationOnRepeatedGetDone verifies the
// observable property of recycling: across many Get/Done cycles the
// pool's New func is invoked only for the working set of concurrently
// in-flight batches (here, one), not once per cycle. Counting New
// invocations is deterministic regardless of sync.Pool's internal
// per-P caching or GC-driven drops; pointer identity is best-effort
// and not a contract we test.
func TestRecycleAvoidsAllocationOnRepeatedGetDone(t *testing.T) {
	pool := NewPool[int](Settings{Events: 4}, nil)
	defer pool.Shutdown()

	// Wrap the existing New func to count allocations of new batch
	// structs. We don't reset to nil; subsequent Gets after a GC-
	// driven drop will increment, but in a synchronous test loop
	// without forced GC the count stays at the working-set size.
	var newCount int
	wrapped := pool.batchPool.New
	pool.batchPool.New = func() any {
		newCount++
		return wrapped()
	}

	q := pool.Connect()
	defer q.Close(true)
	p := q.Producer(queue.ProducerConfig{ACK: func(int) {}})

	const cycles = 100
	for i := 0; i < cycles; i++ {
		_, ok := p.Publish(i)
		require.True(t, ok)
		b, err := q.Get(0)
		require.NoError(t, err)
		require.Equal(t, 1, b.Count())
		b.Done()
	}

	// Under recycling we should see far fewer New invocations than
	// cycles. Allowing a small buffer for sync.Pool / runtime GC
	// effects, anything below cycles/2 is decisive evidence of reuse.
	assert.Less(t, newCount, cycles/2,
		"sync.Pool should have recycled batches across %d cycles (New invoked %d times)", cycles, newCount)
}

// TestRecycledBatchFieldsAreReset verifies the state-reset side of
// recycling by inspecting a freshly-Get'd batch's transient fields.
// Whether or not it's the same object pointer as a previously-Done'd
// batch, every field that could leak from a prior use must be
// initialised by Get.
func TestRecycledBatchFieldsAreReset(t *testing.T) {
	pool := NewPool[int](Settings{Events: 4}, nil)
	defer pool.Shutdown()
	q := pool.Connect()
	defer q.Close(true)
	p := q.Producer(queue.ProducerConfig{ACK: func(int) {}})

	// Do a Get/Done so the pool has a recycled batch with possibly
	// non-zero ackProducers/ackCounts/done/freed/next in its history.
	p.Publish(7)
	b, err := q.Get(0)
	require.NoError(t, err)
	b.Done()

	// New Get: even if sync.Pool hands back the same struct, every
	// observable field must be in its initial state.
	p.Publish(8)
	b2, err := q.Get(0)
	require.NoError(t, err)
	bi, ok := b2.(*batch[int])
	require.True(t, ok, "Get must return a *batch[int]")
	assert.False(t, bi.freed, "freed must be reset")
	assert.False(t, bi.done, "done must be reset")
	assert.Nil(t, bi.next, "next must be reset")
	assert.Empty(t, bi.ackProducers, "ackProducers must be length-zero on use")
	assert.Empty(t, bi.ackCounts, "ackCounts must be length-zero on use")
	assert.Same(t, q, bi.queue, "queue pointer must point at the current Queue")
	b2.Done()
}

// TestBatchRecycleResetsAckSlices verifies the ackProducers and
// ackCounts slices are reused across recycles. Without resetting them
// to length zero on recycle (handled in Pool.putBatch) a subsequent
// Done would accumulate counts on top of the previous batch's
// per-producer state, double-counting events.
func TestBatchRecycleResetsAckSlices(t *testing.T) {
	pool := NewPool[int](Settings{Events: 4}, nil)
	defer pool.Shutdown()
	q := pool.Connect()
	defer q.Close(true)

	counts := make(chan int, 4)
	p := q.Producer(queue.ProducerConfig{ACK: func(n int) { counts <- n }})

	// Run two full cycles so the second one definitely uses a recycled
	// batch struct.
	for cycle := 0; cycle < 2; cycle++ {
		p.Publish(cycle*10 + 1)
		p.Publish(cycle*10 + 2)
		b, err := q.Get(0)
		require.NoError(t, err)
		require.Equal(t, 2, b.Count())

		// Inspect the underlying batch struct's ackProducers/Counts
		// BEFORE Done: they should be empty (cleared by recycle or
		// fresh on first iteration).
		bi, ok := b.(*batch[int])
		require.True(t, ok, "Get must return a *batch[int]")
		require.Empty(t, bi.ackProducers, "ackProducers must start empty on recycle/use")
		require.Empty(t, bi.ackCounts, "ackCounts must start empty on recycle/use")

		b.Done()
		select {
		case n := <-counts:
			assert.Equal(t, 2, n, "ACK callback should fire for exactly the events in this batch (no carry-over)")
		case <-time.After(time.Second):
			t.Fatal("expected ACK callback after Done")
		}
	}
}

// TestRecycleAfterRelease verifies the Release path also returns the
// batch to the pool — repeated Get/Release cycles should not allocate
// a new batch struct each time. Counts New invocations across cycles
// for a deterministic check.
func TestRecycleAfterRelease(t *testing.T) {
	pool := NewPool[int](Settings{Events: 4}, nil)
	defer pool.Shutdown()

	var newCount int
	wrapped := pool.batchPool.New
	pool.batchPool.New = func() any {
		newCount++
		return wrapped()
	}

	q := pool.Connect()
	defer q.Close(true)
	p := q.Producer(queue.ProducerConfig{ACK: func(int) {}})

	const cycles = 100
	for i := 0; i < cycles; i++ {
		_, ok := p.Publish(i)
		require.True(t, ok)
		b, err := q.Get(0)
		require.NoError(t, err)
		bi, ok := b.(*batch[int])
		require.True(t, ok, "Get must return a *batch[int]")
		bi.Release()
	}

	assert.Less(t, newCount, cycles/2,
		"Release path should recycle batches (New invoked %d times in %d cycles)", newCount, cycles)
}

// TestDoubleDoneIsSafe verifies the double-completion guard: a second
// Done (or Release) on a recycled batch returns harmlessly instead of
// corrupting the new owner's state. This is a defensive guard against
// stale references; the contract says callers must complete exactly
// once.
func TestDoubleDoneIsSafe(t *testing.T) {
	pool := NewPool[int](Settings{Events: 4}, nil)
	defer pool.Shutdown()
	q := pool.Connect()
	defer q.Close(true)

	p := q.Producer(queue.ProducerConfig{ACK: func(int) {}})
	p.Publish(1)
	// Close the producer to return any pre-claimed magazine slots before
	// checking pool availability.
	p.Close()
	b, err := q.Get(0)
	require.NoError(t, err)

	// First Done: legitimate completion. Recycles the batch.
	b.Done()
	// Capture the batch's available slot count after Done — should be
	// back to full capacity.
	require.Equal(t, 4, pool.Available())

	// Second Done on the stale reference: guarded by the queue == nil
	// check, must be a no-op. If it weren't, indices would re-release
	// to pool.free and Available would exceed Capacity.
	bi, ok := b.(*batch[int])
	require.True(t, ok, "Get must return a *batch[int]")
	assert.NotPanics(t, func() { b.Done() })
	assert.NotPanics(t, func() { bi.Release() })
	assert.Equal(t, 4, pool.Available(),
		"second Done/Release on a recycled batch must not double-release slots")
}

// TestFreeEntriesAfterRecycleIsSafe verifies that a stale FreeEntries
// call on a recycled batch is a safe no-op (does not zero out slots
// now owned by a fresh consumer).
func TestFreeEntriesAfterRecycleIsSafe(t *testing.T) {
	pool := NewPool[int](Settings{Events: 4}, nil)
	defer pool.Shutdown()
	q := pool.Connect()
	defer q.Close(true)

	p := q.Producer(queue.ProducerConfig{ACK: func(int) {}})
	p.Publish(7)
	b, err := q.Get(0)
	require.NoError(t, err)
	require.Equal(t, 7, b.Entry(0))

	b.Done() // recycle

	// Stale FreeEntries should not panic and not touch fresh state.
	assert.NotPanics(t, func() { b.FreeEntries() })

	// Publish another event and confirm it's intact.
	p.Publish(99)
	b2, err := q.Get(0)
	require.NoError(t, err)
	assert.Equal(t, 99, b2.Entry(0), "fresh batch's event must be unaffected by stale FreeEntries")
	b2.Done()
}

// TestConcurrentRecycleNoCorruption stress-tests the recycle path
// under concurrent producer/consumer load with multiple receivers
// sharing the pool. We rely on the race detector and on slot
// accounting (after all activity, every slot must return to the
// pool's free list) to catch corruption.
func TestConcurrentRecycleNoCorruption(t *testing.T) {
	const (
		pipelines     = 4
		eventsPerPipe = 500
		poolSize      = 32
	)
	pool := NewPool[int](Settings{Events: poolSize}, nil)
	defer pool.Shutdown()

	var wg sync.WaitGroup
	for i := 0; i < pipelines; i++ {
		q := pool.Connect()
		p := q.Producer(queue.ProducerConfig{ACK: func(int) {}})

		wg.Add(2)
		go func(p queue.Producer[int], base int) {
			defer wg.Done()
			defer p.Close()
			for j := 0; j < eventsPerPipe; j++ {
				if _, ok := p.Publish(base*1_000_000 + j); !ok {
					return
				}
			}
		}(p, i)

		go func(q *Queue[int], base int) {
			defer wg.Done()
			defer q.Close(false)
			received := 0
			for received < eventsPerPipe {
				b, err := q.Get(16)
				if err != nil {
					return
				}
				// Verify cross-pipeline isolation: every event we
				// see must come from our base.
				for k := 0; k < b.Count(); k++ {
					v := b.Entry(k)
					if v/1_000_000 != base {
						t.Errorf("pipeline %d saw event from pipeline %d (%d)", base, v/1_000_000, v)
					}
				}
				received += b.Count()
				b.Done()
			}
		}(q, i)
	}

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("concurrent recycle stress did not finish in time")
	}

	assert.Equal(t, poolSize, pool.Available(),
		"every slot must return to the pool after recycling activity (got %d/%d)",
		pool.Available(), poolSize)
}
