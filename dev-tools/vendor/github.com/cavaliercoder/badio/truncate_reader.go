package badio

import (
	"io"
)

type truncateReader struct {
	r io.Reader
	n int64
	o int64
}

// NewTruncateReader returns a Reader that behaves like r except that it will
// return zero count and an io.EOF error once it has read n bytes.
func NewTruncateReader(r io.Reader, n int64) io.Reader {
	return &truncateReader{r: r, n: n}
}

func (c *truncateReader) Read(p []byte) (n int, err error) {
	// reduce read size if it exceeds the breakpoint
	n = len(p)
	if c.o+int64(n) > c.n {
		n = int(c.n - c.o)
	}

	// test if EOF exceeded
	if n == 0 {
		return 0, io.EOF
	}

	// read into buffer
	n, err = c.r.Read(p[0:n])
	c.o += int64(n)

	return
}
