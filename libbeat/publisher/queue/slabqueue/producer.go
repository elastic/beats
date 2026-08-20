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
	"sync/atomic"

	"github.com/elastic/beats/v7/libbeat/publisher/queue"
)

// producer publishes events into one Queue. It acquires a slot from the
// shared pool's free list (blocking when the pool is empty), writes the event
// into that slot, and threads the slot onto its queue's per-pipeline FIFO.
type producer[T any] struct {
	queue *Queue[T]
	cfg   queue.ProducerConfig

	// home is this producer's free-list shard hint, assigned once at creation.
	// It only steers where acquire starts scanning, so it never needs to change
	// even as the pool grows or shrinks.
	home int

	nextID atomic.Uint64
	closed atomic.Bool

	// published counts in-flight + enqueued events; finished counts events whose
	// batches have completed (acked via Done or abandoned via Release). When the
	// producer is closed and finished catches up with published, ackWait closes —
	// or unconditionally on force-close, so a waiter never hangs (see Queue.Close).
	// published is counted before enqueue (and rolled back on failure, see
	// unpublish) so a concurrent Close can't see finished >= published while an
	// event is still in flight.
	published atomic.Uint64
	finished  atomic.Uint64
	ackWait   chan struct{}
	ackOnce   sync.Once
}

// Publish adds an entry, blocking until both this queue is under its per-queue
// cap and the shared pool has a free slot, or the queue/pool is closed.
//
// The per-queue cap is reserved first, before acquiring a pool slot: a queue at
// its cap must not consume pool slots it cannot keep, which would starve other
// queues sharing the pool.
func (p *producer[T]) Publish(entry T) (queue.EntryID, bool) {
	if p.closed.Load() {
		return 0, false
	}
	// Count the event before enqueuing it, else a concurrent Close could see
	// finished >= published and close ackWait while it's still in flight. Each
	// failure path below calls unpublish to undo this.
	p.published.Add(1)
	if !p.queue.reserve() {
		p.unpublish()
		return 0, false
	}
	slotIdx, ok := p.queue.pool.acquire(p.home, p.queue.closeCh)
	if !ok {
		p.queue.releaseLive(1)
		p.unpublish()
		return 0, false
	}
	return p.fill(entry, slotIdx)
}

// TryPublish adds an entry only if this queue is under its per-queue cap and a
// pool slot is immediately available; it never blocks.
func (p *producer[T]) TryPublish(entry T) (queue.EntryID, bool) {
	if p.closed.Load() {
		return 0, false
	}
	// Count the event before enqueuing it, else a concurrent Close could see
	// finished >= published and close ackWait while it's still in flight. Each
	// failure path below calls unpublish to undo this.
	p.published.Add(1)
	if !p.queue.tryReserve() {
		p.unpublish()
		return 0, false
	}
	slotIdx, ok := p.queue.pool.free.tryGrab(p.home)
	if !ok {
		p.queue.releaseLive(1)
		p.unpublish()
		return 0, false
	}
	return p.fill(entry, slotIdx)
}

// Close marks the producer as closed. Subsequent Publish/TryPublish return
// (0, false). The queue may still deliver ACK callbacks for events this
// producer published before Close was called.
func (p *producer[T]) Close() {
	p.closed.Store(true)
	// The events published before Close may already all be acked, in which
	// case no future Done/Release will fire to close ackWait — check now. The
	// producer stays registered with the queue until its ackWait actually
	// closes, so a force-close before its events drain can still unblock it.
	p.maybeCloseAckWait()
}

// ACKWaitChan returns the channel closed once the producer is closed and all
// of its events have been acknowledged (or immediately on force-close).
func (p *producer[T]) ACKWaitChan() <-chan struct{} { return p.ackWait }

// finishN advances this producer's finished count (events acked or abandoned)
// and closes ackWait if the producer is now closed and fully drained. Called
// from batch.Done (for acked events) and batch.Release (for both abandoned and
// drained-successor events).
func (p *producer[T]) finishN(n int) {
	p.finished.Add(uint64(n)) //nolint:gosec // G115: n is a batch event count, always a small positive value
	p.maybeCloseAckWait()
}

// maybeCloseAckWait closes ackWait exactly once, when the producer has been
// closed and every published event has finished, and unregisters the
// producer from the queue's force-close fan-out set (it no longer needs it).
func (p *producer[T]) maybeCloseAckWait() {
	if p.closed.Load() && p.finished.Load() >= p.published.Load() {
		p.ackOnce.Do(func() {
			close(p.ackWait)
			p.queue.removeProducer(p)
		})
	}
}

// forceCloseAckWait closes ackWait unconditionally. Used by Queue.Close to
// guarantee a waiter unblocks on queue teardown even though force-close
// suppresses the per-event ACK callbacks that would otherwise close it.
func (p *producer[T]) forceCloseAckWait() {
	p.ackOnce.Do(func() { close(p.ackWait) })
}

// fill stores entry in the given slot and threads it onto the queue's FIFO.
// If the queue is already closing, the slot is returned to the pool and the
// publish fails.
func (p *producer[T]) fill(entry T, slotIdx int) (queue.EntryID, bool) {
	id := p.nextID.Add(1)
	pool := p.queue.pool
	s := pool.slot(slotIdx)
	s.event = entry
	s.next = -1
	s.producer = p

	q := p.queue
	q.mu.Lock()
	if q.closing {
		// Queue closed between acquire and fill. Return the slot and the
		// per-queue budget unit reserved for it, and undo the publish accounting
		// (the caller reserved it before this event could be enqueued).
		var zero T
		s.event = zero
		s.producer = nil
		q.mu.Unlock()
		pool.releaseSlot(slotIdx)
		q.releaseLive(1)
		p.unpublish()
		return 0, false
	}
	if q.tail == -1 {
		q.head = slotIdx
	} else {
		pool.slot(q.tail).next = slotIdx
	}
	q.tail = slotIdx
	q.count++
	q.mu.Unlock()

	// published was already incremented by Publish/TryPublish before this event
	// could be enqueued, so no further accounting is needed on the success path.
	pool.observer.AddEvent(0)
	q.signal()
	return queue.EntryID(id), true
}

// unpublish undoes the published increment taken at the start of a publish that
// ultimately failed to enqueue its event, then re-checks ackWait: if the
// producer was closed during the failed publish, maybeCloseAckWait may now have
// to close ackWait that it conservatively left open while published was high.
func (p *producer[T]) unpublish() {
	p.published.Add(^uint64(0)) // -1
	p.maybeCloseAckWait()
}
