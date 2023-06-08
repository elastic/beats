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

package beater

import (
	"sync"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/acker"
	"github.com/elastic/beats/v7/libbeat/publisher/pipetool"
	"github.com/elastic/elastic-agent-libs/logp"
)

type PipelineClientWrapper struct {
	wg  sync.WaitGroup
	log *logp.Logger
}

type wrappedClient struct {
	wg     *sync.WaitGroup
	client beat.Client
}

func withPipelineWrapper(pipeline beat.Pipeline, pw *PipelineClientWrapper) beat.Pipeline {
	pipeline = pipetool.WithACKer(pipeline, acker.TrackingCounter(func(_, total int) {
		pw.log.Debugf("ack callback receives with events count of %d", total)
		pw.onACK(total)
	}))

	pipeline = pipetool.WithClientWrapper(pipeline, func(client beat.Client) beat.Client {
		return &wrappedClient{
			wg:     &pw.wg,
			client: client,
		}
	})

	return pipeline
}

func (c *wrappedClient) Publish(event beat.Event) {
	c.wg.Add(1)
	c.client.Publish(event)
}

func (c *wrappedClient) PublishAll(events []beat.Event) {
	c.wg.Add(len(events))
	c.client.PublishAll(events)
}

func (c *wrappedClient) Close() error {
	return c.client.Close()
}

// Wait waits until we received a ACK for every events that were sent
func (s *PipelineClientWrapper) Wait() {
	s.wg.Wait()
}

func (s *PipelineClientWrapper) onACK(n int) {
	s.wg.Add(-1 * n)
}
