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

package pooledqueue

// batch is a queue.Batch[T] over a (possibly non-contiguous) slice of slot
// indices into the pool's backing array.
//
// Concurrency: a batch instance flows along a strict hand-off path —
// Queue.Get creates it, the eventConsumer hands it to an output worker
// via a synchronizing channel send, and the worker is the sole caller
// of FreeEntries (via newBatch in the pipeline package) and Done. Each
// hand-off is a happens-before edge, so the non-atomic state below
// (`freed`, and the post-Done sweep state) is safe without atomic ops.
// The only multi-goroutine interaction is the sweep of pendingBatches
// at the tail of Done, which is serialized through Queue.mu.
type batch[T any] struct {
	queue   *Queue[T]
	indices []int
	freed   bool // true after FreeEntries has cleared the events; read/written by a single goroutine, ordering provided by the consumer→worker channel hand-off

	// next links this batch into Queue.pendingHead/pendingTail, an intrusive
	// FIFO of in-flight batches in publish order. Set under Queue.mu in Get
	// and only cleared by Done's sweep as the front of the list drains.
	next *batch[T]

	// Filled in by Done() before marking the batch ready. The owning Queue
	// invokes these in publish order as the pending list's prefix drains.
	// Reads happen under Queue.mu in the sweep, after `done` is set true.
	done         bool
	ackProducers []*producer[T]
	ackCounts    []int
}

// Count returns the number of events in the batch.
func (b *batch[T]) Count() int { return len(b.indices) }

// Entry returns the i-th event. Must not be called after FreeEntries.
func (b *batch[T]) Entry(i int) T {
	return b.queue.pool.storage[b.indices[i]].event
}

// FreeEntries clears the event field in each slot to allow the GC to collect
// the event data while the batch is still in flight at the output. The slot
// itself is not yet returned to the pool; that happens in Done.
func (b *batch[T]) FreeEntries() {
	if b.freed {
		return
	}
	var zero T
	for _, i := range b.indices {
		b.queue.pool.storage[i].event = zero
	}
	b.freed = true
}

// Done acknowledges the batch. Slots are returned to the pool immediately so
// new publishes can proceed, but producer ACK callbacks are deferred until
// every earlier in-flight batch has also been Done()'d. This matches
// memqueue's ackLoop semantics: ACK callbacks fire in publish order so order-
// sensitive consumers (e.g. filestream's registry tracker) see counts that
// map cleanly onto the events they were published for.
func (b *batch[T]) Done() {
	pool := b.queue.pool

	// Walk slots: collect per-producer counts, clear slot state. Most batches
	// come from a single producer so the linear search stays small.
	var zero T
	var ackProducers []*producer[T]
	var ackCounts []int
	for _, i := range b.indices {
		s := &pool.storage[i]
		if s.producer != nil {
			found := false
			for j, p := range ackProducers {
				if p == s.producer {
					ackCounts[j]++
					found = true
					break
				}
			}
			if !found {
				ackProducers = append(ackProducers, s.producer)
				ackCounts = append(ackCounts, 1)
			}
		}
		if !b.freed {
			s.event = zero
		}
		s.producer = nil
		s.next = -1
	}

	// Return slots to the pool before doing anything else so blocked
	// producers can make progress regardless of where this batch is in the
	// pending list.
	pool.observer.RemoveEvents(len(b.indices), 0)
	for _, i := range b.indices {
		pool.free <- i
	}

	// Mark this batch ready and harvest the contiguous prefix of done
	// batches from the front of the queue's pending list.
	b.ackProducers = ackProducers
	b.ackCounts = ackCounts

	q := b.queue
	q.mu.Lock()
	b.done = true
	// Walk the pending FIFO from the head, peeling off every batch that has
	// completed. toAck holds the prefix to ACK in publish order outside the
	// lock. Anything from a not-yet-done batch onward stays in the list.
	var toAckHead, toAckTail *batch[T]
	for q.pendingHead != nil && q.pendingHead.done {
		ready := q.pendingHead
		q.pendingHead = ready.next
		ready.next = nil
		if toAckTail == nil {
			toAckHead = ready
		} else {
			toAckTail.next = ready
		}
		toAckTail = ready
	}
	if q.pendingHead == nil {
		q.pendingTail = nil
	}
	q.maybeMarkDone()
	forced := q.forced.Load()
	q.mu.Unlock()

	// Slots are already back in the pool above; the remaining work is
	// invoking producer ACK callbacks for any batches whose turn in the
	// publish-order FIFO has come up.
	//
	// Suppress ACK callbacks once the queue has been force-closed.
	// Force-close means the caller explicitly abandoned in-flight events
	// (Close(true) released FIFO slots without acking them); reporting
	// ACKs for the parallel set of in-flight batches that were already
	// out at workers would be inconsistent and could mislead
	// order-sensitive consumers (e.g. filestream's registry tracker).
	// This matches memqueue's behaviour: its ackLoop exits on force-
	// close and no further producer ACK callbacks fire.
	if forced {
		return
	}

	// Invoke ACK callbacks outside the lock in publish (Get) order. A
	// callback that re-publishes through this queue will be free to take the
	// pool/queue locks without deadlocking us. The drained batches are no
	// longer reachable from the queue, so visiting them here is safe.
	for ab := toAckHead; ab != nil; ab = ab.next {
		for i, p := range ab.ackProducers {
			if p.cfg.ACK != nil {
				p.cfg.ACK(ab.ackCounts[i])
			}
		}
	}
}

// Release returns this batch's slot indices to the pool's free list
// without firing producer ACK callbacks. Used by the pipeline on
// shutdown when the consumer is abandoning a batch it cannot deliver;
// pool slots must be reclaimed (otherwise the pool's effective capacity
// shrinks for the process lifetime) but the producer must not be told
// the events were delivered.
//
// Release is safe to call concurrently with Done on different batches;
// it takes Queue.mu to remove this batch from the pending FIFO if it's
// still there. Calling Release on a batch whose Done has already run
// (slots already in pool.free) would double-release, so Release should
// only be called on batches the consumer is *abandoning* — never after
// any Done/Drop path.
//
// Caller contract — IMPORTANT: see queue.Batch.Release. Release must
// only be invoked when no further batches from the same producer are
// in flight. In this repo Release is reached only on pipeline
// shutdown, where that invariant holds.
func (b *batch[T]) Release() {
	pool := b.queue.pool

	// Clear slot state and gather indices for release. Mirrors the
	// slot-cleanup section of Done but skips the ackProducers/ackCounts
	// collection — we explicitly do not want to fire callbacks.
	var zero T
	for _, i := range b.indices {
		s := &pool.storage[i]
		if !b.freed {
			s.event = zero
		}
		s.producer = nil
		s.next = -1
	}

	// Return slots to the pool. pool.free's buffer is sized to the slot
	// count so this never blocks.
	pool.observer.RemoveEvents(len(b.indices), 0)
	for _, i := range b.indices {
		pool.free <- i
	}

	// Remove the batch from the pending FIFO if it's still there.
	// Done's sweep relies on the per-batch `done` flag and walks from
	// the head; for a Released batch we just splice it out wherever it
	// sits so a later Done on an earlier batch doesn't get stuck behind
	// us (and so the sweep can still drain the prefix that's ready).
	q := b.queue
	q.mu.Lock()
	var prev *batch[T]
	for cur := q.pendingHead; cur != nil; cur = cur.next {
		if cur == b {
			if prev == nil {
				q.pendingHead = cur.next
			} else {
				prev.next = cur.next
			}
			if q.pendingTail == cur {
				q.pendingTail = prev
			}
			break
		}
		prev = cur
	}
	b.next = nil
	q.maybeMarkDone()
	q.mu.Unlock()
}
