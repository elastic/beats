package pod

import (
	"encoding/json"
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/kubernetes"
	"github.com/elastic/beats/metricbeat/module/kubernetes/util"
)

func eventMapping(content []byte, perfMetrics *util.PerfMetricsCache) ([]common.MapStr, error) {
	events := []common.MapStr{}

	var summary kubernetes.Summary
	err := json.Unmarshal(content, &summary)
	if err != nil {
		return nil, fmt.Errorf("Cannot unmarshal json response: %s", err)
	}

	node := summary.Node
	nodeCores := perfMetrics.NodeCoresAllocatable.Get(node.NodeName)
	nodeMem := perfMetrics.NodeMemAllocatable.Get(node.NodeName)
	for _, pod := range summary.Pods {
		var usageNanoCores, usageMem int64
		var coresLimit, memLimit float64

		for _, cont := range pod.Containers {
			cuid := util.ContainerUID(pod.PodRef.Namespace, pod.PodRef.Name, cont.Name)
			usageNanoCores += cont.CPU.UsageNanoCores
			usageMem += cont.Memory.UsageBytes
			coresLimit += perfMetrics.ContainerCoresLimit.GetWithDefault(cuid, nodeCores)
			memLimit += perfMetrics.ContainerMemLimit.GetWithDefault(cuid, nodeMem)
		}

		podEvent := common.MapStr{
			mb.ModuleDataKey: common.MapStr{
				"namespace": pod.PodRef.Namespace,
				"node": common.MapStr{
					"name": node.NodeName,
				},
			},
			"name":       pod.PodRef.Name,
			"start_time": pod.StartTime,

			"cpu": common.MapStr{
				"usage": common.MapStr{
					"nanocores": usageNanoCores,
				},
			},

			"memory": common.MapStr{
				"usage": common.MapStr{
					"bytes": usageMem,
				},
			},

			"network": common.MapStr{
				"rx": common.MapStr{
					"bytes":  pod.Network.RxBytes,
					"errors": pod.Network.RxErrors,
				},
				"tx": common.MapStr{
					"bytes":  pod.Network.TxBytes,
					"errors": pod.Network.TxErrors,
				},
			},
		}

		if coresLimit > nodeCores {
			coresLimit = nodeCores
		}

		if memLimit > nodeMem {
			memLimit = nodeMem
		}

		if nodeCores > 0 {
			podEvent.Put("cpu.usage.node.pct", float64(usageNanoCores)/1e9/nodeCores)
		}

		if nodeMem > 0 {
			podEvent.Put("memory.usage.node.pct", float64(usageMem)/nodeMem)
		}

		if coresLimit > 0 {
			podEvent.Put("cpu.usage.limit.pct", float64(usageNanoCores)/1e9/coresLimit)
		}

		if memLimit > 0 {
			podEvent.Put("memory.usage.limit.pct", float64(usageMem)/memLimit)
		}

		events = append(events, podEvent)
	}
	return events, nil
}
