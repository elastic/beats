package mode

import (
	"errors"
	"math/rand"
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/logp"
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
	signaler outputs.Signaler,
	events []common.MapStr,
) error {
	return f.publish(signaler, func() (bool, bool) {
		// loop until all events have been send in case client supports partial sends
		for len(events) > 0 {
			var err error

			total := len(events)
			events, err = f.conns[f.active].PublishEvents(events)
			if err != nil {
				logp.Info("Error publishing events (retrying): %s", err)

				madeProgress := len(events) < total
				return false, madeProgress
			}
		}

		return true, false
	})
}

// PublishEvent forwards a single event. On failure PublishEvent tries to reconnect.
func (f *FailOverConnectionMode) PublishEvent(
	signaler outputs.Signaler,
	event common.MapStr,
) error {
	return f.publish(signaler, func() (bool, bool) {
		if err := f.conns[f.active].PublishEvent(event); err != nil {
			logp.Info("Error publishing events (retrying): %s", err)
			return false, false
		}
		return true, false
	})
}

// publish is used to publish events using the configured protocol client.
// It provides general error handling and back off support used on failed
// send attempts. To be used by PublishEvent and PublishEvents.
// The send callback will try to progress sending traffic and returns kind of
// progress made in ok or resetFail. If ok is set to true, send finished
// processing events. If ok is false but resetFail is set, send was partially
// successful. If send was partially successful, the fail counter is reset thus up
// to maxAttempts send attempts without any progress might be executed.
func (f *FailOverConnectionMode) publish(
	signaler outputs.Signaler,
	send func() (ok bool, resetFail bool),
) error {
	fails := 0
	var err error

	// TODO: we want back off support here? Fail over normally will try another
	// connection.

	for !f.closed && (f.maxAttempts == 0 || fails < f.maxAttempts) {
		ok := false
		resetFail := false

		if err = f.connect(f.active); err != nil {
			logp.Info("Connecting error publishing events (retrying): %s", err)
			goto sendFail
		}

		ok, resetFail = send()
		if !ok {
			goto sendFail
		}

		outputs.SignalCompleted(signaler)
		return nil

	sendFail:
		fails++
		if resetFail {
			fails = 0
		}
		if f.maxAttempts > 0 && fails == f.maxAttempts {
			// max number of attempts reached
			break
		}

		time.Sleep(f.waitRetry)
	}

	outputs.SignalFailed(signaler, err)
	return nil
}
