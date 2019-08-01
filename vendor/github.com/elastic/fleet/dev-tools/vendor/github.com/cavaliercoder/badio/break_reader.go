package badio

import (
	"io"
)

type breakReader struct {
	r io.Reader
	n int64
	o int64
}

// NewBreakReader returns a Reader that behaves like r except that it will
// return a BadIOError (not io.EOF) once it has read n bytes.
func NewBreakReader(r io.Reader, n int64) io.Reader {
	return &breakReader{r: r, n: n}
}

func (c *breakReader) Read(p []byte) (n int, err error) {
	// reduce read size if it exceeds the breakpoint
	n = len(p)
	if c.o+int64(n) > c.n {
		n = int(c.n - c.o)
	}

	// test if break point exceeded
	if n == 0 {
		return 0, newError("Reader is already broken at offset %d (0x%X)", c.o, c.o)
	}

	// read into buffer
	n, err = c.r.Read(p[0:n])
	if err != nil {
		return
	}

	// break?
	c.o += int64(n)
	if c.o >= c.n {
		return n, newError("Reader break point at offset %d (0x%X)", c.o, c.o)
	}

	return
}
