// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package readfile

import (
	"bytes"
	"fmt"
	"io"

	"golang.org/x/text/transform"

	"github.com/elastic/beats/v7/libbeat/common/streambuf"
	"github.com/elastic/beats/v7/libbeat/logp"
)

const unlimited = 0

// lineReader reads lines from underlying reader, decoding the input stream
// using the configured codec. The reader keeps track of bytes consumed
// from raw input stream for every decoded line.
type LineReader struct {
	reader     io.ReadCloser
	bufferSize int
	maxBytes   int // max bytes per line limit to avoid OOM with malformatted files
	nl         []byte
	decodedNl  []byte
	inBuffer   *streambuf.Buffer
	outBuffer  *streambuf.Buffer
	inOffset   int // input buffer read offset
	byteCount  int // number of bytes decoded from input buffer into output buffer
	decoder    transform.Transformer
	logger     *logp.Logger
}

// New creates a new reader object
func NewLineReader(input io.ReadCloser, config Config) (*LineReader, error) {
	encoder := config.Codec.NewEncoder()

	// Create newline char based on encoding
	terminator, ok := lineTerminatorCharacters[config.Terminator]
	if !ok {
		return nil, fmt.Errorf("unknown line terminator: %+v", config.Terminator)
	}

	nl, _, err := transform.Bytes(encoder, terminator)
	if err != nil {
		return nil, err
	}

	return &LineReader{
		reader:     input,
		bufferSize: config.BufferSize,
		maxBytes:   config.MaxBytes,
		decoder:    config.Codec.NewDecoder(),
		nl:         nl,
		decodedNl:  terminator,
		inBuffer:   streambuf.New(nil),
		outBuffer:  streambuf.New(nil),
		logger:     logp.NewLogger("reader_line"),
	}, nil
}

// Next reads the next line until the new line character
func (r *LineReader) Next() ([]byte, int, error) {
	// This loop is need in case advance detects an line ending which turns out
	// not to be one when decoded. If that is the case, reading continues.
	for {
		// read next 'potential' line from input buffer/reader
		err := r.advance()
		if err != nil {
			return nil, 0, err
		}

		// Check last decoded byte really being newline also unencoded
		// if not, continue reading
		buf := r.outBuffer.Bytes()

		// This can happen if something goes wrong during decoding
		if len(buf) == 0 {
			r.logger.Error("Empty buffer returned by advance")
			continue
		}

		if bytes.HasSuffix(buf, r.decodedNl) {
			break
		} else {
			r.logger.Debugf("Line ending char found which wasn't one: %c", buf[len(buf)-1])
			r.logger.Debugf("In %s", string(buf))
		}
	}

	// output buffer contains complete line ending with newline. Extract
	// byte slice from buffer and reset output buffer.
	bytes, err := r.outBuffer.Collect(r.outBuffer.Len())
	r.outBuffer.Reset()
	if err != nil {
		// This should never happen as otherwise we have a broken state
		panic(err)
	}

	// return and reset consumed bytes count
	sz := r.byteCount
	r.byteCount = 0
	return bytes, sz, nil
}

// Reads from the buffer until a new line character is detected
// Returns an error otherwise
func (r *LineReader) advance() error {
	// Initial check if buffer has already a newLine character
	idx := r.inBuffer.IndexFrom(r.inOffset, r.nl)

	// Fill inBuffer until newline sequence has been found in input buffer
	for idx == -1 {
		// Increase search offset to reduce iterations on buffer when looping
		newOffset := r.inBuffer.Len() - len(r.nl)
		if newOffset > r.inOffset {
			r.inOffset = newOffset
		}

		buf := make([]byte, r.bufferSize)

		// Try to read more bytes into buffer
		n, err := r.reader.Read(buf)

		if err == io.EOF && n > 0 {
			// Continue processing the returned bytes. The next call will yield EOF with 0 bytes.
			err = nil
		}

		// Appends buffer also in case of err
		r.inBuffer.Append(buf[:n])
		if err != nil {
			return err
		}

		// Empty read => return buffer error (more bytes required error)
		if n == 0 {
			return streambuf.ErrNoMoreBytes
		}

		// Check if buffer has newLine character
		idx = r.inBuffer.IndexFrom(r.inOffset, r.nl)

		// If max bytes limit per line is set, then drop the lines that are longer
		if r.maxBytes != 0 {
			// If newLine is found, drop the lines longer than maxBytes
			for idx != -1 && idx > r.maxBytes {
				r.logger.Warnf("Exceeded %d max bytes in line limit, skipped %d bytes line", r.maxBytes, idx)
				err = r.inBuffer.Advance(idx + len(r.nl))
				r.inBuffer.Reset()
				r.inOffset = 0
				idx = r.inBuffer.IndexFrom(r.inOffset, r.nl)
			}

			// If newLine is not found and the incoming data buffer exceeded max bytes limit, then skip until the next newLine
			if idx == -1 && r.inBuffer.Len() > r.maxBytes {
				skipped, err := r.skipUntilNewLine(buf)
				if err != nil {
					r.logger.Error("Error skipping until new line, err:", err)
					return err
				}
				r.logger.Warnf("Exceeded %d max bytes in line limit, skipped %d bytes line", r.maxBytes, skipped)
				idx = r.inBuffer.IndexFrom(r.inOffset, r.nl)
			}
		}
	}

	// Found encoded byte sequence for newline in buffer
	// -> decode input sequence into outBuffer
	sz, err := r.decode(idx + len(r.nl))
	if err != nil {
		r.logger.Errorf("Error decoding line: %s", err)
		// In case of error increase size by unencoded length
		sz = idx + len(r.nl)
	}

	// Consume transformed bytes from input buffer
	err = r.inBuffer.Advance(sz)
	r.inBuffer.Reset()

	// Continue scanning input buffer from last position + 1
	r.inOffset = idx + 1 - sz
	if r.inOffset < 0 {
		// Fix inOffset if newline has encoding > 8bits + firl line has been decoded
		r.inOffset = 0
	}

	return err
}

func (r *LineReader) skipUntilNewLine(buf []byte) (int, error) {
	// The length of the line skipped
	skipped := r.inBuffer.Len()

	// Clean up the buffer
	err := r.inBuffer.Advance(skipped)
	r.inBuffer.Reset()

	// Reset inOffset
	r.inOffset = 0

	if err != nil {
		return 0, err
	}

	// Read until the new line is found
	for idx := -1; idx == -1; {
		n, err := r.reader.Read(buf)

		// Check bytes read for newLine
		if n > 0 {
			idx = bytes.Index(buf[:n], r.nl)

			if idx != -1 {
				r.inBuffer.Append(buf[idx+len(r.nl) : n])
				skipped += idx
			} else {
				skipped += n
			}
		}

		if err != nil {
			return skipped, err
		}

		if n == 0 {
			return skipped, streambuf.ErrNoMoreBytes
		}
	}

	return skipped, nil
}

func (r *LineReader) decode(end int) (int, error) {
	var err error
	buffer := make([]byte, 1024)
	inBytes := r.inBuffer.Bytes()
	start := 0

	for start < end {
		var nDst, nSrc int

		nDst, nSrc, err = r.decoder.Transform(buffer, inBytes[start:end], false)
		if err != nil {
			// Check if error is different from destination buffer too short
			if err != transform.ErrShortDst {
				r.outBuffer.Write(inBytes[0:end])
				start = end
				break
			}

			// Reset error as decoding continues
			err = nil
		}

		start += nSrc
		r.outBuffer.Write(buffer[:nDst])
	}

	r.byteCount += start
	return start, err
}

func (r *LineReader) Close() error {
	return r.reader.Close()
}
