package node

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/module/docker"

	"github.com/fsouza/go-dockerclient/external/github.com/docker/api/types/swarm"
)

func eventsMapping(nodesList []swarm.Node, m *MetricSet) []common.MapStr {
	myEvents := []common.MapStr{}

	for _, node := range nodesList {
		myEvents = append(myEvents, eventMapping(&node, m))
	}

	return myEvents
}

func eventMapping(node *swarm.Node, m *MetricSet) common.MapStr {
	event := common.MapStr{
		"createdat": node.Meta.CreatedAt,
		"updatedat": node.Meta.UpdatedAt,
		"id":      node.ID,
		"hostname":    node.Description.Hostname,
		"spec": common.MapStr{
			"role": node.Spec.Role,
			"avaiability": node.Spec.Availability,
		},
		"platform": common.MapStr{
			"architecture": node.Description.Platform.Architecture,
			"os": node.Description.Platform.OS,
		},
		"status": common.MapStr{
			"state": node.Status.State,
			"addr": node.Status.Addr,
		},
		"ressources": common.MapStr{
			"nanocpus": node.Description.Resources.NanoCPUs,
			"memorybytes": node.Description.Resources.MemoryBytes,
		},
		"engine.version": node.Description.Engine.EngineVersion,
	}

	if node.Spec.Role == "manager" {
		//fmt.Println("this is a manager ",node.ManagerStatus.Leader)
		manager := common.MapStr{
			"leader": node.ManagerStatus.Leader,
			"reachability": node.ManagerStatus.Reachability,
			"addr": node.ManagerStatus.Addr,
		}
		event["manager"] = manager
	}

	swarm_labels := docker.DeDotLabels(node.Spec.Annotations.Labels)
	if len(swarm_labels) > 0 {
		event["labels"] = swarm_labels
	}
	engine_labels := docker.DeDotLabels(node.Description.Engine.Labels)
	if len(engine_labels) > 0 {
		event["engine.labels"] = engine_labels
	}

	return event
}
