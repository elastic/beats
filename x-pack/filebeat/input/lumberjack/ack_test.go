// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package lumberjack

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/go-lumber/lj"
)

func TestBatchACKTracker(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		batch := lj.NewBatch(nil)

		acker := newBatchACKTracker(batch.ACK)
		require.False(t, isACKed(batch))

		acker.Ready()
		require.True(t, isACKed(batch))
	})

	t.Run("single_event", func(t *testing.T) {
		batch := lj.NewBatch(nil)

		acker := newBatchACKTracker(batch.ACK)
		acker.Add()
		acker.ACK()
		require.False(t, isACKed(batch))

		acker.Ready()
		require.True(t, isACKed(batch))
	})
}

func isACKed(batch *lj.Batch) bool {
	select {
	case <-batch.Await():
		return true
	default:
		return false
	}
}
