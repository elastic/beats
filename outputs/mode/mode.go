// Package mode defines and implents output strategies with failover or load
// balancing modes for use by output plugins.
package mode

import (
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/outputs"
)

// ConnectionMode takes care of connecting to hosts
// and potentially doing load balancing and/or failover
type ConnectionMode interface {
	// Close will stop the modes it's publisher loop and close all it's
	// associated clients
	Close() error

	// PublishEvents will send all events (potentially asynchronous) to its
	// clients.
	PublishEvents(trans outputs.Signaler, events []common.MapStr) error

	// PublishEvent will send an event to its clients.
	PublishEvent(trans outputs.Signaler, event common.MapStr) error
}

// ProtocolClient interface is a output plugin specific client implementation
// for encoding and publishing events. A ProtocolClient must be able to connection
// to it's sink and indicate connection failures in order to be reconnected byte
// the output plugin.
type ProtocolClient interface {
	// Connect establishes a connection to the clients sink.
	// The connection attempt shall report an error if no connection could been
	// established within the given time interval. A timeout value of 0 == wait
	// forever.
	Connect(timeout time.Duration) error

	// Close closes the established connection.
	Close() error

	// IsConnected indicates the clients connection state. If connection has
	// been lost while publishing events, IsConnected must return false. As long as
	// IsConnected returns false, an output plugin might try to re-establish the
	// connection by calling Connect.
	IsConnected() bool

	// PublishEvents sends events to the clients sink. On failure or timeout err
	// must be set. If connection has been lost, IsConnected must return false
	// in future calls.
	// PublishEvents is free to publish only a subset of given events, even in
	// error case. On return n indicates the number of events guaranteed to be
	// published.
	PublishEvents(events []common.MapStr) (n int, err error)

	// PublishEvent sends one event to the clients sink. On failure and error is
	// returned.
	PublishEvent(event common.MapStr) error
}
