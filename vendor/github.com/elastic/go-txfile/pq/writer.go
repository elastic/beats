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
) (*Writer, error) {
	pageSize := accessor.PageSize()
	if pageSize <= 0 {
		return nil, errInvalidPagesize
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

		var err error
		tail, err = readPageByID(accessor, pagePool, page)
		if err != nil {
			return nil, err
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
	if !w.active {
		return nil
	}

	err := w.doFlush()
	if err != nil {
		return err
	}

	w.active = false
	w.state.buf = nil
	return err
}

func (w *Writer) Write(p []byte) (int, error) {
	if !w.active {
		return 0, errClosed
	}

	if w.state.buf.Avail() <= len(p) {
		if err := w.doFlush(); err != nil {
			return 0, err
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
	if !w.active {
		return errClosed
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
			return err
		}
	}

	return nil
}

// Flush flushes the write buffer. Returns an error if the queue is closed,
// some error occurred or no more space is available in the file.
func (w *Writer) Flush() error {
	if !w.active {
		return errClosed
	}
	return w.doFlush()
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

	tx := w.accessor.BeginWrite()
	defer tx.Close()

	rootPage, queueHdr, err := w.accessor.LoadRootPage(tx)
	if err != nil {
		return err
	}

	traceQueueHeader(queueHdr)

	ok := false
	allocN, err := allocatePages(tx, unallocated, end)
	if err != nil {
		return err
	}
	linkPages(start, end)
	defer cleanup.IfNot(&ok, func() { unassignPages(unallocated, end) })

	traceln("write queue pages")
	last, err := flushPages(tx, start, end)
	if err != nil {
		return err
	}

	// update queue root
	w.updateRootHdr(queueHdr, start, last, allocN)
	rootPage.MarkDirty()

	err = tx.Commit()
	if err != nil {
		return err
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
