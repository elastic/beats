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
	return b.queue.pool.slot(b.indices[i]).event
}

// FreeEntries clears the event field in each slot to allow the GC to collect
// the event data while the batch is still in flight at the output. The slot
// itself is not yet returned to the pool; that happens in Done.
//
// FreeEntries returns early if the batch has already been recycled
// (b.queue == nil). Calling FreeEntries on a recycled batch through a
// stale reference is a contract violation, but we silently no-op rather
// than corrupt the slots a fresh consumer may now hold.
func (b *batch[T]) FreeEntries() {
	if b.freed || b.queue == nil {
		return
	}
	var zero T
	for _, i := range b.indices {
		b.queue.pool.slot(i).event = zero
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
	// Double-completion guard. After recycling, b.queue is cleared;
	// any second call into Done/Release on a stale reference hits
	// this and returns harmlessly. Contract: callers must invoke
	// Done (or Release) exactly once.
	if b.queue == nil {
		return
	}
	pool := b.queue.pool

	// Walk slots: collect per-producer counts, clear slot state. Most
	// batches come from a single producer so the linear search stays
	// small. Reuse b's own slice backing arrays (set by Get from a
	// recycled batch) to avoid allocating per Done. The directory only
	// grows under growMu and always covers an index this batch holds, so
	// load it once instead of per slot.
	var zero T
	b.ackProducers = b.ackProducers[:0]
	b.ackCounts = b.ackCounts[:0]
	d := pool.dir.Load()
	for _, i := range b.indices {
		s := d.slot(i)
		if s.producer != nil {
			found := false
			for j, p := range b.ackProducers {
				if p == s.producer {
					b.ackCounts[j]++
					found = true
					break
				}
			}
			if !found {
				b.ackProducers = append(b.ackProducers, s.producer)
				b.ackCounts = append(b.ackCounts, 1)
			}
		}
		if !b.freed {
			s.event = zero
		}
		s.producer = nil
		s.next = -1
	}

	// Return slots to the pool before doing anything else so blocked
	// producers can make progress regardless of where this batch is in
	// the pending list.
	pool.observer.RemoveEvents(len(b.indices), 0)
	n := len(b.indices)
	pool.releaseSlots(b.indices)
	// These events left circulation; return their per-queue budget so producers
	// blocked on this queue's cap can proceed.
	b.queue.releaseLive(n)

	// Mark this batch done and drain the now-ready prefix of the pending FIFO.
	// forced is read under the same lock so the snapshot is consistent with the
	// drain. Slots are already back in the pool; the only work left is firing
	// the drained prefix's ACK callbacks (in publish order) and recycling them.
	q := b.queue
	q.mu.Lock()
	b.done = true
	toAck := q.drainReadyLocked()
	forced := q.forced.Load()
	q.mu.Unlock()

	pool.fireAndRecycle(toAck, forced)
}

// fireAndRecycle invokes producer ACK callbacks for each batch in the
// publish-order list, then returns the batches to the recycle pool. The list
// comes from Queue.drainReadyLocked and is no longer reachable from the queue,
// so visiting it after q.mu is released is safe; a callback that re-publishes
// through the queue is free to take the pool/queue locks without deadlocking.
//
// ACK callbacks are suppressed once the queue has been force-closed: the caller
// explicitly abandoned in-flight events (Close(true) released FIFO slots without
// acking them), so reporting ACKs for the parallel set of batches already out at
// workers would be inconsistent and could mislead order-sensitive consumers
// (e.g. filestream's registry tracker). This matches memqueue, whose ackLoop
// exits on force-close and fires no further callbacks.
func (p *Pool[T]) fireAndRecycle(head *batch[T], forced bool) {
	if !forced {
		for ab := head; ab != nil; ab = ab.next {
			for i, pr := range ab.ackProducers {
				if pr.cfg.ACK != nil {
					pr.cfg.ACK(ab.ackCounts[i])
				}
			}
		}
	}
	// Capture next before putBatch clears the .next field.
	for ab := head; ab != nil; {
		next := ab.next
		p.putBatch(ab)
		ab = next
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
	// Double-completion guard. See Done's comment — same contract.
	if b.queue == nil {
		return
	}
	pool := b.queue.pool

	// Clear slot state. Mirrors the slot-cleanup section of Done but skips the
	// ackProducers/ackCounts collection — we explicitly do not fire callbacks.
	var zero T
	d := pool.dir.Load()
	for _, i := range b.indices {
		s := d.slot(i)
		if !b.freed {
			s.event = zero
		}
		s.producer = nil
		s.next = -1
	}

	// Return slots to the pool.
	pool.observer.RemoveEvents(len(b.indices), 0)
	n := len(b.indices)
	pool.releaseSlots(b.indices)
	// Return the per-queue budget for the abandoned events.
	b.queue.releaseLive(n)

	// Remove the batch from the pending FIFO if it's still there. Done's sweep
	// relies on the per-batch `done` flag and walks from the head; for a Released
	// batch we splice it out wherever it sits, then drain the now-exposed ready
	// prefix exactly as Done would — otherwise completed-but-stalled successors
	// behind us would sit in pendingHead indefinitely, blocking q.Done() and
	// leaking their ACK callbacks + batch objects.
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
	toAck := q.drainReadyLocked()
	forced := q.forced.Load()
	q.mu.Unlock()

	pool.fireAndRecycle(toAck, forced)

	// b itself is no longer reachable from the queue and the caller
	// has released it — safe to recycle. It is never part of toAck (it was
	// spliced out above and never marked done), so it is recycled here without
	// firing its producer callbacks, which is the whole point of Release.
	pool.putBatch(b)
}
