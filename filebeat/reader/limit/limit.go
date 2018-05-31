package limit

import (
	"github.com/elastic/beats/filebeat/reader"
)

// Reader sets an upper limited on line length. Lines longer
// then the max configured line length will be snapped short.
type Reader struct {
	reader   reader.Reader
	maxBytes int
}

// New creates a new reader limiting the line length.
func New(r reader.Reader, maxBytes int) *Reader {
	return &Reader{reader: r, maxBytes: maxBytes}
}

// Next returns the next line.
func (r *Reader) Next() (reader.Message, error) {
	message, err := r.reader.Next()
	if len(message.Content) > r.maxBytes {
		message.Content = message.Content[:r.maxBytes]
	}
	return message, err
}
