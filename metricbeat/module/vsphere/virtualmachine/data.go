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

package virtualmachine

import (
	"golang.org/x/exp/constraints"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

func (m *MetricSet) eventMapping(data VMData) mapstr.M {

	usedMemory := int64(data.VM.Summary.QuickStats.GuestMemoryUsage) * 1024 * 1024
	usedCPU := data.VM.Summary.QuickStats.OverallCpuUsage
	totalCPU := data.VM.Summary.Config.CpuReservation
	totalMemory := int64(data.VM.Summary.Config.MemorySizeMB) * 1024 * 1024

	freeCPU := max(0, totalCPU-usedCPU)
	freeMemory := max(0, totalMemory-usedMemory)

	event := mapstr.M{
		"name":          data.VM.Summary.Config.Name,
		"os":            data.VM.Summary.Config.GuestFullName,
		"uptime":        data.VM.Summary.QuickStats.UptimeSeconds,
		"status":        data.VM.Summary.OverallStatus,
		"host.id":       data.HostID,
		"host.hostname": data.HostName,
		"cpu": mapstr.M{
			"used":  mapstr.M{"mhz": usedCPU},
			"total": mapstr.M{"mhz": totalCPU},
			"free":  mapstr.M{"mhz": freeCPU},
		},
		"memory": mapstr.M{
			"used": mapstr.M{
				"guest": mapstr.M{"bytes": usedMemory},
				"host":  mapstr.M{"bytes": int64(data.VM.Summary.QuickStats.HostMemoryUsage) * 1024 * 1024},
			},
			"total": mapstr.M{
				"guest": mapstr.M{"bytes": totalMemory},
			},
			"free": mapstr.M{
				"guest": mapstr.M{"bytes": freeMemory},
			},
		},
	}
	if len(data.CustomFields) > 0 {
		event["custom_fields"] = data.CustomFields
	}
	if len(data.NetworkNames) > 0 {
		event["network_names"] = data.NetworkNames
	}
	if len(data.DatastoreNames) > 0 {
		event["datastore.names"] = data.DatastoreNames
	}
	return event
}
func max[T constraints.Ordered](a T, b T) T {
	if a > b {
		return a
	}
	return b
}
