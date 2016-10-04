package diskio

import "github.com/elastic/beats/libbeat/common"

func eventsMapping(blkioStatsList []BlkioStats) []common.MapStr {
	myEvents := []common.MapStr{}
	for _, blkioStats := range blkioStatsList {
		myEvents = append(myEvents, eventMapping(&blkioStats))
	}
	return myEvents
}

func eventMapping(stats *BlkioStats) common.MapStr {
	event := common.MapStr{
		"container": stats.Container.ToMapStr(),
		"reads":     stats.reads,
		"writes":    stats.writes,
		"total":     stats.totals,
	}

	return event
}
