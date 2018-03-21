package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTryToInt(t *testing.T) {
	tests := []struct {
		input   interface{}
		result  int
		resultB bool
	}{
		{
			int(4),
			int(4),
			true,
		},
		{
			int64(3),
			int(3),
			true,
		},
		{
			"5",
			int(5),
			true,
		},
		{
			uint32(12),
			int(12),
			true,
		},
		{
			"abc",
			0,
			false,
		},
		{
			[]string{"123"},
			0,
			false,
		},
		{
			uint64(55),
			int(55),
			true,
		},
	}

	for _, test := range tests {
		a, b := TryToInt(test.input)
		assert.Equal(t, a, test.result)
		assert.Equal(t, b, test.resultB)
	}
}
