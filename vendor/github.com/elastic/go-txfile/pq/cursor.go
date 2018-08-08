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

import "github.com/elastic/go-txfile"

// cursor holds state for iterating events in the queue.
type cursor struct {
	page     txfile.PageID
	off      int
	pageSize int
}

// txCursor is used to advance a cursor within a transaction.
type txCursor struct {
	*cursor
	accessor *access
	tx       *txfile.Tx
	page     *txfile.Page
}

// Nil checks if the cursor is pointing to a page. Returns true, if cursor is
// not pointing to any page in the queue.
func (c *cursor) Nil() bool {
	return c.page == 0
}

func makeTxCursor(tx *txfile.Tx, accessor *access, cursor *cursor) txCursor {
	return txCursor{
		tx:       tx,
		accessor: accessor,
		page:     nil,
		cursor:   cursor,
	}
}

func (c *txCursor) init() reason {
	const op = "pq/cursor-init"

	if c.page != nil {
		return nil
	}
	page, err := c.tx.Page(c.cursor.page)
	if err != nil {
		return c.errWrap(op, err)
	}

	c.page = page
	return nil
}

// Read reads more bytes from the current event into b.  If the end of the
// current event has reached, no bytes will be read.
func (c *txCursor) Read(b []byte) (int, reason) {
	const op = "pq/read-bytes"

	if err := c.init(); err != nil {
		return 0, c.errWrap(op, err)
	}

	if c.Nil() {
		return 0, nil
	}

	to, err := c.readInto(b)
	n := len(b) - len(to)

	if err != nil {
		err = c.errWrap(op, err)
	}
	return n, err
}

// Skip skips the next n bytes.
func (c *txCursor) Skip(n int) reason {
	const op = "pq/skip"

	for n > 0 {
		if c.PageBytes() == 0 {
			ok, err := c.AdvancePage()
			if err != nil {
				return c.errWrap(op, err).of(SeekFail)
			}
			if !ok {
				return c.err(op).report("No page to seek to")
			}
		}

		max := n
		if L := c.PageBytes(); L < max {
			max = L
		}
		c.cursor.off += max
		n -= max
	}

	return nil
}

func (c *txCursor) readInto(to []byte) ([]byte, reason) {
	for len(to) > 0 {
		// try to advance cursor to next page if last read did end at end of page
		if c.PageBytes() == 0 {
			ok, err := c.AdvancePage()
			if !ok || err != nil {
				return to, err
			}
		}

		var n int
		err := c.WithBytes(func(b []byte) { n = copy(to, b) })
		to = to[n:]
		c.cursor.off += n
		if err != nil {
			return to, err
		}
	}

	return to, nil
}

func (c *txCursor) ReadEventHeader() (hdr *eventHeader, err reason) {
	const op = "pq/cursor-read-event-header"

	err = c.WithBytes(func(b []byte) {
		hdr = castEventHeader(b)
		c.off += szEventHeader
	})

	if err != nil {
		err = c.errWrap(op, err)
	}
	return hdr, err
}

func (c *txCursor) PageHeader() (hdr *eventPage, err reason) {
	err = c.WithHdr(func(h *eventPage) { hdr = h })
	return
}

func (c *txCursor) AdvancePage() (ok bool, err reason) {
	const op = "pq/cursor-next-page"

	err = c.WithHdr(func(hdr *eventPage) {
		nextID := txfile.PageID(hdr.next.Get())
		tracef("advance page from %v -> %v\n", c.cursor.page, nextID)
		ok = nextID != 0

		if ok {
			c.cursor.page = nextID
			c.cursor.off = szEventPageHeader
			c.page = nil
		}
	})

	if err != nil {
		err = c.errWrap(op, err)
	}
	return ok, err
}

func (c *txCursor) WithPage(fn func([]byte)) reason {
	if err := c.init(); err != nil {
		return err
	}

	buf, err := c.page.Bytes()
	if err != nil {
		return c.errWrap("", err).of(ReadFail)
	}

	fn(buf)
	return nil
}

func (c *txCursor) WithHdr(fn func(*eventPage)) reason {
	const op = "pq/cursor-read-event-page-header"

	err := c.WithPage(func(b []byte) {
		fn(castEventPageHeader(b))
	})
	if err != nil {
		return c.errWrap(op, err)
	}
	return nil
}

func (c *txCursor) WithBytes(fn func([]byte)) reason {
	const op = "pq/cursor-access-page"

	err := c.WithPage(func(b []byte) { fn(b[c.off:]) })
	if err != nil {
		return c.errWrap(op, err)
	}
	return nil
}

// PageBytes reports the amount of bytes still available in current page
func (c *cursor) PageBytes() int {
	return c.pageSize - c.off
}

func (c *cursor) Reset() {
	*c = cursor{}
}

func (c *txCursor) err(op string) *Error {
	return &Error{op: op, ctx: c.errCtx(c.cursor.page)}
}

func (c *txCursor) errWrap(op string, cause error) *Error {
	return c.err(op).causedBy(cause)
}

func (c *txCursor) errCtx(page txfile.PageID) errorCtx {
	return c.accessor.errPageCtx(page)
}
