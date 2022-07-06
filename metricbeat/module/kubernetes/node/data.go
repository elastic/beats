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

package node

import (
	"encoding/json"
	"fmt"

	"github.com/elastic/beats/v7/metricbeat/module/kubernetes"
	"github.com/elastic/beats/v7/metricbeat/module/kubernetes/util"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func eventMapping(content []byte, logger *logp.Logger) (mapstr.M, error) {
	var summary kubernetes.Summary
	err := json.Unmarshal(content, &summary)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal json response: %w", err)
	}

	node := summary.Node
	nodeEvent := mapstr.M{
		"name": node.NodeName,

		"cpu": mapstr.M{
			"usage": mapstr.M{
				"nanocores": node.CPU.UsageNanoCores,
				"core": mapstr.M{
					"ns": node.CPU.UsageCoreNanoSeconds,
				},
			},
		},

		"memory": mapstr.M{
			"available": mapstr.M{
				"bytes": node.Memory.AvailableBytes,
			},
			"usage": mapstr.M{
				"bytes": node.Memory.UsageBytes,
			},
			"workingset": mapstr.M{
				"bytes": node.Memory.WorkingSetBytes,
			},
			"rss": mapstr.M{
				"bytes": node.Memory.RssBytes,
			},
			"pagefaults":      node.Memory.PageFaults,
			"majorpagefaults": node.Memory.MajorPageFaults,
		},

		"network": mapstr.M{
			"rx": mapstr.M{
				"bytes":  node.Network.RxBytes,
				"errors": node.Network.RxErrors,
			},
			"tx": mapstr.M{
				"bytes":  node.Network.TxBytes,
				"errors": node.Network.TxErrors,
			},
		},

		"fs": mapstr.M{
			"available": mapstr.M{
				"bytes": node.Fs.AvailableBytes,
			},
			"capacity": mapstr.M{
				"bytes": node.Fs.CapacityBytes,
			},
			"used": mapstr.M{
				"bytes": node.Fs.UsedBytes,
			},
			"inodes": mapstr.M{
				"used":  node.Fs.InodesUsed,
				"free":  node.Fs.InodesFree,
				"count": node.Fs.Inodes,
			},
		},

		"runtime": mapstr.M{
			"imagefs": mapstr.M{
				"available": mapstr.M{
					"bytes": node.Runtime.ImageFs.AvailableBytes,
				},
				"capacity": mapstr.M{
					"bytes": node.Runtime.ImageFs.CapacityBytes,
				},
				"used": mapstr.M{
					"bytes": node.Runtime.ImageFs.UsedBytes,
				},
			},
		},
	}

	if node.StartTime != "" {
		util.ShouldPut(nodeEvent, "start_time", node.StartTime, logger)
	}

	return nodeEvent, nil
}
