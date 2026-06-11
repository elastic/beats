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
// Slots are allocated from a free list, which also functions as the counting
// semaphore that bounds the total number of events live across all connected
// pipelines.
//
// The pool is resizable at any time, including while producers and consumers
// are running: its two capacity-bounding structures are both growable. Storage
// is a directory of non-moving chunks (see storage.go) and the free list is a
// sharded semaphore (see freelist.go). The pool grows immediately and shrinks
// lazily, retiring the highest slots as they fall out of circulation so live
// events are never dropped. Its capacity is not set directly — it is derived
// from the connected queues' per-queue caps (see syncTargetToQueues), so the
// pool always tracks the largest queue and the two cannot drift.
//
// Capacity invariant: the in-circulation slot indices are always exactly
// [0, capacity) with no holes. Growth appends indices at the top; shrink removes
// only the current top index and only once it is free. capacity therefore
// doubles as the storage high-water mark, which lets shrink reclaim whole
// trailing chunks without ever invalidating a pointer to a live slot.
type Pool[T any] struct {
	observer queue.Observer

	// dir holds the slot storage as non-moving chunks; see storage.go. It is
	// swapped (never mutated in place) under growMu when the pool grows or
	// reclaims a trailing chunk, and loaded atomically on every slot access.
	dir  atomic.Pointer[directory[T]]
	free *freeList

	// growMu serializes structural resize (capacity changes, directory swaps).
	// capacity and target are stored under it but read lock-free as atomics.
	//
	//   capacity: current physical slot count == in-circulation high-water.
	//   target:   desired budget. capacity == target in steady state; capacity
	//             > target while a lazy shrink is converging.
	growMu   sync.Mutex
	capacity atomic.Int64
	target   atomic.Int64

	// homeCounter hands each producer a stable home shard for the free list,
	// assigned once at producer creation. It keeps ticking as producers
	// (receivers) come and go; the shard count never changes.
	homeCounter atomic.Uint64

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
		observer: observer,
		free:     newFreeList(),
		closed:   make(chan struct{}),
		queues:   make(map[*Queue[T]]struct{}),
	}
	p.batchPool.New = func() any { return &batch[T]{} }
	p.dir.Store(newDirectory[T](settings.Events))
	for i := 0; i < settings.Events; i++ {
		p.free.pushNoSignal(i)
	}
	p.capacity.Store(int64(settings.Events))
	p.target.Store(int64(settings.Events))
	p.observer.MaxEvents(settings.Events)
	return p
}

// setTarget records n as the desired capacity and is safe to call while
// producers and consumers are running. Growth is immediate: when n is above the
// current capacity the new slots are allocated and become acquirable before
// setTarget returns, and any producers blocked on a full pool are woken. Shrink
// is lazy: when n is below the current capacity the target is lowered now and
// the pool retires its highest slots as they are released, converging to n
// without ever discarding a slot that still holds a live, unacked event. The
// effective floor at any instant is the highest in-use slot index, so a
// long-lived event at the top of the range holds the shrink until it drains.
//
// setTarget is unexported on purpose: the pool's capacity is not set directly
// but derived from the per-queue caps via syncTargetToQueues, so callers size
// the pool only by setting Queue.SetTarget and the two can never disagree.
func (p *Pool[T]) setTarget(n int) {
	if n < 1 {
		n = 1
	}
	p.growMu.Lock()
	defer p.growMu.Unlock()
	prev := p.target.Load()
	p.target.Store(int64(n))
	if int64(n) != prev {
		p.observer.MaxEvents(n)
	}
	switch {
	case int64(n) > p.capacity.Load():
		p.growLocked(n)
	case int64(n) < p.capacity.Load():
		p.shrinkLocked()
	}
}

// growLocked raises capacity to n by allocating any missing chunks, publishing
// the new directory, and pushing the new indices [old, n) to the free list. It
// must be called with growMu held. Indices are published only after the
// directory that backs them is stored, so a goroutine that later acquires one
// always observes a directory containing its chunk.
func (p *Pool[T]) growLocked(n int) {
	old := int(p.capacity.Load())
	if n <= old {
		return
	}
	if need := numChunks(n); need > len(p.dir.Load().chunks) {
		cur := p.dir.Load().chunks
		nc := make([]*chunk[T], need)
		copy(nc, cur)
		for i := len(cur); i < need; i++ {
			nc[i] = &chunk[T]{}
		}
		p.dir.Store(&directory[T]{chunks: nc})
	}
	for i := old; i < n; i++ {
		p.free.pushNoSignal(i)
	}
	p.capacity.Store(int64(n))
	// Wake every blocked producer: the added slots may satisfy more than one.
	p.free.wakeAll()
}

// shrinkLocked retires free slots from the top of the index range toward the
// current target. It removes only the highest index, and only while that index
// is free; if the top index is currently live it stops and will be retried by
// the next release. This keeps the in-circulation set contiguous ([0, capacity))
// so trailing chunks can be reclaimed exactly. It must be called with growMu
// held.
func (p *Pool[T]) shrinkLocked() {
	for {
		cap := int(p.capacity.Load())
		if cap <= int(p.target.Load()) {
			return
		}
		top := cap - 1
		if !p.free.removeIndex(top) {
			// The top slot is still in use; it will be retired when released.
			return
		}
		p.capacity.Store(int64(top))
		// Reclaim any trailing chunk the shrink just emptied.
		if want := numChunks(top); want < len(p.dir.Load().chunks) {
			cur := p.dir.Load().chunks
			nc := make([]*chunk[T], want)
			copy(nc, cur[:want])
			p.dir.Store(&directory[T]{chunks: nc})
		}
	}
}

// acquire takes a free slot index for a producer, blocking until one is
// available, the queue closes (closeCh), or the pool shuts down. It returns
// (0, false) when interrupted by close/shutdown. home is the producer's shard
// hint.
func (p *Pool[T]) acquire(home int, closeCh <-chan struct{}) (int, bool) {
	// Mirror the original channel select: a publish racing with shutdown/close
	// fails rather than grabbing a slot that would be immediately abandoned.
	if p.isClosed() || chClosed(closeCh) {
		return 0, false
	}
	if i, ok := p.free.tryGrab(home); ok {
		return i, true
	}
	f := p.free
	for {
		f.gmu.Lock()
		f.waiters.Add(1)
		// Re-check after registering as a waiter: a push (or grow) between the
		// failed tryGrab above and the Wait below would otherwise be lost.
		if i, ok := f.tryGrab(home); ok {
			f.waiters.Add(-1)
			f.gmu.Unlock()
			return i, true
		}
		if p.isClosed() || chClosed(closeCh) {
			f.waiters.Add(-1)
			f.gmu.Unlock()
			return 0, false
		}
		f.gcond.Wait()
		f.waiters.Add(-1)
		f.gmu.Unlock()
		if p.isClosed() || chClosed(closeCh) {
			return 0, false
		}
		if i, ok := f.tryGrab(home); ok {
			return i, true
		}
	}
}

// releaseSlot returns slot index i to the free list. If a lazy shrink is in
// progress (capacity > target) it then attempts to retire the top slot(s) now
// that this release may have freed the current high-water index. The shrink
// check is a single atomic comparison, so the steady-state release path is just
// a push.
func (p *Pool[T]) releaseSlot(i int) {
	p.free.push(i)
	if p.capacity.Load() > p.target.Load() {
		p.growMu.Lock()
		p.shrinkLocked()
		p.growMu.Unlock()
	}
}

// releaseSlots returns a batch of slot indices to the free list, coalescing the
// wake-up into a single broadcast rather than signaling once per slot. This is
// the path batch completion and force-close use: acking a large batch while
// producers are blocked would otherwise take the shared cond lock once per
// slot. The single guarded broadcast is correct for the same reason signal is:
// a parked producer registers as a waiter before re-scanning the shards, so if
// it would miss every pushed slot the waiters count is guaranteed visible here.
func (p *Pool[T]) releaseSlots(indices []int) {
	if len(indices) == 0 {
		return
	}
	p.free.pushBatch(indices)
	if p.capacity.Load() > p.target.Load() {
		p.growMu.Lock()
		p.shrinkLocked()
		p.growMu.Unlock()
	}
	p.free.maybeWakeAll()
}

// isClosed reports whether the pool has been shut down.
func (p *Pool[T]) isClosed() bool {
	select {
	case <-p.closed:
		return true
	default:
		return false
	}
}

// chClosed is a non-blocking check for a closed signal channel.
func chClosed(ch <-chan struct{}) bool {
	if ch == nil {
		return false
	}
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

// getBatch returns a recycled or freshly allocated *batch[T]. Callers
// are responsible for populating it via fillBatch before exposing it.
func (p *Pool[T]) getBatch() *batch[T] {
	b, _ := p.batchPool.Get().(*batch[T])
	return b
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
		// Wake any producers parked on a full pool so they observe the closed
		// state and return instead of blocking forever.
		p.free.wakeAll()
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

// Capacity returns the pool's current physical slot count. While a lazy shrink
// is converging this can exceed Target; in steady state they are equal.
func (p *Pool[T]) Capacity() int { return int(p.capacity.Load()) }

// Target returns the pool's desired capacity (the configured budget). Useful
// for tests and observability.
func (p *Pool[T]) Target() int { return int(p.target.Load()) }

// Available returns the number of currently free slots. Useful for tests and
// observability.
func (p *Pool[T]) Available() int { return p.free.available() }

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
	// A departing queue may have held the largest per-queue cap; resize the pool
	// to the new maximum so it tracks the connected queues.
	p.syncTargetToQueues()
}

// syncTargetToQueues sizes the pool to the largest per-queue cap among the
// connected queues, so the shared pool always tracks the queues rather than
// being set independently (which could drift). It is the single place pool
// capacity is derived: Queue.SetTarget and disconnect call it. Queues with no
// per-queue cap (limit 0) do not contribute, so a pool of only uncapped queues
// keeps whatever capacity it was created with.
func (p *Pool[T]) syncTargetToQueues() {
	if p.isClosed() {
		return
	}
	// Hold p.mu across both the scan and the apply so concurrent syncs serialize:
	// otherwise two callers could each compute a max and then apply them out of
	// order, letting a stale max overwrite a fresh one.
	p.mu.Lock()
	defer p.mu.Unlock()
	max := 0
	for q := range p.queues {
		if l := int(q.limit.Load()); l > max {
			max = l
		}
	}
	if max > 0 {
		p.setTarget(max)
	}
}
