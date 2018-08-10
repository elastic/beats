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
	"io"

	"golang.org/x/text/encoding"
	"golang.org/x/text/transform"

	"github.com/elastic/beats/libbeat/common/streambuf"
	"github.com/elastic/beats/libbeat/logp"
)

type Config struct {
	Codec      encoding.Encoding
	Separator  []byte
	BufferSize int
}

// State stores the offsets of the reader.
type State struct {
	EncodedOffset          int // offset in the encoded file
	ConvertedSegmentOffset int // offset in the UTF-8 segment returned by the decoderReader
	ConvertedStreamOffset  int // number of total processed UTF-8 bytes
}
// lineReader reads lines from underlying reader, decoding the input stream
// using the configured codec. The reader keeps track of bytes consumed
// from raw input stream for every decoded line.
type LineReader struct {
	lineScanner *lineScanner
}

// New creates a new reader object
func NewLineReader(input io.Reader, c Config) (*LineReader, error) {
	decReader, err := newDecoderReader(input, c.Codec, c.BufferSize)
	if err != nil {
		return nil, err
	}

	lineScanner := newLineScanner(decReader, c.Separator, c.BufferSize)

	return &LineReader{
		lineScanner: lineScanner,
	}, nil
}

// InitState initizalizas the state of the reader
// TODO update when registry is refactored
func (r *LineReader) SetState(s State) error {
	if s.ConvertedStreamOffset < s.ConvertedSegmentOffset {
		return fmt.Errorf("invalid state, converted stream offset cannot be lower than converted segment offset")
	}

	r.lineScanner.in.offset = s.EncodedOffset
	r.lineScanner.segmentOffset = s.ConvertedSegmentOffset
	r.lineScanner.streamOffset = s.ConvertedStreamOffset

	// TODO rm when registry is refactored
	err := r.lineScanner.in.seekToLastRead()
	if err != nil {
		return err
	}
	return r.lineScanner.seekToLastRead()
}

// Next reads the next line until the new line character
func (r *LineReader) Next() ([]byte, int, error) {
	return r.lineScanner.scan()
	// TODO send state
}
