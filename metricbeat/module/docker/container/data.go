package container

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/module/docker"

	dc "github.com/fsouza/go-dockerclient"
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

	labels := docker.DeDotLabels(cont.Labels)

	if len(labels) > 0 {
		event["labels"] = labels
	}

	return event
}
