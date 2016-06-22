package processor

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
)

// Line represents a line event with timestamp, content and actual number
// of bytes read from input before decoding.
type Line struct {
	Ts      time.Time     // timestamp the line was read
	Content []byte        // actual line read
	Bytes   int           // total number of bytes read to generate the line
	Fields  common.MapStr // optional fields that can be added by processors
}

// LineProcessor is the interface that wraps the basic Next method for
// getting a new line.
// Next returns the line being read or and error. EOF is returned
// if processor will not return any new lines on subsequent calls.
type LineProcessor interface {
	Next() (Line, error)
}
