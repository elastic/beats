package pod

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
		podEvent := common.MapStr{
			mb.ModuleDataKey: common.MapStr{
				"namespace": pod.PodRef.Namespace,
				"node": common.MapStr{
					"name": node.NodeName,
				},
			},
			"name":       pod.PodRef.Name,
			"start_time": pod.StartTime,

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
		events = append(events, podEvent)
	}
	return events, nil

}
