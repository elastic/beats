// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package bus

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/fleet/x-pack/pkg/bus/topic"

	"github.com/elastic/fleet/x-pack/pkg/id"
)

const aTopic topic.Topic = "A_TOPIC"

type simpleEvent struct {
	message string
	id      id.ID
}

func (s *simpleEvent) ID() id.ID {
	return s.id
}

func newSimpleEvent(message string) *simpleEvent {
	id, err := id.Generate()
	if err != nil {
		panic(err)
	}
	return &simpleEvent{message: message, id: id}
}

func TestEventBus(t *testing.T) {
	testCreateTopic(t)
	testPushToTopic(t)
	testSubscribers(t)
}

func testCreateTopic(t *testing.T) {
	t.Run("Successfully create the topic if it doesn't exist", withBus(t, func(
		t *testing.T,
		bus *EventBus,
	) {
		require.NoError(t, bus.CreateTopic(aTopic))
	}))

	t.Run("Return an error if the topic already exist", withBus(t, func(t *testing.T, bus *EventBus) {
		require.NoError(t, bus.CreateTopic(aTopic))
		require.Error(t, bus.CreateTopic(aTopic))
	}))
}

func testPushToTopic(t *testing.T) {
	t.Run("Returns an error if the topic doesnt exist", withBus(t, func(t *testing.T, bus *EventBus) {
		_, err := bus.Push(aTopic, newSimpleEvent(""))
		require.Error(t, err)
	}))
}

func testSubscribers(t *testing.T) {
	t.Run("Push to a topic with a single subscriber", withBus(t, func(t *testing.T, bus *EventBus) {
		event := newSimpleEvent("hello subscriber")

		var wg sync.WaitGroup
		wg.Add(1)

		require.NoError(t, bus.CreateTopic(aTopic))
		bus.Subscribe(aTopic, func(topic topic.Topic, e Event) {
			assert.Equal(t, aTopic, topic)
			ne := e.(*simpleEvent)
			assert.Equal(t, event.message, ne.message)
			wg.Done()
		})

		_, err := bus.Push(aTopic, event)
		require.NoError(t, err)
		wg.Wait()
	}))

	t.Run("Push to a topic with a multiple subscribers", withBus(t, func(t *testing.T, bus *EventBus) {
		event := newSimpleEvent("hello subscriber")

		var wg sync.WaitGroup
		wg.Add(2)

		handler := func(topic topic.Topic, e Event) {
			assert.Equal(t, aTopic, topic)
			ne := e.(*simpleEvent)
			assert.Equal(t, event.message, ne.message)
			wg.Done()
		}

		require.NoError(t, bus.CreateTopic(aTopic))
		bus.Subscribe(aTopic, handler)
		bus.Subscribe(aTopic, handler)

		_, err := bus.Push(aTopic, event)
		require.NoError(t, err)
		wg.Wait()
	}))

	t.Run("Push a message using the wildcard topic send to all known subscribers", withBus(t, func(
		t *testing.T,
		bus *EventBus,
	) {
		event := newSimpleEvent("hello subscriber")

		var wg sync.WaitGroup
		wg.Add(2)

		handler := func(tt topic.Topic, e Event) {
			assert.Equal(t, topic.AllSubscribers, tt)
			ne := e.(*simpleEvent)
			assert.Equal(t, event.message, ne.message)
			wg.Done()
		}

		other := topic.Topic("other")
		require.NoError(t, bus.CreateTopic(aTopic))
		require.NoError(t, bus.CreateTopic(other))

		bus.Subscribe(aTopic, handler)
		bus.Subscribe(other, handler)

		_, err := bus.Push(topic.AllSubscribers, event)
		require.NoError(t, err)
		wg.Wait()
	}))

	t.Run("Push a message which should be received only by a subset of subscribers", withBus(t, func(
		t *testing.T,
		bus *EventBus,
	) {
		event := newSimpleEvent("hello subscriber")

		var wg sync.WaitGroup
		wg.Add(1)

		handler := func(topic topic.Topic, e Event) {
			assert.Equal(t, aTopic, topic)
			ne := e.(*simpleEvent)
			assert.Equal(t, event.message, ne.message)
			wg.Done()
		}

		other := topic.Topic("other")
		require.NoError(t, bus.CreateTopic(aTopic))
		require.NoError(t, bus.CreateTopic(other))

		bus.Subscribe(aTopic, handler)
		bus.Subscribe(other, handler)

		_, err := bus.Push(aTopic, event)
		require.NoError(t, err)
		wg.Wait()
	}))
}

func withBus(t *testing.T, fn func(t *testing.T, bus *EventBus)) func(t *testing.T) {
	bus, err := NewEventBus(nil)
	require.NoError(t, err)
	return func(t *testing.T) {
		bus.Start()
		defer bus.Stop()
		fn(t, bus)
	}
}
