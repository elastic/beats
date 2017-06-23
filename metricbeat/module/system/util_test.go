// +build !integration
// +build darwin freebsd linux openbsd windows

package system

import (
	"runtime"
	"testing"

	"github.com/elastic/gosigar"
	"github.com/stretchr/testify/assert"
)

func TestCPUMonitorSample(t *testing.T) {
	cpu := &CPUMonitor{lastSample: &gosigar.Cpu{}}
	s, err := cpu.Sample()
	if err != nil {
		t.Fatal(err)
	}

	pct := s.Percentages()
	assert.True(t, pct.User > 0)
	assert.True(t, pct.System > 0)

	normPct := s.NormalizedPercentages()
	assert.True(t, normPct.User > 0)
	assert.True(t, normPct.System > 0)
	assert.True(t, normPct.User <= 100)
	assert.True(t, normPct.System <= 100)

	assert.True(t, pct.User > normPct.User)
	assert.True(t, pct.System > normPct.System)

	ticks := s.Ticks()
	assert.True(t, ticks.User > 0)
	assert.True(t, ticks.System > 0)
}

func TestCPUCoresMonitorSample(t *testing.T) {
	cores := &CPUCoresMonitor{lastSample: make([]gosigar.Cpu, NumCPU)}
	sample, err := cores.Sample()
	if err != nil {
		t.Fatal(err)
	}

	for _, s := range sample {
		normPct := s.Percentages()
		assert.True(t, normPct.User > 0)
		assert.True(t, normPct.User <= 100)
		assert.True(t, normPct.System > 0)
		assert.True(t, normPct.System <= 100)
		assert.True(t, normPct.Idle > 0)
		assert.True(t, normPct.Idle <= 100)

		ticks := s.Ticks()
		assert.True(t, ticks.User > 0)
		assert.True(t, ticks.System > 0)
	}
}

// TestCPUMetricsRounding tests that the returned percentages are rounded to
// four decimal places.
func TestCPUMetricsRounding(t *testing.T) {
	sample := CPUMetrics{
		previousSample: &gosigar.Cpu{
			User: 10855311,
			Sys:  2021040,
			Idle: 17657874,
		},
		currentSample: &gosigar.Cpu{
			User: 10855693,
			Sys:  2021058,
			Idle: 17657876,
		},
	}

	pct := sample.NormalizedPercentages()
	assert.Equal(t, pct.User, 0.9502)
	assert.Equal(t, pct.System, 0.0448)
}

// TestCPUMetricsPercentages tests that CPUMetrics returns the correct
// percentages and normalized percentages.
func TestCPUMetricsPercentages(t *testing.T) {
	NumCPU = 10
	defer func() { NumCPU = runtime.NumCPU() }()

	// This test simulates 30% user and 70% system (normalized), or 3% and 7%
	// respectively when there are 10 CPUs.
	const user, system = 30., 70.

	s0 := gosigar.Cpu{
		User: 10000000,
		Sys:  10000000,
		Idle: 20000000,
		Nice: 0,
	}
	s1 := gosigar.Cpu{
		User: s0.User + uint64(user),
		Sys:  s0.Sys + uint64(system),
		Idle: s0.Idle,
		Nice: 0,
	}
	sample := CPUMetrics{
		previousSample: &s0,
		currentSample:  &s1,
	}

	pct := sample.NormalizedPercentages()
	assert.EqualValues(t, .3, pct.User)
	assert.EqualValues(t, .7, pct.System)

	pct = sample.Percentages()
	assert.EqualValues(t, .3*float64(NumCPU), pct.User)
	assert.EqualValues(t, .7*float64(NumCPU), pct.System)
}

func TestRound(t *testing.T) {
	assert.EqualValues(t, 0.5, Round(0.5))
	assert.EqualValues(t, 0.5, Round(0.50004))
	assert.EqualValues(t, 0.5001, Round(0.50005))

	assert.EqualValues(t, 1234.5, Round(1234.5))
	assert.EqualValues(t, 1234.5, Round(1234.50004))
	assert.EqualValues(t, 1234.5001, Round(1234.50005))
}
