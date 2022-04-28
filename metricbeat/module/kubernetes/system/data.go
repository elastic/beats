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

package system

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

	for _, syscontainer := range node.SystemContainers {
		containerEvent := mapstr.M{
			mb.ModuleDataKey: mapstr.M{
				"node": mapstr.M{
					"name": node.NodeName,
				},
			},
			"container": syscontainer.Name,
			"cpu": mapstr.M{
				"usage": mapstr.M{
					"nanocores": syscontainer.CPU.UsageNanoCores,
					"core": mapstr.M{
						"ns": syscontainer.CPU.UsageCoreNanoSeconds,
					},
				},
			},
			"memory": mapstr.M{
				"usage": mapstr.M{
					"bytes": syscontainer.Memory.UsageBytes,
				},
				"workingset": mapstr.M{
					"bytes": syscontainer.Memory.WorkingSetBytes,
				},
				"rss": mapstr.M{
					"bytes": syscontainer.Memory.RssBytes,
				},
				"pagefaults":      syscontainer.Memory.PageFaults,
				"majorpagefaults": syscontainer.Memory.MajorPageFaults,
			},
		}

		if syscontainer.StartTime != "" {
			util.ShouldPut(containerEvent, "start_time", syscontainer.StartTime, logger)
		}

		events = append(events, containerEvent)
	}

	return events, nil
}
