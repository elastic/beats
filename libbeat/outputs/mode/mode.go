// Package mode defines and implents output strategies with failover or load
// balancing modes for use by output plugins.
package mode

import (
	"errors"
	"expvar"
	"time"

	"github.com/elastic/beats/libbeat/common/op"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
)

// Metrics that can retrieved through the expvar web interface.
var (
	messagesDropped = expvar.NewInt("libbeat.outputs.messages_dropped")
)

// ErrNoHostsConfigured indicates missing host or hosts configuration
var ErrNoHostsConfigured = errors.New("no hosts configuration found")

// ConnectionMode takes care of connecting to hosts
// and potentially doing load balancing and/or failover
type ConnectionMode interface {
	// Close will stop the modes it's publisher loop and close all it's
	// associated clients
	Close() error

	// PublishEvents will send all events (potentially asynchronous) to its
	// clients.
	PublishEvents(sig op.Signaler, opts outputs.Options, data []outputs.Data) error

	// PublishEvent will send an event to its clients.
	PublishEvent(sig op.Signaler, opts outputs.Options, data outputs.Data) error
}

type Connectable interface {
	// Connect establishes a connection to the clients sink.
	// The connection attempt shall report an error if no connection could been
	// established within the given time interval. A timeout value of 0 == wait
	// forever.
	Connect(timeout time.Duration) error

	// Close closes the established connection.
	Close() error
}

// ProtocolClient interface is a output plugin specific client implementation
// for encoding and publishing events. A ProtocolClient must be able to connection
// to it's sink and indicate connection failures in order to be reconnected byte
// the output plugin.
type ProtocolClient interface {
	Connectable

	// PublishEvents sends events to the clients sink. On failure or timeout err
	// must be set.
	// PublishEvents is free to publish only a subset of given events, even in
	// error case. On return nextEvents contains all events not yet published.
	PublishEvents(data []outputs.Data) (nextEvents []outputs.Data, err error)

	// PublishEvent sends one event to the clients sink. On failure and error is
	// returned.
	PublishEvent(data outputs.Data) error
}

// AsyncProtocolClient interface is a output plugin specific client implementation
// for asynchronous encoding and publishing events.
type AsyncProtocolClient interface {
	Connectable

	AsyncPublishEvents(cb func([]outputs.Data, error), data []outputs.Data) error

	AsyncPublishEvent(cb func(error), data outputs.Data) error
}

var (
	// ErrTempBulkFailure indicates PublishEvents fail temporary to retry.
	ErrTempBulkFailure = errors.New("temporary bulk send failure")
)

var (
	debug = logp.MakeDebug("output")
)

func Dropped(i int) {
	messagesDropped.Add(int64(i))
}
