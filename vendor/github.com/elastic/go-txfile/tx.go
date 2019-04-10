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

package txfile

import (
	"sync"
	"time"

	"github.com/elastic/go-txfile/internal/cleanup"
	"github.com/elastic/go-txfile/internal/invariant"
)

// Tx provides access to pages in a File.
// A transaction MUST always be closed, so to guarantee locks being released as
// well.
type Tx struct {
	flags txFlags
	file  *File
	txid  uint // internal correlation id

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

	// transaction stats
	tsStart     time.Time
	accessStats txAccessStats
}

type txAccessStats struct {
	New    uint
	Read   uint
	Update uint
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

func newTx(file *File, id uint64, lock sync.Locker, settings TxOptions) *Tx {
	meta := file.getMetaPage()
	invariant.Check(meta != nil, "file meta is not set")

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

func (tx *Tx) onBegin() {
	o := tx.file.observer
	if o == nil {
		return
	}

	tx.tsStart = time.Now()
	o.OnTxBegin(tx.flags.readonly)
}

// onClose is called when a readonly transaction is closed.
func (tx *Tx) onClose() {
	o := tx.file.observer
	if o == nil {
		return
	}

	accessed := tx.accessStats.Read
	o.OnTxClose(tx.file.stats, TxStats{
		Readonly: true,
		Duration: time.Since(tx.tsStart),
		Total:    accessed,
		Accessed: accessed,
	})
}

// onRollback is called when a writable transaction is closed or rolled back without commit.
func (tx *Tx) onRollback() {
	o := tx.file.observer
	if o == nil {
		return
	}

	read := tx.accessStats.Read
	updated := tx.accessStats.Update
	new := tx.accessStats.New

	o.OnTxClose(tx.file.stats, TxStats{
		Readonly:  false,
		Commit:    false,
		Duration:  time.Since(tx.tsStart),
		Total:     read + updated + new,
		Accessed:  read,
		Updated:   updated,
		Written:   updated + new,
		Allocated: tx.alloc.stats.data.alloc,
		Freed:     tx.alloc.stats.data.freed,
	})
}

// onCommit is called after a writable transaction did succeed.
func (tx *Tx) onCommit() {
	allocStats := &tx.alloc.stats

	fileStats := &tx.file.stats
	fileStats.Size = uint64(tx.file.sizeEstimate)
	fileStats.MetaArea = tx.file.allocator.metaTotal
	fileStats.MetaAllocated = tx.file.allocator.metaTotal - tx.file.allocator.meta.freelist.Avail()
	fileStats.DataAllocated += allocStats.data.alloc - allocStats.data.freed - allocStats.toMeta

	o := tx.file.observer
	if o == nil {
		return
	}

	read := tx.accessStats.Read
	updated := tx.accessStats.Update
	new := tx.accessStats.New

	o.OnTxClose(tx.file.stats, TxStats{
		Readonly:  false,
		Commit:    true,
		Duration:  time.Since(tx.tsStart),
		Total:     read + updated + new,
		Accessed:  read,
		Allocated: allocStats.data.alloc - allocStats.toMeta,
		Freed:     allocStats.data.freed,
		Written:   updated + new,
		Updated:   updated,
	})
}

// onAccess is called when a the memory page pointer is requested.
func (tx *Tx) onAccess() {
	tx.accessStats.Read++
}

func (tx *Tx) onWALTransfer(n int) { // number of wal pages copied  into data area
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
	return tx.getPage("txfile/tx-access-root", tx.rootID)
}

// Rollback rolls back and closes the current transaction.  Rollback returns an
// error if the transaction has already been closed by Close, Rollback or
// Commit.
func (tx *Tx) Rollback() error {
	const op = "txfile/tx-rollback"

	tracef("rollback transaction: %p\n", tx)
	err := tx.finishWith(func() reason {
		tx.rollbackChanges()
		return nil
	})
	if err != nil {
		return tx.errWrap(op, err).of(TxRollbackFail)
	}
	return nil
}

// Commit commits the current transaction to file. The commit step needs to
// take the Exclusive Lock, waiting for readonly transactions to be Closed.
// Returns an error if the transaction has already been closed by Close,
// Rollback or Commit.
func (tx *Tx) Commit() error {
	const op = "txfile/tx-commit"

	tracef("commit transaction: %p\n", tx)
	err := tx.finishWith(tx.commitChanges)
	if err != nil {
		return tx.errWrap(op, err).of(TxCommitFail)
	}
	return nil
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
	const op = "txfile/tx-close"

	tracef("close transaction: %p\n", tx)
	if !tx.flags.active {
		return nil
	}

	err := tx.finishWith(func() reason {
		tx.rollbackChanges()
		return nil
	})
	if err != nil {
		return tx.errWrap(op, err).of(TxRollbackFail)
	}

	return nil
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
	if err := tx.canWrite("txfile/tx-checkpoint"); err != nil {
		return err
	}
	tx.doCheckpointWAL()
	return nil
}

func (tx *Tx) doCheckpointWAL() {
	if tx.flags.checkpoint {
		return
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
		return
	}

	// XXX: Some OS/filesystems might lock up when writing to file
	//      from mmapped area.
	//      -> Copy contents into temporary buffer, such that
	//         write operations are not backed by mmapped pages from same file.
	pageSize := int(tx.PageSize())
	writeBuffer := make([]byte, pageSize*len(ids))
	for i := range ids {
		id, walID := ids[i], walIDS[i]

		contents := tx.access(walID)
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

	tx.onWALTransfer(len(ids))
	tx.flags.checkpoint = true
}

func (tx *Tx) finishWith(fn func() reason) reason {
	if !tx.flags.active {
		return errOf(TxFinished).report("transaction is already closed")
	}
	defer tx.close()

	if tx.flags.readonly {
		tx.onClose()
		return nil
	}

	return fn()
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

func (tx *Tx) commitChanges() reason {
	commitOK := false
	defer cleanup.IfNot(&commitOK, tx.rollbackChanges)

	err := tx.tryCommitChanges()
	commitOK = err == nil
	if !commitOK {
		return err
	}

	traceMetaPage(tx.file.getMetaPage())
	tx.onCommit()
	return nil
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
func (tx *Tx) tryCommitChanges() reason {
	const op = "txfile/tx-commit"

	pending, exclusive := tx.file.locks.Pending(), tx.file.locks.Exclusive()

	// give concurrent read transactions a chance to complete, but don't allow
	// for new read transactions to start while executing the commit
	pending.Lock()
	defer pending.Unlock()

	// On function exit wait on writer to finish outstanding operations, in case
	// we have to return early on error. On success, this is basically a no-op.
	txWriteComplete := false
	defer cleanup.IfNot(&txWriteComplete, func() {
		err := tx.writeSync.Wait()

		// if wait fails, enforce an fsync with error reset flag.
		if err != nil {
			tx.file.writer.Sync(tx.writeSync, syncDataOnly|syncResetErr)
			tx.writeSync.Wait()
		}
	})

	// Flush pages.
	if err := tx.flushPages(op); err != nil {
		return tx.err(op).report("failed to flush dirty pages")
	}

	// 1. finish Tx state updates and free file pages used to hold meta pages
	csWAL, err := tx.commitPrepareWAL()
	if err != nil {
		return err
	}

	csAlloc := tx.commitPrepareAlloc()

	// 2. - 5. Commit changes to file
	metaID, err := tx.tryCommitChangesToFile(&csWAL, &csAlloc)
	if err != nil {
		return err
	}

	// 6. wait for all pages beeing written and synced,
	//    before updating in memory state.
	err = tx.writeSync.Wait()
	txWriteComplete = true
	if err != nil {
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

	// Compare required file size with the real file size and the mmaped region.
	// If the expected file size of the last transaction is < the real file size,
	// we can truncate the file and update the mmaped region.
	// If the expected file size is > the mmaped region, we need to update the mmaped file region.
	// If we fail here, the file and internal state is already updated + valid.
	// But mmap failed on us -> fatal error
	endMarker := tx.file.allocator.data.endMarker
	if metaEnd := tx.file.allocator.meta.endMarker; metaEnd > endMarker {
		endMarker = metaEnd
	}

	// Compute maximum expected file size of current transaction
	// and update the memory mapping if required.
	expectedMMapSize := int64(uint(endMarker) * tx.file.allocator.pageSize)
	maxSize := int64(tx.file.allocator.maxSize)
	pageSize := tx.file.allocator.pageSize
	requiredFileSz, truncate := checkTruncate(&tx.alloc, tx.file.size, expectedMMapSize, maxSize, pageSize)
	if truncate {
		err = tx.file.truncate(requiredFileSz)
	} else if int(expectedMMapSize) > len(tx.file.mapped) {
		err = tx.file.mmapUpdate()
	} else {
		sz := expectedMMapSize
		if sz < tx.file.size {
			sz = tx.file.size
		}

		tx.file.sizeEstimate = sz
	}
	if err != nil {
		return err
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

func (tx *Tx) tryCommitChangesToFile(
	csWAL *walCommitState,
	csAlloc *allocCommitState,
) (metaID int, err reason) {
	newMetaBuf := tx.prepareMetaBuffer()
	newMeta := newMetaBuf.cast()
	newMeta.root.Set(tx.rootID) // update data root

	// 2. allocate new file pages for new meta data to be written
	if err := tx.file.wal.fileCommitAlloc(tx, csWAL); err != nil {
		return metaID, err
	}
	csAlloc.updated = csAlloc.updated || len(csWAL.allocRegions) > 0

	if err := tx.file.allocator.fileCommitAlloc(csAlloc); err != nil {
		return metaID, err
	}

	// 3. serialize page mappings and new freelist
	err = tx.file.wal.fileCommitSerialize(csWAL, uint(tx.PageSize()), tx.scheduleCommitMetaWrite)
	if err != nil {
		return metaID, err
	}

	err = tx.file.allocator.fileCommitSerialize(csAlloc, tx.scheduleCommitMetaWrite)
	if err != nil {
		return metaID, err
	}

	// 4. sync all new contents and metadata before updating the ondisk meta page.
	tx.file.writer.Sync(tx.writeSync, syncDataOnly)

	// 5. finalize on-disk transaction by writing new meta page.
	tx.file.wal.fileCommitMeta(newMeta, csWAL)
	tx.file.allocator.fileCommitMeta(newMeta, csAlloc)
	metaID = tx.syncNewMeta(&newMetaBuf)

	// 6. wait for all pages beeing written and synced,
	//    before updating in memory state.
	return metaID, nil
}

func checkTruncate(
	st *txAllocState,
	sz, mmapSz, maxSz int64,
	pageSize uint,
) (int64, bool) {
	if maxSz <= 0 { // file is unbounded, no truncate required
		return 0, false
	}

	expectedFileSz := mmapSz
	if expectedFileSz < maxSz {
		expectedFileSz = maxSz
	}

	if expectedFileSz >= sz {
		// Required size still surpasses the last known file size -> do not
		// truncate.
		return 0, false
	}

	lastEnd := st.data.endMarker
	if metaEnd := st.meta.endMarker; metaEnd > lastEnd {
		lastEnd = metaEnd
	}

	lastExpectedFileSz := int64(uint(lastEnd) * pageSize)
	if lastExpectedFileSz < maxSz {
		lastExpectedFileSz = maxSz
	}

	// Compute minimum required file size for the last two active transactions (maximum).
	if lastExpectedFileSz > expectedFileSz {
		expectedFileSz = lastExpectedFileSz
	}

	return expectedFileSz, expectedFileSz < sz
}

func (tx *Tx) prepareMetaBuffer() (buf metaBuf) {
	meta := buf.cast()
	*meta = *tx.file.getMetaPage()
	meta.txid.Set(1 + meta.txid.Get())
	return
}

func (tx *Tx) syncNewMeta(buf *metaBuf) int {
	meta := buf.cast()
	meta.Finalize()

	metaID := 1 - tx.file.metaActive
	tx.file.writer.Schedule(tx.writeSync, PageID(metaID), (*buf)[:])
	tx.file.writer.Sync(tx.writeSync, syncDataOnly|syncResetErr)
	return metaID
}

func (tx *Tx) commitPrepareWAL() (walCommitState, reason) {
	var st walCommitState

	tx.file.wal.fileCommitPrepare(&st, &tx.wal)
	if st.checkpoint {
		tx.doCheckpointWAL()
	}

	if st.updated {
		tx.freeMetaRegions(tx.file.wal.metaPages)
	}
	return st, nil
}

func (tx *Tx) commitPrepareAlloc() (state allocCommitState) {
	tx.file.allocator.fileCommitPrepare(&state, &tx.alloc, false)
	if state.updated {
		tx.freeMetaRegions(tx.file.allocator.freelistPages)
	}
	return state
}

func (tx *Tx) freeMetaRegions(rl regionList) {
	tx.metaAllocator().FreeRegions(&tx.alloc, rl)
}

func (tx *Tx) access(id PageID) []byte {
	tx.onAccess()
	return tx.file.mmapedPage(id)
}

// scheduleCommitMetaWrite is used to schedule a page write for the file meta
// data like free list or page mappings. scheduleCommitMetaWrite must only be
// used during file updates in the commit phase.
func (tx *Tx) scheduleCommitMetaWrite(id PageID, buf []byte) reason {
	tx.accessStats.New++
	return tx.scheduleWrite(id, buf)
}

func (tx *Tx) scheduleWrite(id PageID, buf []byte) reason {
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
func (tx *Tx) rollbackChanges() {
	tracef("rollback changes in transaction: %p\n", tx)
	tx.onRollback()

	tx.file.allocator.Rollback(&tx.alloc)

	maxPages := tx.file.allocator.maxPages
	if maxPages == 0 {
		return
	}

	// compute endmarker from before running the last transaction
	endMarker := tx.file.allocator.meta.endMarker
	if dataEnd := tx.file.allocator.data.endMarker; dataEnd > endMarker {
		endMarker = dataEnd
	}

	sz, err := tx.file.file.Size()
	if err != nil {
		// getting file size failed. State is valid, but we can not truncate
		// ¯\_(ツ)_/¯
		return
	}

	truncateSz := uint(endMarker) * tx.file.allocator.pageSize
	if uint(sz) > uint(truncateSz) {
		// ignore truncate error, as truncating a memory mapped file might not be
		// supported by all OSes/filesystems.
		err := tx.file.file.Truncate(int64(truncateSz))
		if err != nil {
			traceln("rollback file truncate failed with:", err)
		}
	}
}

// Page accesses a page by ID. Accessed pages are cached. Retrieving a page
// that has already been accessed, will return a pointer to the same Page object.
// Returns an error if the id is known to be invalid or the page has already
// been freed.
func (tx *Tx) Page(id PageID) (*Page, error) {
	const op = "txfile/tx-access-page"
	return tx.getPage(op, id)
}

func (tx *Tx) getPage(op string, id PageID) (*Page, error) {
	inBounds := id >= 2
	if tx.flags.readonly {
		inBounds = inBounds && id < tx.dataEndID
	} else {
		inBounds = inBounds && id < tx.file.allocator.data.endMarker
	}
	if !inBounds {
		return nil, tx.errWrap(op, raiseOutOfBounds(id))
	}

	if tx.alloc.data.freed.Has(id) || tx.alloc.meta.freed.Has(id) {
		return nil, tx.err(op).of(InvalidOp).
			report("trying to access an already freed page")
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
	const op = "txfile/tx-alloc-page"

	if err := tx.canWrite(op); err != nil {
		return nil, err
	}

	err = tx.allocPagesWith(op, 1, func(p *Page) { page = p })
	return page, err
}

// AllocN allocates n potentially non-contious, yet empty pages.
// Returns an error if the transaction is readonly or no more space is available.
func (tx *Tx) AllocN(n int) (pages []*Page, err error) {
	const op = "txfile/tx-alloc-pages"

	if err := tx.canWrite(op); err != nil {
		return nil, err
	}

	if n <= 0 {
		return nil, nil
	}

	pages, i := make([]*Page, n), 0
	err = tx.allocPagesWith(op, n, func(page *Page) {
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

func (tx *Tx) allocPagesWith(op string, n int, fn func(*Page)) reason {
	count := tx.dataAllocator().AllocRegionsWith(&tx.alloc, uint(n), func(reg region) {
		reg.EachPage(func(id PageID) {
			page := newPage(tx, id)
			page.flags.new = true
			tx.pages[id] = page
			fn(page)
		})
	})
	if count == 0 {
		return tx.err(op).of(OutOfMemory).reportf("not enough memory to allocate %v data page(s)", n)
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
	return tx.flushPages("txfile/tx-flush")
}

func (tx *Tx) flushPages(op string) reason {
	if err := tx.canWrite(op); err != nil {
		return err
	}

	for _, page := range tx.pages {
		if err := page.doFlush("txfile/page-flush"); err != nil {
			return err
		}
	}
	return nil
}

func (tx *Tx) canRead(op string) *Error {
	if !tx.flags.active {
		return tx.err(op).of(TxFinished).report("no read operation on finished transactions allowed")
	}
	return nil
}

func (tx *Tx) canWrite(op string) *Error {
	var kind ErrKind
	var msg string

	if !tx.flags.active {
		kind, msg = TxFinished, "no write operation on finished transactions allowed"
	}
	if tx.flags.readonly {
		kind, msg = TxReadOnly, "no write operation on read only transaction allowed"
	}

	if kind != NoError {
		return tx.err(op).of(kind).report(msg)
	}
	return nil
}

func (tx *Tx) err(op string) *Error {
	return &Error{op: op, ctx: tx.errCtx()}
}

func (tx *Tx) errWrap(op string, cause error) *Error {
	return tx.err(op).causedBy(cause)
}

func (tx *Tx) errCtx() errorCtx {
	ctx := tx.file.errCtx()
	ctx.txid, ctx.isTx = tx.txid, true
	return ctx
}
