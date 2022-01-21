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
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/Shopify/sarama"

	"github.com/elastic/beats/v7/libbeat/common/fmtstr"
	"github.com/elastic/beats/v7/libbeat/common/transport"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/codec"
	"github.com/elastic/beats/v7/libbeat/outputs/outil"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/testing"
)

type client struct {
	log      *logp.Logger
	observer outputs.Observer
	hosts    []string
	topic    outil.Selector
	key      *fmtstr.EventFormatString
	index    string
	codec    codec.Codec
	config   sarama.Config
	mux      sync.Mutex

	producer sarama.AsyncProducer

	wg sync.WaitGroup

	recordHeaders []sarama.RecordHeader
}

type msgRef struct {
	client *client
	count  int32
	total  int
	failed []publisher.Event
	batch  publisher.Batch

	err error
}

var (
	errNoTopicsSelected = errors.New("no topic could be selected")
)

func newKafkaClient(
	observer outputs.Observer,
	hosts []string,
	index string,
	key *fmtstr.EventFormatString,
	topic outil.Selector,
	headers map[string]string,
	writer codec.Codec,
	cfg *sarama.Config,
) (*client, error) {
	c := &client{
		log:      logp.NewLogger(logSelector),
		observer: observer,
		hosts:    hosts,
		topic:    topic,
		key:      key,
		index:    strings.ToLower(index),
		codec:    writer,
		config:   *cfg,
	}

	if len(headers) != 0 {
		recordHeaders := make([]sarama.RecordHeader, 0)
		for k, v := range headers {
			recordHeader := sarama.RecordHeader{
				Key:   []byte(k),
				Value: []byte(v),
			}

			recordHeaders = append(recordHeaders, recordHeader)
		}
		c.recordHeaders = recordHeaders
	}
	return c, nil
}

func (c *client) Connect() error {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.log.Debugf("connect: %v", c.hosts)

	// try to connect
	producer, err := sarama.NewAsyncProducer(c.hosts, &c.config)
	if err != nil {
		c.log.Errorf("Kafka connect fails with: %+v", err)
		return err
	}

	c.producer = producer

	c.wg.Add(2)
	go c.successWorker(producer.Successes())
	go c.errorWorker(producer.Errors())

	return nil
}

func (c *client) Close() error {
	c.mux.Lock()
	defer c.mux.Unlock()
	c.log.Debug("closed kafka client")

	// producer was not created before the close() was called.
	if c.producer == nil {
		return nil
	}

	c.producer.AsyncClose()
	c.wg.Wait()
	c.producer = nil
	return nil
}

func (c *client) Publish(_ context.Context, batch publisher.Batch) error {
	events := batch.Events()
	c.observer.NewBatch(len(events))

	ref := &msgRef{
		client: c,
		count:  int32(len(events)),
		total:  len(events),
		failed: nil,
		batch:  batch,
	}

	ch := c.producer.Input()
	for i := range events {
		d := &events[i]
		msg, err := c.getEventMessage(d)
		if err != nil {
			c.log.Errorf("Dropping event: %+v", err)
			ref.done()
			c.observer.Dropped(1)
			continue
		}

		msg.ref = ref
		msg.initProducerMessage()
		ch <- &msg.msg
	}

	return nil
}

func (c *client) String() string {
	return "kafka(" + strings.Join(c.hosts, ",") + ")"
}

func (c *client) getEventMessage(data *publisher.Event) (*message, error) {
	event := &data.Content
	msg := &message{partition: -1, data: *data}

	value, err := data.Cache.GetValue("partition")
	if err == nil {
		if c.log.IsDebug() {
			c.log.Debugf("got event.Meta[\"partition\"] = %v", value)
		}
		if partition, ok := value.(int32); ok {
			msg.partition = partition
		}
	}

	value, err = data.Cache.GetValue("topic")
	if err == nil {
		if c.log.IsDebug() {
			c.log.Debugf("got event.Meta[\"topic\"] = %v", value)
		}
		if topic, ok := value.(string); ok {
			msg.topic = topic
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
		if _, err := data.Cache.Put("topic", topic); err != nil {
			return nil, fmt.Errorf("setting kafka topic in publisher event failed: %v", err)
		}
	}

	serializedEvent, err := c.codec.Encode(c.index, event)
	if err != nil {
		if c.log.IsDebug() {
			c.log.Debugf("failed event: %v", event)
		}
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
	defer c.log.Debug("Stop kafka ack worker")

	for libMsg := range ch {
		msg := libMsg.Metadata.(*message)
		msg.ref.done()
	}
}

func (c *client) errorWorker(ch <-chan *sarama.ProducerError) {
	defer c.wg.Done()
	defer c.log.Debug("Stop kafka error handler")

	for errMsg := range ch {
		msg := errMsg.Msg.Metadata.(*message)
		msg.ref.fail(msg, errMsg.Err)
	}
}

func (r *msgRef) done() {
	r.dec()
}

func (r *msgRef) fail(msg *message, err error) {
	switch err {
	case sarama.ErrInvalidMessage:
		r.client.log.Errorf("Kafka (topic=%v): dropping invalid message", msg.topic)
		r.client.observer.Dropped(1)

	case sarama.ErrMessageSizeTooLarge, sarama.ErrInvalidMessageSize:
		r.client.log.Errorf("Kafka (topic=%v): dropping too large message of size %v.",
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
	if i > 0 {
		return
	}

	r.client.log.Debug("finished kafka batch")
	stats := r.client.observer

	err := r.err
	if err != nil {
		failed := len(r.failed)
		success := r.total - failed
		r.batch.RetryEvents(r.failed)

		stats.Failed(failed)
		if success > 0 {
			stats.Acked(success)
		}

		r.client.log.Debugf("Kafka publish failed with: %+v", err)
	} else {
		r.batch.ACK()
		stats.Acked(r.total)
	}
}

func (c *client) Test(d testing.Driver) {
	if c.config.Net.TLS.Enable == true {
		d.Warn("TLS", "Kafka output doesn't support TLS testing")
	}

	for _, host := range c.hosts {
		d.Run("Kafka: "+host, func(d testing.Driver) {
			netDialer := transport.TestNetDialer(d, c.config.Net.DialTimeout)
			_, err := netDialer.Dial("tcp", host)
			d.Error("dial up", err)
		})
	}

}
