package publisher

import (
	"testing"

	"github.com/elastic/libbeat/common"
	"github.com/stretchr/testify/assert"
)

func TestSyncPublishEventSuccess(t *testing.T) {
	testPub := newTestPublisherNoBulk(CompletedResponse)
	event := testEvent()

	assert.True(t, testPub.syncPublishEvent(event))

	msgs, err := testPub.outputMsgHandler.waitForMessages(1)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, event, msgs[0].event)
}

func TestSyncPublishEventsSuccess(t *testing.T) {
	testPub := newTestPublisherNoBulk(CompletedResponse)
	events := []common.MapStr{testEvent(), testEvent()}

	assert.True(t, testPub.syncPublishEvents(events))

	msgs, err := testPub.outputMsgHandler.waitForMessages(1)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, events[0], msgs[0].events[0])
	assert.Equal(t, events[1], msgs[0].events[1])
}

func TestSyncPublishEventFailed(t *testing.T) {
	testPub := newTestPublisherNoBulk(FailedResponse)
	event := testEvent()

	assert.False(t, testPub.syncPublishEvent(event))

	msgs, err := testPub.outputMsgHandler.waitForMessages(1)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, event, msgs[0].event)
}

func TestSyncPublishEventsFailed(t *testing.T) {
	testPub := newTestPublisherNoBulk(FailedResponse)
	events := []common.MapStr{testEvent(), testEvent()}

	assert.False(t, testPub.syncPublishEvents(events))

	msgs, err := testPub.outputMsgHandler.waitForMessages(1)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, events[0], msgs[0].events[0])
	assert.Equal(t, events[1], msgs[0].events[1])
}

// Test that PublishEvent returns true when publishing is disabled.
func TestSyncPublisherDisabled(t *testing.T) {
	testPub := newTestPublisherNoBulk(FailedResponse)
	testPub.pub.disabled = true
	event := testEvent()

	assert.True(t, testPub.syncPublishEvent(event))
}
