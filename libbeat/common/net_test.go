// +build !integration

package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsLoopback(t *testing.T) {
	check, err := IsLoopback("127.0.0.1")

	assert.Nil(t, err)
	assert.True(t, check)
}

func TestIsLoopback_false(t *testing.T) {
	check, err := IsLoopback("192.168.1.1")
	assert.Nil(t, err)
	assert.False(t, check)
}

func TestIsLoopback_error(t *testing.T) {
	check, err := IsLoopback("19216811")
	assert.Error(t, err)
	assert.False(t, check)
}
