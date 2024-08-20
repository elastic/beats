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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

func TestEventMapping(t *testing.T) {
	var m *MetricSet
	var ResorcePoolTest = mo.ResourcePool{
		Summary: &types.ResourcePoolSummary{
			Name: "resourcepool-test",
		},
		ManagedEntity: mo.ManagedEntity{
			OverallStatus: "green",
		},
	}

	var metricDataTest = metricData{
		perfMetrics: map[string]interface{}{
			"cpu.usage.average":         int64(100),
			"rescpu.actav1.latest":      int64(10),
			"rescpu.actpk1.latest":      int64(10),
			"cpu.usagemhz.average":      int64(100),
			"mem.usage.average":         int64(10),
			"mem.shared.average":        int64(10),
			"mem.swapin.average":        int64(10),
			"cpu.cpuentitlement.latest": int64(100),
			"mem.mementitlement.latest": int64(10),
		},
		assetsName: assetNames{
			outputVmNames: []string{"vm-1", "vm-2"},
		},
	}

	event := m.eventMapping(ResorcePoolTest, &metricDataTest)

	vmName, _ := event.GetValue("vm.names")
	assert.EqualValues(t, metricDataTest.assetsName.outputVmNames, vmName)

	vmCount, _ := event.GetValue("vm.count")
	assert.EqualValues(t, vmCount, len(metricDataTest.assetsName.outputVmNames))

	status, _ := event.GetValue("status")
	assert.EqualValues(t, "green", status)

	name := event["name"].(string)
	assert.EqualValues(t, name, "resourcepool-test")

	cpuusageAverage, _ := event.GetValue("cpu.usage.pct")
	assert.EqualValues(t, metricDataTest.perfMetrics["cpu.usage.average"], cpuusageAverage)

	cpuUsageMhz, _ := event.GetValue("cpu.usage.mhz")
	assert.EqualValues(t, metricDataTest.perfMetrics["cpu.usagemhz.average"], cpuUsageMhz)

	cpuEntitlement, _ := event.GetValue("cpu.entitlement.mhz")
	assert.EqualValues(t, metricDataTest.perfMetrics["cpu.cpuentitlement.latest"], cpuEntitlement)

	cpuActiveAverage, _ := event.GetValue("cpu.active.average.pct")
	assert.EqualValues(t, metricDataTest.perfMetrics["rescpu.actav1.latest"], cpuActiveAverage)

	cpuActiveMax, _ := event.GetValue("cpu.active.max.pct")
	assert.EqualValues(t, metricDataTest.perfMetrics["rescpu.actpk1.average"], cpuActiveMax)

	memUsage, _ := event.GetValue("memory.usage.pct")
	assert.EqualValues(t, metricDataTest.perfMetrics["mem.usage.average"], memUsage)

	memShared, _ := event.GetValue("memory.shared.bytes")
	assert.EqualValues(t, metricDataTest.perfMetrics["mem.shared.average"].(int64)*1000, memShared)

	memSwap, _ := event.GetValue("memory.swap.bytes")
	assert.EqualValues(t, metricDataTest.perfMetrics["mem.swapin.average"].(int64)*1000, memSwap)

	memEntitlement, _ := event.GetValue("memory.entitlement.mhz")
	assert.EqualValues(t, metricDataTest.perfMetrics["mem.mementitlement.latest"], memEntitlement)
}
