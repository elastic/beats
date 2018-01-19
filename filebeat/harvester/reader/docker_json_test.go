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
		// Missing time
		{
			input:         [][]byte{[]byte(`{"log":"1:M 09 Nov 13:27:36.276 # User requested shutdown...\n","stream":"stdout"}`)},
			stream:        "all",
			expectedError: true,
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
