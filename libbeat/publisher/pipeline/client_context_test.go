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
	"context"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/acker"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/beats/v7/libbeat/publisher/queue/memqueue"
	"github.com/elastic/beats/v7/libbeat/publisher/queue/slabqueue"
	"github.com/elastic/elastic-agent-libs/logp"
)

// newContextTestClient returns an open client publishing to p, wired like
// Pipeline.ConnectWith with el as ClientConfig.EventListener (nil for none).
func newContextTestClient(p queue.Producer[publisher.Event], l beat.ClientListener, el beat.EventListener) *client {
	c := &client{
		logger:         logp.NewNopLogger(),
		producer:       p,
		ctxProducer:    cancellableProducer(p, el),
		observer:       nilObserver,
		eventListener:  el,
		clientListener: l,
	}
	if el == nil {
		c.eventListener = acker.Nil()
	}
	if c.ctxProducer != nil {
		c.closeCtx, c.closeCancel = context.WithCancel(context.Background())
	}
	c.isOpen.Store(true)
	return c
}

// TestClientPublishContextCancelReleasesClose checks that cancelling a
// PublishContext blocked on a full queue returns the ctx error and releases
// the client, so Close no longer waits behind it.
func TestClientPublishContextCancelReleasesClose(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		pool := slabqueue.NewPool[publisher.Event](slabqueue.Settings{Events: 1}, nil)
		q := pool.Connect()
		defer q.Close(true)
		listener := &mockClientListener{}
		c := newContextTestClient(q.Producer(queue.ProducerConfig{}), listener, nil)

		cp, ok := beat.Client(c).(beat.ContextPublisher)
		require.True(t, ok, "the pipeline client must implement beat.ContextPublisher")
		require.NoError(t, cp.PublishContext(t.Context(), beat.Event{}), "the first publish must fill the queue")

		ctx, cancel := context.WithCancel(t.Context())
		errc := make(chan error, 1)
		go func() { errc <- cp.PublishContext(ctx, beat.Event{}) }()
		synctest.Wait()
		select {
		case err := <-errc:
			t.Fatalf("PublishContext returned %v on a full queue; want it to block", err)
		default:
		}

		cancel()
		assert.ErrorIs(t, <-errc, context.Canceled, "a cancelled PublishContext must return the ctx error")
		assert.Equal(t, 1, listener.eventsPublished, "only the first event must be published")
		assert.Equal(t, 1, listener.eventsDroppedOnPublish, "the cancelled event must be reported as dropped")

		// A sync.Mutex wait is not durable in a synctest bubble, so a held
		// mutex would hang Close below instead of failing; check it first.
		require.True(t, c.mutex.TryLock(), "a cancelled PublishContext must release the client mutex")
		c.mutex.Unlock()

		closed := make(chan struct{})
		go func() {
			defer close(closed)
			assert.NoError(t, c.Close(), "Close must succeed")
		}()
		select {
		case <-closed:
		case <-time.After(50 * time.Millisecond):
			t.Error("Close did not return within 50ms of a cancelled PublishContext")
		}
	})
}

// TestClientPublishContextNewContext checks that a publish waits on the ctx it
// is given, not on a cancelled ctx from an earlier call.
func TestClientPublishContextNewContext(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		pool := slabqueue.NewPool[publisher.Event](slabqueue.Settings{Events: 1}, nil)
		q := pool.Connect()
		defer q.Close(true)
		c := newContextTestClient(q.Producer(queue.ProducerConfig{}), &mockClientListener{}, nil)
		require.NoError(t, c.PublishContext(t.Context(), beat.Event{}), "the first publish must fill the queue")

		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		assert.ErrorIs(t, c.PublishContext(ctx, beat.Event{}), context.Canceled, "a publish with a cancelled ctx on a full queue must return the ctx error")

		errc := make(chan error, 1)
		go func() { errc <- c.PublishContext(t.Context(), beat.Event{}) }()
		synctest.Wait()
		select {
		case err := <-errc:
			t.Fatalf("PublishContext with a live ctx returned %v on a full queue; want it to wait", err)
		default:
		}
		ackBatch(t, q)
		assert.NoError(t, <-errc, "the publish must succeed once the queue has room")
	})
}

// TestClientPublishContextClosed checks that PublishContext reports
// ErrClientClosed, and counts the event as dropped, when the client is closed
// and when a producer without PublishContext refuses the event.
func TestClientPublishContextClosed(t *testing.T) {
	tests := []struct {
		name    string
		close   bool
		canDrop bool
	}{
		{name: "client closed", close: true},
		{name: "client closed with DropIfFull", close: true, canDrop: true},
		{name: "fallback producer closed", close: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			listener := &mockClientListener{}
			c := newContextTestClient(&testProducer{
				publish: func(bool, publisher.Event) (queue.EntryID, bool) { return 0, false },
			}, listener, nil)
			c.canDrop = tc.canDrop
			if tc.close {
				require.NoError(t, c.Close(), "Close must succeed")
			}

			cp, ok := beat.Client(c).(beat.ContextPublisher)
			require.True(t, ok, "the pipeline client must implement beat.ContextPublisher")
			err := cp.PublishContext(t.Context(), beat.Event{})
			assert.ErrorIs(t, err, beat.ErrClientClosed, "PublishContext must report the closed client")
			assert.Equal(t, 1, listener.eventsDroppedOnPublish, "the refused event must be reported as dropped")
		})
	}
}

// TestClientPublishContextClosedQueues checks, with real queue producers, that
// PublishContext reports ErrClientClosed, and counts the event as dropped, when
// the queue or the client is closed.
func TestClientPublishContextClosedQueues(t *testing.T) {
	queues := []struct {
		name string
		new  func() queue.Queue[publisher.Event]
	}{
		{name: "slabqueue", new: func() queue.Queue[publisher.Event] {
			return slabqueue.NewPool[publisher.Event](slabqueue.Settings{Events: 4}, nil).Connect()
		}},
		{name: "memqueue", new: func() queue.Queue[publisher.Event] {
			return memqueue.NewQueue[publisher.Event](logp.NewNopLogger(), nil, memqueue.Settings{
				Events: 4, MaxGetRequest: 1, FlushTimeout: time.Millisecond,
			}, 0, nil)
		}},
	}
	for _, qc := range queues {
		for _, closeClient := range []bool{false, true} {
			name := qc.name + "/queue closed"
			if closeClient {
				name = qc.name + "/client closed"
			}
			t.Run(name, func(t *testing.T) {
				q := qc.new()
				defer q.Close(true)
				listener := &mockClientListener{}
				c := newContextTestClient(q.Producer(queue.ProducerConfig{}), listener, nil)
				require.NotNil(t, c.ctxProducer, "the client must publish through the queue's PublishContext")
				if closeClient {
					require.NoError(t, c.Close(), "Close must succeed")
				} else {
					require.NoError(t, q.Close(false), "closing the queue must succeed")
				}

				// Several publishes: the first one to a closed memqueue may
				// still hand its request to the run loop before failing.
				const publishes = 3
				for i := range publishes {
					err := c.PublishContext(t.Context(), beat.Event{})
					assert.ErrorIs(t, err, beat.ErrClientClosed, "publish %d must report the closed client or queue", i)
				}
				assert.Equal(t, publishes, listener.eventsDroppedOnPublish, "every refused event must be reported as dropped")
				assert.Equal(t, 0, listener.eventsPublished, "no event must be published")
			})
		}
	}
}

// TestClientCloseReleasesPublishContext checks that Close releases a
// PublishContext blocked on a full queue whose ctx is never cancelled: Close
// returns promptly and the publish reports ErrClientClosed.
func TestClientCloseReleasesPublishContext(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
	}{
		{name: "background ctx", ctx: context.Background()},
		{name: "cancellable ctx", ctx: t.Context()},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pool := slabqueue.NewPool[publisher.Event](slabqueue.Settings{Events: 1}, nil)
			q := pool.Connect()
			defer q.Close(true) // releases the publish if Close does not
			entered := make(chan struct{}, 2)
			qp, ok := q.Producer(queue.ProducerConfig{}).(queue.ContextProducer[publisher.Event])
			require.True(t, ok, "slabqueue producers must implement queue.ContextProducer")
			p := &signallingProducer{ContextProducer: qp, entered: entered}
			listener := &mockClientListener{}
			c := newContextTestClient(p, listener, nil)
			require.NoError(t, c.PublishContext(tc.ctx, beat.Event{}), "the first publish must fill the queue")
			<-entered

			errc := make(chan error, 1)
			go func() { errc <- c.PublishContext(tc.ctx, beat.Event{}) }()
			<-entered // the publish holds the client mutex
			closed := make(chan error, 1)
			go func() { closed <- c.Close() }()
			select {
			case err := <-closed:
				assert.NoError(t, err, "Close must succeed")
			case <-time.After(5 * time.Second):
				t.Fatal("Close did not release the in-flight PublishContext within 5s")
			}
			assert.ErrorIs(t, <-errc, beat.ErrClientClosed, "a publish released by Close must report the closed client")
			assert.Equal(t, 1, listener.eventsPublished, "only the first event must be published")
			assert.Equal(t, 1, listener.eventsDroppedOnPublish, "the released event must be reported as dropped")
		})
	}
}

// signallingProducer sends on entered each time PublishContext starts, so a
// test knows the client holds its mutex.
type signallingProducer struct {
	queue.ContextProducer[publisher.Event]
	entered chan<- struct{}
}

func (p *signallingProducer) PublishContext(ctx context.Context, e publisher.Event) (queue.EntryID, error) {
	p.entered <- struct{}{}
	return p.ContextProducer.PublishContext(ctx, e)
}

// TestClientPublishContextFiltered checks that an event dropped by processors
// is not an error.
func TestClientPublishContextFiltered(t *testing.T) {
	pool := slabqueue.NewPool[publisher.Event](slabqueue.Settings{Events: 1}, nil)
	q := pool.Connect()
	defer q.Close(true)
	listener := &mockClientListener{}
	c := newContextTestClient(q.Producer(queue.ProducerConfig{}), listener, nil)
	c.processors = &testProcessor{
		processorFn: func(*beat.Event) (*beat.Event, error) { return nil, nil },
	}

	assert.NoError(t, c.PublishContext(t.Context(), beat.Event{}), "a filtered event must not be an error")
	assert.Equal(t, 1, listener.eventsFiltered, "the event must be reported as filtered")
	assert.Equal(t, 0, listener.eventsDroppedOnPublish, "a filtered event must not be reported as dropped")
}

// TestClientPublishContextWithEventListener checks that a client with an event
// listener does not cancel a publish, which would leave the listener counting
// an event that is never ACKed, and publishes like Publish instead.
func TestClientPublishContextWithEventListener(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		pool := slabqueue.NewPool[publisher.Event](slabqueue.Settings{Events: 1}, nil)
		q := pool.Connect()
		defer q.Close(true)
		var acked atomic.Int64
		el := acker.Counting(func(n int) { acked.Add(int64(n)) })
		listener := &mockClientListener{}
		c := newContextTestClient(q.Producer(queue.ProducerConfig{ACK: el.ACKEvents}), listener, el)
		require.NoError(t, c.PublishContext(t.Context(), beat.Event{}), "the first publish must fill the queue")

		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		errc := make(chan error, 1)
		go func() { errc <- c.PublishContext(ctx, beat.Event{}) }()
		synctest.Wait()
		select {
		case err := <-errc:
			t.Fatalf("PublishContext with an event listener returned %v on a full queue; want it to block like Publish", err)
		default:
		}

		ackBatch(t, q)
		assert.NoError(t, <-errc, "PublishContext must publish once the queue has room")
		ackBatch(t, q)
		synctest.Wait()
		assert.Equal(t, int64(2), acked.Load(), "both events must be ACKed, with none left pending")
		assert.Equal(t, 2, listener.eventsPublished, "both events must be published")
		assert.Equal(t, 0, listener.eventsDroppedOnPublish, "no event must be dropped")
	})
}

// ackBatch reads the queued events and ACKs them.
func ackBatch(t *testing.T, q *slabqueue.Queue[publisher.Event]) {
	t.Helper()
	b, err := q.Get(0)
	require.NoError(t, err, "Get must return the queued events")
	b.Done()
	synctest.Wait()
}

// TestConnectWithCancellablePublish checks that ConnectWith makes publishes
// cancellable only for clients without an event listener.
func TestConnectWithCancellablePublish(t *testing.T) {
	tests := []struct {
		name string
		el   beat.EventListener
		want bool
	}{
		{name: "no event listener", el: nil, want: true},
		{name: "event listener", el: acker.Counting(func(int) {}), want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pool := slabqueue.NewPool[publisher.Event](slabqueue.Settings{Events: 1}, nil)
			p := makePipeline(t, Settings{}, pool.Connect())
			defer func() { _ = p.Disconnect(t.Context()) }()

			bc, err := p.ConnectWith(beat.ClientConfig{EventListener: tc.el})
			require.NoError(t, err, "ConnectWith must succeed")
			defer bc.Close()
			cl, ok := bc.(*client)
			require.True(t, ok, "ConnectWith must return a *client")
			got := cl.ctxProducer != nil
			assert.Equal(t, tc.want, got, "whether the client's publishes can be cancelled")
		})
	}
}

// BenchmarkClientPublish measures the client's own publish overhead, with a
// producer that accepts every event at once.
func BenchmarkClientPublish(b *testing.B) {
	newClient := func() *client {
		return newContextTestClient(&nopProducer{}, noopClientListener{}, nil)
	}
	b.Run("Publish", func(b *testing.B) {
		c := newClient()
		b.ReportAllocs()
		for b.Loop() {
			c.Publish(beat.Event{})
		}
	})
	b.Run("PublishContext/background", func(b *testing.B) {
		c := newClient()
		ctx := context.Background()
		b.ReportAllocs()
		for b.Loop() {
			_ = c.PublishContext(ctx, beat.Event{})
		}
	})
	b.Run("PublishContext/cancellable", func(b *testing.B) {
		c := newClient()
		ctx := b.Context()
		b.ReportAllocs()
		for b.Loop() {
			_ = c.PublishContext(ctx, beat.Event{})
		}
	})
}

// nopProducer accepts every event at once.
type nopProducer struct{}

func (nopProducer) Publish(publisher.Event) (queue.EntryID, bool)    { return 0, true }
func (nopProducer) TryPublish(publisher.Event) (queue.EntryID, bool) { return 0, true }
func (nopProducer) PublishContext(context.Context, publisher.Event) (queue.EntryID, error) {
	return 0, nil
}
func (nopProducer) Close()                       {}
func (nopProducer) ACKWaitChan() <-chan struct{} { return closedChan }
