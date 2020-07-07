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
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
)

type mockPublishFn func(publisher.Batch) error

func newMockClient(publishFn mockPublishFn) outputs.Client {
	return &mockClient{publishFn: publishFn}
}

type mockClient struct {
	publishFn mockPublishFn
}

func (c *mockClient) String() string { return "mock_client" }
func (c *mockClient) Close() error   { return nil }
func (c *mockClient) Publish(_ context.Context, batch publisher.Batch) error {
	return c.publishFn(batch)
}

func newMockNetworkClient(publishFn mockPublishFn) outputs.Client {
	return &mockNetworkClient{newMockClient(publishFn)}
}

type mockNetworkClient struct {
	outputs.Client
}

func (c *mockNetworkClient) Connect() error { return nil }

type mockQueue struct{}

func (q mockQueue) Close() error                                     { return nil }
func (q mockQueue) BufferConfig() queue.BufferConfig                 { return queue.BufferConfig{} }
func (q mockQueue) Producer(cfg queue.ProducerConfig) queue.Producer { return mockProducer{} }
func (q mockQueue) Consumer() queue.Consumer                         { return mockConsumer{} }

type mockProducer struct{}

func (p mockProducer) Publish(event publisher.Event) bool    { return true }
func (p mockProducer) TryPublish(event publisher.Event) bool { return true }
func (p mockProducer) Cancel() int                           { return 0 }

type mockConsumer struct{}

func (c mockConsumer) Get(eventCount int) (queue.Batch, error) { return &batch{}, nil }
func (c mockConsumer) Close() error                            { return nil }

type mockBatch struct {
	mu     sync.Mutex
	events []publisher.Event

	onEvents    func()
	onACK       func()
	onDrop      func()
	onRetry     func()
	onCancelled func()
	onReduceTTL func() bool
}

func (b *mockBatch) Events() []publisher.Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	signalFn(b.onEvents)
	return b.events
}

func (b *mockBatch) ACK()       { signalFn(b.onACK) }
func (b *mockBatch) Drop()      { signalFn(b.onDrop) }
func (b *mockBatch) Retry()     { signalFn(b.onRetry) }
func (b *mockBatch) Cancelled() { signalFn(b.onCancelled) }
func (b *mockBatch) RetryEvents(events []publisher.Event) {
	b.updateEvents(events)
	signalFn(b.onRetry)
}

func (b *mockBatch) reduceTTL() bool {
	if b.onReduceTTL != nil {
		return b.onReduceTTL()
	}
	return true
}

func (b *mockBatch) CancelledEvents(events []publisher.Event) {
	b.updateEvents(events)
	signalFn(b.onCancelled)
}

func (b *mockBatch) updateEvents(events []publisher.Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = events
}

func (b *mockBatch) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.events)
}

func (b *mockBatch) withRetryer(r *retryer) *mockBatch {
	return &mockBatch{
		events:      b.events,
		onACK:       b.onACK,
		onDrop:      b.onDrop,
		onRetry:     func() { r.retry(b) },
		onCancelled: func() { r.cancelled(b) },
		onReduceTTL: b.onReduceTTL,
	}
}

func signalFn(fn func()) {
	if fn != nil {
		fn()
	}
}

func randomBatch(min, max int) *mockBatch {
	return &mockBatch{
		events: make([]publisher.Event, randIntBetween(min, max)),
	}
}

// randIntBetween returns a random integer in [min, max)
func randIntBetween(min, max int) int {
	return rand.Intn(max-min) + min
}

func waitUntilTrue(duration time.Duration, fn func() bool) bool {
	end := time.Now().Add(duration)
	for time.Now().Before(end) {
		if fn() {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}
