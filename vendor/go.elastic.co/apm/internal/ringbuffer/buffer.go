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

package ringbuffer

import (
	"bytes"
	"encoding/binary"
	"io"
	"io/ioutil"
)

// BlockHeaderSize is the size of the block header, in bytes.
const BlockHeaderSize = 5

// BlockTag is a block tag, which can be used for classification.
type BlockTag uint8

// BlockHeader holds a fixed-size block header.
type BlockHeader struct {
	// Tag is the block's tag.
	Tag BlockTag

	// Size is the size of the block data, in bytes.
	Size uint32
}

// Buffer is a ring buffer of byte blocks.
type Buffer struct {
	buf       []byte
	headerbuf [BlockHeaderSize]byte
	len       int
	write     int
	read      int

	// Evicted will be called when an old block is evicted to make place for a new one.
	Evicted func(BlockHeader)
}

// New returns a new Buffer with the given size in bytes.
func New(size int) *Buffer {
	return &Buffer{
		buf:     make([]byte, size),
		Evicted: func(BlockHeader) {},
	}
}

// Len returns the number of bytes currently in the buffer, including
// block-accounting bytes.
func (b *Buffer) Len() int {
	return b.len
}

// Cap returns the capacity of the buffer.
func (b *Buffer) Cap() int {
	return len(b.buf)
}

// WriteBlockTo writes the oldest block in b to w, returning the block header and the number of bytes written to w.
func (b *Buffer) WriteBlockTo(w io.Writer) (header BlockHeader, written int64, err error) {
	if b.len == 0 {
		return header, 0, io.EOF
	}
	if n := copy(b.headerbuf[:], b.buf[b.read:]); n < len(b.headerbuf) {
		b.read = copy(b.headerbuf[n:], b.buf[:])
	} else {
		b.read = (b.read + n) % b.Cap()
	}
	b.len -= len(b.headerbuf)
	header.Tag = BlockTag(b.headerbuf[0])
	header.Size = binary.LittleEndian.Uint32(b.headerbuf[1:])
	size := int(header.Size)

	if b.read+size > b.Cap() {
		tail := b.buf[b.read:]
		n, err := w.Write(tail)
		if err != nil {
			b.read = (b.read + size) % b.Cap()
			b.len -= size + len(b.headerbuf)
			return header, int64(n), err
		}
		size -= n
		written = int64(n)
		b.read = 0
		b.len -= n
	}
	n, err := w.Write(b.buf[b.read : b.read+size])
	if err != nil {
		return header, written + int64(n), err
	}
	written += int64(n)
	b.read = (b.read + size) % b.Cap()
	b.len -= size
	return header, written, nil
}

// WriteBlock writes p as a block to b, with tag t.
//
// If len(p)+BlockHeaderSize > b.Cap(), bytes.ErrTooLarge will be returned.
// If the buffer does not currently have room for the block, then the
// oldest blocks will be evicted until enough room is available.
func (b *Buffer) WriteBlock(p []byte, tag BlockTag) (int, error) {
	lenp := len(p)
	if lenp+BlockHeaderSize > b.Cap() {
		return 0, bytes.ErrTooLarge
	}
	for lenp+BlockHeaderSize > b.Cap()-b.Len() {
		header, _, err := b.WriteBlockTo(ioutil.Discard)
		if err != nil {
			return 0, err
		}
		b.Evicted(header)
	}
	b.headerbuf[0] = uint8(tag)
	binary.LittleEndian.PutUint32(b.headerbuf[1:], uint32(lenp))
	if n := copy(b.buf[b.write:], b.headerbuf[:]); n < len(b.headerbuf) {
		b.write = copy(b.buf, b.headerbuf[n:])
	} else {
		b.write = (b.write + n) % b.Cap()
	}
	if n := copy(b.buf[b.write:], p); n < lenp {
		b.write = copy(b.buf, p[n:])
	} else {
		b.write = (b.write + n) % b.Cap()
	}
	b.len += lenp + BlockHeaderSize
	return lenp, nil
}
