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

	"github.com/elastic/elastic-agent-libs/mapstr"
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

	var metricDataTest = metricData{
		perfMetrics: map[string]interface{}{
			"datastore.read.average":              int64(100),
			"datastore.write.average":             int64(200),
			"datastore.datastoreIops.average":     int64(10),
			"datastore.totalReadLatency.average":  int64(100),
			"datastore.totalWriteLatency.average": int64(100),
		},
		assetsName: assetNames{
			outputHsNames: []string{"DC3_H0"},
			outputVmNames: []string{"DC3_H0_VM0"},
		},
	}

	outputEvent := m.eventMapping(DatastoreTest, &metricDataTest)
	testEvent := mapstr.M{
		"fstype": "local",
		"status": "green",
		"iops":   int64(10),
		"host": mapstr.M{
			"count": 1,
			"names": []string{"DC3_H0"},
		},
		"vm": mapstr.M{
			"count": 1,
			"names": []string{"DC3_H0_VM0"},
		},
		"read": mapstr.M{
			"bytes": int64(100000),
			"latency": mapstr.M{
				"total": mapstr.M{
					"ms": int64(100),
				},
			},
		},
		"write": mapstr.M{
			"bytes": int64(200000),
			"latency": mapstr.M{
				"total": mapstr.M{
					"ms": int64(100),
				},
			},
		},
		"capacity": mapstr.M{
			"free": mapstr.M{
				"bytes": int64(5000000),
			},
			"total": mapstr.M{
				"bytes": int64(5000000),
			},
			"used": mapstr.M{
				"bytes": int64(0),
				"pct":   float64(0),
			},
		},
	}

	assert.Exactly(t, outputEvent, testEvent)

}
