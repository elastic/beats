package brokertest

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/publisher"
	"github.com/elastic/beats/libbeat/publisher/beat"
)

func makeEvent(fields common.MapStr) publisher.Event {
	return publisher.Event{
		Content: beat.Event{
			Timestamp: time.Now(),
			Fields:    fields,
		},
	}
}
