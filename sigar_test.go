package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetSystemLoad(t *testing.T) {
	load, err := GetSystemLoad()

	assert.Nil(t, err)
	assert.NotNil(t, load)
	assert.True(t, (0 < load.Load1))
	assert.True(t, (0 < load.Load5))
	assert.True(t, (0 < load.Load15))
}
