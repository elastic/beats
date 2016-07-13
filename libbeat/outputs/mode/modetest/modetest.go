package modetest

import (
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/op"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/mode"
	"github.com/stretchr/testify/assert"
)

type MockClient struct {
	Connected      bool
	CBPublish      func([]common.MapStr) ([]common.MapStr, error)
	CBAsyncPublish func(func([]common.MapStr, error), []common.MapStr) error
	CBClose        func() error
	CBConnect      func(time.Duration) error
}

func NewMockClient(template *MockClient) *MockClient {
	mc := &MockClient{
		Connected:      true,
		CBConnect:      ConnectOK,
		CBClose:        CloseOK,
		CBPublish:      PublishIgnore,
		CBAsyncPublish: AsyncPublishIgnore,
	}

	if template != nil {
		mc.Connected = template.Connected
		if template.CBPublish != nil {
			mc.CBPublish = template.CBPublish
		}
		if template.CBAsyncPublish != nil {
			mc.CBAsyncPublish = template.CBAsyncPublish
		}
		if template.CBClose != nil {
			mc.CBClose = template.CBClose
		}
		if template.CBConnect != nil {
			mc.CBConnect = template.CBConnect
		}
	}

	return mc
}

func (c *MockClient) Connect(timeout time.Duration) error {
	err := c.CBConnect(timeout)
	c.Connected = err == nil
	return err
}

func (c *MockClient) Close() error {
	err := c.CBClose()
	c.Connected = false
	return err
}

func (c *MockClient) PublishEvents(events []common.MapStr) ([]common.MapStr, error) {
	return c.CBPublish(events)
}

func (c *MockClient) PublishEvent(event common.MapStr) error {
	_, err := c.PublishEvents([]common.MapStr{event})
	return err
}

func (c *MockClient) AsyncPublishEvents(cb func([]common.MapStr, error), events []common.MapStr) error {
	return c.CBAsyncPublish(cb, events)
}

func (c *MockClient) AsyncPublishEvent(cb func(error), event common.MapStr) error {
	return c.AsyncPublishEvents(
		func(evts []common.MapStr, err error) { cb(err) },
		[]common.MapStr{event})
}

func SyncClients(n int, tmpl *MockClient) []mode.ProtocolClient {
	cl := make([]mode.ProtocolClient, n)
	for i := 0; i < n; i++ {
		cl[i] = NewMockClient(tmpl)
	}
	return cl
}

func AsyncClients(n int, tmpl *MockClient) []mode.AsyncProtocolClient {
	cl := make([]mode.AsyncProtocolClient, n)
	for i := 0; i < n; i++ {
		cl[i] = NewMockClient(tmpl)
	}
	return cl
}

func TestMode(
	t *testing.T,
	mode mode.ConnectionMode,
	opts outputs.Options,
	events []EventInfo,
	expectedSignals []bool,
	collectedEvents *[][]common.MapStr,
) {
	defer func() {
		err := mode.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	if events == nil {
		return
	}

	numSignals := 0
	for _, pubEvents := range events {
		if pubEvents.Single {
			numSignals += len(pubEvents.Events)
		} else {
			numSignals++
		}
	}

	var expectedEvents [][]common.MapStr
	ch := make(chan op.SignalResponse, numSignals)
	signal := &op.SignalChannel{ch}
	idx := 0
	for _, pubEvents := range events {
		if pubEvents.Single {
			for _, event := range pubEvents.Events {
				_ = mode.PublishEvent(signal, opts, event)
				if expectedSignals[idx] {
					expectedEvents = append(expectedEvents, []common.MapStr{event})
				}
				idx++
			}
		} else {
			_ = mode.PublishEvents(signal, opts, pubEvents.Events)
			if expectedSignals[idx] {
				expectedEvents = append(expectedEvents, pubEvents.Events)
			}
			idx++
		}
	}

	results := make([]bool, len(expectedSignals))
	for i := 0; i < len(expectedSignals); i++ {
		results[i] = <-ch == op.SignalCompleted
	}
	assert.Equal(t, expectedSignals, results)

	if collectedEvents != nil {
		assert.Equal(t, len(expectedEvents), len(*collectedEvents))
		if len(expectedEvents) == len(*collectedEvents) {
			for i := range *collectedEvents {
				expected := expectedEvents[i]
				actual := (*collectedEvents)[i]
				assert.Equal(t, expected, actual)
			}
		}
	}
}
