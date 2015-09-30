package mode

import (
	"errors"
	"math/rand"
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/outputs"
)

// FailOverConnectionMode connects to at most one host by random and swap to
// another host (by random) if currently active host becomes unavailable. If no
// connection is available, the mode blocks until a new connection can be established.
type FailOverConnectionMode struct {
	conns  []ProtocolClient
	active int // id of active connection

	closed bool // mode closed flag to break publisher loop

	timeout   time.Duration // connection timeout
	waitRetry time.Duration // wait time until trying a new connection

	// maximum number of configured send attempts. If set to 0, publisher will
	// block until event has been successfully published.
	maxAttempts int
}

var (
	// ErrNoConnectionConfigured indicates no configured connections for publishing.
	ErrNoConnectionConfigured = errors.New("No connection configured")

	errNoActiveConnection = errors.New("No active connection")
)

// NewFailOverConnectionMode creates a new failover connection mode leveraging
// only one connection at once. If connection becomes unavailable, the mode will
// try to connect to another configured connection.
func NewFailOverConnectionMode(
	clients []ProtocolClient,
	maxAttempts int,
	waitRetry, timeout time.Duration,
) (*FailOverConnectionMode, error) {
	f := &FailOverConnectionMode{
		conns:       clients,
		timeout:     timeout,
		waitRetry:   waitRetry,
		maxAttempts: maxAttempts,
	}

	// Try to connect preliminary, but ignore errors for now.
	// Main publisher loop is responsible to ensure an available connection.
	_ = f.connect(-1)
	return f, nil
}

// Close closes the active connection.
func (f *FailOverConnectionMode) Close() error {
	if !f.closed {
		f.closed = true
		for _, conn := range f.conns {
			if conn.IsConnected() {
				_ = conn.Close()
			}
		}
	}
	return nil
}

func (f *FailOverConnectionMode) connect(active int) error {
	if 0 <= active && active < len(f.conns) && f.conns[active].IsConnected() {
		return nil
	}

	var next int
	switch {
	case len(f.conns) == 0:
		return ErrNoConnectionConfigured
	case len(f.conns) == 1:
		next = 0
	case len(f.conns) == 2 && 0 <= active && active <= 1:
		next = 1 - active
	default:
		for {
			// Connect to random server to potentially spread the
			// load when large number of beats with same set of sinks
			// are started up at about the same time.
			next = rand.Int() % len(f.conns)
			if next != active {
				break
			}
		}
	}

	f.active = next
	if f.conns[next].IsConnected() {
		return nil
	}

	return f.conns[next].Connect(f.timeout)
}

// PublishEvents tries to publish the events with retries if connection becomes
// unavailable. On failure PublishEvents tries to connect to another configured
// connection by random.
func (f *FailOverConnectionMode) PublishEvents(
	trans outputs.Signaler,
	events []common.MapStr,
) error {
	published := 0
	fails := 0
	for !f.closed && (f.maxAttempts == 0 || fails < f.maxAttempts) {
		if err := f.connect(f.active); err != nil {
			fails++
			time.Sleep(f.waitRetry)
			continue
		}

		// loop until all events have been send in case client supports partial sends
		for published < len(events) {
			conn := f.conns[f.active]
			n, err := conn.PublishEvents(events[published:])
			if err != nil {
				break
			}
			published += n
		}

		if published == len(events) {
			outputs.SignalCompleted(trans)
			return nil
		}

		// TODO(sissel): Track how frequently we timeout and reconnect. If we're
		// timing out too frequently, there's really no point in timing out since
		// basically everything is slow or down. We'll want to ratchet up the
		// timeout value slowly until things improve, then ratchet it down once
		// things seem healthy.
		time.Sleep(f.waitRetry)
		fails++
	}

	outputs.SignalFailed(trans)
	return nil
}

// PublishEvent forwards a single event. On failure PublishEvent tries to reconnect.
func (f *FailOverConnectionMode) PublishEvent(
	signaler outputs.Signaler,
	event common.MapStr,
) error {
	fails := 0
	for !f.closed && (f.maxAttempts == 0 || fails < f.maxAttempts) {
		if err := f.connect(f.active); err != nil {
			fails++
			time.Sleep(f.waitRetry)
			continue
		}

		if err := f.conns[f.active].PublishEvent(event); err != nil {
			fails++
			continue
		}

		outputs.SignalCompleted(signaler)
		return nil
	}

	outputs.SignalFailed(signaler)
	return nil
}
