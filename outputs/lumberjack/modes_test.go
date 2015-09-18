package lumberjack

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/libbeat/common"
)

type mockClient struct {
	publish   func([]common.MapStr) (int, error)
	close     func() error
	connected bool
	connect   func(time.Duration) error
}

func (c *mockClient) Connect(timeout time.Duration) error {
	return c.connect(timeout)
}

func (c *mockClient) Close() error {
	return c.close()
}

func (c *mockClient) IsConnected() bool {
	return c.connected
}

func (c *mockClient) PublishEvents(events []common.MapStr) (int, error) {
	return c.publish(events)
}

func connectOK(timeout time.Duration) error {
	return nil
}

func failConnect(n int, err error) func(time.Duration) error {
	count := 0
	return func(timeout time.Duration) error {
		count++
		if count < n {
			return err
		}
		count = 0
		return nil
	}
}

func collectPublish(
	collected *[][]common.MapStr,
) func(events []common.MapStr) (int, error) {
	return func(events []common.MapStr) (int, error) {
		*collected = append(*collected, events)
		return len(events), nil
	}
}

type errNetTimeout struct{}

func (e errNetTimeout) Error() string   { return "errNetTimeout" }
func (e errNetTimeout) Timeout() bool   { return true }
func (e errNetTimeout) Temporary() bool { return false }

func publishTimeoutEvery(
	n int,
	pub func(events []common.MapStr) (int, error),
) func(events []common.MapStr) (int, error) {
	count := 0
	return func(events []common.MapStr) (int, error) {
		if count < n {
			count++
			return pub(events)
		}
		count = 0
		return 0, errNetTimeout{}
	}
}

func closeOK() error {
	return nil
}

func TestSingleSend(t *testing.T) {
	var collected [][]common.MapStr
	mode, _ := newSingleConnectionMode(
		&mockClient{
			connected: false,
			close:     closeOK,
			connect:   connectOK,
			publish:   collectPublish(&collected),
		},
		0,
		1*time.Second,
	)

	events := []common.MapStr{common.MapStr{"hello": "world"}}
	err := mode.PublishEvents(nil, events)
	mode.Close()

	assert.Nil(t, err)
	assert.Equal(t, 1, len(collected))
	assert.Equal(t, events, collected[0])
}

func TestSingleConnectFailConnect(t *testing.T) {
	var collected [][]common.MapStr
	errFail := errors.New("fail connect")
	mode, _ := newSingleConnectionMode(
		&mockClient{
			connected: false,
			close:     closeOK,
			connect:   failConnect(5, errFail),
			publish:   collectPublish(&collected),
		},
		0,
		1*time.Second,
	)

	events := []common.MapStr{common.MapStr{"hello": "world"}}
	err := mode.PublishEvents(nil, events)
	mode.Close()

	assert.Nil(t, err)
	assert.Equal(t, 1, len(collected))
	assert.Equal(t, events, collected[0])
}

func TestFailoverSingleSend(t *testing.T) {
	var collected [][]common.MapStr
	mode, _ := newFailOverConnectionMode(
		[]ProtocolClient{
			&mockClient{
				connected: false,
				close:     closeOK,
				connect:   connectOK,
				publish:   collectPublish(&collected),
			},
			&mockClient{
				connected: false,
				close:     closeOK,
				connect:   connectOK,
				publish:   collectPublish(&collected),
			},
		},
		0,
		1*time.Second,
	)

	events := []common.MapStr{common.MapStr{"hello": "world"}}

	err := mode.PublishEvents(nil, events)
	mode.Close()

	assert.Nil(t, err)
	assert.Equal(t, 1, len(collected))
	assert.Equal(t, events, collected[0])
}

func TestFailoverFlakyConnections(t *testing.T) {
	errFail := errors.New("fail connect")
	var collected [][]common.MapStr
	mode, _ := newFailOverConnectionMode(
		[]ProtocolClient{
			&mockClient{
				connected: false,
				close:     closeOK,
				connect:   failConnect(2, errFail),
				publish:   publishTimeoutEvery(1, collectPublish(&collected)),
			},
			&mockClient{
				connected: false,
				close:     closeOK,
				connect:   failConnect(1, errFail),
				publish:   publishTimeoutEvery(2, collectPublish(&collected)),
			},
		},
		0,
		1*time.Second,
	)

	events := []common.MapStr{common.MapStr{"hello": "world"}}
	for i := 0; i < 10; i++ {
		mode.PublishEvents(nil, events)
	}

	mode.Close()

	assert.Equal(t, 10, len(collected))
	assert.Equal(t, events, collected[0])
}
