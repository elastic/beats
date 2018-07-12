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

package line

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
type Reader struct {
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

// New creates a new reader object
func New(input io.Reader, codec encoding.Encoding, bufferSize int) (*Reader, error) {
	encoder := codec.NewEncoder()

	// Create newline char based on encoding
	nl, _, err := transform.Bytes(encoder, []byte{'\n'})
	if err != nil {
		return nil, err
	}

	return &Reader{
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
func (r *Reader) Next() ([]byte, int, error) {
	// This loop is need in case advance detects an line ending which turns out
	// not to be one when decoded. If that is the case, reading continues.
	for {
		// read next 'potential' line from input buffer/reader
		err := r.advance()
		if err != nil {
			return nil, 0, err
		}

		// Check last decoded byte really being '\n' also unencoded
		// if not, continue reading
		buf := r.outBuffer.Bytes()

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
func (r *Reader) advance() error {
	// Initial check if buffer has already a newLine character
	idx := r.inBuffer.IndexFrom(r.inOffset, r.nl)

	// fill inBuffer until '\n' sequence has been found in input buffer
	for idx == -1 {
		// increase search offset to reduce iterations on buffer when looping
		newOffset := r.inBuffer.Len() - len(r.nl)
		if newOffset > r.inOffset {
			r.inOffset = newOffset
		}

		buf := make([]byte, r.bufferSize)

		// try to read more bytes into buffer
		n, err := r.reader.Read(buf)

		// Appends buffer also in case of err
		r.inBuffer.Append(buf[:n])
		if err != nil {
			return err
		}

		// empty read => return buffer error (more bytes required error)
		if n == 0 {
			return streambuf.ErrNoMoreBytes
		}

		// Check if buffer has newLine character
		idx = r.inBuffer.IndexFrom(r.inOffset, r.nl)
	}

	// found encoded byte sequence for '\n' in buffer
	// -> decode input sequence into outBuffer
	sz, err := r.decode(idx + len(r.nl))
	if err != nil {
		logp.Err("Error decoding line: %s", err)
		// In case of error increase size by unencoded length
		sz = idx + len(r.nl)
	}

	// consume transformed bytes from input buffer
	err = r.inBuffer.Advance(sz)
	r.inBuffer.Reset()

	// continue scanning input buffer from last position + 1
	r.inOffset = idx + 1 - sz
	if r.inOffset < 0 {
		// fix inOffset if '\n' has encoding > 8bits + firl line has been decoded
		r.inOffset = 0
	}

	return err
}

func (r *Reader) decode(end int) (int, error) {
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
