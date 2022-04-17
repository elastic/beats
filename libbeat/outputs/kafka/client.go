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
	"time"

	"github.com/Shopify/sarama"
	"github.com/eapache/go-resiliency/breaker"

	"github.com/menderesk/beats/v7/libbeat/common/fmtstr"
	"github.com/menderesk/beats/v7/libbeat/common/transport"
	"github.com/menderesk/beats/v7/libbeat/logp"
	"github.com/menderesk/beats/v7/libbeat/outputs"
	"github.com/menderesk/beats/v7/libbeat/outputs/codec"
	"github.com/menderesk/beats/v7/libbeat/outputs/outil"
	"github.com/menderesk/beats/v7/libbeat/publisher"
	"github.com/menderesk/beats/v7/libbeat/testing"
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
	done     chan struct{}

	producer sarama.AsyncProducer

	recordHeaders []sarama.RecordHeader

	wg sync.WaitGroup
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
	headers []header,
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
		done:     make(chan struct{}),
	}

	if len(headers) != 0 {
		recordHeaders := make([]sarama.RecordHeader, 0, len(headers))
		for _, h := range headers {
			if h.Key == "" {
				continue
			}
			recordHeader := sarama.RecordHeader{
				Key:   []byte(h.Key),
				Value: []byte(h.Value),
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

	close(c.done)
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
	breakerOpen := false
	defer c.wg.Done()
	defer c.log.Debug("Stop kafka error handler")

	for errMsg := range ch {
		msg := errMsg.Msg.Metadata.(*message)
		msg.ref.fail(msg, errMsg.Err)

		if errMsg.Err == breaker.ErrBreakerOpen {
			// ErrBreakerOpen is a very special case in Sarama. It happens only when
			// there have been repeated critical (broker / topic-level) errors, and it
			// puts Sarama into a state where it immediately rejects all input
			// for 10 seconds, ignoring retry / backoff settings.
			// With this output's current design (in which Publish passes through to
			// Sarama's input channel with no further synchronization), retrying
			// these failed values causes an infinite retry loop that degrades
			// the entire system.
			// "Nice" approaches and why we haven't used them:
			// - Use exposed API to navigate this state and its effect on retries.
			//   * Unfortunately, Sarama's circuit breaker and its errors are
			//     hard-coded and undocumented. We'd like to address this in the
			//     future.
			// - If a batch fails with a circuit breaker error, delay before
			//   retrying it.
			//   * This would fix the most urgent performance issues, but requires
			//     extra bookkeeping because the Kafka output handles each batch
			//     independently. It results in potentially many batches / 10s of
			//     thousands of events being loaded and attempted, even though we
			//     know there's a fatal error early in the first batch. It also
			//     makes it hard to know when each batch should be retried.
			// - In the Kafka Publish method, add a blocking first-pass intake step
			//   that can gate on error conditions, rather than handing off data
			//   to Sarama immediately.
			//   * This would fix the issue but would require a lot of work and
			//     testing, and we need a fix for the release now. It's also a
			//     fairly elaborate workaround for something that might be
			//     easier to fix in the library itself.
			//
			// Instead, we have applied the following fix, which is not very "nice"
			// but satisfies all other important constraints:
			// - When we receive a circuit breaker error, sleep for 10 seconds
			//   (Sarama's hard-coded timeout) on the _error worker thread_.
			//
			// This works because connection-level errors that can trigger the
			// circuit breaker are on the critical path for input processing, and
			// thus blocking on the error channel applies back-pressure to the
			// input channel. This means that if there are any more errors while the
			// error worker is asleep, any call to Publish will block until we
			// start reading again.
			//
			// Reasons this solution is preferred:
			// - It responds immediately to Sarama's global error state, rather than
			//   trying to detect it independently in each batch or adding more
			//   cumbersome synchronization to the output
			// - It gives the minimal delay that is consistent with Sarama's
			//   internal behavior
			// - It requires only a few lines of code and no design changes
			//
			// That said, this is still relying on undocumented library internals
			// for correct behavior, which isn't ideal, but the error itself is an
			// undocumented library internal, so this is de facto necessary for now.
			// We'd like to have a more official / permanent fix merged into Sarama
			// itself in the future.

			// The "breakerOpen" flag keeps us from sleeping the first time we see
			// a circuit breaker error, because it might be an old error still
			// sitting in the channel from 10 seconds ago. So we only end up
			// sleeping every _other_ reported breaker error.
			if breakerOpen {
				// Immediately log the error that presumably caused this state,
				// since the error reporting on this batch will be delayed.
				if msg.ref.err != nil {
					c.log.Errorf("Kafka (topic=%v): %v", msg.topic, msg.ref.err)
				}
				select {
				case <-time.After(10 * time.Second):
					// Sarama's circuit breaker is hard-coded to reject all inputs
					// for 10sec.
				case <-msg.ref.client.done:
					// Allow early bailout if the output itself is closing.
				}
				breakerOpen = false
			} else {
				breakerOpen = true
			}
		}
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
		r.client.observer.Dropped(1)

	case breaker.ErrBreakerOpen:
		// Add this message to the failed list, but don't overwrite r.err since
		// all the breaker error means is "there were a lot of other errors".
		r.failed = append(r.failed, msg.data)

	default:
		r.failed = append(r.failed, msg.data)
		if r.err == nil {
			// Don't overwrite an existing error. This way at tne end of the batch
			// we report the first error that we saw, rather than the last one.
			r.err = err
		}
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
