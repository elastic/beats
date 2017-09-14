package leader

import (
	"encoding/json"

	"github.com/elastic/beats/libbeat/common"
)

type Counts struct {
	Success int64 `json:"success"`
	Fail    int64 `json:"fail"`
}

type Latency struct {
	Average           float64 `json:"average"`
	Current           float64 `json:"current"`
	Maximum           float64 `json:"maximum"`
	Minimum           int64   `json:"minimum"`
	StandardDeviation float64 `json:"standardDeviation"`
}

type FollowersID struct {
	Latency Latency `json:"latency"`
	Counts  Counts  `json:"counts"`
}

type Leader struct {
	Followers map[string]FollowersID `json:"followers"`
	Leader    string                 `json:"leader"`
}

func eventMapping(content []byte) common.MapStr {
	var data Leader
	json.Unmarshal(content, &data)
	event := common.MapStr{
		"followers": data.Followers,
		"leader":    data.Leader,
	}
	return event
}
