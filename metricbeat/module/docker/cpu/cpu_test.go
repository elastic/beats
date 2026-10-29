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
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/metricbeat/module/docker"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/opt"
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
			expectedExists:     true,
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
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			usage := cpuUsageFor(container.StatsResponse{
				PreCPUStats: container.CPUStats{SystemUsage: test.oldSystemUsage},
				CPUStats: container.CPUStats{
					SystemUsage: test.newSystemUsage,
					OnlineCPUs:  4,
				},
			})

			percentage, normalized := usage.calculatePercentages(test.newValue, test.oldValue)
			assertOptionalFloat(t, percentage, test.expectedPercentage, test.expectedExists, "percentage")
			assertOptionalFloat(t, normalized, test.expectedNormalized, test.expectedExists, "normalized percentage")
		})
	}
}

func TestCPUServiceSkipsEmptyCurrentSample(t *testing.T) {
	service := CPUService{}
	rawStats := []docker.Stat{{
		Stats: container.StatsResponse{
			PreCPUStats: container.CPUStats{
				CPUUsage: container.CPUUsage{TotalUsage: 100},
			},
		},
	}, *newDockerStat(validStatsResponse())}

	cpuStats := service.getCPUStatsList(rawStats, false)
	require.Len(t, cpuStats, 1, "empty current stats response must be skipped")
	assert.Equal(t, common.Time(validStatsResponse().Read), cpuStats[0].Time, "valid stats response must be retained")
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
	stats := validStatsResponse()
	stats.PreCPUStats.CPUUsage.UsageInKernelmode = 100
	stats.CPUStats.CPUUsage.UsageInKernelmode = 50
	stats.PreCPUStats.CPUUsage.PercpuUsage = []uint64{100, 20}
	stats.CPUStats.CPUUsage.PercpuUsage = []uint64{50, 120}

	service := CPUService{Cores: true}
	cpuStats := service.getCPUStats(newDockerStat(stats), false)

	assert.Equal(t, uint64(50), cpuStats.UsageInKernelmode, "kernel ticks must retain the current cumulative counter")
	assert.False(t, cpuStats.UsageInKernelmodePercentage.Exists(), "kernel percentage must be absent after a reset")
	assert.False(t, cpuStats.UsageInKernelmodePercentageNormalized.Exists(), "normalized kernel percentage must be absent after a reset")
	assertOptionalFloat(t, cpuStats.TotalUsage, 0.2, true, "total percentage")
	assertOptionalFloat(t, cpuStats.TotalUsageNormalized, 0.1, true, "normalized total percentage")
	assertOptionalFloat(t, cpuStats.UsageInUsermodePercentage, 0.04, true, "user percentage")
	assertOptionalFloat(t, cpuStats.UsageInUsermodePercentageNormalized, 0.02, true, "normalized user percentage")
	assertOptionalFloat(t, cpuStats.SystemUsagePercentage, 2, true, "system percentage")
	assertOptionalFloat(t, cpuStats.SystemUsagePercentageNormalized, 1, true, "normalized system percentage")

	expectedPerCPU := mapstr.M{
		"0": mapstr.M{"ticks": uint64(50)},
		"1": mapstr.M{
			"ticks": uint64(120),
			"pct":   float64(0.2),
			"norm":  mapstr.M{"pct": float64(0.1)},
		},
	}
	assert.Equal(t, expectedPerCPU, cpuStats.PerCPUUsage, "per-core resets must retain ticks and omit percentages")

	reporter := &mbtest.CapturingReporterV2{}
	eventMapping(reporter, &cpuStats)
	events := reporter.GetEvents()
	require.Len(t, events, 1, "one CPU event must be published")

	kernel, ok := events[0].MetricSetFields["kernel"].(mapstr.M)
	require.True(t, ok, "kernel fields must be a map")
	assert.Equal(t, mapstr.M{"ticks": uint64(50)}, kernel, "reset kernel counter must omit percentage fields")

	core, ok := events[0].MetricSetFields["core"].(mapstr.M)
	require.True(t, ok, "core fields must be a map")
	assert.Equal(t, expectedPerCPU, core, "per-core event output must omit reset percentages")

	containerFields, ok := events[0].RootFields["container"].(mapstr.M)
	require.True(t, ok, "container root fields must be a map")
	cpuFields, ok := containerFields["cpu"].(mapstr.M)
	require.True(t, ok, "container CPU fields must be a map")
	assert.InDelta(t, 0.1, cpuFields["usage"], 1e-12, "container CPU usage must retain a valid normalized total")
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
		Read: time.Unix(1700000000, 0),
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
