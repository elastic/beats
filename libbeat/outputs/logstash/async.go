package logstash

import (
	"math"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

type asyncClient struct {
	TransportClient
	*protocol

	windowSize      int
	maxOkWindowSize int // max window size sending was successful for
	maxWindowSize   int
	countTimeoutErr int

	ch chan ackMessage
}

type ackMessage struct {
	connID uint

	tag    tag
	cb     func([]common.MapStr, error)
	events []common.MapStr
	err    error
	count  uint32
}

type tag uint8

const (
	tagComplete tag = iota
	tagSubset
	tagLast
	tagError
)

func newAsyncLumberjackClient(
	conn TransportClient,
	compressLevel int,
	maxWindowSize int,
	timeout time.Duration,
) (*asyncClient, error) {
	p, err := newClientProcol(conn, timeout, compressLevel)
	if err != nil {
		return nil, err
	}

	return &asyncClient{
		TransportClient: conn,
		protocol:        p,
		windowSize:      defaultStartMaxWindowSize,
		maxWindowSize:   maxWindowSize,
	}, nil
}

func (c *asyncClient) Connect(timeout time.Duration) error {
	logp.Debug("logstash", "connect (async)")
	return c.TransportClient.Connect(timeout)
}

func (c *asyncClient) Close() error {
	logp.Debug("logstash", "close (async) connection")
	return c.TransportClient.Close()
}

func (c *asyncClient) AsyncPublishEvent(
	cb func(error),
	event common.MapStr,
) error {
	return c.AsyncPublishEvents(func(_ []common.MapStr, err error) {
		cb(err)
	}, []common.MapStr{event})
}

func (c *asyncClient) AsyncPublishEvents(
	cb func([]common.MapStr, error),
	events []common.MapStr,
) error {
	if !c.IsConnected() {
		return ErrNotConnected
	}

	i := 1
	for len(events) > 0 {
		n, err, sigErr := c.publishWindowed(i, cb, events)
		debug("%v events out of %v events sent to logstash. Continue sending ...", n, len(events))

		if err != nil {
			if err != errAllEventsEncoding {
				// if sigErr != nil we short circuite ACK state machine error handling and
				// have async mode re-try inserting the failed attempts
				return sigErr
			}
		} else {
			i++
		}
	}

	return nil
}

func (c *asyncClient) publishWindowed(
	iteration int,
	cb func([]common.MapStr, error),
	allEvents []common.MapStr,
) (int, error, error) {
	if len(allEvents) == 0 {
		return 0, nil, nil
	}

	events := allEvents
	batchSize := len(events)
	partial := false
	debug("Try to publish %v events to logstash with window size %v",
		len(events), c.windowSize)

	// prepare message payload
	if batchSize > c.windowSize {
		partial = true
		events = events[:c.windowSize]
	}

	count, err := c.sendEvents(events)
	if err != nil {
		if err != errAllEventsEncoding {
			if iteration == 0 {
				// nothing send to ACK state machine yet -> short circuite error handling
				return 0, err, err
			}

			// send failed with some events already to be confirmed by ACK state machine
			// -> send error to state machine to merge ACK receive status with error
			c.ch <- ackMessage{
				tag:    tagError,
				err:    err,
				events: allEvents,
			}
			return 0, err, nil
		}

		// handle errAllEventsEncoding error on first events to be send
		if iteration == 0 {
			// encoding of all events failed. If events have not been split, asynchronously
			// confirm publish ok
			if !partial {
				go func() { cb(nil, nil) }()
				return len(events), nil, nil
			}

			// first set of events, signal errAllEventsEncoding to loop so iteration is not increased
			return len(events), err, nil
		}

		// errAllEventsEncoding at end of bulk -> signal ACK handler of partial sends
		// being finished
		if !partial {
			c.ch <- ackMessage{
				tag:    tagLast,
				err:    err,
				events: nil,
				count:  0,
			}
			return len(events), nil, nil
		}

		// errAllEventsEncoding fail in middle of bulk -> just ignore
		return len(events), nil, nil
	}

	tag := tagComplete
	if partial {
		tag = tagSubset
	}
	c.ch <- ackMessage{
		tag:    tag,
		cb:     cb,
		events: allEvents,
		count:  count,
	}

	c.tryGrowWindow(batchSize)
	return len(events), nil, nil
}

func (c *asyncClient) ackLoop() {
	for {
		var err error
		msg := <-c.ch

		switch msg.tag {
		case tagComplete:
			// just wait for ack
			seq, err := c.awaitACK(msg.count)
			msg.cb(msg.events[seq:], err)

		case tagSubset:
			err = c.ackPartialsLoop(msg)

		case tagError:
			err = msg.err
			msg.cb(msg.events, msg.err)

		default:
			panic("wrong message type received")
		}

		if err != nil {
			logp.Info("logstash publish error: %v", err)
			c.Close() // close connection
		}
	}
}

func (c *asyncClient) ackPartialsLoop(first ackMessage) error {
	current := first
	for {
		// wait for ACK first:
		seq, err := c.awaitACK(current.count)
		if err != nil {
			// ackFailedPartialsLoop tries to close the connection
			c.ackFailedPartialsLoop(current, seq, err)
			return nil
		}

		current = <-c.ch
		switch current.tag {
		case tagSubset:
			break // continue with next subset

		case tagComplete:
			// await last ack and signal completion
			seq, err := c.awaitACK(current.count)
			current.cb(current.events[seq:], err)
			return err

		case tagLast:
			// last message in partial send loop, but not expecting any ACK
			current.cb(current.events, nil)
			return nil

		case tagError:
			// signal error and break partial loop
			current.cb(current.events, err)
			return current.err
		}
	}
}

func (c *asyncClient) ackFailedPartialsLoop(
	failed ackMessage,
	seq uint32,
	err error,
) {
	// signal error and consume all partial messages
	failed.cb(failed.events, err)

	// ignore all message already in queue for partial send
	for {
		msg := <-c.ch
		if msg.tag != tagSubset {
			return
		}
	}
}

func (c *asyncClient) tryGrowWindow(batchSize int) {
	// increase window size by factor 1.5 until max window size
	// (window size grows exponentially)
	// TODO: use duration until ACK to estimate an ok max window size value
	if c.windowSize < batchSize {
		if c.maxOkWindowSize < c.windowSize {
			logp.Debug("logstash", "update max ok window size: %v < %v", c.maxOkWindowSize, c.windowSize)
			c.maxOkWindowSize = c.windowSize

			newWindowSize := int(math.Ceil(1.5 * float64(c.windowSize)))
			logp.Debug("logstash", "increate window size to: %v", newWindowSize)

			if c.windowSize < batchSize && batchSize < newWindowSize {
				logp.Debug("logstash", "set to batchSize: %v", batchSize)
				newWindowSize = batchSize
			}
			if newWindowSize > c.maxWindowSize {
				logp.Debug("logstash", "set to max window size: %v", c.maxWindowSize)
				newWindowSize = c.maxWindowSize
			}
			c.windowSize = newWindowSize
		} else if c.windowSize < c.maxOkWindowSize {
			logp.Debug("logstash", "update current window size: %v", c.windowSize)

			c.windowSize = int(math.Ceil(1.5 * float64(c.windowSize)))
			if c.windowSize > c.maxOkWindowSize {
				logp.Debug("logstash", "set to max ok window size: %v", c.maxOkWindowSize)
				c.windowSize = c.maxOkWindowSize
			}
		}
	}
}
