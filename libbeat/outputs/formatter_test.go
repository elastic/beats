package outputs

import (
	"fmt"
	"github.com/elastic/beats/libbeat/common"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestFormatterWithSimpleMessage(t *testing.T) {
	curTime := common.Time(time.Now())

	event := common.MapStr{
		"@timestamp": curTime,
		"host":       "test-host",
		"type":       "log",
		"message":    "test",
	}

	formatter := "%{message}"

	formattedEvent, _ := FormatEvent(event, formatter)

	assert.Equal(t, "test", string(formattedEvent))

}

func TestFormatterWithTimestampedFormat(t *testing.T) {
	curTime := common.Time(time.Now())

	event := common.MapStr{
		"@timestamp": curTime,
		"host":       "test-host",
		"type":       "log",
		"message":    "test",
	}

	formatter := "%{@timestamp} %{message}"

	formattedEvent, _ := FormatEvent(event, formatter)

	assert.Equal(t, fmt.Sprintf("%s %s", curTime, "test"), string(formattedEvent))

}

func TestFormatterWithNonExistentKey(t *testing.T) {
	curTime := common.Time(time.Now())

	event := common.MapStr{
		"@timestamp": curTime,
		"host":       "test-host",
		"type":       "log",
		"message":    "test",
	}

	formatter := "%{@timestamp} %{nonexistent}"

	formattedEvent, _ := FormatEvent(event, formatter)

	assert.Equal(t, fmt.Sprintf("%s ", curTime), string(formattedEvent))

}
