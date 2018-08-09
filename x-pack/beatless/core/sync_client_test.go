// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package core

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common/atomic"
)

type simpleClient struct {
	EventCount atomic.Int
	Received   chan struct{}
}

func (sc *simpleClient) Publish(event beat.Event) {
	defer close(sc.Received)
	sc.EventCount.Inc()
}

func (sc *simpleClient) PublishAll(events []beat.Event) {
	defer close(sc.Received)
	sc.EventCount.Add(len(events))
}

func newSimpleClient() *simpleClient {
	return &simpleClient{Received: make(chan struct{})}
}

func (sc simpleClient) Close() error { return nil }

func TestSyncClient(t *testing.T) {
	t.Run("Publish", func(t *testing.T) {
		e := beat.Event{}
		events := []interface{}{e}

		c := newSimpleClient()
		sc := NewSyncClient(c)

		// The go routine will be blocked until all the events
		// are acked.
		unblocked := make(chan struct{})
		go func(t *testing.T, e beat.Event) {
			defer close(unblocked)
			err := sc.Publish(context.Background(), e)
			assert.NoError(t, err)
		}(t, e)

		// Wait to receive the events before Acking all of them.
		select {
		case <-c.Received:
			sc.AckEvents(events)
		}

		assert.Equal(t, 1, c.EventCount.Load())

		// Make sure we the goroutine is not blocked.
		select {
		case <-unblocked:
			return
		}
	})

	t.Run("PublishAll single ACK", func(t *testing.T) {
		events := []beat.Event{beat.Event{}, beat.Event{}}

		c := newSimpleClient()
		sc := NewSyncClient(c)

		// The go routine will be blocked until all the events
		// are acked.
		unblocked := make(chan struct{})
		go func(t *testing.T, events []beat.Event) {
			defer close(unblocked)
			err := sc.PublishAll(context.Background(), events)
			assert.NoError(t, err)
		}(t, events)

		// Wait to receive the events before Acking all of them.
		select {
		case <-c.Received:
			// Event instance doesn't need to match since the multiplexer already takes care
			// of routing the correct events.
			sc.AckEvents([]interface{}{beat.Event{}, beat.Event{}})
		}

		assert.Equal(t, 2, c.EventCount.Load())

		// Make sure we the goroutine is not blocked.
		select {
		case <-unblocked:
			return
		}
	})

	t.Run("PublishAll multiple ACKs", func(t *testing.T) {
		events := []beat.Event{beat.Event{}, beat.Event{}}

		c := newSimpleClient()
		sc := NewSyncClient(c)

		// The go routine will be blocked until all the events
		// are acked.
		unblocked := make(chan struct{})
		go func(t *testing.T, events []beat.Event) {
			defer close(unblocked)
			err := sc.PublishAll(context.Background(), events)
			assert.NoError(t, err)
		}(t, events)

		// Wait to receive the events before Acking all of them.
		select {
		case <-c.Received:
			// Event instance doesn't need to match since the multiplexer already takes care
			// of routing the correct events.
			sc.AckEvents([]interface{}{beat.Event{}})
			sc.AckEvents([]interface{}{beat.Event{}})
		}

		assert.Equal(t, 2, c.EventCount.Load())

		// Make sure we the goroutine is not blocked.
		select {
		case <-unblocked:
			return
		}
	})
}
