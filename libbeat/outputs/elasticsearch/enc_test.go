package elasticsearch

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/monitoring/report"
)

func TestJSONEncoderMarshalBeatEvent(t *testing.T) {
	encoder := newJSONEncoder(nil)
	event := beat.Event{
		Timestamp: time.Date(2017, time.November, 7, 12, 0, 0, 0, time.UTC),
		Fields: common.MapStr{
			"field1": "value1",
		},
	}

	err := encoder.Marshal(event)
	if err != nil {
		t.Errorf("Error while marshaling beat.Event using JSONEncoder: %v", err)
	}
	assert.Equal(t, encoder.buf.String(), "{\"@timestamp\":\"2017-11-07T12:00:00.000Z\",\"field1\":\"value1\"}\n",
		"Unexpected marshaled format of beat.Event")
}

func TestJSONEncoderMarshalMonitoringEvent(t *testing.T) {
	encoder := newJSONEncoder(nil)
	event := report.Event{
		Timestamp: time.Date(2017, time.November, 7, 12, 0, 0, 0, time.UTC),
		Fields: common.MapStr{
			"field1": "value1",
		},
	}

	err := encoder.Marshal(event)
	if err != nil {
		t.Errorf("Error while marshaling report.Event using JSONEncoder: %v", err)
	}
	assert.Equal(t, encoder.buf.String(), "{\"timestamp\":\"2017-11-07T12:00:00.000Z\",\"field1\":\"value1\"}\n",
		"Unexpected marshaled format of report.Event")
}
