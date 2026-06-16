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

// The pool's slot storage is a directory of fixed-size chunks instead of a
// single contiguous []slot. This is what makes the pool resizable while
// producers and consumers are live: a chunk, once allocated, is never moved or
// copied, so a goroutine holding `&pool.slot(i)` keeps a stable pointer even
// while the pool grows or shrinks concurrently. Growing allocates new chunks
// and publishes a new directory (a small slice of chunk pointers) behind an
// atomic.Pointer; shrinking drops only trailing chunks whose slots are all out
// of circulation. A plain []slot could not do this — append/realloc moves the
// backing array, splitting a live producer's write and a reader's read across
// two arrays.
//
// Index decomposition uses a power-of-two chunk size so slot lookup is a
// shift + mask with no division.

const (
	// slabChunkShift sets the chunk size (1<<shift slots per chunk). 4096 keeps
	// the per-chunk allocation modest (one chunk covers the default 3200-event
	// pool) while keeping the directory small even for very large pools.
	slabChunkShift = 12
	slabChunkSize  = 1 << slabChunkShift
	slabChunkMask  = slabChunkSize - 1
)

// chunk is a fixed block of slots. It is heap-allocated once and never moved,
// so pointers into it (&c.slots[k]) stay valid for the chunk's lifetime.
type chunk[T any] struct {
	slots [slabChunkSize]slot[T]
}

// directory maps a flat slot index to its chunk. The slice of chunk pointers
// is copied (cheaply) on grow/shrink and swapped atomically; the chunks it
// points at are shared and never copied.
type directory[T any] struct {
	chunks []*chunk[T]
}

// numChunks returns how many chunks are needed to back `capacity` slots, never
// fewer than one.
func numChunks(capacity int) int {
	if capacity <= 0 {
		return 1
	}
	return (capacity + slabChunkSize - 1) >> slabChunkShift
}

// newDirectory allocates a directory whose chunks cover at least `capacity`
// slots. All chunks are allocated eagerly.
func newDirectory[T any](capacity int) *directory[T] {
	n := numChunks(capacity)
	d := &directory[T]{chunks: make([]*chunk[T], n)}
	for i := range d.chunks {
		d.chunks[i] = &chunk[T]{}
	}
	return d
}

// slot returns a stable pointer to slot i. The directory is loaded atomically
// so this is safe to call concurrently with grow/shrink: any index a goroutine
// legitimately holds (one it acquired from the free list, or reached by
// following FIFO `next` links) is always covered by the directory it observes,
// because indices are published to the free list only after the directory that
// contains them has been stored.
func (p *Pool[T]) slot(i int) *slot[T] {
	d := p.dir.Load()
	return &d.chunks[i>>slabChunkShift].slots[i&slabChunkMask]
}
