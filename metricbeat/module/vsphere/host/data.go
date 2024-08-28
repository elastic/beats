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
	"github.com/vmware/govmomi/vim25/mo"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

<<<<<<< HEAD
func eventMapping(hs mo.HostSystem) mapstr.M {
	totalCPU := int64(hs.Summary.Hardware.CpuMhz) * int64(hs.Summary.Hardware.NumCpuCores)
	freeCPU := int64(totalCPU) - int64(hs.Summary.QuickStats.OverallCpuUsage)
	usedMemory := int64(hs.Summary.QuickStats.OverallMemoryUsage) * 1024 * 1024
	freeMemory := int64(hs.Summary.Hardware.MemorySize) - usedMemory

	event := mapstr.M{
		"name": hs.Summary.Config.Name,
		"cpu": mapstr.M{
			"used": mapstr.M{
				"mhz": hs.Summary.QuickStats.OverallCpuUsage,
			},
			"total": mapstr.M{
				"mhz": totalCPU,
			},
			"free": mapstr.M{
				"mhz": freeCPU,
			},
		},
		"memory": mapstr.M{
			"used": mapstr.M{
				"bytes": usedMemory,
			},
			"total": mapstr.M{
				"bytes": hs.Summary.Hardware.MemorySize,
			},
			"free": mapstr.M{
				"bytes": freeMemory,
			},
		},
=======
func (m *HostMetricSet) mapEvent(hs mo.HostSystem, data *metricData) mapstr.M {
	const bytesMultiplier int64 = 1024 * 1024
	event := mapstr.M{
		"name":   hs.Summary.Config.Name,
		"status": hs.Summary.OverallStatus,
		"uptime": hs.Summary.QuickStats.Uptime,
		"cpu":    mapstr.M{"used": mapstr.M{"mhz": hs.Summary.QuickStats.OverallCpuUsage}},
	}

	mapPerfMetricToEvent(event, data.perfMetrics)

	if hw := hs.Summary.Hardware; hw != nil {
		totalCPU := int64(hw.CpuMhz) * int64(hw.NumCpuCores)
		usedMemory := int64(hs.Summary.QuickStats.OverallMemoryUsage) * bytesMultiplier
		event.Put("cpu.total.mhz", totalCPU)
		event.Put("cpu.free.mhz", totalCPU-int64(hs.Summary.QuickStats.OverallCpuUsage))
		event.Put("memory.used.bytes", usedMemory)
		event.Put("memory.free.bytes", hw.MemorySize-usedMemory)
		event.Put("memory.total.bytes", hw.MemorySize)
	} else {
		m.Logger().Debug("'Hardware' or 'Summary' data not found. This is either a parsing error from vsphere library, an error trying to reach host/guest or incomplete information returned from host/guest")
	}

	if len(data.assetNames.outputVmNames) > 0 {
		event.Put("vm.names", data.assetNames.outputVmNames)
		event.Put("vm.count", len(data.assetNames.outputVmNames))
	}

	if len(data.assetNames.outputDsNames) > 0 {
		event.Put("datastore.names", data.assetNames.outputDsNames)
		event.Put("datastore.count", len(data.assetNames.outputDsNames))
	}

	if len(data.assetNames.outputNetworkNames) > 0 {
		event.Put("network_names", data.assetNames.outputNetworkNames)
		event.Put("network.names", data.assetNames.outputNetworkNames)
		event.Put("network.count", len(data.assetNames.outputNetworkNames))
>>>>>>> 93ee5ca3ff ([vSphere][network] Add support for new metrics in network metricset (#40559))
	}

	return event
}
<<<<<<< HEAD
=======

func mapPerfMetricToEvent(event mapstr.M, perfMetricMap map[string]interface{}) {
	const bytesMultiplier int64 = 1024
	if val, exist := perfMetricMap["disk.capacity.usage.average"]; exist {
		event.Put("disk.capacity.usage.bytes", val.(int64)*bytesMultiplier)
	}
	if val, exist := perfMetricMap["disk.deviceLatency.average"]; exist {
		event.Put("disk.devicelatency.average.ms", val)
	}
	if val, exist := perfMetricMap["disk.maxTotalLatency.latest"]; exist {
		event.Put("disk.latency.total.ms", val)
	}
	if val, exist := perfMetricMap["disk.usage.average"]; exist {
		event.Put("disk.total.bytes", val.(int64)*bytesMultiplier)
	}
	if val, exist := perfMetricMap["disk.read.average"]; exist {
		event.Put("disk.read.bytes", val.(int64)*bytesMultiplier)
	}
	if val, exist := perfMetricMap["disk.write.average"]; exist {
		event.Put("disk.write.bytes", val.(int64)*bytesMultiplier)
	}

	if val, exist := perfMetricMap["net.transmitted.average"]; exist {
		event.Put("network.bandwidth.transmitted.bytes", val.(int64)*bytesMultiplier)
	}
	if val, exist := perfMetricMap["net.received.average"]; exist {
		event.Put("network.bandwidth.received.bytes", val.(int64)*bytesMultiplier)
	}
	if val, exist := perfMetricMap["net.usage.average"]; exist {
		event.Put("network.bandwidth.total.bytes", val.(int64)*bytesMultiplier)
	}

	if val, exist := perfMetricMap["net.packetsTx.summation"]; exist {
		event.Put("network.packets.transmitted.count", val)
	}
	if val, exist := perfMetricMap["net.packetsRx.summation"]; exist {
		event.Put("network.packets.received.count", val)
	}

	netErrorsTransmitted, netErrorsTransmittedExist := perfMetricMap["net.errorsTx.summation"]
	if netErrorsTransmittedExist {
		event.Put("network.packets.errors.transmitted.count", netErrorsTransmitted)
	}
	netErrorsReceived, netErrorsReceivedExist := perfMetricMap["net.errorsRx.summation"]
	if netErrorsReceivedExist {
		event.Put("network.packets.errors.received.count", netErrorsReceived)
	}
	if netErrorsTransmittedExist && netErrorsReceivedExist {
		event.Put("network.packets.errors.total.count", netErrorsTransmitted.(int64)+netErrorsReceived.(int64))
	}

	netMulticastTransmitted, netMulticastTransmittedExist := perfMetricMap["net.multicastTx.summation"]
	if netMulticastTransmittedExist {
		event.Put("network.packets.multicast.transmitted.count", netMulticastTransmitted)
	}
	netMulticastReceived, netMulticastReceivedExist := perfMetricMap["net.multicastRx.summation"]
	if netMulticastReceivedExist {
		event.Put("network.packets.multicast.received.count", netMulticastReceived)
	}
	if netMulticastTransmittedExist && netMulticastReceivedExist {
		event.Put("network.packets.multicast.total.count", netMulticastTransmitted.(int64)+netMulticastReceived.(int64))
	}

	netDroppedTransmitted, netDroppedTransmittedExist := perfMetricMap["net.droppedTx.summation"]
	if netDroppedTransmittedExist {
		event.Put("network.packets.dropped.transmitted.count", netDroppedTransmitted)
	}
	netDroppedReceived, netDroppedReceivedExist := perfMetricMap["net.droppedRx.summation"]
	if netDroppedReceivedExist {
		event.Put("network.packets.dropped.received.count", netDroppedReceived)
	}
	if netDroppedTransmittedExist && netDroppedReceivedExist {
		event.Put("network.packets.dropped.total.count", netDroppedTransmitted.(int64)+netDroppedReceived.(int64))
	}
}
>>>>>>> 93ee5ca3ff ([vSphere][network] Add support for new metrics in network metricset (#40559))
