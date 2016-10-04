package container

import (
	"time"

	dc "github.com/fsouza/go-dockerclient"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/module/docker"
)

func eventsMapping(containersList []dc.APIContainers) []common.MapStr {
	myEvents := []common.MapStr{}
	for _, container := range containersList {
		myEvents = append(myEvents, eventMapping(&container))
	}
	return myEvents
}

func eventMapping(cont *dc.APIContainers) common.MapStr {

	event := common.MapStr{
		"created": common.Time(time.Unix(cont.Created, 0)),
		"id":      cont.ID,
		"name":    docker.ExtractContainerName(cont.Names),
		"command": cont.Command,
		"image":   cont.Image,
		"size": common.MapStr{
			"root_fs": cont.SizeRootFs,
			"rw":      cont.SizeRw,
		},
		"status": cont.Status,
	}

	labels := docker.BuildLabelArray(cont.Labels)
	if len(labels) > 0 {
		event["labels"] = labels
	}

	ports := convertContainerPorts(cont.Ports)
	if len(ports) > 0 {
		event["ports"] = ports
	}

	return event
}

func convertContainerPorts(ports []dc.APIPort) []map[string]interface{} {
	var outputPorts = []map[string]interface{}{}
	for _, port := range ports {
		outputPort := common.MapStr{
			"ip": port.IP,
			"port": common.MapStr{
				"private": port.PrivatePort,
				"public":  port.PublicPort,
			},
			"type": port.Type,
		}
		outputPorts = append(outputPorts, outputPort)
	}

	return outputPorts
}
