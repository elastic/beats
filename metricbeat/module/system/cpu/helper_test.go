// +build !integration
// +build darwin freebsd linux openbsd windows

package cpu

import (
	"testing"

	"github.com/elastic/gosigar"
	"github.com/stretchr/testify/assert"
)

func TestGetCpuTimes(t *testing.T) {

	cpu_stat, err := GetCpuTimes()

	assert.NotNil(t, cpu_stat)
	assert.Nil(t, err)

	assert.True(t, (cpu_stat.User > 0))
	assert.True(t, (cpu_stat.Sys > 0))

}

func TestCpuPercentage(t *testing.T) {

	cpu := CPU{}

	cpu1 := CpuTimes{
		Cpu: gosigar.Cpu{
			User:    10855311,
			Nice:    0,
			Sys:     2021040,
			Idle:    17657874,
			Wait:    0,
			Irq:     0,
			SoftIrq: 0,
			Stolen:  0,
		},
	}

	cpu.AddCpuPercentage(&cpu1)

	assert.Equal(t, cpu1.UserPercent, 0.0)
	assert.Equal(t, cpu1.SystemPercent, 0.0)

	cpu2 := CpuTimes{
		Cpu: gosigar.Cpu{
			User:    10855693,
			Nice:    0,
			Sys:     2021058,
			Idle:    17657876,
			Wait:    0,
			Irq:     0,
			SoftIrq: 0,
			Stolen:  0,
		},
	}

	cpu.AddCpuPercentage(&cpu2)

	assert.Equal(t, cpu2.UserPercent, 0.9502)
	assert.Equal(t, cpu2.SystemPercent, 0.0448)
}
