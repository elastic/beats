package container

import (
	"encoding/json"
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/kubernetes"
)

func eventMapping(content []byte) ([]common.MapStr, error) {
	events := []common.MapStr{}
	var summary kubernetes.Summary

	err := json.Unmarshal(content, &summary)
	if err != nil {
		return nil, fmt.Errorf("Cannot unmarshal json response: %s", err)
	}

	node := summary.Node
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
			events = append(events, containerEvent)
		}

	}

	return events, nil

}
