// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package bus

import (
	"fmt"
	"sync"
	"time"

	"github.com/elastic/fleet/x-pack/pkg/bus/topic"
	"github.com/elastic/fleet/x-pack/pkg/core/logger"
)

type subscribers []SubscribeFunc

// EventBus is a minimal implementation of the bus just enough to validate that our architecture
// correctly works.
type EventBus struct {
	log         *logger.Logger
	subscribers map[topic.Topic]subscribers
	queue       chan *ticket
	wg          sync.WaitGroup
	sync.RWMutex
	backlog []*ticket
}

type ticket struct {
	topic     topic.Topic
	event     Event
	delivered bool
	done      chan struct{}
}

func (t *ticket) String() string {
	return fmt.Sprintf("ticket for topic %s, event is %+v", t.topic, t.event)
}

func (t *ticket) Wait() chan struct{} {
	return t.done
}

func (t *ticket) IsDelivered() bool {
	select {
	case <-t.done:
		return true
	default:
	}

	return false
}

func (t *ticket) ack() {
	close(t.done)
}

func newTicket(topic topic.Topic, event Event) *ticket {
	return &ticket{topic: topic, event: event, done: make(chan struct{})}
}

// NewEventBus returns a new events bus.
func NewEventBus(log *logger.Logger) (*EventBus, error) {
	var err error
	if log == nil {
		log, err = logger.New()
		if err != nil {
			return nil, err
		}
	}

	return &EventBus{
		log:         log,
		subscribers: make(map[topic.Topic]subscribers),
		queue:       make(chan *ticket),
	}, nil
}

// CreateTopic creates a new topic on the event bus, topics are statics and must exist before
// registering a new subscriber or pushing new events to the bus.
func (e *EventBus) CreateTopic(topic topic.Topic) error {
	_, ok := e.subscribers[topic]
	if ok {
		return ErrTopicExist
	}
	e.subscribers[topic] = make(subscribers, 0)
	return nil
}

// Subscribe allow a subscriber to register itself to a specific topic.
func (e *EventBus) Subscribe(topic topic.Topic, subscriber SubscribeFunc) error {
	e.Lock()
	defer e.Unlock()

	_, ok := e.subscribers[topic]
	if !ok {
		return ErrTopicNotExist
	}

	e.log.Debugf("Adding new subscriber for topic %s", topic)
	e.subscribers[topic] = append(e.subscribers[topic], subscriber)
	return nil
}

// Push a new event on the bus for a specific topic.
func (e *EventBus) Push(t topic.Topic, event Event) (Tracker, error) {
	e.RLock()
	defer e.RUnlock()

	if t != topic.AllSubscribers {
		_, ok := e.subscribers[t]

		if !ok {
			return nil, ErrTopicNotExist
		}
	}

	ticket := newTicket(t, event)
	go func() {
		select {
		case e.queue <- ticket:
		case <-time.After(5 * time.Minute): // guards agains stuck goroutines
			e.log.Errorf("Failed to send event %+v into '%v'", event, t)
		}
	}()

	return ticket, nil
}

// Start starts the event bus, events pushed to the bus will be distributed to the topics.
func (e *EventBus) Start() {
	e.log.Debug("EventBus is starting")
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		defer e.log.Debug("EventBus is stopped")
		e.worker()
	}()
}

// Stop stops the events bus.
func (e *EventBus) Stop() {
	close(e.queue)
	e.wg.Wait()
}

func (e *EventBus) worker() {
	for t := range e.queue {
		e.RLock()
		var receivedCount int
		if t.topic == topic.AllSubscribers {
			for _, subs := range e.subscribers {
				e.notify(subs, t)
				receivedCount += len(subs)
			}
		} else {
			subs, ok := e.subscribers[t.topic]
			invariant(!ok, fmt.Sprintf("Could not retrieve subscribers for topic %s", t.topic))
			receivedCount += len(subs)
			e.notify(subs, t)
		}

		if receivedCount == 0 {
			e.log.Errorf("Discarding event %+v for topic %s", t.event, t.topic)
		} else {
			t.ack()
		}
		e.RUnlock()
	}
}

func (e *EventBus) notify(subs subscribers, t *ticket) {
	e.log.Debugf("EventBus: pushing %v event %v to %d subscribers.", t.topic, t.event.ID(), len(subs))
	for _, subscriber := range subs {
		subscriber(t.topic, t.event)
	}
}

func invariant(check bool, msg string) {
	if check {
		panic(msg)
	}
}
