package mode

import (
	"errors"
	"math/rand"
	"time"

	"github.com/elastic/beats/libbeat/common"
)

type failOverClient struct {
	conns  []ProtocolClient
	active int
}

type asyncFailOverClient struct {
	conns  []AsyncProtocolClient
	active int
}

type clientList interface {
	Active() int
	Len() int
	Get(i int) Connectable
	Activate(i int)
}

var (
	// ErrNoConnectionConfigured indicates no configured connections for publishing.
	ErrNoConnectionConfigured = errors.New("No connection configured")

	errNoActiveConnection = errors.New("No active connection")
)

func NewFailoverClient(clients []ProtocolClient) []ProtocolClient {
	if len(clients) <= 1 {
		return clients
	}
	return []ProtocolClient{&failOverClient{conns: clients, active: -1}}
}

func (f *failOverClient) Active() int           { return f.active }
func (f *failOverClient) Len() int              { return len(f.conns) }
func (f *failOverClient) Get(i int) Connectable { return f.conns[i] }
func (f *failOverClient) Activate(i int)        { f.active = i }

func (f *failOverClient) Connect(to time.Duration) error {
	return connect(f, to)
}

func (f *failOverClient) IsConnected() bool {
	return f.active >= 0 && f.conns[f.active].IsConnected()
}

func (f *failOverClient) Close() error {
	return closeActive(f)
}

func (f *failOverClient) PublishEvents(events []common.MapStr) ([]common.MapStr, error) {
	if f.active < 0 {
		return events, errNoActiveConnection
	}
	return f.conns[f.active].PublishEvents(events)
}

func (f *failOverClient) PublishEvent(event common.MapStr) error {
	if f.active < 0 {
		return errNoActiveConnection
	}
	return f.conns[f.active].PublishEvent(event)
}

func NewAsyncFailoverClient(clients []AsyncProtocolClient) []AsyncProtocolClient {
	if len(clients) <= 1 {
		return clients
	}
	return []AsyncProtocolClient{
		&asyncFailOverClient{conns: clients, active: -1},
	}
}

func (f *asyncFailOverClient) Active() int           { return f.active }
func (f *asyncFailOverClient) Len() int              { return len(f.conns) }
func (f *asyncFailOverClient) Get(i int) Connectable { return f.conns[i] }
func (f *asyncFailOverClient) Activate(i int)        { f.active = i }

func (f *asyncFailOverClient) Connect(to time.Duration) error {
	return connect(f, to)
}

func (f *asyncFailOverClient) IsConnected() bool {
	return f.active >= 0 && f.conns[f.active].IsConnected()
}

func (f *asyncFailOverClient) Close() error {
	return closeActive(f)
}

func (f *asyncFailOverClient) AsyncPublishEvents(
	cb func([]common.MapStr, error),
	events []common.MapStr,
) error {
	if f.active < 0 {
		return errNoActiveConnection
	}
	return f.conns[f.active].AsyncPublishEvents(cb, events)
}

func (f *asyncFailOverClient) AsyncPublishEvent(
	cb func(error),
	event common.MapStr,
) error {
	if f.active < 0 {
		return errNoActiveConnection
	}
	return f.conns[f.active].AsyncPublishEvent(cb, event)
}

func connect(lst clientList, to time.Duration) error {
	active := lst.Active()
	l := lst.Len()
	next := 0

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

	conn := lst.Get(next)
	lst.Activate(next)
	if conn.IsConnected() {
		return nil
	}

	return conn.Connect(to)
}

func closeActive(lst clientList) error {
	active := lst.Active()
	if active < 0 {
		return nil
	}

	conn := lst.Get(active)
	err := conn.Close()
	lst.Activate(-1)
	return err
}
