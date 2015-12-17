package publisher

import (
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/stretchr/testify/assert"
)

// Test that FilterEvent detects events that do not contain the required fields
// and returns error.
func TestFilterEvent(t *testing.T) {
	var testCases = []struct {
		f   func() common.MapStr
		err string
	}{
		{func() common.MapStr {
			return testEvent()
		}, ""},

		{func() common.MapStr {
			m := testEvent()
			m["@timestamp"] = time.Now()
			return m
		}, "Invalid '@timestamp'"},

		{func() common.MapStr {
			m := testEvent()
			delete(m, "@timestamp")
			return m
		}, "Missing '@timestamp'"},

		{func() common.MapStr {
			m := testEvent()
			delete(m, "type")
			return m
		}, "Missing 'type'"},

		{func() common.MapStr {
			m := testEvent()
			m["type"] = 123
			return m
		}, "Invalid 'type'"},
	}

	for _, test := range testCases {
		assert.Regexp(t, test.err, filterEvent(test.f()))
	}
}

func TestDirectionOut(t *testing.T) {
	publisher := PublisherType{}

	publisher.ipaddrs = []string{"192.145.2.4"}

	event := common.MapStr{
		"src": &common.Endpoint{
			Ip:      "192.145.2.4",
			Port:    3267,
			Name:    "server1",
			Cmdline: "proc1 start",
			Proc:    "proc1",
		},
		"dst": &common.Endpoint{
			Ip:      "192.145.2.5",
			Port:    32232,
			Name:    "server2",
			Cmdline: "proc2 start",
			Proc:    "proc2",
		},
	}

	assert.True(t, updateEventAddresses(&publisher, event))
	assert.True(t, event["client_ip"] == "192.145.2.4")
	assert.True(t, event["direction"] == "out")
}

func TestDirectionIn(t *testing.T) {
	publisher := PublisherType{}

	publisher.ipaddrs = []string{"192.145.2.5"}

	event := common.MapStr{
		"src": &common.Endpoint{
			Ip:      "192.145.2.4",
			Port:    3267,
			Name:    "server1",
			Cmdline: "proc1 start",
			Proc:    "proc1",
		},
		"dst": &common.Endpoint{
			Ip:      "192.145.2.5",
			Port:    32232,
			Name:    "server2",
			Cmdline: "proc2 start",
			Proc:    "proc2",
		},
	}

	assert.True(t, updateEventAddresses(&publisher, event))
	assert.True(t, event["client_ip"] == "192.145.2.4")
	assert.True(t, event["direction"] == "in")
}

func TestNoDirection(t *testing.T) {
	publisher := PublisherType{}

	publisher.ipaddrs = []string{"192.145.2.6"}

	event := common.MapStr{
		"src": &common.Endpoint{
			Ip:      "192.145.2.4",
			Port:    3267,
			Name:    "server1",
			Cmdline: "proc1 start",
			Proc:    "proc1",
		},
		"dst": &common.Endpoint{
			Ip:      "192.145.2.5",
			Port:    32232,
			Name:    "server2",
			Cmdline: "proc2 start",
			Proc:    "proc2",
		},
	}

	assert.True(t, updateEventAddresses(&publisher, event))
	assert.True(t, event["client_ip"] == "192.145.2.4")
	_, ok := event["direction"]
	assert.False(t, ok)
}
