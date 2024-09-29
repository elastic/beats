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

package network

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

func TestEventMapping(t *testing.T) {
	m := &NetworkMetricSet{}
	networkTest := mo.Network{
		Summary: &types.NetworkSummary{
			Accessible: true,
		},
		ManagedEntity: mo.ManagedEntity{
			OverallStatus: "green",
			ConfigStatus:  "green",
			ExtensibleManagedObject: mo.ExtensibleManagedObject{
				Self: types.ManagedObjectReference{
					Type: "Network",
				},
			},
		},
	}

	metricDataTest := metricData{
		assetNames: assetNames{
			outputHostNames: []string{"Host1"},
			outputVmNames:   []string{"VM1"},
		},
	}

	event := m.mapEvent(networkTest, &metricDataTest)

	name, _ := event.GetValue("name")
	assert.NotNil(t, name)

	status, _ := event.GetValue("status")
	assert.Equal(t, types.ManagedEntityStatus("green"), status)

	configStatus, _ := event.GetValue("config.status")
	assert.Equal(t, types.ManagedEntityStatus("green"), configStatus)

	accessible, _ := event.GetValue("accessible")
	assert.True(t, accessible.(bool))

	hostNames, _ := event.GetValue("host.names")
	assert.Equal(t, metricDataTest.assetNames.outputHostNames, hostNames)
	hostCount, _ := event.GetValue("host.count")
	assert.GreaterOrEqual(t, hostCount, 0)

	vmNames, _ := event.GetValue("vm.names")
	assert.Equal(t, metricDataTest.assetNames.outputVmNames, vmNames)
	vmCount, _ := event.GetValue("vm.count")
	assert.GreaterOrEqual(t, vmCount, 0)

	typeName, _ := event.GetValue("type")
	assert.Equal(t, "Network", typeName)
}
