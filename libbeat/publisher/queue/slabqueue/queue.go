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
	"io"
	"sync"
	"sync/atomic"

	"github.com/elastic/beats/v7/libbeat/publisher/queue"
)

// Queue is a per-pipeline façade over a shared Pool. It owns the FIFO of slot
// indices belonging to one pipeline; the actual event storage lives in the
// pool. Queue implements queue.Queue[T].
type Queue[T any] struct {
	pool *Pool[T]

	mu      sync.Mutex
	head    int // index of the head slot in this pipeline's FIFO, or -1
	tail    int // index of the tail slot, or -1
	count   int
	closing bool

	// pendingHead/pendingTail is the intrusive FIFO of batches that have been
	// returned from Get but not yet Done()'d, threaded through batch.next in
	// publish (== Get) order. We use it to fire producer ACK callbacks in
	// publish order even when the output's workers complete batches
	// concurrently and out of order; this preserves the invariant memqueue
	// maintains via its ackLoop pendingBatches list. Done() also blocks until
	// this list drains.
	pendingHead, pendingTail *batch[T]

	// notify wakes Get when new events arrive.
	notify chan struct{}

	closeOnce sync.Once
	closeCh   chan struct{} // closed on Close
	doneOnce  sync.Once
	doneCh    chan struct{} // closed when fully drained or force-closed
	forced    atomic.Bool
}

func newQueue[T any](pool *Pool[T]) *Queue[T] {
	return &Queue[T]{
		pool:    pool,
		head:    -1,
		tail:    -1,
		notify:  make(chan struct{}, 1),
		closeCh: make(chan struct{}),
		doneCh:  make(chan struct{}),
	}
}

// Producer returns a producer that publishes to this queue. The caller must
// call producer.Close() when done to return any pre-claimed magazine slots to
// the pool; failing to do so leaks slots until the pool is shut down.
func (q *Queue[T]) Producer(cfg queue.ProducerConfig) queue.Producer[T] {
	return &producer[T]{queue: q, cfg: cfg, magazine: make([]int, 0, magazineMaxCap)}
}

// Get blocks until at least one event is available (or the queue is closed)
// and returns a batch of up to maxEvents. If maxEvents <= 0, all currently
// queued events are returned.
func (q *Queue[T]) Get(maxEvents int) (queue.Batch[T], error) {
	for {
		q.mu.Lock()
		if q.count > 0 {
			n := q.count
			if maxEvents > 0 && maxEvents < n {
				n = maxEvents
			}
			// Pull a recycled batch from the pool. Its slices retain
			// their backing arrays from previous uses; we reset
			// lengths and append into them. The batch is owned solely
			// by this Queue/consumer/worker chain until Done/Release
			// returns it to the pool.
			b := q.pool.getBatch()
			b.queue = q
			b.indices = b.indices[:0]
			b.done = false
			b.freed = false
			b.next = nil
			cur := q.head
			for i := 0; i < n; i++ {
				b.indices = append(b.indices, cur)
				cur = q.pool.storage[cur].next
			}
			q.head = cur
			if cur == -1 {
				q.tail = -1
			}
			q.count -= n
			if q.pendingTail != nil {
				q.pendingTail.next = b
			} else {
				q.pendingHead = b
			}
			q.pendingTail = b
			q.mu.Unlock()
			q.pool.observer.ConsumeEvents(n, 0)
			return b, nil
		}
		if q.forced.Load() || q.closing {
			q.mu.Unlock()
			return nil, io.EOF
		}
		q.mu.Unlock()

		select {
		case <-q.notify:
			// Loop and try to drain.
		case <-q.closeCh:
			// Loop; the closing/forced flags are now set, the next iteration
			// will either drain remaining events or return EOF.
		case <-q.pool.closed:
			return nil, io.EOF
		}
	}
}

// Close shuts down the queue.
//
// With force=false the queue continues to deliver already-queued events;
// once they all drain (Get returns them and the resulting batches are
// acked), Done() unblocks and producer ACK callbacks fire normally for
// every event that was delivered.
//
// With force=true any events still in the FIFO are released back to the
// pool and Done() unblocks immediately. ACK callbacks are suppressed for
// every event affected by the force-close: the released FIFO events get
// no callback (they were abandoned), and in-flight batches still held by
// workers will release their slots when batch.Done eventually runs but
// will not invoke producer ACK callbacks. This matches memqueue's
// force-close semantics — the caller explicitly gave up on these events
// and consumers that depend on ACK ordering would be misled by partial
// acks for an abandoned set.
func (q *Queue[T]) Close(force bool) error {
	if force {
		q.forced.Store(true)
	}
	q.mu.Lock()
	if !q.closing {
		q.closing = true
	}
	var releaseIndices []int
	if force {
		if q.count > 0 {
			// Walk the FIFO and gather the slots so we can release them back to
			// the pool below (outside the lock, since pool.free is a channel).
			cur := q.head
			for cur != -1 {
				releaseIndices = append(releaseIndices, cur)
				cur = q.pool.storage[cur].next
			}
			q.head = -1
			q.tail = -1
			q.count = 0
		}
		// Drop the in-flight batch list. The batches themselves are still
		// held by workers and will run their Done callback in the normal
		// course; that path observes q.forced and suppresses ACK
		// callbacks (see batch.Done). Clearing the list here ensures
		// "force-closed queue holds no pending batches" as a state
		// invariant — a reader inspecting Queue internals after Close
		// won't see references that the queue logically abandoned.
		q.pendingHead = nil
		q.pendingTail = nil
	}
	q.mu.Unlock()

	// Wake any blocked Get.
	q.closeOnce.Do(func() { close(q.closeCh) })
	q.signal()

	if len(releaseIndices) > 0 {
		var zero T
		for _, i := range releaseIndices {
			q.pool.storage[i].event = zero
			q.pool.storage[i].producer = nil
			q.pool.storage[i].next = -1
		}
		q.pool.observer.RemoveEvents(len(releaseIndices), 0)
		for _, i := range releaseIndices {
			q.pool.free <- i
		}
	}

	if force {
		// Force-close: abandon any in-flight batches and unblock Done()
		// immediately. Their slots will still be returned when the consumer
		// eventually calls batch.Done().
		q.markDone()
	} else {
		// Graceful: signal Done only if the FIFO is empty and there are no
		// in-flight batches left to ack.
		q.mu.Lock()
		q.maybeMarkDone()
		q.mu.Unlock()
	}

	q.pool.disconnect(q)
	return nil
}

// Done returns a channel that is closed once the queue is fully shut down and
// drained (with force=true: immediately; with force=false: when the last
// in-flight batch is acked).
func (q *Queue[T]) Done() <-chan struct{} { return q.doneCh }

// QueueType identifies the implementation.
func (q *Queue[T]) QueueType() string { return QueueType }

// BufferConfig reports the pool's capacity. Note: this is the *shared* upper
// bound across all queues connected to the same pool, not a per-queue cap.
func (q *Queue[T]) BufferConfig() queue.BufferConfig {
	return queue.BufferConfig{MaxEvents: q.pool.settings.Events}
}

// signal wakes a goroutine blocked in Get. Non-blocking: at most one pending
// wake-up is buffered.
func (q *Queue[T]) signal() {
	select {
	case q.notify <- struct{}{}:
	default:
	}
}

// maybeMarkDone closes doneCh if the queue is closing and fully drained: no
// events left in the FIFO and no batches still in flight at the consumer. It
// must be called with q.mu held.
func (q *Queue[T]) maybeMarkDone() {
	if q.closing && q.count == 0 && q.pendingHead == nil {
		q.markDone()
	}
}

// markDone closes doneCh idempotently.
func (q *Queue[T]) markDone() {
	q.doneOnce.Do(func() { close(q.doneCh) })
}
