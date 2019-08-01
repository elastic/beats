package badio

import (
	"crypto/rand"
	"io"
)

type randomReader struct{}

// NewRandomReader returns a Reader that implements Read by perpetually
// reading cryptographically secure pseudorandom numbers.
func NewRandomReader() io.Reader {
	return &randomReader{}
}

func (c *randomReader) Read(p []byte) (int, error) {
	return rand.Read(p)
}
