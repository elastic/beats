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

//go:build !integration

package readfile

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/beats/v7/libbeat/reader/readertest"
)

func TestLimitReaderBufferedFlusher(t *testing.T) {
	deadline := time.Now().Add(time.Minute)

	t.Run("wrapped reader without BufferedFlusher", func(t *testing.T) {
		r := NewLimitReader(&mockReader{[]byte("line")}, 10)
		pending, dl := reader.PendingDeadline(r)
		assert.False(t, pending, "nothing can be pending")
		assert.True(t, dl.IsZero(), "no deadline")
		msg, ok, err := reader.FlushPending(r, true)
		require.NoError(t, err, "FlushPending")
		assert.False(t, ok, "nothing to flush")
		assert.True(t, msg.IsEmpty(), "no message")
	})

	tests := []struct {
		name          string
		maxBytes      int
		content       string
		existingFlags []string
		truncated     bool
	}{
		{name: "at the limit", maxBytes: 5, content: "short"},
		{name: "above the limit", maxBytes: 3, content: "short", truncated: true},
		{name: "above the limit with existing flags", maxBytes: 4, content: "multi\nline", existingFlags: []string{"multiline"}, truncated: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msg := reader.Message{Content: []byte(tc.content), Bytes: len(tc.content) + 1}
			require.NoError(t, msg.AddFlagsWithKey("log.flags", tc.existingFlags...), "adding flag")
			inner := &readertest.PendingReader{Pending: &msg, Deadline: deadline}
			r := NewLimitReader(inner, tc.maxBytes)

			pending, dl := reader.PendingDeadline(r)
			assert.True(t, pending, "pending message of the wrapped reader")
			assert.Equal(t, deadline, dl, "deadline of the wrapped reader")

			got, ok, err := reader.FlushPending(r, true)
			require.NoError(t, err, "FlushPending")
			require.True(t, ok, "FlushPending must flush the pending message")
			assert.Equal(t, []bool{true}, inner.TimedOut, "timedOut must be passed through")
			assert.Equal(t, tc.content[:min(len(tc.content), tc.maxBytes)], string(got.Content), "limited content")
			assert.Equal(t, len(tc.content)+1, got.Bytes, "bytes read must not be changed by the limit")

			flags, err := got.Fields.GetValue("log.flags")
			if tc.truncated {
				require.NoError(t, err, "truncated messages must have log.flags")
				assert.ElementsMatch(t, append([]string{"truncated"}, tc.existingFlags...), flags, "log.flags")
			} else {
				assert.Error(t, err, "messages within the limit must not have log.flags")
			}

			pending, _ = reader.PendingDeadline(r)
			assert.False(t, pending, "nothing pending after flush")
			got, ok, err = reader.FlushPending(r, true)
			require.NoError(t, err, "second FlushPending")
			assert.False(t, ok, "nothing to flush")
			assert.True(t, got.IsEmpty(), "no message when nothing is pending")
		})
	}
}

type mockReader struct {
	line []byte
}

func (m *mockReader) Next() (reader.Message, error) {
	return reader.Message{
		Content: m.line,
	}, nil
}

func (m *mockReader) Close() error { return nil }

var limitTests = []struct {
	line      string
	maxBytes  int
	truncated bool
}{
	{"long-long-line", 5, true},
	{"long-long-line", 3, true},
	{"long-long-line", len("long-long-line"), false},
}

func TestLimitReader(t *testing.T) {
	for _, test := range limitTests {
		r := NewLimitReader(&mockReader{[]byte(test.line)}, test.maxBytes)

		msg, err := r.Next()
		if err != nil {
			t.Fatalf("Error reading from mock reader: %v", err)
		}

		assert.Equal(t, test.maxBytes, len(msg.Content))

		found := false
		statusFlags, err := msg.Fields.GetValue("log.flags")
		if err != nil {
			if !test.truncated {
				assert.False(t, found)
				return
			}
			t.Fatalf("Error getting truncated value: %v", err)
		}

		switch flags := statusFlags.(type) {
		case []string:
			for _, f := range flags {
				if f == "truncated" {
					found = true
				}
			}
		default:
			t.Fatalf("incorrect type for log.flags")
		}

		if test.truncated {
			assert.True(t, found)
		} else {
			assert.False(t, found)
		}
	}
}
