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

package monitors

import (
	"sync"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/acker"
	"github.com/elastic/beats/v7/libbeat/publisher/pipetool"
	"github.com/elastic/elastic-agent-libs/logp"
)

// Defines a synchronous pipeline wrapper interface
type PipelineWrapper interface {
	Wait()
}

type NoopPipelineWrapper struct {
}

// Noop
func (n *NoopPipelineWrapper) Wait() {
}

// Pipeline wrapper that implements synchronous op. Calling Wait() on this client will block until all
// events passed through this pipeline (and any of the linked clients) are ACKed, safe to use concurrently.
type SyncPipelineWrapper struct {
	wg sync.WaitGroup
}

// Used to wrap every client and track emitted vs acked events.
type wrappedClient struct {
	wg     *sync.WaitGroup
	client beat.Client
}

// returns a new pipeline with the provided SyncPipelineClientWrapper.
func WithSyncPipelineWrapper(pipeline beat.Pipeline, pw *SyncPipelineWrapper) beat.Pipeline {
	pipeline = pipetool.WithACKer(pipeline, acker.TrackingCounter(func(_, total int) {
		logp.L().Debugf("ack callback receives with events count of %d", total)
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

// waits until ACK is received for every event that was sent
func (s *SyncPipelineWrapper) Wait() {
	s.wg.Wait()
}

func (s *SyncPipelineWrapper) onACK(n int) {
	s.wg.Add(-1 * n)
}
