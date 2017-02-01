package image

import (
	"github.com/elastic/beats/libbeat/common"

	"github.com/elastic/beats/metricbeat/module/docker"
	dc "github.com/fsouza/go-dockerclient"
	"time"
)

func eventsMapping(imagesList []dc.APIImages) []common.MapStr {
	events := []common.MapStr{}
	for _, image := range imagesList {
		events = append(events, eventMapping(&image))
	}
	return events
}

func eventMapping(image *dc.APIImages) common.MapStr {
	event := common.MapStr{
		"id": common.MapStr{
			"current": image.ID,
			"parent":  image.ParentID,
		},
		"created": common.Time(time.Unix(image.Created, 0)),
		"size": common.MapStr{
			"regular": image.Size,
			"virtual": image.VirtualSize,
		},
		"tags": image.RepoTags,
	}
	labels := docker.DeDotLabels(image.Labels)
	if len(labels) > 0 {
		event["labels"] = labels
	}
	return event
}
