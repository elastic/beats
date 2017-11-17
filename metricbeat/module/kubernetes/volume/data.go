package volume

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
		for _, volume := range pod.Volume {
			volumeEvent := common.MapStr{
				mb.ModuleDataKey: common.MapStr{
					"namespace": pod.PodRef.Namespace,
					"node": common.MapStr{
						"name": node.NodeName,
					},
					"pod": common.MapStr{
						"name": pod.PodRef.Name,
					},
				},

				"name": volume.Name,
				"fs": common.MapStr{
					"available": common.MapStr{
						"bytes": volume.AvailableBytes,
					},
					"capacity": common.MapStr{
						"bytes": volume.CapacityBytes,
					},
					"used": common.MapStr{
						"bytes": volume.UsedBytes,
					},
					"inodes": common.MapStr{
						"used":  volume.InodesUsed,
						"free":  volume.InodesFree,
						"count": volume.Inodes,
					},
				},
			}
			events = append(events, volumeEvent)
		}

	}
	return events, nil
}
