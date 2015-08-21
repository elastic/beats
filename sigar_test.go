package main

import (
	"runtime"
	"testing"
	"time"

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
	assert.True(t, (swap.Total >= 0))
}

func TestPids(t *testing.T) {
	pids, err := Pids()

	assert.NotNil(t, pids)
	assert.Nil(t, err)

	// Assuming at least 2 processes are running
	assert.True(t, (len(pids) > 1))
}

func TestGetProcess(t *testing.T) {
	pids, err := Pids()

	assert.Nil(t, err)

	process, err := GetProcess(pids[len(pids)-1])

	assert.NotNil(t, process)
	assert.Nil(t, err)

	assert.True(t, (process.Pid > 0))
	assert.True(t, (process.Ppid >= 0))
	assert.True(t, (len(process.Name) > 0))
	assert.NotEqual(t, "unknown", process.State)

	// Memory Checks
	assert.True(t, (process.Mem.Size >= 0))
	assert.True(t, (process.Mem.Resident >= 0))
	assert.True(t, (process.Mem.Share >= 0))

	// CPU Checks
	assert.True(t, (len(process.Cpu.Start) > 0))
	assert.True(t, (process.Cpu.Total >= 0))
	assert.True(t, (process.Cpu.User >= 0))
	assert.True(t, (process.Cpu.System >= 0))

	assert.True(t, (process.lastCPUTime.Unix() <= time.Now().Unix()))
}
