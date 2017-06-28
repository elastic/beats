package json

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/publisher/beat"
)

// Event describes the event structure for events
// (in-)directly send to logstash
type event struct {
	Timestamp time.Time     `struct:"@timestamp"`
	Meta      meta          `struct:"@metadata"`
	Fields    common.MapStr `struct:",inline"`
}

// Meta defines common event metadata to be stored in '@metadata'
type meta struct {
	Beat   string                 `struct:"beat"`
	Type   string                 `struct:"type"`
	Fields map[string]interface{} `struct:",inline"`
}

func makeEvent(index string, in *beat.Event) event {
	return event{
		Timestamp: in.Timestamp,
		Meta: meta{
			Beat:   index,
			Type:   "doc",
			Fields: in.Meta,
		},
		Fields: in.Fields,
	}
}
