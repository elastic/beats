package publisher

import (
	"expvar"

	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/publisher"
)

var (
	eventsSent = expvar.NewInt("publish.events")
)

type LogPublisher interface {
	Start()
	Stop()
	Publish() error
}

func New(
	async bool,
	in, out chan []*input.Event,
	pub publisher.Publisher,
) LogPublisher {
	if async {
		return newAsyncLogPublisher(in, out, pub)
	}
	return newSyncLogPublisher(in, out, pub)
}

// getDataEvents returns all events which contain data (not only state updates)
func getDataEvents(events []*input.Event) []common.MapStr {
	dataEvents := make([]common.MapStr, 0, len(events))
	for _, event := range events {
		if event.HasData() {
			dataEvents = append(dataEvents, event.ToMapStr())
		}
	}
	return dataEvents
}
