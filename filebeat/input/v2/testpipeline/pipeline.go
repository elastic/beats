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

package testpipeline

import (
	"sync/atomic"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
)

// PipelineConnector is a mock for beat.PipelineConnector
type PipelineConnector struct {
	errorOnConnect error
	count          atomic.Uint64
	blocked        atomic.Bool
	clients        []beat.Client
}

// NewPipelineConnector returns a PipelineConnector that always succeeds
func NewPipelineConnector() *PipelineConnector {
	return &PipelineConnector{}
}

// NewPipelineConnectorWithError returns a PipelineConnector that always
// fails with err
func NewPipelineConnectorWithError(err error) *PipelineConnector {
	return &PipelineConnector{
		errorOnConnect: err,
	}
}

// ConnectWith returns a client that publishes to this pipeline,
// the client config is ignored. If NewPipelineConnectorWithError was used to
// to create this connector, then a nil client and the error are returned
func (p *PipelineConnector) ConnectWith(beat.ClientConfig) (beat.Client, error) {
	return p.Connect()
}

// Connect returns a client that publishes to this pipeline.
// If NewPipelineConnectorWithError was used to
// to create this connector, then a nil client and the error are returned
func (p *PipelineConnector) Connect() (beat.Client, error) {
	if p.errorOnConnect != nil {
		return nil, p.errorOnConnect
	}

	client := &Client{
		publisher: p.publish,
	}
	p.clients = append(p.clients, client)

	return client, nil
}

// NumClients returns the number of clients created by this
// PipelineConnector.
func (p *PipelineConnector) NumClients() int {
	return len(p.clients)
}

// Blocks the pipeline, once called all Publish calls in the returned clients
// will block until Unblock is called.
func (p *PipelineConnector) Block() {
	p.blocked.Store(true)
}

// Unblock unblocks the pipeline
func (p *PipelineConnector) Unblock() {
	p.blocked.Store(false)
}

// publish "publishes" the event. All clients must call this method
// to publish an event. It increments the published count and will block
// before updating the count if the pipeline is blocked.
func (p *PipelineConnector) publish(evt beat.Event) {
	for p.blocked.Load() {
		time.Sleep(10 * time.Millisecond)
	}

	p.count.Add(1)
}

// EventsPublished returns the number of published events
func (p *PipelineConnector) EventsPublished() uint64 {
	return p.count.Load()
}

// Client is a mock for beat.Client
type Client struct {
	publisher func(beat.Event)
	closed    bool
}

// Publish calls the publisher function on the event
func (c *Client) Publish(evt beat.Event) {
	c.publisher(evt)
}

// PublishAll calls the publisher function on each event
func (c *Client) PublishAll(events []beat.Event) {
	for _, evt := range events {
		c.Publish(evt)
	}
}

// Close sets the closed flag to true
func (c *Client) Close() error {
	c.closed = true
	return nil
}

// Closed returns true if the Client's Close method
// has been called at lest once
func (c *Client) Closed() bool {
	return c.closed
}
