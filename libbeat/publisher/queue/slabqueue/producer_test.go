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

package slabqueue

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/publisher/queue"
)

func assertChanClosed(t *testing.T, ch <-chan struct{}, msg string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal(msg)
	}
}

func assertChanOpen(t *testing.T, ch <-chan struct{}, msg string) {
	t.Helper()
	select {
	case <-ch:
		t.Fatal(msg)
	default:
	}
}

// TestACKWaitClosesAfterCloseAndDrain verifies the channel closes once the
// producer is closed and all its events have been acknowledged via Done.
func TestACKWaitClosesAfterCloseAndDrain(t *testing.T) {
	pool := NewPool[int](Settings{Events: 4}, nil)
	q := pool.Connect()
	p := q.Producer(queue.ProducerConfig{ACK: func(int) {}})

	_, ok := p.Publish(1)
	require.True(t, ok)
	_, ok = p.Publish(2)
	require.True(t, ok)

	// Closing while events are still in flight must NOT close the channel yet.
	p.Close()
	assertChanOpen(t, p.ACKWaitChan(), "ackWait must stay open while events are unacked")

	b, err := q.Get(0)
	require.NoError(t, err)
	b.Done()

	assertChanClosed(t, p.ACKWaitChan(), "ackWait must close once closed and fully acked")
}

// TestACKWaitClosesWhenAlreadyDrainedAtClose covers the path where every event
// is acked before Close: Close itself must close the channel (no later Done
// will fire to do it).
func TestACKWaitClosesWhenAlreadyDrainedAtClose(t *testing.T) {
	pool := NewPool[int](Settings{Events: 4}, nil)
	q := pool.Connect()
	p := q.Producer(queue.ProducerConfig{ACK: func(int) {}})

	_, ok := p.Publish(1)
	require.True(t, ok)
	b, err := q.Get(0)
	require.NoError(t, err)
	b.Done()

	// Fully acked but still open: must not be closed.
	assertChanOpen(t, p.ACKWaitChan(), "ackWait must stay open while producer is open")

	p.Close()
	assertChanClosed(t, p.ACKWaitChan(), "Close on an already-drained producer must close ackWait")
}

// TestACKWaitClosesImmediatelyWithNoEvents verifies a producer that published
// nothing closes its channel as soon as it is closed.
func TestACKWaitClosesImmediatelyWithNoEvents(t *testing.T) {
	pool := NewPool[int](Settings{Events: 4}, nil)
	q := pool.Connect()
	p := q.Producer(queue.ProducerConfig{})

	assertChanOpen(t, p.ACKWaitChan(), "ackWait must be open before Close")
	p.Close()
	assertChanClosed(t, p.ACKWaitChan(), "Close with no published events must close ackWait")
}

// TestACKWaitClosesOnForceClose verifies that force-closing the queue unblocks
// every producer's channel even though force-close suppresses ACK callbacks
// (so the ack accounting alone would never close it).
func TestACKWaitClosesOnForceClose(t *testing.T) {
	pool := NewPool[int](Settings{Events: 4}, nil)
	q := pool.Connect()
	p := q.Producer(queue.ProducerConfig{ACK: func(int) {}})

	_, ok := p.Publish(1)
	require.True(t, ok)
	_, ok = p.Publish(2)
	require.True(t, ok)

	// Events are outstanding and the producer is not even closed; a graceful
	// close would leave ackWait open, but a force-close must unblock it.
	assertChanOpen(t, p.ACKWaitChan(), "ackWait must be open before force close")
	require.NoError(t, q.Close(true))
	assertChanClosed(t, p.ACKWaitChan(), "force close must close ackWait regardless of acks")
}

// TestACKWaitClosesWhenTailBatchReleased verifies that a producer whose tail
// batch is abandoned via Release (not Done) still has its channel closed:
// abandoned events count as resolved.
func TestACKWaitClosesWhenTailBatchReleased(t *testing.T) {
	pool := NewPool[int](Settings{Events: 4}, nil)
	q := pool.Connect()
	p := q.Producer(queue.ProducerConfig{ACK: func(int) {}})

	_, ok := p.Publish(1)
	require.True(t, ok)
	_, ok = p.Publish(2)
	require.True(t, ok)

	b, err := q.Get(0)
	require.NoError(t, err)

	p.Close()
	assertChanOpen(t, p.ACKWaitChan(), "ackWait must be open before the batch is resolved")

	// Abandon the batch instead of acking it. Its events are resolved (not
	// acked) so the channel must still close.
	b.Release()
	assertChanClosed(t, p.ACKWaitChan(), "Release of the tail batch must resolve events and close ackWait")
}

// TestACKWaitClosedForProducerCreatedAfterClose verifies the late-producer
// race is handled: a producer obtained from an already-closing queue gets an
// already-closed channel rather than one that never closes.
func TestACKWaitClosedForProducerCreatedAfterClose(t *testing.T) {
	pool := NewPool[int](Settings{Events: 4}, nil)
	q := pool.Connect()
	require.NoError(t, q.Close(false))

	p := q.Producer(queue.ProducerConfig{})
	assertChanClosed(t, p.ACKWaitChan(), "a producer created on a closing queue must have a closed ackWait")
}
