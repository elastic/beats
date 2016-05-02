package testing

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/stretchr/testify/assert"
)

var cnt = 0

func testEvent() common.MapStr {
	event := common.MapStr{}
	event["message"] = "test"
	event["idx"] = cnt
	cnt++
	return event
}

// Test that ChanClient writes an event to its Channel.
func TestChanClientPublishEvent(t *testing.T) {
	cc := NewChanClient(1)
	e1 := testEvent()
	cc.PublishEvent(e1)
	assert.Equal(t, e1, cc.ReceiveEvent())
}

// Test that ChanClient write events to its Channel.
func TestChanClientPublishEvents(t *testing.T) {
	cc := NewChanClient(1)

	e1, e2 := testEvent(), testEvent()
	cc.PublishEvents([]common.MapStr{e1, e2})
	assert.Equal(t, e1, cc.ReceiveEvent())
	assert.Equal(t, e2, cc.ReceiveEvent())
}
