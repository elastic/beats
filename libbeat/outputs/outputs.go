// Package outputs defines common types and interfaces to be implemented by
// output plugins.

package outputs

import (
	"github.com/elastic/beats/libbeat/monitoring"
	"github.com/elastic/beats/libbeat/publisher"
)

var (
	Metrics = monitoring.Default.NewRegistry("output")

	AckedEvents = monitoring.NewInt(Metrics, "events.acked", monitoring.Report)
	WriteBytes  = monitoring.NewInt(Metrics, "write.bytes", monitoring.Report)
	WriteErrors = monitoring.NewInt(Metrics, "write.errors", monitoring.Report)
)

// Client provides the minimal interface an output must implement to be usable
// with the publisher pipeline.
type Client interface {
	Close() error

	// Publish sends events to the clients sink. A client must synchronously or
	// asynchronously ACK the given batch, once all events have been processed.
	// Using Retry/Cancelled a client can return a batch of unprocessed events to
	// the publisher pipeline. The publisher pipeline (if configured by the output
	// factory) will take care of retrying/dropping events.
	Publish(publisher.Batch) error
}

// NetworkClient defines the required client capabilites for network based
// outputs, that must be reconnectable.
type NetworkClient interface {
	Client
	Connectable
}

// Connectable is optionally implemented by clients that might be able to close
// and reconnect dynamically.
type Connectable interface {
	// Connect establishes a connection to the clients sink.
	// The connection attempt shall report an error if no connection could been
	// established within the given time interval. A timeout value of 0 == wait
	// forever.
	Connect() error
}
