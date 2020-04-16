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
	"math/rand"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/logp"
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
func (c *mockClient) Publish(batch publisher.Batch) error {
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

func (c mockConsumer) Get(eventCount int) (queue.Batch, error) { return &Batch{}, nil }
func (c mockConsumer) Close() error                            { return nil }

func randomBatch(min, max int, wqu workQueue) *Batch {
	numEvents := randIntBetween(min, max)
	events := make([]publisher.Event, numEvents)

	consumer := newEventConsumer(logp.L(), mockQueue{}, &batchContext{})
	retryer := newRetryer(logp.L(), nilObserver, wqu, consumer)

	batch := Batch{
		events: events,
		ctx: &batchContext{
			observer: nilObserver,
			retryer:  retryer,
		},
	}

	return &batch
}

// randIntBetween returns a random integer in [min, max)
func randIntBetween(min, max int) int {
	return rand.Intn(max-min) + min
}

func seedPRNG(t *testing.T) {
	seed := *SeedFlag
	if seed == 0 {
		seed = time.Now().UnixNano()
	}

	t.Logf("reproduce test with `go test ... -seed %v`", seed)
	rand.Seed(seed)
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
