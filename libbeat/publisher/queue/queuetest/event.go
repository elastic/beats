package queuetest

import (
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/publisher"
)

func makeEvent(fields common.MapStr) publisher.Event {
	return publisher.Event{
		Content: beat.Event{
			Timestamp: time.Now(),
			Fields:    fields,
		},
	}
}
