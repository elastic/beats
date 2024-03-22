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

package memqueue

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp"
)

func TestFlushSettingsDoNotBlockFullBatches(t *testing.T) {
	// In previous versions of the queue, setting flush.min_events (currently
	// corresponding to memqueue.Settings.MaxGetRequest) to a high value would
	// delay get requests even if the number of requested events was immediately
	// available. This test verifies that Get requests that can be completely
	// filled do not wait for the flush timer.

	broker := newQueue(
		logp.NewLogger("testing"),
		nil,
		Settings{
			Events:        1000,
			MaxGetRequest: 500,
			FlushTimeout:  10 * time.Second,
		},
		10)

	producer := newProducer(broker, nil, nil, false, nil)
	rl := broker.runLoop
	for i := 0; i < 100; i++ {
		// Pair each publish call with an iteration of the run loop so we
		// get a response.
		go rl.runIteration()
		_, ok := producer.Publish(i)
		require.True(t, ok, "Queue publish call must succeed")
	}

	// The queue now has 100 events, but MaxGetRequest is 500.
	// In the old queue, a Get call now would block until the flush
	// timer expires. With current changes, it should return
	// immediately on any request size up to 100.
	go func() {
		// Run the Get asynchronously so the test itself doesn't block if
		// there's a logical error.
		_, _ = broker.Get(100)
	}()
	rl.runIteration()
	assert.Nil(t, rl.pendingGetRequest, "Queue should have no pending get request since the request should succeed immediately")
	assert.Equal(t, 100, rl.consumedCount, "Queue should have a consumedCount of 100 after a consumer requested all its events")
}

func TestFlushSettingsBlockPartialBatches(t *testing.T) {
	// The previous test confirms that Get requests are handled immediately if
	// there are enough events. This one uses the same setup to confirm that
	// Get requests are delayed if there aren't enough events.

	broker := newQueue(
		logp.NewLogger("testing"),
		nil,
		Settings{
			Events:        1000,
			MaxGetRequest: 500,
			FlushTimeout:  10 * time.Second,
		},
		10)

	producer := newProducer(broker, nil, nil, false, nil)
	rl := broker.runLoop
	for i := 0; i < 100; i++ {
		// Pair each publish call with an iteration of the run loop so we
		// get a response.
		go rl.runIteration()
		_, ok := producer.Publish("some event")
		require.True(t, ok, "Queue publish call must succeed")
	}

	// The queue now has 100 events, and a positive flush timeout, so a
	// request for 101 events should block.
	go func() {
		// Run the Get asynchronously so the test itself doesn't block if
		// there's a logical error.
		_, _ = broker.Get(101)
	}()
	rl.runIteration()
	assert.NotNil(t, rl.pendingGetRequest, "Queue should have a pending get request since the queue doesn't have the requested event count")
	assert.Equal(t, 0, rl.consumedCount, "Queue should have a consumedCount of 0 since the Get request couldn't be completely filled")

	// Now confirm that adding one more event unblocks the request
	go func() {
		_, _ = producer.Publish("some event")
	}()
	rl.runIteration()
	assert.Nil(t, rl.pendingGetRequest, "Queue should have no pending get request since adding an event should unblock the previous one")
	assert.Equal(t, 101, rl.consumedCount, "Queue should have a consumedCount of 101 after adding an event unblocked the pending get request")
}
