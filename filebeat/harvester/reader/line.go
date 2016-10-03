package reader

import (
	"io"

	"golang.org/x/text/encoding"
	"golang.org/x/text/transform"

	"github.com/elastic/beats/libbeat/common/streambuf"
	"github.com/elastic/beats/libbeat/logp"
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

// NewLine creates a new Line reader object
func NewLine(input io.Reader, codec encoding.Encoding, bufferSize int) (*Line, error) {

	encoder := codec.NewEncoder()

	// Create newline char based on encoding
	nl, _, err := transform.Bytes(encoder, []byte{'\n'})
	if err != nil {
		return nil, err
	}

	return &Line{
		reader:     input,
		codec:      codec,
		bufferSize: bufferSize,
		nl:         nl,
		decoder:    codec.NewDecoder(),
		inBuffer:   streambuf.New(nil),
		outBuffer:  streambuf.New(nil),
	}, nil
}

// Next reads the next line until the new line character
func (l *Line) Next() ([]byte, int, error) {

	// This loop is need in case advance detects an line ending which turns out
	// not to be one when decoded. If that is the case, reading continues.
	for {
		// read next 'potential' line from input buffer/reader
		err := l.advance()
		if err != nil {
			return nil, 0, err
		}

		// Check last decoded byte really being '\n' also unencoded
		// if not, continue reading
		buf := l.outBuffer.Bytes()

		// This can happen if something goes wrong during decoding
		if len(buf) == 0 {
			logp.Err("Empty buffer returned by advance")
			continue
		}

		if buf[len(buf)-1] == '\n' {
			break
		} else {
			logp.Debug("line", "Line ending char found which wasn't one: %s", buf[len(buf)-1])
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

// Reads from the buffer until a new line character is detected
// Returns an error otherwise
func (l *Line) advance() error {

	// Initial check if buffer has already a newLine character
	idx := l.inBuffer.IndexFrom(l.inOffset, l.nl)

	// fill inBuffer until '\n' sequence has been found in input buffer
	for idx == -1 {
		// increase search offset to reduce iterations on buffer when looping
		newOffset := l.inBuffer.Len() - len(l.nl)
		if newOffset > l.inOffset {
			l.inOffset = newOffset
		}

		buf := make([]byte, l.bufferSize)

		// try to read more bytes into buffer
		n, err := l.reader.Read(buf)

		// Appends buffer also in case of err
		l.inBuffer.Append(buf[:n])
		if err != nil {
			return err
		}

		// empty read => return buffer error (more bytes required error)
		if n == 0 {
			return streambuf.ErrNoMoreBytes
		}

		// Check if buffer has newLine character
		idx = l.inBuffer.IndexFrom(l.inOffset, l.nl)
	}

	// found encoded byte sequence for '\n' in buffer
	// -> decode input sequence into outBuffer
	sz, err := l.decode(idx + len(l.nl))
	if err != nil {
		logp.Err("Error decoding line: %s", err)
		// In case of error increase size by unencoded length
		sz = idx + len(l.nl)
	}

	// consume transformed bytes from input buffer
	err = l.inBuffer.Advance(sz)
	l.inBuffer.Reset()

	// continue scanning input buffer from last position + 1
	l.inOffset = idx + 1 - sz
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
		if err != nil {
			// Check if error is different from destination buffer too short
			if err != transform.ErrShortDst {
				l.outBuffer.Write(inBytes[0:end])
				start = end
				break
			}

			// Reset error as decoding continues
			err = nil
		}

		start += nSrc
		l.outBuffer.Write(buffer[:nDst])
	}

	l.byteCount += start
	return start, err
}
