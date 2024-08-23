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
	"github.com/vmware/govmomi/vim25/mo"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

func (m *NetworkMetricSet) mapEvent(net mo.Network, data *metricData) mapstr.M {
	event := mapstr.M{}

	event.Put("name", net.Name)
	event.Put("status", net.OverallStatus)
	event.Put("accessible", net.Summary.GetNetworkSummary().Accessible)
	event.Put("config.status", net.ConfigStatus)
	event.Put("type", net.Self.Type)

	if len(data.assetsName.outputHostNames) > 0 {
		event.Put("host.names", data.assetsName.outputHostNames)
		event.Put("host.count", len(data.assetsName.outputHostNames))
	}

	if len(data.assetsName.outputVmNames) > 0 {
		event.Put("vm.names", data.assetsName.outputVmNames)
		event.Put("vm.count", len(data.assetsName.outputVmNames))
	}

	return event
}
