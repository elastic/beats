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
	"fmt"
	"math"
	"math/bits"
	"os"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/elastic/go-txfile/internal/cleanup"
	"github.com/elastic/go-txfile/internal/invariant"
	"github.com/elastic/go-txfile/internal/vfs"
	"github.com/elastic/go-txfile/internal/vfs/osfs"
)

// File provides transactional support to pages of a file. A file is split into
// pages of type PageSize. Pages within the file are only accessible by page IDs
// from within active transactions.
type File struct {
	// Atomic fields.
	// Do not move: Must be 64bit-word aligned on some architectures.
	txids uint64

	observer Observer

	path     string
	readonly bool
	file     vfs.File

	size         int64 // real file size (updated on mmap update only)
	sizeEstimate int64 // estimated real file size based on last update and the total vs. used mmaped region

	locks     lock
	wg        sync.WaitGroup // local async workers wait group
	writer    writer
	allocator allocator
	wal       waLog

	// mmap info
	mapped []byte

	// meta pages
	meta       [2]*metaPage
	metaActive int

	stats FileStats
}

// internal contants
const (
	initBits    uint = 16            // 2 ^ 16 Bytes
	initSize         = 1 << initBits // 64KB
	sz1GB            = 1 << 30
	doubleLimit      = sz1GB // upper limit when to stop doubling the mmaped area

	minRequiredFileSize = initSize
)

var maxMmapSize uint

func init() {
	if math.MaxUint32 == maxUint {
		maxMmapSize = 2 * sz1GB
	} else {
		tmp := uint64(0x1FFFFFFFFFFF)
		maxMmapSize = uint(tmp)
	}
}

// Open opens or creates a new transactional file.
// Open tries to create the file, if the file does not exist yet.  Returns an
// error if file access fails, file can not be locked or file meta pages are
// found to be invalid.
func Open(path string, mode os.FileMode, opts Options) (*File, error) {
	const op = "txfile/open"

	if err := opts.Validate(); err != nil {
		return nil, fileErrWrap(op, path, err)
	}

	file, err := osfs.Open(path, mode)
	if err != nil {
		return nil, fileErrWrap(op, path, err).report("can not open file")
	}

	initOK := false
	defer cleanup.IfNot(&initOK, cleanup.IgnoreError(file.Close))

	waitLock := (opts.Flags & FlagWaitLock) == FlagWaitLock
	if err := file.Lock(true, waitLock); err != nil {
		return nil, fileErrWrap(op, path, err).report("failed to lock file")
	}
	defer cleanup.IfNot(&initOK, cleanup.IgnoreError(file.Unlock))

	// initialize the file
	f, err := openWith(file, opts)
	if err != nil {
		return nil, fileErrWrap(op, path, err).report("failed to open file")
	}

	tracef("open file: %p (%v)\n", f, path)
	traceMetaPage(f.getMetaPage())

	f.reportOpen()

	initOK = true
	return f, nil
}

// openWith implements the actual opening sequence, including file
// initialization and validation.
func openWith(file vfs.File, opts Options) (*File, reason) {
	sz, ferr := file.Size()
	if ferr != nil {
		return nil, wrapErr(ferr)
	}

	isNew := false
	fileExists := sz > 0
	if !fileExists {
		if err := initNewFile(file, opts); err != nil {
			return nil, err
		}

		isNew = true
	}

	meta, metaActive, err := readValidMeta(file)
	if err != nil {
		return nil, err
	}

	pageSize := meta.pageSize.Get()

	maxSize := meta.maxSize.Get()
	if maxSize == 0 && opts.MaxSize > 0 {
		maxSize = opts.MaxSize
	}

	if maxSize > uint64(maxUint) {
		return nil, raiseInvalidParam("max file size to large for this system")
	}

	f, err := newFile(file, opts, metaActive, uint(maxSize), uint(pageSize))
	if err != nil {
		return nil, err
	}

	// Update the files MaxSize after the new file object has been created.
	// This allows us to handle the max size update like a transaction.
	if (!isNew && opts.Flags.check(FlagUpdMaxSize)) && opts.MaxSize != maxSize {
		ok := false
		defer cleanup.IfNot(&ok, cleanup.IgnoreError(f.Close))

		op := growFile
		if opts.MaxSize > 0 && opts.MaxSize < maxSize {
			op = shrinkFile
		}

		err := op(f, opts)
		if err != nil {
			return nil, err
		}

		ok = true
	}

	return f, nil
}

// newFile creates and initializes a new File. File state is initialized
// from file and internal workers will be started.
func newFile(
	file vfs.File,
	opts Options,
	metaActive int,
	maxSize, pageSize uint,
) (*File, reason) {

	f := &File{
		file: file,
		path: file.Name(),
		allocator: allocator{
			maxSize:  maxSize,
			pageSize: pageSize,
		},
		observer: opts.Observer,
	}
	f.locks.init()

	if err := f.mmap(); err != nil {
		return nil, err
	}
	initOK := false
	defer cleanup.IfNot(&initOK, ignoreReason(f.munmap))

	if err := f.init(metaActive, opts); err != nil {
		return nil, err
	}

	invariant.CheckNot(f.allocator.maxSize != 0 && f.allocator.maxPages == 0,
		"page limit not configured on allocator")

	// create asynchronous writer
	f.writer.Init(file, f.allocator.pageSize, opts.Sync)
	f.wg.Add(1)
	go func() {
		defer f.wg.Done()
		f.writer.Run()
	}()

	initOK = true
	return f, nil
}

// init initializes the File state from most recent valid meta-page.
func (f *File) init(metaActive int, opts Options) reason {
	const op = "txfile/init-read-state"

	// reference active meta page for initializing internal structures
	f.metaActive = metaActive
	meta := f.meta[f.metaActive]

	if err := readWALMapping(&f.wal, f.mmapedPage, meta.wal.Get()); err != nil {
		return f.errWrap(op, err).of(InitFailed)
	}

	if err := readAllocatorState(&f.allocator, f, meta, opts); err != nil {
		return f.errWrap(op, err).of(InitFailed)
	}

	return nil
}

func (f *File) reportOpen() {
	const numFileHeaders = 2

	meta := f.getMetaPage()
	fileEnd := uint(meta.dataEndMarker.Get())
	if m := uint(meta.metaEndMarker.Get()); m > fileEnd {
		fileEnd = m
	}

	metaArea := uint(meta.metaTotal.Get())
	metaInUse := metaArea - f.allocator.meta.freelist.Avail()
	dataInUse := fileEnd - numFileHeaders - metaArea - f.allocator.data.freelist.Avail()

	f.stats = FileStats{
		Version:       meta.version.Get(),
		Size:          uint64(f.size),
		MaxSize:       meta.maxSize.Get(),
		PageSize:      meta.pageSize.Get(),
		MetaArea:      metaArea,
		DataAllocated: dataInUse,
		MetaAllocated: metaInUse,
	}

	o := f.observer
	if o == nil {
		return
	}
	o.OnOpen(f.stats)
}

// Close closes the file, after all transactions have been quit. After closing
// a file, no more transactions can be started.
func (f *File) Close() error {
	// zero out f on exit -> using f after close should generate a panic
	defer func() { *f = File{} }()

	tracef("start file shutdown: %p\n", f)
	defer tracef("file closed: %p\n", f)

	// get reserved lock, such that no write transactions can be started
	f.locks.Reserved().Lock()
	defer f.locks.Reserved().Unlock()

	// get pending lock, such that no new read transaction can be started
	f.locks.Pending().Lock()
	defer f.locks.Pending().Unlock()

	// get exclusive lock, waiting for active read transactions to be finished
	f.locks.Exclusive().Lock()
	defer f.locks.Exclusive().Unlock()

	// no other active transactions -> close file
	f.munmap()
	f.writer.Stop()

	errUnlock := f.file.Unlock()
	errClose := f.file.Close()

	// wait for workers to stop
	f.wg.Wait()

	if errUnlock != nil {
		return errUnlock
	}
	return errClose
}

// Readonly returns true if the file has been opened in readonly mode.
func (f *File) Readonly() bool {
	return f.readonly
}

// Begin creates a new read-write transaction. The transaction returned
// does hold the Reserved Lock on the file. Use Close, Rollback, or Commit to
// release the lock.
func (f *File) Begin() (*Tx, error) {
	return f.BeginWith(TxOptions{Readonly: false})
}

// BeginReadonly creates a new readonly transaction. The transaction returned
// does hold the Shared Lock on the file. Use Close() to release the lock.
func (f *File) BeginReadonly() (*Tx, error) {
	return f.BeginWith(TxOptions{Readonly: true})
}

// BeginWith creates a new readonly or read-write transaction, with additional
// transaction settings.
func (f *File) BeginWith(settings TxOptions) (*Tx, error) {
	return f.beginTx(settings)
}

func (f *File) beginTx(settings TxOptions) (*Tx, reason) {
	const op = "txfile/begin-tx"

	if f.readonly && !settings.Readonly {
		msg := "can not start writable transaction on readonly file"
		return nil, f.err(op).of(InvalidOp).report(msg)
	}

	tracef("request new transaction (readonly: %v)\n", settings.Readonly)

	// Acquire transaction log.
	// Unlock on panic, so applications will not be blocked in case they try to
	// defer some close operations on the file.
	ok := false
	lock := f.locks.TxLock(settings.Readonly)
	lock.Lock()
	defer cleanup.IfNot(&ok, lock.Unlock)

	txid := atomic.AddUint64(&f.txids, 1)

	tracef("init new transaction (readonly: %v)\n", settings.Readonly)

	tx := newTx(f, txid, lock, settings)
	tracef("begin transaction: %p (readonly: %v)\n", tx, settings.Readonly)

	tx.onBegin()

	ok = true
	return tx, nil
}

// PageSize returns the files page size in bytes
func (f *File) PageSize() int {
	return int(f.allocator.pageSize)
}

// Offset computes a file offset from PageID and offset within the current
// page.
func (f *File) Offset(id PageID, offset uintptr) uintptr {
	sz := uintptr(f.allocator.pageSize)
	if offset >= sz {
		panic("offset not within page boundary")
	}
	return offset + uintptr(id)*uintptr(f.allocator.pageSize)
}

// SplitOffset splits a file offset into a page ID for accessing the page and
// and offset within the page.
func (f *File) SplitOffset(offset uintptr) (PageID, uintptr) {
	sz := uintptr(f.allocator.pageSize)
	id := PageID(offset / sz)
	off := offset - ((offset / sz) * sz)
	return id, off
}

// truncate updates the file memory-mapping and truncates the file.
// The function ensure the file is not mmapped, so to deal with OSes and
// filesystem not properly truncating mmapped files.
// Due to the memory mapping being updated while truncating, all file locks
// must be held, ensuring no other transaction can read from the file.
func (f *File) truncate(sz int64) reason {
	const op = "txfile/truncate"
	const errMsg = "can not update file size"

	isMMapped := f.mapped != nil

	if isMMapped {
		if err := f.munmap(); err != nil {
			return f.errWrap(op, err).report(errMsg)
		}
	}

	if err := f.file.Truncate(sz); err != nil {
		return f.errWrap(op, err).report(errMsg)
	}

	if isMMapped {
		if err := f.mmap(); err != nil {
			return f.errWrap(op, err).report(errMsg)
		}
	}
	return nil
}

// mmapUpdate updates the mmaped states.
// A go-routine updating the mmaped aread, must hold all locks on the file.
func (f *File) mmapUpdate() (err reason) {
	if err = f.munmap(); err == nil {
		err = f.mmap()
	}
	return
}

// mmap maps the files contents and updates internal pointers into the mmaped memory area.
func (f *File) mmap() reason {
	const op = "txfile/mmap"

	// update real file size
	fileSize, fileErr := f.file.Size()
	if fileErr != nil {
		const msg = "unable to determine file size for mmap region"
		return f.errWrap(op, fileErr).report(msg)
	}
	if fileSize < 0 {
		msg := fmt.Sprintf("file size %v < 0", fileSize)
		return f.err(op).of(InvalidFileSize).report(msg)
	}
	f.size = fileSize
	f.sizeEstimate = fileSize // reset estimate

	maxSize := f.allocator.maxSize
	if em := uint(f.allocator.meta.endMarker); maxSize > 0 && em > f.allocator.maxPages {
		maxSize = em * f.allocator.pageSize
	}
	pageSize := f.allocator.pageSize
	sz, err := computePlatformMmapSize(uint(fileSize), maxSize, uint(pageSize))
	if err != nil {
		return err
	}

	// map file
	buf, fileErr := f.file.MMap(int(sz))
	if err != nil {
		return f.errWrap(op, err).report("can not mmap file")
	}

	f.mapped = buf
	f.meta[0] = castMetaPage(buf[0:])
	f.meta[1] = castMetaPage(buf[pageSize:])

	return nil
}

// munmap unmaps the file and sets internal mapping to nil.
func (f *File) munmap() reason {
	const op = "txfile/munmap"
	err := f.file.MUnmap(f.mapped)
	f.mapped = nil
	if err != nil {
		return f.errWrap(op, err)
	}
	return nil
}

// mmapedPage finds the mmaped page contents by the given pageID.
// The byte buffer can only be used for reading.
func (f *File) mmapedPage(id PageID) []byte {
	pageSize := uint64(f.allocator.pageSize)
	start := uint64(id) * pageSize
	end := start + pageSize
	if uint64(len(f.mapped)) < end {
		return nil
	}

	return f.mapped[start:end]
}

// initNewFile initializes a new, yet empty Files metapages.
func initNewFile(file vfs.File, opts Options) reason {
	const op = "txfile/create"

	var flags uint32
	if opts.MaxSize > 0 && opts.Prealloc {
		flags |= metaFlagPrealloc
		if err := file.Truncate(int64(opts.MaxSize)); err != nil {
			return fileErrWrap(op, file.Name(), err).of(FileCreationFailed).
				report("unable to preallocate file")
		}
	}

	maxSize := opts.MaxSize
	if opts.Flags.check(FlagUnboundMaxSize) {
		maxSize = 0
	}

	pageSize := opts.PageSize
	if opts.PageSize == 0 {
		pageSize = uint32(os.Getpagesize())
		if pageSize < minPageSize {
			pageSize = minPageSize
		}
	}
	if !isPowerOf2(uint64(pageSize)) {
		cause := raiseInvalidParamf("pageSize %v is not power of 2", pageSize)
		return fileErrWrap(op, file.Name(), cause).of(FileCreationFailed)
	}
	if pageSize < minPageSize {
		cause := raiseInvalidParamf("pageSize must be >= %v", minPageSize)
		return fileErrWrap(op, file.Name(), cause).of(FileCreationFailed)
	}

	// create buffer to hold contents for the initial pages:
	// 1. meta page 0
	// 2. meta page 1
	// 3. free list page (only of opts.InitMetaArea > 0)
	buf := make([]byte, pageSize*3)

	// create freelist with meta area only and pre-compute page IDs
	requiredPages := 2
	metaTotal := opts.InitMetaArea
	dataEndMarker := PageID(2)
	metaEndMarker := PageID(0) // no meta area
	freelistPage := PageID(0)
	if metaTotal > 0 {
		requiredPages = 3
		freelistPage = PageID(2)

		// move pages from data area to new meta area by updating markers
		dataEndMarker += PageID(metaTotal)
		metaEndMarker = dataEndMarker

		// write freelist, so to make meta page allocatable
		hdr, body := castFreePage(buf[int(pageSize)*2:])
		hdr.next.Set(0)
		if metaTotal > 1 {
			hdr.count.Set(1)
			encodeRegion(body, true, region{
				id:    freelistPage + 1,
				count: metaTotal - 1,
			})
		}
	}

	// create meta pages
	for i := 0; i < 2; i++ {
		pg := castMetaPage(buf[int(pageSize)*i:])
		pg.Init(flags, pageSize, maxSize)
		pg.txid.Set(uint64(1 - i))
		pg.dataEndMarker.Set(dataEndMarker) // endMarker is index of next to be allocated page at end of file
		pg.metaEndMarker.Set(metaEndMarker)
		pg.metaTotal.Set(uint64(metaTotal))
		pg.freelist.Set(freelistPage)
		pg.Finalize()
	}

	// write initial pages to disk
	err := writeAt(op, file, buf[:int(pageSize)*requiredPages], 0)
	if err == nil {
		if syncErr := file.Sync(vfs.SyncAll); syncErr != nil {
			err = fileErrWrap(op, file.Name(), syncErr)
		}
	}

	if err != nil {
		return fileErrWrap(op, file.Name(), err).of(FileCreationFailed).
			report("io error while initializing data file")
	}
	return nil
}

// readValidMeta tries to read a valid meta page from the file.
// The first valid meta page encountered is returned.
func readValidMeta(f vfs.File) (metaPage, int, reason) {
	var pages [2]metaPage
	var metaErr [2]reason
	var metaActive int
	var err reason

	pages[0], err = readMeta(f, 0)
	if err != nil {
		return metaPage{}, -1, err
	}

	pages[1], err = readMeta(f, int64(pages[0].pageSize.Get()))
	if err != nil {
		return metaPage{}, -1, err
	}

	metaErr[0] = pages[0].Validate()
	metaErr[1] = pages[1].Validate()
	switch {
	case metaErr[0] != nil && metaErr[1] != nil:
		return metaPage{}, -1, metaErr[0]
	case metaErr[0] == nil && metaErr[1] != nil:
		metaActive = 0
	case metaErr[0] != nil && metaErr[1] == nil:
		metaActive = 1
	default:
		// both meta pages valid, choose page with highest transaction number
		tx0 := pages[0].txid.Get()
		tx1 := pages[1].txid.Get()
		if tx0 == tx1 {
			panic("meta pages with same transaction id")
		}

		if int64(tx0-tx1) > 0 { // if tx0 > tx1
			metaActive = 0
		} else {
			metaActive = 1
		}
	}

	return pages[metaActive], metaActive, nil
}

func readMeta(f vfs.File, off int64) (metaPage, reason) {
	const op = "txfile/read-file-meta"

	var buf [unsafe.Sizeof(metaPage{})]byte
	_, err := f.ReadAt(buf[:], off)
	if err != nil {
		reason := fileErrWrap(op, f.Name(), err)
		reason.ctx.SetOffset(off)
		return metaPage{}, reason.report("failed to read file header page")
	}
	return *castMetaPage(buf[:]), nil
}

// computeMmapSize determines the page count in multiple of pages.
// Up to 1GB, the mmaped file area is double (starting at 64KB) on every grows.
// That is, exponential grows with values of 64KB, 128KB, 512KB, 1024KB, and so on.
// Once 1GB is reached, the mmaped area is always a multiple of 1GB.
func computeMmapSize(minSize, maxSize, pageSize uint) (uint, reason) {
	if maxSize != 0 {
		// return maxSize as multiple of pages. Round downwards in case maxSize
		// is not multiple of pages

		if minSize > maxSize {
			maxSize = minSize
		}

		sz := ((maxSize + pageSize - 1) / pageSize) * pageSize
		if sz < initSize {
			return 0, raiseInvalidParamf("max size of %v bytes is too small", maxSize)
		}

		return sz, nil
	}

	if minSize < doubleLimit {
		// grow by next power of 2, starting at 64KB
		initBits := uint(16) // 64KB min
		power2Bits := uint(64 - bits.LeadingZeros64(uint64(minSize)))
		if power2Bits < initBits {
			power2Bits = initBits
		}
		return 1 << power2Bits, nil
	}

	// allocate number of 1GB blocks to fulfill minSize
	sz := ((minSize + (sz1GB - 1)) / sz1GB) * sz1GB
	if sz > maxMmapSize {
		return 0, raiseInvalidParamf("mmap size of %v bytes is too large", sz)
	}

	// ensure we have a multiple of pageSize
	sz = ((sz + pageSize - 1) / pageSize) * pageSize

	return sz, nil
}

// getMetaPage returns a pointer to the meta-page of the last valid transaction
// found.
func (f *File) getMetaPage() *metaPage {
	return f.meta[f.metaActive]
}

// growFile executes a write transaction, growing the files max size setting.
// If opts.Preallocate is set, the file will be truncated to the new file size on success.
func growFile(f *File, opts Options) reason {
	const op = "txfile/grow"

	err := doGrowFile(f, opts)
	if err != nil {
		return fileErrWrap(op, f.path, err).report("failed to increase file size")
	}
	return nil
}

func doGrowFile(f *File, opts Options) reason {
	maxPages, maxSize, err := initTxMaxSize(f, opts.MaxSize)
	if err != nil {
		return err
	}

	// Transaction completed. Update file allocator limits
	f.allocator.maxPages = maxPages
	f.allocator.maxSize = maxSize

	// Allocate space on disk if prealloc is enabled and new file size is bounded.
	if opts.Prealloc && maxSize > 0 {
		if err := f.truncate(int64(maxSize)); err != nil {
			return err
		}
		if err := f.mmapUpdate(); err != nil {
			return wrapErr(err)
		}
	}

	return nil
}

// shrinkFile reconfigures the new max size to a smaller value and tries to
// remove excessive pages from the free list.
// The removal of excessive pages is done in a second transaction, that is
// allowed to fail. Excessive pages are freed in future transactions anyways.
// The file is not truncated yet, as the last 2 transaction must agree on the
// actual file size before truncating. Truncation is postponed to later transactions.
func shrinkFile(f *File, opts Options) reason {
	const op = "txfile/shrink"

	// 1. Start transaction updating the file meta header only. This transaction must succeed.
	maxPages, maxSize, err := initTxMaxSize(f, opts.MaxSize)
	if err != nil {
		return fileErrWrap(op, f.path, err).
			report("failed to reduce the maximum file size")
	}

	// 2. Transaction completed. Update file allocator limits
	f.allocator.maxPages = maxPages
	f.allocator.maxSize = maxSize

	canReleaseRegions := func(area *allocArea) bool {
		end := area.freelist.LastRegion().End()
		return end == area.endMarker && uint(area.endMarker) > maxPages
	}

	// 3. Start a new transaction, trying to remove pages from the freelist
	data := &f.allocator.data
	meta := &f.allocator.meta
	if canReleaseRegions(data) || canReleaseRegions(meta) {
		initTxReleaseRegions(f)
	}

	return nil
}

// initTxMaxSize runs a write transaction, updating the file maxSize
// to the newMaxSize value.
// initTxMaxSize is used when opening an existing file.
// As the file size must be a multiple of the file's page size, the number of
// maximum pages and the actual max file size is returned on success.
func initTxMaxSize(
	f *File,
	newMaxSize uint64,
) (maxPages, maxSize uint, err reason) {
	const op = "txfile/tx-update-maxsize"

	var metaID int
	err = withInitTx(f, func(tx *Tx) reason {
		// create new meta header for new ongoing write transaction
		newMetaBuf := tx.prepareMetaBuffer()
		newMeta := newMetaBuf.cast()

		// update max size
		pageSize := uint(newMeta.pageSize.Get())
		maxSize = uint(newMaxSize)
		maxPages = maxSize / pageSize
		maxSize = maxPages * pageSize // round new max size to multiple of page size
		newMeta.maxSize.Set(uint64(maxSize))

		// sync new transaction state to disk
		metaID = tx.syncNewMeta(&newMetaBuf)
		err := tx.writeSync.Wait()
		if err != nil {
			return f.errWrap(op, err).of(TxFailed).
				report("failed to update the on disk max size header entry")
		}

		return nil
	})

	if err == nil {
		f.metaActive = metaID
	}
	return
}

// initTxReleaseRegions attempts to remove pages from the freelist, that exceed
// the files max size. The transaction is not required to succeed. If it fails,
// we just rollback. Subsequent write transactions will try to continue
// retuning pages to the file systems.
// The transaction is prone to fail if there is not enough space to serialize
// the new freelist to.
// initTxReleaseRegions should only be called if it's clear pages can be
// removed. Otherwise an 'empty' transaction
func initTxReleaseRegions(f *File) {
	withInitTx(f, func(tx *Tx) reason {
		// Init new allocator commit state, returning current meta pages into the
		// freelist.
		var csAlloc allocCommitState
		tx.file.allocator.fileCommitPrepare(&csAlloc, &tx.alloc, true)

		// Compute new free lists and remove page ids > max file size from the end
		// of the freelist. Pages to be freed must border on the files allocation
		// end markers.
		if err := tx.file.allocator.fileCommitAlloc(&csAlloc); err != nil {
			return err
		}

		// Serialize new freelist to disk.
		if err := tx.file.allocator.fileCommitSerialize(&csAlloc, tx.scheduleWrite); err != nil {
			return err
		}
		tx.file.writer.Sync(tx.writeSync, syncDataOnly)

		// Update file meta header.
		newMetaBuf := tx.prepareMetaBuffer()
		newMeta := newMetaBuf.cast()
		tx.file.allocator.fileCommitMeta(newMeta, &csAlloc)
		metaID := tx.syncNewMeta(&newMetaBuf)

		// Finalize on-disk transaction.
		if err := tx.writeSync.Wait(); err != nil {
			return wrapErr(err)
		}

		// Commit allocator changes to in-memory allocator.
		tx.file.allocator.Commit(&csAlloc)

		// Switch the files active meta page to meta page being written.
		tx.file.metaActive = metaID

		return nil
	})
}

func withInitTx(f *File, fn func(tx *Tx) reason) reason {
	tx, err := f.beginTx(TxOptions{Readonly: false})
	if err != nil {
		return err.(reason)
	}

	defer tx.close()

	commitOK := false
	defer cleanup.IfNot(&commitOK, tx.rollbackChanges)

	// use write transactions commit locks. As file if being generated, the
	// locks are not really required, yet. But better execute a correct transaction
	// sequence, here.
	pending, exclusive := tx.file.locks.Pending(), tx.file.locks.Exclusive()

	pending.Lock()
	defer pending.Lock()

	exclusive.Lock()
	defer exclusive.Lock()

	// On function exit wait on writer to finish outstanding operations, in case
	// we have to return early on error. On success, this is basically a no-op.
	defer tx.writeSync.Wait()

	err = fn(tx)
	commitOK = err == nil
	return err
}

func (f *File) err(op string) *Error {
	return fileErr(op, f.path)
}

func (f *File) errWrap(op string, cause error) *Error {
	return fileErrWrap(op, f.path, cause)
}

func (f *File) errCtx() errorCtx { return fileErrCtx(f.path) }

func fileErr(op, path string) *Error {
	return &Error{op: op, ctx: fileErrCtx(path)}
}

func fileErrWrap(op, path string, cause error) *Error {
	return fileErr(op, path).causedBy(cause)
}

func fileErrCtx(path string) errorCtx {
	return errorCtx{file: path}
}
