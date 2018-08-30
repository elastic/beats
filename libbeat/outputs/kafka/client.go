// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package kafka

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/Shopify/sarama"

	"github.com/elastic/beats/libbeat/common/fmtstr"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/codec"
	"github.com/elastic/beats/libbeat/outputs/outil"
	"github.com/elastic/beats/libbeat/publisher"
	"github.com/elastic/beats/libbeat/common"
	"time"
)

type client struct {
	observer outputs.Observer
	hosts    []string
	topic    outil.Selector
	key      *fmtstr.EventFormatString
	index    string
	codec    codec.Codec
	config   sarama.Config

	producer sarama.AsyncProducer

	wg sync.WaitGroup
}

type msgRef struct {
	client  *client
	count   int32
	total   int
	failed  []publisher.Event
	batch   publisher.Batch
	backoff *common.Backoff
	retry   bool

	err error
}

var (
	errNoTopicsSelected = errors.New("no topic could be selected")
	stop = make(chan interface{})
	publish = make(chan *msgRef, 250)
	stopRequested = false
)

func newKafkaClient(
	observer outputs.Observer,
	hosts []string,
	index string,
	key *fmtstr.EventFormatString,
	topic outil.Selector,
	writer codec.Codec,
	cfg *sarama.Config,
) (*client, error) {
	c := &client{
		observer: observer,
		hosts:    hosts,
		topic:    topic,
		key:      key,
		index:    index,
		codec:    writer,
		config:   *cfg,
	}
	return c, nil
}

func (c *client) Connect() error {
	debugf("connect: %v", c.hosts)

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
	go c.publishLoop()

	return nil
}

func (c *client) Close() error {
	debugf("closed kafka client")

	stop <- true
	c.producer.AsyncClose()
	c.wg.Wait()
	c.producer = nil
	return nil
}

func (c *client) reconnect() error {
fmt.Printf("reconnecting\n")
	c.producer.AsyncClose()
	c.wg.Wait()
	c.producer = nil

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

func (c *client) Publish(batch publisher.Batch) error {
	events := batch.Events()
	c.observer.NewBatch(len(events))

	ref := &msgRef{
		client:  c,
		count:   int32(len(events)),
		total:   len(events),
		failed:  nil,
		batch:   batch,
		retry:   false,
		backoff: common.NewBackoff(nil, 1*time.Second, 60*time.Second),
	}

	publish <- ref
	return nil
}

func (c *client) publish(msgRef *msgRef) error {
	if msgRef.retry {
		c.reconnect()
	}
	ch := c.producer.Input()
	events := msgRef.batch.Events()

	fmt.Printf("publishing %d events\n", len(events))
	for i := range events {
		d := &events[i]
		msg, err := c.getEventMessage(d)
		if err != nil {
			logp.Err("Dropping event: %v", err)
			msgRef.done()
			c.observer.Dropped(1)
			continue
		}

		msg.ref = msgRef
		msg.initProducerMessage()
		ch <- &msg.msg
	}

	return nil
}

func (c *client) publishLoop() {
	for !stopRequested {
		select {
		case msgRef := <-publish:
			fmt.Printf("pulled %d events in publish loop\n", len(msgRef.batch.Events()))
			c.publish(msgRef)
		case <-stop:
			fmt.Printf("stop requested\n")
			stopRequested = true
		}
	}

	// flush remaining messages
	for m := range publish {
		fmt.Printf("flushing message\n")
		c.publish(m)
	}
}

func (c *client) String() string {
	return "kafka(" + strings.Join(c.hosts, ",") + ")"
}

func (c *client) getEventMessage(data *publisher.Event) (*message, error) {
	event := &data.Content
	msg := &message{partition: -1, data: *data}
	if event.Meta != nil {
		if value, ok := event.Meta["partition"]; ok {
			if partition, ok := value.(int32); ok {
				msg.partition = partition
			}
		}

		if value, ok := event.Meta["topic"]; ok {
			if topic, ok := value.(string); ok {
				msg.topic = topic
			}
		}
	}
	if msg.topic == "" {
		topic, err := c.topic.Select(event)
		if err != nil {
			return nil, fmt.Errorf("setting kafka topic failed with %v", err)
		}
		if topic == "" {
			return nil, errNoTopicsSelected
		}
		msg.topic = topic
		if event.Meta == nil {
			event.Meta = map[string]interface{}{}
		}
		event.Meta["topic"] = topic
	}

	serializedEvent, err := c.codec.Encode(c.index, event)
	if err != nil {
		return nil, err
	}

	buf := make([]byte, len(serializedEvent))
	copy(buf, serializedEvent)
	msg.value = buf

	// message timestamps have been added to kafka with version 0.10.0.0
	if c.config.Version.IsAtLeast(sarama.V0_10_0_0) {
		msg.ts = event.Timestamp
	}

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
		msg.ref.fail(msg, errMsg.Err)
	}
}

func (c *client) retry(r *msgRef) {
	go func() {
		r.backoff.Wait()
		fmt.Printf("Waited for %d seconds before retrying message\n", r.backoff.Duration()/time.Second)
		publish <- r
	}()
}

func (r *msgRef) done() {
	r.dec()
}

func (r *msgRef) fail(msg *message, err error) {
	switch err {
	case sarama.ErrInvalidMessage:
		logp.Err("Kafka (topic=%v): dropping invalid message", msg.topic)

	case sarama.ErrMessageSizeTooLarge, sarama.ErrInvalidMessageSize:
		logp.Err("Kafka (topic=%v): dropping too large message of size %v.",
			msg.topic,
			len(msg.key)+len(msg.value))

	default:
		r.failed = append(r.failed, msg.data)
		r.err = err
	}
	r.dec()
}

func (r *msgRef) dec() {
	i := atomic.AddInt32(&r.count, -1)
	fmt.Printf("dec called with count-1 = %d\n", i)
	if i > 0 {
		return
	}

	debugf("finished kafka batch")
	stats := r.client.observer

	err := r.err
	if err != nil {
		failed := len(r.failed)
		fmt.Printf("found %d failed events\n", failed)
		success := r.total - failed
		//r.batch.RetryEvents(r.failed)

		mr := &msgRef{
			client:  r.client,
			count:   int32(len(r.failed)),
			total:   len(r.failed),
			failed:  nil,
			batch:   newRetryBatch(r.failed),
			retry:   true,
			backoff: common.NewBackoff(nil, r.backoff.Duration(), 60*time.Second),
			err:     nil,
		}

		mr.client.retry(mr)

		stats.Failed(failed)
		if success > 0 {
			stats.Acked(success)
		}

		debugf("Kafka publish failed with: %v", err)
	} else {
		r.batch.ACK()
		stats.Acked(r.total)
	}
}

type retryBatch struct {
	events   []publisher.Event
}

func newRetryBatch(in []publisher.Event) *retryBatch {
	events := make([]publisher.Event, len(in))
	for i, c := range in {
		events[i] = c
	}
	return &retryBatch{events: events}
}

func (b *retryBatch) Events() []publisher.Event {
	return b.events
}

func (b *retryBatch) ACK() {
	//b.doSignal(BatchSignal{Tag: BatchACK})
}

func (b *retryBatch) Drop() {
	//b.doSignal(BatchSignal{Tag: BatchDrop})
}

func (b *retryBatch) Retry() {
	//b.doSignal(BatchSignal{Tag: BatchRetry})
}

func (b *retryBatch) RetryEvents(events []publisher.Event) {
	//b.doSignal(BatchSignal{Tag: BatchRetryEvents, Events: events})
}

func (b *retryBatch) Cancelled() {
	//b.doSignal(BatchSignal{Tag: BatchCancelled})
}

func (b *retryBatch) CancelledEvents(events []publisher.Event) {
	//b.doSignal(BatchSignal{Tag: BatchCancelledEvents, Events: events})
}
