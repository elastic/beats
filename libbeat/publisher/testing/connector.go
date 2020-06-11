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

import "github.com/elastic/beats/v7/libbeat/beat"

type FakeConnector struct {
	ConnectFunc func(beat.ClientConfig) (beat.Client, error)
}

type FakeClient struct {
	PublishFunc func(beat.Event)
	CloseFunc   func() error
}

var _ beat.PipelineConnector = FakeConnector{}
var _ beat.Client = (*FakeClient)(nil)

func (c FakeConnector) ConnectWith(cfg beat.ClientConfig) (beat.Client, error) {
	return c.ConnectFunc(cfg)
}

func (c FakeConnector) Connect() (beat.Client, error) {
	return c.ConnectWith(beat.ClientConfig{})
}

func (c *FakeClient) Publish(event beat.Event) {
	if c.PublishFunc != nil {
		c.PublishFunc(event)
	}
}

func (c *FakeClient) Close() error {
	if c.CloseFunc == nil {
		return nil
	}
	return c.CloseFunc()
}

func (c *FakeClient) PublishAll(events []beat.Event) {
	for _, event := range events {
		c.Publish(event)
	}
}
