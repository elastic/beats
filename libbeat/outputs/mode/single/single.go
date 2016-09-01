package single

import (
	"errors"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/op"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/mode"
)

// Mode sends all Output on one single connection. If connection is
// not available, the output plugin blocks until the connection is either available
// again or the connection mode is closed by Close.
type Mode struct {
	conn        mode.ProtocolClient
	isConnected bool

	closed bool // mode closed flag to break publisher loop

	timeout time.Duration // connection timeout
	backoff *common.Backoff

	// maximum number of configured send attempts. If set to 0, publisher will
	// block until event has been successfully published.
	maxAttempts int
}

var (
	errNeedBackoff = errors.New("need to backoff")

	debugf = logp.MakeDebug("output")
)

// New creates a new single connection mode using exactly one
// ProtocolClient connection.
func New(
	client mode.ProtocolClient,
	maxAttempts int,
	waitRetry, timeout, maxWaitRetry time.Duration,
) (*Mode, error) {
	s := &Mode{
		conn: client,

		timeout:     timeout,
		backoff:     common.NewBackoff(nil, waitRetry, maxWaitRetry),
		maxAttempts: maxAttempts,
	}

	return s, nil
}

func (s *Mode) connect() error {
	if s.isConnected {
		return nil
	}

	err := s.conn.Connect(s.timeout)
	s.isConnected = err == nil
	return err
}

// Close closes the underlying connection.
func (s *Mode) Close() error {
	s.closed = true
	return s.closeClient()
}

func (s *Mode) closeClient() error {
	err := s.conn.Close()
	s.isConnected = false
	return err
}

// PublishEvents tries to publish the events with retries if connection becomes
// unavailable. On failure PublishEvents tries to reconnect.
func (s *Mode) PublishEvents(
	signaler op.Signaler,
	opts outputs.Options,
	data []outputs.Data,
) error {
	return s.publish(signaler, opts, func() (bool, bool) {
		for len(data) > 0 {
			var err error

			total := len(data)
			data, err = s.conn.PublishEvents(data)
			if err != nil {
				logp.Info("Error publishing events (retrying): %s", err)

				madeProgress := len(data) < total
				return false, madeProgress
			}
		}

		return true, false
	})
}

// PublishEvent forwards a single event. On failure PublishEvent tries to reconnect.
func (s *Mode) PublishEvent(
	signaler op.Signaler,
	opts outputs.Options,
	data outputs.Data,
) error {
	return s.publish(signaler, opts, func() (bool, bool) {
		if err := s.conn.PublishEvent(data); err != nil {
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
func (s *Mode) publish(
	signaler op.Signaler,
	opts outputs.Options,
	send func() (ok bool, resetFail bool),
) error {
	fails := 0
	var err error

	guaranteed := opts.Guaranteed || s.maxAttempts == 0
	for !s.closed && (guaranteed || fails < s.maxAttempts) {

		ok := false
		resetFail := false

		if err := s.connect(); err != nil {
			logp.Err("Connecting error publishing events (retrying): %s", err)
			goto sendFail
		}

		ok, resetFail = send()
		if !ok {
			s.closeClient()
			goto sendFail
		}

		debugf("send completed")
		s.backoff.Reset()
		op.SigCompleted(signaler)
		return nil

	sendFail:
		debugf("send fail")

		fails++
		if resetFail {
			debugf("reset fails")
			s.backoff.Reset()
			fails = 0
		}
		s.backoff.Wait()

		if !guaranteed && (s.maxAttempts > 0 && fails == s.maxAttempts) {
			// max number of attempts reached
			debugf("max number of attempts reached")
			break
		}
	}

	debugf("messages dropped")
	mode.Dropped(1)
	op.SigFailed(signaler, err)
	return nil
}
