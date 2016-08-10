package source

import (
    "os"
    "errors"
)

// restrict file to minimal interface of FileSource to prevent possible casts
// to additional interfaces supported by underlying file
type Pipe struct {
    File *os.File
}

func (p Pipe) Read(b []byte) (int, error) { return p.File.Read(b) }
func (p Pipe) Close() error               { return p.File.Close() }
func (p Pipe) Name() string               { return p.File.Name() }
func (p Pipe) Stat() (os.FileInfo, error) { return p.File.Stat() }
func (p Pipe) Continuable() bool          { return false }
func (p Pipe) Seek(offset int64, whence int) (int64, error) {
    if offset == 0 && (whence == os.SEEK_CUR || whence == os.SEEK_SET) {
        return 0, nil
    } else {
        return 0, errors.New("Seek not supported on pipes")
    }
}
