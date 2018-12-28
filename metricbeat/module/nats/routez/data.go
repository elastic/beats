package routez

import (
	"encoding/json"
	"github.com/elastic/beats/libbeat/common"
	"time"
)

type Routez struct {
	ServerID  string        `json:"server_id"`
	Now       time.Time     `json:"now"`
	NumRoutes int           `json:"num_routes"`
	Routes    []interface{} `json:"routes"`
}

func eventMapping(content []byte) common.MapStr {
	var data Routez
	json.Unmarshal(content, &data)
	// TODO: add error handling
	event := common.MapStr{
		"metrics": data,
	}
	return event
}
