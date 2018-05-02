package diskio

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
)

func eventsMapping(blkioStatsList []BlkioStats) []common.MapStr {
	myEvents := []common.MapStr{}
	for _, blkioStats := range blkioStatsList {
		myEvents = append(myEvents, eventMapping(&blkioStats))
	}
	return myEvents
}

func eventMapping(stats *BlkioStats) common.MapStr {
	event := common.MapStr{
		mb.ModuleDataKey: common.MapStr{
			"container": stats.Container.ToMapStr(),
		},
		"reads":  stats.reads,
		"writes": stats.writes,
		"total":  stats.totals,
		"read": common.MapStr{
			"ops":   stats.serviced.reads,
			"bytes": stats.servicedBytes.reads,
			"rate":  stats.reads,
		},
		"write": common.MapStr{
			"ops":   stats.serviced.writes,
			"bytes": stats.servicedBytes.writes,
			"rate":  stats.writes,
		},
		"summary": common.MapStr{
			"ops":   stats.serviced.totals,
			"bytes": stats.servicedBytes.totals,
			"rate":  stats.totals,
		},
	}

	return event
}
