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
	"sync"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/atomic"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/publisher/queue"

	"github.com/elastic/beats/v7/libbeat/logp"

	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/publisher"
)

func TestPublish(t *testing.T) {
	tests := map[string]struct {
		client outputs.Client
	}{
		"client": {
			&mockClient{},
		},
		"network_client": {
			&mockNetworkClient{},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			rand.Seed(time.Now().UnixNano())

			wqu := makeWorkQueue()
			makeClientWorker(nilObserver, wqu, test.client)

			numEvents := atomic.MakeInt(0)
			var wg sync.WaitGroup
			for batchIdx := 0; batchIdx <= randIntBetween(25, 200); batchIdx++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					batch := randomBatch(50, 150, wqu)

					numEvents.Add(len(batch.Events()))

					wqu <- batch
				}()
			}
			wg.Wait()

			// Give some time for events to be published
			time.Sleep(time.Duration(numEvents.Load()*2) * time.Microsecond)

			c := test.client.(interface{ Published() int })
			assert.Equal(t, numEvents.Load(), c.Published())
		})
	}
}

type mockClient struct{ published int }

func (c *mockClient) Published() int { return c.published }

func (c *mockClient) String() string { return "mock_client" }
func (c *mockClient) Close() error   { return nil }
func (c *mockClient) Publish(batch publisher.Batch) error {
	c.published += len(batch.Events())
	return nil
}

type mockNetworkClient struct{ published int }

func (c *mockNetworkClient) Published() int { return c.published }

func (c *mockNetworkClient) String() string { return "mock_network_client" }
func (c *mockNetworkClient) Close() error   { return nil }
func (c *mockNetworkClient) Publish(batch publisher.Batch) error {
	c.published += len(batch.Events())
	return nil
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
