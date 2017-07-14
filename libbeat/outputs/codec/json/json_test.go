package json

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/publisher/beat"
)

func TestJsonCodec(t *testing.T) {
	expectedValue := `{"@timestamp":"0001-01-01T00:00:00.000Z","@metadata":{"beat":"test","type":"doc"},"msg":"message"}`

	codec := New(false)
	output, err := codec.Encode("test", &beat.Event{Fields: common.MapStr{"msg": "message"}})

	if err != nil {
		t.Errorf("Error during event write %v", err)
	} else {
		if string(output) != expectedValue {
			t.Errorf("Expected value (%s) does not equal with output (%s)", expectedValue, output)
		}
	}
}

func TestJsonWriterPrettyPrint(t *testing.T) {
	expectedValue := `{
  "@timestamp": "0001-01-01T00:00:00.000Z",
  "@metadata": {
    "beat": "test",
    "type": "doc"
  },
  "msg": "message"
}`

	codec := New(true)
	output, err := codec.Encode("test", &beat.Event{Fields: common.MapStr{"msg": "message"}})

	if err != nil {
		t.Errorf("Error during event write %v", err)
	} else {
		if string(output) != expectedValue {
			t.Errorf("Expected value (%s) does not equal with output (%s)", expectedValue, output)
		}
	}
}
