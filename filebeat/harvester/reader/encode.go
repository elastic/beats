package reader

import (
	"io"
	"time"

	"github.com/elastic/beats/filebeat/harvester/encoding"
)

// Encode reader produces lines by reading lines from an io.Reader
// through a decoder converting the reader it's encoding to utf-8.
type Encode struct {
	reader *Line
}

// NewEncode creates a new Encode reader from input reader by applying
// the given codec.
func NewEncode(
	reader io.Reader,
	codec encoding.Encoding,
	bufferSize int,
) (Encode, error) {
	r, err := NewLine(reader, codec, bufferSize)
	return Encode{r}, err
}

// Next reads the next line from it's initial io.Reader
// This converts a io.Reader to a reader.reader
func (p Encode) Next() (Message, error) {
	c, sz, err := p.reader.Next()
	// Creating message object
	return Message{
		Ts:      time.Now(),
		Content: c,
		Bytes:   sz,
	}, err
}
