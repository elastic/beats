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

package pipeline

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
)

type dummyClient struct {
	Received chan int
}

func newDummyClient() *dummyClient {
	return &dummyClient{Received: make(chan int)}
}

func (c *dummyClient) Publish(event beat.Event) {
	c.Received <- 1
}

func (c *dummyClient) PublishAll(events []beat.Event) {
	c.Received <- len(events)
}

func (c *dummyClient) Close() error {
	close(c.Received)
	return nil
}

type dummyPipeline struct {
	client beat.Client
}

func newDummyPipeline(client beat.Client) *dummyPipeline {
	return &dummyPipeline{client: client}
}

func (d *dummyPipeline) Connect() (beat.Client, error) {
	return d.client, nil
}

func (d *dummyPipeline) ConnectWith(cfg beat.ClientConfig) (beat.Client, error) {
	return d.client, nil
}

func TestSyncClient(t *testing.T) {
	receiver := func(c *dummyClient, sc *SyncClient) {
		select {
		case i := <-c.Received:
			sc.onACK(i)
			return
		}
	}

	t.Run("Publish", func(t *testing.T) {
		c := newDummyClient()

		pipeline := newDummyPipeline(c)
		sc, err := NewSyncClient(nil, pipeline, beat.ClientConfig{})
		if !assert.NoError(t, err) {
			return
		}
		defer sc.Close()

		go receiver(c, sc)

		err = sc.Publish(beat.Event{})
		if !assert.NoError(t, err) {
			return
		}
		sc.Wait()
	})

	t.Run("PublishAll single ACK", func(t *testing.T) {
		c := newDummyClient()

		pipeline := newDummyPipeline(c)
		sc, err := NewSyncClient(nil, pipeline, beat.ClientConfig{})
		if !assert.NoError(t, err) {
			return
		}
		defer sc.Close()

		go receiver(c, sc)

		err = sc.PublishAll(make([]beat.Event, 10))
		if !assert.NoError(t, err) {
			return
		}
		sc.Wait()
	})

	t.Run("PublishAll multiple independant ACKs", func(t *testing.T) {
		c := newDummyClient()

		pipeline := newDummyPipeline(c)
		sc, err := NewSyncClient(nil, pipeline, beat.ClientConfig{})
		if !assert.NoError(t, err) {
			return
		}
		defer sc.Close()

		go func(c *dummyClient, sc *SyncClient) {
			select {
			case <-c.Received:
				// simulate multiple acks
				sc.onACK(5)
				sc.onACK(5)
				return
			}
		}(c, sc)

		err = sc.PublishAll(make([]beat.Event, 10))
		if !assert.NoError(t, err) {
			return
		}
		sc.Wait()
	})
}
