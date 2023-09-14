package summarizer

import (
	"github.com/elastic/beats/v7/heartbeat/eventext"
	"github.com/elastic/beats/v7/libbeat/beat"
)

type DropBrowserExtraEvents struct{}

func (d DropBrowserExtraEvents) EachEvent(event *beat.Event, _ error) EachEventActions {
	st := synthType(event)
	// Sending these events can break the kibana UI in various places
	// see: https://github.com/elastic/kibana/issues/166530
	if st == "cmd/status" {
		eventext.CancelEvent(event)
	}

	return 0
}

func (d DropBrowserExtraEvents) BeforeSummary(event *beat.Event) BeforeSummaryActions {
	// noop
	return 0
}

func (d DropBrowserExtraEvents) BeforeRetry() {
	// noop
}
