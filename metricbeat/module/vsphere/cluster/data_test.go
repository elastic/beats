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

package cluster

import (
	"testing"

	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/stretchr/testify/assert"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

func TestEventMapping(t *testing.T) {
	var m *MetricSet
	var ClusterTest = mo.ClusterComputeResource{
		Configuration: types.ClusterConfigInfo{
			DasConfig: types.ClusterDasConfigInfo{
				Enabled:                 types.NewBool(false),
				AdmissionControlEnabled: types.NewBool(true),
			},
		},
	}

	var assetNames = assetNames{
		outputHsNames: []string{"Host_0"},
		outputDsNames: []string{"Datastore_0"},
		outputNtNames: []string{"Network_0"},
	}

	outputEvent := m.eventMapping(ClusterTest, &assetNames)
	testEvent := mapstr.M{
		"das_config": mapstr.M{
			"enabled": false,
			"admission": mapstr.M{
				"control": mapstr.M{
					"enabled": true,
				},
			},
		},
		"host": mapstr.M{
			"count": 1,
			"names": []string{"Host_0"},
		},
		"datastore": mapstr.M{
			"count": 1,
			"names": []string{"Datastore_0"},
		},
		"network": mapstr.M{
			"count": 1,
			"names": []string{"Network_0"},
		},
	}

	assert.Exactly(t, outputEvent, testEvent)

}