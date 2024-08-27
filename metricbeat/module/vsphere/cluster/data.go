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
	"github.com/vmware/govmomi/vim25/mo"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

func (m *MetricSet) eventMapping(cl mo.ClusterComputeResource, data *assetNames) mapstr.M {
	event := mapstr.M{
		"das_config": mapstr.M{
			"enabled": *cl.Configuration.DasConfig.Enabled,
			"admission": mapstr.M{
				"control": mapstr.M{
					"enabled": *cl.Configuration.DasConfig.AdmissionControlEnabled,
				},
			},
		},
	}

	event.Put("host.count", len(data.outputHsNames))
	if len(data.outputHsNames) > 0 {
		event.Put("host.names", data.outputHsNames)
	}

	event.Put("datastore.count", len(data.outputDsNames))
	if len(data.outputDsNames) > 0 {
		event.Put("datastore.names", data.outputDsNames)
	}

	event.Put("network.count", len(data.outputNtNames))
	if len(data.outputNtNames) > 0 {
		event.Put("network.names", data.outputNtNames)
	}

	return event
}
