package reader

import (
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"

	"github.com/stretchr/testify/assert"
)

func TestDockerJSON(t *testing.T) {
	tests := []struct {
		input           []byte
		expectedError   bool
		expectedMessage Message
	}{
		// Common log message
		{
			input: []byte(`{"log":"1:M 09 Nov 13:27:36.276 # User requested shutdown...\n","stream":"stdout","time":"2017-11-09T13:27:36.277747246Z"}`),
			expectedMessage: Message{
				Content: []byte("1:M 09 Nov 13:27:36.276 # User requested shutdown...\n"),
				Fields:  common.MapStr{"stream": "stdout"},
				Ts:      time.Date(2017, 11, 9, 13, 27, 36, 277747246, time.UTC),
			},
		},
		// Wrong JSON
		{
			input:         []byte(`this is not JSON`),
			expectedError: true,
		},
		// Missing time
		{
			input:         []byte(`{"log":"1:M 09 Nov 13:27:36.276 # User requested shutdown...\n","stream":"stdout"}`),
			expectedError: true,
		},
	}

	for _, test := range tests {
		r := mockReader{message: test.input}
		json := NewDockerJSON(r)
		message, err := json.Next()

		assert.Equal(t, test.expectedError, err != nil)

		if !test.expectedError {
			assert.EqualValues(t, test.expectedMessage, message)
		}
	}
}

type mockReader struct {
	message []byte
}

func (m mockReader) Next() (Message, error) {
	return Message{
		Content: m.message,
	}, nil
}
