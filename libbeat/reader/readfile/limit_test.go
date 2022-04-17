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
// +build !integration

package readfile

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/menderesk/beats/v7/libbeat/reader"
)

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
