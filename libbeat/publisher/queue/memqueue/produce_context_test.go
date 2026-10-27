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
	"context"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/elastic-agent-libs/logp"
)

// TestPublishContextCancel checks that a publish waiting on a full queue
// returns the ctx error when its deadline passes, and that the abandoned
// request is never inserted later, so entry IDs and ACK counts stay exact.
func TestPublishContextCancel(t *testing.T) {
	for _, tc := range []struct {
		name string
		ack  bool
	}{
		{name: "forgetful"},
		{name: "ack", ack: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				q := NewQueue[int](logp.NewNopLogger(), nil, Settings{
					Events: 2, MaxGetRequest: 2, FlushTimeout: time.Millisecond,
				}, 0, nil)
				defer q.Close(true)
				var acked atomic.Int64
				var cfg queue.ProducerConfig
				if tc.ack {
					cfg.ACK = func(n int) { acked.Add(int64(n)) }
				}
				p := q.Producer(cfg)
				for i := range 2 {
					_, ok := p.Publish(i)
					require.True(t, ok, "publish %d must succeed", i)
				}

				ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
				defer cancel()
				start := time.Now()
				_, err := p.(queue.ContextProducer[int]).PublishContext(ctx, 100)
				assert.ErrorIs(t, err, context.DeadlineExceeded, "PublishContext on a full queue must return the ctx error")
				assert.Equal(t, 50*time.Millisecond, time.Since(start), "PublishContext must return as soon as the deadline passes")

				b, err := q.Get(2)
				require.NoError(t, err, "Get must return the queued events")
				assert.Equal(t, 2, b.Count(), "both successful events must be queued")
				b.Done()

				id, ok := p.Publish(3)
				require.True(t, ok, "publish after the cancel must succeed")
				assert.Equal(t, queue.EntryID(2), id, "the cancelled entry must not consume an entry ID")
				b, err = q.Get(2)
				require.NoError(t, err, "Get must return the next event")
				require.Equal(t, 1, b.Count(), "the cancelled entry must never be queued")
				assert.Equal(t, 3, b.Entry(0), "the queued entry must be the one published after the cancel")
				b.Done()

				p.Close()
				assertChanClosed(t, p.ACKWaitChan(), "ackWait must close: the cancelled event must not count as published")
				// Let the ackLoop finish: it does not watch for shutdown while
				// handing a deletion to the run loop, so a force-close racing the
				// last Done would strand it.
				synctest.Wait()
				if tc.ack {
					assert.Equal(t, int64(3), acked.Load(), "ACK callbacks must count exactly the published events")
				}
			})
		})
	}
}

// TestPublishContextBackToBack checks that one producer can publish with
// PublishContext repeatedly. The run loop claims each request, so the
// producer must reset the claim before reusing it, else the run loop would
// discard the next request as abandoned and the publish would hang.
func TestPublishContextBackToBack(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		q := NewQueue[int](logp.NewNopLogger(), nil, Settings{
			Events: 4, MaxGetRequest: 4, FlushTimeout: time.Millisecond,
		}, 0, nil)
		defer q.Close(true)
		cp := q.Producer(queue.ProducerConfig{}).(queue.ContextProducer[int])

		for i := range 3 {
			ctx, cancel := context.WithTimeout(t.Context(), time.Second)
			id, err := cp.PublishContext(ctx, i)
			cancel()
			require.NoError(t, err, "publish %d on a queue with room must succeed", i)
			assert.Equal(t, queue.EntryID(i), id, "publish %d must get the next entry ID", i)
		}
	})
}

// TestPublishContextCancelWhilePushChanFull checks that a publish that cannot
// even hand its request to the run loop, because pushChan's buffer is full,
// still returns the ctx error when its deadline passes.
func TestPublishContextCancelWhilePushChanFull(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		q := NewQueue[int](logp.NewNopLogger(), nil, Settings{
			Events: 2, MaxGetRequest: 2, FlushTimeout: time.Millisecond,
		}, 0, nil)
		defer q.Close(true) // releases the blocked producers
		for i := range 2 {
			_, ok := q.Producer(queue.ProducerConfig{}).Publish(i)
			require.True(t, ok, "publish %d must succeed", i)
		}
		// The queue is full, so the run loop stops reading pushChan. Each
		// producer has one request outstanding, so it takes more producers
		// than pushChan's buffer to fill it.
		for i := range cap(q.pushChan) + 2 {
			go q.Producer(queue.ProducerConfig{}).Publish(10 + i)
		}
		synctest.Wait()
		require.Equal(t, cap(q.pushChan), len(q.pushChan), "pushChan must be full")

		cp := q.Producer(queue.ProducerConfig{}).(queue.ContextProducer[int])
		ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
		defer cancel()
		res := make(chan error, 1)
		go func() {
			_, err := cp.PublishContext(ctx, 100)
			res <- err
		}()
		select {
		case err := <-res:
			assert.ErrorIs(t, err, context.DeadlineExceeded, "a publish blocked on a full pushChan must return the ctx error")
		case <-time.After(time.Second):
			t.Error("PublishContext blocked on a full pushChan ignored its ctx deadline")
		}
	})
}

// TestPublishContextCancelBeforeBrokerAccepts drives the run loop by hand: a
// request abandoned before the broker reads it must be discarded, not
// inserted, and the producer must keep working.
func TestPublishContextCancelBeforeBrokerAccepts(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		b := newQueue[int](logp.NewNopLogger(), nil, Settings{Events: 2, MaxGetRequest: 2}, 0, nil)
		p := b.Producer(queue.ProducerConfig{ACK: func(int) {}})
		cp := p.(queue.ContextProducer[int])

		ctx, cancel := context.WithCancel(t.Context())
		res := make(chan error, 1)
		go func() {
			_, err := cp.PublishContext(ctx, 1)
			res <- err
		}()
		synctest.Wait()
		cancel()
		assert.ErrorIs(t, <-res, context.Canceled, "a request not yet accepted must be abandoned on cancel")

		b.runLoop.runIteration() // the broker reads the abandoned request
		assert.Equal(t, 0, b.runLoop.eventCount, "an abandoned request must not be inserted")

		ids := make(chan queue.EntryID, 1)
		go func() {
			id, err := cp.PublishContext(t.Context(), 2)
			assert.NoError(t, err, "publish after an abandoned request must succeed")
			ids <- id
		}()
		synctest.Wait()
		b.runLoop.runIteration()
		assert.Equal(t, queue.EntryID(0), <-ids, "the abandoned request must not consume an entry ID")
		assert.Equal(t, 1, b.runLoop.eventCount, "the new request must be inserted")
		assert.Equal(t, uint64(1), p.(*ackProducer[int]).producedCount.Load(), "the abandoned request must not count as produced")
	})
}

// TestPublishContextRepublishBeforeBrokerReadsAbandoned checks that a request
// abandoned while still buffered cannot be revived by the producer's next
// publish: the next request must not share the abandoned request's claim.
func TestPublishContextRepublishBeforeBrokerReadsAbandoned(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		b := newQueue[int](logp.NewNopLogger(), nil, Settings{Events: 2, MaxGetRequest: 2}, 0, nil)
		cp := b.Producer(queue.ProducerConfig{ACK: func(int) {}}).(queue.ContextProducer[int])

		ctx, cancel := context.WithCancel(t.Context())
		res := make(chan error, 1)
		go func() {
			_, err := cp.PublishContext(ctx, 1)
			res <- err
		}()
		synctest.Wait()
		cancel()
		require.ErrorIs(t, <-res, context.Canceled, "a request not yet accepted must be abandoned on cancel")

		// Publish again while the abandoned request is still in pushChan.
		ids := make(chan queue.EntryID, 1)
		go func() {
			id, err := cp.PublishContext(t.Context(), 2)
			assert.NoError(t, err, "publish after an abandoned request must succeed")
			ids <- id
		}()
		synctest.Wait()
		b.runLoop.runIteration() // reads the abandoned request
		b.runLoop.runIteration() // reads the new request
		assert.Equal(t, queue.EntryID(0), <-ids, "the new request must get the first entry ID")
		assert.Equal(t, 1, b.runLoop.eventCount, "exactly one entry must be queued")
		assert.Equal(t, 2, b.buf[0].event, "the queued entry must be the new value, not the abandoned one")
	})
}

// TestPublishContextCancelAfterBrokerAccepts covers the other side of the
// race: once the broker has inserted the entry, a ctx that ends before the
// producer reads the reply must not turn the publish into a failure.
func TestPublishContextCancelAfterBrokerAccepts(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		b := newQueue[int](logp.NewNopLogger(), nil, Settings{Events: 2, MaxGetRequest: 2}, 0, nil)
		p := b.Producer(queue.ProducerConfig{ACK: func(int) {}})

		ctx, cancel := context.WithCancel(t.Context())
		res := make(chan error, 1)
		go func() {
			_, err := p.(queue.ContextProducer[int]).PublishContext(ctx, 1)
			res <- err
		}()
		synctest.Wait()
		b.runLoop.runIteration() // the broker accepts and replies
		cancel()
		assert.NoError(t, <-res, "an entry the broker accepted must be reported as published")
		assert.Equal(t, 1, b.runLoop.eventCount, "the accepted entry must be in the queue")
	})
}
