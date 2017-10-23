package logstash

import (
	"sync/atomic"
	"time"

	"github.com/elastic/go-lumber/client/v2"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/transport"
)

type asyncClient struct {
	*transport.Client
	client *v2.AsyncClient
	host   string
	win    *window

	connect func() error
}

type msgRef struct {
	client    *asyncClient
	count     int32
	batch     []outputs.Data
	err       error
	cb        func([]outputs.Data, error)
	win       *window
	batchSize int
}

func newAsyncLumberjackClient(
	conn *transport.Client,
	addr string,
	config *logstashConfig,
) (*asyncClient, error) {
	c := &asyncClient{
		Client: conn,
		host:   addr,
	}

	if config.SlowStart {
		maxWindowSize := config.BulkMaxSize
		c.win = newWindower(defaultStartMaxWindowSize, maxWindowSize)
	}

	queueSize := config.Pipelining - 1
	timeout := config.Timeout
	compressLevel := config.CompressionLevel

	enc, err := makeLogstashEventEncoder(config.Index)
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
	logp.Debug("logstash", "connect to logstash host %v", c.host)
	return c.connect()
}

func (c *asyncClient) Close() error {
	logp.Debug("logstash", "close connection to logstash host %v", c.host)
	if c.client != nil {
		err := c.client.Close()
		c.client = nil
		return err
	}
	return c.Client.Close()
}

func (c *asyncClient) AsyncPublishEvent(
	cb func(error),
	data outputs.Data,
) error {
	return c.client.Send(
		func(seq uint32, err error) {
			cb(err)
		},
		[]interface{}{data},
	)
}

func (c *asyncClient) AsyncPublishEvents(
	cb func([]outputs.Data, error),
	data []outputs.Data,
) error {
	publishEventsCallCount.Add(1)

	if len(data) == 0 {
		debug("send nil")
		cb(nil, nil)
		return nil
	}

	ref := &msgRef{
		client:    c,
		count:     1,
		batch:     data,
		batchSize: len(data),
		win:       c.win,
		cb:        cb,
		err:       nil,
	}
	defer ref.dec()

	for len(data) > 0 {
		var (
			n   int
			err error
		)

		if c.win == nil {
			n = len(data)
			err = c.sendEvents(ref, data)
		} else {
			n, err = c.publishWindowed(ref, data)
		}

		debug("%v events out of %v events sent to logstash host %s. Continue sending",
			n, len(data), c.host)

		data = data[n:]
		if err != nil {
			_ = c.Close()
			return err
		}
	}

	return nil
}

func (c *asyncClient) publishWindowed(
	ref *msgRef,
	data []outputs.Data,
) (int, error) {
	batchSize := len(data)
	windowSize := c.win.get()
	debug("Try to publish %v events to logstash host %v with window size %v",
		batchSize, c.host, windowSize)

	// prepare message payload
	if batchSize > windowSize {
		data = data[:windowSize]
	}

	err := c.sendEvents(ref, data)
	if err != nil {
		return 0, err
	}

	return len(data), nil
}

func (c *asyncClient) sendEvents(ref *msgRef, data []outputs.Data) error {
	window := make([]interface{}, len(data))
	for i, d := range data {
		window[i] = d
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
	if r.win != nil {
		r.win.tryGrowWindow(r.batchSize)
	}
	r.dec()
}

func (r *msgRef) fail(n uint32, err error) {
	ackedEvents.Add(int64(n))
	if r.err == nil {
		r.err = err
	}
	r.batch = r.batch[n:]
	if r.win != nil {
		r.win.shrinkWindow()
	}
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
		logp.Err("Failed to publish events (host: %v) caused by: %v", r.client.host, err)
		r.cb(r.batch, err)
		return
	}

	r.cb(nil, nil)
}
