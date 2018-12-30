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
	"github.com/elastic/go-txfile"
	"github.com/elastic/go-txfile/internal/cleanup"
)

// Writer is used to push new events onto the queue.
// The writer uses a write buffer, which is flushed once the buffer is full
// or if Flush is called.
// Only complete events are flushed. If an event is bigger then the configured write buffer,
// the write buffer will grow with the event size.
type Writer struct {
	active bool

	accessor *access
	flushCB  func(uint)

	state writeState
}

type writeState struct {
	buf *buffer

	activeEventCount uint // count number of finished events since last flush
	totalEventCount  uint
	totalAllocPages  uint

	eventID    uint64
	eventBytes int
}

const defaultMinPages = 5

func newWriter(
	accessor *access,
	pagePool *pagePool,
	writeBuffer uint,
	end position,
	flushCB func(uint),
) (*Writer, reason) {
	const op = "pq/create-writer"

	pageSize := accessor.PageSize()
	if pageSize <= 0 {
		return nil, &Error{op: op, kind: InvalidPageSize}
	}

	pages := int(writeBuffer) / pageSize
	if pages <= defaultMinPages {
		pages = defaultMinPages
	}

	var tail *page
	if end.page != 0 {
		traceln("writer load endpage: ", end)

		page := end.page
		off := end.off

		var err reason
		tail, err = accessor.readPageByID(pagePool, page)
		if err != nil {
			return nil, (&Error{op: op}).causedBy(err)
		}

		tail.Meta.EndOff = uint32(off)
	}

	w := &Writer{
		active:   true,
		accessor: accessor,
		state: writeState{
			buf:     newBuffer(pagePool, tail, pages, pageSize, szEventPageHeader),
			eventID: end.id,
		},
		flushCB: flushCB,
	}

	// init buffer with 'first' event to be written
	w.state.buf.ReserveHdr(szEventHeader)
	return w, nil
}

func (w *Writer) close() error {
	const op = "pq/writer-close"

	if !w.active {
		return nil
	}

	err := w.doFlush()
	if err != nil {
		return w.errWrap(op, err)
	}

	w.active = false
	w.state.buf = nil
	return err
}

func (w *Writer) Write(p []byte) (int, error) {
	const op = "pq/write"

	if err := w.canWrite(); err != NoError {
		return 0, w.errOf(op, err)
	}

	if w.state.buf.Avail() <= len(p) {
		if err := w.doFlush(); err != nil {
			return 0, w.errWrap(op, err)
		}
	}

	w.state.buf.Append(p)
	w.state.eventBytes += len(p)

	return len(p), nil
}

// Next is used to indicate the end of the current event.
// If write is used with a streaming encoder, the buffers
// of the actual writer must be flushed before calling Next on this writer.
// Upon next, the queue writer will add the event framing header and footer.
func (w *Writer) Next() error {
	const op = "pq/writer-next"

	if err := w.canWrite(); err != NoError {
		return w.errOf(op, err)
	}

	// finalize current event in buffer and prepare next event
	hdr := castEventHeader(w.state.buf.ActiveEventHdr())
	hdr.sz.Set(uint32(w.state.eventBytes))
	w.state.buf.CommitEvent(w.state.eventID)
	w.state.buf.ReserveHdr(szEventHeader)
	w.state.eventBytes = 0
	w.state.eventID++
	w.state.activeEventCount++

	// check if we need to flush
	if w.state.buf.Avail() <= szEventHeader {
		if err := w.doFlush(); err != nil {
			return w.errWrap(op, err)
		}
	}

	return nil
}

// Flush flushes the write buffer. Returns an error if the queue is closed,
// some error occurred or no more space is available in the file.
func (w *Writer) Flush() error {
	const op = "pq/writer-flush"

	if err := w.canWrite(); err != NoError {
		return w.errOf(op, err)
	}

	if err := w.doFlush(); err != nil {
		return w.errWrap(op, err)
	}

	return nil
}

func (w *Writer) doFlush() error {
	start, end := w.state.buf.Pages()
	if start == nil || start == end {
		return nil
	}

	traceln("writer flush", w.state.activeEventCount)

	// unallocated points to first page in list that must be allocated.  All
	// pages between unallocated and end require a new page to be allocated.
	var unallocated *page
	for current := start; current != end; current = current.Next {
		if !current.Assigned() {
			unallocated = current
			break
		}
	}

	tx, txErr := w.accessor.BeginWrite()
	if txErr != nil {
		return w.errWrap("", txErr)
	}
	defer tx.Close()

	rootPage, queueHdr, err := w.accessor.LoadRootPage(tx)
	if err != nil {
		return w.errWrap("", err)
	}

	traceQueueHeader(queueHdr)

	ok := false
	allocN, txErr := allocatePages(tx, unallocated, end)
	if txErr != nil {
		return w.errWrap("", txErr)
	}
	linkPages(start, end)
	defer cleanup.IfNot(&ok, func() { unassignPages(unallocated, end) })

	traceln("write queue pages")
	last, txErr := flushPages(tx, start, end)
	if txErr != nil {
		return w.errWrap("", txErr)
	}

	// update queue root
	w.updateRootHdr(queueHdr, start, last, allocN)
	rootPage.MarkDirty()

	txErr = tx.Commit()
	if txErr != nil {
		return w.errWrap("", txErr)
	}

	// mark write as success -> no error-cleanup required
	ok = true

	// remove dirty flag from all published pages
	for current := start; current != end; current = current.Next {
		current.UnmarkDirty()
	}

	w.state.buf.Reset(last)

	activeEventCount := w.state.activeEventCount
	w.state.totalEventCount += activeEventCount
	w.state.totalAllocPages += uint(allocN)

	traceln("Write buffer flushed. Total events: %v, total pages allocated: %v",
		w.state.totalEventCount,
		w.state.totalAllocPages)

	w.state.activeEventCount = 0
	if w.flushCB != nil {
		w.flushCB(activeEventCount)
	}

	return nil
}

func (w *Writer) updateRootHdr(hdr *queuePage, start, last *page, allocated int) {
	if hdr.head.offset.Get() == 0 {
		w.accessor.WritePosition(&hdr.head, position{
			page: start.Meta.ID,
			off:  int(start.Meta.FirstOff),
			id:   start.Meta.FirstID,
		})
	}

	hdr.inuse.Set(hdr.inuse.Get() + uint64(allocated))

	endOff := int(last.Meta.EndOff)
	if last == w.state.buf.eventHdrPage {
		endOff = w.state.buf.eventHdrOffset
	}

	w.accessor.WritePosition(&hdr.tail, position{
		page: last.Meta.ID,
		off:  endOff,
		id:   w.state.eventID,
	})

	traceln("writer: update queue header")
	traceQueueHeader(hdr)
}

func (w *Writer) canWrite() ErrKind {
	if !w.active {
		return WriterClosed
	}
	return NoError
}

func (w *Writer) err(op string) *Error { return w.errPage(op, 0) }
func (w *Writer) errPage(op string, page txfile.PageID) *Error {
	return &Error{op: op, ctx: w.errPageCtx(page)}
}

func (w *Writer) errOf(op string, kind ErrKind) *Error {
	return w.err(op).of(kind)
}

func (w *Writer) errWrap(op string, cause error) *Error { return w.errWrapPage(op, cause, 0) }
func (w *Writer) errWrapPage(op string, cause error, page txfile.PageID) *Error {
	return w.errPage(op, page).causedBy(cause)
}

func (w *Writer) errCtx() errorCtx { return w.errPageCtx(0) }
func (w *Writer) errPageCtx(id txfile.PageID) errorCtx {
	return w.accessor.errPageCtx(id)
}

func allocatePages(tx *txfile.Tx, start, end *page) (int, error) {
	if start == nil {
		return 0, nil
	}

	allocN := 0
	for current := start; current != end; current = current.Next {
		allocN++
	}

	tracef("allocate %v queue pages\n", allocN)

	txPages, err := tx.AllocN(allocN)
	if err != nil {
		return 0, err
	}

	// assign new page IDs
	for current, i := start, 0; current != end; current, i = current.Next, i+1 {
		current.Meta.ID = txPages[i].ID()
	}

	return allocN, nil
}

// unassignPages removes page assignments from all pages between start and end,
// so to mark these pages as 'not allocated'.
func unassignPages(start, end *page) {
	for current := start; current != end; current = current.Next {
		current.Meta.ID = 0
	}
}

// Update page headers to point to next page in the list.
func linkPages(start, end *page) {
	for current := start; current.Next != end; current = current.Next {
		tracef("link page %v -> %v\n", current.Meta.ID, current.Next.Meta.ID)
		current.SetNext(current.Next.Meta.ID)
	}
}

// flushPages flushes all pages in the list of pages and returns the last page
// being flushed.
func flushPages(tx *txfile.Tx, start, end *page) (*page, error) {
	last := start
	for current := start; current != end; current = current.Next {
		last = current

		err := flushPage(tx, current)
		if err != nil {
			return nil, err
		}
	}

	return last, nil
}

func flushPage(tx *txfile.Tx, page *page) error {
	page.UpdateHeader()
	tracePageHeader(page.Meta.ID, castEventPageHeader(page.Data))

	diskPage, err := tx.Page(page.Meta.ID)
	if err != nil {
		return err
	}

	err = diskPage.SetBytes(page.Data)
	if err != nil {
		return err
	}

	return diskPage.Flush()
}
