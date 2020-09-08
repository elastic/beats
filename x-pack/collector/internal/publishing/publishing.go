// Package publishing provides support for event publishing and ACK handling.
package publishing

//go:generate godocdown -plain=false -output Readme.md

import (
	"github.com/elastic/go-concert/unison"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/logp"
)

// OutputFactory configures an output.
type OutputFactory interface {
	ConfigureOutput(*logp.Logger, *common.Config) (Output, error)
}

type Plugin struct {
	// Name of the output type.
	Name string

	// Configure the output stability. If the stability is not 'Stable' a message
	// is logged when the output type is configured.
	Stability feature.Stability

	// Deprecated marks the plugin as deprecated. If set a deprecation message is logged if
	// an output is configured.
	Deprecated bool

	// Info contains a short description of the output type.
	Info string

	// Doc contains an optional longer description.
	Doc string

	Configure func(*logp.Logger, *common.Config) (Output, error)
}

// Output represents an configured, but inactive output. The output instance
// should not hold any, such that Open can be called multiple times in order to create
// independent publisher instances.
// The Publisher that is returned by Open must report the events status to the ACKer.
type Output interface {
	// Open creates a Publisher for sending events. Although Open can directly
	// establish a connection it is recommended to not to establish connections
	// lazily when attempting to publish events in order to not block startup or
	// hog on resources in case no events will get published.
	//
	// The cancellation context ctx is only active for the call to Open (in case
	// a connection setup would block).  The context MUST NOT be passed to the
	// output instance. Open can, but is not required, to return without a
	// Publisher if the context has been closed. But pending calls MUST unblock on cancel.
	Open(ctx unison.Canceler, log *logp.Logger, acks ACKCallback) (Publisher, error)
}

// Publisher publishes events.
type Publisher interface {
	// Close closes the publisher, potentially canceling pending publishing request.
	// After Close returns, pending status updates for still in progress should not be
	// send to the ACKer anymore after
	Close() error

	// Publish sends events to its final destination.
	//
	// The publish mode is a hint to the output if the event must be retried or
	// can be dropped. If the output can not publish or enquene the even when
	// DropIfFull is set, it must drop the event immediately.
	//
	// The eventID must be reported to the acker, in case Publish did return
	// without any error.  Failures can be reported asynchronously via the ACKer,
	// or by returning an error during Publish. But only one way of reporting
	// errors must be used.
	// Event status updates should be reported immediately. Output are not
	// required to keep any order.
	//
	// NOTE: In case a batch of events gets ACKed, it is recommended to report
	// the ACK status in reverse order.
	//
	// TODO: add cancelation context as parameter (not done yet due to limitations in libbeat)
	//
	// XXX: consider to combine eventID and event into a special output event,
	//      that carries some callback context instead of an ID.
	Publish(mode beat.PublishMode, eventID EventID, event beat.Event) error
}

// ACKCallback reports the events status. UpdateEventStatus must be called
// exactly once per event ID.
type ACKCallback interface {
	UpdateEventStatus(EventID, EventStatus)
}

// EventID used to track events to be published.
type EventID uint64

// EventStatus describes the state an event us currently in (send, published,
// pending, failed).
type EventStatus uint8

const (
	EventPublished EventStatus = iota
	EventPending
	EventInvalid // invalid events can not be published
	EventFailed  // event could not be send and is finally dropped by the output
)
