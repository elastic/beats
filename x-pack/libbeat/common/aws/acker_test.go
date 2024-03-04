// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
)

func TestEventACKTracker(t *testing.T) {
	acker := NewEventACKTracker()
	acker.Add()
	acker.ACK()

	assert.EqualValues(t, 0, acker.PendingACKs.Load())
}

func TestEventACKTrackerNoACKs(t *testing.T) {
	acker := NewEventACKTracker()
	acker.Wait()

	assert.EqualValues(t, 0, acker.PendingACKs.Load())
}

func TestEventACKHandler(t *testing.T) {
	// Create acker. Add one pending ACK.
	acker := NewEventACKTracker()
	acker.Add()

	// Create an ACK handler and simulate one ACKed event.
	ackHandler := NewEventACKHandler()
	ackHandler.AddEvent(beat.Event{Private: acker}, true)
	ackHandler.ACKEvents(1)

	assert.EqualValues(t, 0, acker.PendingACKs.Load())
}

func TestEventACKHandlerWait(t *testing.T) {
	// Create acker. Add one pending ACK.
	acker := NewEventACKTracker()
	acker.Add()
	acker.ACK()
	acker.Wait()
	acker.Add()

	assert.EqualValues(t, 1, acker.PendingACKs.Load())
}
