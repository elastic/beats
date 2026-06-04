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
	"sync/atomic"

	"github.com/elastic/beats/v7/libbeat/publisher/queue"
)

// magazineMaxCap is the absolute upper bound on slots a single producer may
// pre-claim from the shared depot regardless of pool capacity.
const magazineMaxCap = 256

// producer publishes events into one Queue. It acquires a slot from the
// shared pool's free list (blocking when the pool is empty), writes the event
// into that slot, and threads the slot onto its queue's per-pipeline FIFO.
//
// Each producer holds a magazine: a small pre-claimed stack of slot indices
// (Bonwick & Adams §3.1). Publish pops from the magazine without touching the
// shared depot; the depot is only accessed to bulk-refill when
// the magazine runs dry.
type producer[T any] struct {
	queue          *Queue[T]
	cfg            queue.ProducerConfig
	magazine       []int // pre-claimed slot indices; drained LIFO before touching pool.free
	magazineCeiled int   // cached ceiling; recomputed when connectedQueues changes
	cachedQueues   int   // connectedQueues value at last ceiling computation

	nextID atomic.Uint64
	closed atomic.Bool
}

// Publish adds an entry, blocking if the pool is empty until a slot is freed
// or the queue/pool is closed.
func (p *producer[T]) Publish(entry T) (queue.EntryID, bool) {
	if p.closed.Load() {
		return 0, false
	}
	slotIdx, ok := p.acquireSlot()
	if !ok {
		return 0, false
	}
	return p.fill(entry, slotIdx)
}

// ceiling returns the maximum slots this producer may hold in its magazine:
// pool.Capacity()/(2*connectedQueues), capped at magazineMaxCap. The result
// is cached and only recomputed when the connected-queue count changes, so
// the common path (stable receiver count) pays no lock overhead.
func (p *producer[T]) ceiling() int {
	pool := p.queue.pool
	queues := pool.ConnectedQueues()
	if queues == p.cachedQueues {
		return p.magazineCeiled
	}
	p.cachedQueues = queues
	if queues == 0 {
		queues = 1
	}
	c := pool.Capacity() / (2 * queues)
	if c > magazineMaxCap {
		c = magazineMaxCap
	}
	if c < 1 {
		c = 1
	}
	p.magazineCeiled = c
	return c
}

// acquireSlot returns a slot index, refilling the magazine from pool.free when
// it is empty. Returns false if the queue or pool is closed.
//
// Refill limit = min(len(pool.free)/2, ceiling()).
func (p *producer[T]) acquireSlot() (int, bool) {
	if len(p.magazine) > 0 {
		idx := p.magazine[len(p.magazine)-1]
		p.magazine = p.magazine[:len(p.magazine)-1]
		return idx, true
	}

	// Magazine empty; bulk-fill from the depot. Limit to the smaller of
	// half the currently visible free slots and the per-producer ceiling.
	pool := p.queue.pool
	limit := len(pool.free) / 2
	if c := p.ceiling(); limit > c {
		limit = c
	}

	if limit > 0 {
		p.magazine = p.magazine[:cap(p.magazine)] // reuse pre-allocated backing array
		filled := 0
	fill:
		for filled < limit {
			select {
			case idx := <-pool.free:
				p.magazine[filled] = idx
				filled++
			default:
				break fill
			}
		}
		if filled > 0 {
			p.magazine = p.magazine[:filled]
			idx := p.magazine[filled-1]
			p.magazine = p.magazine[:filled-1]
			return idx, true
		}
		p.magazine = p.magazine[:0]
	}

	// Depot was empty — block until a slot is available or we close.
	select {
	case idx := <-pool.free:
		return idx, true
	case <-p.queue.closeCh:
		return 0, false
	case <-pool.closed:
		return 0, false
	}
}

// TryPublish adds an entry only if a slot is immediately available; it never
// blocks.
func (p *producer[T]) TryPublish(entry T) (queue.EntryID, bool) {
	if p.closed.Load() {
		return 0, false
	}
	// Check the local magazine first before touching the shared channel.
	if len(p.magazine) > 0 {
		idx := p.magazine[len(p.magazine)-1]
		p.magazine = p.magazine[:len(p.magazine)-1]
		return p.fill(entry, idx)
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
	// Return magazine slots to the pool
	for _, idx := range p.magazine {
		p.queue.pool.free <- idx
	}
	p.magazine = p.magazine[:0]
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
		// Queue closed between acquire and fill. Return the current slot and
		// the entire magazine.
		var zero T
		s.event = zero
		s.producer = nil
		q.mu.Unlock()
		pool.free <- slotIdx
		for _, idx := range p.magazine {
			pool.free <- idx
		}
		p.magazine = p.magazine[:0]
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

	pool.observer.AddEvent(0)
	q.signal()
	return queue.EntryID(id), true
}
