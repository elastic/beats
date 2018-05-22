package asset

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetFields(t *testing.T) {

	data := "hello world"
	d, err := EncodeData(data)
	assert.NoError(t, err)

	f := func() string {
		return d
	}

	SetFields("test", f)
	newData, err := GetFields("test")
	assert.NoError(t, err)
	assert.Equal(t, data, string(newData))
}
