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

package pulsar

import (
	"context"
	"fmt"
	"sync"

	"github.com/apache/pulsar-client-go/pulsar"

	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/codec"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/elastic-agent-libs/logp"
)

var (
	_ outputs.NetworkClient = (*client)(nil)
)

type client struct {
	config   *pulsarConfig
	client   pulsar.Client
	producer pulsar.Producer
	codec    codec.Codec
	index    string
	observer outputs.Observer
	log      *logp.Logger
	mutex    sync.Mutex
	wg       sync.WaitGroup
}

// Connect creates a new pulsar client and producer.
func (c *client) Connect() error {
	c.mutex.Lock()
	c.wg.Add(1)
	defer c.mutex.Unlock()
	defer c.wg.Done()

	co, err := c.config.parseClientOptions()
	if err != nil {
		c.log.Errorf("Failed to parse client options: %+v", err)
		return err
	}
	client, err := pulsar.NewClient(co)
	if err != nil {
		c.log.Errorf("Failed to create client: %+v", err)
		return err
	}

	c.client = client
	po := c.config.parseProducerOptions()
	producer, err := client.CreateProducer(po)
	if err != nil {
		c.log.Errorf("Failed to create producer: %+v", err)
		return err
	}
	c.producer = producer
	return nil
}

// Close closes the client and producer.
func (c *client) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.wg.Wait()

	var err error = nil
	if c.producer != nil {
		err = c.producer.FlushWithCtx(context.Background())
		c.producer.Close()
	}
	if c.client != nil {
		c.client.Close()
	}
	return err
}

// Publish sends a batch of events to the broker.
func (c *client) Publish(ctx context.Context, batch publisher.Batch) error {
	events := batch.Events()
	if len(events) == 0 {
		return nil
	}

	for i := range events {
		event := &events[i]
		msg, err := c.getPulsarMessage(event)
		if err != nil {
			c.log.Errorf("Dropping event: %+v", err)
			c.observer.Dropped(1)
			continue
		}
		c.producer.SendAsync(ctx, msg, c.handleSendFailed)
	}
	return nil
}

// String returns a string representation of the client.
func (c *client) String() string {
	return fmt.Sprintf("pulsar(%s)", c.config.Endpoint)
}

// getPulsarMessage returns a pulsar message from a beat event.
func (c *client) getPulsarMessage(d *publisher.Event) (*pulsar.ProducerMessage, error) {
	event := &d.Content
	payload, err := c.codec.Encode(c.index, event)
	if err != nil {
		return nil, err
	}

	message := &pulsar.ProducerMessage{
		Payload: payload,
	}
	return message, nil
}

// handleSendFailed is called when a message fails to be sent to the broker.
func (c *client) handleSendFailed(id pulsar.MessageID, _ *pulsar.ProducerMessage, err error) {
	size := int(id.BatchSize())
	if err != nil {
		c.observer.Failed(size)
	} else {
		c.observer.Acked(size)
	}
}
