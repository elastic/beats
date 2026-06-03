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

import (
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
func (p *producer[T]) Close() { p.closed.Store(true) }

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

	pool.observer.AddEvent(0)
	q.signal()
	return queue.EntryID(id), true
}
