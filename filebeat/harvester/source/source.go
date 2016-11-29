package source

import (
	"io"
	"os"

	"github.com/elastic/beats/filebeat/config"
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

func NewFile(file *os.File, compression string) (FileSource, error) {

	if config.GZipCompression == compression {
		return newGZipFile(file)
	} else {
		return File{file}, nil
	}
}
