package pq

import (
	"github.com/elastic/go-txfile"
	"github.com/elastic/go-txfile/internal/invariant"
)

// Reader is used to iterate events stored in the queue.
type Reader struct {
	accessor *access
	state    readState
	active   bool
}

type readState struct {
	id         uint64
	endID      uint64 // id of next, yet unwritten event.
	eventBytes int    // number of unread bytes in current event

	cursor cursor
}

func newReader(accessor *access) (*Reader, error) {
	return &Reader{
		active:   true,
		accessor: accessor,
		state: readState{
			eventBytes: -1,
			cursor: cursor{
				pageSize: accessor.PageSize(),
			},
		},
	}, nil
}

func (r *Reader) close() {
	r.active = false
}

// Available returns the number of unread events that can be read.
func (r *Reader) Available() uint {
	if !r.active {
		return 0
	}

	func() {
		tx := r.accessor.BeginRead()
		defer tx.Close()
		r.updateQueueState(tx)
	}()

	if r.state.cursor.Nil() {
		return 0
	}

	return uint(r.state.endID - r.state.id)
}

// Read reads the contents of the current event into the buffer.
// Returns 0 without reading if end of the current event has been reached.
// Use `Next` to skip/continue reading the next event.
func (r *Reader) Read(b []byte) (int, error) {
	if !r.active {
		return -1, errClosed
	}

	if r.state.eventBytes <= 0 {
		return 0, nil
	}

	to, err := r.readInto(b)
	return len(b) - len(to), err
}

func (r *Reader) readInto(to []byte) ([]byte, error) {
	tx := r.accessor.BeginRead()
	defer tx.Close()

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
	var err error
	if r.state.eventBytes == 0 {
		r.state.eventBytes = -1
		r.state.id++

		// As page is already in memory, use current transaction to try to skip to
		// next page if no more new event fits into current page.
		if cursor.PageBytes() < szEventHeader {
			cursor.AdvancePage()
		}
	}

	return to, err
}

// Next advances to the next event to be read. The event size in bytes is
// returned.  A size of 0 is reported if no more event is available in the
// queue.
func (r *Reader) Next() (int, error) {
	if !r.active {
		return -1, errClosed
	}

	tx := r.accessor.BeginRead()
	defer tx.Close()

	cursor := makeTxCursor(tx, r.accessor, &r.state.cursor)

	// in event? Skip contents
	if r.state.eventBytes > 0 {
		err := cursor.Skip(r.state.eventBytes)
		if err != nil {
			return 0, err
		}

		r.state.eventBytes = -1
		r.state.id++
	}

	// end of buffered queue state. Update state and check if we did indeed reach
	// the end of the queue.
	if cursor.Nil() || !idLess(r.state.id, r.state.endID) {
		err := r.updateQueueState(tx)
		if err != nil {
			return 0, err
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
			return 0, err
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
		return 0, err
	}
	L := int(hdr.sz.Get())
	r.state.eventBytes = L
	return L, nil
}

func (r *Reader) updateQueueState(tx *txfile.Tx) error {
	root, err := r.accessor.RootHdr(tx)
	if err != nil {
		return err
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
