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
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func (m *MetricSet) mapEvent(data VMData) mapstr.M {
	const bytesMultiplier = int64(1024 * 1024)
	usedMemory := int64(data.VM.Summary.QuickStats.GuestMemoryUsage) * bytesMultiplier
	usedCPU := data.VM.Summary.QuickStats.OverallCpuUsage
	totalCPU := data.VM.Summary.Config.CpuReservation
	totalMemory := int64(data.VM.Summary.Config.MemorySizeMB) * bytesMultiplier

	freeCPU := max(0, totalCPU-usedCPU)
	freeMemory := max(0, totalMemory-usedMemory)

	event := mapstr.M{
		"name":   data.VM.Summary.Config.Name,
		"os":     data.VM.Summary.Config.GuestFullName,
		"uptime": data.VM.Summary.QuickStats.UptimeSeconds,
		"status": data.VM.Summary.OverallStatus,
		"host": mapstr.M{
			"id":       data.HostID,
			"hostname": data.HostName,
		},
		"cpu": mapstr.M{
			"used":  mapstr.M{"mhz": usedCPU},
			"total": mapstr.M{"mhz": totalCPU},
			"free":  mapstr.M{"mhz": freeCPU},
		},
		"memory": mapstr.M{
			"used": mapstr.M{
				"guest": mapstr.M{"bytes": usedMemory},
				"host":  mapstr.M{"bytes": int64(data.VM.Summary.QuickStats.HostMemoryUsage) * bytesMultiplier},
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
		event.Put("network.count", len(data.NetworkNames))
		event.Put("network.names", data.NetworkNames)
		event["network_names"] = data.NetworkNames
	}
	if len(data.DatastoreNames) > 0 {
		event.Put("datastore.count", len(data.DatastoreNames))
		event.Put("datastore.names", data.DatastoreNames)
	}
	if len(data.Snapshots) > 0 {
		event.Put("snapshot.count", len(data.Snapshots))
		event.Put("snapshot.info", data.Snapshots)
	}
	if len(data.triggerdAlarms) > 0 {
		event.Put("triggerd_alarms", data.triggerdAlarms)
	}

	return event
}
