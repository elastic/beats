package badio

import (
	"io"
)

type byteReader struct {
	b byte
}

// NewByteReader returns a Reader that implements Read by returning an infinite
// stream of the given byte.
func NewByteReader(b byte) io.Reader {
	return &byteReader{b}
}

// NewNullReader returns a Reader that implements Read by returning an infinite
// stream of zeros, analogous to `cat /dev/zero`.
func NewNullReader() io.Reader {
	return &byteReader{0x00}
}

func (c *byteReader) Read(p []byte) (int, error) {
	var i int
	for ; i < len(p); i++ {
		p[i] = c.b
	}

	return i, nil
}
