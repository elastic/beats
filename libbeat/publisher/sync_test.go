// +build !integration

package publisher

import (
	"testing"

	"github.com/elastic/beats/libbeat/outputs"
	"github.com/stretchr/testify/assert"
)

func TestSyncPublishEventSuccess(t *testing.T) {
	enableLogging([]string{"*"})
	testPub := newTestPublisherNoBulk(CompletedResponse)
	event := testEvent()

	assert.True(t, testPub.syncPublishEvent(event))

	msgs, err := testPub.outputMsgHandler.waitForMessages(1)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, event, msgs[0].datum)
}

func TestSyncPublishEventsSuccess(t *testing.T) {
	testPub := newTestPublisherNoBulk(CompletedResponse)
	data := []outputs.Data{testEvent(), testEvent()}

	assert.True(t, testPub.syncPublishEvents(data))

	msgs, err := testPub.outputMsgHandler.waitForMessages(1)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, data[0], msgs[0].data[0])
	assert.Equal(t, data[1], msgs[0].data[1])
}

func TestSyncPublishEventFailed(t *testing.T) {
	testPub := newTestPublisherNoBulk(FailedResponse)
	event := testEvent()

	assert.False(t, testPub.syncPublishEvent(event))

	msgs, err := testPub.outputMsgHandler.waitForMessages(1)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, event, msgs[0].datum)
}

func TestSyncPublishEventsFailed(t *testing.T) {
	testPub := newTestPublisherNoBulk(FailedResponse)
	data := []outputs.Data{testEvent(), testEvent()}

	assert.False(t, testPub.syncPublishEvents(data))

	msgs, err := testPub.outputMsgHandler.waitForMessages(1)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, data[0], msgs[0].data[0])
	assert.Equal(t, data[1], msgs[0].data[1])
}

// Test that PublishEvent returns true when publishing is disabled.
func TestSyncPublisherDisabled(t *testing.T) {
	testPub := newTestPublisherNoBulk(FailedResponse)
	testPub.pub.disabled = true
	event := testEvent()

	assert.True(t, testPub.syncPublishEvent(event))
}
