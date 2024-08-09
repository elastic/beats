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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

func TestEventMapping(t *testing.T) {
	var m *MetricSet
	var DatastoreTest = mo.Datastore{
		Summary: types.DatastoreSummary{
			Name:      "datastore-test",
			Type:      "local",
			Capacity:  5000000,
			FreeSpace: 5000000,
		},
		ManagedEntity: mo.ManagedEntity{
			OverallStatus: "green",
		},
		Host: []types.DatastoreHostMount{},
		Vm: []types.ManagedObjectReference{
			{Type: "VirtualMachine", Value: "vm-test"},
		},
	}

	event := m.eventMapping(DatastoreTest, &PerformanceMetrics{})

	VmCount, _ := event.GetValue("vm.count")
	assert.EqualValues(t, 1, VmCount)

	capacityTotal, _ := event.GetValue("capacity.total.bytes")
	assert.EqualValues(t, 5000000, capacityTotal)

	capacityFree, _ := event.GetValue("capacity.free.bytes")
	assert.EqualValues(t, 5000000, capacityFree)

	capacityUsed, _ := event.GetValue("capacity.used.bytes")
	assert.EqualValues(t, 0, capacityUsed)

	capacityUsedPct, _ := event.GetValue("capacity.used.pct")
	assert.EqualValues(t, 0, capacityUsedPct)

}
