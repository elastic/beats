package info

import (
	"github.com/elastic/beats/libbeat/common"
)

// Map data to MapStr
func eventMapping(info map[string]string) common.MapStr {

	event := common.MapStr{
		"version":    info["redis_version"],
		"mode":       info["redis_mode"],
		"os":         info["os"],
		"process_id": info["process_id"],
	}

	return event
}
