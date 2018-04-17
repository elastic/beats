package pq

import "github.com/elastic/go-txfile"

// Access provides transaction support and access to pages and queue header.
// It wraps the Delegate for providing a common interface for working with
// transactions and files.
type access struct {
	Delegate
	rootID  txfile.PageID
	rootOff int
}

func makeAccess(delegate Delegate) (access, error) {
	rootID, rootOff := delegate.Root()
	if rootID == 0 {
		return access{}, errNoQueueRoot
	}

	return access{
		Delegate: delegate,
		rootID:   rootID,
		rootOff:  int(rootOff),
	}, nil
}

// ReadRoot reads the root page into an array.
// ReadRoot create a short lived read transaction for accessing and copying the
// queue root.
func (a *access) ReadRoot() ([SzRoot]byte, error) {
	var buf [SzRoot]byte

	tx := a.BeginRead()
	defer tx.Close()

	return buf, withPage(tx, a.rootID, func(page []byte) error {
		n := copy(buf[:], page[a.rootOff:])
		if n < SzRoot {
			return errIncompleteQueueRoot
		}
		return nil
	})
}

// RootPage accesses the queue root page from within the passed transaction.
func (a *access) RootPage(tx *txfile.Tx) (*txfile.Page, error) {
	return tx.Page(a.rootID)
}

// LoadRootPage accesses the queue root page from within the passed write
// transaction.
// The Root page it's content is loaded into the write buffer for manipulations.
// The page returned is not marked as dirty yet.
func (a *access) LoadRootPage(tx *txfile.Tx) (*txfile.Page, *queuePage, error) {
	var hdr *queuePage
	page, err := a.RootPage(tx)
	if err == nil {
		err = page.Load()
		if err == nil {
			buf, _ := page.Bytes()
			hdr = castQueueRootPage(buf[a.rootOff:])
		}
	}

	return page, hdr, err
}

// RootHdr returns a pointer to the queue root header. The pointer to the
// header is only valid as long as the transaction is still active.
func (a *access) RootHdr(tx *txfile.Tx) (hdr *queuePage, err error) {
	err = withPage(tx, a.rootID, func(buf []byte) error {
		hdr = castQueueRootPage(buf[a.rootOff:])
		return nil
	})
	return
}

// ParsePosition parses an on disk position, providing page id, page offset and
// event id in a more accessible format.
func (a *access) ParsePosition(p *pos) position {
	page, off := a.SplitOffset(uintptr(p.offset.Get()))
	if page != 0 && off == 0 {
		off = uintptr(a.PageSize())
	}

	return position{
		page: page,
		off:  int(off),
		id:   p.id.Get(),
	}
}

// WritePosition serializes a position into it's on-disk representation.
func (a *access) WritePosition(to *pos, pos position) {
	pageOff := pos.off
	if pageOff == a.PageSize() {
		pageOff = 0 // use 0 to mark page offset as end-of-page
	}

	off := a.Offset(pos.page, uintptr(pageOff))
	to.offset.Set(uint64(off))
	to.id.Set(pos.id)
}
