package logstash

import (
	"io"
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/common"
)

type asyncClient struct {
	TransportClient
	*protocol
	mutex sync.Mutex

	win window

	ch   chan ackMessage
	done chan struct{}
	wg   sync.WaitGroup
}

type ackMessage struct {
	connID uint

	tag       tag
	cb        func([]common.MapStr, error)
	outEvents []common.MapStr
	events    []common.MapStr
	err       error
	count     uint32
}

type tag uint8

const (
	tagUndefined tag = iota
	tagComplete
	tagSubset
	tagLast
	tagError
)

const (
	noSeq uint32 = 0xffffffff
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

	c := &asyncClient{
		TransportClient: conn,
		protocol:        p,
	}
	c.win.init(defaultStartMaxWindowSize, maxWindowSize)
	return c, nil
}

func (c *asyncClient) Connect(timeout time.Duration) error {
	debug("connect (async)")
	err := c.TransportClient.Connect(timeout)
	if err == nil {
		c.startACK()
	}
	return err
}

func (c *asyncClient) Close() error {
	debug("close (async) connection")
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

	debug("AsyncPublishEvents")

	if len(events) == 0 {
		debug("send nil")
		cb(nil, nil)
		return nil
	}

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
	var restEvents []common.MapStr
	batchSize := len(events)
	partial := false
	debug("Try to publish %v events to logstash with window size %v",
		len(events), c.win.windowSize)

	windowSize := c.win.get()

	// prepare message payload
	if batchSize > windowSize {
		partial = true
		restEvents = events[windowSize:]
		events = events[:windowSize]
	}

	outEvents, err := c.sendEvents(events)
	count := uint32(len(outEvents))
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

	debug("send message to ack worker")
	if len(outEvents) == len(events) {
		c.ch <- ackMessage{
			tag:    tag,
			cb:     cb,
			events: allEvents,
			count:  count,
		}
	} else {
		c.ch <- ackMessage{
			tag:       tag,
			cb:        cb,
			outEvents: outEvents,
			events:    restEvents,
			count:     count,
		}
	}

	debug("return sender")
	return len(events), nil, nil
}

func (c *asyncClient) startACK() {
	debug("start async ACK handler")
	c.ch = make(chan ackMessage, 1)
	c.done = make(chan struct{})
	c.wg.Add(1)
	go c.ackLoop()
}

func (c *asyncClient) stopACK() {
	debug("stop ackLoop")
	close(c.done)
	c.wg.Wait()
	close(c.ch)
	debug("stopped async ACK handler")
}

func (c *asyncClient) ackLoop() {
	defer c.wg.Done()
	defer debug("finished ackLoop")

	debug("start ackLoop")

	for {
		var err error
		var msg ackMessage

		select {
		case msg = <-c.ch:
		case <-c.done:
			c.drainACKLoop(false, true, io.EOF)
			return
		}

		debug("new ack message")

		inPartial := false
		switch msg.tag {
		case tagComplete:
			// just wait for ack
			var seq uint32
			seq, err = c.awaitACK(len(msg.events), msg.count)
			doCallback(msg, seq, err)

		case tagSubset:
			var end bool
			end, err = c.ackPartialsLoop(msg)
			inPartial = !end

		case tagError:
			err = msg.err
			doCallback(msg, noSeq, err)

		default:
			panic("wrong message type received")
		}

		// on error close and propagate error to all
		// messages left in queue, until worker
		// is stopped by client
		if err != nil {
			c.closeTransport()
			c.drainACKLoop(inPartial, true, err)
			debug("return ackLoop due to error")
			return
		}
	}
}

func (c *asyncClient) drainACKLoop(partial, reported bool, err error) {
	debug("drainACKLoop, p=%v, r=%v, err='%v'", partial, reported, err)
	defer debug("finished drainACKLoop")

	for {
		var msg ackMessage

		select {
		case msg = <-c.ch:
		default:
			return
		}

		switch msg.tag {
		case tagComplete, tagLast:
			if !partial || !reported {
				doCallback(msg, noSeq, err)
			} else {
				partial = false
				reported = false
			}

		case tagError:
			doCallback(msg, noSeq, msg.err)

		case tagSubset:
			partial = true
			if !reported {
				doCallback(msg, noSeq, err)
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
			doCallback(current, seq, err)
			return false, err
		}

		current = <-c.ch
		switch current.tag {
		case tagSubset:
			break // continue with next subset

		case tagComplete:
			// await last ack and signal completion
			seq, err := c.awaitACK(len(current.events), current.count)
			doCallback(current, seq, err)
			return true, err

		case tagLast:
			// last message in partial send loop, but not expecting any ACK
			doCallback(current, noSeq, nil)
			return true, nil

		case tagError:
			// signal error and break partial loop
			doCallback(current, noSeq, err)
			return true, current.err
		}
	}
}

func doCallback(msg ackMessage, seq uint32, err error) {
	events := msg.events
	if seq == noSeq {
		if msg.outEvents != nil {
			events = append(msg.outEvents, msg.events...)
		}
	} else {
		if msg.count > seq {
			if msg.outEvents != nil {
				events = append(msg.outEvents, msg.events...)
			}
			events = events[seq:]
		} else if msg.outEvents == nil {
			events = events[seq:]
		}
	}

	msg.cb(events, err)
}

func (c *asyncClient) awaitACK(batchSize int, count uint32) (uint32, error) {
	seq, err := c.protocol.awaitACK(count)
	debug("awaitACK(%v) => %v, %v", count, seq, err)
	ackedEvents.Add(int64(seq))

	if err != nil {
		eventsNotAcked.Add(int64(batchSize) - int64(seq))
		c.win.shrinkWindow()
	} else {
		c.win.tryGrowWindow(batchSize)
	}
	return seq, err
}
