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

package readfile

import "sync"

// tempBufferPools recycles the per-line scratch buffers used by LineReader,
// sharded by buffer size. A new LineReader is created for every harvested file,
// so workloads with many small files would otherwise allocate (and immediately
// discard) one buffer per file. The buffer is pure internal scratch: its
// contents are always copied into the in/out streambufs before the next read,
// so it never escapes the reader and is safe to reuse.
//
// LineReaders are created with a configurable buffer_size. A single global pool
// keyed only by "[]byte" would mix sizes: a reader configured with a large
// buffer_size would repeatedly discard smaller pooled buffers (losing reuse),
// while a small-buffer reader would retain oversized buffers. Sharding by size
// keeps each pool homogeneous, so every Get returns a correctly sized buffer and
// inputs with different buffer_size values never contend over the same buffers.
// The number of distinct sizes in a process is tiny (usually just the default),
// so the map stays small. Pools store *[]byte to avoid an allocation on Put.
var tempBufferPools sync.Map // map[int]*sync.Pool

// poolForSize returns the pool dedicated to buffers of the given size, creating
// it on first use.
func poolForSize(size int) *sync.Pool {
	if p, ok := tempBufferPools.Load(size); ok {
		return p.(*sync.Pool)
	}
	p, _ := tempBufferPools.LoadOrStore(size, &sync.Pool{})
	return p.(*sync.Pool)
}

// getTempBuffer returns a scratch buffer of length size, reusing a pooled one
// of the same size when available. Because each pool is homogeneous, a pooled
// buffer always has the exact capacity requested.
func getTempBuffer(size int) []byte {
	if size <= 0 {
		return nil
	}
	if v := poolForSize(size).Get(); v != nil {
		return (*v.(*[]byte))[:size]
	}
	return make([]byte, size)
}

// putTempBuffer returns a scratch buffer to the pool matching its capacity.
func putTempBuffer(b []byte) {
	if cap(b) == 0 {
		return
	}
	// Key by capacity, not length, so the buffer rejoins the pool for its true
	// size regardless of how it was resliced.
	poolForSize(cap(b)).Put(&b)
}
