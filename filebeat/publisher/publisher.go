package publisher

import (
	"errors"

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
	Published(events []*common.MapStr) bool
}

func New(
	async bool,
	in chan []*common.MapStr,
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
func getDataEvents(events []*common.MapStr) (dataEvents []common.MapStr, meta []common.MapStr) {
	dataEvents = make([]common.MapStr, 0, len(events))
	meta = make([]common.MapStr, 0, len(events))
	for _, event := range events {
		if ok, _ := event.HasKey("meta"); ok {
			mIface, err := event.GetValue("meta"); if err != nil {
				meta = append(meta, mIface.(common.MapStr))
			}
			event.Delete("meta")
		}
		dataEvents = append(dataEvents, *event)

	}
	return dataEvents, meta
}
