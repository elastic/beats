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

package mock

import (
	"fmt"
	"sync"

	"github.com/elastic/beats/v7/libbeat/beat"
)

// MockClient is a mock implementation of the beat.Client interface.
type MockClient struct {
	published []beat.Event // Slice to store published events

	closed bool       // Flag to indicate if the client is closed
	mu     sync.Mutex // Mutex to synchronize access to the published events slice
}

// GetEvents returns all the events published by the mock client.
func (m *MockClient) GetEvents() []beat.Event {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.published
}

// Publish publishes a single event.
func (m *MockClient) Publish(e beat.Event) {
	es := make([]beat.Event, 1)
	es = append(es, e)

	m.PublishAll(es)
}

// PublishAll publishes multiple events.
func (m *MockClient) PublishAll(es []beat.Event) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.published = append(m.published, es...)
}

// Close closes the mock client.
func (m *MockClient) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return fmt.Errorf("mock already closed")
	}

	m.closed = true
	return nil
}

// MockPipeline is a mock implementation of the beat.Pipeline interface.
type MockPipeline struct {
	c  beat.Client // Client used by the pipeline
	mu sync.Mutex  // Mutex to synchronize access to the client
}

// ConnectWith connects the mock pipeline with a client using the provided configuration.
func (mp *MockPipeline) ConnectWith(config beat.ClientConfig) (beat.Client, error) {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	c := &MockClient{}

	mp.c = c

	return c, nil
}

// Connect connects the mock pipeline with a client using the default configuration.
func (mp *MockPipeline) Connect() (beat.Client, error) {
	return mp.ConnectWith(beat.ClientConfig{})
}
