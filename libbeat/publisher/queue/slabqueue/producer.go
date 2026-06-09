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

	nextID atomic.Uint64
	closed atomic.Bool

	// published counts events successfully threaded onto the FIFO; resolved
	// counts events whose batches have completed — either acknowledged (Done)
	// or abandoned (Release). Abandoned events count as resolved so a producer
	// whose tail batch is Released on shutdown does not strand ackWait. When
	// the producer is closed and resolved has caught up with published, ackWait
	// is closed. ackWait is also closed unconditionally when the queue is
	// force-closed, so a waiter can never hang past teardown — see Queue.Close.
	published atomic.Uint64
	resolved  atomic.Uint64
	ackWait   chan struct{}
	ackOnce   sync.Once
}

// Publish adds an entry, blocking if the pool is empty until a slot is freed
// or the queue/pool is closed.
func (p *producer[T]) Publish(entry T) (queue.EntryID, bool) {
	if p.closed.Load() {
		return 0, false
	}
	var slotIdx int
	select {
	case slotIdx = <-p.queue.pool.free:
	case <-p.queue.closeCh:
		return 0, false
	case <-p.queue.pool.closed:
		return 0, false
	}
	return p.fill(entry, slotIdx)
}

// TryPublish adds an entry only if a slot is immediately available; it never
// blocks.
func (p *producer[T]) TryPublish(entry T) (queue.EntryID, bool) {
	if p.closed.Load() {
		return 0, false
	}
	var slotIdx int
	select {
	case slotIdx = <-p.queue.pool.free:
	default:
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

// resolveN advances this producer's resolved count (events acked or abandoned)
// and closes ackWait if the producer is now closed and fully drained. Called
// from batch.Done (for acked events) and batch.Release (for both abandoned and
// drained-successor events).
func (p *producer[T]) resolveN(n int) {
	p.resolved.Add(uint64(n)) //nolint:gosec // G115: n is a batch event count, always a small positive value
	p.maybeCloseAckWait()
}

// maybeCloseAckWait closes ackWait exactly once, when the producer has been
// closed and every published event has been resolved, and unregisters the
// producer from the queue's force-close fan-out set (it no longer needs it).
func (p *producer[T]) maybeCloseAckWait() {
	if p.closed.Load() && p.resolved.Load() >= p.published.Load() {
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
	s := &pool.storage[slotIdx]
	s.event = entry
	s.next = -1
	s.producer = p
	s.producerID = id

	q := p.queue
	q.mu.Lock()
	if q.closing {
		// Queue closed between acquire and fill. Return the slot.
		var zero T
		s.event = zero
		s.producer = nil
		q.mu.Unlock()
		pool.free <- slotIdx
		return 0, false
	}
	if q.tail == -1 {
		q.head = slotIdx
	} else {
		pool.storage[q.tail].next = slotIdx
	}
	q.tail = slotIdx
	q.count++
	q.mu.Unlock()

	// Count the event for ack-wait accounting only on the success path, so a
	// publish that loses the closing race (handled above) doesn't inflate the
	// outstanding count and strand ackWait.
	p.published.Add(1)

	pool.observer.AddEvent(0)
	q.signal()
	return queue.EntryID(id), true
}
