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

package reader

import (
	"errors"
	"io"
	"time"
)

// Reader is the interface that wraps the basic Next method for
// getting a new message.
// Next returns the message being read or and error. EOF is returned
// if reader will not return any new message on subsequent calls.
type Reader interface {
	io.Closer
	Next() (Message, error)
}

// ErrReadDeadline is returned by a read whose deadline (set via
// DeadlineSetter.SetReadDeadline) elapsed before a line was available.
var ErrReadDeadline = errors.New("read deadline exceeded")

// ErrWouldBlock is returned by non-blocking sources when no new data is
// currently available to read, without indicating a terminal EOF. Stateful
// readers that aggregate multiple records (e.g. multiline) retain their
// in-progress state and propagate ErrWouldBlock rather than flushing early.
var ErrWouldBlock = errors.New("no data available, would block")

// BufferedFlusher is implemented by stateful readers (such as multiline) that
// may hold an incomplete message across non-blocking reads (ErrWouldBlock).
// Wrapping readers delegate to the reader they wrap so a caller holding the
// outer pipeline can inspect and flush buffered state.
type BufferedFlusher interface {
	// PendingDeadline reports whether the reader currently holds buffered
	// state for an incomplete message, and if that state expires after a
	// timeout, the deadline at which it must be flushed. A zero deadline with
	// pending == true means the buffered state has no timeout (e.g. count mode)
	// and waits indefinitely until enough data arrives or the source closes.
	PendingDeadline() (pending bool, deadline time.Time)

	// FlushPending flushes and returns a buffered incomplete message, if any.
	// If timedOut is true, the flush was triggered by PendingDeadline elapsing;
	// otherwise it is a forced flush (e.g. harvester close or shutdown drain).
	// When ok is false, no message was buffered.
	FlushPending(timedOut bool) (msg Message, ok bool, err error)
}

// PendingDeadline reports whether r (or a reader it wraps) holds buffered
// state for an incomplete message and when that state expires.
func PendingDeadline(r Reader) (bool, time.Time) {
	if bf, ok := r.(BufferedFlusher); ok {
		return bf.PendingDeadline()
	}
	return false, time.Time{}
}

// FlushPending flushes and returns a buffered incomplete message from r (or a
// reader it wraps), if any.
func FlushPending(r Reader, timedOut bool) (Message, bool, error) {
	if bf, ok := r.(BufferedFlusher); ok {
		return bf.FlushPending(timedOut)
	}
	return Message{}, false, nil
}

// DeadlineSetter is implemented by readers whose blocking wait for more data can
// be bounded by a deadline, allowing a timeout to be enforced synchronously
// without a background goroutine. Wrapping readers delegate to the reader they
// wrap; the reader that actually blocks (e.g. the file reader) honors it.
type DeadlineSetter interface {
	// SetReadDeadline bounds how long the next blocking read may wait for data. A
	// zero time clears the deadline. It returns true if the deadline is honored by
	// this reader or one it wraps, so a caller can detect support and otherwise
	// fall back. When the deadline elapses mid-wait, the read returns ErrReadDeadline.
	SetReadDeadline(t time.Time) bool
}

// SetReadDeadline sets a read deadline on r if r (or a reader it wraps) supports
// it, returning whether it was honored.
func SetReadDeadline(r Reader, t time.Time) bool {
	d, ok := r.(DeadlineSetter)
	return ok && d.SetReadDeadline(t)
}
