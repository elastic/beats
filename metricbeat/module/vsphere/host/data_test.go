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
	var m *MetricSet
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

	var PerformanceMetricsTest = PerformanceMetrics{
		NetUsage:                100,
		NetDroppedTransmitted:   5,
		NetDroppedReceived:      3,
		NetMulticastTransmitted: 10,
		NetMulticastReceived:    8,
		NetErrorsTransmitted:    1,
		NetErrorsReceived:       2,
		NetPacketTransmitted:    1500,
		NetPacketReceived:       1450,
		NetReceived:             2000,
		NetTransmitted:          1800,
		DiskWrite:               500,
		DiskRead:                600,
		DiskUsage:               700,
		DiskMaxTotalLatency:     100,
		DiskDeviceLatency:       50,
		DiskCapacityUsage:       80,
	}

	event := m.eventMapping(HostSystemTest, &PerformanceMetricsTest, []string{"network-1", "network-2"}, []string{"datastore-1", "datastore-2"}, []string{"vm-1", "vm-2"})

	cpuUsed, _ := event.GetValue("cpu.used.mhz")
	assert.EqualValues(t, 67, cpuUsed)

	cpuTotal, _ := event.GetValue("cpu.total.mhz")
	assert.EqualValues(t, 4588, cpuTotal)

	cpuFree, _ := event.GetValue("cpu.free.mhz")
	assert.EqualValues(t, 4521, cpuFree)

	memoryUsed, _ := event.GetValue("memory.used.bytes")
	assert.EqualValues(t, int64(2251799812636672), memoryUsed)

	memoryTotal, _ := event.GetValue("memory.total.bytes")
	assert.EqualValues(t, int64(2251799812636672), memoryTotal)

	memoryFree, _ := event.GetValue("memory.free.bytes")
	assert.EqualValues(t, 0, memoryFree)

	// New asserts for PerformanceMetricsTest
	diskCapacityUsage, _ := event.GetValue("disk.capacity.usage.bytes")
	assert.EqualValues(t, PerformanceMetricsTest.DiskCapacityUsage*1000, diskCapacityUsage)
	diskDevicelatency, _ := event.GetValue("disk.devicelatency.average.ms")
	assert.EqualValues(t, PerformanceMetricsTest.DiskDeviceLatency, diskDevicelatency)
	diskLatency, _ := event.GetValue("disk.latency.total.ms")
	assert.EqualValues(t, PerformanceMetricsTest.DiskMaxTotalLatency, diskLatency)
	diskTotal, _ := event.GetValue("disk.total.bytes")
	assert.EqualValues(t, PerformanceMetricsTest.DiskUsage*1000, diskTotal)
	diskRead, _ := event.GetValue("disk.read.bytes")
	assert.EqualValues(t, PerformanceMetricsTest.DiskRead*1000, diskRead)
	diskWrite, _ := event.GetValue("disk.write.bytes")
	assert.EqualValues(t, PerformanceMetricsTest.DiskWrite*1000, diskWrite)

	networkBandwidthTransmitted, _ := event.GetValue("network.bandwidth.transmitted.bytes")
	assert.EqualValues(t, PerformanceMetricsTest.NetTransmitted*1000, networkBandwidthTransmitted)
	networkBandwidthReceived, _ := event.GetValue("network.bandwidth.received.bytes")
	assert.EqualValues(t, PerformanceMetricsTest.NetReceived*1000, networkBandwidthReceived)
	networkBandwidthTotal, _ := event.GetValue("network.bandwidth.total.bytes")
	assert.EqualValues(t, PerformanceMetricsTest.NetUsage*1000, networkBandwidthTotal)
	networkPacketsTransmitted, _ := event.GetValue("network.packets.transmitted.count")
	assert.EqualValues(t, PerformanceMetricsTest.NetPacketTransmitted, networkPacketsTransmitted)
	networkPacketsReceived, _ := event.GetValue("network.packets.received.count")
	assert.EqualValues(t, PerformanceMetricsTest.NetPacketReceived, networkPacketsReceived)
	networkPacketsErrorsTransmitted, _ := event.GetValue("network.packets.errors.transmitted.count")
	assert.EqualValues(t, PerformanceMetricsTest.NetErrorsTransmitted, networkPacketsErrorsTransmitted)
	networkPacketsErrorsReceived, _ := event.GetValue("network.packets.errors.received.count")
	assert.EqualValues(t, PerformanceMetricsTest.NetErrorsReceived, networkPacketsErrorsReceived)
	networkPacketsErrorsTotal, _ := event.GetValue("network.packets.errors.total.count")
	assert.EqualValues(t, PerformanceMetricsTest.NetErrorsTransmitted+PerformanceMetricsTest.NetErrorsReceived, networkPacketsErrorsTotal)
	networkPacketsMulticastTransmitted, _ := event.GetValue("network.packets.multicast.transmitted.count")
	assert.EqualValues(t, PerformanceMetricsTest.NetMulticastTransmitted, networkPacketsMulticastTransmitted)
	networkPacketsMulticastReceived, _ := event.GetValue("network.packets.multicast.received.count")
	assert.EqualValues(t, PerformanceMetricsTest.NetMulticastReceived, networkPacketsMulticastReceived)
	networkPacketsMulticastTotal, _ := event.GetValue("network.packets.multicast.total.count")
	assert.EqualValues(t, PerformanceMetricsTest.NetMulticastTransmitted+PerformanceMetricsTest.NetMulticastReceived, networkPacketsMulticastTotal)
	networkPacketsDroppedTransmitted, _ := event.GetValue("network.packets.dropped.transmitted.count")
	assert.EqualValues(t, PerformanceMetricsTest.NetDroppedTransmitted, networkPacketsDroppedTransmitted)
	networkPacketsDroppedReceived, _ := event.GetValue("network.packets.dropped.received.count")
	assert.EqualValues(t, PerformanceMetricsTest.NetDroppedReceived, networkPacketsDroppedReceived)
	networkPacketsDroppedTotal, _ := event.GetValue("network.packets.dropped.total.count")
	assert.EqualValues(t, PerformanceMetricsTest.NetDroppedTransmitted+PerformanceMetricsTest.NetDroppedReceived, networkPacketsDroppedTotal)

	networkNames, _ := event.GetValue("network_names")
	assert.EqualValues(t, []string{"network-1", "network-2"}, networkNames)

	vmNames, _ := event.GetValue("vm.names")
	assert.EqualValues(t, []string{"vm-1", "vm-2"}, vmNames)

	datastoreNames, _ := event.GetValue("datastore.names")
	assert.EqualValues(t, []string{"datastore-1", "datastore-2"}, datastoreNames)
}
