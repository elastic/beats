// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kvstore

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
)

func TestTxTracker_Add(t *testing.T) {
	txTracker := NewTxTracker(context.Background())

	assert.Equal(t, int64(0), txTracker.count(), "new tracker should have nothing pending")
	txTracker.Add()
	assert.Equal(t, int64(1), txTracker.count(), "Add should increment pending")
}

func TestTxTracker_Ack(t *testing.T) {
	t.Run("zero_before_wait", func(t *testing.T) {
		txTracker := NewTxTracker(context.Background())
		txTracker.Add()
		txTracker.Ack()
		assert.Equal(t, int64(0), txTracker.count(), "Ack should decrement pending")
		assert.False(t, released(txTracker), "Ack released Wait before Wait was called")
	})

	t.Run("zero_after_wait", func(t *testing.T) {
		txTracker := NewTxTracker(context.Background())
		txTracker.Add()
		done := waitAsync(txTracker)
		assert.True(t, blocked(txTracker, done), "Wait did not block with an event pending")

		txTracker.Ack()
		assert.True(t, released(txTracker), "the last Ack did not release Wait")
		assert.True(t, returned(done, time.Second), "Wait did not return after the last ACK")
	})
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
		assert.True(t, blocked(txTracker, done), "Wait did not block with an event pending")

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
		assert.True(t, blocked(txTracker, done), "Wait did not block with an event pending")

		txTracker.Ack()
		assert.True(t, returned(done, time.Second), "Wait did not return after the last ACK")
	})

	t.Run("wait_twice", func(t *testing.T) {
		txTracker := NewTxTracker(context.Background())
		txTracker.Add()
		txTracker.Ack()
		assert.True(t, returned(waitAsync(txTracker), time.Second), "first Wait did not return")
		assert.True(t, returned(waitAsync(txTracker), time.Second), "second Wait did not return")
	})

	t.Run("cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		txTracker := NewTxTracker(ctx)
		txTracker.Add()

		done := waitAsync(txTracker)
		cancel()
		assert.True(t, returned(done, time.Second), "Wait did not return after cancellation")
		assert.Equal(t, int64(1), txTracker.count(), "cancellation should not change pending")
	})
}

func TestTxACKHandler(t *testing.T) {
	t.Run("all_ack", func(t *testing.T) {
		txTracker := NewTxTracker(context.Background())
		handler := NewTxACKHandler()

		txTracker.Add()
		assert.Equal(t, int64(1), txTracker.count(), "Add should increment pending")

		handler.AddEvent(beat.Event{
			Private: txTracker,
		}, true)
		handler.ACKEvents(1)

		txTracker.Wait()

		assert.Zero(t, txTracker.count(), "all events should be ACK-ed after Wait")
	})

	t.Run("wait_ack", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		txTracker := NewTxTracker(ctx)
		handler := NewTxACKHandler()

		txTracker.Add()
		assert.Equal(t, int64(1), txTracker.count(), "Add should increment pending")

		handler.AddEvent(beat.Event{
			Private: txTracker,
		}, true)

		txTracker.Wait()

		assert.Equal(t, int64(1), txTracker.count(), "the event should still be pending after a cancelled Wait")
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
		assert.True(t, blocked(txTracker, done), "Wait did not block with an event pending")

		handler.ACKEvents(1)
		assert.True(t, returned(done, time.Second), "Wait did not return after the last ACK")
	})
}

// count returns t's pending count.
func (t *TxTracker) count() int64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.pending
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

// blocked reports whether a Wait call on t has started, has not been
// released, and has not returned on done, the channel from waitAsync.
// It gives Wait up to a second to start. Ack and Wait release while
// holding the lock, so once Wait has started the release check does
// not depend on scheduling.
func blocked(t *TxTracker, done <-chan struct{}) bool {
	deadline := time.Now().Add(time.Second)
	for {
		t.mu.Lock()
		waiting := t.waiting
		t.mu.Unlock()
		if waiting {
			return !released(t) && !isClosed(done)
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(time.Millisecond)
	}
}

// released reports whether t has released Wait.
func released(t *TxTracker) bool {
	return isClosed(t.done)
}

// isClosed reports whether c is closed, without blocking.
func isClosed(c <-chan struct{}) bool {
	select {
	case _, ok := <-c:
		return !ok
	default:
		return false
	}
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
