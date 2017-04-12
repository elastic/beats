package node

import (
	"encoding/json"
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/kubelet"
)

func eventMapping(content []byte) ([]common.MapStr, error) {
	events := []common.MapStr{}

	var summary kubelet.Summary
	err := json.Unmarshal(content, &summary)
	if err != nil {
		return nil, fmt.Errorf("Cannot unmarshal json response: %s", err)
	}

	node := summary.Node
	for _, pod := range summary.Pods {
		for _, volume := range pod.Volume {
			volumeEvent := common.MapStr{
				mb.ModuleData: common.MapStr{
					"pod": common.MapStr{
						"name":      pod.PodRef.Name,
						"namespace": pod.PodRef.Namespace,
						"node":      node.NodeName,
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
