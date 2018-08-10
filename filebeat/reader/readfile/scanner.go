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

	"github.com/elastic/beats/libbeat/common/streambuf"
)

// lineScanner scans for new lines separated by a configurable byte slice.
type lineScanner struct {
	in         *decoderReader
	separator  []byte
	bufferSize int

	buf           *streambuf.Buffer
	segmentOffset int
	streamOffset  int
}

func newLineScanner(in *decoderReader, separator []byte, bufferSize int) *lineScanner {
	return &lineScanner{
		in:         in,
		separator:  separator,
		bufferSize: bufferSize,
		buf:        streambuf.New(nil),
	}
}

// seekToLastRead moves the fp to the last read position.
// The number of bytes which needs to be read is the size of the
// configured harvester_buffer_size when the state was written.
func (s *lineScanner) seekToLastRead() error {
	if s.streamOffset == 0 {
		return nil
	}

	remaining := s.segmentOffset
	for remaining > 0 {
		size := s.bufferSize
		if remaining < size {
			size = remaining
		}

		b := make([]byte, size)
		n, err := s.in.read(b)
		remaining -= n
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}

	return nil
}

// Scan reads from the underlying decoder reader and returns decoded lines.
func (s *lineScanner) scan() ([]byte, int, error) {
	idx := s.buf.Index(s.separator)
	for !separatorFound(idx) {
		b := make([]byte, s.bufferSize)
		n, err := s.in.read(b)
		s.buf.Append(b[:n])
		if err != nil {
			return nil, 0, err
		}

		s.segmentOffset = 0
		idx = s.buf.Index(s.separator)
	}

	return s.line(idx)
}

// separatorFound checks if a new separator was found.
func separatorFound(i int) bool {
	return i != -1
}

// line sets the offset of the scanner and returns a line.
func (s *lineScanner) line(i int) ([]byte, int, error) {
	line, err := s.buf.CollectUntil(s.separator)
	if err != nil {
		panic(err)
	}

	s.segmentOffset += i
	s.streamOffset += i
	s.buf.Reset()

	return line, len(line), nil
}
