// +build !integration
// +build darwin freebsd linux openbsd windows

package load

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetSystemLoad(t *testing.T) {

	if runtime.GOOS == "windows" {
		return //no load data on windows
	}

	load, err := GetSystemLoad()

	assert.NotNil(t, load)
	assert.Nil(t, err)

	assert.True(t, (load.Load1 > 0))
	assert.True(t, (load.Load5 > 0))
	assert.True(t, (load.Load15 > 0))
}
