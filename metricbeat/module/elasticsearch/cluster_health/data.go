package cluster_health

import (
	"encoding/json"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
)

func eventsMapping(content []byte) ([]common.MapStr, error) {
	var events []common.MapStr

	clusterStruct := struct {
		ClusterName   string `json:"cluster_name"`
		Status        string `json:"status"`
		NumberOfNodes int    `json:"number_of_nodes"`
		ActiveShards  int    `json:"active_shards"`
	}{}

	err := json.Unmarshal(content, &clusterStruct)

	if err != nil {
		return events, err
	}

	event := common.MapStr{
		"name":   clusterStruct.ClusterName,
		"status": clusterStruct.Status,
		"nodes": common.MapStr{
			"active": common.MapStr{
				"count": clusterStruct.NumberOfNodes,
			},
		},
		"shards": common.MapStr{
			"active": common.MapStr{
				"count": clusterStruct.ActiveShards,
			},
		},
	}
	event[mb.NamespaceKey] = "cluster"
	events = append(events, event)
	return events, nil
}
