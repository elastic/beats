package diskio

import (
	"github.com/elastic/beats/libbeat/common"
)

func eventsMapping(blkioStatsList []BlkioStats) []common.MapStr {
	myEvents := []common.MapStr{}
	for _, blkioStats := range blkioStatsList {
		myEvents = append(myEvents, eventMapping(&blkioStats))
	}
	return myEvents
}
func eventMapping(myBlkioStats *BlkioStats) common.MapStr {
	event := common.MapStr{
		"@timestamp": myBlkioStats.Time,
		"container": common.MapStr{
			"id":     myBlkioStats.MyContainer.Id,
			"name":   myBlkioStats.MyContainer.Name,
			"labels": myBlkioStats.MyContainer.Labels,
		},
		"blkio": common.MapStr{
			"reads":  myBlkioStats.reads,
			"writes": myBlkioStats.writes,
			"Total":  myBlkioStats.totals,
		},
	}
	return event
}
