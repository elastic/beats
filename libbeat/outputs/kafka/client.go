package kafka

import (
	"encoding/json"
	"expvar"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Shopify/sarama"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

type client struct {
	hosts   []string
	topic   string
	useType bool
	config  sarama.Config

	producer sarama.AsyncProducer

	wg sync.WaitGroup

	isConnected int32
}

type msgRef struct {
	count int32
	err   atomic.Value
	batch []common.MapStr
	cb    func([]common.MapStr, error)
}

var (
	ackedEvents            = expvar.NewInt("libbeatKafkaPublishedAndAckedEvents")
	eventsNotAcked         = expvar.NewInt("libbeatKafkaPublishedButNotAckedEvents")
	publishEventsCallCount = expvar.NewInt("libbeatKafkaPublishEventsCallCount")
)

func newKafkaClient(hosts []string, topic string, useType bool, cfg *sarama.Config) (*client, error) {
	c := &client{
		hosts:   hosts,
		useType: useType,
		topic:   topic,
		config:  *cfg,
	}
	return c, nil
}

func (c *client) Connect(timeout time.Duration) error {
	debugf("connect: %v", c.hosts)

	c.config.Net.DialTimeout = timeout

	// try to connect
	producer, err := sarama.NewAsyncProducer(c.hosts, &c.config)
	if err != nil {
		logp.Err("Kafka connect fails with: %v", err)
		return err
	}

	c.producer = producer

	c.wg.Add(2)
	go c.successWorker(producer.Successes())
	go c.errorWorker(producer.Errors())
	atomic.StoreInt32(&c.isConnected, 1)

	return nil
}

func (c *client) Close() error {
	if c.IsConnected() {
		debugf("closed kafka client")

		c.producer.AsyncClose()
		c.wg.Wait()
		atomic.StoreInt32(&c.isConnected, 0)
		c.producer = nil
	}
	return nil
}

func (c *client) IsConnected() bool {
	return atomic.LoadInt32(&c.isConnected) != 0
}

func (c *client) AsyncPublishEvent(
	cb func(error),
	event common.MapStr,
) error {
	return c.AsyncPublishEvents(func(_ []common.MapStr, err error) {
		cb(err)
	}, []common.MapStr{event})
}

func (c *client) AsyncPublishEvents(
	cb func([]common.MapStr, error),
	events []common.MapStr,
) error {
	publishEventsCallCount.Add(1)
	debugf("publish events")

	ref := &msgRef{
		count: int32(len(events)),
		batch: events,
		cb:    cb,
	}

	ch := c.producer.Input()

	for _, event := range events {
		topic := c.topic
		if c.useType {
			topic = event["type"].(string)
		}

		jsonEvent, err := json.Marshal(event)
		if err != nil {
			ref.done()
			continue
		}

		msg := &sarama.ProducerMessage{
			Metadata: ref,
			Topic:    topic,
			Value:    sarama.ByteEncoder(jsonEvent),
		}

		ch <- msg
	}

	return nil
}

func (c *client) successWorker(ch <-chan *sarama.ProducerMessage) {
	defer c.wg.Done()
	defer debugf("Stop kafka ack worker")

	for msg := range ch {
		ref := msg.Metadata.(*msgRef)
		ref.done()
	}
}

func (c *client) errorWorker(ch <-chan *sarama.ProducerError) {
	defer c.wg.Done()
	defer debugf("Stop kafka error handler")

	for errMsg := range ch {
		msg := errMsg.Msg
		ref := msg.Metadata.(*msgRef)
		ref.fail(errMsg.Err)
	}
}

func (r *msgRef) done() {
	r.dec()
}

func (r *msgRef) fail(err error) {
	debugf("Kafka publish failed with: %v", err)

	r.err.Store(err)
	r.dec()
}

func (r *msgRef) dec() {
	i := atomic.AddInt32(&r.count, -1)
	if i > 0 {
		return
	}

	debugf("finished kafka batch")

	var err error
	v := r.err.Load()
	if v != nil {
		err = v.(error)
		eventsNotAcked.Add(int64(len(r.batch)))
		r.cb(r.batch, err)
	} else {
		ackedEvents.Add(int64(len(r.batch)))
		r.cb(nil, nil)
	}
}
