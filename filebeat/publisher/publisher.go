package publisher

import (
	"errors"

	"github.com/elastic/beats/filebeat/util"
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
	Published(events []*util.Data) bool
}

func New(
	async bool,
	in chan []*util.Data,
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
func getDataEvents(entries []*util.Data) (dataEvents []common.MapStr, meta []common.MapStr) {
	dataEvents = make([]common.MapStr, 0, len(entries))
	meta = make([]common.MapStr, 0, len(entries))
	for _, data := range entries {
		if data.HasEvent() {
			dataEvents = append(dataEvents, data.GetEvent())
			meta = append(meta, data.GetMetadata())
		}
	}
	return dataEvents, meta
}
