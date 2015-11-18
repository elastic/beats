package harvester

import (
	"io"
	"time"

	"golang.org/x/text/encoding"
	"golang.org/x/text/transform"

	"github.com/elastic/libbeat/common/streambuf"
)

// timedReader keeps track of last time bytes have been read from underlying
// reader.
type timedReader struct {
	reader       io.Reader
	lastReadTime time.Time // last time we read some data from input stream
}

// lineReader reads lines from underlying reader, decoding the input stream
// using the configured codec. The reader keeps track of bytes consumed
// from raw input stream for every decoded line.
type lineReader struct {
	rawInput   io.Reader
	codec      encoding.Encoding
	bufferSize int

	nl        []byte
	inBuffer  *streambuf.Buffer
	outBuffer *streambuf.Buffer
	inOffset  int // input buffer read offset
	byteCount int // number of bytes decoded from input buffer into output buffer
	decoder   transform.Transformer
}

const maxConsecutiveEmptyReads = 100

func newTimedReader(reader io.Reader) *timedReader {
	r := &timedReader{
		reader: reader,
	}
	return r
}

func (r *timedReader) Read(p []byte) (int, error) {
	var err error
	n := 0

	for i := maxConsecutiveEmptyReads; i > 0; i-- {
		n, err = r.reader.Read(p)
		if n > 0 {
			r.lastReadTime = time.Now()
			break
		}

		if err != nil {
			break
		}
	}

	return n, err
}

func newLineReader(
	input io.Reader,
	codec encoding.Encoding,
	bufferSize int,
) (*lineReader, error) {
	l := &lineReader{}

	if err := l.init(input, codec, bufferSize); err != nil {
		return nil, err
	}

	return l, nil
}

func (l *lineReader) init(
	input io.Reader,
	codec encoding.Encoding,
	bufferSize int,
) error {
	l.rawInput = input
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

func (l *lineReader) next() ([]byte, int, error) {
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
		panic(err)
	}

	// return and reset consumed bytes count
	sz := l.byteCount
	l.byteCount = 0
	return bytes, sz, nil
}

func (l *lineReader) advance() error {
	var idx int
	var err error

	// fill inBuffer until '\n' sequence has been found in input buffer
	for {
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
		n, err := l.rawInput.Read(buf)
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

func (l *lineReader) decode(end int) (int, error) {
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

// partial returns current state of decoded input bytes and amount of bytes
// processed so far. If decoder has detected an error in input stream, the error
// will be returned.
func (l *lineReader) partial() ([]byte, int, error) {
	// decode all input buffer
	sz, err := l.decode(l.inBuffer.Len())
	l.inBuffer.Advance(sz)
	l.inBuffer.Reset()

	l.inOffset -= sz
	if l.inOffset < 0 {
		l.inOffset = 0
	}

	// return current state of outBuffer, but do not consume any content yet
	bytes := l.outBuffer.Bytes()
	sz = l.byteCount
	return bytes, sz, err
}

// dropPartial drops current output buffer of decoded characters returning total number
// of input bytes consumed
func (l *lineReader) dropPartial() int {
	l.outBuffer.Advance(l.outBuffer.Len())
	l.outBuffer.Reset()
	sz := l.byteCount
	l.byteCount = 0
	return sz
}
