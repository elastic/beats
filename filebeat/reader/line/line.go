package line

import (
	"io"

	"golang.org/x/text/encoding"
	"golang.org/x/text/transform"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/streambuf"
	"github.com/elastic/beats/libbeat/logp"
)

// lineReader reads lines from underlying reader, decoding the input stream
// using the configured codec. The reader keeps track of bytes consumed
// from raw input stream for every decoded line.
type Reader struct {
	lineScanner *lineScanner
}

// New creates a new reader object
func New(input io.Reader, codec encoding.Encoding, bufferSize int) (*Reader, error) {
	encoder := codec.NewEncoder()

	// Create newline char based on encoding
	nl, _, err := transform.Bytes(encoder, []byte{'\n'})
	if err != nil {
		return nil, err
	}

	decReader := newDecoderReader(input, codec)
	lineScanner := newLineScanner(decReader, nl, bufferSize)

	return &Reader{
		lineScanner: lineScanner,
	}, nil
}

// Next reads the next line until the new line character
func (r *Reader) Next() ([]byte, int, error) {
	return r.lineScanner.scan()
}

func (r *Reader) GetState() common.MapStr {
	return common.MapStr{
		"decoder": common.MapStr{
			"decoder": r.lineScanner.in.decodedOffset,
			"encoded": r.lineScanner.in.encodedOffset,
			"file":    r.lineScanner.in.fileOffset,
		},
		"scanner": common.MapStr{
			"line": common.MapStr{
				"buffer": r.lineScanner.bufOffset,
				"total":  r.lineScanner.offset,
			},
		},
	}
}

type decoderReader struct {
	in      io.Reader
	decoder transform.Transformer

	fileOffset    int
	encodedOffset int
	decodedOffset int
}

func newDecoderReader(in io.Reader, codec encoding.Encoding) *decoderReader {
	return &decoderReader{
		in:            in,
		decoder:       codec.NewDecoder(),
		fileOffset:    0,
		encodedOffset: 0,
		decodedOffset: 0,
	}
}

func (r *decoderReader) read(buf []byte) (int, error) {
	buffer := make([]byte, len(buf))
	n, err := r.in.Read(buffer)
	if n == 0 {
		return 0, streambuf.ErrNoMoreBytes
	}

	nDst, nSrc, err := r.decoder.Transform(buf, buffer, false)
	if err != nil {
		return 0, err
	}

	r.fileOffset = r.fileOffset + n
	r.encodedOffset = r.encodedOffset + nSrc
	r.decodedOffset = r.decodedOffset + nDst

	return nDst, nil
}

type lineScanner struct {
	in         *decoderReader
	nl         []byte
	bufferSize int

	buf       *streambuf.Buffer
	bufOffset int
	offset    int
}

func newLineScanner(in *decoderReader, nl []byte, bufferSize int) *lineScanner {
	return &lineScanner{
		in:         in,
		nl:         nl,
		bufferSize: bufferSize,
		buf:        streambuf.New(nil),
		bufOffset:  0,
		offset:     0,
	}
}

// Scan reads from the underlying decoder reader and returns decoded lines.
func (s *lineScanner) scan() ([]byte, int, error) {
	idx := s.buf.IndexFrom(s.bufOffset, s.nl)
	for !newLineFound(idx) {
		s.bufOffset = 0

		b := make([]byte, s.bufferSize)
		n, err := s.in.read(b)
		if err != nil {
			return nil, 0, err
		}

		// This can happen if something goes wrong during decoding
		if n == 0 {
			logp.Err("Empty buffer returned by read")
		}

		s.buf.Append(b[:n])
		idx = s.buf.IndexFrom(s.bufOffset, s.nl)
	}

	return s.line(idx)
}

// newLineFound checks if a new line was found.
func newLineFound(i int) bool {
	return i != -1
}

// line sets the offset of the scanner and returns a line.
func (s *lineScanner) line(i int) ([]byte, int, error) {
	line, err := s.buf.CollectUntil(s.nl)
	if err != nil {
		panic(err)
	}

	s.offset = s.offset + i + len(s.nl)
	s.bufOffset = s.bufOffset + i + len(s.nl)
	s.buf.Advance(s.bufOffset)
	return line, len(line), nil
}
