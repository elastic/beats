// +build !windows,!integration

package file

import (
	"io/ioutil"
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetOSFileState(t *testing.T) {
	file, err := ioutil.TempFile("", "")
	assert.Nil(t, err)

	fileinfo, err := file.Stat()
	assert.Nil(t, err)

	state := GetOSState(fileinfo)

	assert.True(t, state.Inode > 0)

	if runtime.GOOS == "openbsd" {
		// The first device on OpenBSD has an ID of 0 so allow this.
		assert.True(t, state.Device >= 0, "Device %d", state.Device)
	} else {
		assert.True(t, state.Device > 0, "Device %d", state.Device)
	}
}

func TestGetOSFileStateStat(t *testing.T) {
	file, err := ioutil.TempFile("", "")
	assert.Nil(t, err)

	fileinfo, err := os.Stat(file.Name())
	assert.Nil(t, err)

	state := GetOSState(fileinfo)

	assert.True(t, state.Inode > 0)

	if runtime.GOOS == "openbsd" {
		// The first device on OpenBSD has an ID of 0 so allow this.
		assert.True(t, state.Device >= 0, "Device %d", state.Device)
	} else {
		assert.True(t, state.Device > 0, "Device %d", state.Device)
	}
}
