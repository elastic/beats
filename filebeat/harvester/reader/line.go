package reader

import (
	"io"

	"golang.org/x/text/encoding"
	"golang.org/x/text/transform"

	"github.com/elastic/beats/libbeat/common/streambuf"
)

// lineReader reads lines from underlying reader, decoding the input stream
// using the configured codec. The reader keeps track of bytes consumed
// from raw input stream for every decoded line.
type Line struct {
	reader     io.Reader
	codec      encoding.Encoding
	bufferSize int
	nl         []byte
	inBuffer   *streambuf.Buffer
	outBuffer  *streambuf.Buffer
	inOffset   int // input buffer read offset
	byteCount  int // number of bytes decoded from input buffer into output buffer
	decoder    transform.Transformer
}

func NewLine(
	input io.Reader,
	codec encoding.Encoding,
	bufferSize int,
) (*Line, error) {
	l := &Line{}

	if err := l.init(input, codec, bufferSize); err != nil {
		return nil, err
	}

	return l, nil
}

func (l *Line) init(
	reader io.Reader,
	codec encoding.Encoding,
	bufferSize int,
) error {
	l.reader = reader
	l.codec = codec
	l.bufferSize = bufferSize

	l.codec.NewEncoder()
	nl, _, err := transform.Bytes(l.codec.NewEncoder(), []byte{'\n'})
	if err != nil {
		return err
	}

	l.nl = nl
	l.decoder = l.codec.NewDecoder()
	l.inBuffer = streambuf.New(nil)
	l.outBuffer = streambuf.New(nil)
	return nil
}

func (l *Line) Next() ([]byte, int, error) {
	for {
		// read next 'potential' line from input buffer/reader
		err := l.advance()
		if err != nil {
			return nil, 0, err
		}

		// check last decoded byte really being '\n'
		buf := l.outBuffer.Bytes()
		if buf[len(buf)-1] == '\n' {
			break
		}
	}

	// output buffer contains complete line ending with '\n'. Extract
	// byte slice from buffer and reset output buffer.
	bytes, err := l.outBuffer.Collect(l.outBuffer.Len())
	l.outBuffer.Reset()
	if err != nil {
		// This should never happen as otherwise we have a broken state
		panic(err)
	}

	// return and reset consumed bytes count
	sz := l.byteCount
	l.byteCount = 0
	return bytes, sz, nil
}

func (l *Line) advance() error {
	var idx int
	var err error

	// fill inBuffer until '\n' sequence has been found in input buffer
	for {
		// Check if buffer has newLine character
		idx = l.inBuffer.IndexFrom(l.inOffset, l.nl)
		if idx >= 0 {
			break
		}
		if err != nil {
			// if no newline and last read returned error, return error now
			return err
		}

		// increase search offset to reduce iterations on buffer when looping
		newOffset := l.inBuffer.Len() - len(l.nl)
		if newOffset > l.inOffset {
			l.inOffset = newOffset
		}

		// try to read more bytes into buffer
		n := 0
		buf := make([]byte, l.bufferSize)
		n, err := l.reader.Read(buf)
		l.inBuffer.Append(buf[:n])
		if n == 0 && err != nil {
			// return error only if no bytes have been received. Otherwise try to
			// parse '\n' before returning the error.
			return err
		}

		// empty read => return buffer error (more bytes required error)
		if n == 0 {
			return streambuf.ErrNoMoreBytes
		}
	}

	// found encoded byte sequence for '\n' in buffer
	// -> decode input sequence into outBuffer
	sz, err := l.decode(idx + len(l.nl))

	// consume transformed bytes from input buffer
	err = l.inBuffer.Advance(sz)
	l.inBuffer.Reset()

	l.inOffset = idx + 1 - sz // continue scanning input buffer from last position + 1
	if l.inOffset < 0 {
		// fix inOffset if '\n' has encoding > 8bits + fill line has been decoded
		l.inOffset = 0
	}

	return err
}

func (l *Line) decode(end int) (int, error) {
	var err error
	buffer := make([]byte, 1024)
	inBytes := l.inBuffer.Bytes()
	start := 0

	for start < end {
		var nDst, nSrc int

		nDst, nSrc, err = l.decoder.Transform(buffer, inBytes[start:end], false)
		start += nSrc

		l.outBuffer.Write(buffer[:nDst])

		if err != nil {
			if err == transform.ErrShortDst { // continue transforming
				continue
			}
			break
		}
	}

	l.byteCount += start
	return start, err
}
