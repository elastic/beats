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
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
)

// freeList is the pool's counting semaphore: it holds the indices of every
// currently-free slot and bounds the number of live events at the pool's
// capacity. It replaces the buffered `chan int` the pool used originally for
// two reasons:
//
//   - A channel's buffer is fixed at creation, so it cannot back a resizable
//     pool. The freeList's per-shard stacks grow and shrink freely.
//   - Under multiple concurrent producers the single channel serializes every
//     acquire and release on one lock. The freeList shards that traffic across
//     K independent locks, which benchmarks show matches the channel at one
//     producer and is ~2x faster at 2-8 producers in steady state.
//
// Routing: a slot index's "home" shard is i & mask, fixed for the life of the
// pool and invariant under grow/shrink. release always returns an index to its
// home shard regardless of which goroutine releases it, which keeps shards
// balanced with no rebalancing. acquire takes a home *hint* (a per-producer
// starting point) but scans every shard, so correctness never depends on the
// hint — only contention does.
//
// Blocking: when every shard is empty (the pool is full and producers must
// wait for backpressure to clear) acquirers park on a single shared cond. That
// shared point is touched only on the slow path, so it is not a steady-state
// contention source. Waking uses the re-check-after-register pattern (register
// as a waiter, then re-scan the shards before parking) so a release that
// happens between the failed scan and the park cannot be lost.
type freeList struct {
	shards []freeShard
	mask   int // len(shards)-1; len(shards) is a power of two

	// waiters counts parked acquirers. It is read without the lock on the
	// release fast path (a racy "is anyone waiting?" check) and is also
	// mutated and read under gmu where it is authoritative.
	waiters atomic.Int64

	gmu   sync.Mutex
	gcond *sync.Cond
}

// freeShard is one independently-locked stack of free slot indices.
type freeShard struct {
	mu  sync.Mutex
	idx []int
}

// newFreeList builds an empty free list. Indices are added by the pool (via
// push/pushNoSignal) once their backing storage exists.
func newFreeList() *freeList {
	k := freeListShardCount()
	f := &freeList{
		shards: make([]freeShard, k),
		mask:   k - 1,
	}
	f.gcond = sync.NewCond(&f.gmu)
	return f
}

// freeListShardCount picks a fixed shard count from CPU parallelism, rounded up
// to a power of two and floored at 4. It is intentionally independent of the
// number of connected receivers: receivers come and go, but the shard count
// (and therefore every index's home shard) must never change, or the routing
// invariant would break.
func freeListShardCount() int {
	n := runtime.GOMAXPROCS(0)
	k := 1
	for k < n {
		k <<= 1
	}
	if k < 4 {
		k = 4
	}
	return k
}

// tryGrab pops a free index, scanning shards from the home hint. It never
// blocks. The second return is false when every shard is empty.
func (f *freeList) tryGrab(home int) (int, bool) {
	for off := 0; off <= f.mask; off++ {
		sh := &f.shards[(home+off)&f.mask]
		sh.mu.Lock()
		if n := len(sh.idx); n > 0 {
			x := sh.idx[n-1]
			sh.idx = sh.idx[:n-1]
			sh.mu.Unlock()
			return x, true
		}
		sh.mu.Unlock()
	}
	return 0, false
}

// push returns index i to its home shard and wakes one parked acquirer if any
// might be waiting.
func (f *freeList) push(i int) {
	f.pushNoSignal(i)
	f.signal()
}

// pushNoSignal returns index i to its home shard without waking acquirers. Used
// by grow, which pushes a batch of new indices and then broadcasts once.
func (f *freeList) pushNoSignal(i int) {
	sh := &f.shards[i&f.mask]
	sh.mu.Lock()
	sh.idx = append(sh.idx, i)
	sh.mu.Unlock()
}

// pushBatch returns a batch of indices to their home shards, locking each shard
// at most once instead of once per index. For a large batch (e.g. acking a full
// queue) this turns O(len) contended lock operations into O(shards). The slice
// is reordered in place — callers must not rely on its order afterward, which
// holds for every release path (the batch is recycled, the close list is
// local). Holding each shard lock only for the contiguous bulk append keeps the
// critical sections short, so it does not starve concurrent acquirers.
func (f *freeList) pushBatch(indices []int) {
	if len(indices) <= 1 {
		if len(indices) == 1 {
			f.pushNoSignal(indices[0])
		}
		return
	}
	// Sorting by the shard key clusters every shard's indices into a contiguous
	// run; we then append each run under a single lock.
	slices.SortFunc(indices, func(a, b int) int {
		return (a & f.mask) - (b & f.mask)
	})
	for start := 0; start < len(indices); {
		shard := indices[start] & f.mask
		end := start + 1
		for end < len(indices) && indices[end]&f.mask == shard {
			end++
		}
		sh := &f.shards[shard]
		sh.mu.Lock()
		sh.idx = append(sh.idx, indices[start:end]...)
		sh.mu.Unlock()
		start = end
	}
}

// removeIndex removes a specific free index from its home shard, returning
// false if it is not currently free (i.e. it is live at a producer/consumer).
// Used by the pool's top-down shrink to retire the highest slot once it is
// free. The linear scan is bounded by the shard's length; shrink is rare and
// off the hot path.
func (f *freeList) removeIndex(x int) bool {
	sh := &f.shards[x&f.mask]
	sh.mu.Lock()
	defer sh.mu.Unlock()
	for k := range sh.idx {
		if sh.idx[k] == x {
			last := len(sh.idx) - 1
			sh.idx[k] = sh.idx[last]
			sh.idx = sh.idx[:last]
			return true
		}
	}
	return false
}

// signal wakes a single parked acquirer, but only if one might be waiting. The
// fast unlocked read of waiters keeps the steady-state release path off the
// shared cond lock entirely; the shard append in push happens-before this read,
// and a parked acquirer increments waiters before re-scanning the shards, so if
// an acquirer would miss the just-pushed index this read is guaranteed to
// observe it as a waiter (see the freeList doc comment).
func (f *freeList) signal() {
	if f.waiters.Load() == 0 {
		return
	}
	f.gmu.Lock()
	if f.waiters.Load() > 0 {
		f.gcond.Signal()
	}
	f.gmu.Unlock()
}

// wakeAll wakes every parked acquirer. Used on grow (new slots may satisfy
// several blocked producers) and on close/shutdown (so parked producers can
// observe the closed state and return).
func (f *freeList) wakeAll() {
	f.gmu.Lock()
	f.gcond.Broadcast()
	f.gmu.Unlock()
}

// maybeWakeAll wakes every parked acquirer, but only if one might be waiting.
// Used by the bulk release path to coalesce many slot returns into one wake-up.
// The unlocked waiters read is safe for the same reason signal's is.
func (f *freeList) maybeWakeAll() {
	if f.waiters.Load() == 0 {
		return
	}
	f.wakeAll()
}

// available reports the number of free indices by summing the shards. It locks
// each shard in turn, so it is meant for tests and observability, not the hot
// path; keeping the count off the hot path avoids a single contended atomic on
// every acquire and release.
func (f *freeList) available() int {
	n := 0
	for s := range f.shards {
		f.shards[s].mu.Lock()
		n += len(f.shards[s].idx)
		f.shards[s].mu.Unlock()
	}
	return n
}
