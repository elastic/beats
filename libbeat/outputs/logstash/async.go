package logstash

import (
	"sync"
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
	win    window
	ticker *time.Ticker
	mutex  sync.Mutex

	connect func() error
}

type msgRef struct {
	count     int32
	batch     []outputs.Data
	err       error
	cb        func([]outputs.Data, error)
	win       *window
	batchSize int
	client    *v2.AsyncClient
	mu        *sync.Mutex
}

func newAsyncLumberjackClient(
	conn *transport.Client,
	queueSize int,
	compressLevel int,
	maxWindowSize int,
	timeout time.Duration,
	ttl time.Duration,
	beat string,
) (*asyncClient, error) {
	c := &asyncClient{}
	c.Client = conn
	c.win.init(defaultStartMaxWindowSize, maxWindowSize)
	if ttl > 0 {
		c.ticker = time.NewTicker(ttl)
	}

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
	if c.ticker != nil {
		c.ticker.Stop()
	}
	logp.Debug("logstash", "close connection")
	if c.client != nil {
		err := closeClient(c.client, &c.mutex)
		c.client = nil
		return err
	}
	return c.Client.Close()
}

func closeClient(c *v2.AsyncClient, mutex *sync.Mutex) error {
	mutex.Lock()
	defer mutex.Unlock()
	return c.Close()
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
		count:     1,
		batch:     data,
		batchSize: len(data),
		win:       &c.win,
		cb:        cb,
		err:       nil,
		mu:        &c.mutex,
	}
	defer ref.dec()

	if c.ticker != nil {
		select {
		case <-c.ticker.C:
			if err := c.connect(); err != nil {
				return err
			}
			// reset window size on reconnect
			c.win.windowSize = int32(defaultStartMaxWindowSize)
			ref.client = c.client
		default:
		}
	}
	for len(data) > 0 {
		n, err := c.publishWindowed(ref, data)

		debug("%v events out of %v events sent to logstash. Continue sending",
			n, len(data))

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
	debug("Try to publish %v events to logstash with window size %v",
		batchSize, windowSize)

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
	if ref.client != nil {
		return ref.client.Send(ref.callback, window)
	}
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
	outputs.AckedEvents.Add(int64(n))
	r.batch = r.batch[n:]
	r.win.tryGrowWindow(r.batchSize)
	r.dec()
}

func (r *msgRef) fail(n uint32, err error) {
	ackedEvents.Add(int64(n))
	outputs.AckedEvents.Add(int64(n))
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
		logp.Err("Failed to publish events caused by: %v", err)
		r.cb(r.batch, err)
	} else {
		r.cb(nil, nil)
		if r.client != nil {
			_ = closeClient(r.client, r.mu)
			r.client = nil
		}
	}
}
