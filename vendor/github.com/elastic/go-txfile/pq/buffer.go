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

import "github.com/elastic/go-txfile/internal/invariant"

// buffer holds allocated and yet unallocated in-memory pages, for appending
// events to.
type buffer struct {
	// in-memory pages
	head, tail *page

	pool *pagePool

	// settings (values don't change after init)
	pageSize    int
	hdrSize     int
	payloadSize int // effective page contents

	// page write state
	avail   int    // available space before forcing flush
	payload []byte // byte slice of available payload/bytes in the current page
	page    *page  // current page

	// Event write state. Stores reference to start of current events, so we can
	// put in the event header once the current event is finished.
	eventHdrPage   *page
	eventHdrOffset int
	eventHdrSize   int

	// stats
	countPages uint
}

func newBuffer(pool *pagePool, page *page, pages, pageSize, hdrSz int) *buffer {
	payloadSz := pageSize - hdrSz
	avail := payloadSz * pages

	tracef("init writer buffer with pages=%v, pageSize=%v, hdrSize=%v, avail=%v\n",
		pages, pageSize, hdrSz, avail)

	b := &buffer{
		head:           nil,
		tail:           nil,
		pool:           pool,
		pageSize:       pageSize,
		hdrSize:        hdrSz,
		payloadSize:    payloadSz,
		avail:          avail,
		payload:        nil,
		page:           nil,
		eventHdrPage:   nil,
		eventHdrOffset: -1,
		eventHdrSize:   -1,
	}

	if page != nil {
		// init with end of on-disk list from former writes
		b.head = page
		b.tail = page

		contentsLength := int(page.Meta.EndOff) - b.hdrSize
		b.avail -= contentsLength
		b.payload = page.Data[page.Meta.EndOff:]
		b.page = page
		b.countPages++
	}

	return b
}

// Avail returns amount of bytes available. Returns a value <0, if contents in
// buffer exceeds the high-water-marks.
func (b *buffer) Avail() int {
	return b.avail
}

// Append adds more bytes to the current event. Use `CommitEvent` to finalize the
// writing of the current event.
// If required append adds new unallocated pages to the write buffer.
func (b *buffer) Append(data []byte) {
	for len(data) > 0 {
		if len(b.payload) == 0 {
			b.advancePage()
		}

		n := copy(b.payload, data)
		b.payload = b.payload[n:]
		data = data[n:]
		b.avail -= n

		tracef("writer: append %v bytes to (page: %v, off: %v, avail: %v)\n", n, b.page.Meta.ID, b.page.Meta.EndOff, b.avail)

		b.page.Meta.EndOff += uint32(n)
	}
}

func (b *buffer) advancePage() {
	// link new page into list
	page := b.newPage()
	if b.tail == nil {
		b.head = page
		b.tail = page
	} else {
		b.tail.Next = page
		b.tail = page
	}

	b.page = page
	b.payload = page.Payload()
	page.Meta.EndOff = uint32(szEventPageHeader)
}

func (b *buffer) newPage() *page {
	b.countPages++
	return b.pool.NewPage()
}

func (b *buffer) releasePage(p *page) {
	b.countPages--
	b.pool.Release(p)
}

// ReserveHdr reserves space for the next event header in the write buffer.
// The start position in the buffer is tracked by the buffer, until the event is
// finished via CommitEvent.
func (b *buffer) ReserveHdr(n int) []byte {
	if n > b.payloadSize {
		return nil
	}

	invariant.Check(b.eventHdrPage == nil, "can not reserve a new event header if recent event is not finished yet")

	// reserve n bytes in payload
	if len(b.payload) < n {
		b.advancePage()
	}

	payloadWritten := b.payloadSize - len(b.payload)
	b.eventHdrPage = b.page
	b.eventHdrPage.Meta.EndOff += uint32(n)
	b.eventHdrOffset = b.hdrSize + payloadWritten
	b.eventHdrSize = n
	b.payload = b.payload[n:]
	b.avail -= n

	return b.ActiveEventHdr()
}

// ActiveEventHdr returns the current event header bytes content for writing/reading.
func (b *buffer) ActiveEventHdr() []byte {
	if b.eventHdrPage == nil {
		return nil
	}

	off := b.eventHdrOffset
	return b.eventHdrPage.Data[off : off+b.eventHdrSize]
}

// CommitEvent marks the current event being finished. Finalize pages
// and prepare for next event.
func (b *buffer) CommitEvent(id uint64) {
	invariant.Check(b.eventHdrPage != nil, "no active event")

	page := b.eventHdrPage
	meta := &page.Meta
	if meta.FirstOff == 0 {
		meta.FirstOff = uint32(b.eventHdrOffset)
		meta.FirstID = id
	}
	meta.LastID = id
	page.MarkDirty()

	// mark all event pages as dirty
	for current := b.eventHdrPage; current != nil; current = current.Next {
		current.MarkDirty()
	}
	// mark head as dirty if yet unlinked
	if b.head != b.eventHdrPage && b.head.Next == b.eventHdrPage {
		b.head.MarkDirty()
	}

	b.eventHdrPage = nil
	b.eventHdrOffset = -1
	b.eventHdrSize = -1
}

// Pages returns start and end page to be serialized.
// The `end` page must not be serialized
func (b *buffer) Pages() (start, end *page, n uint) {
	traceln("get buffer active page range")

	if b.head == nil || !b.head.Dirty() {
		traceln("buffer empty")
		return nil, nil, 0
	}

	if b.eventHdrPage == nil {
		traceln("no active page")

		if b.tail.Dirty() {
			traceln("tail is dirty")
			return b.head, nil, b.countPages
		}

		traceln("tail is not dirty")
		for current := b.head; current != nil; current = current.Next {
			if !current.Dirty() {
				return b.head, current, n
			}
			n++
		}

		invariant.Unreachable("tail if list dirty and not dirty?")
	}

	end = b.eventHdrPage
	n = b.countPages
	if end.Dirty() {
		traceln("active page is dirty")
		end = end.Next
	} else {
		traceln("active page is clean")
		n--
	}
	return b.head, end, n
}

// Reset removes all but the last page non-dirty page from the buffer.
// The last written page is still required for writing/linking new events/pages.
func (b *buffer) Reset(last *page) {
	if b.head == nil {
		return
	}

	// Find last page not to be removed. A non-dirty page must not be removed
	// if the next page is dirty, so to update the on-disk link.
	// If no page is dirty, keep last page for linking.
	pages := 0
	end := b.head
	for current := b.head; current.Next != nil && current != b.eventHdrPage; current = current.Next {
		if current.Next.Dirty() || current == last {
			end = current
			break
		}
		end = current.Next
		pages++
	}

	tracef("reset pages (%v)\n", pages)

	invariant.Check(end != nil, "must not empty page list on reset")

	// release pages
	spaceFreed := 0
	for page := b.head; page != end; {
		freed := int(page.Meta.EndOff) - szEventPageHeader
		tracef("writer: release page %v (%v)\n", page.Meta.ID, freed)

		next := page.Next
		spaceFreed += freed
		b.releasePage(page)
		page = next
	}
	b.head = end

	// update memory usage counters
	b.avail += spaceFreed
}
