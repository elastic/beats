package publisher

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/stretchr/testify/assert"
)

const (
	flushInterval time.Duration = 10 * time.Millisecond
	maxBatchSize                = 10
	queueSize                   = 4 * maxBatchSize
	bulkQueueSize               = 1
)

var shutdownError = errors.New("Failed to shutdown bulkWorker")

// Send a single event to the bulkWorker and verify that the event
// is sent after the flush timeout occurs.
func TestBulkWorkerSendSingle(t *testing.T) {
	enableLogging([]string{"*"})
	var wg sync.WaitGroup
	mh := &testMessageHandler{
		response: CompletedResponse,
		msgs:     make(chan message, queueSize),
	}
	bw := newBulkWorker(&wg, queueSize, bulkQueueSize, mh, flushInterval, maxBatchSize)

	s := newTestSignaler()
	m := testMessage(s, testEvent())
	bw.send(m)
	msgs, err := mh.waitForMessages(1)
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, s.wait())
	assert.Equal(t, m.event, msgs[0].events[0])

	err = wait(bw.shutdown, shutdownError)
	if err != nil {
		t.Fatal(err)
	}
}

// Send a single event to a bulkWorker with a large flushInterval
// and verify that the event is sent (flushed) after shutdown.
func TestBulkWorkerShutdownFlushSingle(t *testing.T) {
	var wg sync.WaitGroup
	mh := &testMessageHandler{
		response: CompletedResponse,
		msgs:     make(chan message, queueSize),
	}
	bw := newBulkWorker(&wg, queueSize, bulkQueueSize, mh, 30*time.Second, maxBatchSize)

	s := newTestSignaler()
	m := testMessage(s, testEvent())
	bw.send(m)

	errW := wait(bw.shutdown, shutdownError)
	if errW != nil {
		t.Fatal(errW)
	}

	msgs, err := mh.waitForMessages(1)
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, s.wait())
	assert.Equal(t, m.event, msgs[0].events[0])
}

// Send a batch of events to the bulkWorker and verify that a single
// message is distributed (not triggered by flush timeout).
func TestBulkWorkerSendBatch(t *testing.T) {
	// Setup
	var wg sync.WaitGroup
	mh := &testMessageHandler{
		response: CompletedResponse,
		msgs:     make(chan message, queueSize),
	}
	bw := newBulkWorker(&wg, queueSize, 0, mh, time.Duration(time.Hour), maxBatchSize)

	events := make([]common.MapStr, maxBatchSize)
	for i := range events {
		events[i] = testEvent()
	}
	s := newTestSignaler()
	m := testBulkMessage(s, events)
	bw.send(m)

	// Validate
	outMsgs, err := mh.waitForMessages(1)
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, s.wait())
	assert.Len(t, outMsgs[0].events, maxBatchSize)
	assert.Equal(t, m.events[0], outMsgs[0].events[0])

	err = wait(bw.shutdown, shutdownError)
	if err != nil {
		t.Fatal(err)
	}
}

// Send more events than the configured maximum batch size and then validate
// that the events are split across two messages.
func TestBulkWorkerSendBatchGreaterThanMaxBatchSize(t *testing.T) {
	// Setup
	var wg sync.WaitGroup
	mh := &testMessageHandler{
		response: CompletedResponse,
		msgs:     make(chan message),
	}
	bw := newBulkWorker(&wg, queueSize, 0, mh, flushInterval, maxBatchSize)

	// Send
	events := make([]common.MapStr, maxBatchSize+1)
	for i := range events {
		events[i] = testEvent()
	}
	s := newTestSignaler()
	m := testBulkMessage(s, events)
	bw.send(m)

	// Read first message and verify no Completed or Failed signal has
	// been received in the sent message.
	outMsgs, err := mh.waitForMessages(1)
	if err != nil {
		t.Fatal(err)
	}
	assert.False(t, s.isDone())
	assert.Len(t, outMsgs[0].events, maxBatchSize)
	assert.Equal(t, m.events[0:maxBatchSize], outMsgs[0].events[0:maxBatchSize])

	// Read the next message and verify the sent message received the
	// Completed signal.
	outMsgs, err = mh.waitForMessages(1)
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, s.wait())
	assert.Len(t, outMsgs[0].events, 1)
	assert.Equal(t, m.events[maxBatchSize], outMsgs[0].events[0])

	err = wait(bw.shutdown, shutdownError)
	if err != nil {
		t.Fatal(err)
	}
}
