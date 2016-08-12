// +build !integration

package publish

import (
	"fmt"
	"testing"
	"time"

	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/libbeat/common/op"
	pubtest "github.com/elastic/beats/libbeat/publisher/testing"
	"github.com/stretchr/testify/assert"
)

func makeEvents(name string, n int) []*input.Event {
	var events []*input.Event
	for i := 0; i < n; i++ {
		event := &input.Event{
			ReadTime:     time.Now(),
			InputType:    "log",
			DocumentType: "log",
			Bytes:        100,
		}
		events = append(events, event)
	}
	return events
}

func TestPublisherModes(t *testing.T) {
	tests := []struct {
		title string
		async bool
		order []int
	}{
		{"sync", false, []int{1, 2, 3, 4, 5, 6}},
		{"async ordered signal", true, []int{1, 2, 3, 4, 5, 6}},
		{"async out of order signal", true, []int{5, 2, 3, 1, 4, 6}},
	}

	for i, test := range tests {
		t.Logf("run publisher test (%v): %v", i, test.title)

		pubChan := make(chan []*input.Event, len(test.order)+1)
		regChan := make(chan []*input.Event, len(test.order)+1)
		client := pubtest.NewChanClient(0)

		pub := New(test.async, pubChan, regChan, pubtest.PublisherWithClient(client))
		pub.Start()

		var events [][]*input.Event
		for i := range test.order {
			tmp := makeEvents(fmt.Sprintf("msg: %v", i), 1)
			pubChan <- tmp
			events = append(events, tmp)
		}

		var msgs []pubtest.PublishMessage
		for _ = range test.order {
			m := <-client.Channel
			msgs = append(msgs, m)
		}

		for _, i := range test.order {
			op.SigCompleted(msgs[i-1].Context.Signal)
		}

		var regEvents [][]*input.Event
		for _ = range test.order {
			regEvents = append(regEvents, <-regChan)
		}
		pub.Stop()

		// validate order
		for i := range events {
			assert.Equal(t, events[i], regEvents[i])
		}
	}
}
