package reader

import (
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"

	"github.com/stretchr/testify/assert"
)

func TestDockerJSON(t *testing.T) {
	tests := []struct {
		input           [][]byte
		stream          string
		expectedError   bool
		expectedMessage Message
	}{
		// Common log message
		{
			input:  [][]byte{[]byte(`{"log":"1:M 09 Nov 13:27:36.276 # User requested shutdown...\n","stream":"stdout","time":"2017-11-09T13:27:36.277747246Z"}`)},
			stream: "all",
			expectedMessage: Message{
				Content: []byte("1:M 09 Nov 13:27:36.276 # User requested shutdown...\n"),
				Fields:  common.MapStr{"stream": "stdout"},
				Ts:      time.Date(2017, 11, 9, 13, 27, 36, 277747246, time.UTC),
			},
		},
		// Wrong JSON
		{
			input:         [][]byte{[]byte(`this is not JSON`)},
			stream:        "all",
			expectedError: true,
		},
		// Wrong CRI
		{
			input:         [][]byte{[]byte(`2017-09-12T22:32:21.212861448Z stdout`)},
			stream:        "all",
			expectedError: true,
		},
		// Wrong CRI
		{
			input:         [][]byte{[]byte(`{this is not JSON nor CRI`)},
			stream:        "all",
			expectedError: true,
		},
		// Missing time
		{
			input:         [][]byte{[]byte(`{"log":"1:M 09 Nov 13:27:36.276 # User requested shutdown...\n","stream":"stdout"}`)},
			stream:        "all",
			expectedError: true,
		},
		// CRI log
		{
			input:  [][]byte{[]byte(`2017-09-12T22:32:21.212861448Z stdout 2017-09-12 22:32:21.212 [INFO][88] table.go 710: Invalidating dataplane cache`)},
			stream: "all",
			expectedMessage: Message{
				Content: []byte("2017-09-12 22:32:21.212 [INFO][88] table.go 710: Invalidating dataplane cache"),
				Fields:  common.MapStr{"stream": "stdout"},
				Ts:      time.Date(2017, 9, 12, 22, 32, 21, 212861448, time.UTC),
			},
		},
		// Filtering stream
		{
			input: [][]byte{
				[]byte(`{"log":"filtered\n","stream":"stdout","time":"2017-11-09T13:27:36.277747246Z"}`),
				[]byte(`{"log":"unfiltered\n","stream":"stderr","time":"2017-11-09T13:27:36.277747246Z"}`),
				[]byte(`{"log":"unfiltered\n","stream":"stdout","time":"2017-11-09T13:27:36.277747246Z"}`),
			},
			stream: "stderr",
			expectedMessage: Message{
				Content: []byte("unfiltered\n"),
				Fields:  common.MapStr{"stream": "stderr"},
				Ts:      time.Date(2017, 11, 9, 13, 27, 36, 277747246, time.UTC),
			},
		},
		// Filtering stream
		{
			input: [][]byte{
				[]byte(`2017-10-12T13:32:21.232861448Z stdout 2017-10-12 13:32:21.212 [INFO][88] table.go 710: Invalidating dataplane cache`),
				[]byte(`2017-11-12T23:32:21.212771448Z stderr 2017-11-12 23:32:21.212 [ERROR][77] table.go 111: error`),
				[]byte(`2017-12-12T10:32:21.212864448Z stdout 2017-12-12 10:32:21.212 [WARN][88] table.go 222: Warn`),
			},
			stream: "stderr",
			expectedMessage: Message{
				Content: []byte("2017-11-12 23:32:21.212 [ERROR][77] table.go 111: error"),
				Fields:  common.MapStr{"stream": "stderr"},
				Ts:      time.Date(2017, 11, 12, 23, 32, 21, 212771448, time.UTC),
			},
		},
	}

	for _, test := range tests {
		r := &mockReader{messages: test.input}
		json := NewDockerJSON(r, test.stream)
		message, err := json.Next()

		assert.Equal(t, test.expectedError, err != nil)

		if err == nil {
			assert.EqualValues(t, test.expectedMessage, message)
		}
	}
}

type mockReader struct {
	messages [][]byte
}

func (m *mockReader) Next() (Message, error) {
	message := m.messages[0]
	m.messages = m.messages[1:]
	return Message{
		Content: message,
	}, nil
}
