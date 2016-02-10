package harvester

import (
	"io"
	"os"
)

type LogSource interface {
	io.ReadCloser
	Name() string
}

type FileSource interface {
	LogSource
	Stat() (os.FileInfo, error)
	Continuable() bool // can we continue processing after EOF?
}

// restrict file to minimal interface of FileSource to prevent possible casts
// to additional interfaces supported by underlying file
type pipeSource struct{ file *os.File }

func (p pipeSource) Read(b []byte) (int, error) { return p.file.Read(b) }
func (p pipeSource) Close() error               { return p.file.Close() }
func (p pipeSource) Name() string               { return p.file.Name() }
func (p pipeSource) Stat() (os.FileInfo, error) { return p.file.Stat() }
func (p pipeSource) Continuable() bool          { return false }

type fileSource struct{ *os.File }

func (fileSource) Continuable() bool { return true }
