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

func eventMapping(content []byte, metricsStorage *util.MetricsStorage, logger *logp.Logger) ([]mapstr.M, error) {
	events := []mapstr.M{}

	var summary kubernetes.Summary
	err := json.Unmarshal(content, &summary)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal json response: %w", err)
	}

	node := summary.Node
	nodeMetricOwner := util.GetMetricOwner(node.NodeName, util.NODE_METRIC_PREFIX)
	nodeCores, _ := metricsStorage.GetMetricWithDefault(nodeMetricOwner, util.NODE_CORES_ALLOCATABLE_METRIC, 0.0)
	nodeMem, _ := metricsStorage.GetMetricWithDefault(nodeMetricOwner, util.NODE_MEMORY_ALLOCATABLE_METRIC, 0.0)
	for _, pod := range summary.Pods {
		var usageNanoCores, usageMem, availMem, rss, workingSet, pageFaults, majorPageFaults uint64
		var containerCoreLimits, containerMemLimits float64

		for _, cont := range pod.Containers {
			cuid := util.ContainerUID(pod.PodRef.Namespace, pod.PodRef.Name, cont.Name)
			usageNanoCores += cont.CPU.UsageNanoCores
			usageMem += cont.Memory.UsageBytes
			availMem += cont.Memory.AvailableBytes
			rss += cont.Memory.RssBytes
			workingSet += cont.Memory.WorkingSetBytes
			pageFaults += cont.Memory.PageFaults
			majorPageFaults += cont.Memory.MajorPageFaults

			containerMetricOwner := util.GetMetricOwner(cuid, util.CONTAINER_METRIC_PREFIX)

			containerCoreLimit, _ := metricsStorage.GetMetricWithDefault(containerMetricOwner, util.CONTAINER_CORES_LIMIT_METRIC, nodeCores)
			containerCoreLimits += containerCoreLimit

			containerMemLimit, _ := metricsStorage.GetMetricWithDefault(containerMetricOwner, util.CONTAINER_MEMORY_LIMIT_METRIC, nodeMem)
			containerMemLimits += containerMemLimit
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

		if nodeCores > 0 && containerCoreLimits > nodeCores {
			containerCoreLimits = nodeCores
		}

		if nodeMem > 0 && containerMemLimits > nodeMem {
			containerMemLimits = nodeMem
		}

		if nodeCores > 0 {
			kubernetes2.ShouldPut(podEvent, "cpu.usage.node.pct", float64(usageNanoCores)/1e9/nodeCores, logger)
		}

		if containerCoreLimits > 0 {
			kubernetes2.ShouldPut(podEvent, "cpu.usage.limit.pct", float64(usageNanoCores)/1e9/containerCoreLimits, logger)
		}

		if usageMem > 0 {
			if nodeMem > 0 {
				kubernetes2.ShouldPut(podEvent, "memory.usage.node.pct", float64(usageMem)/nodeMem, logger)
			}
			if containerMemLimits > 0 {
				kubernetes2.ShouldPut(podEvent, "memory.usage.limit.pct", float64(usageMem)/containerMemLimits, logger)
				kubernetes2.ShouldPut(podEvent, "memory.working_set.limit.pct", float64(workingSet)/containerMemLimits, logger)
			}
		}

		if workingSet > 0 && usageMem == 0 {
			if nodeMem > 0 {
				kubernetes2.ShouldPut(podEvent, "memory.usage.node.pct", float64(workingSet)/nodeMem, logger)
			}
			if containerMemLimits > 0 {
				kubernetes2.ShouldPut(podEvent, "memory.usage.limit.pct", float64(workingSet)/containerMemLimits, logger)

				kubernetes2.ShouldPut(podEvent, "memory.working_set.limit.pct", float64(workingSet)/containerMemLimits, logger)
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
