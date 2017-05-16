package harvester

import (
	"io"
	"os"
)

type Source interface {
	io.ReadCloser
	Name() string
	Stat() (os.FileInfo, error)
	Continuable() bool // can we continue processing after EOF?
	HasState() bool    // does this source have a state?
}
