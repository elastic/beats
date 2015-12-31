package logstash

import (
	"io"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

type asyncClient struct {
	TransportClient
	*protocol
	mutex sync.Mutex

	windowSize      int32
	maxOkWindowSize int // max window size sending was successful for
	maxWindowSize   int

	countTimeoutErr int

	ch   chan ackMessage
	done chan struct{}
	wg   sync.WaitGroup
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
		windowSize:      int32(defaultStartMaxWindowSize),
		maxWindowSize:   maxWindowSize,
	}, nil
}

func (c *asyncClient) Connect(timeout time.Duration) error {
	logp.Debug("logstash", "connect (async)")
	err := c.TransportClient.Connect(timeout)
	if err == nil {
		c.startACK()
	}
	return err
}

func (c *asyncClient) Close() error {
	logp.Debug("logstash", "close (async) connection")
	c.stopACK()
	return c.closeTransport()
}

func (c *asyncClient) IsConnected() bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return c.TransportClient.IsConnected()
}

func (c *asyncClient) closeTransport() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.TransportClient.IsConnected() {
		return c.TransportClient.Close()
	}
	return nil
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
	publishEventsCallCount.Add(1)

	i := 1
	for len(events) > 0 {
		n, err, sigErr := c.publishWindowed(i, cb, events)
		debug("%v events out of %v events sent to logstash. Continue sending ...", n, len(events))
		events = events[n:]

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

	windowSize := int(atomic.LoadInt32(&c.windowSize))

	// prepare message payload
	if batchSize > windowSize {
		partial = true
		events = events[:windowSize]
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
				cb:     cb,
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
				go cb(nil, nil)
				return len(events), nil, nil
			}

			// first set of events, signal errAllEventsEncoding to loop so iteration is not increased
			return len(events), err, nil
		}

		// errAllEventsEncoding at end of bulk -> signal ACK handler of partial sends
		// being finished
		if !partial {
			c.ch <- ackMessage{
				cb:     cb,
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

	return len(events), nil, nil
}

func (c *asyncClient) startACK() {
	c.ch = make(chan ackMessage, 1)
	c.done = make(chan struct{})
	c.wg.Add(1)
	go c.ackLoop()
}

func (c *asyncClient) stopACK() {
	close(c.done)
	close(c.ch)
	c.wg.Wait()
}

func (c *asyncClient) ackLoop() {
	defer c.wg.Done()

	for {
		var err error
		var msg ackMessage

		select {
		case msg = <-c.ch:
		case <-c.done:
			c.drainACKLoop(false, true, io.EOF)
			return
		}

		inPartial := false
		switch msg.tag {
		case tagComplete:
			// just wait for ack
			var seq uint32
			seq, err = c.awaitACK(len(msg.events), msg.count)
			msg.cb(msg.events[seq:], err)

		case tagSubset:
			var end bool
			end, err = c.ackPartialsLoop(msg)
			inPartial = !end

		case tagError:
			err = msg.err
			msg.cb(msg.events, err)

		default:
			panic("wrong message type received")
		}

		// on error close and propagate error to all
		// messages left in queue, until worker
		// is stopped by client
		if err != nil {
			c.closeTransport()
			c.drainACKLoop(inPartial, true, err)
			return
		}

	}
}

func (c *asyncClient) drainACKLoop(partial, reported bool, err error) {
	for msg := range c.ch {
		switch msg.tag {
		case tagComplete, tagLast:
			if !partial || !reported {
				msg.cb(msg.events, err)
			} else {
				partial = false
				reported = false
			}

		case tagError:
			msg.cb(msg.events, msg.err)

		case tagSubset:
			partial = true
			if !reported {
				msg.cb(msg.events, err)
				reported = true
			}
		}
	}
}

func (c *asyncClient) ackPartialsLoop(first ackMessage) (bool, error) {
	current := first
	for {
		// wait for ACK first:
		seq, err := c.awaitACK(len(current.events), current.count)
		if err != nil {
			current.cb(current.events[seq:], err)
			return false, err
		}

		current = <-c.ch
		switch current.tag {
		case tagSubset:
			break // continue with next subset

		case tagComplete:
			// await last ack and signal completion
			seq, err := c.awaitACK(len(current.events), current.count)
			current.cb(current.events[seq:], err)
			return true, err

		case tagLast:
			// last message in partial send loop, but not expecting any ACK
			current.cb(current.events, nil)
			return true, nil

		case tagError:
			// signal error and break partial loop
			current.cb(current.events, err)
			return true, current.err
		}
	}
}

func (c *asyncClient) awaitACK(batchSize int, count uint32) (uint32, error) {
	seq, err := c.protocol.awaitACK(count)
	debug("awaitACK(%v) => %v, %v", count, seq, err)

	ackedEvents.Add(int64(seq))

	if err != nil {
		eventsNotAcked.Add(int64(batchSize) - int64(seq))
		c.shrinkWindow()
	} else {
		c.tryGrowWindow(batchSize)
	}
	return seq, err
}

func (c *asyncClient) tryGrowWindow(batchSize int) {
	windowSize := int(c.windowSize)

	if windowSize <= batchSize {
		if c.maxOkWindowSize < windowSize {
			logp.Debug("logstash", "update max ok window size: %v < %v", c.maxOkWindowSize, c.windowSize)
			c.maxOkWindowSize = windowSize

			newWindowSize := int(math.Ceil(1.5 * float64(windowSize)))
			logp.Debug("logstash", "increase window size to: %v", newWindowSize)

			if windowSize <= batchSize && batchSize < newWindowSize {
				logp.Debug("logstash", "set to batchSize: %v", batchSize)
				newWindowSize = batchSize
			}
			if newWindowSize > c.maxWindowSize {
				logp.Debug("logstash", "set to max window size: %v", c.maxWindowSize)
				newWindowSize = int(c.maxWindowSize)
			}

			windowSize = newWindowSize
		} else if windowSize < c.maxOkWindowSize {
			logp.Debug("logstash", "update current window size: %v", c.windowSize)

			windowSize = int(math.Ceil(1.5 * float64(windowSize)))
			if windowSize > c.maxOkWindowSize {
				logp.Debug("logstash", "set to max ok window size: %v", c.maxOkWindowSize)
				windowSize = c.maxOkWindowSize
			}
		}

		atomic.StoreInt32(&c.windowSize, int32(windowSize))
	}
}

func (c *asyncClient) shrinkWindow() {
	windowSize := int(c.windowSize)
	orig := windowSize

	windowSize = windowSize / 2
	if windowSize < minWindowSize {
		windowSize = minWindowSize
		if windowSize == orig {
			return
		}
	}

	atomic.StoreInt32(&c.windowSize, int32(windowSize))
}
