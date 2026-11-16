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
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/opt"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/metricbeat/module/docker"
	"github.com/elastic/beats/v7/pkg/mapstr"
)

func cpuUsageFor(stats container.StatsResponse) *CPUUsage {
	return &CPUUsage{Stat: &docker.Stat{Stats: stats}}
}

func TestCPUUsageCalculatePercentages(t *testing.T) {
	tests := []struct {
		name               string
		oldValue           uint64
		newValue           uint64
		oldSystemUsage     uint64
		newSystemUsage     uint64
		expectedPercentage float64
		expectedNormalized float64
		expectedExists     bool
		// otherReset mutates the sample, e.g. to reset a counter other than
		// the one under test.
		otherReset func(*container.StatsResponse)
	}{
		{
			name:               "valid delta",
			oldValue:           50,
			newValue:           500000050,
			oldSystemUsage:     100,
			newSystemUsage:     1000000100,
			expectedPercentage: 2,
			expectedNormalized: 0.5,
			expectedExists:     true,
		},
		{
			name:               "zero counter delta",
			oldValue:           10,
			newValue:           10,
			oldSystemUsage:     100,
			newSystemUsage:     1000000100,
			expectedPercentage: 0,
			expectedNormalized: 0,
			expectedExists:     true,
		},
		{
			name:               "zero system delta",
			oldValue:           10,
			newValue:           20,
			oldSystemUsage:     100,
			newSystemUsage:     100,
			expectedPercentage: 0,
			expectedNormalized: 0,
			expectedExists:     true,
		},
		{
			name:               "zero counter delta during system reset",
			oldValue:           10,
			newValue:           10,
			oldSystemUsage:     200,
			newSystemUsage:     100,
			expectedPercentage: 0,
			expectedNormalized: 0,
			expectedExists:     false,
		},
		{
			name:           "counter reset",
			oldValue:       100,
			newValue:       2,
			oldSystemUsage: 100,
			newSystemUsage: 1000000100,
		},
		{
			name:           "system counter reset",
			oldValue:       10,
			newValue:       20,
			oldSystemUsage: 200,
			newSystemUsage: 100,
		},
		{
			name:           "zero counter delta during total counter reset",
			oldValue:       10,
			newValue:       10,
			oldSystemUsage: 100,
			newSystemUsage: 1000000100,
			otherReset: func(s *container.StatsResponse) {
				s.CPUStats.CPUUsage.TotalUsage = 1
				s.PreCPUStats.CPUUsage.TotalUsage = 5
			},
		},
		{
			name:           "increased counter during kernel counter reset",
			oldValue:       50,
			newValue:       500000050,
			oldSystemUsage: 100,
			newSystemUsage: 1000000100,
			otherReset: func(s *container.StatsResponse) {
				s.CPUStats.CPUUsage.UsageInKernelmode = 1
				s.PreCPUStats.CPUUsage.UsageInKernelmode = 5
			},
		},
		{
			name:           "increased counter during user counter reset",
			oldValue:       50,
			newValue:       500000050,
			oldSystemUsage: 100,
			newSystemUsage: 1000000100,
			otherReset: func(s *container.StatsResponse) {
				s.CPUStats.CPUUsage.UsageInUsermode = 1
				s.PreCPUStats.CPUUsage.UsageInUsermode = 5
			},
		},
		{
			name:           "increased counter during per-CPU counter reset",
			oldValue:       50,
			newValue:       500000050,
			oldSystemUsage: 100,
			newSystemUsage: 1000000100,
			otherReset: func(s *container.StatsResponse) {
				s.CPUStats.CPUUsage.PercpuUsage = []uint64{10, 1}
				s.PreCPUStats.CPUUsage.PercpuUsage = []uint64{5, 5}
			},
		},
		{
			name:               "per-CPU length mismatch is not a reset",
			oldValue:           50,
			newValue:           500000050,
			oldSystemUsage:     100,
			newSystemUsage:     1000000100,
			expectedPercentage: 2,
			expectedNormalized: 0.5,
			expectedExists:     true,
			otherReset: func(s *container.StatsResponse) {
				s.CPUStats.CPUUsage.PercpuUsage = []uint64{1}
				s.PreCPUStats.CPUUsage.PercpuUsage = []uint64{5, 5}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stats := container.StatsResponse{
				PreCPUStats: container.CPUStats{SystemUsage: test.oldSystemUsage},
				CPUStats: container.CPUStats{
					SystemUsage: test.newSystemUsage,
					OnlineCPUs:  4,
				},
			}
			if test.otherReset != nil {
				test.otherReset(&stats)
			}
			usage := cpuUsageFor(stats)

			percentage, normalized := usage.calculatePercentages(test.newValue, test.oldValue)
			assertOptionalFloat(t, percentage, test.expectedPercentage, test.expectedExists, "percentage")
			assertOptionalFloat(t, normalized, test.expectedNormalized, test.expectedExists, "normalized percentage")
		})
	}
}

func TestCPUServiceSkipsEmptySamples(t *testing.T) {
	noRead := validStatsResponse()
	noRead.Read = time.Time{}
	noPreRead := validStatsResponse()
	noPreRead.PreRead = time.Time{}

	tests := []struct {
		name  string
		stats container.StatsResponse
	}{
		{name: "empty current sample", stats: noRead},
		{name: "empty previous sample", stats: noPreRead},
		{name: "both empty", stats: container.StatsResponse{
			PreCPUStats: container.CPUStats{CPUUsage: container.CPUUsage{TotalUsage: 100}},
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := CPUService{}
			rawStats := []docker.Stat{*newDockerStat(test.stats), *newDockerStat(validStatsResponse())}

			cpuStats := service.getCPUStatsList(rawStats, false)
			require.Len(t, cpuStats, 1, "sample with a missing read timestamp must be skipped")
			assert.Equal(t, common.Time(validStatsResponse().Read), cpuStats[0].Time, "valid stats response must be retained")
		})
	}
}

func TestCPUUsageSystemDeltaReset(t *testing.T) {
	usage := cpuUsageFor(container.StatsResponse{
		PreCPUStats: container.CPUStats{SystemUsage: 200},
		CPUStats:    container.CPUStats{SystemUsage: 100, OnlineCPUs: 2},
	})

	assert.Equal(t, uint64(0), usage.SystemDelta(), "system reset must not unsigned-underflow")
	assert.False(t, usage.System().Exists(), "system percentage must be absent after a system reset")
	assert.False(t, usage.SystemNormalized().Exists(), "normalized system percentage must be absent after a system reset")
}

func TestCPUServiceCounterResetOutput(t *testing.T) {
	// Every case resets a single counter; the whole sample must be
	// invalidated, including counters that increased.
	tests := []struct {
		name  string
		reset func(*container.StatsResponse)
	}{
		{"total usage", func(s *container.StatsResponse) { s.PreCPUStats.CPUUsage.TotalUsage = 300 }},
		{"kernel usage", func(s *container.StatsResponse) { s.PreCPUStats.CPUUsage.UsageInKernelmode = 100 }},
		{"user usage", func(s *container.StatsResponse) { s.PreCPUStats.CPUUsage.UsageInUsermode = 100 }},
		{"system usage", func(s *container.StatsResponse) { s.PreCPUStats.SystemUsage = 3000 }},
		{"per-CPU usage", func(s *container.StatsResponse) { s.PreCPUStats.CPUUsage.PercpuUsage = []uint64{200, 20} }},
	}

	expectedPerCPU := mapstr.M{
		"0": mapstr.M{"ticks": uint64(150)},
		"1": mapstr.M{"ticks": uint64(120)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stats := validStatsResponse()
			stats.PreCPUStats.CPUUsage.PercpuUsage = []uint64{100, 20}
			stats.CPUStats.CPUUsage.PercpuUsage = []uint64{150, 120}
			test.reset(&stats)

			service := CPUService{Cores: true}
			cpuStats := service.getCPUStats(newDockerStat(stats), false)

			assert.Equal(t, uint64(20), cpuStats.UsageInKernelmode, "kernel ticks must retain the current cumulative counter")
			assert.Equal(t, uint64(30), cpuStats.UsageInUsermode, "user ticks must retain the current cumulative counter")
			assert.Equal(t, uint64(2000), cpuStats.SystemUsage, "system ticks must retain the current cumulative counter")
			assert.False(t, cpuStats.TotalUsage.Exists(), "total percentage must be absent after a reset")
			assert.False(t, cpuStats.TotalUsageNormalized.Exists(), "normalized total percentage must be absent after a reset")
			assert.False(t, cpuStats.UsageInKernelmodePercentage.Exists(), "kernel percentage must be absent after a reset")
			assert.False(t, cpuStats.UsageInKernelmodePercentageNormalized.Exists(), "normalized kernel percentage must be absent after a reset")
			assert.False(t, cpuStats.UsageInUsermodePercentage.Exists(), "user percentage must be absent after a reset")
			assert.False(t, cpuStats.UsageInUsermodePercentageNormalized.Exists(), "normalized user percentage must be absent after a reset")
			assert.False(t, cpuStats.SystemUsagePercentage.Exists(), "system percentage must be absent after a reset")
			assert.False(t, cpuStats.SystemUsagePercentageNormalized.Exists(), "normalized system percentage must be absent after a reset")
			assert.Equal(t, expectedPerCPU, cpuStats.PerCPUUsage, "per-core resets must retain ticks and omit percentages")

			reporter := &mbtest.CapturingReporterV2{}
			eventMapping(reporter, &cpuStats)
			events := reporter.GetEvents()
			require.Len(t, events, 1, "one CPU event must be published")

			fields := events[0].MetricSetFields
			assert.NotContains(t, fields, "total", "unavailable total percentages must not create an empty field")
			assert.Equal(t, mapstr.M{"ticks": uint64(20)}, fields["kernel"], "kernel must only contain ticks")
			assert.Equal(t, mapstr.M{"ticks": uint64(30)}, fields["user"], "user must only contain ticks")
			assert.Equal(t, mapstr.M{"ticks": uint64(2000)}, fields["system"], "system must only contain ticks")
			assert.Equal(t, expectedPerCPU, fields["core"], "per-core event output must only contain ticks")

			containerFields, ok := events[0].RootFields["container"].(mapstr.M)
			require.True(t, ok, "container root fields must be a map")
			assert.NotContains(t, containerFields, "cpu", "container.cpu.usage must be omitted after a reset")
		})
	}
}

func TestCPUServiceValidSampleOutput(t *testing.T) {
	stats := validStatsResponse()
	stats.PreCPUStats.CPUUsage.PercpuUsage = []uint64{100, 20}
	stats.CPUStats.CPUUsage.PercpuUsage = []uint64{150, 120}

	service := CPUService{Cores: true}
	cpuStats := service.getCPUStats(newDockerStat(stats), false)

	assertOptionalFloat(t, cpuStats.TotalUsage, 0.2, true, "total percentage")
	assertOptionalFloat(t, cpuStats.TotalUsageNormalized, 0.1, true, "normalized total percentage")
	assertOptionalFloat(t, cpuStats.UsageInKernelmodePercentage, 0.02, true, "kernel percentage")
	assertOptionalFloat(t, cpuStats.UsageInKernelmodePercentageNormalized, 0.01, true, "normalized kernel percentage")
	assertOptionalFloat(t, cpuStats.UsageInUsermodePercentage, 0.04, true, "user percentage")
	assertOptionalFloat(t, cpuStats.UsageInUsermodePercentageNormalized, 0.02, true, "normalized user percentage")
	assertOptionalFloat(t, cpuStats.SystemUsagePercentage, 2, true, "system percentage")
	assertOptionalFloat(t, cpuStats.SystemUsagePercentageNormalized, 1, true, "normalized system percentage")
	assert.Equal(t, mapstr.M{
		"ticks": uint64(150),
		"pct":   float64(0.1),
		"norm":  mapstr.M{"pct": float64(0.05)},
	}, cpuStats.PerCPUUsage["0"], "per-core percentages must be present for a valid sample")
}

func TestEventMappingOmitsUnavailableTotalPercentage(t *testing.T) {
	stats := validStatsResponse()
	stats.PreCPUStats.CPUUsage.TotalUsage = 100
	stats.CPUStats.CPUUsage.TotalUsage = 50

	service := CPUService{}
	cpuStats := service.getCPUStats(newDockerStat(stats), false)
	assert.False(t, cpuStats.TotalUsage.Exists(), "total percentage must be absent after a reset")
	assert.False(t, cpuStats.TotalUsageNormalized.Exists(), "normalized total percentage must be absent after a reset")

	reporter := &mbtest.CapturingReporterV2{}
	eventMapping(reporter, &cpuStats)
	events := reporter.GetEvents()
	require.Len(t, events, 1, "one CPU event must be published")
	assert.NotContains(t, events[0].MetricSetFields, "total", "unavailable total percentages must not create an empty field")

	containerFields, ok := events[0].RootFields["container"].(mapstr.M)
	require.True(t, ok, "container root fields must be a map")
	assert.NotContains(t, containerFields, "cpu", "container.cpu.usage must be omitted when total percentage is unavailable")
}

func validStatsResponse() container.StatsResponse {
	return container.StatsResponse{
		Read:    time.Unix(1700000000, 0),
		PreRead: time.Unix(1699999999, 0),
		PreCPUStats: container.CPUStats{
			SystemUsage: 1000,
			CPUUsage: container.CPUUsage{
				TotalUsage:        100,
				UsageInKernelmode: 10,
				UsageInUsermode:   10,
			},
		},
		CPUStats: container.CPUStats{
			SystemUsage: 2000,
			OnlineCPUs:  2,
			CPUUsage: container.CPUUsage{
				TotalUsage:        200,
				UsageInKernelmode: 20,
				UsageInUsermode:   30,
			},
		},
	}
}

func newDockerStat(stats container.StatsResponse) *docker.Stat {
	return &docker.Stat{
		Container: &container.Summary{
			ID:     "container-id",
			Image:  "image",
			Names:  []string{"/container-name"},
			Labels: map[string]string{},
		},
		Stats: stats,
	}
}

func assertOptionalFloat(t *testing.T, actual opt.Float, expected float64, expectedExists bool, field string) {
	t.Helper()
	require.Equal(t, expectedExists, actual.Exists(), "%s presence", field)
	if expectedExists {
		assert.InDelta(t, expected, actual.ValueOr(0), 1e-12, "%s value", field)
	}
}
