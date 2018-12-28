package subsz

import (
	"encoding/json"
	"github.com/elastic/beats/libbeat/common"
)

type Subsz struct {
	NumSubscriptions int `json:"num_subscriptions"`
	NumCache         int `json:"num_cache"`
	NumInserts       int `json:"num_inserts"`
	NumRemoves       int `json:"num_removes"`
	NumMatches       int `json:"num_matches"`
	CacheHitRate     int `json:"cache_hit_rate"`
	MaxFanout        int `json:"max_fanout"`
	AvgFanout        int `json:"avg_fanout"`
}

func eventMapping(content []byte) common.MapStr {
	var data Subsz
	json.Unmarshal(content, &data)
	// TODO: add error handling
	event := common.MapStr{
		"metrics": data,
	}
	return event
}
