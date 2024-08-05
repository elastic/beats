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

func (m *MetricSet) eventMapping(hs mo.HostSystem, perfMertics *PerformanceMetrics, networkNames []string) mapstr.M {
	totalErrorPacketsCount := perfMertics.NetErrorsTransmitted + perfMertics.NetErrorsReceived
	totalMulticastPacketsCount := perfMertics.NetMulticastTransmitted + perfMertics.NetMulticastReceived
	totalDroppedPacketsCount := perfMertics.NetDroppedTransmitted + perfMertics.NetDroppedReceived
	totalCPU := int64(0)
	freeCPU := int64(0)
	freeMemory := int64(0)
	totalMemory := int64(0)
	usedMemory := int64(0)

	if hs.Summary.Hardware != nil {
		totalCPU = int64(hs.Summary.Hardware.CpuMhz) * int64(hs.Summary.Hardware.NumCpuCores)
		freeCPU = totalCPU - int64(hs.Summary.QuickStats.OverallCpuUsage)
		usedMemory = int64(hs.Summary.QuickStats.OverallMemoryUsage) * 1024 * 1024
		freeMemory = hs.Summary.Hardware.MemorySize - usedMemory
		totalMemory = hs.Summary.Hardware.MemorySize
	} else {
		m.Logger().Debug("'Hardware' or 'Summary' data not found. This is either a parsing error from vsphere library, an error trying to reach host/guest or incomplete information returned from host/guest")
	}

	event := mapstr.M{
		"name":   hs.Summary.Config.Name,
		"status": hs.Summary.OverallStatus,
		"uptime": hs.Summary.QuickStats.Uptime,
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
		"disk": mapstr.M{
			"device": mapstr.M{
				"latency": mapstr.M{
					"ms": perfMertics.DiskDeviceLatency,
				},
			},
			"latency": mapstr.M{
				"total": mapstr.M{
					"ms": perfMertics.DiskMaxTotalLatency,
				},
			},
			"total": mapstr.M{
				"bytes": perfMertics.DiskUsage,
			},
			"read": mapstr.M{
				"bytes": perfMertics.DiskRead,
			},
			"write": mapstr.M{
				"bytes": perfMertics.DiskWrite,
			},
		},
		"memory": mapstr.M{
			"used": mapstr.M{
				"bytes": usedMemory,
			},
			"total": mapstr.M{
				"bytes": totalMemory,
			},
			"free": mapstr.M{
				"bytes": freeMemory,
			},
		},
		"network": mapstr.M{
			"bandwidth": mapstr.M{
				"transmitted": mapstr.M{
					"bytes": perfMertics.NetTransmitted,
				},
				"received": mapstr.M{
					"bytes": perfMertics.NetReceived,
				},
				"total": mapstr.M{
					"bytes": perfMertics.NetUsage,
				},
			},
			"packets": mapstr.M{
				"transmitted": mapstr.M{
					"count": perfMertics.NetPacketTransmitted,
				},
				"received": mapstr.M{
					"count": perfMertics.NetPacketReceived,
				},
				"errors": mapstr.M{
					"transmitted": mapstr.M{
						"count": perfMertics.NetErrorsTransmitted,
					},
					"received": mapstr.M{
						"count": perfMertics.NetErrorsReceived,
					},
					"total": mapstr.M{
						"count": totalErrorPacketsCount,
					},
				},
				"multicast": mapstr.M{
					"transmitted": mapstr.M{
						"count": perfMertics.NetMulticastTransmitted,
					},
					"received": mapstr.M{
						"count": perfMertics.NetMulticastReceived,
					},
					"total": mapstr.M{
						"count": totalMulticastPacketsCount,
					},
				},
				"dropped": mapstr.M{
					"transmitted": mapstr.M{
						"count": perfMertics.NetDroppedTransmitted,
					},
					"received": mapstr.M{
						"count": perfMertics.NetDroppedReceived,
					},
					"total": mapstr.M{
						"count": totalDroppedPacketsCount,
					},
				},
			},
		},
	}

	if len(networkNames) > 0 {
		event.Put("network_names", networkNames)
	}

	return event
}
