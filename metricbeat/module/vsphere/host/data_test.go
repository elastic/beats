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

package host

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

func TestEventMapping(t *testing.T) {
	var HostSystemTest = mo.HostSystem{
		Summary: types.HostListSummary{
			Host: &types.ManagedObjectReference{Type: "HostSystem", Value: "ha-host"},
			Hardware: &types.HostHardwareSummary{
				MemorySize:  2251799812636672,
				CpuMhz:      2294,
				NumCpuCores: 2,
			},
			Config: types.HostConfigSummary{
				Name: "localhost.localdomain",
			},
			QuickStats: types.HostListSummaryQuickStats{
				OverallCpuUsage:    67,
				OverallMemoryUsage: math.MaxInt32,
			},
		},
	}

	event := eventMapping(HostSystemTest)

	cpuUsed, _ := event.GetValue("cpu.used.mhz")
	assert.EqualValues(t, 67, cpuUsed)

	cpuTotal, _ := event.GetValue("cpu.total.mhz")
	assert.EqualValues(t, 4588, cpuTotal)

	cpuFree, _ := event.GetValue("cpu.free.mhz")
	assert.EqualValues(t, 4521, cpuFree)

	memoryUsed, _ := event.GetValue("memory.used.bytes")
	assert.EqualValues(t, 2251799812636672, memoryUsed)

	memoryTotal, _ := event.GetValue("memory.total.bytes")
	assert.EqualValues(t, 2251799812636672, memoryTotal)

	memoryFree, _ := event.GetValue("memory.free.bytes")
	assert.EqualValues(t, 0, memoryFree)
}
