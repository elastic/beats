package json

import (
	"testing"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

func TestJsonCodec(t *testing.T) {
	expectedValue := `{"@timestamp":"0001-01-01T00:00:00.000Z","@metadata":{"beat":"test","type":"doc","version":"1.2.3"},"msg":"message"}`

	codec := New(false, "1.2.3")
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
    "type": "doc",
    "version": "1.2.3"
  },
  "msg": "message"
}`

	codec := New(true, "1.2.3")
	output, err := codec.Encode("test", &beat.Event{Fields: common.MapStr{"msg": "message"}})

	if err != nil {
		t.Errorf("Error during event write %v", err)
	} else {
		if string(output) != expectedValue {
			t.Errorf("Expected value (%s) does not equal with output (%s)", expectedValue, output)
		}
	}
}
