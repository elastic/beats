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
	in *decoderScanner
}

// New creates a new reader object
func New(input io.Reader, codec encoding.Encoding, bufferSize int) (*Reader, error) {
	lineReader, err := newLineReader(input, codec, bufferSize)
	if err != nil {
		return nil, err
	}

	return &Reader{
		in: newDecoderScanner(lineReader, codec, bufferSize),
	}, nil
}

type lineReader struct {
	reader     io.Reader
	bufferSize int
	nl         []byte

	buffer *streambuf.Buffer
	offset int
}

func newLineReader(reader io.Reader, codec encoding.Encoding, bufferSize int) (*lineReader, error) {
	encoder := codec.NewEncoder()

	// Create newline char based on encoding
	nl, _, err := transform.Bytes(encoder, []byte{'\n'})
	if err != nil {
		return nil, err
	}

	return &lineReader{
		reader:     reader,
		bufferSize: bufferSize,
		nl:         nl,
		buffer:     streambuf.New(nil),
		offset:     0,
	}, nil
}

func (l *lineReader) Read(buf []byte) (int, error) {
	idx := l.buffer.Index(l.nl)

	if !isNewLine(idx) {
		buf := make([]byte, l.bufferSize)
		n, err := l.reader.Read(buf)
		if err != nil {
			return 0, err
		}
		l.buffer.Append(buf[:n])

		idx = l.buffer.Index(l.nl)
	}

	line, err := l.buffer.CollectUntil(l.nl)
	if err != nil {
		return 0, err
	}
	return copy(buf, line), nil
}

func isNewLine(idx int) bool {
	return idx != -1
}

type decoderScanner struct {
	reader     io.Reader
	decoder    transform.Transformer
	bufferSize int
	byteCount  int
}

func newDecoderScanner(reader io.Reader, codec encoding.Encoding, bufferSize int) *decoderScanner {
	return &decoderScanner{
		reader:     reader,
		decoder:    codec.NewDecoder(),
		bufferSize: bufferSize,
		byteCount:  0,
	}
}

func (d *decoderScanner) Scan() ([]byte, int, error) {
	buf := make([]byte, d.bufferSize)
	n, err := d.reader.Read(buf)
	if err != nil {
		return nil, 0, err
	}

	return transform.Bytes(d.decoder, buf[:n])
}

func (r *Reader) Next() ([]byte, int, error) {
	// This loop is need in case advance detects an line ending which turns out
	// not to be one when decoded. If that is the case, reading continues.
	var buf []byte
	var n int
	for {
		// read next 'potential' line from input buffer/reader
		buf, n, err := r.in.Scan()
		if err != nil {
			return nil, 0, err
		}

		if buf[len(buf)-1] == '\n' {
			return buf, n, err
		} else {
			logp.Debug("line", "Line ending char found which wasn't one: %s", buf[len(buf)-1])
		}
	}

	return buf, n, nil
}
