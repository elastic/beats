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
	"reflect"
	"testing"

	"github.com/docker/docker/api/types"

	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/metricbeat/module/docker"
)

var cpuService CPUService

func cpuUsageFor(stats types.StatsJSON) *CPUUsage {
	u := CPUUsage{
		Stat:        &docker.Stat{Stats: stats},
		systemDelta: 1000000000, // Nanoseconds in a second
	}
	return &u
}

func TestCPUService_PerCpuUsage(t *testing.T) {
	oldPerCpuValuesTest := [][]uint64{{1, 9, 9, 5}, {1, 2, 3, 4}, {0, 0, 0, 0}}
	newPerCpuValuesTest := [][]uint64{{100000001, 900000009, 900000009, 500000005}, {101, 202, 303, 404}, {0, 0, 0, 0}}
	var statsList = make([]types.StatsJSON, 3)
	var onlineCPUS = uint32(4)
	for index := range statsList {
		statsList[index].PreCPUStats.CPUUsage.PercpuUsage = oldPerCpuValuesTest[index]
		statsList[index].CPUStats.CPUUsage.PercpuUsage = newPerCpuValuesTest[index]
		statsList[index].CPUStats.OnlineCPUs = onlineCPUS
	}
	testCase := []struct {
		given    types.StatsJSON
		expected common.MapStr
	}{
		{statsList[0], common.MapStr{
			"0": common.MapStr{"pct": float64(0.40), "norm": common.MapStr{"pct": float64(0.40) / float64(onlineCPUS)}},
			"1": common.MapStr{"pct": float64(3.60), "norm": common.MapStr{"pct": float64(3.60) / float64(onlineCPUS)}},
			"2": common.MapStr{"pct": float64(3.60), "norm": common.MapStr{"pct": float64(3.60) / float64(onlineCPUS)}},
			"3": common.MapStr{"pct": float64(2.00), "norm": common.MapStr{"pct": float64(2.00) / float64(onlineCPUS)}},
		}},
		{statsList[1], common.MapStr{
			"0": common.MapStr{"pct": float64(0.0000004), "norm": common.MapStr{"pct": float64(0.0000004) / float64(onlineCPUS)}},
			"1": common.MapStr{"pct": float64(0.0000008), "norm": common.MapStr{"pct": float64(0.0000008) / float64(onlineCPUS)}},
			"2": common.MapStr{"pct": float64(0.0000012), "norm": common.MapStr{"pct": float64(0.0000012) / float64(onlineCPUS)}},
			"3": common.MapStr{"pct": float64(0.0000016), "norm": common.MapStr{"pct": float64(0.0000016) / float64(onlineCPUS)}},
		}},
		{statsList[2], common.MapStr{
			"0": common.MapStr{"pct": float64(0), "norm": common.MapStr{"pct": float64(0) / float64(onlineCPUS)}},
			"1": common.MapStr{"pct": float64(0), "norm": common.MapStr{"pct": float64(0) / float64(onlineCPUS)}},
			"2": common.MapStr{"pct": float64(0), "norm": common.MapStr{"pct": float64(0) / float64(onlineCPUS)}},
			"3": common.MapStr{"pct": float64(0), "norm": common.MapStr{"pct": float64(0) / float64(onlineCPUS)}},
		}},
	}
	for _, tt := range testCase {
		usage := cpuUsageFor(tt.given)
		out := usage.PerCPU()
		// Remove ticks for test
		for _, s := range out {
			s.(common.MapStr).Delete("ticks")
		}
		if !equalEvent(tt.expected, out) {
			t.Errorf("PerCPUUsage(%v) => %v, want %v", tt.given.CPUStats.CPUUsage.PercpuUsage, out, tt.expected)
		}
	}
}

func TestCPUService_TotalUsage(t *testing.T) {
	oldTotalValuesTest := []uint64{100, 50, 10}
	totalValuesTest := []uint64{2, 500000050, 10}
	var statsList = make([]types.StatsJSON, 3)
	var onlineCPUS = uint32(4)
	for index := range statsList {
		statsList[index].PreCPUStats.CPUUsage.TotalUsage = oldTotalValuesTest[index]
		statsList[index].CPUStats.CPUUsage.TotalUsage = totalValuesTest[index]
		statsList[index].CPUStats.OnlineCPUs = onlineCPUS
	}
	testCase := []struct {
		given    types.StatsJSON
		expected float64
	}{
		{statsList[0], -1},
		{statsList[1], 2},
		{statsList[2], 0},
	}
	for _, tt := range testCase {
		usage := cpuUsageFor(tt.given)
		out := usage.Total()
		if tt.expected != out {
			t.Errorf("totalUsage(%v) => %v, want %v", tt.given.CPUStats.CPUUsage.TotalUsage, out, tt.expected)
		}
	}
}

func TestCPUService_TotalUsageNormalized(t *testing.T) {
	oldTotalValuesTest := []uint64{100, 50, 10}
	totalValuesTest := []uint64{2, 500000050, 10}
	var statsList = make([]types.StatsJSON, 3)
	var onlineCPUS = uint32(4)
	for index := range statsList {
		statsList[index].PreCPUStats.CPUUsage.TotalUsage = oldTotalValuesTest[index]
		statsList[index].CPUStats.CPUUsage.TotalUsage = totalValuesTest[index]
		statsList[index].CPUStats.OnlineCPUs = onlineCPUS
	}
	testCase := []struct {
		given    types.StatsJSON
		expected float64
	}{
		{statsList[0], -1},
		{statsList[1], 0.5},
		{statsList[2], 0},
	}
	for _, tt := range testCase {
		usage := cpuUsageFor(tt.given)
		out := usage.TotalNormalized()
		if tt.expected != out {
			t.Errorf("totalUsageNormalized(%v) => %v, want %v", tt.given.CPUStats.CPUUsage.TotalUsage, out, tt.expected)
		}
	}
}

func TestCPUService_UsageInKernelmode(t *testing.T) {
	usageOldValuesTest := []uint64{100, 10, 500000050}
	usageValuesTest := []uint64{3, 500000010, 500000050}
	var statsList = make([]types.StatsJSON, 3)
	var onlineCPUS = uint32(4)
	for index := range statsList {
		statsList[index].PreCPUStats.CPUUsage.UsageInKernelmode = usageOldValuesTest[index]
		statsList[index].CPUStats.CPUUsage.UsageInKernelmode = usageValuesTest[index]
		statsList[index].CPUStats.OnlineCPUs = onlineCPUS
	}
	testCase := []struct {
		given    types.StatsJSON
		expected float64
	}{
		{statsList[0], -1},
		{statsList[1], 2},
		{statsList[2], 0},
	}
	for _, tt := range testCase {
		usage := cpuUsageFor(tt.given)
		out := usage.InKernelMode()
		if out != tt.expected {
			t.Errorf("usageInKernelmode(%v) => %v, want %v", tt.given.CPUStats.CPUUsage.UsageInKernelmode, out, tt.expected)
		}
	}
}

func TestCPUService_UsageInKernelmodeNormalized(t *testing.T) {
	usageOldValuesTest := []uint64{100, 10, 500000050}
	usageValuesTest := []uint64{3, 500000010, 500000050}
	var statsList = make([]types.StatsJSON, 3)
	var onlineCPUS = uint32(4)
	for index := range statsList {
		statsList[index].PreCPUStats.CPUUsage.UsageInKernelmode = usageOldValuesTest[index]
		statsList[index].CPUStats.CPUUsage.UsageInKernelmode = usageValuesTest[index]
		statsList[index].CPUStats.OnlineCPUs = onlineCPUS
	}
	testCase := []struct {
		given    types.StatsJSON
		expected float64
	}{
		{statsList[0], -1},
		{statsList[1], 0.5},
		{statsList[2], 0},
	}
	for _, tt := range testCase {
		usage := cpuUsageFor(tt.given)
		out := usage.InKernelModeNormalized()
		if out != tt.expected {
			t.Errorf("usageInKernelmodeNormalized(%v) => %v, want %v", tt.given.CPUStats.CPUUsage.UsageInKernelmode, out, tt.expected)
		}
	}
}

func TestCPUService_UsageInUsermode(t *testing.T) {
	usageOldValuesTest := []uint64{0, 1965, 500}
	usageValuesTest := []uint64{500000000, 325, 1000000500}
	var statsList = make([]types.StatsJSON, 3)
	var onlineCPUS = uint32(4)
	for index := range statsList {
		statsList[index].PreCPUStats.CPUUsage.UsageInUsermode = usageOldValuesTest[index]
		statsList[index].CPUStats.CPUUsage.UsageInUsermode = usageValuesTest[index]
		statsList[index].CPUStats.OnlineCPUs = onlineCPUS
	}
	testCase := []struct {
		given    types.StatsJSON
		expected float64
	}{
		{statsList[0], 2},
		{statsList[1], -1},
		{statsList[2], 4},
	}
	for _, tt := range testCase {
		usage := cpuUsageFor(tt.given)
		out := usage.InUserMode()
		if out != tt.expected {
			t.Errorf("usageInUsermode(%v) => %v, want %v", tt.given.CPUStats.CPUUsage.UsageInUsermode, out, tt.expected)
		}
	}
}

func TestCPUService_UsageInUsermodeNormalized(t *testing.T) {
	usageOldValuesTest := []uint64{0, 1965, 500}
	usageValuesTest := []uint64{500000000, 325, 1000000500}
	var statsList = make([]types.StatsJSON, 3)
	var onlineCPUS = uint32(4)
	for index := range statsList {
		statsList[index].PreCPUStats.CPUUsage.UsageInUsermode = usageOldValuesTest[index]
		statsList[index].CPUStats.CPUUsage.UsageInUsermode = usageValuesTest[index]
		statsList[index].CPUStats.OnlineCPUs = onlineCPUS
	}
	testCase := []struct {
		given    types.StatsJSON
		expected float64
	}{
		{statsList[0], 0.5},
		{statsList[1], -1},
		{statsList[2], 1},
	}
	for _, tt := range testCase {
		usage := cpuUsageFor(tt.given)
		out := usage.InUserModeNormalized()
		if out != tt.expected {
			t.Errorf("usageInUsermodeNormalized(%v) => %v, want %v", tt.given.CPUStats.CPUUsage.UsageInUsermode, out, tt.expected)
		}
	}
}

func equalEvent(expectedEvent common.MapStr, event common.MapStr) bool {
	return reflect.DeepEqual(expectedEvent, event)
}
