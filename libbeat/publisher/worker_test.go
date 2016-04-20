// +build !integration

package publisher

import (
	"sync/atomic"
	"testing"

	"github.com/elastic/beats/libbeat/common/op"
	"github.com/stretchr/testify/assert"
)

// Test sending events through the messageWorker.
func TestMessageWorkerSend(t *testing.T) {
	enableLogging([]string{"*"})

	client := &client{canceler: op.NewCanceler()}

	// Setup
	ws := newWorkerSignal()
	mh := &testMessageHandler{msgs: make(chan message, 10), response: true}
	mw := newMessageWorker(ws, 10, 0, mh)

	// Send an event.
	s1 := newTestSignaler()
	m1 := message{client: client, context: Context{Signal: s1}}
	mw.send(m1)

	// Send another event.
	s2 := newTestSignaler()
	m2 := message{client: client, context: Context{Signal: s2}}
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

	ws.stop()
	assert.True(t, atomic.LoadUint32(&mh.stopped) == 1)
}
