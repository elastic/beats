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
	"time"

	"github.com/elastic/beats/v7/libbeat/publisher/queue"
)

// DefaultGetDebounce is the coalescing window a newly connected queue starts
// with. Get waits this long after events first appear before draining the FIFO,
// so a trickle of events yields one modest batch (and one downstream Publish
// goroutine) rather than one per event. It bounds the output worker's goroutine
// fan-out to roughly downstream_round_trip / debounce without capping batch
// size, so throughput is largely unaffected.
//
// This value was tuned from the result of BenchmarkGetDebounce, which sweeps a range of
// debounce values and reports. Based on the results, 1ms was chosen.
//
// debounce=0 lets the mixed receiver case spray ~430 tiny-batch Publish goroutines,
// while any value >=~250us collapses that to ~100 with no further gain; throughput then
// declines ~1-2% per additional ms, so 1ms keeps ~98.5% of peak throughput,
// ~100 goroutines, and roughly halved CPU/event. The balance point before extra
// debounce only costs throughput.
var DefaultGetDebounce = 1 * time.Millisecond

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

	// producers is the set of open producers publishing into this queue. It
	// exists so Close can fan out and unblock every producer's ACKWaitChan on
	// (force-)close — force-close suppresses per-event ACK callbacks, so the
	// ack accounting alone would never close those channels. Producers add
	// themselves in Producer and remove themselves in Close; guarded by mu.
	producers map[*producer[T]]struct{}

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

	// Per-queue live-event cap. This bounds the events live (published but not
	// yet acked) on this one queue, independently of and in addition to the
	// shared pool budget: a producer blocks when this queue reaches its limit
	// even if the pool still has free slots. It is what lets several queues on
	// one pool each enforce their own configured size while the pool is sized to
	// the largest of them.
	//
	//   live:  events published to this queue but not yet released (FIFO +
	//          in-flight). Maintained as an atomic so the reserve fast path is
	//          lock-free, mirroring the pool free list.
	//   limit: this queue's cap (0 = unlimited, bounded only by the pool).
	//   limWaiters/limMu/limCond: park point for producers blocked on the cap,
	//          touched only on the blocking slow path.
	live       atomic.Int64
	limit      atomic.Int64
	limWaiters atomic.Int64
	limMu      sync.Mutex
	limCond    *sync.Cond

	closeOnce sync.Once
	closeCh   chan struct{} // closed on Close
	doneOnce  sync.Once
	doneCh    chan struct{} // closed when fully drained or force-closed
	forced    atomic.Bool

	debounce time.Duration // coalescing window for Get
}

func newQueue[T any](pool *Pool[T]) *Queue[T] {
	q := &Queue[T]{
		pool:      pool,
		head:      -1,
		tail:      -1,
		notify:    make(chan struct{}, 1),
		closeCh:   make(chan struct{}),
		doneCh:    make(chan struct{}),
		producers: make(map[*producer[T]]struct{}),
		debounce:  DefaultGetDebounce,
	}
	q.limCond = sync.NewCond(&q.limMu)
	return q
}

// SetTarget sets this queue's per-queue live-event cap to n, and is safe to call
// while the queue is in use. A producer blocks when this queue reaches n live
// events even if the pool still has free slots. n <= 0 means uncapped (bounded
// only by the pool). Lowering the cap takes effect lazily as in-flight events
// drain; raising it immediately unblocks any producers parked on the old cap.
//
// Setting a queue's cap also resizes the shared pool to the largest cap among
// the queues connected to it, so the pool always tracks its queues: a queue's
// cap can never be larger than the pool that backs it, and the pool shrinks when
// the queue holding the largest cap is lowered or leaves. The pool is therefore
// driven entirely through the queues — callers set per-queue caps, not the pool
// size directly.
func (q *Queue[T]) SetTarget(n int) {
	if n < 0 {
		n = 0
	}
	q.limit.Store(int64(n))
	// Resize the pool first (a raised cap grows the pool), then wake producers
	// parked on the cap so they find both the budget and a pool slot available.
	q.pool.syncTargetToQueues()
	q.wakeLimitWaiters()
}

// tryReserve takes one unit of this queue's live-event budget if it is under the
// cap, without blocking. It is the lock-free fast path: a CAS on the live
// counter, with limit==0 meaning unlimited.
func (q *Queue[T]) tryReserve() bool {
	for {
		lim := q.limit.Load()
		if lim <= 0 {
			q.live.Add(1)
			return true
		}
		cur := q.live.Load()
		if cur >= lim {
			return false
		}
		if q.live.CompareAndSwap(cur, cur+1) {
			return true
		}
	}
}

// reserve takes one unit of this queue's live-event budget, blocking until the
// queue is under its cap or the queue is closed. It returns false only when the
// queue is closing. Like the pool's acquire, it re-checks after registering as a
// waiter so a release between the failed fast path and the park is not lost.
func (q *Queue[T]) reserve() bool {
	if q.tryReserve() {
		return true
	}
	for {
		q.limMu.Lock()
		q.limWaiters.Add(1)
		if q.tryReserve() {
			q.limWaiters.Add(-1)
			q.limMu.Unlock()
			return true
		}
		if q.isClosing() {
			q.limWaiters.Add(-1)
			q.limMu.Unlock()
			return false
		}
		q.limCond.Wait()
		q.limWaiters.Add(-1)
		q.limMu.Unlock()
		if q.isClosing() {
			return false
		}
		if q.tryReserve() {
			return true
		}
	}
}

// releaseLive returns n units to this queue's live-event budget (called when
// slots are released back to the pool) and wakes a producer parked on the cap.
func (q *Queue[T]) releaseLive(n int) {
	if n <= 0 {
		return
	}
	q.live.Add(int64(-n))
	q.wakeLimitWaiters()
}

// wakeLimitWaiters wakes producers parked on the per-queue cap, but only if one
// might be waiting. The racy waiters read is safe for the same reason the pool's
// signal is: a parked producer registers as a waiter before re-checking the cap.
func (q *Queue[T]) wakeLimitWaiters() {
	if q.limWaiters.Load() == 0 {
		return
	}
	q.limMu.Lock()
	q.limCond.Broadcast()
	q.limMu.Unlock()
}

// isClosing reports whether Close has been called on this queue.
func (q *Queue[T]) isClosing() bool { return chClosed(q.closeCh) }

// Producer returns a producer that publishes to this queue. Each producer is
// given a stable home shard for the pool's free list, spread across shards by
// a pool-global counter so concurrent producers tend to land on different
// shards.
func (q *Queue[T]) Producer(cfg queue.ProducerConfig) queue.Producer[T] {
	home := int((q.pool.homeCounter.Add(1) - 1) & uint64(q.pool.free.mask)) //nolint:gosec // G115: masked by the shard count, always a small non-negative index
	p := &producer[T]{queue: q, cfg: cfg, home: home, ackWait: make(chan struct{})}
	q.mu.Lock()
	if q.closing {
		// The queue is already (force-)closing. A producer created now will
		// never see its events drain, so hand back one whose ackWait is
		// already closed rather than registering it for a fan-out that has
		// already happened.
		q.mu.Unlock()
		p.forceCloseAckWait()
		return p
	}
	q.producers[p] = struct{}{}
	q.mu.Unlock()
	return p
}

// removeProducer unregisters a producer from the force-close fan-out set. Called
// from producer.Close; safe to call for a producer that was never registered
// (e.g. one created after the queue began closing).
func (q *Queue[T]) removeProducer(p *producer[T]) {
	q.mu.Lock()
	delete(q.producers, p)
	q.mu.Unlock()
}

// Get returns a batch of events from this pipeline's FIFO. It blocks until at
// least one event is available, applies the queue's debounce coalescing window,
// then returns everything currently queued. It returns io.EOF once the queue is
// closed and drained.
func (q *Queue[T]) Get(maxEvents int) (queue.Batch[T], error) {
	debounced := false
	for {
		q.mu.Lock()
		if q.count > 0 {
			if !debounced && q.debounce > 0 && !q.closing && !q.forced.Load() {
				q.mu.Unlock()
				timer := time.NewTimer(q.debounce)
				select {
				case <-timer.C:
				case <-q.closeCh:
					timer.Stop()
				case <-q.pool.closed:
					timer.Stop()
					return nil, io.EOF
				}
				debounced = true
				continue
			}
			n := q.count
			if maxEvents > 0 {
				n = min(n, maxEvents)
			}
			b := q.buildBatchLocked(n)
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

// buildBatchLocked removes the first n events from this pipeline's FIFO and
// returns them as a recycled batch, appended to the pending-ack list so
// producer ACK callbacks still fire in publish order. It must be called with
// q.mu held and 0 < n <= q.count.
func (q *Queue[T]) buildBatchLocked(n int) *batch[T] {
	b := q.pool.getBatch()
	b.queue = q
	d := q.pool.dir.Load()
	cur := q.head
	for i := 0; i < n; i++ {
		b.indices = append(b.indices, cur)
		cur = d.slot(cur).next
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
	return b
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
	q.closing = true
	// On force-close, snapshot and clear the producer set so we can fan out an
	// unconditional ackWait close outside the lock — force-close suppresses the
	// per-event ACK callbacks that would otherwise close those channels. A
	// graceful close leaves the set intact: events keep draining and each
	// producer's ackWait closes naturally as its events are acked (and a later
	// force-close, e.g. on timeout, can still fan out to them).
	var ackWaitProducers []*producer[T]
	var releaseIndices []int
	if force {
		if len(q.producers) > 0 {
			ackWaitProducers = make([]*producer[T], 0, len(q.producers))
			for p := range q.producers {
				ackWaitProducers = append(ackWaitProducers, p)
			}
			q.producers = make(map[*producer[T]]struct{})
		}
		if q.count > 0 {
			// Walk the FIFO and gather the slots so we can release them back to
			// the pool below, outside the lock, to keep the critical section short.
			cur := q.head
			for cur != -1 {
				releaseIndices = append(releaseIndices, cur)
				cur = q.pool.slot(cur).next
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

	// Wake any blocked Get, and any producer parked waiting for a free slot on
	// this queue so it can observe closeCh and return.
	q.closeOnce.Do(func() {
		close(q.closeCh)
		q.pool.free.wakeAll()
		// Wake producers parked on this queue's per-queue cap so they observe
		// closeCh and return.
		q.wakeLimitWaiters()
	})
	q.signal()

	// Force-close: unblock every still-open producer's ACKWaitChan, since the
	// suppressed ACK callbacks will never advance their ack accounting.
	for _, p := range ackWaitProducers {
		p.forceCloseAckWait()
	}

	if len(releaseIndices) > 0 {
		var zero T
		for _, i := range releaseIndices {
			s := q.pool.slot(i)
			s.event = zero
			s.producer = nil
			s.next = -1
		}
		q.pool.observer.RemoveEvents(len(releaseIndices), 0)
		q.pool.releaseSlots(releaseIndices)
		// These FIFO events left circulation; return their per-queue budget.
		q.releaseLive(len(releaseIndices))
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

// BufferConfig reports this queue's effective maximum live events: the smaller
// of its per-queue cap (if set) and the shared pool budget. With no per-queue
// cap it is just the pool budget.
func (q *Queue[T]) BufferConfig() queue.BufferConfig {
	maxEvents := q.pool.Target()
	if lim := int(q.limit.Load()); lim > 0 {
		maxEvents = min(maxEvents, lim)
	}
	return queue.BufferConfig{MaxEvents: maxEvents}
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

// drainReadyLocked peels every already-Done() batch off the head of the pending
// FIFO and returns them as a list in publish order, the prefix whose producer
// ACK callbacks are now ready to fire. Anything from a not-yet-done batch onward
// stays in the list. It also re-checks whether the queue is now fully drained.
// Returned batches are no longer reachable from the queue, so the caller may ACK
// and recycle them once it releases q.mu. Must be called with q.mu held.
func (q *Queue[T]) drainReadyLocked() *batch[T] {
	var head, tail *batch[T]
	for q.pendingHead != nil && q.pendingHead.done {
		ready := q.pendingHead
		q.pendingHead = ready.next
		ready.next = nil
		if tail == nil {
			head = ready
		} else {
			tail.next = ready
		}
		tail = ready
	}
	if q.pendingHead == nil {
		q.pendingTail = nil
	}
	q.maybeMarkDone()
	return head
}
