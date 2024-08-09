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
			Runtime: types.ResourcePoolRuntimeInfo{
				OverallStatus: "green",
			},
		},
	}

	event := m.eventMapping(ResorcePoolTest, &PerformanceMetrics{})

	cpuUsed, _ := event.GetValue("name")
	assert.EqualValues(t, "resourcepool-test", cpuUsed)

	cpuTotal, _ := event.GetValue("status")
	assert.EqualValues(t, "green", cpuTotal)
}
