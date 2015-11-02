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
	published := 0
	fails := 0
	var backoffCount uint
	var err error

	for !s.closed && (s.maxAttempts == 0 || fails < s.maxAttempts) {
		if err := s.connect(); err != nil {
			logp.Info("Connecting error publishing events (retrying): %s", err)
			goto sendFail
		}

		for published < len(events) {
			n, err := s.conn.PublishEvents(events[published:])
			if err != nil {
				logp.Info("Error publishing events (retrying): %s", err)
				break
			}

			fails = 0
			published += n
		}

		if published == len(events) {
			outputs.SignalCompleted(signaler)
			return nil
		}

	sendFail:
		logp.Info("send fail")
		backoff := time.Duration(int64(s.waitRetry) * (1 << backoffCount))
		if backoff > s.maxWaitRetry {
			backoff = s.maxWaitRetry
		} else {
			backoffCount++
		}
		logp.Info("backoff retry: %v", backoff)
		time.Sleep(backoff)
		fails++
	}

	outputs.SignalFailed(signaler, err)
	return nil
}

// PublishEvent forwards a single event. On failure PublishEvent tries to reconnect.
func (s *SingleConnectionMode) PublishEvent(
	signaler outputs.Signaler,
	event common.MapStr,
) error {
	fails := 0

	var err error

	for !s.closed && (s.maxAttempts == 0 || fails < s.maxAttempts) {
		if err = s.connect(); err != nil {
			logp.Info("Connecting error publishing event (retrying): %s", err)

			fails++
			time.Sleep(s.waitRetry)
			continue
		}
		if err := s.conn.PublishEvent(event); err != nil {
			logp.Info("Error publishing event (retrying): %s", err)

			fails++
			continue
		}

		outputs.SignalCompleted(signaler)
		return nil
	}

	outputs.SignalFailed(signaler, err)
	return nil
}
