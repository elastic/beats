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

package container

import (
	"time"

	"github.com/elastic/beats/metricbeat/mb"

	"github.com/docker/docker/api/types"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/module/docker"
)

func eventMapping(cont *types.Container, dedot bool) (event mb.Event) {
	mapOfRootFieldsResults := make(map[string]interface{})
	mapOfRootFieldsResults["service.name"] = "container"

	// Container fields in ECS
	mapOfRootFieldsResults["container.id"] = cont.ID
	mapOfRootFieldsResults["container.image.name"] = cont.Image
	mapOfRootFieldsResults["container.name"] = docker.ExtractContainerName(cont.Names)

	// Docker container metrics
	mapOfMetricSetFieldResults := make(map[string]interface{})
	mapOfMetricSetFieldResults["created"] = common.Time(time.Unix(cont.Created, 0))
	mapOfMetricSetFieldResults["command"] = cont.Command
	mapOfMetricSetFieldResults["ip_addresses"] = extractIPAddresses(cont.NetworkSettings)
	mapOfMetricSetFieldResults["size.root_fs"] = cont.SizeRootFs
	mapOfMetricSetFieldResults["size.rw"] = cont.SizeRw
	mapOfMetricSetFieldResults["status"] = cont.Status

	labels := docker.DeDotLabels(cont.Labels, dedot)
	if len(labels) > 0 {
		mapOfRootFieldsResults["container.labels"] = labels
	}

	event.RootFields = mapOfRootFieldsResults
	event.MetricSetFields = mapOfMetricSetFieldResults
	return event
}

func extractIPAddresses(networks *types.SummaryNetworkSettings) []string {
	ipAddresses := make([]string, 0, len(networks.Networks))
	for _, network := range networks.Networks {
		ipAddresses = append(ipAddresses, network.IPAddress)
	}
	return ipAddresses
}
