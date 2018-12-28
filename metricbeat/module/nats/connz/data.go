package connz

import (
	"encoding/json"
	"github.com/elastic/beats/libbeat/common"
	"time"
)

type Connz struct {
	ServerID       string        `json:"server_id"`
	Now            time.Time     `json:"now"`
	NumConnections int           `json:"num_connections"`
	Total          int           `json:"total"`
	Offset         int           `json:"offset"`
	Limit          int           `json:"limit"`
	Connections    []interface{} `json:"connections"`
}

func eventMapping(content []byte) common.MapStr {
	var data Connz
	json.Unmarshal(content, &data)
	// TODO: add error handling
	event := common.MapStr{
		"metrics": data,
	}
	return event
}
