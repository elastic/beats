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
	"github.com/menderesk/beats/v7/libbeat/beat"
)

type nilPipeline struct{}

type nilClient struct {
	eventer beat.ClientEventer
	acker   beat.ACKer
}

var _nilPipeline = (*nilPipeline)(nil)

// NewNilPipeline returns a new pipeline that is compatible with
// beats.PipelineConnector. The pipeline will discard all events that have been
// published. Client ACK handlers will still be executed, but the callbacks
// will be executed immediately when the event is published.
func NewNilPipeline() beat.PipelineConnector { return _nilPipeline }

func (p *nilPipeline) Connect() (beat.Client, error) {
	return p.ConnectWith(beat.ClientConfig{})
}

func (p *nilPipeline) ConnectWith(cfg beat.ClientConfig) (beat.Client, error) {
	return &nilClient{
		eventer: cfg.Events,
		acker:   cfg.ACKHandler,
	}, nil
}

func (c *nilClient) Publish(event beat.Event) {
	c.PublishAll([]beat.Event{event})
}

func (c *nilClient) PublishAll(events []beat.Event) {
	L := len(events)
	if L == 0 {
		return
	}

	if c.acker != nil {
		for _, event := range events {
			c.acker.AddEvent(event, true)
		}
		c.acker.ACKEvents(len(events))
	}
}

func (c *nilClient) Close() error {
	if c.eventer != nil {
		c.eventer.Closing()
		c.eventer.Closed()
	}
	return nil
}
