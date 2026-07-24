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

package cpu

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/opt"
	"github.com/elastic/elastic-agent-system-metrics/dev-tools/systemtests"
)

func TestMonitorSample(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	cpu, err := New(systemtests.DockerTestResolver(logger))
	require.NoError(t, err)
	s, err := cpu.Fetch()
	require.NoError(t, err)

	metricOpts := MetricOpts{Percentages: true, NormalizedPercentages: true, Ticks: true}
	evt, err := s.Format(metricOpts)
	assert.NoError(t, err, "error in Format")
	testPopulatedEvent(evt, t, true)
}

func TestCoresMonitorSample(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	cores, err := New(systemtests.DockerTestResolver(logger))
	require.NoError(t, err)

	cpuMetrics, err := Get(cores)
	assert.NoError(t, err, "error in Get()")

	cores.lastSample = CPUMetrics{list: make([]CPU, len(cpuMetrics.list))}
	sample, err := cores.FetchCores()
	require.NoError(t, err)

	for _, s := range sample {
		metricOpts := MetricOpts{Percentages: true, Ticks: true}
		evt, err := s.Format(metricOpts)
		assert.NoError(t, err, "error in Format")
		testPopulatedEvent(evt, t, false)
	}
}

func testPopulatedEvent(evt mapstr.M, t *testing.T, norm bool) {
	user, err := evt.GetValue("user.pct")
	assert.NoError(t, err, "error getting user.pct")
	system, err := evt.GetValue("system.pct")
	assert.NoError(t, err, "error getting system.pct")
	userPct, ok := user.(float64)
	assert.True(t, ok)
	systemPct, ok := system.(float64)
	assert.True(t, ok)
	assert.Positive(t, userPct)
	assert.Positive(t, systemPct)

	if norm {
		normUser, err := evt.GetValue("user.norm.pct")
		assert.NoError(t, err, "error getting user.norm.pct")
		normUserPct, ok := normUser.(float64)
		assert.True(t, ok)
		normSystem, err := evt.GetValue("system.norm.pct")
		assert.NoError(t, err, "error getting system.norm.pct")
		normSystemPct, ok := normSystem.(float64)
		assert.True(t, ok)
		assert.Positive(t, normUserPct)
		assert.Positive(t, normSystemPct)
		assert.LessOrEqual(t, normUserPct, float64(100))
		assert.LessOrEqual(t, normSystemPct, float64(100))

		assert.Greater(t, userPct, normUserPct)
		assert.Greater(t, systemPct, normSystemPct)
	}

	userTicks, err := evt.GetValue("user.ticks")
	assert.NoError(t, err, "error getting user.ticks")
	userTicksVal, ok := userTicks.(uint64)
	assert.True(t, ok)
	assert.Positive(t, userTicksVal)
	systemTicks, err := evt.GetValue("system.ticks")
	assert.NoError(t, err, "error getting system.ticks")
	systemTicksVal, ok := systemTicks.(uint64)
	assert.True(t, ok)
	assert.Positive(t, systemTicksVal)
}

// TestMetricsRounding tests that the returned percentages are rounded to
// four decimal places.
func TestMetricsRounding(t *testing.T) {

	sample := Metrics{
		previousSample: CPU{
			User: opt.UintWith(10855311),
			Sys:  opt.UintWith(2021040),
			Idle: opt.UintWith(17657874),
		},
		currentSample: CPU{
			User: opt.UintWith(10855693),
			Sys:  opt.UintWith(2021058),
			Idle: opt.UintWith(17657876),
		},
	}

	evt, err := sample.Format(MetricOpts{NormalizedPercentages: true})
	assert.NoError(t, err, "error in Format")
	normUser, err := evt.GetValue("user.norm.pct")
	assert.NoError(t, err, "error getting user.norm.pct")
	normSystem, err := evt.GetValue("system.norm.pct")
	assert.NoError(t, err, "error getting system.norm.pct")

	normUserPct, ok := normUser.(float64)
	assert.True(t, ok)
	normSystemPct, ok := normSystem.(float64)
	assert.True(t, ok)
	assert.InDelta(t, 0.9502, normUserPct, 0.0001)
	assert.InDelta(t, 0.0448, normSystemPct, 0.0001)
}

// TestMetricsPercentages tests that Metrics returns the correct
// percentages and normalized percentages.
func TestMetricsPercentages(t *testing.T) {
	numCores := 10
	// This test simulates 30% user and 70% system (normalized), or 3% and 7%
	// respectively when there are 10 CPUs.
	const userTest, systemTest = 30., 70.

	s0 := CPU{
		User: opt.UintWith(10000000),
		Sys:  opt.UintWith(10000000),
		Idle: opt.UintWith(20000000),
		Nice: opt.UintWith(0),
	}
	s1 := CPU{
		User: opt.UintWith(s0.User.ValueOr(0) + uint64(userTest)),
		Sys:  opt.UintWith(s0.Sys.ValueOr(0) + uint64(systemTest)),
		Idle: s0.Idle,
		Nice: opt.UintWith(0),
	}
	sample := Metrics{
		count:          numCores,
		isTotals:       true,
		previousSample: s0,
		currentSample:  s1,
	}

	evt, err := sample.Format(MetricOpts{NormalizedPercentages: true, Percentages: true})
	assert.NoError(t, err, "error in Format")

	user, err := evt.GetValue("user.norm.pct")
	assert.NoError(t, err, "error getting user.norm.pct")
	system, err := evt.GetValue("system.norm.pct")
	assert.NoError(t, err, "error getting system.norm.pct")
	idle, err := evt.GetValue("idle.norm.pct")
	assert.NoError(t, err, "error getting idle.norm.pct")
	total, err := evt.GetValue("total.norm.pct")
	assert.NoError(t, err, "error getting total.norm.pct")
	userPct, ok := user.(float64)
	assert.True(t, ok)
	systemPct, ok := system.(float64)
	assert.True(t, ok)
	idlePct, ok := idle.(float64)
	assert.True(t, ok)
	totalPct, ok := total.(float64)
	assert.True(t, ok)
	assert.InDelta(t, 0.3, userPct, 0.0001)
	assert.InDelta(t, 0.7, systemPct, 0.0001)
	assert.InDelta(t, 0.0, idlePct, 0.0001)
	assert.InDelta(t, 1.0, totalPct, 0.0001)
}
