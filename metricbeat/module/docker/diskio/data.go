package diskio
import (
	"github.com/elastic/beats/libbeat/common"
	//"fmt"
)

func eventsMapping( blkioStatsList []BlkioStats) [] common.MapStr {
	myEvents := [] common.MapStr{}
	for _, blkioStats := range blkioStatsList {
		myEvents = append(myEvents, eventMapping(&blkioStats))
	}
	return myEvents
}
func eventMapping( myBlkioStats *BlkioStats) common.MapStr{
	event := common.MapStr{
		"@timestamp":      myBlkioStats.Time,
		"type":            "blkio",
		"container": common.MapStr{
			"id":  myBlkioStats.MyContainer.Id,
			"name": myBlkioStats.MyContainer.Name,
			"labels": myBlkioStats.MyContainer.Labels,
		},
		//"dockerSocket": mycpuStats.MyContainer.Socket,
		"blkio": common.MapStr{
			"readsPS":       myBlkioStats.reads,
			"writesPS":        myBlkioStats.writes,
			"TotalPS": myBlkioStats.totals,
		},
	}
	return event
}

