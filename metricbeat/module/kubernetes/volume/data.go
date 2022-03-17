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

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/kubernetes"
)

func eventMapping(content []byte) ([]common.MapStr, error) {
	events := []common.MapStr{}

	var summary kubernetes.Summary
	err := json.Unmarshal(content, &summary)
	if err != nil {
		return nil, fmt.Errorf("Cannot unmarshal json response: %s", err)
	}

	node := summary.Node
	for _, pod := range summary.Pods {
		for _, volume := range pod.Volume {
			volumeEvent := common.MapStr{
				mb.ModuleDataKey: common.MapStr{
					"namespace": pod.PodRef.Namespace,
					"node": common.MapStr{
						"name": node.NodeName,
					},
					"pod": common.MapStr{
						"name": pod.PodRef.Name,
					},
				},

				"name": volume.Name,
				"fs": common.MapStr{
					"available": common.MapStr{
						"bytes": volume.AvailableBytes,
					},
					"capacity": common.MapStr{
						"bytes": volume.CapacityBytes,
					},
					"used": common.MapStr{
						"bytes": volume.UsedBytes,
					},
					"inodes": common.MapStr{
						"used":  volume.InodesUsed,
						"free":  volume.InodesFree,
						"count": volume.Inodes,
					},
				},
			}
			if volume.CapacityBytes > 0 {
				volumeEvent.Put("fs.used.pct", float64(volume.UsedBytes)/float64(volume.CapacityBytes))
			}
			if volume.Inodes > 0 {
				volumeEvent.Put("fs.inodes.pct", float64(volume.InodesUsed)/float64(volume.Inodes))
			}
			events = append(events, volumeEvent)
		}

	}
	return events, nil
}
