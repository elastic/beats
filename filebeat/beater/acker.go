package beater

import (
	"github.com/elastic/beats/filebeat/util"
	"github.com/elastic/beats/libbeat/publisher/beat"
)

// eventAcker handles publisher pipeline ACKs and forwards
// them to the registrar.
type eventACKer struct {
	out successLogger
}

type successLogger interface {
	Published(events []*util.Data) bool
}

func newEventACKer(out successLogger) *eventACKer {
	return &eventACKer{out: out}
}

func (a *eventACKer) ackEvents(events []beat.Event) {
	data := make([]*util.Data, 0, len(events))
	for _, event := range events {
		p := event.Private
		if p == nil {
			continue
		}

		datum, ok := p.(*util.Data)
		if !ok || !datum.HasState() {
			continue
		}

		data = append(data, datum)
	}

	if len(data) > 0 {
		a.out.Published(data)
	}
}
