package mode

import (
	"sync"
	"testing"
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/outputs"
	"github.com/stretchr/testify/assert"
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

func (c *mockClient) PublishEvent(event common.MapStr) error {
	_, err := c.PublishEvents([]common.MapStr{event})
	return err
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
