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

// acker is used to asynchronously ack and remove events from the queue.
type acker struct {
	accessor *access
	active   bool

	hdrOffset uintptr
	observer  Observer

	totalEventCount uint
	totalFreedPages uint

	ackCB func(events, pages uint)
}

// ackState records the changes required to finish the ACK step.
type ackState struct {
	free []txfile.PageID // Collect page ids to be freed.
	head position        // New queue head, pointing to first event in first available page
	read position        // New on-disk read pointer, pointing to first not-yet ACKed event.
}

func newAcker(accessor *access, off uintptr, o Observer, cb func(uint, uint)) *acker {
	return &acker{
		hdrOffset: off,
		observer:  o,
		active:    true,
		accessor:  accessor,
		ackCB:     cb,
	}
}

func (a *acker) close() {
	a.active = false
}

// handle processes an ACK by freeing pages and
// updating the head and read positions in the queue root.
// So to not interfere with concurrent readers potentially updating pointers
// or adding new contents to a page, the last event page in the queue will never
// be freed. Still the read pointer might point past the last page.
func (a *acker) handle(n uint) error {
	const op = "pq/ack"

	if n == 0 {
		return nil
	}

	if !a.active {
		return a.err(op).of(QueueClosed)
	}

	traceln("acker: pq ack events:", n)

	start := time.Now()
	events, pages, err := a.cleanup(n)
	if o := a.observer; o != nil {
		failed := err != nil
		o.OnQueueACK(a.hdrOffset, ACKStats{
			Duration: time.Since(start),
			Failed:   failed,
			Events:   events,
			Pages:    pages,
		})
	}
	return err
}

func (a *acker) cleanup(n uint) (events uint, pages uint, err error) {
	const op = "pq/ack-cleanup"

	state, err := a.initACK(n)
	events, pages = n, uint(len(state.free))
	if err != nil {
		return events, pages, a.errWrap(op, err)
	}

	// start write transaction to free pages and update the next read offset in
	// the queue root
	tx, txErr := a.accessor.BeginCleanup()
	if txErr != nil {
		return events, pages, a.errWrap(op, txErr).report("failed to init cleanup tx")
	}
	defer tx.Close()

	traceln("acker: free data pages:", len(state.free))
	for _, id := range state.free {
		page, err := tx.Page(id)
		if err != nil {
			return events, pages, a.errWrapPage(op, err, id).report("can not access page to be freed")
		}

		traceln("free page", id)
		if err := page.Free(); err != nil {
			return events, pages, a.errWrapPage(op, err, id).report("releasing page failed")
		}
	}

	// update queue header
	hdrPage, hdr, err := a.accessor.LoadRootPage(tx)
	if err != nil {
		return events, pages, err
	}
	a.accessor.WritePosition(&hdr.head, state.head)
	a.accessor.WritePosition(&hdr.read, state.read)
	hdr.inuse.Set(hdr.inuse.Get() - uint64(len(state.free)))
	hdrPage.MarkDirty()

	traceQueueHeader(hdr)

	if err := tx.Commit(); err != nil {
		return events, pages, a.errWrap(op, err).report("failed to commit changes")
	}

	a.totalEventCount += n
	a.totalFreedPages += uint(len(state.free))
	tracef("Acked events. total events acked: %v, total pages freed: %v \n", a.totalEventCount, a.totalFreedPages)

	if a.ackCB != nil {
		a.ackCB(n, uint(len(state.free)))
	}

	return events, pages, nil
}

// initACK uses a read-transaction to collect pages to be removed from list and
// find offset of next read required to start reading the next un-acked event.
func (a *acker) initACK(n uint) (ackState, reason) {
	const op = "pq/ack-precompute"

	tx, txErr := a.accessor.BeginRead()
	if txErr != nil {
		return ackState{}, a.errWrap(op, txErr)
	}
	defer tx.Close()

	hdr, err := a.accessor.RootHdr(tx)
	if err != nil {
		return ackState{}, err
	}

	headPos, startPos, endPos := a.queueRange(hdr)
	startID := startPos.id
	endID := startID + uint64(n)
	if startPos.page == 0 {
		return ackState{}, a.err(op).of(ACKEmptyQueue)
	}
	if !idLessEq(endID, endPos.id) {
		return ackState{}, a.err(op).of(ACKTooMany)
	}

	c := makeTxCursor(tx, a.accessor, &cursor{
		page:     headPos.page,
		off:      headPos.off,
		pageSize: a.accessor.PageSize(),
	})

	// Advance through pages and collect ids of all pages to be freed.
	// Free all pages, but the very last data page, so to not interfere with
	// concurrent writes.
	ids, cleanAll, err := a.collectFreePages(&c, endID)
	if err != nil {
		return ackState{}, a.errWrap(op, err)
	}

	// find offset of next event to start reading from
	var head, read position
	if !cleanAll {
		head, read, err = a.findNewStartPositions(&c, endID)
		if err != nil {
			return ackState{}, a.errWrap(op, err)
		}
	} else {
		head = endPos
		read = endPos
	}

	return ackState{
		free: ids,
		head: head,
		read: read,
	}, nil
}

// queueRange finds the start and end positions of not yet acked events in the
// queue.
func (a *acker) queueRange(hdr *queuePage) (head, start, end position) {
	start = a.accessor.ParsePosition(&hdr.read)
	head = a.accessor.ParsePosition(&hdr.head)
	if start.page == 0 {
		start = head
	}

	end = a.accessor.ParsePosition(&hdr.tail)
	return
}

// collectFreePages collects all pages to be freed. A page can be freed if all
// events within the page have been acked. We want to free all pages, but the
// very last data page, so to not interfere with concurrent writes.
// All pages up to endID will be collected.
func (a *acker) collectFreePages(c *txCursor, endID uint64) ([]txfile.PageID, bool, reason) {
	const op = "pq/collect-acked-pages"
	var (
		ids      []txfile.PageID
		lastID   uint64
		cleanAll = false
	)

	for {
		hdr, err := c.PageHeader()
		if err != nil {
			return nil, false, a.errWrap(op, err)
		}

		next := hdr.next.Get()

		// stop searching if current page is the last page. The last page must
		// be active for the writer to add more events and link new pages.
		isWritePage := next == 0

		// stop searching if endID is in the current write page
		dataOnlyPage := hdr.off.Get() == 0 // no event starts within this page
		if !dataOnlyPage {
			lastID = hdr.last.Get()

			// inc 'lastID', so to hold on current page if endID would point to next
			// the page. This helps the reader, potentially pointing to the current
			// page, if next page has not been committed when reading events.
			lastID++

			// remove page if endID points past current data page
			keepPage := isWritePage || idLessEq(endID, lastID)
			if keepPage {
				break
			}
		}

		if isWritePage {
			cleanAll = true
			invariant.Checkf(lastID+1 == endID, "last event ID (%v) and ack event id (%v) missmatch", lastID, endID)
			break
		}

		// found intermediate page with ACKed events/contents
		// -> add page id to freelist and advance to next page
		ids = append(ids, c.cursor.page)
		ok, err := c.AdvancePage()
		if err != nil {
			return nil, false, a.errWrap(op, err)
		}
		invariant.Check(ok, "page list linkage broken")
	}

	return ids, cleanAll, nil
}

// findNewStartPositions skips acked events, so to find the new head and read pointers to be set
// in the updated queue header.
func (a *acker) findNewStartPositions(c *txCursor, id uint64) (head, read position, err reason) {
	const op = "pq/ack-compute-new-start"

	var hdr *eventPage

	hdr, err = c.PageHeader()
	if err != nil {
		return head, read, a.errWrap(op, err)
	}

	head = position{
		page: c.cursor.page,
		off:  int(hdr.off.Get()),
		id:   hdr.first.Get(),
	}

	if id == head.id {
		read = head
		return head, read, nil
	}

	// skip contents in current page until we did reach start of next event.
	c.cursor.off = head.off
	for currentID := head.id; currentID != id; currentID++ {
		var evtHdr *eventHeader
		evtHdr, err = c.ReadEventHeader()
		if err != nil {
			return head, read, a.errWrap(op, err)
		}

		err = c.Skip(int(evtHdr.sz.Get()))
		if err != nil {
			return head, read, a.errWrap(op, err)
		}
	}

	read = position{
		page: c.cursor.page,
		off:  c.cursor.off,
		id:   id,
	}
	return head, read, nil
}

// Active returns the total number of active, not yet ACKed events.
func (a *acker) Active() (uint, reason) {
	const op = "pq/count-active-events"

	tx, txErr := a.accessor.BeginRead()
	if txErr != nil {
		return 0, a.errWrap(op, txErr)
	}
	defer tx.Close()

	hdr, err := a.accessor.RootHdr(tx)
	if err != nil {
		return 0, a.errWrap(op, err)
	}

	// Empty queue?
	if hdr.tail.offset.Get() == 0 {
		return 0, nil
	}

	var start, end uint64

	end = hdr.tail.id.Get()
	if hdr.read.offset.Get() != 0 {
		start = hdr.read.id.Get()
	} else {
		start = hdr.head.id.Get()
	}

	return uint(end - start), nil
}

func (a *acker) err(op string) *Error { return a.errPage(op, 0) }
func (a *acker) errPage(op string, page txfile.PageID) *Error {
	return &Error{op: op, ctx: a.errPageCtx(page)}
}

func (a *acker) errWrap(op string, cause error) *Error { return a.errWrapPage(op, cause, 0) }
func (a *acker) errWrapPage(op string, cause error, page txfile.PageID) *Error {
	return a.errPage(op, page).causedBy(cause)
}

func (a *acker) errCtx() errorCtx { return a.accessor.errCtx() }
func (a *acker) errPageCtx(id txfile.PageID) errorCtx {
	return a.accessor.errPageCtx(id)
}
