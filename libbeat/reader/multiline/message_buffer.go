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
	"github.com/menderesk/beats/v7/libbeat/reader"
)

type messageBuffer struct {
	maxBytes       int // bytes stored in content
	maxLines       int
	separator      []byte
	skipNewline    bool
	last           []byte
	numLines       int
	processedLines int
	truncated      int
	err            error // last seen error
	message        reader.Message
}

func newMessageBuffer(maxBytes, maxLines int, separator []byte, skipNewline bool) *messageBuffer {
	return &messageBuffer{
		maxBytes:    maxBytes,
		maxLines:    maxLines,
		separator:   separator,
		skipNewline: skipNewline,
		message:     reader.Message{},
		err:         nil,
	}
}

func (b *messageBuffer) startNewMessage(msg reader.Message) {
	b.clear()
	b.load(msg)
}

// load loads the reader with the given message. It is recommended to either
// run clear or finalize before.
func (b *messageBuffer) load(m reader.Message) {
	b.addLine(m)
	// Timestamp of first message is taken as overall timestamp
	b.message.Ts = m.Ts
	b.message.AddFields(m.Fields)
}

// clearBuffer resets the reader buffer variables
func (b *messageBuffer) clear() {
	b.message = reader.Message{}
	b.last = nil
	b.numLines = 0
	b.processedLines = 0
	b.truncated = 0
	b.err = nil
}

// addLine adds the read content to the message
// The content is only added if maxBytes and maxLines is not exceed. In case one of the
// two is exceeded, addLine keeps processing but does not add it to the content.
func (b *messageBuffer) addLine(m reader.Message) {
	if m.Bytes <= 0 {
		return
	}

	sz := len(b.message.Content)
	addSeparator := len(b.message.Content) > 0 && len(b.separator) > 0 && !b.skipNewline
	if addSeparator {
		sz += len(b.separator)
	}

	space := b.maxBytes - sz

	maxBytesReached := (b.maxBytes <= 0 || space > 0)
	maxLinesReached := (b.maxLines <= 0 || b.numLines < b.maxLines)

	if maxBytesReached && maxLinesReached {
		if space < 0 || space > len(m.Content) {
			space = len(m.Content)
		}

		tmp := b.message.Content
		if addSeparator {
			tmp = append(tmp, b.separator...)
		}
		b.message.Content = append(tmp, m.Content[:space]...)
		b.numLines++

		// add number of truncated bytes to fields
		diff := len(m.Content) - space
		if diff > 0 {
			b.truncated += diff
		}
	} else {
		// increase the number of skipped bytes, if cannot add
		b.truncated += len(m.Content)

	}
	b.processedLines++

	b.last = m.Content
	b.message.Bytes += m.Bytes
	b.message.AddFields(m.Fields)
}

// finalize writes the existing content into the returned message and resets all reader variables.
func (b *messageBuffer) finalize() reader.Message {
	if b.truncated > 0 {
		b.message.AddFlagsWithKey("log.flags", "truncated")
	}

	if b.numLines > 1 {
		b.message.AddFlagsWithKey("log.flags", "multiline")
	}

	// Copy message from existing content
	msg := b.message

	b.clear()
	return msg
}

func (b *messageBuffer) setErr(err error) {
	b.err = err
}

func (b *messageBuffer) isEmpty() bool {
	return b.numLines == 0
}

func (b *messageBuffer) isEmptyMessage() bool {
	return b.message.Bytes == 0
}
