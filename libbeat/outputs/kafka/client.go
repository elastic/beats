package kafka

import (
	"encoding/json"
	"expvar"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Shopify/sarama"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/fmtstr"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/outil"
)

type client struct {
	hosts  []string
	topic  outil.Selector
	key    *fmtstr.EventFormatString
	config sarama.Config

	producer sarama.AsyncProducer

	wg sync.WaitGroup
}

type msgRef struct {
	count int32
	batch []outputs.Data
	cb    func([]outputs.Data, error)

	err error
}

var (
	ackedEvents            = expvar.NewInt("libbeat.kafka.published_and_acked_events")
	eventsNotAcked         = expvar.NewInt("libbeat.kafka.published_but_not_acked_events")
	publishEventsCallCount = expvar.NewInt("libbeat.kafka.call_count.PublishEvents")
)

func newKafkaClient(
	hosts []string,
	key *fmtstr.EventFormatString,
	topic outil.Selector,
	cfg *sarama.Config,
) (*client, error) {
	c := &client{
		hosts:  hosts,
		topic:  topic,
		key:    key,
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
	data outputs.Data,
) error {
	return c.AsyncPublishEvents(func(_ []outputs.Data, err error) {
		cb(err)
	}, []outputs.Data{data})
}

func (c *client) AsyncPublishEvents(
	cb func([]outputs.Data, error),
	data []outputs.Data,
) error {
	publishEventsCallCount.Add(1)
	debugf("publish events")

	ref := &msgRef{
		count: int32(len(data)),
		batch: data,
		cb:    cb,
	}

	ch := c.producer.Input()

	for i := range data {
		d := &data[i]

		msg, err := c.getEventMessage(d)
		if err != nil {
			logp.Err("Dropping event: %v", err)
			ref.done()
			continue
		}
		msg.ref = ref

		msg.initProducerMessage()
		ch <- &msg.msg
	}

	return nil
}

func (c *client) getEventMessage(data *outputs.Data) (*message, error) {
	event := data.Event
	msg := messageFromData(data)
	if msg.topic != "" {
		return msg, nil
	}

	msg.event = event

	topic, err := c.topic.Select(event)
	if err != nil {
		return nil, fmt.Errorf("setting kafka topic failed with %v", err)
	}
	msg.topic = topic

	jsonEvent, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("json encoding failed with %v", err)
	}
	msg.value = jsonEvent

	// message timestamps have been added to kafka with version 0.10.0.0
	var ts time.Time
	if c.config.Version.IsAtLeast(sarama.V0_10_0_0) {
		if tsRaw, ok := event["@timestamp"]; ok {
			if tmp, ok := tsRaw.(common.Time); ok {
				ts = time.Time(tmp)
			} else if tmp, ok := tsRaw.(time.Time); ok {
				ts = tmp
			}
		}
	}
	msg.ts = ts

	if c.key != nil {
		if key, err := c.key.RunBytes(event); err == nil {
			msg.key = key
		}
	}

	return msg, nil
}

func (c *client) successWorker(ch <-chan *sarama.ProducerMessage) {
	defer c.wg.Done()
	defer debugf("Stop kafka ack worker")

	for libMsg := range ch {
		msg := libMsg.Metadata.(*message)
		msg.ref.done()
	}
}

func (c *client) errorWorker(ch <-chan *sarama.ProducerError) {
	defer c.wg.Done()
	defer debugf("Stop kafka error handler")

	for errMsg := range ch {
		msg := errMsg.Msg.Metadata.(*message)
		msg.ref.fail(errMsg.Err)
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
