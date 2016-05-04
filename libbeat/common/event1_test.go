package common

import (
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/stretchr/testify/assert"
)

func TestCreateEvent(t *testing.T) {
	logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"*"})

	event := Event{
		"@timestamp": Timestamp(time.Now()),
		"code":       Int(30),
		"type":       Str("test"),
		"http": Nested(Dict{
			"response": Str("200 OK"),
		}),
	}
	assert.Equal(t, event["code"].Int(), 30)
	assert.Equal(t, event["type"].Str(), "test")
	assert.Equal(t, event["http"].String(), "map[response:200 OK]")
	logp.Debug("event", "%v", event)
}
