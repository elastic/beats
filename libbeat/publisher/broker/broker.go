package broker

import (
	"io"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/publisher"
)

// Factory for creating a broker used by a pipeline instance.
type Factory func(*common.Config) (Broker, error)

// Broker is responsible for accepting, forwarding and ACKing events.
// A broker will receive and buffer single events from its producers.
// Consumers will receive events in batches from the brokers buffers.
// Once a consumer has finished processing a batch, it must ACK the batch, for
// the broker to advance its buffers. Events in progress or ACKed are not readable
// from the broker.
// When the broker decides it is safe to progress (events have been ACKed by
// consumer or flush to some other intermediate storage), it will send an ACK signal
// with the number of ACKed events to the Producer (ACK happens in batches).
type Broker interface {
	io.Closer

	Producer(cfg ProducerConfig) Producer
	Consumer() Consumer
}

// ProducerConfig as used by the Pipeline to configure some custom callbacks
// between pipeline and broker.
type ProducerConfig struct {
	// if ACK is set, the callback will be called with number of events being ACKed
	// by the broker
	ACK func(count int)

	// OnDrop provided to the broker, to report events being silently dropped by
	// the broker. For example an async producer close and publish event,
	// with close happening early might result in the event being dropped. The callback
	// gives a brokers user a chance to keep track of total number of events
	// being buffered by the broker.
	OnDrop func(count int)
}

// Producer interface to be used by the pipelines client to forward events to be
// published to the broker.
// When a producer calls `Cancel`, it's up to the broker to send or remove
// events not yet ACKed.
// Note: A broker is still allowed to send the ACK signal after Cancel. The
//       pipeline client must filter out ACKs after cancel.
type Producer interface {
	Publish(event publisher.Event)
	TryPublish(event publisher.Event) bool
	Cancel() int
}

// Consumer interface to be used by the pipeline output workers.
// The `Get` method retrieves a batch of events up to size `sz`. If sz <= 0,
// the batch size is up to the broker.
type Consumer interface {
	Get(sz int) (Batch, error)
	Close() error
}

// Batch of events to be returned to Consumers. The `ACK` method will send the
// ACK signal to the broker.
type Batch interface {
	Events() []publisher.Event
	ACK()
}
