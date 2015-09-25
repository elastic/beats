package publisher

import (
	"reflect"
	"testing"

	"github.com/elastic/libbeat/common"
	"github.com/stretchr/testify/assert"
)

// Test that the correct client type is returned based on the given
// ClientOptions.
func TestGetClient(t *testing.T) {
	c := &client{
		publisher: &PublisherType{
			asyncPublisher: &asyncPublisher{},
			syncPublisher:  &syncPublisher{},
		},
	}
	asyncClient := c.publisher.asyncPublisher.client()
	syncClient := c.publisher.syncPublisher.client()

	var testCases = []struct {
		in  []ClientOption
		out eventPublisher
	}{
		// Add new client options here:
		{[]ClientOption{}, asyncClient},
		{[]ClientOption{Confirm}, syncClient},
	}

	for _, test := range testCases {
		expected := reflect.ValueOf(test.out)
		actual := reflect.ValueOf(c.getClient(test.in))
		assert.Equal(t, expected.Pointer(), actual.Pointer())
	}
}

// Test that ChanClient writes an event to its Channel.
func TestChanClientPublishEvent(t *testing.T) {
	cc := &ChanClient{
		Channel: make(chan common.MapStr, 1),
	}

	e1 := testEvent()
	cc.PublishEvent(e1)
	assert.Equal(t, e1, <-cc.Channel)
}

// Test that ChanClient write events to its Channel.
func TestChanClientPublishEvents(t *testing.T) {
	cc := &ChanClient{
		Channel: make(chan common.MapStr, 2),
	}

	e1, e2 := testEvent(), testEvent()
	cc.PublishEvents([]common.MapStr{e1, e2})
	assert.Equal(t, e1, <-cc.Channel)
	assert.Equal(t, e2, <-cc.Channel)
}
