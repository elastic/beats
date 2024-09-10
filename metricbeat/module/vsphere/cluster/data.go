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

func (m *ClusterMetricSet) mapEvent(cl mo.ClusterComputeResource, data *metricData) mapstr.M {
	event := mapstr.M{
		"host": mapstr.M{
			"count": len(data.assetNames.outputHostNames),
			"names": data.assetNames.outputHostNames,
		},
		"datastore": mapstr.M{
			"count": len(data.assetNames.outputDatastoreNames),
			"names": data.assetNames.outputDatastoreNames,
		},
		"network": mapstr.M{
			"count": len(data.assetNames.outputNetworkNames),
			"names": data.assetNames.outputNetworkNames,
		},
		"name": cl.Name,
	}

	if len(data.alertNames) > 0 {
		event.Put("alert.names", data.alertNames)
	}

	if cl.Configuration.DasConfig.Enabled != nil {
		event.Put("das_config.enabled", *cl.Configuration.DasConfig.Enabled)
	}

	if cl.Configuration.DasConfig.AdmissionControlEnabled != nil {
		event.Put("das_config.admission.control.enabled", *cl.Configuration.DasConfig.AdmissionControlEnabled)
	}

	return event
}
