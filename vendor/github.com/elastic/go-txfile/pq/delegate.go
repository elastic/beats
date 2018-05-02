package pq

import "github.com/elastic/go-txfile"

// Delegate is used by the persistent queue to query common parameters and
// start transactions when required.
type Delegate interface {
	// PageSize reports the page size to be used by the backing file.
	PageSize() int

	// Root returns the queues root on file.
	Root() (txfile.PageID, uintptr)

	Offset(id txfile.PageID, offset uintptr) uintptr

	SplitOffset(uintptr) (txfile.PageID, uintptr)

	// BeginWrite must create a read-write transaction for use by the writer.
	// The transaction will be used to allocate pages and flush the current write
	// buffer.
	BeginWrite() *txfile.Tx

	// BeginRead must return a readonly transaction.
	BeginRead() *txfile.Tx

	// BeginCleanup must return a read-write transaction for the ACK handling to
	// remove events. No new contents will be written, but pages will be freed
	// and the queue root page being updated.
	BeginCleanup() *txfile.Tx
}

// standaloneDelegate wraps a txfile.File into a standalone queue only file.
// The delegate sets the files root to the queue header.
type standaloneDelegate struct {
	file *txfile.File
	root txfile.PageID
}

// NewStandaloneDelegate creates a standaonle Delegate from an txfile.File
// instance.  This function will allocate and initialize the queue root page.
func NewStandaloneDelegate(f *txfile.File) (Delegate, error) {
	tx := f.Begin()
	defer tx.Close()

	root := tx.Root()
	if root == 0 {
		var err error

		root, err = initQueueRoot(tx)
		if err != nil {
			return nil, err
		}
	}

	return &standaloneDelegate{file: f, root: root}, nil
}

func initQueueRoot(tx *txfile.Tx) (txfile.PageID, error) {
	page, err := tx.Alloc()
	if err != nil {
		return 0, err
	}

	buf := MakeRoot()
	if err := page.SetBytes(buf[:]); err != nil {
		return 0, err
	}

	tx.SetRoot(page.ID())
	return page.ID(), tx.Commit()
}

// PageSize returns the files page size.
func (d *standaloneDelegate) PageSize() int {
	return d.file.PageSize()
}

// Root finds the queue root page and offset.
func (d *standaloneDelegate) Root() (txfile.PageID, uintptr) {
	return d.root, 0
}

func (d *standaloneDelegate) Offset(id txfile.PageID, offset uintptr) uintptr {
	return d.file.Offset(id, offset)
}

func (d *standaloneDelegate) SplitOffset(offset uintptr) (txfile.PageID, uintptr) {
	return d.file.SplitOffset(offset)
}

// BeginWrite creates a new transaction for flushing the write buffers to disk.
func (d *standaloneDelegate) BeginWrite() *txfile.Tx {
	return d.file.BeginWith(txfile.TxOptions{
		WALLimit: 3,
	})
}

// BeginRead returns a readonly transaction.
func (d *standaloneDelegate) BeginRead() *txfile.Tx {
	return d.file.BeginReadonly()
}

// BeginCleanup creates a new write transaction configured for cleaning up used
// events/pages only.
func (d *standaloneDelegate) BeginCleanup() *txfile.Tx {
	return d.file.BeginWith(txfile.TxOptions{
		EnableOverflowArea: true,
		WALLimit:           3,
	})
}
