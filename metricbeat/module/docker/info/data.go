package info

import (
	"github.com/elastic/beats/libbeat/common"

	dc "github.com/fsouza/go-dockerclient"
)

func eventMapping(info *dc.DockerInfo) common.MapStr {

	event := common.MapStr{
		"id": info.ID,
		"containers": common.MapStr{
			"total":   info.Containers,
			"running": info.ContainersRunning,
			"paused":  info.ContainersPaused,
			"stopped": info.ContainersStopped,
		},
		"images": info.Images,
	}

	return event
}
