// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package framework

import (
	"fmt"
	"sync"

	"github.com/elastic/beats/v7/libbeat/beat"
)

type mockClient struct {
	publishLog   []*beat.Event
	pipeline     beat.Pipeline
	closed       bool
	mtx          sync.Mutex
	clientConfig beat.ClientConfig
}

func (c *mockClient) IsClosed() bool {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	return c.closed
}

func (c *mockClient) Publish(e beat.Event) {
	if c.clientConfig.Processing.Processor != nil {
		outE, _ := c.clientConfig.Processing.Processor.Run(&e)
		e = *outE
	}
	c.PublishAll([]beat.Event{e})
}

func (c *mockClient) PublishAll(events []beat.Event) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	for _, e := range events {
		eLocal := e
		c.publishLog = append(c.publishLog, &eLocal)
	}
}

func (c *mockClient) Wait() {
}

func (c *mockClient) Close() error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	if c.closed {
		return fmt.Errorf("mock client already closed")
	}

	c.closed = true
	return nil
}

func (c *mockClient) PublishedEvents() []*beat.Event {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	return c.publishLog
}

type mockPipeline struct {
	Clients []*mockClient
	mtx     sync.Mutex
}

func (pc *mockPipeline) Connect() (beat.Client, error) {
	return pc.ConnectWith(beat.ClientConfig{})
}

func (pc *mockPipeline) ConnectWith(cc beat.ClientConfig) (beat.Client, error) {
	pc.mtx.Lock()
	defer pc.mtx.Unlock()

	c := &mockClient{pipeline: pc, clientConfig: cc}

	pc.Clients = append(pc.Clients, c)

	return c, nil
}

func (pc *mockPipeline) PublishedEvents() []*beat.Event {
	pc.mtx.Lock()
	defer pc.mtx.Unlock()

	var events []*beat.Event
	for _, c := range pc.Clients {
		events = append(events, c.PublishedEvents()...)
	}

	return events
}
