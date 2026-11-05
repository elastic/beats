// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kvstore

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
)

func TestTxTracker_Add(t *testing.T) {
	txTracker := NewTxTracker(context.Background())

	require.Equal(t, int64(0), txTracker.pending, "new tracker should have nothing pending")
	txTracker.Add()
	require.Equal(t, int64(1), txTracker.pending, "Add should increment pending")
}

func TestTxTracker_Wait(t *testing.T) {
	t.Run("nothing_pending", func(t *testing.T) {
		txTracker := NewTxTracker(context.Background())
		assert.True(t, returned(waitAsync(txTracker), time.Second), "Wait with nothing pending should return")
	})

	t.Run("ack_after_wait", func(t *testing.T) {
		txTracker := NewTxTracker(context.Background())
		txTracker.Add()

		done := waitAsync(txTracker)
		assert.False(t, returned(done, 50*time.Millisecond), "Wait returned with an event pending")

		txTracker.Ack()
		assert.True(t, returned(done, time.Second), "Wait did not return after the last ACK")
	})

	// The output can acknowledge every event published so far while
	// publishing is still going on. That must not release a later Wait.
	t.Run("zero_before_last_add", func(t *testing.T) {
		txTracker := NewTxTracker(context.Background())
		txTracker.Add()
		txTracker.Ack()
		txTracker.Add()
		txTracker.Add()
		txTracker.Ack()

		done := waitAsync(txTracker)
		assert.False(t, returned(done, 50*time.Millisecond), "Wait returned with an event pending")

		txTracker.Ack()
		assert.True(t, returned(done, time.Second), "Wait did not return after the last ACK")
	})

	t.Run("cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		txTracker := NewTxTracker(ctx)
		txTracker.Add()

		done := waitAsync(txTracker)
		cancel()
		assert.True(t, returned(done, time.Second), "Wait did not return after cancellation")
		assert.Equal(t, int64(1), txTracker.pending, "cancellation should not change pending")
	})
}

func TestTxACKHandler(t *testing.T) {
	t.Run("all-ack", func(t *testing.T) {
		txTracker := NewTxTracker(context.Background())
		handler := NewTxACKHandler()

		txTracker.Add()
		require.Equal(t, int64(1), txTracker.pending, "Add should increment pending")

		handler.AddEvent(beat.Event{
			Private: txTracker,
		}, true)
		handler.ACKEvents(1)

		txTracker.Wait()

		require.Zero(t, txTracker.pending, "all events should be ACK-ed after Wait")
	})

	t.Run("wait-ack", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		txTracker := NewTxTracker(ctx)
		handler := NewTxACKHandler()

		txTracker.Add()
		require.Equal(t, int64(1), txTracker.pending, "Add should increment pending")

		handler.AddEvent(beat.Event{
			Private: txTracker,
		}, true)

		txTracker.Wait()

		require.Equal(t, int64(1), txTracker.pending, "the event should still be pending after a cancelled Wait")
	})

	// ACKs arrive from the pipeline while the sync is still publishing.
	t.Run("ack_during_publish", func(t *testing.T) {
		txTracker := NewTxTracker(context.Background())
		handler := NewTxACKHandler()

		publish := func(n int) {
			for range n {
				txTracker.Add()
				handler.AddEvent(beat.Event{Private: txTracker}, true)
			}
		}
		publish(3)
		handler.ACKEvents(3)
		publish(2)
		handler.ACKEvents(1)

		done := waitAsync(txTracker)
		assert.False(t, returned(done, 50*time.Millisecond), "Wait returned with an event pending")

		handler.ACKEvents(1)
		assert.True(t, returned(done, time.Second), "Wait did not return after the last ACK")
	})
}

// waitAsync calls t.Wait in a new goroutine and returns a channel that is
// closed when Wait returns.
func waitAsync(t *TxTracker) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		t.Wait()
		close(done)
	}()
	return done
}

// returned reports whether done is closed within d.
func returned(done <-chan struct{}, d time.Duration) bool {
	select {
	case <-done:
		return true
	case <-time.After(d):
		return false
	}
}
