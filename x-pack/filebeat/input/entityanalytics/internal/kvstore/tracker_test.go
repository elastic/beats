// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kvstore

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
)

func TestTxTracker_Ack(t *testing.T) {
	txTracker := NewTxTracker(context.Background())
	txTracker.pending.Inc()

	txTracker.Ack()

	require.ErrorIs(t, txTracker.ctx.Err(), context.Canceled)
}

func TestTxTracker_Add(t *testing.T) {
	txTracker := NewTxTracker(context.Background())

	require.Equal(t, 0, txTracker.pending.Load())
	txTracker.Add()
	require.Equal(t, 1, txTracker.pending.Load())
}

func TestTxTracker_Wait(t *testing.T) {
	txTracker := NewTxTracker(context.Background())
	txTracker.Wait()

	require.ErrorIs(t, txTracker.ctx.Err(), context.Canceled)
}

func TestTxACKHandler(t *testing.T) {
	t.Run("all-ack", func(t *testing.T) {
		txTracker := NewTxTracker(context.Background())
		handler := NewTxACKHandler()

		txTracker.Add()
		require.Equal(t, 1, txTracker.pending.Load())

		handler.AddEvent(beat.Event{
			Private: txTracker,
		}, true)
		handler.ACKEvents(1)

		txTracker.Wait()

		require.Zero(t, txTracker.pending.Load())
	})

	t.Run("wait-ack", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		txTracker := NewTxTracker(ctx)
		handler := NewTxACKHandler()

		txTracker.Add()
		require.Equal(t, 1, txTracker.pending.Load())

		handler.AddEvent(beat.Event{
			Private: txTracker,
		}, true)

		txTracker.Wait()

		require.Equal(t, 1, txTracker.pending.Load())
	})
}
