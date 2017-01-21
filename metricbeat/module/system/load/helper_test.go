// +build !integration
// +build darwin freebsd linux openbsd

package load

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetSystemLoad(t *testing.T) {

	load, err := GetSystemLoad()

	assert.NotNil(t, load)
	assert.Nil(t, err)

	assert.True(t, (load.Load1 > 0))
	assert.True(t, (load.Load5 > 0))
	assert.True(t, (load.Load15 > 0))
}
