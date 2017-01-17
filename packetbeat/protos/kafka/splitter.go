package kafka

import (
	"errors"
	"time"

	"github.com/elastic/beats/libbeat/common/streambuf"
)

type splitter struct {
	buf     streambuf.Buffer
	config  *splitterConfig
	message *rawMessage

	onMessage func(message *rawMessage) error
}

type splitterConfig struct {
	maxBytes int
}

// Error code if stream exceeds max allowed size on append.
var (
	ErrStreamTooLarge = errors.New("Stream data too large")
)

func (s *splitter) init(
	cfg *splitterConfig, onMessage func(*rawMessage) error,
) {
	*s = splitter{
		buf:       streambuf.Buffer{},
		config:    cfg,
		onMessage: onMessage,
	}
}

func (s *splitter) append(data []byte) error {
	_, err := s.buf.Write(data)
	if err != nil {
		return err
	}

	if s.config.maxBytes > 0 && s.buf.Total() > s.config.maxBytes {
		return ErrStreamTooLarge
	}
	return nil
}

func (s *splitter) feed(ts time.Time, data []byte) error {
	if err := s.append(data); err != nil {
		return err
	}

	for s.buf.Total() > 0 {
		if s.message == nil {
			// allocate new message object to be used by parser with current timestamp
			s.message = s.newMessage(ts)
		}

		msg, err := s.next()
		if err != nil {
			return err
		}
		if msg == nil {
			break // wait for more data
		}

		// reset buffer and message -> handle next message in buffer
		s.buf.Reset()
		s.message = nil

		// call message handler callback
		if err := s.onMessage(msg); err != nil {
			return err
		}
	}

	return nil
}

func (s *splitter) newMessage(ts time.Time) *rawMessage {
	return &rawMessage{
		TS: ts,
	}
}

func (s *splitter) next() (*rawMessage, error) {
	count, err := s.buf.ReadNetUint32At(0)
	if err != nil {
		if err == streambuf.ErrNoMoreBytes {
			err = nil
		}
		return nil, err
	}

	// TODO: check `count` exceeds max stream limit

	if !s.buf.Avail(int(count) + 4) {
		return nil, nil
	}

	debugf("new kafka message of size: %v", count)

	s.buf.Advance(4)
	msg := s.message
	msg.payload, err = s.buf.Collect(int(count))
	return msg, err
}
