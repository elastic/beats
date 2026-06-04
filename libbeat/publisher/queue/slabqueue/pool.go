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

	// batchPool recycles *batch[T] objects. Queue.Get pulls a batch from
	// here and resets it instead of heap-allocating, and batch.Done /
	// batch.Release return finished batches once the queue and consumer
	// have released them. The backing arrays for `indices`,
	// `ackProducers`, and `ackCounts` are kept across recycles, so the
	// hot path is near zero-alloc after warmup.
	//
	// sync.Pool is per-P-local under the hood; cross-receiver contention
	// is only paid in the rare case a P falls back to the shared store.
	// In normal operation each consumer's Get and each worker's Done
	// hit their own goroutine's local pool.
	batchPool sync.Pool

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
	p.batchPool.New = func() any { return &batch[T]{} }
	for i := 0; i < settings.Events; i++ {
		p.free <- i
	}
	p.observer.MaxEvents(settings.Events)
	return p
}

// getBatch returns a recycled or freshly allocated *batch[T]. Callers
// are responsible for populating it via fillBatch before exposing it.
func (p *Pool[T]) getBatch() *batch[T] {
	return p.batchPool.Get().(*batch[T])
}

// putBatch resets a batch's transient state and returns it to the
// recycle pool. The backing arrays for indices/ackProducers/ackCounts
// are kept so subsequent uses reuse the same memory. The caller must
// guarantee that no one still holds a usable reference to b.
func (p *Pool[T]) putBatch(b *batch[T]) {
	// Clear strong references so the GC isn't held up by recycled
	// batches and so a stray call into a recycled batch fails fast
	// (b.queue == nil triggers the double-completion guards).
	b.queue = nil
	b.indices = b.indices[:0]
	b.ackProducers = b.ackProducers[:0]
	b.ackCounts = b.ackCounts[:0]
	b.next = nil
	b.done = false
	b.freed = false
	p.batchPool.Put(b)
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

// Shutdown closes the pool and force-closes every connected Queue. Callers
// typically Close each queue individually first; Shutdown then completes the
// pool-level teardown. Calling Shutdown without a prior per-queue Close
// is still safe — the force-close path here ensures each queue's doneCh
// fires so anything waiting on q.Done() unblocks. Without this, the pool
// would close pool.closed (unblocking Get) but leave doneCh unset because
// q.closing was never assigned, deadlocking any q.Done() observer.
func (p *Pool[T]) Shutdown() {
	p.closeOnce.Do(func() {
		close(p.closed)
		p.mu.Lock()
		queues := make([]*Queue[T], 0, len(p.queues))
		for q := range p.queues {
			queues = append(queues, q)
		}
		p.mu.Unlock()
		// Close is idempotent, so queues the caller already closed are
		// not affected. Force is used because pool.Shutdown is itself a
		// "we're done with this pool" signal — no graceful drain.
		for _, q := range queues {
			_ = q.Close(true)
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
