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

package queuetest

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// TestSingleProducerConsumer tests buffered events for a producer getting
// cancelled will not be consumed anymore. Concurrent producer/consumer pairs
// might still have active events not yet ACKed (not tested here).
//
// Note: queues not requiring consumers to ACK a events in order to
//       return ACKs to the producer are not supported by this test.
func TestProducerCancelRemovesEvents(t *testing.T, factory QueueFactory) {
	fn := withOptLogOutput(true, func(t *testing.T) {
		var (
			i  int
			N1 = 3
			N2 = 10
		)

		log := NewTestLogger(t)
		b := factory(t)
		defer b.Close()

		log.Debug("create first producer")
		producer := b.Producer(queue.ProducerConfig{
			ACK:          func(int) {}, // install function pointer, so 'cancel' will remove events
			DropOnCancel: true,
		})

		for ; i < N1; i++ {
			log.Debugf("send event %v to first producer", i)
			producer.Publish(makeEvent(mapstr.M{
				"value": i,
			}))
		}

		// cancel producer
		log.Debugf("cancel producer")
		producer.Cancel()

		// reconnect and send some more events
		log.Debug("connect new producer")
		producer = b.Producer(queue.ProducerConfig{})
		for ; i < N2; i++ {
			log.Debugf("send event %v to new producer", i)
			producer.Publish(makeEvent(mapstr.M{
				"value": i,
			}))
		}

		// consumer all events
		consumer := b.Consumer()
		total := N2 - N1
		events := make([]publisher.Event, 0, total)
		for len(events) < total {
			batch, err := consumer.Get(-1) // collect all events
			if err != nil {
				panic(err)
			}

			events = append(events, batch.Events()...)
			batch.ACK()
		}

		// verify
		if total != len(events) {
			assert.Equal(t, total, len(events))
			return
		}

		for i, event := range events {
			value := event.Content.Fields["value"].(int)
			assert.Equal(t, i+N1, value)
		}
	})

	fn(t)
}
