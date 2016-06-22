package processor

import (
	"io"
	"time"

	"github.com/elastic/beats/filebeat/harvester/encoding"
	"github.com/elastic/beats/filebeat/harvester/reader"
)

// LineEncoder produces lines by reading lines from an io.Reader
// through a decoder converting the reader it's encoding to utf-8.
type LineEncoder struct {
	reader *reader.Line
}

// NewLineEncoder creates a new LineEncoder from input reader by applying
// the given codec.
func NewLineEncoder(
	in io.Reader,
	codec encoding.Encoding,
	bufferSize int,
) (LineEncoder, error) {
	r, err := reader.NewLine(in, codec, bufferSize)
	return LineEncoder{r}, err
}

// Next reads the next line from it's initial io.Reader
func (p LineEncoder) Next() (Line, error) {
	c, sz, err := p.reader.Next()
	return Line{Ts: time.Now(), Content: c, Bytes: sz}, err
}
