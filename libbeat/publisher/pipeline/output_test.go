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
	"flag"
	"math/rand"
	"sync"
	"testing"
	"testing/quick"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
)

var (
	SeedFlag = flag.Int64("seed", 0, "Randomization seed")
)

func TestPublish(t *testing.T) {
	tests := map[string]func(uint) publishCountable{
		"client":         newMockClient,
		"network_client": newMockNetworkClient,
	}

	for name, ctor := range tests {
		t.Run(name, func(t *testing.T) {
			seedPRNG(t)

			err := quick.Check(func(i uint) bool {
				numBatches := 3000 + (i % 1000) // between 3000 and 3999

				wqu := makeWorkQueue()
				client := ctor(0)
				makeClientWorker(nilObserver, wqu, client)

				numEvents := atomic.MakeUint(0)
				for batchIdx := uint(0); batchIdx <= numBatches; batchIdx++ {
					batch := randomBatch(50, 150, wqu)
					numEvents.Add(uint(len(batch.Events())))
					wqu <- batch
				}

				// Give some time for events to be published
				timeout := 20 * time.Second

				// Make sure that all events have eventually been published
				return waitUntilTrue(timeout, func() bool {
					return numEvents.Load() == client.Published()
				})
			}, nil)

			if err != nil {
				t.Error(err)
			}
		})
	}
}

func TestPublishWithClose(t *testing.T) {
	tests := map[string]func(uint) publishCountable{
		"client":         newMockClient,
		"network_client": newMockNetworkClient,
	}

	for name, ctor := range tests {
		t.Run(name, func(t *testing.T) {
			seedPRNG(t)

			err := quick.Check(func(i uint) bool {
				numBatches := 3000 + (i % 1000) // between 3000 and 3999

				wqu := makeWorkQueue()
				numEvents := atomic.MakeUint(0)

				var wg sync.WaitGroup
				for batchIdx := uint(0); batchIdx <= numBatches; batchIdx++ {
					wg.Add(1)
					go func() {
						defer wg.Done()

						batch := randomBatch(50, 150, wqu)
						numEvents.Add(uint(len(batch.Events())))
						wqu <- batch
					}()
				}

				client := ctor(numEvents.Load() / 2) // Stop short of publishing all events
				worker := makeClientWorker(nilObserver, wqu, client)

				// Close worker before all batches have had time to be published
				err := worker.Close()
				require.NoError(t, err)

				published := client.Published()
				assert.Less(t, published, numEvents.Load())

				// Start new worker to drain work queue
				client = ctor(0)
				makeClientWorker(nilObserver, wqu, client)
				wg.Wait()

				// Give some time for events to be published
				timeout := 20 * time.Second

				// Make sure that all events have eventually been published
				return waitUntilTrue(timeout, func() bool {
					return numEvents.Load() == client.Published()+published
				})
			}, nil)

			if err != nil {
				t.Error(err)
			}
		})
	}
}

type publishCountable interface {
	outputs.Client
	Published() uint
}

func newMockClient(publishLimit uint) publishCountable {
	return &mockClient{publishLimit: publishLimit}
}

type mockClient struct {
	mu           sync.RWMutex
	publishLimit uint
	published    uint
}

func (c *mockClient) Published() uint {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.published
}

func (c *mockClient) String() string { return "mock_client" }
func (c *mockClient) Close() error   { return nil }
func (c *mockClient) Publish(batch publisher.Batch) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Block publishing
	if c.publishLimit > 0 && c.published >= c.publishLimit {
		time.Sleep(10 * time.Second)
		return nil
	}

	c.published += uint(len(batch.Events()))
	return nil
}

func newMockNetworkClient(publishLimit uint) publishCountable {
	return &mockNetworkClient{newMockClient(publishLimit)}
}

type mockNetworkClient struct {
	publishCountable
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
		time.Sleep(100 * time.Nanosecond)
	}
	return false
}
