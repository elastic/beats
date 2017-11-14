package bus

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEmit(t *testing.T) {
	bus := New("name")
	listener := bus.Subscribe()

	bus.Publish(Event{
		"foo": "bar",
	})

	event := <-listener.Events()
	assert.Equal(t, event["foo"], "bar")
}

func TestEmitOrder(t *testing.T) {
	bus := New("name")
	listener := bus.Subscribe()
	bus.Publish(Event{"first": "event"})
	bus.Publish(Event{"second": "event"})

	event1 := <-listener.Events()
	event2 := <-listener.Events()
	assert.Equal(t, event1, Event{"first": "event"})
	assert.Equal(t, event2, Event{"second": "event"})
}

func TestSubscribeFilter(t *testing.T) {
	bus := New("name")
	listener := bus.Subscribe("second")

	bus.Publish(Event{"first": "event"})
	bus.Publish(Event{"second": "event"})

	event := <-listener.Events()
	assert.Equal(t, event, Event{"second": "event"})
}

func TestMultipleListeners(t *testing.T) {
	bus := New("name")
	listener1 := bus.Subscribe("a")
	listener2 := bus.Subscribe("a", "b")

	bus.Publish(Event{"a": "event"})
	bus.Publish(Event{"a": 1, "b": 2})

	event1 := <-listener1.Events()
	event2 := <-listener1.Events()
	assert.Equal(t, event1, Event{"a": "event"})
	assert.Equal(t, event2, Event{"a": 1, "b": 2})

	event1 = <-listener2.Events()
	assert.Equal(t, event1, Event{"a": 1, "b": 2})

	select {
	case event2 = <-listener2.Events():
		t.Error("Got unexpected event:", event2)
	default:
	}
}

func TestListenerClose(t *testing.T) {
	bus := New("name")
	listener := bus.Subscribe()

	bus.Publish(Event{"first": "event"})
	bus.Publish(Event{"second": "event"})

	listener.Stop()

	bus.Publish(Event{"third": "event"})

	event := <-listener.Events()
	assert.Equal(t, event, Event{"first": "event"})
	event = <-listener.Events()
	assert.Equal(t, event, Event{"second": "event"})

	// Channel was closed, we get an empty event
	event = <-listener.Events()
	assert.Equal(t, event, Event(nil))
}
