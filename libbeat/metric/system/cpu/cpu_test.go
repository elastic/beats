// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

// +build !integration
// +build darwin freebsd linux openbsd windows

package cpu

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/gosigar"
)

var (
	// numCores is the number of CPU cores in the system. Changes to operating
	// system CPU allocation after process startup are not reflected.
	numCores = runtime.NumCPU()
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
	cores := &CoresMonitor{lastSample: make([]gosigar.Cpu, numCores)}
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
	numCores = 10
	defer func() { numCores = runtime.NumCPU() }()

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

	//bypass the Metrics API so we can have a constant CPU value
	pct = cpuPercentages(&s0, &s1, numCores)
	assert.EqualValues(t, .3*float64(numCores), pct.User)
	assert.EqualValues(t, .7*float64(numCores), pct.System)
	assert.EqualValues(t, .0*float64(numCores), pct.Idle)
	assert.EqualValues(t, 1.*float64(numCores), pct.Total)
}
