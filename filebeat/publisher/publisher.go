package publisher

import (
	"errors"
	"expvar"

	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/publisher"
)

var (
	eventsSent = expvar.NewInt("publish.events")
)

// LogPublisher provides functionality to start and stop a publisher worker.
type LogPublisher interface {
	Start()
	Stop()
}

// Output is used to report successfully published events.
type Output interface {
	// Published will be run after events have been acknowledged by the outputs.
	Send(events []*input.Event) bool
}

func New(
	async bool,
	input chan []*input.Event,
	output Output,
	pub publisher.Publisher,
) LogPublisher {
	if async {
		return newAsyncLogPublisher(input, output, pub)
	}
	return newSyncLogPublisher(input, output, pub)
}

var (
	sigPublisherStop = errors.New("publisher was stopped")
)

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
