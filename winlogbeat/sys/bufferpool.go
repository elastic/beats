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

package sys

import (
	"sync"
)

// bufferPool contains a pool of PooledByteBuffer objects.
var bufferPool = sync.Pool{
	New: func() interface{} { return &PooledByteBuffer{ByteBuffer: NewByteBuffer(1024)} },
}

// PooledByteBuffer is an expandable buffer backed by a byte slice.
type PooledByteBuffer struct {
	*ByteBuffer
}

// NewPooledByteBuffer return a PooledByteBuffer from the pool. The returned value must
// be released with Free().
func NewPooledByteBuffer() *PooledByteBuffer {
	b := bufferPool.Get().(*PooledByteBuffer)
	b.Reset()
	return b
}

// Free returns the PooledByteBuffer to the pool.
func (b *PooledByteBuffer) Free() {
	if b == nil {
		return
	}
	bufferPool.Put(b)
}
