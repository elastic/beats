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
	"go.uber.org/zap/zaptest/observer"

	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/elastic-agent-libs/logp"
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
		test := test

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
		test := test

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

func TestLogTruncatedMessage(t *testing.T) {
	tests := []struct {
		name       string
		limit      int
		messages   []reader.Message
		assertFunc func(t *testing.T, logs []observer.LoggedEntry)
	}{
		{
			name:  "truncated",
			limit: 10,
			messages: []reader.Message{
				{Content: []byte("line1\nline2\nline3"), Bytes: 15},
			},
			assertFunc: func(t *testing.T, logs []observer.LoggedEntry) {
				assert.Len(t, logs, 1)
				assert.Equal(t, "Multiline message is too large, truncated to the limit of 5 lines or 10 bytes", logs[0].Message)
				assert.Equal(t, "warn", logs[0].Level.String())
			},
		},
		{
			name:  "not truncated",
			limit: 15,
			messages: []reader.Message{
				{Content: []byte("line1\nline2"), Bytes: 10},
			},
			assertFunc: func(t *testing.T, logs []observer.LoggedEntry) {
				assert.Empty(t, logs)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logp.DevelopmentSetup(logp.ToObserverOutput())

			buf := getTestMessageBuffer(tt.limit, false, tt.messages)
			buf.finalize()

			logs := logp.ObserverLogs().FilterLoggerName("reader_multiline").TakeAll()
			tt.assertFunc(t, logs)
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
