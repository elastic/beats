// +build !integration
// +build darwin freebsd linux openbsd windows

package memory

import (
	"runtime"
	"testing"

	"github.com/elastic/gosigar"
	"github.com/stretchr/testify/assert"
)

func TestGetMemory(t *testing.T) {
	mem, err := GetMemory()

	assert.NotNil(t, mem)
	assert.Nil(t, err)

	assert.True(t, (mem.Total > 0))
	assert.True(t, (mem.Used > 0))
	assert.True(t, (mem.Free >= 0))
	assert.True(t, (mem.ActualFree >= 0))
	assert.True(t, (mem.ActualUsed > 0))
}

func TestGetSwap(t *testing.T) {

	if runtime.GOOS == "windows" {
		return //no load data on windows
	}

	swap, err := GetSwap()

	assert.NotNil(t, swap)
	assert.Nil(t, err)

	assert.True(t, (swap.Total >= 0))
	assert.True(t, (swap.Used >= 0))
	assert.True(t, (swap.Free >= 0))
}

func TestMemPercentage(t *testing.T) {

	m := MemStat{
		Mem: gosigar.Mem{
			Total: 7,
			Used:  5,
			Free:  2,
		},
	}
	AddMemPercentage(&m)
	assert.Equal(t, m.UsedPercent, 0.7143)

	m = MemStat{
		Mem: gosigar.Mem{Total: 0},
	}
	AddMemPercentage(&m)
	assert.Equal(t, m.UsedPercent, 0.0)
}

func TestActualMemPercentage(t *testing.T) {

	m := MemStat{
		Mem: gosigar.Mem{
			Total:      7,
			ActualUsed: 5,
			ActualFree: 2,
		},
	}
	AddMemPercentage(&m)
	assert.Equal(t, m.ActualUsedPercent, 0.7143)

	m = MemStat{
		Mem: gosigar.Mem{
			Total: 0,
		},
	}
	AddMemPercentage(&m)
	assert.Equal(t, m.ActualUsedPercent, 0.0)
}
