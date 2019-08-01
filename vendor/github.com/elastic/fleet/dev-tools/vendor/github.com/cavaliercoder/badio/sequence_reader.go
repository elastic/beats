package badio

import (
	"io"
)

type sequenceReader struct {
	b []byte
	o int64
}

// NewSequenceReader returns a Reader that implements Read by perpetually
// repeating the given byte sequence.
func NewSequenceReader(b []byte) io.Reader {
	return &sequenceReader{b: b, o: 0}
}

func (c *sequenceReader) Read(p []byte) (int, error) {
	if len(c.b) == 0 {
		return 0, newError("invalid sequence length")
	}

	n := 0
	for n < len(p) {
		// copy one sequence
		for ; c.o < int64(len(c.b)) && n < len(p); c.o++ {
			p[n] = c.b[c.o]
			n++
		}

		// reset offset if there's more to copy
		if n < len(p) {
			c.o = 0
		}
	}

	return n, nil
}
