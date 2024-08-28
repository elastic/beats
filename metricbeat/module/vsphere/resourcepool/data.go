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

func (m *ResourcePoolMetricSet) mapEvent(rp mo.ResourcePool, data *metricData) mapstr.M {
	event := mapstr.M{
		"name":   rp.Name,
		"status": rp.OverallStatus,
	}

	if rp.Summary.GetResourcePoolSummary().QuickStats != nil {
		event.Put("cpu.usage.mhz", rp.Summary.GetResourcePoolSummary().QuickStats.OverallCpuUsage)
		event.Put("cpu.demand.mhz", rp.Summary.GetResourcePoolSummary().QuickStats.OverallCpuDemand)
		event.Put("cpu.entitlement.mhz", rp.Summary.GetResourcePoolSummary().QuickStats.DistributedCpuEntitlement)
		event.Put("cpu.entitlement.static.mhz", rp.Summary.GetResourcePoolSummary().QuickStats.StaticCpuEntitlement)
		event.Put("memory.usage.guest.bytes", rp.Summary.GetResourcePoolSummary().QuickStats.GuestMemoryUsage*1024*1024)
		event.Put("memory.usage.host.bytes", rp.Summary.GetResourcePoolSummary().QuickStats.HostMemoryUsage*1024*1024)
		event.Put("memory.entitlement.bytes", rp.Summary.GetResourcePoolSummary().QuickStats.DistributedMemoryEntitlement*1024*1024)
		event.Put("memory.entitlement.static.mhz", rp.Summary.GetResourcePoolSummary().QuickStats.StaticMemoryEntitlement)
		event.Put("memory.private.bytes", rp.Summary.GetResourcePoolSummary().QuickStats.PrivateMemory*1024*1024)
		event.Put("memory.shared.bytes", rp.Summary.GetResourcePoolSummary().QuickStats.SharedMemory*1024*1024)
		event.Put("memory.swapped.bytes", rp.Summary.GetResourcePoolSummary().QuickStats.SwappedMemory*1024*1024)
		event.Put("memory.ballooned.bytes", rp.Summary.GetResourcePoolSummary().QuickStats.BalloonedMemory*1024*1024)
		event.Put("memory.overhead.bytes", rp.Summary.GetResourcePoolSummary().QuickStats.OverheadMemory*1024*1024)
		event.Put("memory.overhead.consumed.bytes", rp.Summary.GetResourcePoolSummary().QuickStats.ConsumedOverheadMemory*1024*1024)
		event.Put("memory.compressed.bytes", rp.Summary.GetResourcePoolSummary().QuickStats.CompressedMemory*1024)
	}

	if len(data.assetNames.outputVmNames) > 0 {
		event.Put("vm.names", data.assetNames.outputVmNames)
		event.Put("vm.count", len(data.assetNames.outputVmNames))
	}
	return event
}
