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

package otelqueue

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/publisher/queue"
)

// TestPublishAndGet verifies basic single-pipeline publish/get/ack flow.
func TestPublishAndGet(t *testing.T) {
	pool := NewPool[int](Settings{Events: 4}, nil)
	q := pool.Connect()

	p := q.Producer(queue.ProducerConfig{})
	_, ok := p.Publish(1)
	require.True(t, ok)
	_, ok = p.Publish(2)
	require.True(t, ok)
	_, ok = p.Publish(3)
	require.True(t, ok)

	assert.Equal(t, 4, pool.Capacity())
	assert.Equal(t, 1, pool.Available())
	assert.Equal(t, 3, pool.Capacity()-pool.Available(), "pool should report 3 slots in use")

	b, err := q.Get(0)
	require.NoError(t, err)
	require.Equal(t, 3, b.Count())
	assert.Equal(t, 1, b.Entry(0))
	assert.Equal(t, 2, b.Entry(1))
	assert.Equal(t, 3, b.Entry(2))

	b.Done()
	assert.Equal(t, 4, pool.Available(), "all slots should be released after Done")
}

// TestPerPipelineFIFOIsolation verifies that two pipelines share the pool but
// each delivers only its own events to its own consumer, in its own publish
// order.
func TestPerPipelineFIFOIsolation(t *testing.T) {
	pool := NewPool[int](Settings{Events: 6}, nil)
	qA := pool.Connect()
	qB := pool.Connect()

	pA := qA.Producer(queue.ProducerConfig{})
	pB := qB.Producer(queue.ProducerConfig{})

	// Interleave publishes across pipelines.
	pA.Publish(10)
	pB.Publish(20)
	pA.Publish(11)
	pB.Publish(21)
	pA.Publish(12)

	bA, err := qA.Get(0)
	require.NoError(t, err)
	require.Equal(t, 3, bA.Count())
	assert.Equal(t, []int{10, 11, 12}, []int{bA.Entry(0), bA.Entry(1), bA.Entry(2)})

	bB, err := qB.Get(0)
	require.NoError(t, err)
	require.Equal(t, 2, bB.Count())
	assert.Equal(t, []int{20, 21}, []int{bB.Entry(0), bB.Entry(1)})

	bA.Done()
	bB.Done()
	assert.Equal(t, 6, pool.Available())
}

// TestPublishBlocksWhenPoolExhausted verifies the pool's free list acts as a
// counting semaphore: a Publish blocks until a slot is freed.
func TestPublishBlocksWhenPoolExhausted(t *testing.T) {
	pool := NewPool[int](Settings{Events: 2}, nil)
	defer pool.Shutdown()

	qA := pool.Connect()
	defer qA.Close(true)
	qB := pool.Connect()
	defer qB.Close(true)

	pA := qA.Producer(queue.ProducerConfig{})
	pB := qB.Producer(queue.ProducerConfig{})

	// Fill the pool entirely with A's events.
	_, ok := pA.Publish(1)
	require.True(t, ok)
	_, ok = pA.Publish(2)
	require.True(t, ok)
	assert.Equal(t, 0, pool.Available())

	// B's Publish should block. Run it in a goroutine and assert it doesn't
	// complete within a short window.
	publishedB := make(chan struct{})
	go func() {
		defer close(publishedB)
		pB.Publish(99)
	}()

	select {
	case <-publishedB:
		t.Fatal("Publish to exhausted pool should have blocked")
	case <-time.After(200 * time.Millisecond):
		// expected: still blocked
	}

	// Drain one of A's events, returning a slot to the pool.
	bA, err := qA.Get(1)
	require.NoError(t, err)
	require.Equal(t, 1, bA.Count())
	bA.Done()

	// B's Publish should now unblock.
	select {
	case <-publishedB:
		// good
	case <-time.After(time.Second):
		t.Fatal("Publish should have unblocked after a slot was freed")
	}
}

// TestTryPublish_ReturnsFalseWhenFull confirms TryPublish never blocks.
func TestTryPublish_ReturnsFalseWhenFull(t *testing.T) {
	pool := NewPool[int](Settings{Events: 1}, nil)
	defer pool.Shutdown()
	q := pool.Connect()
	defer q.Close(true)

	p := q.Producer(queue.ProducerConfig{})
	_, ok := p.TryPublish(1)
	require.True(t, ok)
	_, ok = p.TryPublish(2)
	assert.False(t, ok, "TryPublish should return false when the pool is full")
}

// TestProducerACKCallback verifies that the per-producer ACK callback fires
// with the correct event count when batches are acknowledged.
func TestProducerACKCallback(t *testing.T) {
	pool := NewPool[int](Settings{Events: 8}, nil)
	q := pool.Connect()

	acked := make(chan int, 4)
	p := q.Producer(queue.ProducerConfig{ACK: func(n int) { acked <- n }})

	for i := 0; i < 5; i++ {
		p.Publish(i)
	}
	b, err := q.Get(0)
	require.NoError(t, err)
	b.Done()

	select {
	case n := <-acked:
		assert.Equal(t, 5, n)
	case <-time.After(time.Second):
		t.Fatal("expected producer ACK callback to fire")
	}
}

// TestACKCallbackFiresInPublishOrder verifies that when a later batch is
// Done()'d before an earlier one, the producer ACK callback for the later
// batch is deferred until the earlier batch is also Done. Without this,
// order-sensitive consumers (e.g. filestream's EventPrivateReporter) would
// map ACK counts to the wrong events and corrupt their registry.
func TestACKCallbackFiresInPublishOrder(t *testing.T) {
	pool := NewPool[int](Settings{Events: 8}, nil)
	q := pool.Connect()

	ackedCounts := make(chan int, 4)
	p := q.Producer(queue.ProducerConfig{ACK: func(n int) { ackedCounts <- n }})

	for i := 0; i < 4; i++ {
		p.Publish(i)
	}

	// Read two batches of size 2; b1 = events 0,1; b2 = events 2,3.
	b1, err := q.Get(2)
	require.NoError(t, err)
	require.Equal(t, 2, b1.Count())
	b2, err := q.Get(2)
	require.NoError(t, err)
	require.Equal(t, 2, b2.Count())

	// Done the LATER batch first. Its ACK callback must NOT fire yet.
	b2.Done()
	select {
	case n := <-ackedCounts:
		t.Fatalf("ACK fired for later batch before earlier was done (n=%d)", n)
	case <-time.After(100 * time.Millisecond):
		// expected
	}

	// Done the earlier batch. Now both ACK callbacks should fire in publish
	// order: first for b1 (count=2), then for b2 (count=2).
	b1.Done()
	select {
	case n := <-ackedCounts:
		assert.Equal(t, 2, n, "first ACK should fire for the earlier batch")
	case <-time.After(time.Second):
		t.Fatal("first ACK callback did not fire")
	}
	select {
	case n := <-ackedCounts:
		assert.Equal(t, 2, n, "second ACK should fire for the later batch")
	case <-time.After(time.Second):
		t.Fatal("second ACK callback did not fire")
	}
}

// TestSlotsReleasedBeforeACKOrderingResolves verifies that even when a later
// batch's ACK is held back waiting for an earlier batch, its slots are still
// released to the pool immediately. This keeps the queue from stalling under
// out-of-order ack completion.
func TestSlotsReleasedBeforeACKOrderingResolves(t *testing.T) {
	pool := NewPool[int](Settings{Events: 4}, nil)
	q := pool.Connect()

	p := q.Producer(queue.ProducerConfig{ACK: func(int) {}})
	for i := 0; i < 4; i++ {
		p.Publish(i)
	}
	assert.Equal(t, 0, pool.Available(), "pool should be full")

	b1, err := q.Get(2)
	require.NoError(t, err)
	b2, err := q.Get(2)
	require.NoError(t, err)

	// Done b2 first; its slots should be released even though ordering
	// holds back its ACK callback.
	b2.Done()
	assert.Equal(t, 2, pool.Available(), "b2 slots should be released even with deferred ACK")

	b1.Done()
	assert.Equal(t, 4, pool.Available(), "all slots should be released after both batches Done")
}

// TestCloseWaitsForInFlightBatches verifies the regression fix for Done()
// semantics: Done() must not fire until every batch handed out by Get has
// been Done()'d, not just until the FIFO is empty.
func TestCloseWaitsForInFlightBatches(t *testing.T) {
	pool := NewPool[int](Settings{Events: 4}, nil)
	defer pool.Shutdown()
	q := pool.Connect()

	p := q.Producer(queue.ProducerConfig{})
	p.Publish(1)
	p.Publish(2)

	// Drain the FIFO into an in-flight batch; q.count is now 0 but the
	// batch hasn't been Done()'d yet.
	b, err := q.Get(0)
	require.NoError(t, err)

	q.Close(false)

	// Done() must NOT fire while the batch is still in flight, even though
	// the FIFO is empty.
	select {
	case <-q.Done():
		t.Fatal("Done() fired before in-flight batch was Done()'d")
	case <-time.After(150 * time.Millisecond):
		// expected
	}

	// Done()'ing the batch should now release the queue's Done signal.
	b.Done()
	select {
	case <-q.Done():
		// expected
	case <-time.After(time.Second):
		t.Fatal("Done() did not fire after the last in-flight batch completed")
	}
}

// TestPerPipelineACKIsolation verifies that pipelines' ACK callbacks are
// independent: acking A's batch fires A's callback, not B's.
func TestPerPipelineACKIsolation(t *testing.T) {
	pool := NewPool[int](Settings{Events: 8}, nil)
	qA := pool.Connect()
	qB := pool.Connect()

	ackedA := make(chan int, 1)
	ackedB := make(chan int, 1)
	pA := qA.Producer(queue.ProducerConfig{ACK: func(n int) { ackedA <- n }})
	pB := qB.Producer(queue.ProducerConfig{ACK: func(n int) { ackedB <- n }})

	pA.Publish(1)
	pA.Publish(2)
	pB.Publish(10)

	bA, err := qA.Get(0)
	require.NoError(t, err)
	bA.Done()

	select {
	case n := <-ackedA:
		assert.Equal(t, 2, n)
	case <-time.After(time.Second):
		t.Fatal("A's ACK callback should fire")
	}
	select {
	case <-ackedB:
		t.Fatal("B's ACK callback must not fire when only A is acked")
	case <-time.After(100 * time.Millisecond):
		// expected
	}
}

// TestGetBlocksUntilPublish verifies Get waits for the first event.
func TestGetBlocksUntilPublish(t *testing.T) {
	pool := NewPool[int](Settings{Events: 2}, nil)
	defer pool.Shutdown()
	q := pool.Connect()
	defer q.Close(true)

	done := make(chan int, 1)
	go func() {
		b, err := q.Get(0)
		if err != nil {
			done <- -1
			return
		}
		done <- b.Entry(0)
		b.Done()
	}()

	// Give the goroutine time to block on Get.
	time.Sleep(50 * time.Millisecond)

	p := q.Producer(queue.ProducerConfig{})
	p.Publish(42)

	select {
	case v := <-done:
		assert.Equal(t, 42, v)
	case <-time.After(time.Second):
		t.Fatal("Get did not unblock after a publish")
	}
}

// TestCloseUnblocksGet verifies a pending Get returns EOF when the queue is
// closed.
func TestCloseUnblocksGet(t *testing.T) {
	pool := NewPool[int](Settings{Events: 2}, nil)
	defer pool.Shutdown()
	q := pool.Connect()

	got := make(chan error, 1)
	go func() {
		_, err := q.Get(0)
		got <- err
	}()
	time.Sleep(50 * time.Millisecond)

	q.Close(false)

	select {
	case err := <-got:
		assert.Error(t, err, "Get should return EOF after Close")
	case <-time.After(time.Second):
		t.Fatal("Get did not unblock after Close")
	}
}

// TestCloseForceReleasesSlots verifies that force-close drops in-flight events
// and releases their slots.
func TestCloseForceReleasesSlots(t *testing.T) {
	pool := NewPool[int](Settings{Events: 4}, nil)
	defer pool.Shutdown()
	q := pool.Connect()

	p := q.Producer(queue.ProducerConfig{})
	for i := 0; i < 3; i++ {
		p.Publish(i)
	}
	assert.Equal(t, 1, pool.Available())

	q.Close(true)
	assert.Equal(t, 4, pool.Available(), "force-close should release all queue slots")

	select {
	case <-q.Done():
		// expected
	case <-time.After(time.Second):
		t.Fatal("Done() should fire immediately on force-close")
	}
}

// TestCloseGracefulWaitsForDrain verifies Close(false) waits for in-flight
// events to be acked before Done() fires.
func TestCloseGracefulWaitsForDrain(t *testing.T) {
	pool := NewPool[int](Settings{Events: 4}, nil)
	defer pool.Shutdown()
	q := pool.Connect()
	p := q.Producer(queue.ProducerConfig{})

	p.Publish(1)
	p.Publish(2)

	// Close non-forced; Done should not fire while events are pending.
	q.Close(false)
	select {
	case <-q.Done():
		t.Fatal("Done() must not fire before events are drained")
	case <-time.After(100 * time.Millisecond):
		// expected
	}

	// Drain and ack; Done should then fire.
	b, err := q.Get(0)
	require.NoError(t, err)
	b.Done()

	select {
	case <-q.Done():
		// expected
	case <-time.After(time.Second):
		t.Fatal("Done() should fire after drain on Close(false)")
	}
}

// TestPublishAfterCloseFails verifies new publishes fail after Close.
func TestPublishAfterCloseFails(t *testing.T) {
	pool := NewPool[int](Settings{Events: 2}, nil)
	defer pool.Shutdown()
	q := pool.Connect()
	q.Close(false)

	p := q.Producer(queue.ProducerConfig{})
	_, ok := p.Publish(1)
	assert.False(t, ok, "Publish should fail after Close")
}

// TestSlotMemoryClearedOnAck verifies that when a slot is released back to the
// pool, the event reference is cleared so the GC can collect the underlying
// payload.
func TestSlotMemoryClearedOnAck(t *testing.T) {
	pool := NewPool[*int](Settings{Events: 2}, nil)
	defer pool.Shutdown()
	q := pool.Connect()

	p := q.Producer(queue.ProducerConfig{})
	v := 42
	p.Publish(&v)

	b, err := q.Get(0)
	require.NoError(t, err)
	b.Done()

	// The slot's event reference should now be nil so the GC can reclaim it.
	for i := range pool.storage {
		assert.Nil(t, pool.storage[i].event, "slot %d should have a nil event after Done", i)
		assert.Nil(t, pool.storage[i].producer, "slot %d should have no producer after Done", i)
	}
}

// TestConcurrentPublishersAndConsumers exercises the pool under contention.
func TestConcurrentPublishersAndConsumers(t *testing.T) {
	const (
		pipelines     = 3
		eventsPerPipe = 200
		poolSize      = 16
	)

	pool := NewPool[int](Settings{Events: poolSize}, nil)
	defer pool.Shutdown()

	var wg sync.WaitGroup
	for i := 0; i < pipelines; i++ {
		q := pool.Connect()
		p := q.Producer(queue.ProducerConfig{})

		wg.Add(2)
		// Producer.
		go func(p queue.Producer[int], base int) {
			defer wg.Done()
			for j := 0; j < eventsPerPipe; j++ {
				_, ok := p.Publish(base*1000 + j)
				if !ok {
					return
				}
			}
		}(p, i)

		// Consumer.
		go func(q *Queue[int], base int) {
			defer wg.Done()
			defer q.Close(false)
			received := 0
			for received < eventsPerPipe {
				b, err := q.Get(8)
				if err != nil {
					return
				}
				for k := 0; k < b.Count(); k++ {
					v := b.Entry(k)
					// All events received must come from this pipeline.
					if v/1000 != base {
						t.Errorf("pipeline %d received event from pipeline %d (%d)", base, v/1000, v)
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
		t.Fatal("test did not finish in time")
	}

	assert.Equal(t, poolSize, pool.Available(), "all slots should be released")
}
