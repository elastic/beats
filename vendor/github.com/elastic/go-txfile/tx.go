package txfile

import (
	"fmt"
	"sync"

	"github.com/elastic/go-txfile/internal/cleanup"
)

// Tx provides access to pages in a File.
// A transaction MUST always be closed, so to guarantee locks being released as
// well.
type Tx struct {
	flags     txFlags
	file      *File
	lock      sync.Locker
	writeSync *txWriteSync
	rootID    PageID
	dataEndID PageID

	// pages accessed by the transaction
	pages map[PageID]*Page

	// allocation/free state
	alloc txAllocState

	// scheduled WAL updates
	wal txWalState
}

// TxOptions adds some per transaction options user can set.
type TxOptions struct {
	// Readonly transaction.
	Readonly bool

	// Allow write transaction to allocate meta pages from overflow area.
	// Potentially increasing the file size past the configured max size.
	// This setting should only be used to guarantee progress when having a
	// transaction only freeing pages.
	// Later transactions will try to release pages from the overflow area and
	// truncate the file, such that we have a chance to operate within max-size
	// limits again.
	EnableOverflowArea bool

	// MetaAreaGrowPercentage sets the percentage of meta pages in use, until
	// the meta-area grows again. The value must be between 0 and 100.
	// The default value is 80%.
	MetaAreaGrowPercentage int

	// Number of pages in wal overwrite log to automatically trigger
	// CheckpointWAL on commit.
	WALLimit uint
}

type txFlags struct {
	readonly   bool
	active     bool
	checkpoint bool // mark wal checkpoint has been applied
}

func newTx(file *File, lock sync.Locker, settings TxOptions) *Tx {
	meta := file.getMetaPage()

	rootID := meta.root.Get()
	dataEndMarker := meta.dataEndMarker.Get()

	tx := &Tx{
		flags: txFlags{
			readonly: settings.Readonly,
			active:   true,
		},
		file:      file,
		lock:      lock,
		rootID:    rootID,
		dataEndID: dataEndMarker,

		pages: map[PageID]*Page{},
	}

	if !settings.Readonly {
		tx.writeSync = newTxWriteSync()
		tx.alloc = file.allocator.makeTxAllocState(
			settings.EnableOverflowArea,
			settings.MetaAreaGrowPercentage,
		)
		tx.wal = file.wal.makeTxWALState(settings.WALLimit)
	}

	return tx
}

// Writable returns true if the transaction supports file modifications.
func (tx *Tx) Writable() bool {
	return !tx.flags.readonly
}

// Readonly returns true if no modifications to the page are allowed. Trying to
// write to a readonly page might result in a non-recoverable panic.
func (tx *Tx) Readonly() bool {
	return tx.flags.readonly
}

// Active returns true if the transaction can still be used to access pages.
// A transaction becomes inactive after Close, Commit or Rollback.
// Errors within a transaction might inactivate the transaction as well.
// When encountering errors, one should check if the transaction still can be used.
func (tx *Tx) Active() bool {
	return tx.flags.active
}

// PageSize returns the file page size.
func (tx *Tx) PageSize() int {
	return int(tx.file.allocator.pageSize)
}

// Root returns the data root page id. This ID must be set via SetRoot
// to indicate the start of application data to later transactions.
// On new files, the default root is 0, as no application data are stored yet.
func (tx *Tx) Root() PageID {
	return tx.rootID
}

// SetRoot sets the new root page id, indicating the new start of application
// data. SetRoot should be set by the first write transaction, when the file is
// generated first.
func (tx *Tx) SetRoot(id PageID) {
	tx.rootID = id
}

// RootPage returns the application data root page, if the root id has been set
// in the past. Returns nil, if no root page is set.
func (tx *Tx) RootPage() (*Page, error) {
	if tx.rootID < 2 {
		return nil, nil
	}
	return tx.Page(tx.rootID)
}

// Rollback rolls back and closes the current transaction.  Rollback returns an
// error if the transaction has already been closed by Close, Rollback or
// Commit.
func (tx *Tx) Rollback() error {
	tracef("rollback transaction: %p\n", tx)
	return tx.finishWith(tx.rollbackChanges)
}

// Commit commits the current transaction to file. The commit step needs to
// take the Exclusive Lock, waiting for readonly transactions to be Closed.
// Returns an error if the transaction has already been closed by Close,
// Rollback or Commit.
func (tx *Tx) Commit() error {
	tracef("commit transaction: %p\n", tx)
	return tx.finishWith(tx.commitChanges)
}

// Close closes the transaction, releasing any locks held by the transaction.
// It is safe to call Close multiple times. Close on an inactive transaction
// will be ignored.
// A non-committed read-write transaction will be rolled back on close.
// To guaranteed the File and Locking state being valid, even on panic or early return on error,
// one should also defer the Close operation on new transactions.
// For example:
//
//     tx := f.Begin()
//     defer tx.Close()
//
//     err := some operation
//     if err != nil {
//       return err
//     }
//
//     return tx.Commit()
//
func (tx *Tx) Close() error {
	tracef("close transaction: %p\n", tx)
	if !tx.flags.active {
		return nil
	}
	return tx.finishWith(tx.rollbackChanges)
}

// CheckpointWAL copies all overwrite pages contents into the original pages.
// Only already committed pages from older transactions will be overwritten.
// Checkpointing only copies the contents and marks the overwrite pages as
// freed. The final transaction Commit is required, to propage the WAL mapping changes
// to all other transactions.
// Dirty pages are not overwritten. Manual checkpointing should be executed at
// the end of a transaction, right before committing, so to reduce writes if
// contents is to be overwritten anyways.
func (tx *Tx) CheckpointWAL() error {
	if err := tx.canWrite(); err != nil {
		return err
	}
	return tx.doCheckpointWAL()
}

func (tx *Tx) doCheckpointWAL() error {
	if tx.flags.checkpoint {
		return nil
	}

	// collect page ids that would have an old WAL page
	// entry still alive after this transaction.
	ids := make([]PageID, 0, len(tx.file.wal.mapping))
	walIDS := make([]PageID, 0, len(tx.file.wal.mapping))
	for id, walID := range tx.file.wal.mapping {
		page := tx.pages[id]
		if page != nil {
			if page.flags.dirty {
				// wal pages of dirty pages will be freed on flush -> do not copy
				continue
			}
		}

		ids = append(ids, id)
		walIDS = append(walIDS, walID)
	}

	if len(ids) == 0 {
		return nil
	}

	// XXX: Some OS/filesystems might lock up when writing to file
	//      from mmapped area.
	//      -> Copy contents into temporary buffer, such that
	//         write operations are not backed by mmapped pages from same file.
	pageSize := int(tx.PageSize())
	writeBuffer := make([]byte, pageSize*len(ids))
	for i := range ids {
		id, walID := ids[i], walIDS[i]

		contents := tx.file.mmapedPage(walID)
		if contents == nil {
			panic("invalid WAL mapping")
		}

		tracef("checkpoint copy from WAL page %v -> %v\n", walID, id)

		n := copy(writeBuffer, contents)
		buf := writeBuffer[:n]
		writeBuffer = writeBuffer[n:]

		tx.file.writer.Schedule(tx.writeSync, id, buf)
		tx.freeWALID(id, walID)
	}

	tx.flags.checkpoint = true
	return nil
}

func (tx *Tx) finishWith(fn func() error) error {
	if !tx.flags.active {
		return errTxFinished
	}
	defer tx.close()

	if !tx.flags.readonly {
		return fn()
	}
	return nil
}

func (tx *Tx) close() {
	tx.flags.active = false
	tx.pages = nil
	tx.alloc = txAllocState{}
	tx.wal = txWalState{}
	tx.writeSync = nil
	tx.file = nil
	tx.lock.Unlock()
}

func (tx *Tx) commitChanges() error {
	commitOK := false
	defer cleanup.IfNot(&commitOK, cleanup.IgnoreError(tx.rollbackChanges))

	err := tx.tryCommitChanges()
	if commitOK = err == nil; !commitOK {
		return err
	}

	traceMetaPage(tx.file.getMetaPage())
	return err
}

// tryCommitChanges attempts to write flush all pages written and update the
// files state by writing the new meta data and finally the meta page.
// So to keep the most recent transaction successfully committed usable/consistent,
// tryCommitChanges is not allowed to re-use any pages freed within this transaction.
//
// rough commit sequence:
// 1. get pending lock, so no new readers can be started
// 2. flush all dirty pages.
//   - dirty pages overwriting existing contents will, will allocate
//     a new WAL page to be written to
//   - If dirty page already has an WAL page, overwrite the original page and
//     return WAL page to allocator
// 3. if WAL was updated (pages added/removed):
//    - free pages holding the old WAL mapping
//    - write new WAL mapping
// 4. if pages have been freed/allocated:
//    - free pages holding the old free list entries
//    - write new free list
// 5. fsync, to ensure all updates have been executed before updating the meta page
// 6. acquire esclusive lock -> no more readers/writers accessing the file
// 6. update the meta page
// 7. fsync
// 8. update internal structures
// 9. release locks
func (tx *Tx) tryCommitChanges() error {
	pending, exclusive := tx.file.locks.Pending(), tx.file.locks.Exclusive()

	var newMetaBuf metaBuf
	newMeta := newMetaBuf.cast()
	*newMeta = *tx.file.getMetaPage()        // init new meta header from current active meta header
	newMeta.txid.Set(1 + newMeta.txid.Get()) // inc txid
	newMeta.root.Set(tx.rootID)              // update data root

	// give concurrent read transactions a chance to complete, but don't allow
	// for new read transactions to start while executing the commit
	pending.Lock()
	defer pending.Unlock()

	// On function exit wait on writer to finish outstanding operations, in case
	// we have to return early on error. On success, this is basically a no-op.
	defer tx.writeSync.Wait()

	// Flush pages.
	if err := tx.Flush(); err != nil {
		return fmt.Errorf("dirty pages flushing failed with %v", err)
	}

	// 1. finish Tx state updates and free file pages used to hold meta pages
	csWAL, err := tx.commitPrepareWAL()
	if err != nil {
		return err
	}

	var csAlloc allocCommitState
	tx.file.allocator.fileCommitPrepare(&csAlloc, &tx.alloc)

	// 2. allocate new file pages for new meta data to be written
	if err := tx.file.wal.fileCommitAlloc(tx, &csWAL); err != nil {
		return err
	}
	csAlloc.updated = csAlloc.updated || len(csWAL.allocRegions) > 0

	if err := tx.file.allocator.fileCommitAlloc(&csAlloc); err != nil {
		return err
	}

	// 3. serialize page mappings and new freelist
	err = tx.file.wal.fileCommitSerialize(&csWAL, uint(tx.PageSize()), tx.scheduleWrite)
	if err != nil {
		return err
	}

	err = tx.file.allocator.fileCommitSerialize(&csAlloc, tx.scheduleWrite)
	if err != nil {
		return err
	}

	// 4. sync all new contents and metadata before updating the ondisk meta page.
	tx.file.writer.Sync(tx.writeSync)

	// 5. finalize on-disk transaction be writing new meta page.
	tx.file.wal.fileCommitMeta(newMeta, &csWAL)
	tx.file.allocator.fileCommitMeta(newMeta, &csAlloc)
	newMeta.Finalize()
	metaID := 1 - tx.file.metaActive
	tx.file.writer.Schedule(tx.writeSync, PageID(metaID), newMetaBuf[:])
	tx.file.writer.Sync(tx.writeSync)

	// 6. wait for all pages beeing written and synced,
	//    before updating in memory state.
	if err := tx.writeSync.Wait(); err != nil {
		return err
	}

	// At this point the transaction has been completed on file level.
	// Update internal structures as well, so future transactions
	// will use the new serialized transaction state.

	// We have only one active write transaction + freelist is not shared with read transactions
	// -> update freelist state before waiting for the exclusive lock to be available
	tx.file.allocator.Commit(&csAlloc)

	// Wait for all read transactions to finish before updating global references
	// to new contents.
	exclusive.Lock()
	defer exclusive.Unlock()

	// Update the WAL mapping.
	tx.file.wal.Commit(&csWAL)

	// Switch the files active meta page to meta page being written.
	tx.file.metaActive = metaID

	// check + apply mmap update. If we fail here, the file and internal
	// state is already updated + valid.
	// But mmap failed on us -> fatal error
	endMarker := tx.file.allocator.data.endMarker
	if metaEnd := tx.file.allocator.meta.endMarker; metaEnd > endMarker {
		endMarker = metaEnd
	}
	fileSize := uint(endMarker) * tx.file.allocator.pageSize
	if int(fileSize) > len(tx.file.mapped) {
		err = tx.file.mmapUpdate()
	}

	traceln("tx stats:")
	traceln("  available data pages:", tx.file.allocator.DataAllocator().Avail(nil))
	traceln("  available meta pages:", tx.file.allocator.meta.freelist.Avail())
	traceln("  total meta pages:", tx.file.allocator.metaTotal)
	traceln("    freelist pages:", len(tx.file.allocator.freelistPages))
	traceln("    wal mapping pages:", len(tx.file.wal.metaPages))
	traceln("  max pages:", tx.file.allocator.maxPages)
	traceln("  wal mapped pages:", len(tx.file.wal.mapping))

	return nil
}

func (tx *Tx) commitPrepareWAL() (walCommitState, error) {
	var st walCommitState

	tx.file.wal.fileCommitPrepare(&st, &tx.wal)
	if st.checkpoint {
		if err := tx.doCheckpointWAL(); err != nil {
			return st, err
		}
	}

	if st.updated {
		tx.metaAllocator().FreeRegions(&tx.alloc, tx.file.wal.metaPages)
	}
	return st, nil
}

func (tx *Tx) access(id PageID) []byte {
	return tx.file.mmapedPage(id)
}

func (tx *Tx) scheduleWrite(id PageID, buf []byte) error {
	tx.file.writer.Schedule(tx.writeSync, id, buf)
	return nil
}

// rollbackChanges undoes all changes scheduled.
// Potentially changes to be undone:
//  1. WAL:
//    - mapping is only updated after ACK.
//    - pages have been allocated from meta area -> only restore freelists
//  2. Allocations:
//    - restore freelists, by returning allocated page
//      ids < old endmarker to freelists
//    - restore old end markers.
//    - move pages allocated into meta area back into data area
//  3. File:
//    - With page flushing or transaction failing late during commit,
//      file might have been grown.
//      =>
//        - Truncate file only if pages in overflow area have been allocated.
//        - If maxSize == 0, truncate file to old end marker.
func (tx *Tx) rollbackChanges() error {
	tx.file.allocator.Rollback(&tx.alloc)

	maxPages := tx.file.allocator.maxPages
	if maxPages == 0 {
		return nil
	}

	// compute endmarker from before running the last transaction
	endMarker := tx.file.allocator.meta.endMarker
	if dataEnd := tx.file.allocator.data.endMarker; dataEnd > endMarker {
		endMarker = dataEnd
	}

	sz, err := tx.file.file.Size()
	if err != nil {
		// getting file size failed. State is valid, but we can not truncate :/
		return err
	}

	truncateSz := uint(endMarker) * tx.file.allocator.pageSize
	if uint(sz) > uint(truncateSz) {
		return tx.file.file.Truncate(int64(truncateSz))
	}

	return nil
}

// Page accesses a page by ID. Accessed pages are cached. Retrieving a page
// that has already been accessed, will return a pointer to the same Page object.
// Returns an error if the id is known to be invalid or the page has already
// been freed.
func (tx *Tx) Page(id PageID) (*Page, error) {
	inBounds := id >= 2
	if tx.flags.readonly {
		inBounds = inBounds && id < tx.dataEndID
	} else {
		inBounds = inBounds && id < tx.file.allocator.data.endMarker
	}
	if !inBounds {
		return nil, errOutOfBounds
	}

	if tx.alloc.data.freed.Has(id) || tx.alloc.meta.freed.Has(id) {
		return nil, errFreedPage
	}

	if p := tx.pages[id]; p != nil {
		return p, nil
	}

	page := newPage(tx, id)
	if walID := tx.file.wal.Get(id); walID != 0 {
		page.ondiskID = walID
	}

	tx.pages[id] = page
	return page, nil
}

// Alloc allocates a new writable page with yet empty contents.
// Use Load(), Bytes and MarkDirty(), or SetBytes() to fill the page with
// new contents.
// Returns an error if the transaction is readonly or no more space is available.
func (tx *Tx) Alloc() (page *Page, err error) {
	if err := tx.canWrite(); err != nil {
		return nil, err
	}

	err = tx.allocPagesWith(1, func(p *Page) { page = p })
	return
}

// AllocN allocates n potentially non-contious, yet empty pages.
// Returns an error if the transaction is readonly or no more space is available.
func (tx *Tx) AllocN(n int) (pages []*Page, err error) {
	if err := tx.canWrite(); err != nil {
		return nil, err
	}

	if n <= 0 {
		return nil, nil
	}

	pages, i := make([]*Page, n), 0
	err = tx.allocPagesWith(n, func(page *Page) {
		pages[i], i = page, i+1
	})
	if err != nil {
		return nil, err
	}
	return pages, nil
}

func (tx *Tx) dataAllocator() *dataAllocator {
	return tx.file.allocator.DataAllocator()
}

func (tx *Tx) metaAllocator() *metaAllocator {
	return tx.file.allocator.MetaAllocator()
}

func (tx *Tx) walAllocator() *walAllocator {
	return tx.file.allocator.WALPageAllocator()
}

func (tx *Tx) allocPagesWith(n int, fn func(*Page)) error {
	count := tx.dataAllocator().AllocRegionsWith(&tx.alloc, uint(n), func(reg region) {
		reg.EachPage(func(id PageID) {
			page := newPage(tx, id)
			page.flags.new = true
			tx.pages[id] = page
			fn(page)
		})
	})
	if count == 0 {
		return errOutOfMemory
	}
	return nil
}

func (tx *Tx) freePage(id PageID) {
	tx.dataAllocator().Free(&tx.alloc, id)
}

func (tx *Tx) allocWALID(orig PageID) PageID {
	id := tx.walAllocator().Alloc(&tx.alloc)
	if id != 0 {
		tx.wal.Set(orig, id)
	}
	return id
}

func (tx *Tx) freeWALID(id, walID PageID) {
	tx.walAllocator().Free(&tx.alloc, walID)
	tx.wal.Release(id)
}

// Flush flushes all dirty pages within the transaction.
func (tx *Tx) Flush() error {
	if err := tx.canWrite(); err != nil {
		return err
	}

	for _, page := range tx.pages {
		if err := page.doFlush(); err != nil {
			return err
		}
	}
	return nil
}

func (tx *Tx) canWrite() error {
	if !tx.flags.active {
		return errTxFinished
	}
	if tx.flags.readonly {
		return errTxReadonly
	}
	return nil
}
