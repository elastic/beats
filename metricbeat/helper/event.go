package helper

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

func getIndex(event common.MapStr, indexName string) string {
	// Set index from event if set

	if _, ok := event["index"]; ok {
		indexName, ok = event["index"].(string)
		if !ok {
			logp.Err("Index couldn't be overwritten because event index is not string")
		}
		delete(event, "index")
	}
	return indexName
}

func getType(event common.MapStr, typeName string) string {

	// Set type from event if set
	if _, ok := event["type"]; ok {
		typeName, ok = event["type"].(string)
		if !ok {
			logp.Err("Type couldn't be overwritten because event type is not string")
		}
		delete(event, "type")
	}

	return typeName
}

func getTimestamp(event common.MapStr, timestamp common.Time) common.Time {

	// Set timestamp from event if set, move it to the top level
	// If not set, timestamp is created
	if _, ok := event["@timestamp"]; ok {
		timestamp, ok = event["@timestamp"].(common.Time)
		if !ok {
			logp.Err("Timestamp couldn't be overwritten because event @timestamp is not common.Time")
		}
		delete(event, "@timestamp")
	}
	return timestamp
}
