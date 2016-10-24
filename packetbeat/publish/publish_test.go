// +build !integration

package publish

import (
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/publisher"
	"github.com/stretchr/testify/assert"
)

func testEvent() common.MapStr {
	event := common.MapStr{}
	event["@timestamp"] = common.Time(time.Now())
	event["type"] = "test"
	event["src"] = &common.Endpoint{}
	event["dst"] = &common.Endpoint{}
	return event
}

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
		}, "invalid '@timestamp'"},

		{func() common.MapStr {
			m := testEvent()
			delete(m, "@timestamp")
			return m
		}, "missing '@timestamp'"},

		{func() common.MapStr {
			m := testEvent()
			delete(m, "type")
			return m
		}, "missing 'type'"},

		{func() common.MapStr {
			m := testEvent()
			m["type"] = 123
			return m
		}, "invalid 'type'"},
	}

	for _, test := range testCases {
		assert.Regexp(t, test.err, validateEvent(test.f()))
	}
}

func TestDirectionOut(t *testing.T) {
	publisher := newTestPublisher([]string{"192.145.2.4"})
	ppub, _ := NewPublisher(publisher, 1000, 1, false)

	event := common.MapStr{
		"src": &common.Endpoint{
			IP:      "192.145.2.4",
			Port:    3267,
			Name:    "server1",
			Cmdline: "proc1 start",
			Proc:    "proc1",
		},
		"dst": &common.Endpoint{
			IP:      "192.145.2.5",
			Port:    32232,
			Name:    "server2",
			Cmdline: "proc2 start",
			Proc:    "proc2",
		},
	}

	assert.True(t, ppub.normalizeTransAddr(event))
	assert.True(t, event["client_ip"] == "192.145.2.4")
	assert.True(t, event["direction"] == "out")
}

func TestDirectionIn(t *testing.T) {
	publisher := newTestPublisher([]string{"192.145.2.5"})
	ppub, _ := NewPublisher(publisher, 1000, 1, false)

	event := common.MapStr{
		"src": &common.Endpoint{
			IP:      "192.145.2.4",
			Port:    3267,
			Name:    "server1",
			Cmdline: "proc1 start",
			Proc:    "proc1",
		},
		"dst": &common.Endpoint{
			IP:      "192.145.2.5",
			Port:    32232,
			Name:    "server2",
			Cmdline: "proc2 start",
			Proc:    "proc2",
		},
	}

	assert.True(t, ppub.normalizeTransAddr(event))
	assert.True(t, event["client_ip"] == "192.145.2.4")
	assert.True(t, event["direction"] == "in")
}

func newTestPublisher(ips []string) *publisher.BeatPublisher {
	p := &publisher.BeatPublisher{}
	p.IPAddrs = ips
	return p
}

func TestNoDirection(t *testing.T) {
	publisher := newTestPublisher([]string{"192.145.2.6"})
	ppub, _ := NewPublisher(publisher, 1000, 1, false)

	event := common.MapStr{
		"src": &common.Endpoint{
			IP:      "192.145.2.4",
			Port:    3267,
			Name:    "server1",
			Cmdline: "proc1 start",
			Proc:    "proc1",
		},
		"dst": &common.Endpoint{
			IP:      "192.145.2.5",
			Port:    32232,
			Name:    "server2",
			Cmdline: "proc2 start",
			Proc:    "proc2",
		},
	}

	assert.True(t, ppub.normalizeTransAddr(event))
	assert.True(t, event["client_ip"] == "192.145.2.4")
	_, ok := event["direction"]
	assert.False(t, ok)
}
