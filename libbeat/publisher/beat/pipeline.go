package beat

import (
	"time"
)

type Pipeline interface {
	Connect() (Client, error)
	ConnectWith(ClientConfig) (Client, error)
	Close() error
}

// Client holds a connection to the beats publisher pipeline
type Client interface {
	Publish(Event)
	PublishAll([]Event)
	Close() error
}

// ClientConfig defines common configuration options one can pass to
// Pipeline.ConnectWith to control the clients behavior and provide ACK support.
type ClientConfig struct {
	PublishMode PublishMode

	// Processors passes additional processor to the client, to be executed before
	// the pipeline processors.
	Processor Processor

	// WaitClose sets the maximum duration to wait on ACK, if client still has events
	// active non-acknowledged events in the publisher pipeline.
	// WaitClose is only effective if one of ACKCount, ACKEvents and ACKLastEvents
	// is configured
	WaitClose time.Duration

	// ACK handler strategies.
	// Note: ack handlers are run in another go-routine owned by the publisher pipeline.
	//       They should not block for to long, to not block the internal buffers for
	//       too long (buffers can only be freed after ACK has been processed).
	// Note: It's not supported to configure multiple ack handler types. Use at
	//       most one.

	// ACKCount reports the number of published events recently acknowledged
	// by the pipeline.
	ACKCount func(int)

	// ACKEvents reports the events recently acknowledged by the pipeline.
	// Note: The slice passed must be copied if the events are to be processed
	//       after the handler returns.
	ACKEvents func([]Event)

	// ACKLastEvent reports the last ACKed event out of a batch of ACKed events only.
	ACKLastEvent func(Event)
}

// Processor defines the minimal required interface for processor, that can be
// registered with the publisher pipeline.
type Processor interface {
	String() string // print full processor description
	Run(in Event) (event Event, publish bool, err error)
}

// PublishMode enum sets some requirements on the client connection to the beats
// publisher pipeline
type PublishMode uint8

const (
	// DefaultGuarantees are up to the pipeline configuration, as configured by the
	// operator.
	DefaultGuarantees PublishMode = iota

	// GuaranteedSend ensures events are retried until acknowledged by the output.
	// Normally guaranteed sending should be used with some client ACK-handling
	// to update state keeping track of the sending status.
	GuaranteedSend

	// DropIfFull drops an event to be send if the pipeline is currently full.
	// This ensures a beats internals can continue processing if the pipeline has
	// filled up. Usefull if an event stream must be processed to keep internal
	// state up-to-date.
	DropIfFull
)
