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

func (m *MetricSet) eventMapping(hs mo.HostSystem, perfMetrics *PerformanceMetrics, networkNames, datastoreNames, virtualmachine []string) mapstr.M {
	event := mapstr.M{
		"name":   hs.Summary.Config.Name,
		"status": hs.Summary.OverallStatus,
		"uptime": hs.Summary.QuickStats.Uptime,
		"cpu":    mapstr.M{"used": mapstr.M{"mhz": hs.Summary.QuickStats.OverallCpuUsage}},
		"disk": mapstr.M{
			"capacity":      mapstr.M{"usage": mapstr.M{"bytes": perfMetrics.DiskCapacityUsage * 1000}},
			"devicelatency": mapstr.M{"average": mapstr.M{"ms": perfMetrics.DiskDeviceLatency}},
			"latency":       mapstr.M{"total": mapstr.M{"ms": perfMetrics.DiskMaxTotalLatency}},
			"total":         mapstr.M{"bytes": perfMetrics.DiskUsage * 1000},
			"read":          mapstr.M{"bytes": perfMetrics.DiskRead * 1000},
			"write":         mapstr.M{"bytes": perfMetrics.DiskWrite * 1000},
		},
		"network": mapstr.M{
			"bandwidth": mapstr.M{
				"transmitted": mapstr.M{"bytes": perfMetrics.NetTransmitted * 1000},
				"received":    mapstr.M{"bytes": perfMetrics.NetReceived * 1000},
				"total":       mapstr.M{"bytes": perfMetrics.NetUsage * 1000},
			},
			"packets": mapstr.M{
				"transmitted": mapstr.M{"count": perfMetrics.NetPacketTransmitted},
				"received":    mapstr.M{"count": perfMetrics.NetPacketReceived},
				"errors": mapstr.M{
					"transmitted": mapstr.M{"count": perfMetrics.NetErrorsTransmitted},
					"received":    mapstr.M{"count": perfMetrics.NetErrorsReceived},
					"total":       mapstr.M{"count": perfMetrics.NetErrorsTransmitted + perfMetrics.NetErrorsReceived},
				},
				"multicast": mapstr.M{
					"transmitted": mapstr.M{"count": perfMetrics.NetMulticastTransmitted},
					"received":    mapstr.M{"count": perfMetrics.NetMulticastReceived},
					"total":       mapstr.M{"count": perfMetrics.NetMulticastTransmitted + perfMetrics.NetMulticastReceived},
				},
				"dropped": mapstr.M{
					"transmitted": mapstr.M{"count": perfMetrics.NetDroppedTransmitted},
					"received":    mapstr.M{"count": perfMetrics.NetDroppedReceived},
					"total":       mapstr.M{"count": perfMetrics.NetDroppedTransmitted + perfMetrics.NetDroppedReceived},
				},
			},
		},
	}
	if hw := hs.Summary.Hardware; hw != nil {
		totalCPU := int64(hw.CpuMhz) * int64(hw.NumCpuCores)
		usedMemory := int64(hs.Summary.QuickStats.OverallMemoryUsage) * 1024 * 1024
		event.Put("cpu.total.mhz", totalCPU)
		event.Put("cpu.free.mhz", totalCPU-int64(hs.Summary.QuickStats.OverallCpuUsage))
		event.Put("memory.used.bytes", usedMemory)
		event.Put("memory.free.bytes", hw.MemorySize-usedMemory)
		event.Put("memory.total.bytes", hw.MemorySize)
	} else {
		m.Logger().Debug("'Hardware' or 'Summary' data not found. This is either a parsing error from vsphere library, an error trying to reach host/guest or incomplete information returned from host/guest")
	}

	if len(virtualmachine) > 0 {
		event.Put("vm.names", virtualmachine)
		event.Put("vm.count", len(virtualmachine))
	}

	if len(datastoreNames) > 0 {
		event.Put("datastore.names", datastoreNames)
		event.Put("datastore.count", len(datastoreNames))
	}

	if len(networkNames) > 0 {
		event.Put("network_names", networkNames)
		event.Put("network_count", len(networkNames))
	}

	return event
}
