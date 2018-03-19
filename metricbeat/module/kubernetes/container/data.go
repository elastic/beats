package container

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

				"name":       container.Name,
				"start_time": container.StartTime,

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

			if nodeCores > 0 {
				containerEvent.Put("cpu.usage.node.pct", float64(container.CPU.UsageNanoCores)/1e9/nodeCores)
			}

			if nodeMem > 0 {
				containerEvent.Put("memory.usage.node.pct", float64(container.Memory.UsageBytes)/nodeMem)
			}

			cuid := util.ContainerUID(pod.PodRef.Namespace, pod.PodRef.Name, container.Name)
			coresLimit := perfMetrics.ContainerCoresLimit.GetWithDefault(cuid, nodeCores)
			memLimit := perfMetrics.ContainerMemLimit.GetWithDefault(cuid, nodeMem)

			if coresLimit > 0 {
				containerEvent.Put("cpu.usage.limit.pct", float64(container.CPU.UsageNanoCores)/1e9/coresLimit)
			}

			if memLimit > 0 {
				containerEvent.Put("memory.usage.limit.pct", float64(container.Memory.UsageBytes)/memLimit)
			}

			events = append(events, containerEvent)
		}

	}

	return events, nil
}
