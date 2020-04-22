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
	"math"
	"math/rand"
	"sync"
	"testing"
	"testing/quick"
	"time"

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

func TestMakeClientWorker(t *testing.T) {
	tests := map[string]func(mockPublishFn) outputs.Client{
		"client":         newMockClient,
		"network_client": newMockNetworkClient,
	}

	for name, ctor := range tests {
		t.Run(name, func(t *testing.T) {
			seedPRNG(t)

			err := quick.Check(func(i uint) bool {
				numBatches := 300 + (i % 100) // between 300 and 399

				var published atomic.Uint
				publishFn := func(batch publisher.Batch) error {
					published.Add(uint(len(batch.Events())))
					return nil
				}

				wqu := makeWorkQueue()
				client := ctor(publishFn)
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
					return numEvents == published
				})
			}, nil)

			if err != nil {
				t.Error(err)
			}
		})
	}
}

func TestMakeClientWorkerAndClose(t *testing.T) {
	tests := map[string]func(mockPublishFn) outputs.Client{
		"client":         newMockClient,
		"network_client": newMockNetworkClient,
	}

	const minEventsInBatch = 50

	for name, ctor := range tests {
		t.Run(name, func(t *testing.T) {
			seedPRNG(t)

			err := quick.Check(func(i uint) bool {
				numBatches := 1000 + (i % 100) // between 1000 and 1099

				wqu := makeWorkQueue()
				numEvents := atomic.MakeUint(0)

				var wg sync.WaitGroup
				wg.Add(1)
				go func() {
					defer wg.Done()
					for batchIdx := uint(0); batchIdx <= numBatches; batchIdx++ {
						batch := randomBatch(minEventsInBatch, 150, wqu)
						numEvents.Add(uint(len(batch.Events())))
						wqu <- batch
					}
				}()

				// Publish at least 1 batch worth of events but no more than 20% events
				publishLimit := uint(math.Max(minEventsInBatch, float64(numEvents.Load())*0.2))

				var publishedFirst atomic.Uint
				blockCtrl := make(chan struct{})
				blockingPublishFn := func(batch publisher.Batch) error {
					// Emulate blocking. Upon unblocking the in-flight batch that was
					// blocked is published.
					if publishedFirst.Load() >= publishLimit {
						<-blockCtrl
					}

					publishedFirst.Add(uint(len(batch.Events())))
					return nil
				}

				client := ctor(blockingPublishFn)
				worker := makeClientWorker(nilObserver, wqu, client)

				// Allow the worker to make *some* progress before we close it
				timeout := 10 * time.Second
				progress := waitUntilTrue(timeout, func() bool {
					return publishedFirst.Load() >= publishLimit
				})
				if !progress {
					return false
				}

				// Close worker before all batches have had time to be published
				err := worker.Close()
				require.NoError(t, err)
				close(blockCtrl)

				// Start new worker to drain work queue
				var publishedLater atomic.Uint
				countingPublishFn := func(batch publisher.Batch) error {
					publishedLater.Add(uint(len(batch.Events())))
					return nil
				}

				client = ctor(countingPublishFn)
				makeClientWorker(nilObserver, wqu, client)
				wg.Wait()

				// Make sure that all events have eventually been published
				timeout = 20 * time.Second
				return waitUntilTrue(timeout, func() bool {
					return numEvents.Load() == publishedFirst.Load()+publishedLater.Load()
				})
			}, &quick.Config{MaxCount: 25})

			if err != nil {
				t.Error(err)
			}
		})
	}
}

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
		time.Sleep(1 * time.Millisecond)
	}
	return false
}
