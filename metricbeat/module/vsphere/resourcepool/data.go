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

func (m *MetricSet) eventMapping(rp mo.ResourcePool, data *metricData) mapstr.M {
	event := mapstr.M{
		"name":   rp.Summary.GetResourcePoolSummary().Name,
		"status": rp.OverallStatus,
	}
	mapPerfMetricToEvent(event, data.perfMetrics)

	if len(data.assetsName.outputVmNames) > 0 {
		event.Put("vm.names", data.assetsName.outputVmNames)
		event.Put("vm.count", len(data.assetsName.outputVmNames))
	}
	return event
}

func mapPerfMetricToEvent(event mapstr.M, perfMetrics map[string]interface{}) {
	if val, exist := perfMetrics["cpu.usage.average"]; exist {
		event.Put("cpu.usage.pct", val)
	}
	if val, exist := perfMetrics["cpu.usagemhz.average"]; exist {
		event.Put("cpu.usage.mhz", val)
	}
	if val, exist := perfMetrics["cpu.cpuentitlement.latest"]; exist {
		event.Put("cpu.entitlement.mhz", val)
	}
	if val, exist := perfMetrics["rescpu.actav1.latest"]; exist {
		event.Put("cpu.active.average.pct", val)
	}
	if val, exist := perfMetrics["rescpu.actpk1.average"]; exist {
		event.Put("cpu.active.max.pct", val)
	}

	if val, exist := perfMetrics["mem.usage.average"]; exist {
		event.Put("memory.usage.pct", val)
	}
	if val, exist := perfMetrics["mem.shared.average"]; exist {
		event.Put("memory.shared.bytes", val.(int64)*1000)
	}
	if val, exist := perfMetrics["mem.swapin.average"]; exist {
		event.Put("memory.swap.bytes", val.(int64)*1000)
	}
	if val, exist := perfMetrics["mem.mementitlement.latest"]; exist {
		event.Put("memory.entitlement.mhz", val)
	}
}
