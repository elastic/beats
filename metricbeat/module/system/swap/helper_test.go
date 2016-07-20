// +build !integration
// +build darwin freebsd linux openbsd windows

package swap

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetSwap(t *testing.T) {

	if runtime.GOOS == "windows" {
		return //no swap data on windows
	}

	swap, err := GetSwap()

	assert.NotNil(t, swap)
	assert.Nil(t, err)

	assert.True(t, (swap.Total >= 0))
	assert.True(t, (swap.Used >= 0))
	assert.True(t, (swap.Free >= 0))
}
