// +build !integration

package publisher

import (
	"testing"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/stretchr/testify/assert"
)

func TestAsyncPublishEvent(t *testing.T) {
	enableLogging([]string{"*"})
	// Init
	testPub := newTestPublisherNoBulk(CompletedResponse)
	event := testEvent()

	defer testPub.Stop()

	// Execute. Async PublishEvent always immediately returns true.
	assert.True(t, testPub.asyncPublishEvent(event))

	// Validate
	msgs, err := testPub.outputMsgHandler.waitForMessages(1)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, event, msgs[0].datum)
}

func TestAsyncPublishEvents(t *testing.T) {
	// Init
	testPub := newTestPublisherNoBulk(CompletedResponse)
	events := []outputs.Data{testEvent(), testEvent()}

	defer testPub.Stop()

	// Execute. Async PublishEvent always immediately returns true.
	assert.True(t, testPub.asyncPublishEvents(events))

	// Validate
	msgs, err := testPub.outputMsgHandler.waitForMessages(1)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, events[0], msgs[0].data[0])
	assert.Equal(t, events[1], msgs[0].data[1])
}

func TestAsyncShutdownPublishEvents(t *testing.T) {
	// Init
	testPub := newTestPublisherNoBulk(CompletedResponse)
	events := []outputs.Data{testEvent(), testEvent()}

	// Execute. Async PublishEvent always immediately returns true.
	assert.True(t, testPub.asyncPublishEvents(events))

	testPub.Stop()

	// Validate
	msgs := testPub.outputMsgHandler.msgs
	close(msgs)
	assert.Equal(t, 1, len(msgs))
	msg := <-msgs
	assert.Equal(t, events[0], msg.data[0])
	assert.Equal(t, events[1], msg.data[1])
}

func TestBulkAsyncPublishEvent(t *testing.T) {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"*"})
	}

	// Init
	testPub := newTestPublisherWithBulk(CompletedResponse)
	event := testEvent()

	defer testPub.Stop()

	// Execute. Async PublishEvent always immediately returns true.
	assert.True(t, testPub.asyncPublishEvent(event))

	// validate
	msgs, err := testPub.outputMsgHandler.waitForMessages(1)
	if err != nil {
		t.Fatal(err)
	}

	// Bulk outputer always sends bulk messages (even if only one event is
	// present)
	assert.Equal(t, event, msgs[0].datum)
}

func TestBulkAsyncPublishEvents(t *testing.T) {
	// Init
	testPub := newTestPublisherWithBulk(CompletedResponse)
	events := []outputs.Data{testEvent(), testEvent()}

	defer testPub.Stop()

	// Async PublishEvent always immediately returns true.
	assert.True(t, testPub.asyncPublishEvents(events))

	msgs, err := testPub.outputMsgHandler.waitForMessages(1)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, events[0], msgs[0].data[0])
	assert.Equal(t, events[1], msgs[0].data[1])
}

func TestBulkAsyncShutdownPublishEvents(t *testing.T) {
	// Init
	testPub := newTestPublisherWithBulk(CompletedResponse)
	events := []outputs.Data{testEvent(), testEvent()}

	// Async PublishEvent always immediately returns true.
	assert.True(t, testPub.asyncPublishEvents(events))

	testPub.Stop()

	// Validate
	msgs := testPub.outputMsgHandler.msgs
	close(msgs)
	assert.Equal(t, 1, len(msgs))
	msg := <-msgs
	assert.Equal(t, events[0], msg.data[0])
	assert.Equal(t, events[1], msg.data[1])
}
