package modetest

import (
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common/op"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/mode"
	"github.com/stretchr/testify/assert"
)

type MockClient struct {
	Connected      bool
	CBPublish      func([]outputs.Data) ([]outputs.Data, error)
	CBAsyncPublish func(func([]outputs.Data, error), []outputs.Data) error
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

func (c *MockClient) PublishEvents(data []outputs.Data) ([]outputs.Data, error) {
	return c.CBPublish(data)
}

func (c *MockClient) PublishEvent(data outputs.Data) error {
	_, err := c.PublishEvents([]outputs.Data{data})
	return err
}

func (c *MockClient) AsyncPublishEvents(cb func([]outputs.Data, error), data []outputs.Data) error {
	return c.CBAsyncPublish(cb, data)
}

func (c *MockClient) AsyncPublishEvent(cb func(error), data outputs.Data) error {
	return c.AsyncPublishEvents(
		func(evts []outputs.Data, err error) { cb(err) },
		[]outputs.Data{data})
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
	data []EventInfo,
	expectedSignals []bool,
	collected *[][]outputs.Data,
) {
	defer func() {
		err := mode.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	if data == nil {
		return
	}

	results, expectedData := PublishWith(t, mode, opts, data, expectedSignals)
	assert.Equal(t, expectedSignals, results)

	if collected != nil {
		assert.Equal(t, len(expectedData), len(*collected))
		if len(expectedData) == len(*collected) {
			for i := range *collected {
				expected := expectedData[i]
				actual := (*collected)[i]
				assert.Equal(t, expected, actual)
			}
		}
	}
}

func PublishWith(
	t *testing.T,
	mode mode.ConnectionMode,
	opts outputs.Options,
	data []EventInfo,
	expectedSignals []bool,
) ([]bool, [][]outputs.Data) {
	return doPublishWith(t, mode, opts, data, func(i int) bool {
		return expectedSignals[i]
	})
}

func PublishAllWith(
	t *testing.T,
	mode mode.ConnectionMode,
	data []EventInfo,
) ([]bool, [][]outputs.Data) {
	opts := outputs.Options{Guaranteed: true}
	expectSignal := func(_ int) bool { return true }
	return doPublishWith(t, mode, opts, data, expectSignal)
}

func doPublishWith(
	t *testing.T,
	mode mode.ConnectionMode,
	opts outputs.Options,
	data []EventInfo,
	expectedSignals func(int) bool,
) ([]bool, [][]outputs.Data) {
	if data == nil {
		return nil, nil
	}

	numSignals := 0
	for _, pubEvents := range data {
		if pubEvents.Single {
			numSignals += len(pubEvents.Data)
		} else {
			numSignals++
		}
	}

	var expectedData [][]outputs.Data
	ch := make(chan op.SignalResponse, numSignals)
	signal := &op.SignalChannel{ch}
	idx := 0
	for _, pubEvents := range data {
		if pubEvents.Single {
			for _, event := range pubEvents.Data {
				_ = mode.PublishEvent(signal, opts, event)
				if expectedSignals(idx) {
					expectedData = append(expectedData, []outputs.Data{event})
				}
				idx++
			}
		} else {
			_ = mode.PublishEvents(signal, opts, pubEvents.Data)
			if expectedSignals(idx) {
				expectedData = append(expectedData, pubEvents.Data)
			}
			idx++
		}
	}

	var signals []bool
	for i := 0; i < idx; i++ {
		signals = append(signals, <-ch == op.SignalCompleted)
	}

	return signals, expectedData
}
