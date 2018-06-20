package beater

import (
	"github.com/elastic/beats/filebeat/input/file"
	"github.com/elastic/beats/libbeat/logp"
)

// eventAcker handles publisher pipeline ACKs and forwards
// them to the registrar or directly to the stateless logger.
type eventACKer struct {
	stateful  statefulLogger
	stateless statelessLogger
	log       *logp.Logger
}

type statefulLogger interface {
	Published(states []file.State)
}

type statelessLogger interface {
	Published(c int) bool
}

func newEventACKer(stateless statelessLogger, stateful statefulLogger) *eventACKer {
	return &eventACKer{stateless: stateless, stateful: stateful, log: logp.NewLogger("acker")}
}

func (a *eventACKer) ackEvents(data []interface{}) {
	stateless := 0
	states := make([]file.State, 0, len(data))
	for _, datum := range data {
		if datum == nil {
			stateless++
			continue
		}

		st, ok := datum.(file.State)
		if !ok {
			stateless++
			continue
		}

		states = append(states, st)
	}

	if len(states) > 0 {
		a.log.Debugw("stateful ack", "count", len(states))
		a.stateful.Published(states)
	}

	if stateless > 0 {
		a.log.Debugw("stateless ack", "count", stateless)
		a.stateless.Published(stateless)
	}
}
