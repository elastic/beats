package image

import (
	"time"

	"github.com/docker/docker/api/types"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/module/docker"
)

func eventsMapping(imagesList []types.ImageSummary, dedot bool) []common.MapStr {
	events := []common.MapStr{}
	for _, image := range imagesList {
		events = append(events, eventMapping(&image, dedot))
	}
	return events
}

func eventMapping(image *types.ImageSummary, dedot bool) common.MapStr {
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
	if len(image.Labels) > 0 {
		labels := docker.DeDotLabels(image.Labels, dedot)
		event["labels"] = labels
	}
	return event
}
