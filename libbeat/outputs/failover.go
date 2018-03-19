package outputs

import (
	"errors"
	"fmt"
	"math/rand"

	"github.com/elastic/beats/libbeat/publisher"
	"github.com/elastic/beats/libbeat/testing"
)

type failoverClient struct {
	clients []NetworkClient
	active  int
}

var (
	// ErrNoConnectionConfigured indicates no configured connections for publishing.
	ErrNoConnectionConfigured = errors.New("No connection configured")

	errNoActiveConnection = errors.New("No active connection")
)

// NewFailoverClient combines a set of NetworkClients into one NetworkClient instances,
// with at most one active client. If the active client fails, another client
// will be used.
func NewFailoverClient(clients []NetworkClient) NetworkClient {
	if len(clients) == 1 {
		return clients[0]
	}

	return &failoverClient{
		clients: clients,
		active:  -1,
	}
}

func (f *failoverClient) Connect() error {
	var (
		next   int
		active = f.active
		l      = len(f.clients)
	)

	switch {
	case l == 0:
		return ErrNoConnectionConfigured
	case l == 1:
		next = 0
	case l == 2 && 0 <= active && active <= 1:
		next = 1 - active
	default:
		for {
			// Connect to random server to potentially spread the
			// load when large number of beats with same set of sinks
			// are started up at about the same time.
			next = rand.Int() % l
			if next != active {
				break
			}
		}
	}

	client := f.clients[next]
	f.active = next
	return client.Connect()
}

func (f *failoverClient) Close() error {
	if f.active < 0 {
		return errNoActiveConnection
	}
	return f.clients[f.active].Close()
}

func (f *failoverClient) Publish(batch publisher.Batch) error {
	if f.active < 0 {
		batch.Retry()
		return errNoActiveConnection
	}
	return f.clients[f.active].Publish(batch)
}

func (f *failoverClient) Test(d testing.Driver) {
	for i, client := range f.clients {
		c, ok := client.(testing.Testable)
		d.Run(fmt.Sprintf("Client %d", i), func(d testing.Driver) {
			if !ok {
				d.Fatal("output", errors.New("client doesn't support testing"))
			}
			c.Test(d)
		})
	}
}
