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
	"encoding/json"
	"fmt"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/kubernetes"
	"github.com/elastic/beats/v7/metricbeat/module/kubernetes/util"
)

func eventMapping(content []byte, metricsRepo *util.MetricsRepo) ([]common.MapStr, error) {
	events := []common.MapStr{}
	var summary kubernetes.Summary

	err := json.Unmarshal(content, &summary)
	if err != nil {
		return nil, fmt.Errorf("Cannot unmarshal json response: %s", err)
	}

	node := summary.Node

	nodeCores := 0.0
	nodeMem := 0.0

	nodeStore := metricsRepo.GetNodeStore(node.NodeName)
	nodeMetrics := nodeStore.GetNodeMetrics()
	if nodeMetrics.CoresAllocatable != nil {
		nodeCores = nodeMetrics.CoresAllocatable.Value
	}
	if nodeMetrics.MemoryAllocatable != nil {
		nodeMem = nodeMetrics.MemoryAllocatable.Value
	}

	for _, pod := range summary.Pods {
		podId := util.NewPodId(pod.PodRef.Namespace, pod.PodRef.Name)
		podStore := nodeStore.GetPodStore(podId)

		for _, container := range pod.Containers {
			containerEvent := common.MapStr{
				mb.ModuleDataKey: common.MapStr{
					"namespace": pod.PodRef.Namespace,
					"node": common.MapStr{
						"name": node.NodeName,
					},
					"pod": common.MapStr{
						"name": pod.PodRef.Name,
					},
				},

				"name": container.Name,

				"cpu": common.MapStr{
					"usage": common.MapStr{
						"nanocores": container.CPU.UsageNanoCores,
						"core": common.MapStr{
							"ns": container.CPU.UsageCoreNanoSeconds,
						},
					},
				},

				"memory": common.MapStr{
					"available": common.MapStr{
						"bytes": container.Memory.AvailableBytes,
					},
					"usage": common.MapStr{
						"bytes": container.Memory.UsageBytes,
					},
					"workingset": common.MapStr{
						"bytes": container.Memory.WorkingSetBytes,
					},
					"rss": common.MapStr{
						"bytes": container.Memory.RssBytes,
					},
					"pagefaults":      container.Memory.PageFaults,
					"majorpagefaults": container.Memory.MajorPageFaults,
				},

				"rootfs": common.MapStr{
					"available": common.MapStr{
						"bytes": container.Rootfs.AvailableBytes,
					},
					"capacity": common.MapStr{
						"bytes": container.Rootfs.CapacityBytes,
					},
					"used": common.MapStr{
						"bytes": container.Rootfs.UsedBytes,
					},
					"inodes": common.MapStr{
						"used": container.Rootfs.InodesUsed,
					},
				},

				"logs": common.MapStr{
					"available": common.MapStr{
						"bytes": container.Logs.AvailableBytes,
					},
					"capacity": common.MapStr{
						"bytes": container.Logs.CapacityBytes,
					},
					"used": common.MapStr{
						"bytes": container.Logs.UsedBytes,
					},
					"inodes": common.MapStr{
						"used":  container.Logs.InodesUsed,
						"free":  container.Logs.InodesFree,
						"count": container.Logs.Inodes,
					},
				},
			}

			if container.StartTime != "" {
				containerEvent.Put("start_time", container.StartTime)
			}

			if nodeCores > 0 {
				containerEvent.Put("cpu.usage.node.pct", float64(container.CPU.UsageNanoCores)/1e9/nodeCores)
			}

			if nodeMem > 0 {
				containerEvent.Put("memory.usage.node.pct", float64(container.Memory.UsageBytes)/nodeMem)
			}

			containerStore := podStore.GetContainerStore(container.Name)
			containerMetrics := containerStore.GetContainerMetrics()

			containerCoresLimit := nodeCores
			if containerMetrics.CoresLimit != nil {
				containerCoresLimit = containerMetrics.CoresLimit.Value
			}

			containerMemLimit := nodeMem
			if containerMetrics.MemoryLimit != nil {
				containerMemLimit = containerMetrics.MemoryLimit.Value
			}

			// NOTE:
			// we don't currently check if `containerMemLimit` > `nodeMem` as we do in `kubernetes/pod/data.go`.
			// There we do check, since if a container doesn't have a limit set, it will inherit the node limits and the sum of all
			// the container limits can be greater than the node limits. We assume here the user can set correct limits on containers.

			if containerCoresLimit > 0 {
				containerEvent.Put("cpu.usage.limit.pct", float64(container.CPU.UsageNanoCores)/1e9/containerCoresLimit)
			}

			if containerMemLimit > 0 {
				containerEvent.Put("memory.usage.limit.pct", float64(container.Memory.UsageBytes)/containerMemLimit)
			}

			events = append(events, containerEvent)
		}

	}

	return events, nil
}
