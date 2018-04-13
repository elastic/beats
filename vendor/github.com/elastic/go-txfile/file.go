package txfile

import (
	"fmt"
	"math"
	"math/bits"
	"os"
	"sync"
	"unsafe"

	"github.com/elastic/go-txfile/internal/cleanup"
	"github.com/elastic/go-txfile/internal/invariant"
)

// File provides transactional support to pages of a file. A file is split into
// pages of type PageSize. Pages within the file are only accessible by page IDs
// from within active transactions.
type File struct {
	path      string
	file      vfsFile
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
}

// Options provides common file options used when opening or creating a file.
type Options struct {
	// MaxSize sets the maximum file size in bytes. This should be a multiple of PageSize.
	// If it's not a multiple of PageSize, the actual files maximum size is rounded downwards
	// to the next multiple of PageSize.
	// A value of 0 indicates the file can grow without limits.
	MaxSize uint64

	// PageSize sets the files page size on file creation. PageSize is ignored if
	// the file already exists.
	// If PageSize is not configured, the OSes main memory page size is selected.
	PageSize uint32

	// Prealloc disk space if MaxSize is set.
	Prealloc bool

	// Open file in readonly mode.
	Readonly bool
}

// Open opens or creates a new transactional file.
// Open tries to create the file, if the file does not exist yet.  Returns an
// error if file access fails, file can not be locked or file meta pages are
// found to be invalid.
func Open(path string, mode os.FileMode, opts Options) (*File, error) {
	file, err := openOSFile(path, mode)
	if err != nil {
		return nil, err
	}

	initOK := false
	defer cleanup.IfNot(&initOK, cleanup.IgnoreError(file.Close))

	// Create exclusive lock on the file and initialize the file state.
	var f *File
	if err = file.Lock(true, true); err == nil {
		// initialize the file
		f, err = openWith(file, opts)
	}
	if err != nil {
		return nil, err
	}

	initOK = true

	tracef("open file: %p (%v)\n", f, path)
	traceMetaPage(f.getMetaPage())
	return f, nil
}

// openWith implements the actual opening sequence, including file
// initialization and validation.
func openWith(file vfsFile, opts Options) (*File, error) {
	sz, err := file.Size()
	if err != nil {
		return nil, err
	}

	fileExists := sz > 0
	if !fileExists {
		if err := initNewFile(file, opts); err != nil {
			return nil, err
		}
	}

	meta, err := readValidMeta(file)
	if err != nil {
		return nil, err
	}

	pageSize := meta.pageSize.Get()
	maxSize := meta.maxSize.Get()
	if maxSize == 0 && opts.MaxSize > 0 {
		maxSize = opts.MaxSize
	}

	if maxSize > uint64(maxUint) {
		return nil, errFileSizeTooLage
	}

	return newFile(file, opts, uint(maxSize), uint(pageSize))
}

// newFile creates and initializes a new File. File state is initialized
// from file and internal workers will be started.
func newFile(file vfsFile, opts Options, maxSize, pageSize uint) (*File, error) {

	f := &File{
		file: file,
		path: file.Name(),
		allocator: allocator{
			maxSize:  maxSize,
			pageSize: pageSize,
		},
	}
	f.locks.init()

	if err := f.mmap(); err != nil {
		return nil, err
	}
	initOK := false
	defer cleanup.IfNot(&initOK, cleanup.IgnoreError(f.munmap))

	if err := f.init(opts); err != nil {
		return nil, err
	}

	invariant.CheckNot(f.allocator.maxSize != 0 && f.allocator.maxPages == 0,
		"page limit not configured on allocator")

	// create asynchronous writer
	f.writer.Init(file, f.allocator.pageSize)
	f.wg.Add(1)
	go func() {
		defer f.wg.Done()
		f.writer.Run()
	}()

	initOK = true
	return f, nil
}

// init initializes the File state from most recent valid meta-page.
func (f *File) init(opts Options) error {
	// validate meta pages and set active meta page id
	var metaErr [2]error
	metaErr[0] = f.meta[0].Validate()
	metaErr[1] = f.meta[1].Validate()
	switch {
	case metaErr[0] != nil && metaErr[1] != nil:
		return metaErr[0]
	case metaErr[0] == nil && metaErr[1] != nil:
		f.metaActive = 1
	case metaErr[0] != nil && metaErr[1] == nil:
		f.metaActive = 1
	default:
		// both meta pages valid, choose page with highest transaction number
		tx0 := f.meta[0].txid.Get()
		tx1 := f.meta[1].txid.Get()
		if tx0 == tx1 {
			panic("meta pages with same transaction id")
		}

		if int64(tx0-tx1) > 0 { // if tx0 > tx1
			f.metaActive = 0
		} else {
			f.metaActive = 1
		}
	}

	// reference active meta page for initializing internal structures
	meta := f.meta[f.metaActive]

	if err := readWALMapping(&f.wal, f.mmapedPage, meta.wal.Get()); err != nil {
		return err
	}

	return readAllocatorState(&f.allocator, f, meta, opts)
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

	err := f.file.Close()

	// wait for workers to stop
	f.wg.Wait()

	return err
}

// Begin creates a new read-write transaction. The transaction returned
// does hold the Reserved Lock on the file. Use Close, Rollback, or Commit to
// release the lock.
func (f *File) Begin() *Tx {
	return f.BeginWith(TxOptions{Readonly: false})
}

// BeginReadonly creates a new readonly transaction. The transaction returned
// does hold the Shared Lock on the file. Use Close() to release the lock.
func (f *File) BeginReadonly() *Tx {
	return f.BeginWith(TxOptions{Readonly: true})
}

// BeginWith creates a new readonly or read-write transaction, with additional
// transaction settings.
func (f *File) BeginWith(settings TxOptions) *Tx {
	tracef("request new transaction (readonly: %v)\n", settings.Readonly)
	lock := f.locks.TxLock(settings.Readonly)
	lock.Lock()
	tracef("init new transaction (readonly: %v)\n", settings.Readonly)
	tx := newTx(f, lock, settings)
	tracef("begin transaction: %p (readonly: %v)\n", tx, settings.Readonly)
	return tx
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

// mmapUpdate updates the mmaped states.
// A go-routine updating the mmaped aread, must hold all locks on the file.
func (f *File) mmapUpdate() (err error) {
	if err = f.munmap(); err == nil {
		err = f.mmap()
	}
	return
}

// mmap maps the files contents and updates internal pointers into the mmaped memory area.
func (f *File) mmap() error {
	fileSize, err := f.file.Size()
	if err != nil {
		return err
	}

	if fileSize < 0 {
		return errInvalidFileSize
	}

	maxSize := f.allocator.maxSize
	if em := uint(f.allocator.meta.endMarker); maxSize > 0 && em > f.allocator.maxPages {
		maxSize = em * f.allocator.pageSize
	}
	pageSize := f.allocator.pageSize
	sz, err := computeMmapSize(uint(fileSize), maxSize, uint(pageSize))
	if err != nil {
		return err
	}

	// map file
	buf, err := f.file.MMap(int(sz))
	if err != nil {
		return err
	}

	f.mapped = buf
	f.meta[0] = castMetaPage(buf[0:])
	f.meta[1] = castMetaPage(buf[pageSize:])

	return nil
}

// munmap unmaps the file and sets internal mapping to nil.
func (f *File) munmap() error {
	err := f.file.MUnmap(f.mapped)
	f.mapped = nil
	return err
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
func initNewFile(file vfsFile, opts Options) error {
	var flags uint32
	if opts.MaxSize > 0 && opts.Prealloc {
		flags |= metaFlagPrealloc
		if err := file.Truncate(int64(opts.MaxSize)); err != nil {
			return fmt.Errorf("truncation failed with %v", err)
		}
	}

	pageSize := opts.PageSize
	if opts.PageSize == 0 {
		pageSize = uint32(os.Getpagesize())
		if pageSize < minPageSize {
			pageSize = minPageSize
		}
	}
	if !isPowerOf2(uint64(pageSize)) {
		return fmt.Errorf("pageSize %v is no power of 2", pageSize)
	}
	if pageSize < minPageSize {
		return fmt.Errorf("pageSize must be > %v", minPageSize)
	}

	// create buffer to hold contents for the four initial pages:
	// 1. meta page 0
	// 2. meta page 1
	// 3. free list page
	buf := make([]byte, pageSize*3)

	// write meta pages
	for i := 0; i < 2; i++ {
		pg := castMetaPage(buf[int(pageSize)*i:])
		pg.Init(flags, pageSize, opts.MaxSize)
		pg.txid.Set(uint64(1 - i))
		pg.dataEndMarker.Set(2) // endMarker is index of next to be allocated page at end of file
		pg.Finalize()
	}

	// write initial pages to disk
	err := writeAt(file, buf, 0)
	if err == nil {
		err = file.Sync()
	}

	if err != nil {
		return fmt.Errorf("initializing data file failed with %v", err)
	}
	return nil
}

// readValidMeta tries to read a valid meta page from the file.
// The first valid meta page encountered is returned.
func readValidMeta(f vfsFile) (metaPage, error) {
	meta, err := readMeta(f, 0)
	if err != nil {
		return meta, err
	}

	if err := meta.Validate(); err != nil {
		// try next metapage
		offset := meta.pageSize.Get()
		if meta, err = readMeta(f, int64(offset)); err != nil {
			return meta, err
		}
		return meta, meta.Validate()
	}
	return meta, nil
}

func readMeta(f vfsFile, off int64) (metaPage, error) {
	var buf [unsafe.Sizeof(metaPage{})]byte
	_, err := f.ReadAt(buf[:], off)
	return *castMetaPage(buf[:]), err
}

// computeMmapSize determines the page count in multiple of pages.
// Up to 1GB, the mmaped file area is double (starting at 64KB) on every grows.
// That is, exponential grows with values of 64KB, 128KB, 512KB, 1024KB, and so on.
// Once 1GB is reached, the mmaped area is always a multiple of 1GB.
func computeMmapSize(minSize, maxSize, pageSize uint) (uint, error) {
	const (
		initBits    uint = 16            // 2 ^ 16 Bytes
		initSize         = 1 << initBits // 64KB
		sz1GB            = 1 << 30
		doubleLimit      = sz1GB // upper limit when to stop doubling the mmaped area
	)

	var maxMapSize uint
	if math.MaxUint32 == maxUint {
		maxMapSize = 2 * sz1GB
	} else {
		tmp := uint64(0x1FFFFFFFFFFF)
		maxMapSize = uint(tmp)
	}

	if maxSize != 0 {
		// return maxSize as multiple of pages. Round downwards in case maxSize
		// is not multiple of pages

		if minSize > maxSize {
			maxSize = minSize
		}

		sz := ((maxSize + pageSize - 1) / pageSize) * pageSize
		if sz < initSize {
			return 0, fmt.Errorf("max size of %v bytes is too small", maxSize)
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
	if sz > maxMapSize {
		return 0, errMmapTooLarge
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
