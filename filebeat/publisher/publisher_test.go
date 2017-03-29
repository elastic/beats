// +build !integration

package publisher

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/libbeat/common/op"
	pubtest "github.com/elastic/beats/libbeat/publisher/testing"
	"github.com/stretchr/testify/assert"
)

type collectLogger struct {
	wg     *sync.WaitGroup
	events [][]*input.Data
}

func (l *collectLogger) Published(events []*input.Data) bool {
	l.wg.Done()
	l.events = append(l.events, events)
	return true
}

func makeEvents(name string, n int) []*input.Data {
	var events []*input.Data
	for i := 0; i < n; i++ {
		event := &input.Event{
			EventMeta: input.EventMeta{
				ReadTime:     time.Now(),
				InputType:    "log",
				DocumentType: "log",
				Bytes:        100,
			},
		}

		eventHolder := event.GetData()
		events = append(events, &eventHolder)
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

		wg := sync.WaitGroup{}

		pubChan := make(chan []*input.Data, len(test.order)+1)
		collector := &collectLogger{&wg, nil}
		client := pubtest.NewChanClient(0)

		pub := New(test.async, pubChan, collector,
			pubtest.PublisherWithClient(client))
		pub.Start()

		var events [][]*input.Data
		for i := range test.order {
			tmp := makeEvents(fmt.Sprintf("msg: %v", i), 1)
			wg.Add(1)
			pubChan <- tmp
			events = append(events, tmp)
		}

		var msgs []pubtest.PublishMessage
		for range test.order {
			m := <-client.Channel
			msgs = append(msgs, m)
		}

		for _, i := range test.order {
			op.SigCompleted(msgs[i-1].Context.Signal)
		}

		wg.Wait()
		pub.Stop()

		// validate order
		assert.Equal(t, len(events), len(collector.events))
		for i := range events {
			assert.Equal(t, events[i], collector.events[i])
		}
	}
}
