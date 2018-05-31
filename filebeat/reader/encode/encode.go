package encode

import (
	"io"
	"time"

	"github.com/elastic/beats/filebeat/reader"
	"github.com/elastic/beats/filebeat/reader/encode/encoding"
	"github.com/elastic/beats/filebeat/reader/line"
)

// Reader produces lines by reading lines from an io.Reader
// through a decoder converting the reader it's encoding to utf-8.
type Reader struct {
	reader *line.Reader
}

// New creates a new Encode reader from input reader by applying
// the given codec.
func New(
	r io.Reader,
	codec encoding.Encoding,
	bufferSize int,
) (Reader, error) {
	eReader, err := line.New(r, codec, bufferSize)
	return Reader{eReader}, err
}

// Next reads the next line from it's initial io.Reader
// This converts a io.Reader to a reader.reader
func (r Reader) Next() (reader.Message, error) {
	c, sz, err := r.reader.Next()
	// Creating message object
	return reader.Message{
		Ts:      time.Now(),
		Content: c,
		Bytes:   sz,
	}, err
}
