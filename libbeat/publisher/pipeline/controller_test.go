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
	"sync"
	"testing"
	"testing/quick"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/beats/v7/libbeat/internal/testutil"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/beats/v7/libbeat/publisher/queue/memqueue"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"

	//"github.com/elastic/beats/v7/libbeat/tests/resources"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutputReload(t *testing.T) {
	tests := map[string]func(mockPublishFn) outputs.Client{
		"client":         newMockClient,
		"network_client": newMockNetworkClient,
	}

	for name, ctor := range tests {
		t.Run(name, func(t *testing.T) {
			testutil.SeedPRNG(t)

			// Flaky check: https://github.com/elastic/beats/issues/21656
			//goroutines := resources.NewGoroutinesChecker()
			//defer goroutines.Check(t)

			err := quick.Check(func(q uint) bool {
				numEventsToPublish := 15000 + (q % 5000) // 15000 to 19999
				numOutputReloads := 350 + (q % 150)      // 350 to 499

				queueConfig := conf.Namespace{}
				conf, _ := conf.NewConfigFrom(
					fmt.Sprintf("mem.events: %v", numEventsToPublish))
				_ = queueConfig.Unpack(conf)

				var publishedCount atomic.Uint
				countingPublishFn := func(batch publisher.Batch) error {
					publishedCount.Add(uint(len(batch.Events())))
					return nil
				}

				pipeline, err := New(
					beat.Info{},
					Monitors{},
					queueConfig,
					outputs.Group{},
					Settings{},
				)
				require.NoError(t, err)
				defer pipeline.Close()

				var wg sync.WaitGroup
				wg.Add(1)
				go func() {
					// Our initial pipeline has no outputs set, so we need
					// to create the client in a goroutine since any
					// Connect calls will block until the pipeline has an
					// output.
					pipelineClient, err := pipeline.Connect()
					require.NoError(t, err)
					defer pipelineClient.Close()
					for i := uint(0); i < numEventsToPublish; i++ {
						pipelineClient.Publish(beat.Event{})
					}
					wg.Done()
				}()

				for i := uint(0); i < numOutputReloads; i++ {
					outputClient := ctor(countingPublishFn)
					out := outputs.Group{
						Clients: []outputs.Client{outputClient},
					}
					pipeline.outputController.Set(out)
				}

				wg.Wait()

				timeout := 20 * time.Second
				return waitUntilTrue(timeout, func() bool {
					return numEventsToPublish == publishedCount.Load()
				})
			}, &quick.Config{MaxCount: 25})

			if err != nil {
				t.Error(err)
			}
		})
	}
}

func TestSetEmptyOutputsSendsNilChannel(t *testing.T) {
	// Just fill out enough to confirm what's sent to the event consumer,
	// we don't want to start up real helper routines.
	controller := outputController{
		consumer: &eventConsumer{
			targetChan: make(chan consumerTarget, 2),
		},
	}
	controller.Set(outputs.Group{})

	// Two messages should be sent to eventConsumer's targetChan:
	// one to clear the old target while the state is updating,
	// and one with the new metadata after the state update is
	// complete. Since we're setting an empty output group, both
	// of these calls should have a nil target channel.
	target := <-controller.consumer.targetChan
	assert.Nil(t, target.ch, "consumerTarget should receive a nil channel to block batch assembly")
	target = <-controller.consumer.targetChan
	assert.Nil(t, target.ch, "consumerTarget should receive a nil channel to block batch assembly")
}

func TestQueueCreatedOnlyAfterOutputExists(t *testing.T) {
	controller := outputController{
		// Set event limit to 1 so we can easily tell if our settings
		// were used to create the queue.
		queueFactory: memqueue.FactoryForSettings(
			memqueue.Settings{Events: 1},
		),
		consumer: &eventConsumer{
			// We aren't testing the values sent to eventConsumer, we
			// just need a placeholder here so outputController can
			// send configuration updates without blocking.
			targetChan: make(chan consumerTarget, 4),
		},
		observer: nilObserver,
	}
	// Set to an empty output group. This should not create a queue.
	controller.Set(outputs.Group{})
	require.Nil(t, controller.queue, "Queue should be nil after setting empty output")

	controller.Set(outputs.Group{
		Clients: []outputs.Client{newMockClient(nil)},
	})
	require.NotNil(t, controller.queue, "Queue should be created after setting nonempty output")
	assert.Equal(t, 1, controller.queue.BufferConfig().MaxEvents, "Queue should be created using provided settings")
}

func TestOutputQueueFactoryTakesPrecedence(t *testing.T) {
	// If there are queue settings provided by both the pipeline and
	// the output, the output settings should be used.
	controller := outputController{
		queueFactory: memqueue.FactoryForSettings(
			memqueue.Settings{Events: 1},
		),
		consumer: &eventConsumer{
			targetChan: make(chan consumerTarget, 4),
		},
		observer: nilObserver,
	}
	controller.Set(outputs.Group{
		Clients:      []outputs.Client{newMockClient(nil)},
		QueueFactory: memqueue.FactoryForSettings(memqueue.Settings{Events: 2}),
	})

	// The pipeline queue settings has max events 1, the output has
	// max events 2, the result should be a queue with max events 2.
	assert.Equal(t, 2, controller.queue.BufferConfig().MaxEvents, "Queue should be created using settings from the output")
}

func TestFailedQueueFactoryRevertsToDefault(t *testing.T) {
	defaultSettings, _ := memqueue.SettingsForUserConfig(nil)
	failedFactory := func(_ *logp.Logger, _ func(int), _ int) (queue.Queue, error) {
		return nil, fmt.Errorf("This queue creation intentionally failed")
	}
	controller := outputController{
		queueFactory: failedFactory,
		consumer: &eventConsumer{
			targetChan: make(chan consumerTarget, 4),
		},
		observer: nilObserver,
		monitors: Monitors{
			Logger: logp.NewLogger("tests"),
		},
	}
	controller.Set(outputs.Group{
		Clients: []outputs.Client{newMockClient(nil)},
	})

	assert.Equal(t, defaultSettings.Events, controller.queue.BufferConfig().MaxEvents, "Queue should fall back on default settings when input is invalid")
}

func TestQueueProducerBlocksUntilOutputIsSet(t *testing.T) {
	controller := outputController{
		queueFactory: memqueue.FactoryForSettings(memqueue.Settings{Events: 1}),
		consumer: &eventConsumer{
			targetChan: make(chan consumerTarget, 4),
		},
		observer: nilObserver,
	}
	// Send producer requests from different goroutines. They should all
	// block, because there is no queue, but they should become unblocked
	// once we set a nonempty output.
	const producerCount = 10
	remaining := atomic.MakeInt(producerCount)
	for i := 0; i < producerCount; i++ {
		go func() {
			controller.queueProducer(queue.ProducerConfig{})
			remaining.Dec()
		}()
	}
	allStarted := waitUntilTrue(time.Second, func() bool {
		return len(controller.pendingRequests) == producerCount
	})
	assert.True(t, allStarted, "All queueProducer requests should be saved as pending requests by outputController")
	assert.Equal(t, producerCount, remaining.Load(), "No queueProducer request should return before an output is set")

	// Set the output, then ensure that it unblocks all the waiting goroutines.
	controller.Set(outputs.Group{
		Clients: []outputs.Client{newMockClient(nil)},
	})
	allFinished := waitUntilTrue(time.Second, func() bool {
		return remaining.Load() == 0
	})
	assert.True(t, allFinished, "All queueProducer requests should be unblocked once an output is set")
}
