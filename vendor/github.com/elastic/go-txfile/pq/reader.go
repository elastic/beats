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

package pq

import (
	"time"

	"github.com/elastic/go-txfile"
	"github.com/elastic/go-txfile/internal/invariant"
)

// Reader is used to iterate events stored in the queue.
type Reader struct {
	accessor *access
	state    readState
	active   bool

	tx *txfile.Tx

	hdrOff   uintptr
	observer Observer
	txStart  time.Time
	stats    ReadStats
}

type readState struct {
	id            uint64
	endID         uint64 // id of next, yet unwritten event.
	totEventBytes int    // number of total bytes in current event
	eventBytes    int    // number of unread bytes in current event

	cursor cursor
}

func newReader(observer Observer, accessor *access) *Reader {
	return &Reader{
		active:   true,
		accessor: accessor,
		observer: observer,
		state: readState{
			eventBytes:    -1,
			totEventBytes: -1,
			cursor: cursor{
				pageSize: accessor.PageSize(),
			},
		},
	}
}

func (r *Reader) close() {
	r.active = false
}

// Available returns the number of unread events that can be read.
func (r *Reader) Available() (uint, error) {
	const op = "pq/reader-available"

	if err := r.canRead(); err != NoError {
		return 0, r.errOf(op, err)
	}

	tx := r.tx
	err := r.updateQueueState(tx)
	if err != nil {
		return 0, r.errWrap(op, err)
	}

	if r.state.cursor.Nil() {
		return 0, nil
	}

	return uint(r.state.endID - r.state.id), nil
}

// Begin starts a new read transaction, shared between multiple read calls.
// User must execute Done, to close the file transaction.
func (r *Reader) Begin() error {
	const op = "pq/reader-begin"

	var sig ErrKind = NoError
	switch {
	case r.isClosed():
		sig = ReaderClosed
	case r.isTxActive():
		sig = UnexpectedActiveTx
	}

	if sig != NoError {
		return r.errOf(op, sig)
	}

	tx, err := r.beginTx()
	if err != nil {
		return r.errWrap(op, err)
	}

	r.tx = tx
	r.txStart = time.Now()
	r.stats = ReadStats{} // zero out last stats on begin
	return nil
}

// Done closes the active read transaction.
func (r *Reader) Done() {
	if r.tx == nil {
		return
	}

	r.tx.Close()

	if r.state.eventBytes < 0 && r.state.totEventBytes > 0 {
		// did read complete event -> adapt stats
		r.adoptEventStats()
	}

	r.stats.Duration = time.Since(r.txStart)
	if o := r.observer; o != nil {
		o.OnQueueRead(r.hdrOff, r.stats)
	}

	r.tx = nil
}

// Read reads the contents of the current event into the buffer.
// Returns 0 without reading if end of the current event has been reached.
// Use `Next` to skip/continue reading the next event.
// If Begin is not been called before Read, a temporary read transaction is
// created.
func (r *Reader) Read(b []byte) (int, error) {
	const op = "pq/read-event"

	if err := r.canRead(); err != NoError {
		return -1, r.errOf(op, err)
	}

	if r.state.eventBytes <= 0 {
		return 0, nil
	}

	to, err := r.readInto(b)
	n := len(b) - len(to)
	if err != nil {
		return n, r.errWrap(op, err)
	}
	return len(b) - len(to), nil
}

func (r *Reader) readInto(to []byte) ([]byte, reason) {
	tx := r.tx
	n := r.state.eventBytes
	if L := len(to); L < n {
		n = L
	}

	cursor := makeTxCursor(tx, r.accessor, &r.state.cursor)
	for n > 0 {
		consumed, err := cursor.Read(to[:n])
		to = to[consumed:]
		n -= consumed
		r.state.eventBytes -= consumed

		if err != nil {
			return to, err
		}
	}

	// end of event -> advance to next event
	var err reason
	if r.state.eventBytes == 0 {
		r.state.eventBytes = -1
		r.state.id++

		// As page is already in memory, use current transaction to try to skip to
		// next page if no more new event fits into current page.
		if cursor.PageBytes() < szEventHeader {
			_, err = cursor.AdvancePage()
		}
	}

	return to, err
}

// Next advances to the next event to be read. The event size in bytes is
// returned.  A size of 0 is reported if no more event is available in the
// queue.
// If Begin is not been called before Next, a temporary read transaction is
// created.
func (r *Reader) Next() (int, error) {
	const op = "op/reader-next"

	if err := r.canRead(); err != NoError {
		return -1, r.errOf(op, err)
	}

	tx := r.tx
	cursor := makeTxCursor(tx, r.accessor, &r.state.cursor)

	r.adoptEventStats()

	// in event? Skip contents
	if r.state.eventBytes > 0 {
		err := cursor.Skip(r.state.eventBytes)
		if err != nil {
			return 0, r.errWrap(op, err)
		}

		r.state.eventBytes = -1
		r.state.id++
	}

	// end of buffered queue state. Update state and check if we did indeed reach
	// the end of the queue.
	if cursor.Nil() || !idLess(r.state.id, r.state.endID) {
		err := r.updateQueueState(tx)
		if err != nil {
			return 0, r.errWrap(op, err)
		}

		// end of queue
		if cursor.Nil() || !idLess(r.state.id, r.state.endID) {
			return 0, nil
		}
	}

	// Advance page and initialize cursor if event header does not fit into
	// current page.
	if cursor.PageBytes() < szEventHeader {
		// cursor was not advanced by last read. The acker will not have deleted
		// the current page -> try to advance now.
		ok, err := cursor.AdvancePage()
		if err != nil {
			return 0, err
		}
		invariant.Check(ok, "page list linkage broken")

		hdr, err := cursor.PageHeader()
		if err != nil {
			return 0, r.errWrap(op, err)
		}

		id := hdr.first.Get()
		off := int(hdr.off.Get())
		invariant.Check(r.state.id == id, "page start event id mismatch")
		invariant.CheckNot(off == 0, "page event offset missing")
		r.state.cursor.off = off
	}

	// Initialize next event read by determining event size.
	hdr, err := cursor.ReadEventHeader()
	if err != nil {
		return 0, r.errWrap(op, err)
	}
	L := int(hdr.sz.Get())
	r.state.eventBytes = L
	r.state.totEventBytes = L
	return L, nil
}

func (r *Reader) adoptEventStats() {
	if r.state.totEventBytes < 0 {
		// no active event
		return
	}

	// update stats:
	skipping := r.state.eventBytes > 0

	if skipping {
		r.stats.Skipped++
		r.stats.BytesSkipped += uint(r.state.eventBytes)
		r.stats.BytesTotal += uint(r.state.totEventBytes - r.state.eventBytes)
	} else {
		bytes := uint(r.state.totEventBytes)
		r.stats.BytesTotal += bytes
		if r.stats.Read == 0 {
			r.stats.BytesMin = bytes
			r.stats.BytesMax = bytes
		} else {
			if r.stats.BytesMin > bytes {
				r.stats.BytesMin = bytes
			}
			if r.stats.BytesMax < bytes {
				r.stats.BytesMax = bytes
			}
		}

		r.stats.Read++
	}
}

func (r *Reader) updateQueueState(tx *txfile.Tx) reason {
	const op = "pq/reader-update-queue-state"

	root, err := r.accessor.RootHdr(tx)
	if err != nil {
		return r.errWrap(op, err)
	}

	// Initialize cursor, if queue was empty on previous (without any pages).
	if r.state.cursor.Nil() {
		head := r.findReadStart(root)
		tail := r.accessor.ParsePosition(&root.tail)

		r.state.id = head.id
		r.state.cursor.page = head.page
		r.state.cursor.off = head.off
		r.state.endID = tail.id
	} else {
		r.state.endID = root.tail.id.Get()
	}
	return nil
}

func (r *Reader) findReadStart(root *queuePage) position {
	head := r.accessor.ParsePosition(&root.read)
	if head.page != 0 {
		return head
	}
	return r.accessor.ParsePosition(&root.head)
}

func (r *Reader) beginTx() (*txfile.Tx, reason) {
	tx, err := r.accessor.BeginRead()
	if err != nil {
		return nil, r.errWrap("", err).report("failed to start read transaction")
	}
	return tx, nil
}

func (r *Reader) canRead() ErrKind {
	if r.isClosed() {
		return ReaderClosed
	}
	if !r.isTxActive() {
		return InactiveTx
	}
	return NoError
}

func (r *Reader) isClosed() bool {
	return !r.active
}

func (r *Reader) isTxActive() bool {
	return r.tx != nil
}

func (r *Reader) err(op string) *Error {
	return &Error{op: op, ctx: r.errCtx()}
}

func (r *Reader) errOf(op string, kind ErrKind) *Error {
	return r.err(op).of(kind)
}

func (r *Reader) errWrap(op string, cause error) *Error {
	return r.err(op).causedBy(cause)
}

func (r *Reader) errCtx() errorCtx {
	return r.accessor.errCtx()
}
