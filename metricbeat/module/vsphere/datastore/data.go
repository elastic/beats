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

func (m *MetricSet) eventMapping(ds mo.Datastore, data *metricData) mapstr.M {
	event := mapstr.M{
		"name":   ds.Summary.Name,
		"fstype": ds.Summary.Type,
		"status": ds.OverallStatus,
		"host": mapstr.M{
			"count": len(data.assetNames.outputHsNames),
		},
		"vm": mapstr.M{
			"count": len(data.assetNames.outputVmNames),
		},
		"capacity": mapstr.M{
			"total": mapstr.M{
				"bytes": ds.Summary.Capacity,
			},
			"free": mapstr.M{
				"bytes": ds.Summary.FreeSpace,
			},
			"used": mapstr.M{
				"bytes": ds.Summary.Capacity - ds.Summary.FreeSpace,
			},
		},
	}

	if ds.Summary.Capacity > 0 {
		usedSpacePercent := float64(ds.Summary.Capacity-ds.Summary.FreeSpace) / float64(ds.Summary.Capacity)
		event.Put("capacity.used.pct", usedSpacePercent)
	}

	if len(data.assetNames.outputHsNames) > 0 {
		event.Put("host.names", data.assetNames.outputHsNames)
	}

	if len(data.assetNames.outputVmNames) > 0 {
		event.Put("vm.names", data.assetNames.outputVmNames)
	}

	mapPerfMetricToEvent(event, data.perfMetrics)

	return event
}

func mapPerfMetricToEvent(event mapstr.M, perfMetricMap map[string]interface{}) {
	const bytesMultiplier = 1000

	if val, exist := perfMetricMap["datastore.read.average"]; exist {
		event.Put("read.bytes", val.(int64)*bytesMultiplier)
	}
	if val, exist := perfMetricMap["datastore.totalReadLatency.average"]; exist {
		event.Put("read.latency.total.ms", val)
	}
	if val, exist := perfMetricMap["datastore.write.average"]; exist {
		event.Put("write.bytes", val.(int64)*bytesMultiplier)
	}
	if val, exist := perfMetricMap["datastore.totalWriteLatency.average"]; exist {
		event.Put("write.latency.total.ms", val)
	}
	if val, exist := perfMetricMap["datastore.datastoreIops.average"]; exist {
		event.Put("iops", val)
	}
}
