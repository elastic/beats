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

import "github.com/elastic/beats/libbeat/common/streambuf"

type lineScanner struct {
	in         *decoderReader
	separator  []byte
	bufferSize int

	symlen         []uint8
	buf            *streambuf.Buffer
	offset         int
	bytesOffset    int
	lastMessageLen int
}

func newLineScanner(in *decoderReader, separator []byte, bufferSize int) *lineScanner {
	return &lineScanner{
		in:          in,
		separator:   separator,
		bufferSize:  bufferSize,
		buf:         streambuf.New(nil),
		offset:      0,
		bytesOffset: 0,
		symlen:      []uint8{},
	}
}

// Scan reads from the underlying decoder reader and returns decoded lines.
func (s *lineScanner) scan() ([]byte, int, error) {
	idx := s.buf.Index(s.separator)
	for !separatorFound(idx) {
		b := make([]byte, s.bufferSize)
		n, err := s.in.read(b)
		if err != nil {
			return nil, 0, err
		}

		s.buf.Append(b[:n])
		s.symlen = append(s.symlen, s.in.symbolsLen()...)
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

	var msgSymbols int
	msgSymbols, s.symlen, err = s.in.msgSize(s.symlen, len(line))
	if err != nil {
		return nil, 0, err
	}

	s.bytesOffset += msgSymbols
	s.offset += i
	s.buf.Reset()

	return line, msgSymbols, nil
}
