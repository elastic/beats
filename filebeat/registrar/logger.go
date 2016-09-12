package registrar

import "github.com/elastic/beats/filebeat/input"

// The registrar logger can be used to Log events with the registry for which the
// state has to be persisted.
type Logger struct {
	done chan struct{}
	ch   chan<- []*input.Event
}

// newLogger creates a new logger for the given registrar
func newLogger(reg *Registrar) *Logger {
	return &Logger{
		done: make(chan struct{}),
		ch:   reg.Channel,
	}
}

func (l *Logger) Close() { close(l.done) }

// Log logs the events to the registrar
func (l *Logger) Log(events []*input.Event) bool {
	select {
	case <-l.done:
		// set ch to nil, so no more events will be send after channel close signal
		// has been processed the first time.
		// Note: nil channels will block, so only done channel will be actively
		//       report 'closed'.
		l.ch = nil
		return false
	case l.ch <- events:
		return true
	}
}
