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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestEventMapping(t *testing.T) {
	var m MetricSet

	VirtualMachineTest := mo.VirtualMachine{
		Summary: types.VirtualMachineSummary{
			OverallStatus: "green",
			Config: types.VirtualMachineConfigSummary{
				Name:           "localhost.localdomain",
				GuestFullName:  "otherGuest",
				MemorySizeMB:   70,
				CpuReservation: 2294, // MHz
			},
			QuickStats: types.VirtualMachineQuickStats{
				UptimeSeconds:    10,
				OverallCpuUsage:  30, // MHz
				GuestMemoryUsage: 40, // MB
				HostMemoryUsage:  50, // MB
			},
		},
	}

	data := VMData{
		VM:             VirtualMachineTest,
		HostID:         "host-1234",
		HostName:       "test-host",
		NetworkNames:   []string{"network-1", "network-2"},
		DatastoreNames: []string{"ds1", "ds2"},
		CustomFields: mapstr.M{
			"customField1": "value1",
			"customField2": "value2",
		},
	}

	event := m.eventMapping(data)

	// Test CPU values
	cpuUsed, _ := event.GetValue("cpu.used.mhz")
	assert.EqualValues(t, 30, cpuUsed)

	cpuTotal, _ := event.GetValue("cpu.total.mhz")
	assert.EqualValues(t, 2294, cpuTotal)

	cpuFree, _ := event.GetValue("cpu.free.mhz")
	assert.EqualValues(t, 2264, cpuFree)

	// Test Memory values
	memoryUsed, _ := event.GetValue("memory.used.guest.bytes")
	assert.EqualValues(t, int64(40*1024*1024), memoryUsed)

	memoryHostUsed, _ := event.GetValue("memory.used.host.bytes")
	assert.EqualValues(t, int64(50*1024*1024), memoryHostUsed)

	memoryTotal, _ := event.GetValue("memory.total.guest.bytes")
	assert.EqualValues(t, int64(70*1024*1024), memoryTotal)

	memoryFree, _ := event.GetValue("memory.free.guest.bytes")
	assert.EqualValues(t, int64(30*1024*1024), memoryFree)

	// Test custom fields
	customField1, _ := event.GetValue("custom_fields.customField1")
	assert.EqualValues(t, "value1", customField1)

	customField2, _ := event.GetValue("custom_fields.customField2")
	assert.EqualValues(t, "value2", customField2)

	// Test network names
	networkNames, _ := event.GetValue("network_names")
	assert.ElementsMatch(t, []string{"network-1", "network-2"}, networkNames)

	// Test datastore names
	datastoreNames, _ := event.GetValue("datastore.names")
	assert.ElementsMatch(t, []string{"ds1", "ds2"}, datastoreNames)
}
