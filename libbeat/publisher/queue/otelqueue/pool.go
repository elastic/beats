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

	"github.com/elastic/beats/v7/libbeat/publisher/queue"
)

// slot stores one event and the per-pipeline linked-list metadata used to
// thread events from the same pipeline together. The actual FIFO heads/tails
// live on each Queue; this struct just carries the "next" link plus the
// producer needed for ACK callbacks.
type slot[T any] struct {
	event      T
	next       int // index of the next slot in the owning pipeline's FIFO, or -1
	producer   *producer[T]
	producerID uint64
}

// Pool is the shared backing storage for events from multiple pipelines.
// Slots are allocated from a free list (a buffered channel of indices),
// which also functions as the counting semaphore that bounds the total
// number of events live across all connected pipelines.
type Pool[T any] struct {
	settings Settings
	observer queue.Observer

	storage []slot[T]
	free    chan int

	closeOnce sync.Once
	closed    chan struct{}

	// queues is the set of Queue façades currently connected to the pool. It
	// is used to broadcast a shutdown notification when the pool itself is
	// closed and is the authoritative source for ConnectedQueues(); individual
	// Queue.Close calls remove themselves via disconnect.
	mu     sync.Mutex
	queues map[*Queue[T]]struct{}
}

// NewPool returns an initialized pool with all slots free.
//
// observer may be nil; if non-nil the pool will report metrics through it
// (MaxEvents on creation; AddEvent / ConsumeEvents / RemoveEvents as the
// queues run).
func NewPool[T any](settings Settings, observer queue.Observer) *Pool[T] {
	if settings.Events <= 0 {
		settings.Events = 1
	}
	if observer == nil {
		observer = queue.NewQueueObserver(nil) // nilObserver
	}

	p := &Pool[T]{
		settings: settings,
		observer: observer,
		storage:  make([]slot[T], settings.Events),
		free:     make(chan int, settings.Events),
		closed:   make(chan struct{}),
		queues:   make(map[*Queue[T]]struct{}),
	}
	for i := 0; i < settings.Events; i++ {
		p.free <- i
	}
	p.observer.MaxEvents(settings.Events)
	return p
}

// Connect returns a new per-pipeline Queue façade backed by this pool. Each
// connected pipeline must call (*Queue).Close when it is finished; the pool is
// only safe to call Shutdown on once every connected queue is closed.
func (p *Pool[T]) Connect() *Queue[T] {
	q := newQueue(p)
	p.mu.Lock()
	p.queues[q] = struct{}{}
	p.mu.Unlock()
	return q
}

// Shutdown closes the pool, signaling any pending Publish calls to return.
// Per-pipeline queues should normally be closed first; Shutdown is the
// pool-level final teardown.
func (p *Pool[T]) Shutdown() {
	p.closeOnce.Do(func() {
		close(p.closed)
		// Wake any queues blocked in Get.
		p.mu.Lock()
		queues := make([]*Queue[T], 0, len(p.queues))
		for q := range p.queues {
			queues = append(queues, q)
		}
		p.mu.Unlock()
		for _, q := range queues {
			q.signal()
		}
	})
}

// Capacity returns the pool's total slot count.
func (p *Pool[T]) Capacity() int { return p.settings.Events }

// Available returns the number of currently free slots. Useful for tests and
// observability.
func (p *Pool[T]) Available() int { return len(p.free) }

// ConnectedQueues returns the number of Queue façades currently connected.
func (p *Pool[T]) ConnectedQueues() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.queues)
}

// disconnect is called by Queue.Close to unregister itself from the pool.
func (p *Pool[T]) disconnect(q *Queue[T]) {
	p.mu.Lock()
	delete(p.queues, q)
	p.mu.Unlock()
}
