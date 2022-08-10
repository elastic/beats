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

package buffer

import "sync"

// ringBuffer is a buffer with a fixed number of items that can be tracked.
//
// We assume that the size of the buffer is greater than one.
// the buffer should be thread-safe.
type ringBuffer struct {
	mu      sync.Mutex
	entries []interface{}
	i       int
	full    bool
}

// newBuffer returns a reference to a new ringBuffer with set size.
func newBuffer(size int) *ringBuffer {
	return &ringBuffer{
		entries: make([]interface{}, size),
	}
}

// add will add the passed entry to the buffer.
func (r *ringBuffer) add(entry interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries[r.i] = entry
	r.i = (r.i + 1) % len(r.entries)
	if r.i == 0 {
		r.full = true
	}
}

// getAll returns all entries in the buffer in order
func (r *ringBuffer) getAll() []interface{} {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.i == 0 && !r.full {
		return []interface{}{}
	}
	if !r.full {
		return r.entries[:r.i]
	}
	if r.full && r.i == 0 {
		return r.entries
	}
	return append(r.entries[r.i:], r.entries[:r.i]...)
}
