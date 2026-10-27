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
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/publisher/queue"
)

// publishResult is the outcome of a PublishContext call run by publishAsync.
type publishResult struct {
	id  queue.EntryID
	err error
}

// publishAsync runs PublishContext on its own goroutine and delivers the
// result on the returned channel.
func publishAsync(ctx context.Context, p queue.Producer[int], v int) <-chan publishResult {
	ch := make(chan publishResult, 1)
	go func() {
		id, err := p.(queue.ContextProducer[int]).PublishContext(ctx, v)
		ch <- publishResult{id: id, err: err}
	}()
	return ch
}

// waitDeadline publishes with a 50ms deadline on a producer whose publish must
// block, and checks it returns the deadline error exactly when the deadline
// passes (synctest's clock makes "within 50ms" exact).
func waitDeadline(t *testing.T, p queue.Producer[int]) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	res := publishAsync(ctx, p, 2)
	synctest.Wait()
	select {
	case r := <-res:
		t.Fatalf("PublishContext returned (%d, %v) before its deadline; want it to wait", r.id, r.err)
	default:
	}
	r := <-res
	assert.ErrorIs(t, r.err, context.DeadlineExceeded, "PublishContext must return the ctx error when the deadline passes")
	assert.Equal(t, 50*time.Millisecond, time.Since(start), "PublishContext must return as soon as the deadline passes")
}

func TestPublishContextCancelWhileWaitingForPoolSlot(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		pool := NewPool[int](Settings{Events: 1}, nil)
		q := pool.Connect()
		defer q.Close(true)
		p := q.Producer(queue.ProducerConfig{ACK: func(int) {}})
		_, ok := p.Publish(1)
		require.True(t, ok, "first publish must fill the pool")

		waitDeadline(t, p)

		assert.Equal(t, 0, pool.Available(), "the cancelled publish must not take or free a pool slot")
		assert.Equal(t, int64(1), q.live.Load(), "the cancelled publish must return its queue budget")

		b, err := q.Get(0)
		require.NoError(t, err, "Get must return the first event")
		assert.Equal(t, 1, b.Count(), "only the first event must be queued")
		b.Done()
		assert.Equal(t, 1, pool.Available(), "the pool slot must be free once the first event is acked")

		// The cancelled publish rolled back its accounting, so ackWait closes.
		p.Close()
		assertChanClosed(t, p.ACKWaitChan(), "ackWait must close: the cancelled event must not count as published")
	})
}

func TestPublishContextCancelWhileWaitingForQueueCap(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		pool := NewPool[int](Settings{Events: 4}, nil)
		q := pool.Connect()
		defer q.Close(true)
		q.SetTarget(2)
		p := q.Producer(queue.ProducerConfig{ACK: func(int) {}})
		_, ok := p.Publish(1)
		require.True(t, ok, "first publish must succeed")
		_, ok = p.Publish(2)
		require.True(t, ok, "second publish must fill the queue cap")

		waitDeadline(t, p)

		assert.Equal(t, int64(2), q.live.Load(), "the cancelled publish must return its queue budget")
		assert.Equal(t, 0, pool.Available(), "the cancelled publish must not take a pool slot")

		p.Close()
		b, err := q.Get(0)
		require.NoError(t, err, "Get must return the queued events")
		assert.Equal(t, 2, b.Count(), "only the two successful events must be queued")
		b.Done()
		assertChanClosed(t, p.ACKWaitChan(), "ackWait must close: the cancelled event must not count as published")
	})
}

// TestPublishContextCancelPassesPoolWakeup checks a cancelled waiter cannot
// strand another: with two producers parked on a full pool, one cancelling
// while a slot frees, the slot must still go to a publisher. The release goes
// through releaseSlot, which wakes one waiter with Signal.
func TestPublishContextCancelPassesPoolWakeup(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		pool := NewPool[int](Settings{Events: 1}, nil)
		q := pool.Connect()
		idx, ok := pool.free.tryGrab(0)
		require.True(t, ok, "the test must hold the only pool slot")
		p := q.Producer(queue.ProducerConfig{})

		ctx, cancel := context.WithCancel(t.Context())
		cancelled := publishAsync(ctx, p, 1)
		synctest.Wait() // park the cancelled waiter first, so Signal picks it
		waiting := publishAsync(t.Context(), p, 2)
		synctest.Wait()

		pool.releaseSlot(idx)
		cancel()
		synctest.Wait()

		assert.Equal(t, 0, pool.Available(), "the freed slot must be taken by a publisher, not stranded")
		q.Close(true) // unblock whichever waiter is left
		a, b := <-cancelled, <-waiting
		published := 0
		for _, r := range []publishResult{a, b} {
			if r.err == nil {
				published++
			}
		}
		assert.Equal(t, 1, published, "exactly one waiter must publish into the freed slot (got %v and %v)", a.err, b.err)
	})
}

// TestPublishContextCancelPassesCapWakeup is the per-queue cap counterpart of
// TestPublishContextCancelPassesPoolWakeup.
func TestPublishContextCancelPassesCapWakeup(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		pool := NewPool[int](Settings{Events: 4}, nil)
		q := pool.Connect()
		q.SetTarget(1)
		require.True(t, q.tryReserve(), "the test must hold the only unit of queue budget")
		p := q.Producer(queue.ProducerConfig{})

		ctx, cancel := context.WithCancel(t.Context())
		cancelled := publishAsync(ctx, p, 1)
		synctest.Wait()
		waiting := publishAsync(t.Context(), p, 2)
		synctest.Wait()

		q.releaseLive(1)
		cancel()
		synctest.Wait()

		assert.Equal(t, int64(1), q.live.Load(), "the freed budget must be taken by a publisher, not stranded")
		q.Close(true)
		a, b := <-cancelled, <-waiting
		published := 0
		for _, r := range []publishResult{a, b} {
			if r.err == nil {
				published++
			}
		}
		assert.Equal(t, 1, published, "exactly one waiter must publish into the freed budget (got %v and %v)", a.err, b.err)
	})
}

// TestPublishContextClosed checks a closed queue reports queue.ErrClosed.
func TestPublishContextClosed(t *testing.T) {
	pool := NewPool[int](Settings{Events: 1}, nil)
	q := pool.Connect()
	p := q.Producer(queue.ProducerConfig{})
	require.NoError(t, q.Close(true), "Close must succeed")
	_, err := p.(queue.ContextProducer[int]).PublishContext(t.Context(), 1)
	assert.ErrorIs(t, err, queue.ErrClosed, "PublishContext on a closed queue must return ErrClosed")
}
