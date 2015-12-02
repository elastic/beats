package publisher

import (
	"testing"

	"github.com/elastic/libbeat/common"
	"github.com/stretchr/testify/assert"
)

func TestAsyncPublishEvent(t *testing.T) {
	// Init
	testPub := newTestPublisherNoBulk(CompletedResponse)
	event := testEvent()

	// Execute. Async PublishEvent always immediately returns true.
	assert.True(t, testPub.asyncPublishEvent(event))

	// Validate
	msgs, err := testPub.outputMsgHandler.waitForMessages(1)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, event, msgs[0].event)
}

func TestAsyncPublishEvents(t *testing.T) {
	// Init
	testPub := newTestPublisherNoBulk(CompletedResponse)
	events := []common.MapStr{testEvent(), testEvent()}

	// Execute. Async PublishEvent always immediately returns true.
	assert.True(t, testPub.asyncPublishEvents(events))

	// Validate
	msgs, err := testPub.outputMsgHandler.waitForMessages(1)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, events[0], msgs[0].events[0])
	assert.Equal(t, events[1], msgs[0].events[1])
}

func TestBulkAsyncPublishEvent(t *testing.T) {
	// Init
	testPub := newTestPublisherWithBulk(CompletedResponse)
	event := testEvent()

	// Execute. Async PublishEvent always immediately returns true.
	assert.True(t, testPub.asyncPublishEvent(event))

	// validate
	msgs, err := testPub.outputMsgHandler.waitForMessages(1)
	if err != nil {
		t.Fatal(err)
	}
	// Bulk outputer always sends bulk messages (even if only one event is
	// present)
	assert.Equal(t, event, msgs[0].events[0])
}

func TestBulkAsyncPublishEvents(t *testing.T) {
	// init
	testPub := newTestPublisherWithBulk(CompletedResponse)
	events := []common.MapStr{testEvent(), testEvent()}

	// Async PublishEvent always immediately returns true.
	assert.True(t, testPub.asyncPublishEvents(events))

	msgs, err := testPub.outputMsgHandler.waitForMessages(1)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, events[0], msgs[0].events[0])
	assert.Equal(t, events[1], msgs[0].events[1])
}
