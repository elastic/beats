// +build !integration

package mode

import (
	"sync"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/op"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/stretchr/testify/assert"
)

type mockClient struct {
	publish      func([]common.MapStr) ([]common.MapStr, error)
	asyncPublish func(func([]common.MapStr, error), []common.MapStr) error
	close        func() error
	connect      func(time.Duration) error
}

func enableLogging(selectors []string) {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, selectors)
	}
}

func (c *mockClient) Connect(timeout time.Duration) error {
	return c.connect(timeout)
}

func (c *mockClient) Close() error {
	return c.close()
}

func (c *mockClient) PublishEvents(events []common.MapStr) ([]common.MapStr, error) {
	return c.publish(events)
}

func (c *mockClient) PublishEvent(event common.MapStr) error {
	_, err := c.PublishEvents([]common.MapStr{event})
	return err
}

func (c *mockClient) AsyncPublishEvents(cb func([]common.MapStr, error), events []common.MapStr) error {
	return c.asyncPublish(cb, events)
}

func (c *mockClient) AsyncPublishEvent(cb func(error), event common.MapStr) error {
	return c.AsyncPublishEvents(
		func(evts []common.MapStr, err error) { cb(err) },
		[]common.MapStr{event})
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
) func(events []common.MapStr) ([]common.MapStr, error) {
	mutex := sync.Mutex{}
	return func(events []common.MapStr) ([]common.MapStr, error) {
		mutex.Lock()
		defer mutex.Unlock()

		*collected = append(*collected, events)
		return nil, nil
	}
}

func asyncCollectPublish(
	collected *[][]common.MapStr,
) func(func([]common.MapStr, error), []common.MapStr) error {
	mutex := sync.Mutex{}
	return func(cb func([]common.MapStr, error), events []common.MapStr) error {
		mutex.Lock()
		defer mutex.Unlock()

		*collected = append(*collected, events)
		cb(nil, nil)
		return nil
	}
}

type errNetTimeout struct{}

func (e errNetTimeout) Error() string   { return "errNetTimeout" }
func (e errNetTimeout) Timeout() bool   { return true }
func (e errNetTimeout) Temporary() bool { return false }

func publishTimeoutEvery(
	n int,
	pub func(events []common.MapStr) ([]common.MapStr, error),
) func(events []common.MapStr) ([]common.MapStr, error) {
	count := 0
	return func(events []common.MapStr) ([]common.MapStr, error) {
		if count < n {
			count++
			return pub(events)
		}
		count = 0
		return events, errNetTimeout{}
	}
}

func publishFailWith(
	n int,
	err error,
	pub func([]common.MapStr) ([]common.MapStr, error),
) func([]common.MapStr) ([]common.MapStr, error) {
	count := 0
	return func(events []common.MapStr) ([]common.MapStr, error) {
		if count < n {
			count++
			return events, err
		}
		count = 0
		return pub(events)
	}
}

func publishFailStart(
	n int,
	pub func(events []common.MapStr) ([]common.MapStr, error),
) func(events []common.MapStr) ([]common.MapStr, error) {
	return publishFailWith(n, errNetTimeout{}, pub)
}

func asyncFailStart(
	n int,
	pub func(func([]common.MapStr, error), []common.MapStr) error,
) func(func([]common.MapStr, error), []common.MapStr) error {
	return asyncFailStartWith(n, errNetTimeout{}, pub)
}

func asyncFailStartWith(
	n int,
	err error,
	pub func(func([]common.MapStr, error), []common.MapStr) error,
) func(func([]common.MapStr, error), []common.MapStr) error {
	count := 0
	return func(cb func([]common.MapStr, error), events []common.MapStr) error {
		if count < n {
			count++
			debug("fail with(%v): %v", count, err)
			return err
		}

		count = 0
		debug("forward events")
		return pub(cb, events)
	}
}

func asyncFailWith(
	n int,
	err error,
	pub func(func([]common.MapStr, error), []common.MapStr) error,
) func(func([]common.MapStr, error), []common.MapStr) error {
	count := 0
	return func(cb func([]common.MapStr, error), events []common.MapStr) error {
		if count < n {
			count++
			go func() {
				cb(events, err)
			}()
			return nil
		}

		count = 0
		return pub(cb, events)
	}
}

func closeOK() error {
	return nil
}

var testEvent = common.MapStr{
	"msg": "hello world",
}

var testNoOpts = outputs.Options{}
var testGuaranteed = outputs.Options{Guaranteed: true}

func testMode(
	t *testing.T,
	mode ConnectionMode,
	opts outputs.Options,
	events []eventInfo,
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
		if pubEvents.single {
			numSignals += len(pubEvents.events)
		} else {
			numSignals++
		}
	}

	var expectedEvents [][]common.MapStr
	ch := make(chan op.SignalResponse, numSignals)
	signal := &op.SignalChannel{ch}
	idx := 0
	for _, pubEvents := range events {
		if pubEvents.single {
			for _, event := range pubEvents.events {
				_ = mode.PublishEvent(signal, opts, event)
				if expectedSignals[idx] {
					expectedEvents = append(expectedEvents, []common.MapStr{event})
				}
				idx++
			}
		} else {
			_ = mode.PublishEvents(signal, opts, pubEvents.events)
			if expectedSignals[idx] {
				expectedEvents = append(expectedEvents, pubEvents.events)
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
		for i := range *collectedEvents {
			expected := expectedEvents[i]
			actual := (*collectedEvents)[i]
			assert.Equal(t, expected, actual)
		}
	}
}

type eventInfo struct {
	single bool
	events []common.MapStr
}

func singleEvent(e common.MapStr) []eventInfo {
	events := []common.MapStr{e}
	return []eventInfo{
		{single: true, events: events},
	}
}

func multiEvent(n int, event common.MapStr) []eventInfo {
	var events []common.MapStr
	for i := 0; i < n; i++ {
		events = append(events, event)
	}
	return []eventInfo{{single: false, events: events}}
}

func repeat(n int, evt []eventInfo) []eventInfo {
	var events []eventInfo
	for _, e := range evt {
		events = append(events, e)
	}
	return events
}

func signals(s ...bool) []bool {
	return s
}

func dummyMockClientFactory(host string) (ProtocolClient, error) {
	return &mockClient{
		close: func() error { return nil },
	}, nil
}
