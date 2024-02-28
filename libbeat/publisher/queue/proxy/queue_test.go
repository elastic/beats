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

package proxyqueue

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/elastic/beats/v7/libbeat/publisher/queue"
)

// Because acknowledgments are partially asynchronous (acknowledging
// a batch notifies the queue, which then notifies the original producer
// callback), we can't make a fully deterministic test for ACK counts
// since in principle it depends on the scheduler.
// Nevertheless, in practice the latency should be very low. testACKListener
// is a helper object to track ACK state while allowing for timeouts when
// some propagation delay is unavoidable.
type testACKListener struct {
	sync.Mutex

	ackedCount int

	// If not enough ACKs have been received yet, waitForTotalACKs sets
	// waiting to true and listens on updateChan.
	// If waiting is set when the ACK callback is called, then it sends
	// on updateChan to wake up waitForTotalACKs.
	waiting    bool
	updateChan chan struct{}
}

func TestBasicEventFlow(t *testing.T) {
	logger := logp.NewLogger("proxy-queue-tests")

	// Create a proxy queue where each batch is at most 2 events
	testQueue := NewQueue(logger, nil, Settings{BatchSize: 2})
	defer testQueue.Close()

	listener := newTestACKListener()
	producer := testQueue.Producer(queue.ProducerConfig{
		ACK: listener.ACK,
	})
	// Try to publish 3 events, only the first two should succeed until we read a batch
	success := producer.TryPublish(1)
	assert.True(t, success)
	success = producer.TryPublish(2)
	assert.True(t, success)
	success = producer.TryPublish(3)
	assert.False(t, success, "Current batch should only fit two events")

	batch, err := testQueue.Get(0)
	assert.NoError(t, err, "Should be able to read a batch")
	assert.Equal(t, 0, listener.ackedCount, "No batches have been acked yet")
	batch.Done()
	assert.NoError(t, listener.waitForTotalACKs(2, time.Second))

	// Make sure that reading an event unblocked the queue
	success = producer.TryPublish(4)
	assert.True(t, success, "Queue should accept incoming event")
}

func TestBlockedProducers(t *testing.T) {
	logger := logp.NewLogger("proxy-queue-tests")

	// Create a proxy queue where each batch is at most 2 events
	testQueue := NewQueue(logger, nil, Settings{BatchSize: 2})
	defer testQueue.Close()

	listener := newTestACKListener()

	// Create many producer goroutines and send an event through each
	// one. Only two events can be in the queue at any one time, so
	// the rest of the producers will block until we read enough batches
	// from the queue.
	const PRODUCER_COUNT = 10
	for i := 0; i < PRODUCER_COUNT; i++ {
		go func(producerID int) {
			producer := testQueue.Producer(queue.ProducerConfig{
				ACK: listener.ACK,
			})
			producer.Publish(producerID)
		}(i)
	}

	consumedEventCount := 0
	batches := []queue.Batch{}
	// First, read all the events. We should be able to do this successfully
	// even before any have been acknowledged.
	for consumedEventCount < PRODUCER_COUNT {
		batch, err := testQueue.Get(0)
		assert.NoError(t, err)
		consumedEventCount += batch.Count()
		batches = append(batches, batch)
	}

	assert.Equal(t, 0, listener.ackedCount, "No batches have been acked yet")
	for _, batch := range batches {
		batch.Done()
	}
	assert.NoError(t, listener.waitForTotalACKs(PRODUCER_COUNT, time.Second))
}

func TestOutOfOrderACK(t *testing.T) {
	logger := logp.NewLogger("proxy-queue-tests")

	// Create a proxy queue where each batch is at most 2 events
	testQueue := NewQueue(logger, nil, Settings{BatchSize: 2})
	defer testQueue.Close()

	listener := newTestACKListener()
	producer := testQueue.Producer(queue.ProducerConfig{
		ACK: listener.ACK,
	})

	const BATCH_COUNT = 10
	batches := []queue.Batch{}
	for i := 0; i < BATCH_COUNT; i++ {
		// Publish two events
		success := producer.Publish(0)
		assert.True(t, success, "Publish should succeed")
		success = producer.Publish(0)
		assert.True(t, success, "Publish should succeed")

		// Consume a batch, which should contain the events we just published
		batch, err := testQueue.Get(0)
		assert.NoError(t, err)
		batch.FreeEntries()
		assert.Equal(t, 2, batch.Count())

		batches = append(batches, batch)
	}

	// Acknowledge all except the first batch
	for _, batch := range batches[1:] {
		batch.Done()
	}
	// Make sure that no ACKs come in even if we wait a bit
	err := listener.waitForTotalACKs(1, 50*time.Millisecond)
	assert.Error(t, err, "No ACK callbacks should have been called yet")

	// ACKing the first batch should unblock all the rest
	batches[0].Done()
	assert.NoError(t, listener.waitForTotalACKs(BATCH_COUNT*2, time.Second))
}

func TestWriteAfterClose(t *testing.T) {
	logger := logp.NewLogger("proxy-queue-tests")

	testQueue := NewQueue(logger, nil, Settings{BatchSize: 2})
	producer := testQueue.Producer(queue.ProducerConfig{})
	testQueue.Close()

	// Make sure Publish fails instead of blocking
	success := producer.Publish(1)
	assert.False(t, success, "Publish should fail since queue is closed")
}

func newTestACKListener() *testACKListener {
	return &testACKListener{
		updateChan: make(chan struct{}, 1),
	}
}

// ACK should be provided to the queue producer. It can be safely called from
// multiple goroutines.
func (l *testACKListener) ACK(count int) {
	l.Lock()
	l.ackedCount += count
	if l.waiting {
		// If waitFortotalACKs is waiting on something, wake it up so it can retry.
		l.waiting = false
		l.updateChan <- struct{}{}
	}
	l.Unlock()
}

// flush should be called on timeout, to clear updateChan if needed.
func (l *testACKListener) flush() {
	l.Lock()
	select {
	case <-l.updateChan:
	default:
	}
	l.waiting = false
	l.Unlock()
}

// waitForTotalACKs waits until the specified number of total ACKs have been
// received, or the timeout interval is exceeded. It should only be called
// from a single goroutine at once.
func (l *testACKListener) waitForTotalACKs(targetCount int, timeout time.Duration) error {
	timeoutChan := time.After(timeout)
	for {
		l.Lock()
		if l.ackedCount >= targetCount {
			l.Unlock()
			return nil
		}
		// Not enough ACKs have been sent yet, so we have to wait.
		l.waiting = true
		l.Unlock()
		select {
		case <-l.updateChan:
			// New ACKs came in, retry
			continue
		case <-timeoutChan:
			l.flush()
			return fmt.Errorf("timed out waiting for acknowledgments: have %d, wanted %d", l.ackedCount, targetCount)
		}
	}
}
