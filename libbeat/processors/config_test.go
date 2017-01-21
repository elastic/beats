package processors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractString(t *testing.T) {
	input := "test"

	v, err := extractString(input)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, input, v)
}
