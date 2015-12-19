package processor

import (
	"io"
	"time"

	"github.com/elastic/beats/filebeat/harvester/encoding"
)

// Line represents a line event with timestamp, content and actual number
// of bytes read from input before decoding.
type Line struct {
	Ts      time.Time // timestamp the line was read
	Content []byte    // actual line read
	Bytes   int       // total number of bytes read to generate the line
}

// LineProcessor is the interface that wraps the basic Next method for
// getting a new line.
// Next returns the line being read or and error. EOF is returned
// if processor will not return any new lines on subsequent calls.
type LineProcessor interface {
	Next() (Line, error)
}

// LineSource produces lines by reading lines from an io.Reader
// through a decoder converting the reader it's encoding to utf-8.
type LineSource struct {
	reader *encoding.LineReader
}

// StripNewline processor removes the last trailing newline characters from
// read lines.
type StripNewline struct {
	reader LineProcessor
}

// LimitProcessor sets an upper limited on line length. Lines longer
// then the max configured line length will be snapped short.
type LimitProcessor struct {
	reader   LineProcessor
	maxBytes int
}

// NewLineSource creates a new LineSource from input reader by applying
// the given codec.
func NewLineSource(
	in io.Reader,
	codec encoding.Encoding,
	bufferSize int,
) (LineSource, error) {
	r, err := encoding.NewLineReader(in, codec, bufferSize)
	return LineSource{r}, err
}

// Next reads the next line from it's initial io.Reader
func (p LineSource) Next() (Line, error) {
	c, sz, err := p.reader.Next()
	return Line{Ts: time.Now(), Content: c, Bytes: sz}, err
}

// NewStripNewline creates a new line reader stripping the last tailing newline.
func NewStripNewline(r LineProcessor) *StripNewline {
	return &StripNewline{r}
}

// Next returns the next line.
func (p *StripNewline) Next() (Line, error) {
	line, err := p.reader.Next()
	if err != nil {
		return line, err
	}

	L := line.Content
	line.Content = L[:len(L)-lineEndingChars(L)]
	return line, err
}

// NewLimitProcessor creates a new processor limiting the line length.
func NewLimitProcessor(in LineProcessor, maxBytes int) *LimitProcessor {
	return &LimitProcessor{reader: in, maxBytes: maxBytes}
}

// Next returns the next line.
func (p *LimitProcessor) Next() (Line, error) {
	line, err := p.reader.Next()
	if len(line.Content) > p.maxBytes {
		line.Content = line.Content[:p.maxBytes]
	}
	return line, err
}

// isLine checks if the given byte array is a line, means has a line ending \n
func isLine(l []byte) bool {
	return l != nil && len(l) > 0 && l[len(l)-1] == '\n'
}

// lineEndingChars returns the number of line ending chars the given by array has
// In case of Unix/Linux files, it is -1, in case of Windows mostly -2
func lineEndingChars(l []byte) int {
	if !isLine(l) {
		return 0
	}

	if len(l) > 1 && l[len(l)-2] == '\r' {
		return 2
	}
	return 1
}
