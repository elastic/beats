package cfgtype

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnpack(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		expected ByteSize
	}{
		{
			name:     "friendly human value",
			s:        "1KiB",
			expected: ByteSize(1024),
		},
		{
			name:     "raw bytes",
			s:        "2024",
			expected: ByteSize(2024),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := ByteSize(0)
			err := s.Unpack(test.s)
			if !assert.NoError(t, err) {
				return
			}
			assert.Equal(t, test.expected, s)
		})
	}
}
