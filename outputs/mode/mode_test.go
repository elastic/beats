package mode

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/outputs"
	"github.com/stretchr/testify/assert"
)

type mockClient struct {
	publish   func([]common.MapStr) ([]common.MapStr, error)
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

func (c *mockClient) PublishEvents(events []common.MapStr) ([]common.MapStr, error) {
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
) func(events []common.MapStr) ([]common.MapStr, error) {
	mutex := sync.Mutex{}
	return func(events []common.MapStr) ([]common.MapStr, error) {
		mutex.Lock()
		defer mutex.Unlock()

		*collected = append(*collected, events)
		return nil, nil
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

func closeOK() error {
	return nil
}

var testEvent = common.MapStr{
	"msg": "hello world",
}

func testMode(
	t *testing.T,
	mode ConnectionMode,
	events []eventInfo,
	expectedSignals []bool,
	collectedEvents *[][]common.MapStr,
) {
	defer mode.Close()

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
	ch := make(chan bool, numSignals)
	signal := outputs.NewChanSignal(ch)
	idx := 0
	for _, pubEvents := range events {
		if pubEvents.single {
			for _, event := range pubEvents.events {
				_ = mode.PublishEvent(signal, event)
				if expectedSignals[idx] {
					expectedEvents = append(expectedEvents, []common.MapStr{event})
				}
				idx++
			}
		} else {
			_ = mode.PublishEvents(signal, pubEvents.events)
			if expectedSignals[idx] {
				expectedEvents = append(expectedEvents, pubEvents.events)
			}
			idx++
		}
	}

	results := make([]bool, len(expectedSignals))
	for i := 0; i < len(expectedSignals); i++ {
		results[i] = <-ch
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

func TestMakeEmptyClientFail(t *testing.T) {
	config := outputs.MothershipConfig{}
	clients, err := MakeClients(config, dummyMockClientFactory)
	assert.Equal(t, ErrNoHostsConfigured, err)
	assert.Equal(t, 0, len(clients))
}

func TestMakeSingleClient(t *testing.T) {
	config := outputs.MothershipConfig{
		Hosts: []string{"single"},
	}

	clients, err := MakeClients(config, dummyMockClientFactory)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(clients))
}

func TestMakeSingleClientWorkers(t *testing.T) {
	config := outputs.MothershipConfig{
		Hosts:  []string{"single"},
		Worker: 3,
	}

	clients, err := MakeClients(config, dummyMockClientFactory)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(clients))
}

func TestMakeTwoClient(t *testing.T) {
	config := outputs.MothershipConfig{
		Hosts: []string{"client1", "client2"},
	}

	clients, err := MakeClients(config, dummyMockClientFactory)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(clients))
}

func TestMakeTwoClientWorkers(t *testing.T) {
	config := outputs.MothershipConfig{
		Hosts:  []string{"client1", "client2"},
		Worker: 3,
	}

	clients, err := MakeClients(config, dummyMockClientFactory)
	assert.Nil(t, err)
	assert.Equal(t, 6, len(clients))
}

func TestMakeTwoClientFail(t *testing.T) {
	config := outputs.MothershipConfig{
		Hosts:  []string{"client1", "client2"},
		Worker: 3,
	}

	testError := errors.New("test")

	i := 1
	_, err := MakeClients(config, func(host string) (ProtocolClient, error) {
		if i%3 == 0 {
			return nil, testError
		}
		i++
		return dummyMockClientFactory(host)
	})
	assert.Equal(t, testError, err)
}

func dummyMockClientFactory(host string) (ProtocolClient, error) {
	return &mockClient{
		close: func() error { return nil },
	}, nil
}
