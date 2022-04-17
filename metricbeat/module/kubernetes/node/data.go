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

	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/logp"
	"github.com/menderesk/beats/v7/metricbeat/module/kubernetes"
	"github.com/menderesk/beats/v7/metricbeat/module/kubernetes/util"
)

func eventMapping(content []byte, logger *logp.Logger) (common.MapStr, error) {
	var summary kubernetes.Summary
	err := json.Unmarshal(content, &summary)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal json response: %w", err)
	}

	node := summary.Node
	nodeEvent := common.MapStr{
		"name": node.NodeName,

		"cpu": common.MapStr{
			"usage": common.MapStr{
				"nanocores": node.CPU.UsageNanoCores,
				"core": common.MapStr{
					"ns": node.CPU.UsageCoreNanoSeconds,
				},
			},
		},

		"memory": common.MapStr{
			"available": common.MapStr{
				"bytes": node.Memory.AvailableBytes,
			},
			"usage": common.MapStr{
				"bytes": node.Memory.UsageBytes,
			},
			"workingset": common.MapStr{
				"bytes": node.Memory.WorkingSetBytes,
			},
			"rss": common.MapStr{
				"bytes": node.Memory.RssBytes,
			},
			"pagefaults":      node.Memory.PageFaults,
			"majorpagefaults": node.Memory.MajorPageFaults,
		},

		"network": common.MapStr{
			"rx": common.MapStr{
				"bytes":  node.Network.RxBytes,
				"errors": node.Network.RxErrors,
			},
			"tx": common.MapStr{
				"bytes":  node.Network.TxBytes,
				"errors": node.Network.TxErrors,
			},
		},

		"fs": common.MapStr{
			"available": common.MapStr{
				"bytes": node.Fs.AvailableBytes,
			},
			"capacity": common.MapStr{
				"bytes": node.Fs.CapacityBytes,
			},
			"used": common.MapStr{
				"bytes": node.Fs.UsedBytes,
			},
			"inodes": common.MapStr{
				"used":  node.Fs.InodesUsed,
				"free":  node.Fs.InodesFree,
				"count": node.Fs.Inodes,
			},
		},

		"runtime": common.MapStr{
			"imagefs": common.MapStr{
				"available": common.MapStr{
					"bytes": node.Runtime.ImageFs.AvailableBytes,
				},
				"capacity": common.MapStr{
					"bytes": node.Runtime.ImageFs.CapacityBytes,
				},
				"used": common.MapStr{
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
