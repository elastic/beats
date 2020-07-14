// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package core

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
