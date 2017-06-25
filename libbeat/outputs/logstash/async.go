package logstash

import (
	"net"
	"time"

	"github.com/elastic/beats/libbeat/common/atomic"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/transport"
	"github.com/elastic/beats/libbeat/publisher"
	"github.com/elastic/go-lumber/client/v2"
)

type asyncClient struct {
	*transport.Client
	client *v2.AsyncClient
	win    window

	connect func() error
}

type msgRef struct {
	count     atomic.Uint32
	batch     publisher.Batch
	slice     []publisher.Event
	err       error
	win       *window
	batchSize int
}

func newAsyncClient(conn *transport.Client, config *Config) (*asyncClient, error) {
	c := &asyncClient{}
	c.Client = conn
	c.win.init(defaultStartMaxWindowSize, config.BulkMaxSize)

	if config.TTL != 0 {
		logp.Warn(`The async Logstash client does not support the "ttl" option`)
	}

	enc := makeLogstashEventEncoder(config.Index)

	queueSize := config.Pipelining - 1
	timeout := config.Timeout
	compressLvl := config.CompressionLevel
	clientFactory := makeClientFactory(queueSize, timeout, enc, compressLvl)

	var err error
	c.client, err = clientFactory(c.Client)
	if err != nil {
		return nil, err
	}

	c.connect = func() error {
		err := c.Client.Connect()
		if err == nil {
			c.client, err = clientFactory(c.Client)
		}
		return err
	}

	return c, nil
}

func makeClientFactory(
	queueSize int,
	timeout time.Duration,
	enc func(interface{}) ([]byte, error),
	compressLvl int,
) func(net.Conn) (*v2.AsyncClient, error) {
	return func(conn net.Conn) (*v2.AsyncClient, error) {
		return v2.NewAsyncClientWithConn(conn, queueSize,
			v2.JSONEncoder(enc),
			v2.Timeout(timeout),
			v2.CompressionLevel(compressLvl),
		)
	}
}

func (c *asyncClient) Connect() error {
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

func (c *asyncClient) BatchSize() int {
	return c.win.get()
}

func (c *asyncClient) Publish(batch publisher.Batch) error {
	publishEventsCallCount.Add(1)

	events := batch.Events()
	if len(events) == 0 {
		batch.ACK()
		return nil
	}

	window := make([]interface{}, len(events))
	for i := range events {
		window[i] = &events[i]
	}

	ref := &msgRef{
		count:     atomic.MakeUint32(1),
		batch:     batch,
		slice:     events,
		batchSize: len(events),
		win:       &c.win,
		err:       nil,
	}
	defer ref.dec()

	for len(events) > 0 {
		n, err := c.publishWindowed(ref, events)

		debugf("%v events out of %v events sent to logstash. Continue sending",
			n, len(events))

		events = events[n:]
		if err != nil {
			_ = c.Close()
			return err
		}
	}

	return nil
}

func (c *asyncClient) publishWindowed(
	ref *msgRef,
	events []publisher.Event,
) (int, error) {
	batchSize := len(events)
	windowSize := c.win.get()

	debugf("Try to publish %v events to logstash with window size %v",
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

func (c *asyncClient) sendEvents(ref *msgRef, events []publisher.Event) error {
	window := make([]interface{}, len(events))
	for i := range events {
		window[i] = &events[i].Content
	}
	ref.count.Inc()
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
	r.slice = r.slice[n:]
	r.win.tryGrowWindow(r.batchSize)
	r.dec()
}

func (r *msgRef) fail(n uint32, err error) {
	ackedEvents.Add(int64(n))
	outputs.AckedEvents.Add(int64(n))

	if r.err == nil {
		r.err = err
	}
	r.slice = r.slice[n:]
	r.win.shrinkWindow()
	r.dec()
}

func (r *msgRef) dec() {
	i := r.count.Dec()
	if i > 0 {
		return
	}

	err := r.err
	if err == nil {
		r.batch.ACK()
		return
	}

	rest := int64(len(r.slice))
	r.batch.RetryEvents(r.slice)
	eventsNotAcked.Add(rest)
	logp.Err("Failed to publish events caused by: %v", err)
}
