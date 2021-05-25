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

package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common"
)

func TestMonitorSample(t *testing.T) {
	cpu := &Monitor{lastSample: CPUMetrics{}}
	s, err := cpu.Fetch()
	if err != nil {
		t.Fatal(err)
	}
	evt := common.MapStr{}
	s.Percentages(&evt)
	s.NormalizedPercentages(&evt)
	s.Ticks(&evt)
	testPopulatedEvent(evt, t, true)
}

func TestCoresMonitorSample(t *testing.T) {

	cpuMetrics, err := Get("")
	assert.NoError(t, err, "error in Get()")

	cores := &Monitor{lastSample: CPUMetrics{list: make([]CPU, len(cpuMetrics.list))}}
	sample, err := cores.FetchCores()
	if err != nil {
		t.Fatal(err)
	}

	for _, s := range sample {
		evt := common.MapStr{}
		s.Percentages(&evt)
		s.Ticks(&evt)
		testPopulatedEvent(evt, t, false)
	}
}

func testPopulatedEvent(evt common.MapStr, t *testing.T, norm bool) {
	user, err := evt.GetValue("user.pct")
	assert.NoError(t, err, "error getting user.pct")
	system, err := evt.GetValue("system.pct")
	assert.NoError(t, err, "error getting system.pct")
	assert.True(t, user.(float64) > 0)
	assert.True(t, system.(float64) > 0)

	if norm {
		normUser, err := evt.GetValue("user.norm.pct")
		assert.NoError(t, err, "error getting user.norm.pct")
		assert.True(t, normUser.(float64) > 0)
		normSystem, err := evt.GetValue("system.norm.pct")
		assert.NoError(t, err, "error getting system.norm.pct")
		assert.True(t, normSystem.(float64) > 0)
		assert.True(t, normUser.(float64) <= 100)
		assert.True(t, normSystem.(float64) <= 100)

		assert.True(t, user.(float64) > normUser.(float64))
		assert.True(t, system.(float64) > normSystem.(float64))
	}

	userTicks, err := evt.GetValue("user.ticks")
	assert.NoError(t, err, "error getting user.ticks")
	assert.True(t, userTicks.(uint64) > 0)
	systemTicks, err := evt.GetValue("system.ticks")
	assert.NoError(t, err, "error getting system.ticks")
	assert.True(t, systemTicks.(uint64) > 0)
}

// TestMetricsRounding tests that the returned percentages are rounded to
// four decimal places.
func TestMetricsRounding(t *testing.T) {
	makePtr := func(i uint64) *uint64 {
		return &i
	}
	sample := Metrics{
		previousSample: CPU{
			user: makePtr(10855311),
			sys:  makePtr(2021040),
			idle: makePtr(17657874),
		},
		currentSample: CPU{
			user: makePtr(10855693),
			sys:  makePtr(2021058),
			idle: makePtr(17657876),
		},
	}

	evt := common.MapStr{}
	sample.NormalizedPercentages(&evt)

	normUser, err := evt.GetValue("user.norm.pct")
	assert.NoError(t, err, "error getting user.norm.pct")
	normSystem, err := evt.GetValue("system.norm.pct")
	assert.NoError(t, err, "error getting system.norm.pct")

	assert.Equal(t, normUser.(float64), 0.9502)
	assert.Equal(t, normSystem.(float64), 0.0448)
}

// TestMetricsPercentages tests that Metrics returns the correct
// percentages and normalized percentages.
func TestMetricsPercentages(t *testing.T) {
	numCores := 10
	makePtr := func(i uint64) *uint64 {
		return &i
	}
	// This test simulates 30% user and 70% system (normalized), or 3% and 7%
	// respectively when there are 10 CPUs.
	const userTest, systemTest = 30., 70.

	s0 := CPU{
		user: makePtr(10000000),
		sys:  makePtr(10000000),
		idle: makePtr(20000000),
		nice: makePtr(0),
	}
	s1 := CPU{
		user: makePtr(*s0.user + uint64(userTest)),
		sys:  makePtr(*s0.sys + uint64(systemTest)),
		idle: s0.idle,
		nice: makePtr(0),
	}
	sample := Metrics{
		count:          numCores,
		isTotals:       true,
		previousSample: s0,
		currentSample:  s1,
	}

	evt := common.MapStr{}
	sample.NormalizedPercentages(&evt)
	sample.Percentages(&evt)

	user, err := evt.GetValue("user.norm.pct")
	assert.NoError(t, err, "error getting user.norm.pct")
	system, err := evt.GetValue("system.norm.pct")
	assert.NoError(t, err, "error getting system.norm.pct")
	idle, err := evt.GetValue("idle.norm.pct")
	assert.NoError(t, err, "error getting idle.norm.pct")
	total, err := evt.GetValue("total.norm.pct")
	assert.NoError(t, err, "error getting total.norm.pct")
	assert.EqualValues(t, .3, user.(float64))
	assert.EqualValues(t, .7, system.(float64))
	assert.EqualValues(t, .0, idle.(float64))
	assert.EqualValues(t, 1., total.(float64))
}
