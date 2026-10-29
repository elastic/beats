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

package diskqueue

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/beats/v7/libbeat/publisher/queue/queuetest"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/paths"
)

// TestACKWaitClosesOnClose verifies the disk queue producer's ack-wait channel
// is open until Close and closed afterward. The disk queue persists events
// durably and does not track in-memory acknowledgments, so Close is the only
// signal to wait for, and repeated Close calls must remain safe.
func TestACKWaitClosesOnClose(t *testing.T) {
	p := &diskQueueProducer{
		done:    make(chan struct{}),
		ackWait: make(chan struct{}),
	}

	select {
	case <-p.ACKWaitChan():
		t.Fatal("ackWait must be open before Close")
	default:
	}

	p.Close()

	select {
	case <-p.ACKWaitChan():
	default:
		t.Fatal("ackWait must be closed after Close")
	}

	// Close is idempotent: a second call must not panic on the already-closed
	// channels.
	p.Close()
}

// TestPublishContextCancelWhileBlocked checks that cancelling a PublishContext
// blocked on a full queue returns the ctx error, and that the event is never
// added, even once the queue has room again.
func TestPublishContextCancelWhileBlocked(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		reg := monitoring.NewRegistry()
		dq, p := newProducerTestQueue(t, reg)
		accepted := fillQueue(t, p)

		ctx, cancel := context.WithCancel(t.Context())
		errc := make(chan error, 1)
		go func() {
			_, err := p.PublishContext(ctx, makeLargeTestEvent("cancelled"))
			errc <- err
		}()
		synctest.Wait()
		select {
		case err := <-errc:
			t.Fatalf("PublishContext returned %v on a full queue; want it to block", err)
		default:
		}

		cancel()
		synctest.Wait()
		select {
		case err := <-errc:
			assert.ErrorIs(t, err, context.Canceled, "a cancelled PublishContext should return the ctx error")
		default:
			t.Fatal("PublishContext should return once its ctx is cancelled")
		}

		// Free the queue: the withdrawn event must not be added now.
		consumed := 0
		for consumed < accepted {
			b, err := dq.Get(0)
			require.NoError(t, err, "reading the accepted events should succeed")
			consumed += b.Count()
			b.Done()
		}
		synctest.Wait()
		assertRegistryUint(t, reg, "queue.added.events", uint64(accepted), "the cancelled event should never be added") //nolint:gosec // accepted is a small positive count

		p.Close()
		closeQueueAndWait(t, dq)
	})
}

// TestPublishContextQueueClosedWhileBlocked checks that closing the queue
// releases a PublishContext blocked on it with queue.ErrClosed.
func TestPublishContextQueueClosedWhileBlocked(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dq, p := newProducerTestQueue(t, nil)
		fillQueue(t, p)

		errc := make(chan error, 1)
		go func() {
			_, err := p.PublishContext(t.Context(), makeLargeTestEvent("blocked"))
			errc <- err
		}()
		synctest.Wait()
		select {
		case err := <-errc:
			t.Fatalf("PublishContext returned %v on a full queue; want it to block", err)
		default:
		}

		closeQueueAndWait(t, dq)
		assert.ErrorIs(t, <-errc, queue.ErrClosed, "closing the queue should release a blocked PublishContext with ErrClosed")
		p.Close()
	})
}

// TestPublishContextRejected checks that an event the queue can never take
// is reported as queue.ErrRejected, with the reason.
func TestPublishContextRejected(t *testing.T) {
	tests := []struct {
		name  string
		event publisher.Event
		// reason matches the error message.
		reason string
	}{
		{
			name:   "too large",
			event:  makeLargeTestEvent(strings.Repeat("x", 32*1024)),
			reason: `event size \d+ exceeds the segment buffer limit ` + strconv.Itoa(16*1024-segmentHeaderSize),
		},
		{
			// The CBOR encoder cannot serialize a channel.
			name:   "not serializable",
			event:  queuetest.MakeEvent(mapstr.M{"ch": make(chan int)}),
			reason: `serializing event`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				dq, p := newProducerTestQueue(t, nil)

				_, err := p.PublishContext(t.Context(), tc.event)
				require.ErrorIs(t, err, queue.ErrRejected, "an event the queue cannot take should be rejected")
				assert.Regexp(t, tc.reason, err.Error(), "the error should give the reason")
				_, ok := p.Publish(tc.event)
				assert.False(t, ok, "Publish should refuse the event too")

				p.Close()
				closeQueueAndWait(t, dq)
			})
		})
	}
}

// TestPublishContextCancelAnswer checks what PublishContext returns when its
// ctx ends while the core loop holds the request, for each answer the core
// loop can give after the producer asks it to withdraw the request. A stub
// stands in for the core loop so the order is fixed.
func TestPublishContextCancelAnswer(t *testing.T) {
	tests := []struct {
		name   string
		answer error
		want   error
	}{
		{name: "added before the cancel", answer: nil, want: nil},
		{name: "withdrawn", answer: errCancelled, want: context.Canceled},
		{name: "queue closed", answer: queue.ErrClosed, want: queue.ErrClosed},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dq := &diskQueue{
				producerWriteRequestChan: make(chan producerWriteRequest),
				producerCancelChan:       make(chan chan error),
				close:                    make(chan struct{}),
			}
			p := &diskQueueProducer{
				queue:   dq,
				encoder: newEventEncoder(SerializationCBOR),
				done:    make(chan struct{}),
			}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			go func() {
				<-dq.producerWriteRequestChan
				cancel()
				responseChan := <-dq.producerCancelChan
				responseChan <- tc.answer
			}()

			_, err := p.PublishContext(ctx, makeLargeTestEvent("event"))
			if tc.want == nil {
				assert.NoError(t, err, "an event the core loop added should be reported as published despite the cancelled ctx")
			} else {
				assert.ErrorIs(t, err, tc.want, "PublishContext should report the core loop's answer")
			}
		})
	}
}

// newProducerTestQueue returns a disk queue in a temporary directory, with room
// for about a dozen events from makeLargeTestEvent and reporting to reg if it
// is not nil, and a producer for it.
func newProducerTestQueue(t *testing.T, reg *monitoring.Registry) (*diskQueue, *diskQueueProducer) {
	t.Helper()
	settings := DefaultSettings()
	settings.Path = t.TempDir()
	settings.MaxBufferSize = 64 * 1024
	settings.MaxSegmentSize = 16 * 1024
	dq, err := NewQueue(logptest.NewTestingLogger(t, ""), queue.NewQueueObserver(reg), settings, nil, &paths.Path{})
	require.NoError(t, err, "the disk queue should be created")
	p, ok := dq.Producer(queue.ProducerConfig{}).(*diskQueueProducer)
	require.True(t, ok, "the disk queue should return a *diskQueueProducer")
	return dq, p
}

// fillQueue publishes events with TryPublish until the queue is full, and
// returns how many it accepted.
func fillQueue(t *testing.T, p *diskQueueProducer) int {
	t.Helper()
	n := 0
	for {
		_, ok := p.TryPublish(makeLargeTestEvent(strconv.Itoa(n)))
		if !ok {
			return n
		}
		n++
		require.Less(t, n, 100, "the disk queue should fill up")
	}
}

func makeLargeTestEvent(msg string) publisher.Event {
	return queuetest.MakeEvent(mapstr.M{
		"message": msg,
		"payload": strings.Repeat("x", 4*1024),
	})
}
