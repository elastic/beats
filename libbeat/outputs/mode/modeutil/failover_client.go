package modeutil

import (
	"errors"
	"math/rand"
	"time"

	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/mode"
)

type failOverClient struct {
	conns  []mode.ProtocolClient
	active int
}

type asyncFailOverClient struct {
	conns  []mode.AsyncProtocolClient
	active int
}

type clientList interface {
	Active() int
	Len() int
	Get(i int) mode.Connectable
	Activate(i int)
}

var (
	// ErrNoConnectionConfigured indicates no configured connections for publishing.
	ErrNoConnectionConfigured = errors.New("No connection configured")

	errNoActiveConnection = errors.New("No active connection")
)

func NewFailoverClient(clients []mode.ProtocolClient) []mode.ProtocolClient {
	if len(clients) <= 1 {
		return clients
	}
	return []mode.ProtocolClient{&failOverClient{conns: clients, active: -1}}
}

func (f *failOverClient) Active() int                { return f.active }
func (f *failOverClient) Len() int                   { return len(f.conns) }
func (f *failOverClient) Get(i int) mode.Connectable { return f.conns[i] }
func (f *failOverClient) Activate(i int)             { f.active = i }

func (f *failOverClient) Connect(to time.Duration) error {
	return connect(f, to)
}

func (f *failOverClient) Close() error {
	return closeActive(f)
}

func (f *failOverClient) PublishEvents(data []outputs.Data) ([]outputs.Data, error) {
	if f.active < 0 {
		return data, errNoActiveConnection
	}
	return f.conns[f.active].PublishEvents(data)
}

func (f *failOverClient) PublishEvent(data outputs.Data) error {
	if f.active < 0 {
		return errNoActiveConnection
	}
	return f.conns[f.active].PublishEvent(data)
}

func NewAsyncFailoverClient(clients []mode.AsyncProtocolClient) []mode.AsyncProtocolClient {
	if len(clients) <= 1 {
		return clients
	}
	return []mode.AsyncProtocolClient{
		&asyncFailOverClient{conns: clients, active: -1},
	}
}

func (f *asyncFailOverClient) Active() int                { return f.active }
func (f *asyncFailOverClient) Len() int                   { return len(f.conns) }
func (f *asyncFailOverClient) Get(i int) mode.Connectable { return f.conns[i] }
func (f *asyncFailOverClient) Activate(i int)             { f.active = i }

func (f *asyncFailOverClient) Connect(to time.Duration) error {
	return connect(f, to)
}

func (f *asyncFailOverClient) Close() error {
	return closeActive(f)
}

func (f *asyncFailOverClient) AsyncPublishEvents(
	cb func([]outputs.Data, error),
	data []outputs.Data,
) error {
	if f.active < 0 {
		return errNoActiveConnection
	}
	return f.conns[f.active].AsyncPublishEvents(cb, data)
}

func (f *asyncFailOverClient) AsyncPublishEvent(
	cb func(error),
	data outputs.Data,
) error {
	if f.active < 0 {
		return errNoActiveConnection
	}
	return f.conns[f.active].AsyncPublishEvent(cb, data)
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
