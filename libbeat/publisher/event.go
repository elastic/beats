package publisher

import (
	"github.com/elastic/beats/libbeat/beat"
)

// Batch is used to pass a batch of events to the outputs and asynchronously listening
// for signals from these outpts. After a batch is processed (completed or
// errors), one of the signal methods must be called.
type Batch interface {
	Events() []Event

	// signals
	ACK()
	Drop()
	Retry()
	RetryEvents(events []Event)
	Cancelled()
	CancelledEvents(events []Event)
}

// Event is used by the publisher pipeline and broker to pass additional
// meta-data to the consumers/outputs.
type Event struct {
	Content beat.Event
	Flags   EventFlags
}

// EventFlags provides additional flags/option types  for used with the outputs.
type EventFlags uint8

const (
	// GuaranteedSend requires an output to not drop the event on failure, but
	// retry until ACK.
	GuaranteedSend EventFlags = 0x01
)

// Guaranteed checks if the event must not be dropped by the output or the
// publisher pipeline.
func (e *Event) Guaranteed() bool {
	return (e.Flags & GuaranteedSend) == GuaranteedSend
}
