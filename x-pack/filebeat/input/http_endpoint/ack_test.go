// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package http_endpoint

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBatchACKTracker(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		tracker := make(ack)

		acker := newBatchACKTracker(tracker.ACK)
		require.False(t, tracker.wasACKed())

		acker.Ready()
		require.True(t, tracker.wasACKed())
	})

	t.Run("single_event", func(t *testing.T) {
		tracker := make(ack)

		acker := newBatchACKTracker(tracker.ACK)
		acker.Add()
		acker.ACK()
		require.False(t, tracker.wasACKed())

		acker.Ready()
		require.True(t, tracker.wasACKed())
	})
}

type ack chan struct{}

func (a ack) ACK() {
	close(a)
}

func (a ack) wasACKed() bool {
	select {
	case <-a:
		return true
	default:
		return false
	}
}
