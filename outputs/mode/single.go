package mode

import (
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/logp"
	"github.com/elastic/libbeat/outputs"
)

// SingleConnectionMode sends all Output on one single connection. If connection is
// not available, the output plugin blocks until the connection is either available
// again or the connection mode is closed by Close.
type SingleConnectionMode struct {
	conn ProtocolClient

	closed bool // mode closed flag to break publisher loop

	timeout      time.Duration // connection timeout
	waitRetry    time.Duration // wait time until reconnect
	maxWaitRetry time.Duration // Maximum send/retry timeout in backoff case.

	// maximum number of configured send attempts. If set to 0, publisher will
	// block until event has been successfully published.
	maxAttempts int
}

// NewSingleConnectionMode creates a new single connection mode using exactly one
// ProtocolClient connection.
func NewSingleConnectionMode(
	client ProtocolClient,
	maxAttempts int,
	waitRetry, timeout, maxWaitRetry time.Duration,
) (*SingleConnectionMode, error) {
	s := &SingleConnectionMode{
		conn: client,

		timeout:      timeout,
		waitRetry:    waitRetry,
		maxWaitRetry: maxWaitRetry,

		maxAttempts: maxAttempts,
	}

	_ = s.connect() // try to connect, but ignore errors for now
	return s, nil
}

func (s *SingleConnectionMode) connect() error {
	if s.conn.IsConnected() {
		return nil
	}
	return s.conn.Connect(s.timeout)
}

// Close closes the underlying connection.
func (s *SingleConnectionMode) Close() error {
	s.closed = true
	return s.conn.Close()
}

// PublishEvents tries to publish the events with retries if connection becomes
// unavailable. On failure PublishEvents tries to reconnect.
func (s *SingleConnectionMode) PublishEvents(
	signaler outputs.Signaler,
	events []common.MapStr,
) error {
	return s.publish(signaler, func() (bool, bool) {
		for len(events) > 0 {
			var err error

			total := len(events)
			events, err = s.conn.PublishEvents(events)
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
func (s *SingleConnectionMode) PublishEvent(
	signaler outputs.Signaler,
	event common.MapStr,
) error {
	return s.publish(signaler, func() (bool, bool) {
		if err := s.conn.PublishEvent(event); err != nil {
			logp.Info("Error publishing event (retrying): %s", err)
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
func (s *SingleConnectionMode) publish(
	signaler outputs.Signaler,
	send func() (ok bool, resetFail bool),
) error {
	fails := 0
	var backoffCount uint
	var err error

	for !s.closed && (s.maxAttempts == 0 || fails < s.maxAttempts) {
		ok := false
		resetFail := false

		if err := s.connect(); err != nil {
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
		if s.maxAttempts > 0 && fails == s.maxAttempts {
			// max number of attempts reached
			break
		}

		logp.Info("send fail")
		backoff := time.Duration(int64(s.waitRetry) * (1 << backoffCount))
		if backoff > s.maxWaitRetry {
			backoff = s.maxWaitRetry
		} else {
			backoffCount++
		}
		logp.Info("backoff retry: %v", backoff)
		time.Sleep(backoff)
	}

	outputs.SignalFailed(signaler, err)
	return nil
}
