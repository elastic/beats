// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/menderesk/beats/v7/libbeat/beat"
)

func TestEventACKTracker(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	acker := NewEventACKTracker(ctx)
	acker.Add()
	acker.ACK()

	assert.EqualValues(t, 0, acker.PendingACKs)
	assert.ErrorIs(t, acker.ctx.Err(), context.Canceled)
}

func TestEventACKTrackerNoACKs(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	acker := NewEventACKTracker(ctx)
	acker.Wait()

	assert.EqualValues(t, 0, acker.PendingACKs)
	assert.ErrorIs(t, acker.ctx.Err(), context.Canceled)
}

func TestEventACKHandler(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	// Create acker. Add one pending ACK.
	acker := NewEventACKTracker(ctx)
	acker.Add()

	// Create an ACK handler and simulate one ACKed event.
	ackHandler := NewEventACKHandler()
	ackHandler.AddEvent(beat.Event{Private: acker}, true)
	ackHandler.ACKEvents(1)

	assert.EqualValues(t, 0, acker.PendingACKs)
	assert.ErrorIs(t, acker.ctx.Err(), context.Canceled)
}

func TestEventACKHandlerWait(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	// Create acker. Add one pending ACK.
	acker := NewEventACKTracker(ctx)
	acker.Add()
	acker.ACK()
	acker.Wait()
	acker.Add()

	assert.EqualValues(t, 1, acker.PendingACKs)
	assert.ErrorIs(t, acker.ctx.Err(), context.Canceled)
}
