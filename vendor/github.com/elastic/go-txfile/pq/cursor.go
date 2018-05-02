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

func (c *txCursor) init() error {
	if c.page != nil {
		return nil
	}
	page, err := c.tx.Page(c.cursor.page)
	if err != nil {
		return err
	}

	c.page = page
	return nil
}

// Read reads more bytes from the current event into b.  If the end of the
// current event has reached, no bytes will be read.
func (c *txCursor) Read(b []byte) (int, error) {
	if err := c.init(); err != nil {
		return 0, err
	}

	if c.Nil() {
		return 0, nil
	}

	to, err := c.readInto(b)
	return len(b) - len(to), err
}

// Skip skips the next n bytes.
func (c *txCursor) Skip(n int) error {
	for n > 0 {
		if c.PageBytes() == 0 {
			ok, err := c.AdvancePage()
			if err != nil {
				return err
			}
			if !ok {
				return errSeekPageFailed
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

func (c *txCursor) readInto(to []byte) ([]byte, error) {
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

func (c *txCursor) ReadEventHeader() (hdr *eventHeader, err error) {
	err = c.WithBytes(func(b []byte) {
		hdr = castEventHeader(b)
		c.off += szEventHeader
	})
	return
}

func (c *txCursor) PageHeader() (hdr *eventPage, err error) {
	err = c.WithHdr(func(h *eventPage) {
		hdr = h
	})
	return
}

func (c *txCursor) AdvancePage() (ok bool, err error) {
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
	return
}

func (c *txCursor) WithPage(fn func([]byte)) error {
	if err := c.init(); err != nil {
		return err
	}

	buf, err := c.page.Bytes()
	if err != nil {
		return err
	}

	fn(buf)
	return nil
}

func (c *txCursor) WithHdr(fn func(*eventPage)) error {
	return c.WithPage(func(b []byte) {
		fn(castEventPageHeader(b))
	})
}

func (c *txCursor) WithBytes(fn func([]byte)) error {
	return c.WithPage(func(b []byte) {
		fn(b[c.off:])
	})
}

// PageBytes reports the amount of bytes still available in current page
func (c *cursor) PageBytes() int {
	return c.pageSize - c.off
}

func (c *cursor) Reset() {
	*c = cursor{}
}
