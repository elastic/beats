package source

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
	HasState() bool    // does this source have a state?
}
