// +build !integration

package input

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetOSFileState(t *testing.T) {
	file, err := ioutil.TempFile("", "")
	assert.Nil(t, err)

	fileinfo, err := file.Stat()
	assert.Nil(t, err)

	state := GetOSFileState(fileinfo)

	assert.True(t, state.IdxHi > 0)
	assert.True(t, state.IdxLo > 0)
	assert.True(t, state.Vol > 0)
}

func TestGetOSFileStateStat(t *testing.T) {
	file, err := ioutil.TempFile("", "")
	assert.Nil(t, err)

	fileinfo, err := os.Stat(file.Name())
	assert.Nil(t, err)

	state := GetOSFileState(fileinfo)

	assert.True(t, state.IdxHi > 0)
	assert.True(t, state.IdxLo > 0)
	assert.True(t, state.Vol > 0)
}
