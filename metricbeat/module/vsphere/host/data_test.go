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
	var m *HostMetricSet
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

	var metricDataTest = metricData{
		perfMetrics: map[string]interface{}{
			"disk.capacity.usage.average": int64(100),
			"disk.deviceLatency.average":  int64(5),
			"disk.maxTotalLatency.latest": int64(3),
			"disk.usage.average":          int64(10),
			"disk.read.average":           int64(8),
			"disk.write.average":          int64(1),
			"net.transmitted.average":     int64(2),
			"net.received.average":        int64(1500),
			"net.usage.average":           int64(1450),
			"net.packetsTx.summation":     int64(2000),
			"net.packetsRx.summation":     int64(1800),
			"net.errorsTx.summation":      int64(500),
			"net.errorsRx.summation":      int64(600),
			"net.multicastTx.summation":   int64(700),
			"net.multicastRx.summation":   int64(100),
			"net.droppedTx.summation":     int64(50),
			"net.droppedRx.summation":     int64(80),
		},
		assetNames: assetNames{
			outputNetworkNames: []string{"network-1", "network-2"},
			outputDsNames:      []string{"datastore-1", "datastore-2"},
			outputVmNames:      []string{"vm-1", "vm-2"},
		},
	}

	event := m.mapEvent(HostSystemTest, &metricDataTest)

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
	assert.EqualValues(t, metricDataTest.perfMetrics["disk.capacity.usage.average"].(int64)*1024, diskCapacityUsage)
	diskDevicelatency, _ := event.GetValue("disk.devicelatency.average.ms")
	assert.EqualValues(t, metricDataTest.perfMetrics["disk.deviceLatency.average"], diskDevicelatency)
	diskLatency, _ := event.GetValue("disk.latency.total.ms")
	assert.EqualValues(t, metricDataTest.perfMetrics["disk.maxTotalLatency.latest"], diskLatency)
	diskTotal, _ := event.GetValue("disk.total.bytes")
	assert.EqualValues(t, metricDataTest.perfMetrics["disk.usage.average"].(int64)*1024, diskTotal)
	diskRead, _ := event.GetValue("disk.read.bytes")
	assert.EqualValues(t, metricDataTest.perfMetrics["disk.read.average"].(int64)*1024, diskRead)
	diskWrite, _ := event.GetValue("disk.write.bytes")
	assert.EqualValues(t, metricDataTest.perfMetrics["disk.write.average"].(int64)*1024, diskWrite)

	networkBandwidthTransmitted, _ := event.GetValue("network.bandwidth.transmitted.bytes")
	assert.EqualValues(t, metricDataTest.perfMetrics["net.transmitted.average"].(int64)*1024, networkBandwidthTransmitted)
	networkBandwidthReceived, _ := event.GetValue("network.bandwidth.received.bytes")
	assert.EqualValues(t, metricDataTest.perfMetrics["net.received.average"].(int64)*1024, networkBandwidthReceived)
	networkBandwidthTotal, _ := event.GetValue("network.bandwidth.total.bytes")
	assert.EqualValues(t, metricDataTest.perfMetrics["net.usage.average"].(int64)*1024, networkBandwidthTotal)
	networkPacketsTransmitted, _ := event.GetValue("network.packets.transmitted.count")
	assert.EqualValues(t, metricDataTest.perfMetrics["net.packetsTx.summation"], networkPacketsTransmitted)
	networkPacketsReceived, _ := event.GetValue("network.packets.received.count")
	assert.EqualValues(t, metricDataTest.perfMetrics["net.packetsRx.summation"], networkPacketsReceived)
	networkPacketsErrorsTransmitted, _ := event.GetValue("network.packets.errors.transmitted.count")
	assert.EqualValues(t, metricDataTest.perfMetrics["net.errorsTx.summation"], networkPacketsErrorsTransmitted)
	networkPacketsErrorsReceived, _ := event.GetValue("network.packets.errors.received.count")
	assert.EqualValues(t, metricDataTest.perfMetrics["net.errorsRx.summation"], networkPacketsErrorsReceived)
	networkPacketsErrorsTotal, _ := event.GetValue("network.packets.errors.total.count")
	assert.EqualValues(t, metricDataTest.perfMetrics["net.errorsTx.summation"].(int64)+metricDataTest.perfMetrics["net.errorsRx.summation"].(int64), networkPacketsErrorsTotal)
	networkPacketsMulticastTransmitted, _ := event.GetValue("network.packets.multicast.transmitted.count")
	assert.EqualValues(t, metricDataTest.perfMetrics["net.multicastTx.summation"], networkPacketsMulticastTransmitted)
	networkPacketsMulticastReceived, _ := event.GetValue("network.packets.multicast.received.count")
	assert.EqualValues(t, metricDataTest.perfMetrics["net.multicastRx.summation"], networkPacketsMulticastReceived)
	networkPacketsMulticastTotal, _ := event.GetValue("network.packets.multicast.total.count")
	assert.EqualValues(t, metricDataTest.perfMetrics["net.multicastTx.summation"].(int64)+metricDataTest.perfMetrics["net.multicastRx.summation"].(int64), networkPacketsMulticastTotal)
	networkPacketsDroppedTransmitted, _ := event.GetValue("network.packets.dropped.transmitted.count")
	assert.EqualValues(t, metricDataTest.perfMetrics["net.droppedTx.summation"], networkPacketsDroppedTransmitted)
	networkPacketsDroppedReceived, _ := event.GetValue("network.packets.dropped.received.count")
	assert.EqualValues(t, metricDataTest.perfMetrics["net.droppedRx.summation"], networkPacketsDroppedReceived)
	networkPacketsDroppedTotal, _ := event.GetValue("network.packets.dropped.total.count")
	assert.EqualValues(t, metricDataTest.perfMetrics["net.droppedTx.summation"].(int64)+metricDataTest.perfMetrics["net.droppedRx.summation"].(int64), networkPacketsDroppedTotal)

	networkNames, _ := event.GetValue("network_names")
	assert.EqualValues(t, metricDataTest.assetNames.outputNetworkNames, networkNames)

	vmNames, _ := event.GetValue("vm.names")
	assert.EqualValues(t, metricDataTest.assetNames.outputVmNames, vmNames)

	datastoreNames, _ := event.GetValue("datastore.names")
	assert.EqualValues(t, metricDataTest.assetNames.outputDsNames, datastoreNames)
}
