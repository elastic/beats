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

package testing

import (
	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/common/atomic"
)

// ClientCounter can be used to create a beat.PipelineConnector that count
// pipeline connects and disconnects.
type ClientCounter struct {
	total  atomic.Int
	active atomic.Int
}

// FakeConnector implements the beat.PipelineConnector interface.
// The ConnectFunc is called for each connection attempt, and must be provided
// by tests that wish to use FakeConnector. If ConnectFunc is nil tests will panic
// if there is a connection attempt.
type FakeConnector struct {
	ConnectFunc func(beat.ClientConfig) (beat.Client, error)
}

// FakeClient implements the beat.Client interface. The implementation of a
// custom PublishFunc and CloseFunc are optional.
type FakeClient struct {
	// If set PublishFunc is called for each event that is published by a producer.
	PublishFunc func(beat.Event)

	// If set CloseFunc is called on Close. Otherwise Close returns nil.
	CloseFunc func() error
}

var _ beat.PipelineConnector = FakeConnector{}
var _ beat.Client = (*FakeClient)(nil)

// ConnectWith calls the ConnectFunc with the given configuration.
func (c FakeConnector) ConnectWith(cfg beat.ClientConfig) (beat.Client, error) {
	return c.ConnectFunc(cfg)
}

// Connect calls the ConnectFunc with an empty configuration.
func (c FakeConnector) Connect() (beat.Client, error) {
	return c.ConnectWith(beat.ClientConfig{})
}

// Publish calls PublishFunc, if PublishFunc is not nil.
func (c *FakeClient) Publish(event beat.Event) {
	if c.PublishFunc != nil {
		c.PublishFunc(event)
	}
}

// Close calls CloseFunc, if CloseFunc is not nil. Otherwise it returns nil.
func (c *FakeClient) Close() error {
	if c.CloseFunc == nil {
		return nil
	}
	return c.CloseFunc()
}

// PublishAll calls PublishFunc for each event in the given slice.
func (c *FakeClient) PublishAll(events []beat.Event) {
	for _, event := range events {
		c.Publish(event)
	}
}

// FailingConnector creates a pipeline that will always fail with the
// configured error value.
func FailingConnector(err error) beat.PipelineConnector {
	return &FakeConnector{
		ConnectFunc: func(_ beat.ClientConfig) (beat.Client, error) {
			return nil, err
		},
	}
}

// ConstClient returns a pipeline that always returns the pre-configured beat.Client instance.
func ConstClient(client beat.Client) beat.PipelineConnector {
	return &FakeConnector{
		ConnectFunc: func(_ beat.ClientConfig) (beat.Client, error) {
			return client, nil
		},
	}
}

// ChClient create a beat.Client that will forward all events to the given channel.
func ChClient(ch chan beat.Event) beat.Client {
	return &FakeClient{
		PublishFunc: func(event beat.Event) {
			ch <- event
		},
	}
}

// Active returns the number of currently active connections.
func (c *ClientCounter) Active() int { return c.active.Load() }

// Total returns the total number of calls to Connect.
func (c *ClientCounter) Total() int { return c.total.Load() }

// BuildConnector create a pipeline that updates the active and tocal
// connection counters on Connect and Close calls.
func (c *ClientCounter) BuildConnector() beat.PipelineConnector {
	return FakeConnector{
		ConnectFunc: func(_ beat.ClientConfig) (beat.Client, error) {
			c.total.Inc()
			c.active.Inc()
			return &FakeClient{
				CloseFunc: func() error {
					c.active.Dec()
					return nil
				},
			}, nil
		},
	}
}
