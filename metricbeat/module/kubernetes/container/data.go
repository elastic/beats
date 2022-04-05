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
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/kubernetes"
	"github.com/elastic/beats/v7/metricbeat/module/kubernetes/util"
)

func eventMapping(content []byte, perfMetrics *util.PerfMetricsCache, logger *logp.Logger) ([]common.MapStr, error) {
	events := []common.MapStr{}
	var summary kubernetes.Summary

	err := json.Unmarshal(content, &summary)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal json response: %w", err)
	}

	node := summary.Node
	nodeCores := perfMetrics.NodeCoresAllocatable.Get(node.NodeName)
	nodeMem := perfMetrics.NodeMemAllocatable.Get(node.NodeName)
	for _, pod := range summary.Pods {
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
				util.ShouldPut(containerEvent, "start_time", container.StartTime, logger)
			}

			if nodeCores > 0 {
				util.ShouldPut(containerEvent, "cpu.usage.node.pct", float64(container.CPU.UsageNanoCores)/1e9/nodeCores, logger)
			}

			if nodeMem > 0 {
				util.ShouldPut(containerEvent, "memory.usage.node.pct", float64(container.Memory.UsageBytes)/nodeMem, logger)
			}

			cuid := util.ContainerUID(pod.PodRef.Namespace, pod.PodRef.Name, container.Name)
			coresLimit := perfMetrics.ContainerCoresLimit.GetWithDefault(cuid, nodeCores)
			memLimit := perfMetrics.ContainerMemLimit.GetWithDefault(cuid, nodeMem)

			if coresLimit > 0 {
				util.ShouldPut(containerEvent, "cpu.usage.limit.pct", float64(container.CPU.UsageNanoCores)/1e9/coresLimit, logger)
			}

			if memLimit > 0 {
				util.ShouldPut(containerEvent, "memory.usage.limit.pct", float64(container.Memory.UsageBytes)/memLimit, logger)
				util.ShouldPut(containerEvent, "memory.workingset.limit.pct", float64(container.Memory.WorkingSetBytes)/memLimit, logger)
			}

			events = append(events, containerEvent)
		}

	}

	return events, nil
}

// ecsfields maps container events fields to container ecs fields
func ecsfields(containerEvent common.MapStr, logger *logp.Logger) common.MapStr {
	ecsfields := common.MapStr{}

	name, err := containerEvent.GetValue("name")
	if err == nil {
		util.ShouldPut(ecsfields, "name", name, logger)
	}

	cpuUsage, err := containerEvent.GetValue("cpu.usage.node.pct")
	if err == nil {
		util.ShouldPut(ecsfields, "cpu.usage", cpuUsage, logger)
	}

	memUsage, err := containerEvent.GetValue("memory.usage.node.pct")
	if err == nil {
		util.ShouldPut(ecsfields, "memory.usage", memUsage, logger)
	}

	return ecsfields
}
