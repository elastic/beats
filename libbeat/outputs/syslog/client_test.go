// +build !integration

package syslog

import (
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs/transport"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockClient struct {
	mock.Mock
}

type MockTransportClient struct {
	mock.Mock
}

func TestCreateSyslogString(t *testing.T) {
	tc := &transport.Client{}
	c := newClient(tc, "testprogram", 1, 6)
	c.Hostname = "localhost"

	timestamp, _ := time.Parse(time.RFC3339, "2016-04-28T18:59:52Z")
	parsed_time := common.Time(timestamp)
	var message = new(string)
	*message = "test"
	event := common.MapStr{"message": message, "@timestamp": parsed_time}

	line, _ := c.CreateSyslogString(event)

	// Assert that the resulting line matches expectations
	line_control := "<14>2016-04-28T18:59:52Z localhost testprogram: test\n"
	assert.Equal(t, line, line_control, "Both lines should match.")
}
