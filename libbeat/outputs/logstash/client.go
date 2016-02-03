package logstash

import (
	"errors"
	"expvar"
	"math"
	"net"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// Metrics that can retrieved through the expvar web interface.
var (
	ackedEvents            = expvar.NewInt("libbeatLogstashPublishedAndAckedEvents")
	eventsNotAcked         = expvar.NewInt("libbeatLogstashPublishedButNotAckedEvents")
	publishEventsCallCount = expvar.NewInt("libbeatLogstashPublishEventsCallCount")
)

// client implements the ProtocolClient interface to be used
// with different mode. The client implements slow start with low window sizes +
// window size backoff in case of long running transactions.
//
// it is suggested to use lumberjack in conjunction with roundRobinConnectionMode
// if logstash becomes unresponsive
type client struct {
	TransportClient
	*protocol

	windowSize      int
	maxOkWindowSize int // max window size sending was successful for
	maxWindowSize   int
	countTimeoutErr int
}

const (
	minWindowSize             int = 1
	defaultStartMaxWindowSize int = 10
	maxAllowedTimeoutErr      int = 3
)

// errors
var (
	ErrNotConnected = errors.New("lumberjack client is not connected")
)

func newLumberjackClient(
	conn TransportClient,
	compressLevel int,
	maxWindowSize int,
	timeout time.Duration,
) (*client, error) {
	p, err := newClientProcol(conn, timeout, compressLevel)
	if err != nil {
		return nil, err
	}

	return &client{
		TransportClient: conn,
		protocol:        p,
		windowSize:      defaultStartMaxWindowSize,
		maxWindowSize:   maxWindowSize,
	}, nil
}

func (l *client) Connect(timeout time.Duration) error {
	logp.Debug("logstash", "connect")
	return l.TransportClient.Connect(timeout)
}

func (l *client) Close() error {
	logp.Debug("logstash", "close connection")
	return l.TransportClient.Close()
}

func (l *client) PublishEvent(event common.MapStr) error {
	_, err := l.PublishEvents([]common.MapStr{event})
	return err
}

// PublishEvents sends all events to logstash. On error a slice with all events
// not published or confirmed to be processed by logstash will be returned.
func (l *client) PublishEvents(
	events []common.MapStr,
) ([]common.MapStr, error) {
	publishEventsCallCount.Add(1)
	totalNumberOfEvents := len(events)
	for len(events) > 0 {
		n, err := l.publishWindowed(events)

		logp.Debug("logstash", "%v events out of %v events sent to logstash. Continue sending ...", n, len(events))
		events = events[n:]
		if err != nil {
			eventsNotAcked.Add(int64(len(events)))
			ackedEvents.Add(int64(totalNumberOfEvents - len(events)))
			return events, err
		}
	}
	ackedEvents.Add(int64(totalNumberOfEvents))
	return nil, nil
}

// publishWindowed published events with current maximum window size to logstash
// returning the total number of events sent (due to window size, or acks until
// failure).
func (l *client) publishWindowed(events []common.MapStr) (int, error) {
	if len(events) == 0 {
		return 0, nil
	}

	batchSize := len(events)
	debug("Try to publish %v events to logstash with window size %v",
		batchSize, l.windowSize)

	// prepare message payload
	if batchSize > l.windowSize {
		events = events[:l.windowSize]
	}

	outEvents, err := l.sendEvents(events)
	count := uint32(len(outEvents))
	if err != nil {
		if err == errAllEventsEncoding {
			return len(events), nil
		}
		return l.onFail(0, err)
	}

	if seq, err := l.awaitACK(count); err != nil {
		return l.onFail(int(seq), err)
	}

	l.tryGrowWindowSize(batchSize)
	return len(events), nil
}

func (l *client) onFail(n int, err error) (int, error) {
	l.shrinkWindow()

	// if timeout error, back off and ignore error
	nerr, ok := err.(net.Error)
	if !ok || !nerr.Timeout() {
		// no timeout error, close connection and return error
		_ = l.Close()
		return n, err
	}

	// if we've seen 3 consecutive timeout errors, close connection
	l.countTimeoutErr++
	if l.countTimeoutErr == maxAllowedTimeoutErr {
		_ = l.Close()
		return n, err
	}

	// timeout error. events. Send
	// mode might try to publish again with reduce window size or ask another
	// client to send events
	return n, nil
}

// Increase window size by factor 1.5 until max window size
// (window size grows exponentially)
// TODO: use duration until ACK to estimate an ok max window size value
func (l *client) tryGrowWindowSize(batchSize int) {
	if l.windowSize <= batchSize {
		if l.maxOkWindowSize < l.windowSize {
			logp.Debug("logstash", "update max ok window size: %v < %v", l.maxOkWindowSize, l.windowSize)
			l.maxOkWindowSize = l.windowSize

			newWindowSize := int(math.Ceil(1.5 * float64(l.windowSize)))
			logp.Debug("logstash", "increase window size to: %v", newWindowSize)

			if l.windowSize <= batchSize && batchSize < newWindowSize {
				logp.Debug("logstash", "set to batchSize: %v", batchSize)
				newWindowSize = batchSize
			}
			if newWindowSize > l.maxWindowSize {
				logp.Debug("logstash", "set to max window size: %v", l.maxWindowSize)
				newWindowSize = l.maxWindowSize
			}
			l.windowSize = newWindowSize
		} else if l.windowSize < l.maxOkWindowSize {
			logp.Debug("logstash", "update current window size: %v", l.windowSize)

			l.windowSize = int(math.Ceil(1.5 * float64(l.windowSize)))
			if l.windowSize > l.maxOkWindowSize {
				logp.Debug("logstash", "set to max ok window size: %v", l.maxOkWindowSize)
				l.windowSize = l.maxOkWindowSize
			}
		}
	}
}

func (l *client) shrinkWindow() {
	l.windowSize = l.windowSize / 2
	if l.windowSize < minWindowSize {
		l.windowSize = minWindowSize
	}
}
