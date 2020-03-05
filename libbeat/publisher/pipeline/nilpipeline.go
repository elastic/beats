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

import "github.com/elastic/beats/v7/libbeat/beat"

type NilPipeline struct{}

type nilClient struct {
	eventer      beat.ClientEventer
	ackCount     func(int)
	ackEvents    func([]interface{})
	ackLastEvent func(interface{})
}

var _nilPipeline = (*NilPipeline)(nil)

func NewNilPipeline() *NilPipeline { return _nilPipeline }

func (p *NilPipeline) Connect() (beat.Client, error) {
	return p.ConnectWith(beat.ClientConfig{})
}

func (p *NilPipeline) ConnectWith(cfg beat.ClientConfig) (beat.Client, error) {
	return &nilClient{
		eventer:      cfg.Events,
		ackCount:     cfg.ACKCount,
		ackEvents:    cfg.ACKEvents,
		ackLastEvent: cfg.ACKLastEvent,
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

	if c.ackLastEvent != nil {
		c.ackLastEvent(events[L-1].Private)
	}
	if c.ackEvents != nil {
		tmp := make([]interface{}, L)
		for i := range events {
			tmp[i] = events[i].Private
		}
		c.ackEvents(tmp)
	}
	if c.ackCount != nil {
		c.ackCount(L)
	}
}

func (c *nilClient) Close() error {
	if c.eventer != nil {
		c.eventer.Closing()
		c.eventer.Closed()
	}
	return nil
}
