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

package resourcepool

import (
	"github.com/vmware/govmomi/vim25/mo"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

func (m *MetricSet) eventMapping(rp mo.ResourcePool, perfMertics *PerformanceMetrics) mapstr.M {
	event := mapstr.M{
		"name":   rp.Summary.GetResourcePoolSummary().Name,
		"status": rp.OverallStatus,
		"cpu": mapstr.M{
			"entitlement": mapstr.M{
				"mhz": perfMertics.CPUEntitlementLatest,
			},
			"usage": mapstr.M{
				"pct": perfMertics.CPUUsageAverage,
				"mhz": perfMertics.CPUUsageMHzAverage,
			},
			"active": mapstr.M{
				"average": mapstr.M{
					"pct": perfMertics.ResCPUActAv1Latest,
				},
				"max": mapstr.M{
					"pct": perfMertics.ResCPUActPk1Latest,
				},
			},
		},
		"memory": mapstr.M{
			"entitlement": mapstr.M{
				"mhz": perfMertics.MemEntitlementLatest,
			},
			"usage": mapstr.M{
				"pct": perfMertics.MemUsageAverage,
			},
			"shared": mapstr.M{
				"bytes": int64(perfMertics.MemSharedAverage) * 1000,
			},
			"swap": mapstr.M{
				"bytes": int64(perfMertics.MemSwapInAverage) * 1000,
			},
		},
	}

	return event
}
