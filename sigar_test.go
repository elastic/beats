package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetSystemLoad(t *testing.T) {
	load, err := GetSystemLoad()

	assert.Nil(t, err)
	assert.NotNil(t, load)
	assert.True(t, (load.Load1 > 0))
	assert.True(t, (load.Load5 > 0))
	assert.True(t, (load.Load15 > 0))
}

func TestGetMemory(t *testing.T) {
	mem, err := GetMemory()

	assert.Nil(t, err)
	assert.NotNil(t, mem)

	assert.True(t, (mem.Total > 0))
	assert.True(t, (mem.Used > 0))
	assert.True(t, (mem.Free >= 0))
	assert.True(t, (mem.ActualFree >= 0))
	assert.True(t, (mem.ActualUsed > 0))
}
