package txfile

import "errors"

var (
	// file meta page validation errors

	errMagic    = errors.New("invalid magic number")
	errVersion  = errors.New("invalid version number")
	errChecksum = errors.New("checksum mismatch")

	// file sizing errors

	errMmapTooLarge    = errors.New("mmap too large")
	errFileSizeTooLage = errors.New("max file size to large for this system")
	errInvalidFileSize = errors.New("invalid file size")

	// page access/allocation errors

	errOutOfBounds   = errors.New("out of bounds page id")
	errOutOfMemory   = errors.New("out of memory")
	errFreedPage     = errors.New("trying to access an already freed page")
	errPageFlushed   = errors.New("page is already flushed")
	errTooManyBytes  = errors.New("contents exceeds page size")
	errNoPageData    = errors.New("accessing page without contents")
	errFreeDirtyPage = errors.New("freeing dirty page")

	// transaction errors

	errTxFinished = errors.New("transaction has already been closed")
	errTxReadonly = errors.New("readonly transaction")
)
