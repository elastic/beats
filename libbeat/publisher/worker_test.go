// +build !integration

package publisher

import (
	"sync/atomic"
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/stretchr/testify/assert"
)

// Test sending events through the messageWorker.
func TestMessageWorkerSend(t *testing.T) {
	enableLogging([]string{"*"})

	// Setup
	ws := common.NewWorkerSignal()
	mh := &testMessageHandler{msgs: make(chan message, 10), response: true}
	mw := newMessageWorker(ws, 10, 0, mh)

	// Send an event.
	s1 := newTestSignaler()
	m1 := message{context: Context{Signal: s1}}
	mw.send(m1)

	// Send another event.
	s2 := newTestSignaler()
	m2 := message{context: Context{Signal: s2}}
	mw.send(m2)

	// Verify that the messageWorker pushed the two messages to the
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

	ws.Stop()
	assert.True(t, atomic.LoadUint32(&mh.stopped) == 1)
}

// Test that events sent before shutdown are pushed to the messageHandler.
func TestMessageWorkerShutdownSend(t *testing.T) {
	enableLogging([]string{"*"})

	// Setup
	ws := common.NewWorkerSignal()
	mh := &testMessageHandler{msgs: make(chan message, 10), response: true}
	mw := newMessageWorker(ws, 10, 0, mh)

	// Send an event.
	s1 := newTestSignaler()
	m1 := message{context: Context{Signal: s1}}
	mw.send(m1)

	// Send another event.
	s2 := newTestSignaler()
	m2 := message{context: Context{Signal: s2}}
	mw.send(m2)

	ws.Stop()
	assert.True(t, atomic.LoadUint32(&mh.stopped) == 1)

	// Verify that the messageWorker pushed the two messages to the
	// messageHandler.
	close(mh.msgs)
	assert.Equal(t, 2, len(mh.msgs))

	// Verify the messages and the signals.
	assert.Equal(t, <-mh.msgs, m1)
	assert.True(t, s1.wait())
	assert.Equal(t, <-mh.msgs, m2)
	assert.True(t, s2.wait())
}
