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

package datastore

import (
	"github.com/vmware/govmomi/vim25/mo"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

func (m *MetricSet) eventMapping(ds mo.Datastore, perfMertics *PerformanceMetrics) mapstr.M {
	usedSpaceBytes := ds.Summary.Capacity - ds.Summary.FreeSpace

	event := mapstr.M{
		"read": mapstr.M{
			"bytes": perfMertics.DsRead * 1000,
			"latency": mapstr.M{
				"total": mapstr.M{
					"ms": perfMertics.DsReadLatency,
				},
			},
		},
		"write": mapstr.M{
			"bytes": perfMertics.DsWrite * 1000,
			"latency": mapstr.M{
				"total": mapstr.M{
					"ms": perfMertics.DsWriteLatency,
				},
			},
		},
		"iops":   perfMertics.DsIops,
		"name":   ds.Summary.Name,
		"fstype": ds.Summary.Type,
		"status": ds.OverallStatus,
		"capacity": mapstr.M{
			"total": mapstr.M{
				"bytes": ds.Summary.Capacity,
			},
			"free": mapstr.M{
				"bytes": ds.Summary.FreeSpace,
			},
			"used": mapstr.M{
				"bytes": usedSpaceBytes,
			},
		},
	}

	var usedSpacePercent float64
	if ds.Summary.Capacity > 0 {
		usedSpacePercent = float64(ds.Summary.Capacity-ds.Summary.FreeSpace) / float64(ds.Summary.Capacity)
		event.Put("capacity.used.pct", usedSpacePercent)
	}

	if len(ds.Host) > 0 {
		event.Put("host.names", ds.Host)
	}

	if len(ds.Vm) > 0 {
		event.Put("vm.names", ds.Vm)
	}

	return event
}
