// +build !integration
// +build darwin freebsd linux openbsd windows

package cpu

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/gosigar"
)

func TestMonitorSample(t *testing.T) {
	cpu := &Monitor{lastSample: &gosigar.Cpu{}}
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

func TestCoresMonitorSample(t *testing.T) {
	cores := &CoresMonitor{lastSample: make([]gosigar.Cpu, NumCores)}
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
		assert.True(t, normPct.Total > 0)
		assert.True(t, normPct.Total <= 100)

		ticks := s.Ticks()
		assert.True(t, ticks.User > 0)
		assert.True(t, ticks.System > 0)
	}
}

// TestMetricsRounding tests that the returned percentages are rounded to
// four decimal places.
func TestMetricsRounding(t *testing.T) {
	sample := Metrics{
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

// TestMetricsPercentages tests that Metrics returns the correct
// percentages and normalized percentages.
func TestMetricsPercentages(t *testing.T) {
	NumCores = 10
	defer func() { NumCores = runtime.NumCPU() }()

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
	sample := Metrics{
		previousSample: &s0,
		currentSample:  &s1,
	}

	pct := sample.NormalizedPercentages()
	assert.EqualValues(t, .3, pct.User)
	assert.EqualValues(t, .7, pct.System)
	assert.EqualValues(t, .0, pct.Idle)
	assert.EqualValues(t, 1., pct.Total)

	pct = sample.Percentages()
	assert.EqualValues(t, .3*float64(NumCores), pct.User)
	assert.EqualValues(t, .7*float64(NumCores), pct.System)
	assert.EqualValues(t, .0*float64(NumCores), pct.Idle)
	assert.EqualValues(t, 1.*float64(NumCores), pct.Total)
}
