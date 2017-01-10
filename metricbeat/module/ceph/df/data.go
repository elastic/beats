package df

import (
	"encoding/json"
	"fmt"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

const (
	measurement = "ceph"
	typeMon     = "monitor"
	typeOsd     = "osd"
	osdPrefix   = "ceph-osd"
	monPrefix   = "ceph-mon"
	sockSuffix  = "asok"
)

func eventsMapping(input string) []common.MapStr {

	data := make(map[string]interface{})
	err := json.Unmarshal([]byte(input), &data)
	if err != nil {
		logp.Err("An error occurred while parsing data for getting ceph df: %v", err)
	}

	statsFields, ok := data["stats"].(map[string]interface{})
	if !ok {
		logp.Err("An error occurred while parsing data for getting ceph df stats: %v", err)
	}

	statsevent := common.MapStr{}
	for tag, datapoints := range statsFields {
		event := common.MapStr{
			"stats." + tag: datapoints,
		}
		statsevent = common.MapStrUnion(statsevent, event)
	}

	eventsDfmap, errDfMap := decodeDf(data)
	if errDfMap != nil {
		logp.Err("An error occurred while parsing data for getting ceph df: %v", errDfMap)
	}

	return append(eventsDfmap, statsevent)
}

func decodeDf(dfmap map[string]interface{}) ([]common.MapStr, error) {
	newevent := common.MapStr{}
	myEvents := []common.MapStr{}

	for key, value := range dfmap {
		switch value.(type) {
		case []interface{}:
			if key == "pools" {
				for _, stats := range value.([]interface{}) {
					stats_map, ok := stats.(map[string]interface{})
					if !ok {
						return nil, fmt.Errorf("WARNING - unable to decode df stats")
					}

					for tag, datapoints := range stats_map {
						event := common.MapStr{
							"pools." + tag: datapoints,
						}
						newevent = common.MapStrUnion(newevent, event)
					}
					myEvents = append(myEvents, newevent)

				}
			}
		}
	}
	return myEvents, nil
}
