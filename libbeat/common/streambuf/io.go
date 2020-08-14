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

package streambuf

import (
	"io"
	"unicode/utf8"
)

func (b *Buffer) ioErr() error {
	err := b.Err()
	if err == ErrUnexpectedEOB || err == ErrNoMoreBytes {
		return io.EOF
	}
	return err
}

func (b *Buffer) ioBufferEndError() error {
	err := b.bufferEndError()
	if err == ErrUnexpectedEOB || err == ErrNoMoreBytes {
		return io.EOF
	}
	return err
}

// ReadByte reads and returns next byte from the buffer.
// If no byte is available returns either ErrNoMoreBytes (if buffer allows
// adding more bytes) or io.EOF
func (b *Buffer) ReadByte() (byte, error) {
	if b.Failed() {
		return 0, b.ioErr()
	}
	if !b.Avail(1) {
		return 0, b.ioBufferEndError()
	}
	c := b.data[b.mark]
	b.Advance(1)
	return c, nil
}

// Unreads the last byte returned by most recent read operation.
func (b *Buffer) UnreadByte() error {
	err := b.ioErr()
	if err != nil && err != io.EOF {
		return err
	}
	if b.mark == 0 {
		return ErrOutOfRange
	}

	if b.mark == b.offset {
		b.offset--
	}
	b.mark--
	b.available++
	return nil
}

// WriteByte appends the byte c to the buffer if buffer is not fixed.
func (b *Buffer) WriteByte(c byte) error {
	p := [1]byte{c}
	_, err := b.Write(p[:])
	return err
}

// Read reads up to len(p) bytes into p if buffer is not in a failed state.
// Returns ErrNoMoreBytes or io.EOF (fixed buffer) if no bytes are available.
func (b *Buffer) Read(p []byte) (int, error) {
	if b.Failed() {
		return 0, b.ioErr()
	}
	if b.Len() == 0 {
		return 0, b.ioBufferEndError()
	}

	tmp := b.Bytes()
	n := copy(p, tmp)
	b.Advance(n)
	return n, nil
}

// Write writes p to the buffer if buffer is not fixed. Returns the number of
// bytes written or ErrOperationNotAllowed if buffer is fixed.
func (b *Buffer) Write(p []byte) (int, error) {
	err := b.doAppend(p, false, -1)
	if err != nil {
		return 0, b.ioErr()
	}
	return len(p), nil
}

// ReadFrom reads data from r until error or io.EOF and appends it to the buffer.
// The amount of bytes read is returned plus any error except io.EOF.
func (b *Buffer) ReadFrom(r io.Reader) (int64, error) {
	err := b.err
	if err != nil && err != ErrNoMoreBytes {
		return 0, b.ioErr()
	}
	if b.fixed {
		return 0, ErrOperationNotAllowed
	}

	var buf [4096]byte
	var total int64
	for {
		n, err := r.Read(buf[:])
		if err != nil {
			if err == io.EOF {
				break
			}
			return total, err
		}
		_, err = b.Write(buf[:n])
		if err != nil {
			return total, err
		}
		total += int64(n)
	}

	return total, nil
}

// ReadRune reads and returns the next UTF-8-encoded Unicode code point from the
// buffer. If no bytes are available, the error returned is ErrNoMoreBytes (if
// buffer supports adding more bytes) or io.EOF. If the bytes are an erroneous
// UTF-8 encoding, it consumes one byte and returns U+FFFD, 1.
func (b *Buffer) ReadRune() (rune, int, error) {
	if b.err != nil {
		return 0, 0, b.ioErr()
	}
	if b.available == 0 {
		return 0, 0, b.ioBufferEndError()
	}

	if c := b.data[b.mark]; c < utf8.RuneSelf {
		b.Advance(1)
		return rune(c), 1, nil
	}
	c, size := utf8.DecodeRune(b.data[b.mark:])
	b.Advance(size)
	return c, size, nil
}

// ReadAt reads bytes at off into p starting at the buffer its read marker.
// The read marker is not updated. If number of bytes returned is less len(p) or
// no bytes are available at off, io.EOF will be returned in err. If off is < 0,
// err is set to ErrOutOfRange.
func (b *Buffer) ReadAt(p []byte, off int64) (n int, err error) {
	if b.err != nil {
		return 0, b.ioErr()
	}

	if off < 0 {
		return 0, ErrOutOfRange
	}

	off += int64(b.mark)
	if off >= int64(len(b.data)) {
		return 0, ErrOutOfRange
	}

	end := off + int64(len(p))
	if end > int64(len(b.data)) {
		err = io.EOF
		end = int64(len(b.data))
	}
	copy(p, b.data[off:end])
	return int(end - off), err
}

// WriteAt writes the content of p at off starting at recent read marker
// (already consumed bytes). Returns number of bytes written n = len(p) and err
// is nil if off and off+len(p) are within bounds, else n=0 and err is set to
// ErrOutOfRange.
func (b *Buffer) WriteAt(p []byte, off int64) (n int, err error) {
	if b.err != nil {
		return 0, b.ioErr()
	}

	end := off + int64(b.mark) + int64(len(p))
	maxInt := int((^uint(0)) >> 1)
	if off < 0 || end > int64(maxInt) {
		return 0, ErrOutOfRange
	}

	// copy p into buffer
	n = copy(b.sliceAt(int(off), len(p)), p)
	return n, nil
}
