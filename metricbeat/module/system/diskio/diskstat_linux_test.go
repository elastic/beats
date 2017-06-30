// +build linux

package diskio

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Get_CLK_TCK(t *testing.T) {
	//usually the tick is 100
	assert.Equal(t, uint32(100), Get_CLK_TCK())
}
