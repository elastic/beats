package publisher

import (
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test sending events through the messageWorker.
func TestMessageWorkerSend(t *testing.T) {
	// Setup
	ws := &workerSignal{}
	ws.Init()
	mh := &testMessageHandler{msgs: make(chan message, 10), response: true}
	mw := newMessageWorker(ws, 10, mh)

	// Send an event.
	s1 := newTestSignaler()
	m1 := message{context: context{signal: s1}}
	mw.send(m1)

	// Send another event.
	s2 := newTestSignaler()
	m2 := message{context: context{signal: s2}}
	mw.send(m2)

	// Verify that the messageWorker pushed to two messages to the
	// messageHandler.
	msgs, err := mh.waitForMessages(2)
	if err != nil {
		t.Fatal(err)
	}

	// Verify the messages and the signals.
	assert.Contains(t, msgs, m1)
	assert.True(t, s1.wait())
	assert.Contains(t, msgs, m2)
	assert.True(t, s2.wait())

	// Verify that stopping workerSignal causes a onStop notification
	// in the messageHandler.
	ws.stop()
	assert.True(t, atomic.LoadUint32(&mh.stopped) == 1)
}

// Test that stopQueue invokes the Failed callback on all events in the queue.
func TestMessageWorkerStopQueue(t *testing.T) {
	s1 := newTestSignaler()
	m1 := message{context: context{signal: s1}}

	s2 := newTestSignaler()
	m2 := message{context: context{signal: s2}}

	qu := make(chan message, 2)
	qu <- m1
	qu <- m2

	stopQueue(qu)
	assert.False(t, s1.wait())
	assert.False(t, s2.wait())
}
