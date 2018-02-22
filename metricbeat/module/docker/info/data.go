package info

import (
	"github.com/docker/docker/api/types"

	"github.com/elastic/beats/libbeat/common"
)

func eventMapping(info *types.Info) common.MapStr {
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
