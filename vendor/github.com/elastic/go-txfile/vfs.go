package txfile

import (
	"io"
	"os"
)

type vfsFile interface {
	io.Closer
	io.WriterAt
	io.ReaderAt

	Name() string
	Size() (int64, error)
	Sync() error
	Truncate(int64) error

	Lock(exclusive, blocking bool) error
	Unlock() error

	MMap(sz int) ([]byte, error)
	MUnmap([]byte) error
}

type osFile struct {
	*os.File
	state osFileState
}

func openOSFile(path string, mode os.FileMode) (*osFile, error) {
	flags := os.O_RDWR | os.O_CREATE
	f, err := os.OpenFile(path, flags, mode)
	return &osFile{File: f}, err
}

func (o *osFile) Size() (int64, error) {
	stat, err := o.File.Stat()
	if err != nil {
		return -1, err
	}
	return stat.Size(), nil
}
