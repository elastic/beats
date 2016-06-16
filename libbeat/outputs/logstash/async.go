package logstash

import (
	"sync/atomic"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs/transport"
	"github.com/urso/go-lumber/client/v2"
)

type asyncClient struct {
	*transport.Client
	client *v2.AsyncClient
	win    window

	connect func() error
}

type msgRef struct {
	count     int32
	batch     []common.MapStr
	err       error
	cb        func([]common.MapStr, error)
	win       *window
	batchSize int
}

func newAsyncLumberjackClient(
	conn *transport.Client,
	queueSize int,
	compressLevel int,
	maxWindowSize int,
	timeout time.Duration,
	beat string,
) (*asyncClient, error) {
	c := &asyncClient{}
	c.Client = conn
	c.win.init(defaultStartMaxWindowSize, maxWindowSize)

	enc, err := makeLogstashEventEncoder(beat)
	if err != nil {
		return nil, err
	}

	c.connect = func() error {
		err := c.Client.Connect()
		if err == nil {
			c.client, err = v2.NewAsyncClientWithConn(c.Client,
				queueSize,
				v2.JSONEncoder(enc),
				v2.Timeout(timeout),
				v2.CompressionLevel(compressLevel))
		}
		return err
	}
	return c, nil
}

func (c *asyncClient) Connect(timeout time.Duration) error {
	logp.Debug("logstash", "connect")
	return c.connect()
}

func (c *asyncClient) Close() error {
	logp.Debug("logstash", "close connection")
	if c.client != nil {
		err := c.client.Close()
		c.client = nil
		return err
	}
	return c.Client.Close()
}

func (c *asyncClient) AsyncPublishEvent(
	cb func(error),
	event common.MapStr,
) error {
	data := []interface{}{event}
	return c.client.Send(func(seq uint32, err error) { cb(err) }, data)
}

func (c *asyncClient) AsyncPublishEvents(
	cb func([]common.MapStr, error),
	events []common.MapStr,
) error {
	publishEventsCallCount.Add(1)

	if len(events) == 0 {
		debug("send nil")
		cb(nil, nil)
		return nil
	}

	ref := &msgRef{
		count:     1,
		batch:     events,
		batchSize: len(events),
		win:       &c.win,
		cb:        cb,
		err:       nil,
	}

	for len(events) > 0 {
		n, err := c.publishWindowed(ref, events)

		debug("%v events out of %v events sent to logstash. Continue sending",
			n, len(events))

		events = events[n:]
		if err != nil {
			c.win.shrinkWindow()
			_ = c.Close()

			logp.Err("Failed to publish events caused by: %v", err)
			eventsNotAcked.Add(int64(len(events)))
			return err
		}
	}
	ref.dec()

	return nil
}

func (c *asyncClient) publishWindowed(
	ref *msgRef,
	events []common.MapStr,
) (int, error) {
	batchSize := len(events)
	windowSize := c.win.get()
	debug("Try to publish %v events to logstash with window size %v",
		batchSize, windowSize)

	// prepare message payload
	if batchSize > windowSize {
		events = events[:windowSize]
	}

	err := c.sendEvents(ref, events)
	if err != nil {
		return 0, err
	}

	return len(events), nil
}

func (c *asyncClient) sendEvents(ref *msgRef, events []common.MapStr) error {
	window := make([]interface{}, len(events))
	for i, event := range events {
		window[i] = event
	}
	atomic.AddInt32(&ref.count, 1)
	return c.client.Send(ref.callback, window)
}

func (r *msgRef) callback(seq uint32, err error) {
	if err != nil {
		r.fail(seq, err)
	} else {
		r.done(seq)
	}
}

func (r *msgRef) done(n uint32) {
	ackedEvents.Add(int64(n))
	r.batch = r.batch[n:]
	r.win.tryGrowWindow(r.batchSize)
	r.dec()
}

func (r *msgRef) fail(n uint32, err error) {
	ackedEvents.Add(int64(n))
	r.err = err
	r.batch = r.batch[n:]
	r.win.shrinkWindow()
	r.dec()
}

func (r *msgRef) dec() {
	i := atomic.AddInt32(&r.count, -1)
	if i > 0 {
		return
	}

	err := r.err
	if err != nil {
		eventsNotAcked.Add(int64(len(r.batch)))
		r.cb(r.batch, err)
	} else {
		r.cb(nil, nil)
	}
}
