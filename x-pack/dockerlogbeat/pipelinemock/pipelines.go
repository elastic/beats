// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pipelinemock

import (
	"fmt"
	"sync"

	"github.com/elastic/beats/v7/libbeat/beat"
)

// MockBeatClient mocks the Client interface
type MockBeatClient struct {
	publishes []beat.Event
	closed    bool
	mtx       sync.Mutex
}

// GetEvents returns the published events
func (c *MockBeatClient) GetEvents() []beat.Event {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	return c.publishes
}

// Publish mocks the Client Publish method
func (c *MockBeatClient) Publish(e beat.Event) {
	c.PublishAll([]beat.Event{e})
}

// PublishAll mocks the Client PublishAll method
func (c *MockBeatClient) PublishAll(events []beat.Event) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	c.publishes = append(c.publishes, events...)

}

// Close mocks the Client Close method
func (c *MockBeatClient) Close() error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	if c.closed {
		return fmt.Errorf("mock client already closed")
	}

	c.closed = true
	return nil
}

// MockPipelineConnector mocks the PipelineConnector interface
type MockPipelineConnector struct {
	clients []*MockBeatClient
	mtx     sync.Mutex
}

// GetAllEvents returns all events associated with a pipeline
func (pc *MockPipelineConnector) GetAllEvents() []beat.Event {
	var evList []beat.Event
	for _, clientEvents := range pc.clients {
		evList = append(evList, clientEvents.GetEvents()...)
	}

	return evList
}

// Connect mocks the PipelineConnector Connect method
func (pc *MockPipelineConnector) Connect() (beat.Client, error) {
	return pc.ConnectWith(beat.ClientConfig{})
}

// ConnectWith mocks the PipelineConnector ConnectWith method
func (pc *MockPipelineConnector) ConnectWith(beat.ClientConfig) (beat.Client, error) {
	pc.mtx.Lock()
	defer pc.mtx.Unlock()

	c := &MockBeatClient{}

	pc.clients = append(pc.clients, c)

	return c, nil
}

// HasConnectedClients returns true if there are clients connected.
func (pc *MockPipelineConnector) HasConnectedClients() bool {
	pc.mtx.Lock()
	defer pc.mtx.Unlock()

	return len(pc.clients) > 0
}
