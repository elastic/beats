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
	"sync"

	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/libbeat/common/acker"
	"github.com/elastic/beats/v8/libbeat/logp"
	"github.com/elastic/beats/v8/libbeat/publisher/pipetool"
)

// Client implements the interface used by all the functionbeat function, we only implement a synchronous
// client. This interface superseed the core beat.Client interface and can return errors on publish.
type ISyncClient interface {
	// Publish accepts a unique events and will publish it to the pipeline.
	Publish(beat.Event) error

	// PublishAll accepts a list of multiple events and will publish them to the pipeline.
	PublishAll([]beat.Event) error

	// Close closes the current client, no events will be accepted, this method can block if we still
	// need to ACK on events.
	Close() error

	// Wait blocks until the publisher pipeline send the ACKS for all the events.
	Wait()
}

// SyncClient wraps an existing beat.Client and provide a sync interface.
type SyncClient struct {
	// Chain callbacks already defined in the original ClientConfig
	ackCount     func(int)
	ackEvents    func([]interface{})
	ackLastEvent func(interface{})

	client beat.Client
	wg     sync.WaitGroup
	log    *logp.Logger
}

// NewSyncClient creates a new sync clients from the provided configuration, existing ACKs handlers
// defined in the configuration will be proxied by this object.
func NewSyncClient(log *logp.Logger, pipeline beat.Pipeline, cfg beat.ClientConfig) (*SyncClient, error) {
	if log == nil {
		log = logp.NewLogger("")
	}
	s := &SyncClient{log: log.Named("sync client")}

	pipeline = pipetool.WithACKer(pipeline, acker.TrackingCounter(func(_, total int) {
		log.Debugf("ack callback receives with events count of %d", total)
		s.onACK(total)
	}))

	c, err := pipeline.ConnectWith(cfg)
	if err != nil {
		return nil, err
	}

	s.client = c

	return s, nil
}

// Publish publishes one event to the pipeline and return.
func (s *SyncClient) Publish(event beat.Event) error {
	s.log.Debug("Publish 1 event")
	s.wg.Add(1)
	s.client.Publish(event)
	return nil
}

// PublishAll publish a slice of events to the pipeline and return.
func (s *SyncClient) PublishAll(events []beat.Event) error {
	s.log.Debugf("Publish %d events", len(events))
	s.wg.Add(len(events))
	s.client.PublishAll(events)
	return nil
}

// Close closes the wrapped beat.Client.
func (s *SyncClient) Close() error {
	s.wg.Wait()
	return s.client.Close()
}

// Wait waits until we received a ACK for every events that were sent, this is useful in the
// context of serverless, because when the handler return the execution of the process is suspended.
func (s *SyncClient) Wait() {
	s.wg.Wait()
}

func (s *SyncClient) onACK(n int) {
	s.wg.Add(-1 * n)
}
