package line

import (
	"io"

	"golang.org/x/text/encoding"
	"golang.org/x/text/transform"

	"github.com/elastic/beats/libbeat/common/streambuf"
)

// lineReader reads lines from underlying reader, decoding the input stream
// using the configured codec. The reader keeps track of bytes consumed
// from raw input stream for every decoded line.
type Reader struct {
	lineScanner *lineScanner
}

// New creates a new reader object
func New(input io.Reader, codec encoding.Encoding, bufferSize int) (*Reader, error) {
	decReader := newDecoderReader(input, codec)
	lineScanner := newLineScanner(decReader, bufferSize)

	return &Reader{
		lineScanner: lineScanner,
	}, nil
}

// Next reads the next line until the new line character
func (r *Reader) Next() ([]byte, int, error) {
	return r.lineScanner.scan()
}

type decoderReader struct {
	in      io.Reader
	decoder transform.Transformer
}

func newDecoderReader(in io.Reader, codec encoding.Encoding) *decoderReader {
	decoder := codec.NewDecoder()
	decReader := decoder.Reader(in)

	return &decoderReader{
		in: decReader,
	}
}

func (r *decoderReader) read(buf []byte) (int, error) {
	return r.in.Read(buf)
}

type lineScanner struct {
	in         *decoderReader
	bufferSize int

	buf    *streambuf.Buffer
	offset int
}

func newLineScanner(in *decoderReader, bufferSize int) *lineScanner {
	return &lineScanner{
		in:         in,
		bufferSize: bufferSize,
		buf:        streambuf.New(nil),
		offset:     0,
	}
}

// Scan reads from the underlying decoder reader and returns decoded lines.
func (s *lineScanner) scan() ([]byte, int, error) {
	idx := s.buf.IndexRune('\n')
	for !newLineFound(idx) {
		b := make([]byte, s.bufferSize)
		n, err := s.in.read(b)
		if err != nil {
			return nil, 0, err
		}

		s.buf.Append(b[:n])
		idx = s.buf.IndexRune('\n')
	}

	return s.line(idx)
}

// newLineFound checks if a new line was found.
func newLineFound(i int) bool {
	return i != -1
}

// line sets the offset of the scanner and returns a line.
func (s *lineScanner) line(i int) ([]byte, int, error) {
	line, err := s.buf.CollectUntilRune('\n')
	if err != nil {
		panic(err)
	}

	s.offset += i
	s.buf.Reset()
	return line, len(line), nil
}
