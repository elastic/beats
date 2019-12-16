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

package cpu

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
)

func eventsMapping(r mb.ReporterV2, cpuStatsList []CPUStats) {
	for _, cpuStats := range cpuStatsList {
		eventMapping(r, &cpuStats)
	}
}

func eventMapping(r mb.ReporterV2, stats *CPUStats) {
	fields := common.MapStr{
		"core": stats.PerCPUUsage,
		"total": common.MapStr{
			"pct": stats.TotalUsage,
			"norm": common.MapStr{
				"pct": stats.TotalUsageNormalized,
			},
		},
		"kernel": common.MapStr{
			"ticks": stats.UsageInKernelmode,
			"pct":   stats.UsageInKernelmodePercentage,
			"norm": common.MapStr{
				"pct": stats.UsageInKernelmodePercentageNormalized,
			},
		},
		"user": common.MapStr{
			"ticks": stats.UsageInUsermode,
			"pct":   stats.UsageInUsermodePercentage,
			"norm": common.MapStr{
				"pct": stats.UsageInUsermodePercentageNormalized,
			},
		},
		"system": common.MapStr{
			"ticks": stats.SystemUsage,
			"pct":   stats.SystemUsagePercentage,
			"norm": common.MapStr{
				"pct": stats.SystemUsagePercentageNormalized,
			},
		},
	}

	r.Event(mb.Event{
		RootFields:      stats.Container.ToMapStr(),
		MetricSetFields: fields,
	})
}
