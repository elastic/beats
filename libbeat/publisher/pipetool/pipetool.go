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

package pipetool

import "github.com/elastic/beats/v7/libbeat/beat"

// connectEditPipeline modifies the client configuration using edit before calling
// edit.
type connectEditPipeline struct {
	parent beat.PipelineConnector
	edit   ConfigEditor
}

// ConfigEditor modifies the client configuration before connecting to a Pipeline.
type ConfigEditor func(beat.ClientConfig) (beat.ClientConfig, error)

func (p *connectEditPipeline) Connect() (beat.Client, error) {
	return p.ConnectWith(beat.ClientConfig{})
}

func (p *connectEditPipeline) ConnectWith(cfg beat.ClientConfig) (beat.Client, error) {
	cfg, err := p.edit(cfg)
	if err != nil {
		return nil, err
	}
	return p.parent.ConnectWith(cfg)
}

// wrapClientPipeline applies edit to the beat.Client returned by Connect and ConnectWith.
// The edit function can wrap the client to add additional functionality to clients
// that connect to the pipeline.
type wrapClientPipeline struct {
	parent  beat.PipelineConnector
	wrapper ClientWrapper
}

// ClientWrapper allows client instances to be wrapped.
type ClientWrapper func(beat.Client) beat.Client

func (p *wrapClientPipeline) Connect() (beat.Client, error) {
	return p.ConnectWith(beat.ClientConfig{})
}

func (p *wrapClientPipeline) ConnectWith(cfg beat.ClientConfig) (beat.Client, error) {
	client, err := p.parent.ConnectWith(cfg)
	if err == nil {
		client = p.wrapper(client)
	}
	return client, err
}

// countingClient adds and substracts from a counter when events have been
// published, dropped or ACKed. The countingClient can be used to keep track of
// inflight events for a beat.Client instance. The counter is updated after the
// client has been disconnected from the publisher pipeline via 'Closed'.
type countingClient struct {
	counter eventCounter
	client  beat.Client
}

type eventCounter interface {
	Add(n int)
	Done()
}

type countingEventer struct {
	wgEvents eventCounter
}

type combinedEventer struct {
	a, b beat.ClientEventer
}

func (c *countingClient) Publish(event beat.Event) {
	c.counter.Add(1)
	c.client.Publish(event)
}

func (c *countingClient) PublishAll(events []beat.Event) {
	c.counter.Add(len(events))
	c.client.PublishAll(events)
}

func (c *countingClient) Close() error {
	return c.client.Close()
}

func (*countingEventer) Closing()   {}
func (*countingEventer) Closed()    {}
func (*countingEventer) Published() {}

func (c *countingEventer) FilteredOut(_ beat.Event) {}
func (c *countingEventer) DroppedOnPublish(_ beat.Event) {
	c.wgEvents.Done()
}

func (c *combinedEventer) Closing() {
	c.a.Closing()
	c.b.Closing()
}

func (c *combinedEventer) Closed() {
	c.a.Closed()
	c.b.Closed()
}

func (c *combinedEventer) Published() {
	c.a.Published()
	c.b.Published()
}

func (c *combinedEventer) FilteredOut(event beat.Event) {
	c.a.FilteredOut(event)
	c.b.FilteredOut(event)
}

func (c *combinedEventer) DroppedOnPublish(event beat.Event) {
	c.a.DroppedOnPublish(event)
	c.b.DroppedOnPublish(event)
}

// WithClientConfigEdit creates a pipeline connector, that allows the
// beat.ClientConfig to be modified before connecting to the underlying
// pipeline.
// The edit function is applied before calling Connect or ConnectWith.
func WithClientConfigEdit(pipeline beat.PipelineConnector, edit ConfigEditor) beat.PipelineConnector {
	return &connectEditPipeline{parent: pipeline, edit: edit}
}

// WithDefaultGuarantee sets the default sending guarantee to `mode` if the
// beat.ClientConfig does not set the mode explicitly.
func WithDefaultGuarantees(pipeline beat.PipelineConnector, mode beat.PublishMode) beat.PipelineConnector {
	return WithClientConfigEdit(pipeline, func(cfg beat.ClientConfig) (beat.ClientConfig, error) {
		if cfg.PublishMode == beat.DefaultGuarantees {
			cfg.PublishMode = mode
		}
		return cfg, nil
	})
}

// WithClientWrapper calls wrap on beat.Client instance, after a successful
// call to `pipeline.Connect` or `pipeline.ConnectWith`. The wrap function can
// wrap the client to provide additional functionality.
func WithClientWrapper(pipeline beat.PipelineConnector, wrap ClientWrapper) beat.PipelineConnector {
	return &wrapClientPipeline{parent: pipeline, wrapper: wrap}
}

// WithPipelineEventCounter adds a counter to the pipeline that keeps track of
// all events published, dropped and ACKed by any active client.
// The type accepted by counter is compatible with sync.WaitGroup.
func WithPipelineEventCounter(pipeline beat.PipelineConnector, counter eventCounter) beat.PipelineConnector {
	counterListener := &countingEventer{counter}

	pipeline = WithClientConfigEdit(pipeline, func(config beat.ClientConfig) (beat.ClientConfig, error) {
		if evts := config.Events; evts != nil {
			config.Events = &combinedEventer{evts, counterListener}
		} else {
			config.Events = counterListener
		}
		return config, nil
	})

	pipeline = WithClientWrapper(pipeline, func(client beat.Client) beat.Client {
		return &countingClient{
			counter: counter,
			client:  client,
		}
	})
	return pipeline
}
