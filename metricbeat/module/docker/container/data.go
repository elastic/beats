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

	"github.com/docker/docker/api/types"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/docker"
)

func eventsMapping(r mb.ReporterV2, containersList []types.Container, dedot bool) {
	for _, container := range containersList {
		eventMapping(r, &container, dedot)
	}
}

func eventMapping(r mb.ReporterV2, cont *types.Container, dedot bool) {
	event := common.MapStr{
		"container": common.MapStr{
			"id": cont.ID,
			"image": common.MapStr{
				"name": cont.Image,
			},
			"name":    docker.ExtractContainerName(cont.Names),
			"runtime": "docker",
		},
		"docker": common.MapStr{
			"container": common.MapStr{
				"created":      common.Time(time.Unix(cont.Created, 0)),
				"command":      cont.Command,
				"ip_addresses": extractIPAddresses(cont.NetworkSettings),
				"size": common.MapStr{
					"root_fs": cont.SizeRootFs,
					"rw":      cont.SizeRw,
				},
				"status": cont.Status,
			},
		},
	}

	labels := docker.DeDotLabels(cont.Labels, dedot)

	if len(labels) > 0 {
		event.Put("docker.container.labels", labels)
	}

	r.Event(mb.Event{
		RootFields: event,
	})
}

func extractIPAddresses(networks *types.SummaryNetworkSettings) []string {
	// Handle alternate platforms like VMWare's VIC that might not have this data.
	if networks == nil {
		return []string{}
	}
	ipAddresses := make([]string, 0, len(networks.Networks))
	for _, network := range networks.Networks {
		if len(network.IPAddress) > 0 {
			ipAddresses = append(ipAddresses, network.IPAddress)
		}
	}
	return ipAddresses
}
