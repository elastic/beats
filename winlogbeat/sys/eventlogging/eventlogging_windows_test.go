package eventlogging

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaxParam(t *testing.T) {
	for _, data := range []struct {
		s   string
		max int
	}{
		{"", 0},
		{"0", 0},
		{"hello %0", 0},
		{"The service %1 failed to %2", 2},
		{"%3 %2 %1", 3},
		{"%", 0},
		{"8%%%33%0%%99", 33},
		{"The program %1 %6 on host %7 failed to %2 on %3 %4 due to %5", 7},
		{"%12345", 12345},
	} {
		max, err := getMaxInsertArgument(stringToUTF16Bytes(data.s))
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, data.max, max, data.s)
	}
}

func TestBadMaxParam(t *testing.T) {
	for _, data := range []struct {
		s   []byte
		err string
	}{
		{[]byte{}, "utf16 string is not terminated"},
		{[]byte{0}, "utf16 string has odd length"},
		{[]byte{0, 1}, "utf16 string is not terminated"},
		{[]byte{1, 0}, "utf16 string is not terminated"},
		{[]byte{1, 0, 0}, "utf16 string has odd length"},
	} {
		max, err := getMaxInsertArgument(data.s)
		assert.NotNil(t, err, data.s)
		assert.Equal(t, 0, max)
		assert.Equal(t, data.err, err.Error(), data.s)
	}
}

func stringToUTF16Bytes(s string) []byte {
	n := len(s)
	result := make([]byte, 2*n+2)
	for idx, chr := range s {
		result[idx*2] = byte(chr)
	}
	return result
}
