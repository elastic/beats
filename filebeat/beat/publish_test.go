package beat

import (
	"fmt"
	"testing"
	"time"

	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/publisher"
	"github.com/stretchr/testify/assert"
)

func makeEvents(name string, n int) []*input.FileEvent {
	var events []*input.FileEvent
	for i := 0; i < n; i++ {
		event := &input.FileEvent{
			ReadTime:     time.Now(),
			InputType:    "log",
			DocumentType: "log",
			Bytes:        100,
			Offset:       int64(i),
			Source:       &name,
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

		pubChan := make(chan []*input.FileEvent, len(test.order)+1)
		regChan := make(chan []*input.FileEvent, len(test.order)+1)
		client := publisher.ExtChanClient{make(chan publisher.PublishMessage)}

		pub := newPublisher(test.async, pubChan, regChan, client)
		pub.Start()

		var events [][]*input.FileEvent
		for i := range test.order {
			tmp := makeEvents(fmt.Sprintf("msg: %v", i), 1)
			pubChan <- tmp
			events = append(events, tmp)
		}

		var msgs []publisher.PublishMessage
		for _ = range test.order {
			m := <-client.Channel
			msgs = append(msgs, m)
		}

		for _, i := range test.order {
			outputs.SignalCompleted(msgs[i-1].Context.Signal)
		}

		var regEvents [][]*input.FileEvent
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
