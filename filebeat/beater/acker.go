package beater

import (
	"github.com/elastic/beats/filebeat/input/file"
)

// eventAcker handles publisher pipeline ACKs and forwards
// them to the registrar.
type eventACKer struct {
	out successLogger
}

type successLogger interface {
	Published(states []file.State)
}

func newEventACKer(out successLogger) *eventACKer {
	return &eventACKer{out: out}
}

func (a *eventACKer) ackEvents(data []interface{}) {
	states := make([]file.State, 0, len(data))
	for _, datum := range data {
		if datum == nil {
			continue
		}

		st, ok := datum.(file.State)
		if !ok {
			continue
		}

		states = append(states, st)
	}

	if len(states) > 0 {
		a.out.Published(states)
	}
}
