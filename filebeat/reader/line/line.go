package line

import (
	"io"

	"golang.org/x/text/encoding"
	"golang.org/x/text/transform"

	"github.com/elastic/beats/libbeat/common"
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
	decReader := newDecoderReader(input, codec, bufferSize)

	encoder := codec.NewEncoder()
	nl, _, err := transform.Bytes(encoder, []byte{'\n'})
	if err != nil {
		return nil, err
	}
	lineScanner := newLineScanner(decReader, bufferSize, nl)

	return &Reader{
		lineScanner: lineScanner,
	}, nil
}

// Next reads the next line until the new line character
func (r *Reader) Next() ([]byte, int, error) {
	return r.lineScanner.scan()
}

type decoderReader struct {
	in         io.Reader
	decoder    transform.Transformer
	buf        *streambuf.Buffer
	bufferSize int

	inOffset      int
	bytesOffset   int
	decodedOffset int
}

func (r *Reader) GetState() common.MapStr {
	return common.MapStr{
		"decoder": common.MapStr{
			"file":    r.lineScanner.in.inOffset,
			"encoded": r.lineScanner.in.bytesOffset,
			"decoded": r.lineScanner.in.decodedOffset,
		},
		"scanner": common.MapStr{
			"line": r.lineScanner.offset,
		},
	}
}

func newDecoderReader(in io.Reader, codec encoding.Encoding, bufferSize int) *decoderReader {
	return &decoderReader{
		in:         in,
		decoder:    codec.NewDecoder(),
		buf:        streambuf.New(nil),
		bufferSize: bufferSize,

		inOffset:      0,
		bytesOffset:   0,
		decodedOffset: 0,
	}
}

func (r *decoderReader) read(buf []byte) (int, error) {
	return r.in.Read(buf)
	b := make([]byte, r.bufferSize)
	n, err := r.in.Read(b)
	if err != nil {
		return 0, err
	}

	r.inOffset += n
	symlen := make([]int, n)
	nBytes, nSymbols, err := r.conv(b[:n], buf, symlen)
	if err != nil {
		return 0, err
	}

	r.bytesOffset += nBytes
	r.decodedOffset += nSymbols

	return nSymbols, nil
}

// conv
func (r *decoderReader) conv(in []byte, out []byte, symlen []int) (int, int, error) {
	var err error
	nBytes := 0
	nSymbols := 0
	bufSymLen := make([]int, len(symlen))

	i := 0
	for i < srcLen {
		j := i + 1
		for j <= srcLen {
			deChar := make([]byte, 1024)
			nDst, nSrc, err := r.decoder.Transform(deChar, in[i:j], false)
			if err != nil {
				if err == transform.ErrShortSrc {
					j++

					if srcLen < j {
						return nBytes, nSymbols, err
					}
					continue
				}
				err = nil
			}
			bufSymLen[nSymbols] = nDst
			r.buf.Write(deChar[:nDst])
			nBytes += nSrc
			nSymbols++
			break
		}
		i = j
	}

	b, err := r.buf.Collect(nSymbols)
	if err != nil {
		panic(err)
	}
	copy(out, b)

	return nBytes, nSymbols, err
}

type lineScanner struct {
	in         *decoderReader
	nl         []byte
	bufferSize int

	buf    *streambuf.Buffer
	offset int
}

func newLineScanner(in *decoderReader, bufferSize int, nl []byte) *lineScanner {
	return &lineScanner{
		in:         in,
		nl:         nl,
		bufferSize: bufferSize,
		buf:        streambuf.New(nil),
		offset:     0,
	}
}

// Scan reads from the underlying decoder reader and returns decoded lines.
func (s *lineScanner) scan() ([]byte, int, error) {
	idx := s.buf.Index(s.nl)
	for !newLineFound(idx) {
		b := make([]byte, s.bufferSize)
		n, err := s.in.read(b)
		if err != nil {
			return nil, 0, err
		}

		s.buf.Append(b[:n])
		idx = s.buf.Index(s.nl)
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
