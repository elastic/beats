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

package multiline

import (
	"io"

	"github.com/menderesk/beats/v7/libbeat/reader"
)

type counterReader struct {
	reader     reader.Reader
	state      func(*counterReader) (reader.Message, error)
	linesCount int // number of lines to collect
	msgBuffer  *messageBuffer
}

func newMultilineCountReader(
	r reader.Reader,
	separator string,
	maxBytes int,
	config *Config,
) (reader.Reader, error) {
	maxLines := config.LinesCount
	if l := config.MaxLines; l != nil && 0 < *l {
		maxLines = *l
	}

	return &counterReader{
		reader:     r,
		state:      (*counterReader).readFirst,
		linesCount: config.LinesCount,
		msgBuffer:  newMessageBuffer(maxBytes, maxLines, []byte(separator), config.SkipNewLine),
	}, nil
}

// Next returns next multi-line event.
func (cr *counterReader) Next() (reader.Message, error) {
	return cr.state(cr)
}

func (cr *counterReader) readFirst() (reader.Message, error) {
	for {
		message, err := cr.reader.Next()
		if err != nil {
			return message, err
		}

		if message.Bytes == 0 {
			continue
		}

		cr.msgBuffer.startNewMessage(message)
		if cr.msgBuffer.processedLines == cr.linesCount {
			msg := cr.msgBuffer.finalize()
			return msg, nil
		}

		cr.setState((*counterReader).readNext)
		return cr.readNext()
	}
}

func (cr *counterReader) readNext() (reader.Message, error) {
	for {
		message, err := cr.reader.Next()
		if err != nil {
			// handle error without any bytes returned from reader
			if message.Bytes == 0 {
				// no lines buffered -> return error
				if cr.msgBuffer.isEmpty() {
					return reader.Message{}, err
				}

				// lines buffered, return multiline and error on next read
				msg := cr.msgBuffer.finalize()
				cr.msgBuffer.setErr(err)
				cr.setState((*counterReader).readFailed)
				return msg, nil
			}

			// handle error with some content being returned by reader and
			// line matching multiline criteria or no multiline started yet
			if cr.msgBuffer.isEmptyMessage() {
				cr.msgBuffer.addLine(message)

				// return multiline and error on next read
				msg := cr.msgBuffer.finalize()
				cr.msgBuffer.setErr(err)
				cr.setState((*counterReader).readFailed)
				return msg, nil
			}
		}

		// add line to current multiline event
		cr.msgBuffer.addLine(message)
		if cr.msgBuffer.processedLines == cr.linesCount {
			msg := cr.msgBuffer.finalize()
			cr.setState((*counterReader).readFirst)
			return msg, nil
		}
	}
}

func (cr *counterReader) readFailed() (reader.Message, error) {
	err := cr.msgBuffer.err
	cr.msgBuffer.setErr(nil)
	cr.resetState()
	return reader.Message{}, err
}

// resetState sets state of the reader to readFirst
func (cr *counterReader) resetState() {
	cr.setState((*counterReader).readFirst)
}

// setState sets state to the given function
func (cr *counterReader) setState(next func(cr *counterReader) (reader.Message, error)) {
	cr.state = next
}

func (cr *counterReader) Close() error {
	cr.setState((*counterReader).readClosed)
	return cr.reader.Close()
}

func (cr *counterReader) readClosed() (reader.Message, error) {
	return reader.Message{}, io.EOF
}
