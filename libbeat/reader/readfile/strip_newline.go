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

	"github.com/elastic/beats/libbeat/reader"
)

// StripNewline reader removes the last trailing newline characters from
// read lines.
type StripNewline struct {
	reader         reader.Reader
	nl             []byte
	lineEndingFunc func(*StripNewline, []byte) int
}

// New creates a new line reader stripping the last tailing newline.
func NewStripNewline(r reader.Reader, terminator LineTerminator) *StripNewline {
	lineEndingFunc := (*StripNewline).lineEndingChars
	if terminator == AutoLineTerminator {
		lineEndingFunc = (*StripNewline).autoLineEndingChars
	}

	return &StripNewline{
		reader:         r,
		nl:             lineTerminatorCharacters[terminator],
		lineEndingFunc: lineEndingFunc,
	}
}

// Next returns the next line.
func (p *StripNewline) Next() (reader.Message, error) {
	message, err := p.reader.Next()
	if err != nil {
		return message, err
	}

	L := message.Content
	message.Content = L[:len(L)-p.lineEndingFunc(p, L)]

	return message, err
}

// isLine checks if the given byte array is a line, means has a line ending \n
func (p *StripNewline) isLine(l []byte) bool {
	return bytes.HasSuffix(l, p.nl)
}

func (p *StripNewline) lineEndingChars(l []byte) int {
	if !p.isLine(l) {
		return 0
	}

	return len(p.nl)
}

func (p *StripNewline) autoLineEndingChars(l []byte) int {
	if !p.isLine(l) {
		return 0
	}

	if len(l) > 1 && l[len(l)-2] == '\r' {
		return 2
	}
	return 1
}
