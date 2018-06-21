package container

import (
	"encoding/json"
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/kubernetes"
	"github.com/elastic/beats/metricbeat/module/kubernetes/util"
)

func eventMapping(content []byte, stateMetrics []common.MapStr) ([]common.MapStr, error) {
	var summary kubernetes.Summary

	err := json.Unmarshal(content, &summary)
	if err != nil {
		return nil, fmt.Errorf("Cannot unmarshal json response: %s", err)
	}

	node := summary.Node
	containers := []common.MapStr{}
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

			containers = append(containers, containerEvent)
		}
	}

	events := util.MergeEvents(containers, stateMetrics,
		map[string]string{
			mb.ModuleDataKey + ".node.name": node.NodeName,
		},
		[]string{mb.NamespaceKey},
		[]string{
			mb.ModuleDataKey + ".namespace",
			mb.ModuleDataKey + ".pod.name",
			"name",
		},
	)

	// Calculate pct fields
	for _, event := range events {
		memLimit := util.GetFloat64(event, "memory.limit.bytes")
		if memLimit > 0 {
			event.Put("memory.usage.limit.pct", float64(util.GetInt64(event, "memory.usage.bytes"))/memLimit)
		}

		cpuLimit := util.GetFloat64(event, "cpu.limit.cores")
		if cpuLimit > 0 {
			event.Put("cpu.usage.limit.pct", float64(util.GetInt64(event, "cpu.usage.nanocores"))/cpuLimit/1e9)
		}
	}

	return events, nil
}
