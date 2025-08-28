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

package pod

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
		var usageNanoCores, usageMem, availMem, rss, workingSet, pageFaults, majorPageFaults uint64
		var podCoreLimit, podMemLimit float64

		podId := util.NewPodId(pod.PodRef.Namespace, pod.PodRef.Name)
		podStore := nodeStore.GetPodStore(podId)

		allContainersCPULimitsDefined := true
		allContainersMemoryLimitsDefined := true

		for _, container := range pod.Containers {
			usageNanoCores += container.CPU.UsageNanoCores
			usageMem += container.Memory.UsageBytes
			availMem += container.Memory.AvailableBytes
			rss += container.Memory.RssBytes
			workingSet += container.Memory.WorkingSetBytes
			pageFaults += container.Memory.PageFaults
			majorPageFaults += container.Memory.MajorPageFaults

			containerStore := podStore.GetContainerStore(container.Name)
			containerMetrics := containerStore.GetContainerMetrics()

			// podCoreLimit and podMemLimit are defined only if all of Pod containers have a limit defined, otherwise the limit will be set to 0
			if allContainersCPULimitsDefined && containerMetrics.CoresLimit == nil {
				allContainersCPULimitsDefined = false
				podCoreLimit = 0.0
			}
			if allContainersCPULimitsDefined {
				podCoreLimit += containerMetrics.CoresLimit.Value
			}

			if allContainersMemoryLimitsDefined && containerMetrics.MemoryLimit == nil {
				allContainersMemoryLimitsDefined = false
				podMemLimit = 0.0
			}
			if allContainersMemoryLimitsDefined {
				podMemLimit += containerMetrics.MemoryLimit.Value
			}
		}

		podEvent := mapstr.M{
			mb.ModuleDataKey: mapstr.M{
				"namespace": pod.PodRef.Namespace,
				"node": mapstr.M{
					"name": node.NodeName,
				},
			},
			"name": pod.PodRef.Name,
			"uid":  pod.PodRef.UID,

			"cpu": mapstr.M{
				"usage": mapstr.M{
					"nanocores": usageNanoCores,
				},
			},

			"memory": mapstr.M{
				"usage": mapstr.M{
					"bytes": usageMem,
				},
				"available": mapstr.M{
					"bytes": availMem,
				},
				"working_set": mapstr.M{
					"bytes": workingSet,
				},
				"rss": mapstr.M{
					"bytes": rss,
				},
				"page_faults":       pageFaults,
				"major_page_faults": majorPageFaults,
			},

			"network": mapstr.M{
				"rx": mapstr.M{
					"bytes":  pod.Network.RxBytes,
					"errors": pod.Network.RxErrors,
				},
				"tx": mapstr.M{
					"bytes":  pod.Network.TxBytes,
					"errors": pod.Network.TxErrors,
				},
			},
		}

		if pod.StartTime != "" {
			kubernetes2.ShouldPut(podEvent, "start_time", pod.StartTime, logger)
		}

		// `nodeCores` can be 0 if `state_node` and/or `node` metricsets are disabled
		if nodeCores > 0 {
			kubernetes2.ShouldPut(podEvent, "cpu.usage.node.pct", float64(usageNanoCores)/1e9/nodeCores, logger)
		}

		if podCoreLimit > 0 {
			kubernetes2.ShouldPut(podEvent, "cpu.usage.limit.pct", float64(usageNanoCores)/1e9/podCoreLimit, logger)
		}

		if usageMem > 0 {
			// `nodeMem` can be 0 if `state_node` and/or `node` metricsets are disabled
			if nodeMem > 0 {
				kubernetes2.ShouldPut(podEvent, "memory.usage.node.pct", float64(usageMem)/nodeMem, logger)
			}
			if podMemLimit > 0 {
				kubernetes2.ShouldPut(podEvent, "memory.usage.limit.pct", float64(usageMem)/podMemLimit, logger)
				kubernetes2.ShouldPut(podEvent, "memory.working_set.limit.pct", float64(workingSet)/podMemLimit, logger)
			}
		}

		if workingSet > 0 && usageMem == 0 {
			if nodeMem > 0 {
				kubernetes2.ShouldPut(podEvent, "memory.usage.node.pct", float64(workingSet)/nodeMem, logger)
			}
			if podMemLimit > 0 {
				kubernetes2.ShouldPut(podEvent, "memory.usage.limit.pct", float64(workingSet)/podMemLimit, logger)

				kubernetes2.ShouldPut(podEvent, "memory.working_set.limit.pct", float64(workingSet)/podMemLimit, logger)
			}
		}

		events = append(events, podEvent)
	}
	return events, nil
}

// ecsfields maps pod events fields to container ecs fields
func ecsfields(podEvent mapstr.M, logger *logp.Logger) mapstr.M {
	ecsfields := mapstr.M{}

	egressBytes, err := podEvent.GetValue("network.tx.bytes")
	if err == nil {
		kubernetes2.ShouldPut(ecsfields, "network.egress.bytes", egressBytes, logger)
	}

	ingressBytes, err := podEvent.GetValue("network.rx.bytes")
	if err == nil {
		kubernetes2.ShouldPut(ecsfields, "network.ingress.bytes", ingressBytes, logger)
	}

	return ecsfields
}
