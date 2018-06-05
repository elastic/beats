package pod

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
	pods := []common.MapStr{}
	for _, pod := range summary.Pods {
		var usageNanoCores, usageMem int64
		for _, cont := range pod.Containers {
			usageNanoCores += cont.CPU.UsageNanoCores
			usageMem += cont.Memory.UsageBytes
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

		pods = append(pods, podEvent)
	}

	events := util.MergeEvents(pods, stateMetrics,
		map[string]string{
			mb.ModuleDataKey + ".node.name": node.NodeName,
		},
		[]string{mb.NamespaceKey},
		[]string{
			mb.ModuleDataKey + ".namespace",
			"name",
		},
	)

	return events, nil
}
