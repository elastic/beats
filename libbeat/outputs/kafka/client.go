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
	"github.com/elastic/beats/libbeat/outputs/outil"
)

type client struct {
	hosts  []string
	topic  outil.Selector
	config sarama.Config

	producer sarama.AsyncProducer

	wg sync.WaitGroup
}

type msgRef struct {
	count int32
	batch []common.MapStr
	cb    func([]common.MapStr, error)

	err error
}

var (
	ackedEvents            = expvar.NewInt("libbeat.kafka.published_and_acked_events")
	eventsNotAcked         = expvar.NewInt("libbeat.kafka.published_but_not_acked_events")
	publishEventsCallCount = expvar.NewInt("libbeat.kafka.call_count.PublishEvents")
)

func newKafkaClient(
	hosts []string,
	topic outil.Selector,
	cfg *sarama.Config,
) (*client, error) {
	c := &client{
		hosts:  hosts,
		topic:  topic,
		config: *cfg,
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

	return nil
}

func (c *client) Close() error {
	debugf("closed kafka client")

	c.producer.AsyncClose()
	c.wg.Wait()
	c.producer = nil
	return nil
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
		topic, err := c.topic.Select(event)

		var ts time.Time

		// message timestamps have been added to kafka with version 0.10.0.0
		if c.config.Version.IsAtLeast(sarama.V0_10_0_0) {
			if tsRaw, ok := event["@timestamp"]; ok {
				if tmp, ok := tsRaw.(common.Time); ok {
					ts = time.Time(tmp)
				} else if tmp, ok := tsRaw.(time.Time); ok {
					ts = tmp
				}
			}
		}

		jsonEvent, err := json.Marshal(event)
		if err != nil {
			ref.done()
			continue
		}

		msg := &sarama.ProducerMessage{
			Metadata:  ref,
			Topic:     topic,
			Value:     sarama.ByteEncoder(jsonEvent),
			Timestamp: ts,
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
	r.err = err
	r.dec()
}

func (r *msgRef) dec() {
	i := atomic.AddInt32(&r.count, -1)
	if i > 0 {
		return
	}

	debugf("finished kafka batch")

	err := r.err
	if err != nil {
		eventsNotAcked.Add(int64(len(r.batch)))
		debugf("Kafka publish failed with: %v", err)
		r.cb(r.batch, err)
	} else {
		ackedEvents.Add(int64(len(r.batch)))
		r.cb(nil, nil)
	}
}
