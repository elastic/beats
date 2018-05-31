package encode

import (
	"io"
	"time"

	"github.com/elastic/beats/filebeat/reader"
	"github.com/elastic/beats/filebeat/reader/encode/encoding"
	"github.com/elastic/beats/filebeat/reader/line"
)

// Encode reader produces lines by reading lines from an io.Reader
// through a decoder converting the reader it's encoding to utf-8.
type Encode struct {
	reader *line.Line
}

// NewEncode creates a new Encode reader from input reader by applying
// the given codec.
func NewEncode(
	r io.Reader,
	codec encoding.Encoding,
	bufferSize int,
) (Encode, error) {
	eReader, err := line.NewLine(r, codec, bufferSize)
	return Encode{eReader}, err
}

// Next reads the next line from it's initial io.Reader
// This converts a io.Reader to a reader.reader
func (p Encode) Next() (reader.Message, error) {
	c, sz, err := p.reader.Next()
	// Creating message object
	return reader.Message{
		Ts:      time.Now(),
		Content: c,
		Bytes:   sz,
	}, err
}
