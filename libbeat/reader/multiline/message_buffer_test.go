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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestMessageBufferAddLine(t *testing.T) {
	testcases := map[string]struct {
		skipNewline bool
		lines       [][]byte
		expected    reader.Message
	}{
		"concatenating two events with newlines": {
			skipNewline: false,
			lines: [][]byte{
				[]byte("line1"),
				[]byte("line2"),
			},
			expected: reader.Message{
				Content: []byte("line1\nline2"),
			},
		},
		"concatenating two events without newlines": {
			skipNewline: true,
			lines: [][]byte{
				[]byte("{\"key1\": \"value\","),
				[]byte("\"key2\": \"value\"}"),
			},
			expected: reader.Message{
				Content: []byte("{\"key1\": \"value\",\"key2\": \"value\"}"),
			},
		},
	}

	for name, test := range testcases {

		t.Run(name, func(t *testing.T) {
			buf := getTestMessageBuffer(1024, test.skipNewline, nil)
			for _, l := range test.lines {
				m := reader.Message{Content: l, Bytes: len(l)}
				buf.addLine(m)
			}
			assert.Equal(t, test.expected.Content, buf.message.Content)
		})
	}
}

func TestMessageBufferPrivate(t *testing.T) {
	testcases := map[string]struct {
		messages []reader.Message
		expected any
	}{
		"preserves nil private value when no source message sets it": {
			messages: []reader.Message{
				{Content: []byte("line1"), Bytes: 5},
				{Content: []byte("line2"), Bytes: 5},
			},
			expected: nil,
		},
		"preserves private value for one source message": {
			messages: []reader.Message{
				{Content: []byte("line1"), Bytes: 5, Private: "cursor-1"},
			},
			expected: "cursor-1",
		},
		"aggregates private values for multiple source messages": {
			messages: []reader.Message{
				{Content: []byte("line1"), Bytes: 5, Private: "cursor-1"},
				{Content: []byte("line2"), Bytes: 5, Private: "cursor-2"},
				{Content: []byte("line3"), Bytes: 5, Private: "cursor-3"},
			},
			expected: []any{"cursor-1", "cursor-2", "cursor-3"},
		},
	}

	for name, test := range testcases {
		t.Run(name, func(t *testing.T) {
			buf := newMessageBuffer(1024, 5, []byte("\n"), false)
			for _, message := range test.messages {
				buf.addLine(message)
			}

			msg := buf.finalize()
			assert.Equal(
				t,
				test.expected,
				msg.Private,
				"Private should preserve scalar values for one source "+
					"message and aggregate multiple source messages",
			)
		})
	}
}

func TestFinalizeMessage(t *testing.T) {
	testcases := map[string]struct {
		maxBytes int
		lines    [][]byte
		expected reader.Message
	}{
		"one liner with no flags": {
			maxBytes: 1024,
			lines: [][]byte{
				[]byte("one line"),
			},
			expected: reader.Message{
				Content: []byte("one line"),
			},
		},
		"truncated one liner message": {
			maxBytes: 20,
			lines: [][]byte{
				[]byte("tooooooooooooooooooo looooooong line"),
			},
			expected: reader.Message{
				Content: []byte("tooooooooooooooooooo"),
				Fields:  mapstr.M{"log": mapstr.M{"flags": []string{"truncated"}}},
			},
		},
		"untruncated multiline message": {
			maxBytes: 1024,
			lines: [][]byte{
				[]byte("line1"),
				[]byte("line2"),
			},
			expected: reader.Message{
				Content: []byte("line1\nline2"),
				Fields:  mapstr.M{"log": mapstr.M{"flags": []string{"multiline"}}},
			},
		},
		"truncated multiline message": {
			maxBytes: 8,
			lines: [][]byte{
				[]byte("line1"),
				[]byte("line2"),
			},
			expected: reader.Message{
				Content: []byte("line1\nli"),
				Fields:  mapstr.M{"log": mapstr.M{"flags": []string{"truncated", "multiline"}}},
			},
		},
	}

	for name, test := range testcases {

		t.Run(name, func(t *testing.T) {
			var messages []reader.Message
			for _, l := range test.lines {
				messages = append(messages, reader.Message{Content: l, Bytes: len(l)})
			}
			buf := getTestMessageBuffer(test.maxBytes, false, messages)
			actualMsg := buf.finalize()

			assert.Equal(t, test.expected.Content, actualMsg.Content)
			assert.Equal(t, test.expected.Fields, actualMsg.Fields)
		})
	}

}

func getTestMessageBuffer(maxBytes int, skipNewline bool, messages []reader.Message) *messageBuffer {
	buf := newMessageBuffer(maxBytes, 5, []byte("\n"), skipNewline)
	buf.clear()

	for _, m := range messages {
		buf.addLine(m)
	}

	return buf
}
