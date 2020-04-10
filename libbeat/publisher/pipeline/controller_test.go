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
	"testing"
	"testing/quick"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/beats/v7/libbeat/publisher/queue/memqueue"
	"github.com/elastic/beats/v7/libbeat/tests/resources"

	"github.com/stretchr/testify/require"
)

func TestOutputReload(t *testing.T) {
	tests := map[string]func(mockPublishFn) outputs.Client{
		//"client":         newMockClient,
		"network_client": newMockNetworkClient,
	}

	for name, ctor := range tests {
		t.Run(name, func(t *testing.T) {
			seedPRNG(t)

			goroutines := resources.NewGoroutinesChecker()
			defer goroutines.Check(t)

			err := quick.Check(func(q uint) bool {
				//numEventsToPublish := 500 + (q % 1000) // 500 to 1499
				//numOutputReloads := 5 + (q % 10)       // 5 to 14
				numEventsToPublish := uint(20000)
				numOutputReloads := uint(15)

				queueFactory := func(ackListener queue.ACKListener) (queue.Queue, error) {
					return memqueue.NewQueue(
						logp.L(),
						memqueue.Settings{
							ACKListener: ackListener,
							Events:      int(numEventsToPublish),
						}), nil
				}

				var publishedCount atomic.Uint
				countingPublishFn := func(batch publisher.Batch) error {
					publishedCount.Add(uint(len(batch.Events())))
					fmt.Printf("published so far: %v\n", publishedCount.Load())
					return nil
				}

				pipeline, err := New(
					beat.Info{},
					Monitors{},
					queueFactory,
					outputs.Group{},
					Settings{},
				)
				require.NoError(t, err)
				defer pipeline.Close()

				pipelineClient, err := pipeline.Connect()
				require.NoError(t, err)
				defer pipelineClient.Close()

				go func() {
					for i := uint(0); i < numEventsToPublish; i++ {
						pipelineClient.Publish(beat.Event{})
					}
				}()

				for i := uint(0); i < numOutputReloads; i++ {
					outputClient := ctor(countingPublishFn)
					out := outputs.Group{
						Clients: []outputs.Client{outputClient},
					}
					fmt.Println("reloading output...")
					pipeline.output.Set(out)
				}

				timeout := 5 * time.Second
				success := waitUntilTrue(timeout, func() bool {
					return uint(numEventsToPublish) == publishedCount.Load()
				})
				if !success {
					fmt.Printf(
						"numOutputReloads = %v, numEventsToPublish = %v, publishedCounted = %v\n",
						numOutputReloads, numEventsToPublish, publishedCount.Load(),
					)
				}
				return success
			}, &quick.Config{MaxCount: 1})

			if err != nil {
				t.Error(err)
			}
		})
	}
}
