package publisher

import (
	"errors"

	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"
	"github.com/elastic/beats/libbeat/publisher"
)

var (
	eventsSent = monitoring.NewInt(nil, "publish.events")
)

// LogPublisher provides functionality to start and stop a publisher worker.
type LogPublisher interface {
	Start()
	Stop()
}

// SuccessLogger is used to report successfully published events.
type SuccessLogger interface {

	// Published will be run after events have been acknowledged by the outputs.
	Published(events []*input.Data) bool
}

func New(
	async bool,
	in chan []*input.Data,
	out SuccessLogger,
	pub publisher.Publisher,
) LogPublisher {
	if async {
		logp.Warn("publish_async is experimental and will be removed in a future version!")
		return newAsyncLogPublisher(in, out, pub)
	}
	return newSyncLogPublisher(in, out, pub)
}

var (
	sigPublisherStop = errors.New("publisher was stopped")
)

// getDataEvents returns all events which contain data (not only state updates)
// together with their associated metadata
func getDataEvents(events []*input.Data) (dataEvents []common.MapStr, meta []common.MapStr) {
	dataEvents = make([]common.MapStr, 0, len(events))
	meta = make([]common.MapStr, 0, len(events))
	for _, event := range events {
		if event.HasData() {
			dataEvents = append(dataEvents, event.Event)
			meta = append(meta, event.GetMetadata())
		}
	}
	return dataEvents, meta
}
