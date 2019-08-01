// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package bus

import (
	"errors"

	"github.com/elastic/fleet/x-pack/pkg/bus/topic"
	"github.com/elastic/fleet/x-pack/pkg/id"
)

// Common errors returned by the bus.
var (
	ErrTopicExist    = errors.New("topic already exists")
	ErrTopicNotExist = errors.New("topic does not exist")
)

// Event is a common unit of information passed around our system, events are not limited to a single
// type. Subscriber register themself to the bus to receive events from specific topics, a topic is
// also not limited to a single type of event but can accepted events of multiples types.
// It will be up to the subscriber to assert the events into a concrete data type.
type Event interface {
	ID() id.ID
}

// SubscribeFunc defines the callback used by a subscriber to receive events from a specific topic.
type SubscribeFunc func(topic.Topic, Event)

// Bus defines the interface to implement and event bus in our system, the bus is used to decouple
// the different part of our system.
type Bus interface {
	// Subscribe allows a consumer to subscribe to Topic returns an error if we cannot register to the
	// specific topic.
	Subscribe(topic.Topic, SubscribeFunc) error

	// Push pushes an event to a single topic, on success it will return a Tracker that can
	// be used to query information about the delivery of a specific Event or an error if we
	// cannot push the event to the specific topic.
	Push(topic.Topic, Event) (Tracker, error)

	// Start starts the event bus and send events.
	Start()

	// Stop stops the event bus to send events.
	Stop()
}

// Tracker allows to query information about the delivery of a specific event.
type Tracker interface {
	// Wait blocks until the delivery is done.
	Wait() chan struct{}

	// IsDelivered allow to query if the message was delivered for all the consumers.
	IsDelivered() bool
}
