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
	"github.com/vmware/govmomi/vim25/mo"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

const (
	mbToBytes = 1024 * 1024
	kbToBytes = 1024
)

func (m *ResourcePoolMetricSet) mapEvent(rp mo.ResourcePool, data *metricData) mapstr.M {
	event := mapstr.M{
		"name":   rp.Name,
		"status": rp.OverallStatus,
	}

	quickStats := rp.Summary.GetResourcePoolSummary().QuickStats
	if quickStats == nil {
		return event
	}

	event.Put("cpu.usage.mhz", quickStats.OverallCpuUsage)
	event.Put("cpu.demand.mhz", quickStats.OverallCpuDemand)
	event.Put("cpu.entitlement.mhz", quickStats.DistributedCpuEntitlement)
	event.Put("cpu.entitlement.static.mhz", quickStats.StaticCpuEntitlement)
	event.Put("memory.usage.guest.bytes", quickStats.GuestMemoryUsage*mbToBytes)
	event.Put("memory.usage.host.bytes", quickStats.HostMemoryUsage*mbToBytes)
	event.Put("memory.entitlement.bytes", quickStats.DistributedMemoryEntitlement*mbToBytes)
	event.Put("memory.entitlement.static.bytes", quickStats.StaticMemoryEntitlement*mbToBytes)
	event.Put("memory.private.bytes", quickStats.PrivateMemory*mbToBytes)
	event.Put("memory.shared.bytes", quickStats.SharedMemory*mbToBytes)
	event.Put("memory.swapped.bytes", quickStats.SwappedMemory*mbToBytes)
	event.Put("memory.ballooned.bytes", quickStats.BalloonedMemory*mbToBytes)
	event.Put("memory.overhead.bytes", quickStats.OverheadMemory*mbToBytes)
	event.Put("memory.overhead.consumed.bytes", quickStats.ConsumedOverheadMemory*mbToBytes)
	event.Put("memory.compressed.bytes", quickStats.CompressedMemory*kbToBytes)

	if len(data.assetNames.outputVmNames) > 0 {
		event.Put("vm.names", data.assetNames.outputVmNames)
		event.Put("vm.count", len(data.assetNames.outputVmNames))
	}

	return event
}
