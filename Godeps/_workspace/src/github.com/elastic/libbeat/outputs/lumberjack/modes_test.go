package lumberjack

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/outputs"
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

func alwaysFailConnect(err error) func(time.Duration) error {
	return func(timeout time.Duration) error {
		return err
	}
}

func collectPublish(
	collected *[][]common.MapStr,
) func(events []common.MapStr) (int, error) {
	mutex := sync.Mutex{}
	return func(events []common.MapStr) (int, error) {
		mutex.Lock()
		defer mutex.Unlock()

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

func publishFailStart(
	n int,
	pub func(events []common.MapStr) (int, error),
) func(events []common.MapStr) (int, error) {
	count := 0
	return func(events []common.MapStr) (int, error) {
		if count < n {
			count++
			return 0, errNetTimeout{}
		}
		count = 0
		return pub(events)
	}
}

func closeOK() error {
	return nil
}

var testEvent = common.MapStr{
	"msg": "hello world",
}

func testMode(
	t *testing.T,
	mode ConnectionMode,
	events [][]common.MapStr,
	expectedSignal bool,
	collectedEvents *[][]common.MapStr,
) {
	defer mode.Close()

	if events != nil {
		ch := make(chan bool, 1)
		signal := outputs.NewChanSignal(ch)
		for _, pubEvents := range events {
			_ = mode.PublishEvents(signal, pubEvents)

			result := <-ch
			assert.Equal(t, expectedSignal, result)
		}

		if collectedEvents != nil {
			assert.Equal(t, len(events), len(*collectedEvents))
			for i := range *collectedEvents {
				expected := events[i]
				actual := (*collectedEvents)[i]
				assert.Equal(t, expected, actual)
			}
		}
	}
}

func singleEvent(e common.MapStr) [][]common.MapStr {
	return [][]common.MapStr{
		[]common.MapStr{e},
	}
}

func repeatEvent(n int, e common.MapStr) [][]common.MapStr {
	event := []common.MapStr{e}
	var events [][]common.MapStr
	for i := 0; i < n; i++ {
		events = append(events, event)
	}
	return events
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
		3,
		0,
		100*time.Millisecond,
	)
	testMode(t, mode, singleEvent(testEvent), true, &collected)
}

func TestSingleConnectFailConnect(t *testing.T) {
	var collected [][]common.MapStr
	errFail := errors.New("fail connect")
	mode, _ := newSingleConnectionMode(
		&mockClient{
			connected: false,
			close:     closeOK,
			connect:   failConnect(2, errFail),
			publish:   collectPublish(&collected),
		},
		3,
		0,
		100*time.Millisecond,
	)
	testMode(t, mode, singleEvent(testEvent), true, &collected)
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
		3,
		0,
		100*time.Millisecond,
	)
	testMode(t, mode, singleEvent(testEvent), true, &collected)
}

func TestFailoverFlakyConnections(t *testing.T) {
	// t.SkipNow()

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
		3,
		1*time.Millisecond,
		100*time.Millisecond,
	)
	testMode(t, mode, repeatEvent(10, testEvent), true, &collected)
}

func TestLoadBalancerStartStop(t *testing.T) {
	mode, _ := newLoadBalancerMode(
		[]ProtocolClient{
			&mockClient{
				connected: false,
				close:     closeOK,
				connect:   connectOK,
			},
		},
		1,
		100*time.Millisecond,
		100*time.Millisecond,
	)
	testMode(t, mode, nil, true, nil)
}

func TestLoadBalancerFailSendWithoutActiveConnections(t *testing.T) {
	errFail := errors.New("fail connect")
	mode, _ := newLoadBalancerMode(
		[]ProtocolClient{
			&mockClient{
				connected: false,
				close:     closeOK,
				connect:   alwaysFailConnect(errFail),
			},
		},
		2,
		100*time.Millisecond,
		100*time.Millisecond,
	)
	testMode(t, mode, singleEvent(testEvent), false, nil)
}

func TestLoadBalancerOKSend(t *testing.T) {
	var collected [][]common.MapStr
	mode, _ := newLoadBalancerMode(
		[]ProtocolClient{
			&mockClient{
				connected: false,
				close:     closeOK,
				connect:   connectOK,
				publish:   collectPublish(&collected),
			},
		},
		2,
		100*time.Millisecond,
		100*time.Millisecond,
	)
	testMode(t, mode, singleEvent(testEvent), true, &collected)
}

func TestLoadBalancerFlakyConnectionOkSend(t *testing.T) {
	var collected [][]common.MapStr
	mode, _ := newLoadBalancerMode(
		[]ProtocolClient{
			&mockClient{
				connected: true,
				close:     closeOK,
				connect:   connectOK,
				publish:   publishFailStart(1, collectPublish(&collected)),
			},
			&mockClient{
				connected: true,
				close:     closeOK,
				connect:   connectOK,
				publish:   publishFailStart(1, collectPublish(&collected)),
			},
		},
		3,
		100*time.Millisecond,
		100*time.Millisecond,
	)
	testMode(t, mode, singleEvent(testEvent), true, &collected)
}
