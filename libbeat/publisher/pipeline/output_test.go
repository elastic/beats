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
	"fmt"
	"math"
	"sync"
	"testing"
	"testing/quick"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/publisher"
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
				numEvents := atomic.MakeUint(0)

				wqu := makeWorkQueue()
				retryer := newRetryer(logp.NewLogger("test"), nilObserver, wqu, nil)
				defer retryer.close()

				var published atomic.Uint
				var batchCount int
				publishFn := func(batch publisher.Batch) error {
					batchCount++
					published.Add(uint(len(batch.Events())))
					fmt.Printf("received batch %p of size: %d -> events: %d/%d, batches: %d/%d\n",
						batch, len(batch.Events()),
						published.Load(), numEvents.Load(),
						batchCount, numBatches,
					)
					return nil
				}

				client := ctor(publishFn)

				worker := makeClientWorker(nilObserver, wqu, client)
				defer func() {
					fmt.Println("shut down client worker")
					worker.Close()
				}()

				for i := uint(0); i < numBatches; i++ {
					batch := randomBatch(50, 150).withRetryer(retryer)
					numEvents.Add(uint(len(batch.Events())))
					wqu <- batch
					fmt.Printf("send batch %p of size: %d\n", batch, len(batch.Events()))
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

func TestReplaceClientWorker(t *testing.T) {
	tests := map[string]func(mockPublishFn) outputs.Client{
		"client":         newMockClient,
		"network_client": newMockNetworkClient,
	}

	const minEventsInBatch = 50
	const maxEventsInBatch = 150

	for name, ctor := range tests {
		t.Run(name, func(t *testing.T) {
			seedPRNG(t)

			err := quick.Check(func(i uint) bool {
				numBatches := 10000 + (i % 100) // between 1000 and 1099

				fmt.Printf("Starting test with numBatch: %d\n", numBatches)
				defer fmt.Printf("Finished test with numBatch: %d\n", numBatches)

				wqu := makeWorkQueue()
				retryer := newRetryer(logp.NewLogger("test"), nilObserver, wqu, nil)
				defer retryer.close()

				var batches []publisher.Batch
				var numEvents int
				for i := uint(0); i < numBatches; i++ {
					batch := randomBatch(minEventsInBatch, maxEventsInBatch).withRetryer(retryer)
					numEvents += batch.Len()
					batches = append(batches, batch)
				}

				var wg sync.WaitGroup
				wg.Add(1)
				go func() {
					fmt.Println("start forwarding batches")
					defer fmt.Println("stopped forwarding batches")

					defer wg.Done()
					for _, batch := range batches {
						wqu <- batch
						// fmt.Printf("send batch %p of size: %d\n", batch, len(batch.Events()))
					}
				}()

				// Publish at least 1 batch worth of events but no more than 20% events
				publishLimit := uint(math.Max(minEventsInBatch, float64(numEvents)*0.2))

				var publishedFirst atomic.Uint
				blockCtrl := make(chan struct{})
				var batchCount int
				blockingPublishFn := func(batch publisher.Batch) error {
					batchCount++
					// fmt.Printf("(blocking) received batch %p of size: %d -> events: %d/%d, batches: %d/%d\n",
					// 	batch, len(batch.Events()),
					// 	publishedFirst.Load(), numEvents,
					// 	batchCount, numBatches,
					// )

					// Emulate blocking. Upon unblocking the in-flight batch that was
					// blocked is published.
					if publishedFirst.Load() >= publishLimit {
						// fmt.Printf("(blocking) blocking processing waiting for signal. count=%v, limit=%v\n",
						// 	publishedFirst.Load(),
						// 	publishLimit,
						// )
						<-blockCtrl
					}

					publishedFirst.Add(uint(len(batch.Events())))
					// if publishedFirst.Load() >= publishLimit {
					// 	time.Sleep(500 * time.Millisecond)
					// }
					return nil
				}

				client := ctor(blockingPublishFn)
				worker := makeClientWorker(nilObserver, wqu, client)

				// Allow the worker to make *some* progress before we close it
				timeout := 10 * time.Second
				progress := waitUntilTrue(timeout, func() bool {
					// fmt.Printf("waiting progress: count=%d, limit=%d\n",
					// 	publishedFirst.Load(),
					// 	publishLimit,
					// )
					return publishedFirst.Load() >= publishLimit
				})
				if !progress {
					return false
				}
				fmt.Println("progress detected")

				// Close worker before all batches have had time to be published
				fmt.Println("closing worker")
				err := worker.Close()
				require.NoError(t, err)
				fmt.Println("worker closed")

				fmt.Println("unblock output")
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
					return numEvents == int(publishedFirst.Load()+publishedLater.Load())
				})
			}, &quick.Config{MaxCount: 25})

			if err != nil {
				t.Error(err)
			}
		})
	}
}
