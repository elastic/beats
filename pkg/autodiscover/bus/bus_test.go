// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//nolint:forbidigo // tests verify bus behavior with the global logger
package bus

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/logp"
)

func TestEmit(t *testing.T) {
	bus := New(logp.L(), "name")
	listener := bus.Subscribe()

	bus.Publish(Event{
		"foo": "bar",
	})

	event := <-listener.Events()
	assert.Equal(t, "bar", event["foo"])
}

func TestEmitOrder(t *testing.T) {
	bus := New(logp.L(), "name")
	listener := bus.Subscribe()
	bus.Publish(Event{"first": "event"})
	bus.Publish(Event{"second": "event"})

	event1 := <-listener.Events()
	event2 := <-listener.Events()
	assert.Equal(t, Event{"first": "event"}, event1)
	assert.Equal(t, Event{"second": "event"}, event2)
}

func TestSubscribeFilter(t *testing.T) {
	bus := New(logp.L(), "name")
	listener := bus.Subscribe("second")

	bus.Publish(Event{"first": "event"})
	bus.Publish(Event{"second": "event"})

	event := <-listener.Events()
	assert.Equal(t, Event{"second": "event"}, event)
}

func TestMultipleListeners(t *testing.T) {
	bus := New(logp.L(), "name")
	listener1 := bus.Subscribe("a")
	listener2 := bus.Subscribe("a", "b")

	bus.Publish(Event{"a": "event"})
	bus.Publish(Event{"a": 1, "b": 2})

	event1 := <-listener1.Events()
	event2 := <-listener1.Events()
	assert.Equal(t, Event{"a": "event"}, event1)
	assert.Equal(t, Event{"a": 1, "b": 2}, event2)

	event1 = <-listener2.Events()
	assert.Equal(t, Event{"a": 1, "b": 2}, event1)

	select {
	case event2 = <-listener2.Events():
		t.Error("Got unexpected event:", event2)
	default:
	}
}

func TestListenerClose(t *testing.T) {
	bus := New(logp.L(), "name")
	listener := bus.Subscribe()

	bus.Publish(Event{"first": "event"})
	bus.Publish(Event{"second": "event"})

	listener.Stop()

	bus.Publish(Event{"third": "event"})

	event := <-listener.Events()
	assert.Equal(t, Event{"first": "event"}, event)
	event = <-listener.Events()
	assert.Equal(t, Event{"second": "event"}, event)

	// Channel was closed, we get an empty event
	event = <-listener.Events()
	assert.Equal(t, event, Event(nil))
}

func TestUnsubscribedBus(t *testing.T) {
	bus := NewBusWithStore(logp.L(), "name", 2)
	bus.Publish(Event{"first": "event"})

	listener := bus.Subscribe()
	bus.Publish(Event{"second": "event"})
	event := <-listener.Events()
	event1 := <-listener.Events()
	assert.Equal(t, Event{"first": "event"}, event)
	assert.Equal(t, Event{"second": "event"}, event1)

	bus.Publish(Event{"a": 1, "b": 2})
	event2 := <-listener.Events()
	assert.Equal(t, Event{"a": 1, "b": 2}, event2)
}
