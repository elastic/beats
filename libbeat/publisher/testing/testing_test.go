package testing

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

var cnt = 0

func testEvent() beat.Event {
	event := beat.Event{
		Fields: common.MapStr{
			"message": "test",
			"idx":     cnt,
		},
	}
	cnt++
	return event
}

// Test that ChanClient writes an event to its Channel.
func TestChanClientPublishEvent(t *testing.T) {
	cc := NewChanClient(1)
	e1 := testEvent()
	cc.Publish(e1)
	assert.Equal(t, e1, cc.ReceiveEvent())
}

// Test that ChanClient write events to its Channel.
func TestChanClientPublishEvents(t *testing.T) {
	cc := NewChanClient(1)

	e1, e2 := testEvent(), testEvent()
	go cc.PublishAll([]beat.Event{e1, e2})
	assert.Equal(t, e1, cc.ReceiveEvent())
	assert.Equal(t, e2, cc.ReceiveEvent())
}
