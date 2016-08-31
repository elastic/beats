// +build !integration

package publisher

import (
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/outputs"
	"github.com/stretchr/testify/assert"
)

const (
	flushInterval time.Duration = 10 * time.Millisecond
	maxBatchSize                = 10
	queueSize                   = 4 * maxBatchSize
	bulkQueueSize               = 1
)

// Send a single event to the bulkWorker and verify that the event
// is sent after the flush timeout occurs.
func TestBulkWorkerSendSingle(t *testing.T) {
	enableLogging([]string{"*"})
	ws := newWorkerSignal()
	defer ws.stop()

	mh := &testMessageHandler{
		response: CompletedResponse,
		msgs:     make(chan message, queueSize),
	}
	bw := newBulkWorker(ws, queueSize, bulkQueueSize, mh, flushInterval, maxBatchSize)

	s := newTestSignaler()
	m := testMessage(s, testEvent())
	bw.send(m)
	msgs, err := mh.waitForMessages(1)
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, s.wait())
	assert.Equal(t, m.datum, msgs[0].data[0])
}

// Send a batch of events to the bulkWorker and verify that a single
// message is distributed (not triggered by flush timeout).
func TestBulkWorkerSendBatch(t *testing.T) {
	// Setup
	ws := newWorkerSignal()
	defer ws.stop()

	mh := &testMessageHandler{
		response: CompletedResponse,
		msgs:     make(chan message, queueSize),
	}
	bw := newBulkWorker(ws, queueSize, 0, mh, time.Duration(time.Hour), maxBatchSize)

	data := make([]outputs.Data, maxBatchSize)
	for i := range data {
		data[i] = testEvent()
	}
	s := newTestSignaler()
	m := testBulkMessage(s, data)
	bw.send(m)

	// Validate
	outMsgs, err := mh.waitForMessages(1)
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, s.wait())
	assert.Len(t, outMsgs[0].data, maxBatchSize)
	assert.Equal(t, m.data[0], outMsgs[0].data[0])
}

// Send more events than the configured maximum batch size and then validate
// that the events are split across two messages.
func TestBulkWorkerSendBatchGreaterThanMaxBatchSize(t *testing.T) {
	// Setup
	ws := newWorkerSignal()
	defer ws.stop()

	mh := &testMessageHandler{
		response: CompletedResponse,
		msgs:     make(chan message),
	}
	bw := newBulkWorker(ws, queueSize, 0, mh, flushInterval, maxBatchSize)

	// Send
	data := make([]outputs.Data, maxBatchSize+1)
	for i := range data {
		data[i] = testEvent()
	}
	s := newTestSignaler()
	m := testBulkMessage(s, data)
	bw.send(m)

	// Read first message and verify no Completed or Failed signal has
	// been received in the sent message.
	outMsgs, err := mh.waitForMessages(1)
	if err != nil {
		t.Fatal(err)
	}
	assert.False(t, s.isDone())
	assert.Len(t, outMsgs[0].data, maxBatchSize)
	assert.Equal(t, m.data[0:maxBatchSize], outMsgs[0].data[0:maxBatchSize])

	// Read the next message and verify the sent message received the
	// Completed signal.
	outMsgs, err = mh.waitForMessages(1)
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, s.wait())
	assert.Len(t, outMsgs[0].data, 1)
	assert.Equal(t, m.data[maxBatchSize], outMsgs[0].data[0])
}
