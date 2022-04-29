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

package volume

import (
	"encoding/json"
	"fmt"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/kubernetes"
	"github.com/elastic/beats/v7/metricbeat/module/kubernetes/util"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func eventMapping(content []byte, logger *logp.Logger) ([]mapstr.M, error) {
	events := []mapstr.M{}

	var summary kubernetes.Summary
	err := json.Unmarshal(content, &summary)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal json response: %w", err)
	}

	node := summary.Node
	for _, pod := range summary.Pods {
		for _, volume := range pod.Volume {
			volumeEvent := mapstr.M{
				mb.ModuleDataKey: mapstr.M{
					"namespace": pod.PodRef.Namespace,
					"node": mapstr.M{
						"name": node.NodeName,
					},
					"pod": mapstr.M{
						"name": pod.PodRef.Name,
					},
				},

				"name": volume.Name,
				"fs": mapstr.M{
					"available": mapstr.M{
						"bytes": volume.AvailableBytes,
					},
					"capacity": mapstr.M{
						"bytes": volume.CapacityBytes,
					},
					"used": mapstr.M{
						"bytes": volume.UsedBytes,
					},
					"inodes": mapstr.M{
						"used":  volume.InodesUsed,
						"free":  volume.InodesFree,
						"count": volume.Inodes,
					},
				},
			}
			if volume.CapacityBytes > 0 {
				util.ShouldPut(volumeEvent, "fs.used.pct", float64(volume.UsedBytes)/float64(volume.CapacityBytes), logger)
			}
			if volume.Inodes > 0 {
				util.ShouldPut(volumeEvent, "fs.inodes.pct", float64(volume.InodesUsed)/float64(volume.Inodes), logger)
			}
			events = append(events, volumeEvent)
		}

	}
	return events, nil
}
