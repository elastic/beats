package source

import (
	"compress/gzip"
	"io"
	"os"
)

// restrict file to minimal interface of FileSource to prevent possible casts
// to additional interfaces supported by underlying file
type Gzip struct {
	*os.File
	gzipReader *gzip.Reader
}

func newGZipFile(osf *os.File) (FileSource, error) {
	gzipReader, err := gzip.NewReader(osf)
	if err != nil {
		return nil, err
	} else {
		return Gzip{osf, gzipReader}, nil
	}
}

func (Gzip) Continuable() bool { return false }

func (gf Gzip) Close() error {
	err1 := gf.gzipReader.Close()
	err2 := gf.File.Close()
	if err2 != nil {
		return err2
	} else {
		return err1
	}
}

func (gf Gzip) Read(p []byte) (n int, err error) {
	n, err = gf.gzipReader.Read(p)

	// Gzip should only be ended when n = 0 and EOF
	if err == io.EOF && n > 0 {
		err = nil
	}
	return n, err
}
