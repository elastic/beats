package osdpoolstats

import (
	"encoding/json"
	"fmt"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

func eventsMapping(input string) []common.MapStr {

	data := make([]map[string]interface{}, 0)
	err := json.Unmarshal([]byte(input), &data)
	if err != nil {
		logp.Err("An error occurred while parsing data for getting ceph osdpoolstats: %v", err)
	}

	eventsOsdPoolStatsmap, errOsdPoolStatsMap := decodeOsdPoolStats(data)
	if errOsdPoolStatsMap != nil {
		logp.Err("An error occurred while decoding data for ceph osdpoolstats: %v", errOsdPoolStatsMap)
	}

	return eventsOsdPoolStatsmap
}

func decodeOsdPoolStats(osdmap []map[string]interface{}) ([]common.MapStr, error) {
	myEvents := []common.MapStr{}

	// ceph.pool.stats: records pre pool IO and recovery throughput
	for _, pool := range osdmap {
		pool_name, ok := pool["pool_name"].(string)
		if !ok {
			return nil, fmt.Errorf("WARNING - unable to decode osd pool stats name")
		}

		osdevent := common.MapStr{
			"name": pool_name,
		}

		// Note: the 'recovery' object looks broken (in hammer), so it's omitted
		objects := []string{
			"client_io_rate",
			"recovery_rate",
		}
		for _, object := range objects {
			objectdata, ok := pool[object].(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("WARNING - unable to decode osd pool stats")
			}
			for key, value := range objectdata {
				event := common.MapStr{
					key: value,
				}
				osdevent = common.MapStrUnion(osdevent, event)
			}
		}
		myEvents = append(myEvents, osdevent)
	}

	return myEvents, nil
}
