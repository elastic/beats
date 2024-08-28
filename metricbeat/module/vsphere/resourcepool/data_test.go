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

package resourcepool

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

func TestEventMapping(t *testing.T) {
	var m *ResourcePoolMetricSet
	var ResourcePoolTest = mo.ResourcePool{
		ManagedEntity: mo.ManagedEntity{
			OverallStatus: "green",
			Name:          "resourcepool-test",
		},
		Summary: &types.ResourcePoolSummary{
			QuickStats: &types.ResourcePoolQuickStats{
				OverallCpuUsage:              100,
				OverallCpuDemand:             100,
				GuestMemoryUsage:             100 * 1024 * 1024,
				HostMemoryUsage:              70 * 1024 * 1024,
				DistributedCpuEntitlement:    50,
				DistributedMemoryEntitlement: 50 * 1024 * 1024,
				StaticCpuEntitlement:         40,
				StaticMemoryEntitlement:      78,
				PrivateMemory:                10 * 1024 * 1024,
				SharedMemory:                 20 * 1024 * 1024,
				SwappedMemory:                30 * 1024 * 1024,
				BalloonedMemory:              40 * 1024 * 1024,
				OverheadMemory:               50 * 1024 * 1024,
				ConsumedOverheadMemory:       60 * 1024 * 1024,
				CompressedMemory:             70 * 1024,
			},
		},
	}

	var metricDataTest = metricData{
		assetsNames: assetNames{
			outputVmNames: []string{"vm-1", "vm-2"},
		},
	}

	event := m.mapEvent(ResourcePoolTest, &metricDataTest) // Ensure this is within a function

	vmName, _ := event.GetValue("vm.names")
	assert.EqualValues(t, metricDataTest.assetsNames.outputVmNames, vmName)

	vmCount, _ := event.GetValue("vm.count")
	assert.EqualValues(t, vmCount, len(metricDataTest.assetsNames.outputVmNames))

	status, _ := event.GetValue("status")
	assert.EqualValues(t, "green", status)

	name := event["name"].(string)
	assert.EqualValues(t, name, "resourcepool-test")

	cpuUsage, _ := event.GetValue("cpu.usage.mhz")
	assert.GreaterOrEqual(t, cpuUsage, int64(0))

	cpuDemand, _ := event.GetValue("cpu.demand.mhz")
	assert.GreaterOrEqual(t, cpuDemand, int64(0))

	guestMemoryUsage, _ := event.GetValue("memory.usage.guest.bytes")
	assert.GreaterOrEqual(t, guestMemoryUsage, int64(0))

	hostMemoryUsage, _ := event.GetValue("memory.usage.host.bytes")
	assert.GreaterOrEqual(t, hostMemoryUsage, int64(0))

	cpuEntitlement, _ := event.GetValue("cpu.entitlement.mhz")
	assert.GreaterOrEqual(t, cpuEntitlement, int64(0))

	memoryEntitlement, _ := event.GetValue("memory.entitlement.bytes")
	assert.GreaterOrEqual(t, memoryEntitlement, int64(0))

	cpuStaticEntitlement, _ := event.GetValue("cpu.entitlement.static.mhz")
	assert.GreaterOrEqual(t, cpuStaticEntitlement, int32(0))

	memoryStaticEntitlement, _ := event.GetValue("memory.entitlement.static.mhz")
	assert.GreaterOrEqual(t, memoryStaticEntitlement, int32(0))

	memoryPrivate, _ := event.GetValue("memory.private.bytes")
	assert.GreaterOrEqual(t, memoryPrivate, int64(0))

	memoryShared, _ := event.GetValue("memory.shared.bytes")
	assert.GreaterOrEqual(t, memoryShared, int64(0))

	memorySwapped, _ := event.GetValue("memory.swapped.bytes")
	assert.GreaterOrEqual(t, memorySwapped, int64(0))

	memoryBallooned, _ := event.GetValue("memory.ballooned.bytes")
	assert.GreaterOrEqual(t, memoryBallooned, int64(0))

	memoryOverhead, _ := event.GetValue("memory.overhead.bytes")
	assert.GreaterOrEqual(t, memoryOverhead, int64(0))

	memoryOverheadConsumed, _ := event.GetValue("memory.overhead.consumed.bytes")
	assert.GreaterOrEqual(t, memoryOverheadConsumed, int64(0))

	memoryCompressed, _ := event.GetValue("memory.compressed.bytes")
	assert.GreaterOrEqual(t, memoryCompressed, int64(0))
}
