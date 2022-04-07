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

package readjson

import (
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/reader"
)

func TestDockerJSON(t *testing.T) {
	tests := []struct {
		name            string
		input           [][]byte
		stream          string
		partial         bool
		format          string
		criflags        bool
		expectedError   error
		expectedMessage reader.Message
	}{
		{
			name:   "Common log message",
			input:  [][]byte{[]byte(`{"log":"1:M 09 Nov 13:27:36.276 # User requested shutdown...\n","stream":"stdout","time":"2017-11-09T13:27:36.277747246Z"}`)},
			stream: "all",
			expectedMessage: reader.Message{
				Content: []byte("1:M 09 Nov 13:27:36.276 # User requested shutdown...\n"),
				Fields:  common.MapStr{"stream": "stdout"},
				Ts:      time.Date(2017, 11, 9, 13, 27, 36, 277747246, time.UTC),
				Bytes:   122,
			},
		},
		{
			name:          "Wrong JSON",
			input:         [][]byte{[]byte(`this is not JSON`)},
			stream:        "all",
			expectedError: io.EOF,
			expectedMessage: reader.Message{
				Bytes: 16,
			},
		},
		{
			name:   "0 length message",
			input:  [][]byte{[]byte(`{"log":"","stream":"stdout","time":"2017-11-09T13:27:36.277747246Z"}`)},
			stream: "all",
			expectedMessage: reader.Message{
				Content: []byte(""),
				Fields:  common.MapStr{"stream": "stdout"},
				Ts:      time.Date(2017, 11, 9, 13, 27, 36, 277747246, time.UTC),
				Bytes:   68,
			},
		},
		{
			name:          "Wrong CRI",
			input:         [][]byte{[]byte(`2017-09-12T22:32:21.212861448Z stdout`)},
			stream:        "all",
			expectedError: io.EOF,
			expectedMessage: reader.Message{
				Bytes: 37,
			},
		},
		{
			name:          "Wrong CRI",
			input:         [][]byte{[]byte(`{this is not JSON nor CRI`)},
			stream:        "all",
			expectedError: io.EOF,
			expectedMessage: reader.Message{
				Bytes: 25,
			},
		},
		{
			name:          "Missing time",
			input:         [][]byte{[]byte(`{"log":"1:M 09 Nov 13:27:36.276 # User requested shutdown...\n","stream":"stdout"}`)},
			stream:        "all",
			expectedError: io.EOF,
			expectedMessage: reader.Message{
				Bytes: 82,
			},
		},
		{
			name:   "CRI log no tags",
			input:  [][]byte{[]byte(`2017-09-12T22:32:21.212861448Z stdout 2017-09-12 22:32:21.212 [INFO][88] table.go 710: Invalidating dataplane cache`)},
			stream: "all",
			expectedMessage: reader.Message{
				Content: []byte("2017-09-12 22:32:21.212 [INFO][88] table.go 710: Invalidating dataplane cache"),
				Fields:  common.MapStr{"stream": "stdout"},
				Ts:      time.Date(2017, 9, 12, 22, 32, 21, 212861448, time.UTC),
				Bytes:   115,
			},
			criflags: false,
		},
		{
			name:   "CRI log",
			input:  [][]byte{[]byte(`2017-09-12T22:32:21.212861448Z stdout F 2017-09-12 22:32:21.212 [INFO][88] table.go 710: Invalidating dataplane cache`)},
			stream: "all",
			expectedMessage: reader.Message{
				Content: []byte("2017-09-12 22:32:21.212 [INFO][88] table.go 710: Invalidating dataplane cache"),
				Fields:  common.MapStr{"stream": "stdout"},
				Ts:      time.Date(2017, 9, 12, 22, 32, 21, 212861448, time.UTC),
				Bytes:   117,
			},
			criflags: true,
		},
		{
			name: "Filtering stream, bytes count accounts for all (filtered and not)",
			input: [][]byte{
				[]byte(`{"log":"filtered\n","stream":"stdout","time":"2017-11-09T13:27:36.277747246Z"}`),
				[]byte(`{"log":"unfiltered\n","stream":"stderr","time":"2017-11-09T13:27:36.277747246Z"}`),
				[]byte(`{"log":"unfiltered\n","stream":"stdout","time":"2017-11-09T13:27:36.277747246Z"}`),
			},
			stream: "stderr",
			expectedMessage: reader.Message{
				Content: []byte("unfiltered\n"),
				Fields:  common.MapStr{"stream": "stderr"},
				Ts:      time.Date(2017, 11, 9, 13, 27, 36, 277747246, time.UTC),
				Bytes:   158,
			},
		},
		{
			name: "Filtering CRI stream, bytes count accounts for all (filtered and not)",
			input: [][]byte{
				[]byte(`2017-10-12T13:32:21.232861448Z stdout F 2017-10-12 13:32:21.212 [INFO][88] table.go 710: Invalidating dataplane cache`),
				[]byte(`2017-11-12T23:32:21.212771448Z stderr F 2017-11-12 23:32:21.212 [ERROR][77] table.go 111: error`),
				[]byte(`2017-12-12T10:32:21.212864448Z stdout F 2017-12-12 10:32:21.212 [WARN][88] table.go 222: Warn`),
			},
			stream: "stderr",
			expectedMessage: reader.Message{
				Content: []byte("2017-11-12 23:32:21.212 [ERROR][77] table.go 111: error"),
				Fields:  common.MapStr{"stream": "stderr"},
				Ts:      time.Date(2017, 11, 12, 23, 32, 21, 212771448, time.UTC),
				Bytes:   212,
			},
			criflags: true,
		},
		{
			name: "Split lines",
			input: [][]byte{
				[]byte(`{"log":"1:M 09 Nov 13:27:36.276 # User requested ","stream":"stdout","time":"2017-11-09T13:27:36.277747246Z"}`),
				[]byte(`{"log":"shutdown...\n","stream":"stdout","time":"2017-11-09T13:27:36.277747246Z"}`),
			},
			stream:  "stdout",
			partial: true,
			expectedMessage: reader.Message{
				Content: []byte("1:M 09 Nov 13:27:36.276 # User requested shutdown...\n"),
				Fields:  common.MapStr{"stream": "stdout"},
				Ts:      time.Date(2017, 11, 9, 13, 27, 36, 277747246, time.UTC),
				Bytes:   190,
			},
		},
		{
			name: "CRI Split lines",
			input: [][]byte{
				[]byte(`2017-10-12T13:32:21.232861448Z stdout P 2017-10-12 13:32:21.212 [INFO][88] table.go 710: Invalidating dataplane cache`),
				[]byte(`2017-11-12T23:32:21.212771448Z stdout F  error`),
			},
			stream:  "stdout",
			partial: true,
			expectedMessage: reader.Message{
				Content: []byte("2017-10-12 13:32:21.212 [INFO][88] table.go 710: Invalidating dataplane cache error"),
				Fields:  common.MapStr{"stream": "stdout"},
				Ts:      time.Date(2017, 10, 12, 13, 32, 21, 232861448, time.UTC),
				Bytes:   163,
			},
			criflags: true,
		},
		{
			name: "Split lines and remove \\n",
			input: [][]byte{
				[]byte("2017-10-12T13:32:21.232861448Z stdout P 2017-10-12 13:32:21.212 [INFO][88] table.go 710: Invalidating dataplane cache\n"),
				[]byte("2017-11-12T23:32:21.212771448Z stdout F  error"),
			},
			stream: "stdout",
			expectedMessage: reader.Message{
				Content: []byte("2017-10-12 13:32:21.212 [INFO][88] table.go 710: Invalidating dataplane cache error"),
				Fields:  common.MapStr{"stream": "stdout"},
				Ts:      time.Date(2017, 10, 12, 13, 32, 21, 232861448, time.UTC),
				Bytes:   164,
			},
			partial:  true,
			criflags: true,
		},
		{
			name: "Split lines with partial disabled",
			input: [][]byte{
				[]byte(`{"log":"1:M 09 Nov 13:27:36.276 # User requested ","stream":"stdout","time":"2017-11-09T13:27:36.277747246Z"}`),
				[]byte(`{"log":"shutdown...\n","stream":"stdout","time":"2017-11-09T13:27:36.277747246Z"}`),
			},
			stream:  "stdout",
			partial: false,
			expectedMessage: reader.Message{
				Content: []byte("1:M 09 Nov 13:27:36.276 # User requested "),
				Fields:  common.MapStr{"stream": "stdout"},
				Ts:      time.Date(2017, 11, 9, 13, 27, 36, 277747246, time.UTC),
				Bytes:   109,
			},
		},
		{
			name:          "Force CRI with JSON logs",
			input:         [][]byte{[]byte(`{"log":"1:M 09 Nov 13:27:36.276 # User requested shutdown...\n","stream":"stdout"}`)},
			stream:        "all",
			format:        "cri",
			expectedError: io.EOF,
			expectedMessage: reader.Message{
				Bytes: 82,
			},
		},
		{
			name:          "Force JSON with CRI logs",
			input:         [][]byte{[]byte(`2017-09-12T22:32:21.212861448Z stdout 2017-09-12 22:32:21.212 [INFO][88] table.go 710: Invalidating dataplane cache`)},
			stream:        "all",
			format:        "docker",
			expectedError: io.EOF,
			expectedMessage: reader.Message{
				Bytes: 115,
			},
		},
		{
			name:   "Force CRI log no tags",
			input:  [][]byte{[]byte(`2017-09-12T22:32:21.212861448Z stdout 2017-09-12 22:32:21.212 [INFO][88] table.go 710: Invalidating dataplane cache`)},
			stream: "all",
			expectedMessage: reader.Message{
				Content: []byte("2017-09-12 22:32:21.212 [INFO][88] table.go 710: Invalidating dataplane cache"),
				Fields:  common.MapStr{"stream": "stdout"},
				Ts:      time.Date(2017, 9, 12, 22, 32, 21, 212861448, time.UTC),
				Bytes:   115,
			},
			format:   "cri",
			criflags: false,
		},
		{
			name:   "Force CRI log with flags",
			input:  [][]byte{[]byte(`2017-09-12T22:32:21.212861448Z stdout F 2017-09-12 22:32:21.212 [INFO][88] table.go 710: Invalidating dataplane cache`)},
			stream: "all",
			expectedMessage: reader.Message{
				Content: []byte("2017-09-12 22:32:21.212 [INFO][88] table.go 710: Invalidating dataplane cache"),
				Fields:  common.MapStr{"stream": "stdout"},
				Ts:      time.Date(2017, 9, 12, 22, 32, 21, 212861448, time.UTC),
				Bytes:   117,
			},
			format:   "cri",
			criflags: true,
		},
		{
			name: "Force CRI split lines",
			input: [][]byte{
				[]byte(`2017-10-12T13:32:21.232861448Z stdout P 2017-10-12 13:32:21.212 [INFO][88] table.go 710: Invalidating dataplane cache`),
				[]byte(`2017-11-12T23:32:21.212771448Z stdout F  error`),
			},
			stream:  "stdout",
			partial: true,
			expectedMessage: reader.Message{
				Content: []byte("2017-10-12 13:32:21.212 [INFO][88] table.go 710: Invalidating dataplane cache error"),
				Fields:  common.MapStr{"stream": "stdout"},
				Ts:      time.Date(2017, 10, 12, 13, 32, 21, 232861448, time.UTC),
				Bytes:   163,
			},
			format:   "cri",
			criflags: true,
		},
		{
			name: "Force CRI split lines and remove \\n",
			input: [][]byte{
				[]byte("2017-10-12T13:32:21.232861448Z stdout P 2017-10-12 13:32:21.212 [INFO][88] table.go 710: Invalidating dataplane cache\n"),
				[]byte("2017-11-12T23:32:21.212771448Z stdout F  error"),
			},
			stream: "stdout",
			expectedMessage: reader.Message{
				Content: []byte("2017-10-12 13:32:21.212 [INFO][88] table.go 710: Invalidating dataplane cache error"),
				Fields:  common.MapStr{"stream": "stdout"},
				Ts:      time.Date(2017, 10, 12, 13, 32, 21, 232861448, time.UTC),
				Bytes:   164,
			},
			partial:  true,
			format:   "cri",
			criflags: true,
		},
		{
			name: "Error parsing still keeps good bytes count",
			input: [][]byte{
				[]byte(`{"log":"1:M 09 Nov 13:27:36.276 # User requested ","stream":"stdout","time":"2017-11-09T13:27:36.277747246Z"}`),
				[]byte(`{"log":"shutdown...\n","stream`),
			},
			stream:        "stdout",
			expectedError: io.EOF,
			expectedMessage: reader.Message{
				Bytes: 139,
			},
			partial: true,
		},
		{
			name: "Docker AttributesSplit lines",
			input: [][]byte{
				[]byte(`{"log":"hello\n","stream":"stdout","attrs":{"KEY1":"value1","KEY2":"value2"},"time":"2017-11-09T13:27:36.277747246Z"}`),
			},
			stream:  "stdout",
			partial: true,
			expectedMessage: reader.Message{
				Content: []byte("hello\n"),
				Fields:  common.MapStr{"docker": common.MapStr{"attrs": map[string]string{"KEY1": "value1", "KEY2": "value2"}}, "stream": "stdout"},
				Ts:      time.Date(2017, 11, 9, 13, 27, 36, 277747246, time.UTC),
				Bytes:   117,
			},
		},
		{
			name:          "Corrupted log message line",
			input:         [][]byte{[]byte(`36.276 # User requested shutdown...\n","stream":"stdout","time":"2017-11-09T13:27:36.277747246Z"}`)},
			stream:        "all",
			expectedError: io.EOF,
			expectedMessage: reader.Message{
				Bytes: 97,
			},
		},
		{
			name: "Corrupted log message line is skipped, keep correct bytes count",
			input: [][]byte{
				[]byte(`36.276 # User requested shutdown...\n","stream":"stdout","time":"2017-11-09T13:27:36.277747246Z"}`),
				[]byte(`{"log":"1:M 09 Nov 13:27:36.276 # User requested","stream":"stdout","time":"2017-11-09T13:27:36.277747246Z"}`),
			},
			stream: "all",
			expectedMessage: reader.Message{
				Content: []byte("1:M 09 Nov 13:27:36.276 # User requested"),
				Fields:  common.MapStr{"stream": "stdout"},
				Ts:      time.Date(2017, 11, 9, 13, 27, 36, 277747246, time.UTC),
				Bytes:   205,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := &mockReader{messages: test.input}
			json := New(r, test.stream, test.partial, test.format, test.criflags)
			message, err := json.Next()

			if test.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, test.expectedError, err)
			} else {
				assert.NoError(t, err)
			}

			if err == nil {
				assert.EqualValues(t, test.expectedMessage, message)
			} else {
				assert.Equal(t, test.expectedMessage.Bytes, message.Bytes)
			}
		})
	}
}

type mockReader struct {
	messages [][]byte
}

func (m *mockReader) Next() (reader.Message, error) {
	if len(m.messages) < 1 {
		return reader.Message{
			Content: []byte{},
			Bytes:   0,
		}, io.EOF
	}
	message := m.messages[0]
	m.messages = m.messages[1:]
	return reader.Message{
		Content: message,
		Bytes:   len(message),
	}, nil
}

func (m *mockReader) Close() error { return nil }
