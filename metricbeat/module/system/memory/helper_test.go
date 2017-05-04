// +build !integration
// +build darwin freebsd linux openbsd windows

package memory

import (
	"runtime"
	"testing"

	"github.com/shirou/gopsutil/mem"
	"github.com/stretchr/testify/assert"
)

func TestGetMemory(t *testing.T) {
	stat, err := mem.VirtualMemory()

	assert.NotNil(t, stat)
	assert.Nil(t, err)

	assert.True(t, (stat.Total > 0))
	assert.True(t, (stat.Used > 0))
	assert.True(t, (stat.Free >= 0))
	assert.True(t, (stat.Available >= 0))
}

func TestGetSwap(t *testing.T) {

	if runtime.GOOS == "windows" {
		return //no swap data on windows
	}

	swap, err := mem.SwapMemory()

	assert.NotNil(t, swap)
	assert.Nil(t, err)

	assert.True(t, (swap.Total >= 0))
	assert.True(t, (swap.Used >= 0))
	assert.True(t, (swap.Free >= 0))
}

func TestGetPercentage(t *testing.T) {

	assert.Equal(t, GetPercentage(5, 7), 0.7143)
	assert.Equal(t, GetPercentage(5, 0), 0.0)
}
