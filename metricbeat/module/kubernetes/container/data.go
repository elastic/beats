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

	kubernetes2 "github.com/elastic/beats/v7/libbeat/autodiscover/providers/kubernetes"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/kubernetes"
	"github.com/elastic/beats/v7/metricbeat/module/kubernetes/util"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func eventMapping(content []byte, metricsRepo *util.MetricsRepo, logger *logp.Logger) ([]mapstr.M, error) {
	events := []mapstr.M{}
	var summary kubernetes.Summary

	err := json.Unmarshal(content, &summary)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal json response: %w", err)
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
			containerEvent := mapstr.M{
				mb.ModuleDataKey: mapstr.M{
					"namespace": pod.PodRef.Namespace,
					"node": mapstr.M{
						"name": node.NodeName,
					},
					"pod": mapstr.M{
						"name": pod.PodRef.Name,
					},
				},

				"name": container.Name,

				"cpu": mapstr.M{
					"usage": mapstr.M{
						"nanocores": container.CPU.UsageNanoCores,
						"core": mapstr.M{
							"ns": container.CPU.UsageCoreNanoSeconds,
						},
					},
				},

				"memory": mapstr.M{
					"available": mapstr.M{
						"bytes": container.Memory.AvailableBytes,
					},
					"usage": mapstr.M{
						"bytes": container.Memory.UsageBytes,
					},
					"workingset": mapstr.M{
						"bytes": container.Memory.WorkingSetBytes,
					},
					"rss": mapstr.M{
						"bytes": container.Memory.RssBytes,
					},
					"pagefaults":      container.Memory.PageFaults,
					"majorpagefaults": container.Memory.MajorPageFaults,
				},

				"rootfs": mapstr.M{
					"available": mapstr.M{
						"bytes": container.Rootfs.AvailableBytes,
					},
					"capacity": mapstr.M{
						"bytes": container.Rootfs.CapacityBytes,
					},
					"used": mapstr.M{
						"bytes": container.Rootfs.UsedBytes,
					},
					"inodes": mapstr.M{
						"used": container.Rootfs.InodesUsed,
					},
				},

				"logs": mapstr.M{
					"available": mapstr.M{
						"bytes": container.Logs.AvailableBytes,
					},
					"capacity": mapstr.M{
						"bytes": container.Logs.CapacityBytes,
					},
					"used": mapstr.M{
						"bytes": container.Logs.UsedBytes,
					},
					"inodes": mapstr.M{
						"used":  container.Logs.InodesUsed,
						"free":  container.Logs.InodesFree,
						"count": container.Logs.Inodes,
					},
				},
			}

			if container.StartTime != "" {
				kubernetes2.ShouldPut(containerEvent, "start_time", container.StartTime, logger)
			}

			if nodeCores > 0 {
				kubernetes2.ShouldPut(containerEvent, "cpu.usage.node.pct", float64(container.CPU.UsageNanoCores)/1e9/nodeCores, logger)
			}

			if nodeMem > 0 {
				kubernetes2.ShouldPut(containerEvent, "memory.usage.node.pct", float64(container.Memory.UsageBytes)/nodeMem, logger)
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
				kubernetes2.ShouldPut(containerEvent, "cpu.usage.limit.pct", float64(container.CPU.UsageNanoCores)/1e9/containerCoresLimit, logger)
			}

			if containerMemLimit > 0 {
				kubernetes2.ShouldPut(containerEvent, "memory.usage.limit.pct", float64(container.Memory.UsageBytes)/containerMemLimit, logger)
				kubernetes2.ShouldPut(containerEvent, "memory.workingset.limit.pct", float64(container.Memory.WorkingSetBytes)/containerMemLimit, logger)
			}

			events = append(events, containerEvent)
		}

	}

	return events, nil
}

// ecsfields maps container events fields to container ecs fields
func ecsfields(containerEvent mapstr.M, logger *logp.Logger) mapstr.M {
	ecsfields := mapstr.M{}

	name, err := containerEvent.GetValue("name")
	if err == nil {
		kubernetes2.ShouldPut(ecsfields, "name", name, logger)
	}

	cpuUsage, err := containerEvent.GetValue("cpu.usage.node.pct")
	if err == nil {
		kubernetes2.ShouldPut(ecsfields, "cpu.usage", cpuUsage, logger)
	}

	memUsage, err := containerEvent.GetValue("memory.usage.node.pct")
	if err == nil {
		kubernetes2.ShouldPut(ecsfields, "memory.usage", memUsage, logger)
	}

	return ecsfields
}
