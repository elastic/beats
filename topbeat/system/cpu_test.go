// +build !integration

package system

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCpuTimes(t *testing.T) {

	cpu_stat, err := GetCpuTimes()

	assert.NotNil(t, cpu_stat)
	assert.Nil(t, err)

	assert.True(t, (cpu_stat.User > 0))
	assert.True(t, (cpu_stat.Sys > 0))

}
