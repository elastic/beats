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
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/reader"
)

type counterReader struct {
	reader    reader.Reader
	state     func(*counterReader) (reader.Message, error)
	logger    *logp.Logger
	msgBuffer *messageBuffer
}

func newMultilineCountReader(
	r reader.Reader,
	separator string,
	maxBytes int,
	config *Config,
) (reader.Reader, error) {
	return &counterReader{
		reader:    r,
		state:     (*counterReader).readFirstCount,
		msgBuffer: newMessageBuffer(maxBytes, config.LinesCount, []byte(separator), config.SkipNewLine),
		logger:    logp.NewLogger("reader_counter_multiline"),
	}, nil
}

// Next returns next multi-line event.
func (cr *counterReader) Next() (reader.Message, error) {
	return cr.state(cr)
}

func (cr *counterReader) readFailedCount() (reader.Message, error) {
	err := cr.msgBuffer.err
	cr.msgBuffer.setErr(nil)
	cr.resetCountState()
	return reader.Message{}, err
}

// resetCountState sets state of the reader to readFirst
func (cr *counterReader) resetCountState() {
	cr.setState((*counterReader).readFirstCount)
}

func (cr *counterReader) readFirstCount() (reader.Message, error) {
	for {
		message, err := cr.reader.Next()
		if err != nil {
			return message, err
		}

		if message.Bytes == 0 {
			continue
		}

		// Start new multiline event
		cr.msgBuffer.clear()
		cr.msgBuffer.load(message)
		cr.setState((*counterReader).readNextCount)
		return cr.readNextCount()
	}
}

func (cr *counterReader) readNextCount() (reader.Message, error) {
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
				cr.setState((*counterReader).readFailedCount)
				return msg, nil
			}

			// handle error with some content being returned by reader and
			// line matching multiline criteria or no multiline started yet
			if cr.msgBuffer.isEmptyMessage() {
				cr.msgBuffer.addLine(message)

				// return multiline and error on next read
				msg := cr.msgBuffer.finalize()
				cr.msgBuffer.setErr(err)
				cr.setState((*counterReader).readFailedCount)
				return msg, nil
			}
		}

		// if enough lines are aggregated, return multiline event
		if !cr.msgBuffer.isEmptyMessage() && cr.msgBuffer.numLines == cr.msgBuffer.maxLines {
			msg := cr.msgBuffer.finalize()
			cr.msgBuffer.load(message)
			return msg, nil
		}

		// add line to current multiline event
		cr.msgBuffer.addLine(message)
	}
}

// resetState sets state of the reader to readFirst
func (cr *counterReader) resetState() {
	cr.setState((*counterReader).readFirstCount)
}

// setState sets state to the given function
func (cr *counterReader) setState(next func(cr *counterReader) (reader.Message, error)) {
	cr.state = next
}
