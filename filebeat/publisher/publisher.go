package publisher

import (
	"errors"
	"expvar"

	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/publisher"
)

var (
	eventsSent       = expvar.NewInt("publish.events")
	sigPublisherStop = errors.New("publisher was stopped")
)

// Publisher provides functionality to start and stop a publisher worker.
type Publisher interface {
	Start()
	Stop()
}

// Logger is used to log successfully published events.
// This can be used for example to log events to the registry
type Logger interface {
	// Published will be run after events have been acknowledged by the outputs.
	Log(events []*input.Event) bool
}

// New creates new sync Publisher. If async is set to true, an async publisher is returned.
func New(
	async bool,
	in chan []*input.Event,
	logger Logger,
	pub publisher.Publisher,
) Publisher {
	if async {
		return newAsyncPublisher(in, logger, pub)
	}
	return newSyncPublisher(in, logger, pub)
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
