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

// ByteBuffer is an expandable buffer backed by a byte slice.
type ByteBuffer struct {
	buf    []byte
	offset int
}

// NewByteBuffer creates a new ByteBuffer with an initial capacity of
// initialSize.
func NewByteBuffer(initialSize int) *ByteBuffer {
	return &ByteBuffer{buf: make([]byte, initialSize)}
}

// Write appends the contents of p to the buffer, growing the buffer as needed.
// The return value is the length of p; err is always nil.
func (b *ByteBuffer) Write(p []byte) (int, error) {
	if len(b.buf) < b.offset+len(p) {
		// Create a buffer larger than needed so we don't spend lots of time
		// allocating and copying.
		spaceNeeded := len(b.buf) - b.offset + len(p)
		largerBuf := make([]byte, 2*len(b.buf)+spaceNeeded)
		copy(largerBuf, b.buf[:b.offset])
		b.buf = largerBuf
	}
	n := copy(b.buf[b.offset:], p)
	b.offset += n
	return n, nil
}

// Reset resets the buffer to be empty. It retains the same underlying storage.
func (b *ByteBuffer) Reset() {
	b.offset = 0
	b.buf = b.buf[:cap(b.buf)]
}

// Bytes returns a slice of length b.Len() holding the bytes that have been
// written to the buffer.
func (b *ByteBuffer) Bytes() []byte {
	return b.buf[:b.offset]
}

// Len returns the number of bytes that have been written to the buffer.
func (b *ByteBuffer) Len() int {
	return b.offset
}

// PtrAt returns a pointer to the given offset of the buffer.
func (b *ByteBuffer) PtrAt(offset int) *byte {
	if offset > b.offset-1 {
		return nil
	}
	return &b.buf[offset]
}

// Reserve reserves n bytes by increasing the buffer's length. It may allocate
// a new underlying buffer discarding any existing contents.
func (b *ByteBuffer) Reserve(n int) {
	b.offset = n

	if n > cap(b.buf) {
		// Allocate new larger buffer with len=n.
		b.buf = make([]byte, n)
	} else {
		b.buf = b.buf[:n]
	}
}
