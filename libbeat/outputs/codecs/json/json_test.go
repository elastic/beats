package json

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"
)

func TestJsonCodec(t *testing.T) {
	expectedValue := "{\"msg\":\"message\"}"

	codec := New(false)
	output, err := codec.Encode(common.MapStr{"msg": "message"})

	if err != nil {
		t.Errorf("Error during event write %v", err)
	} else {
		if string(output) != expectedValue {
			t.Errorf("Expected value (%s) does not equal with output (%s)", expectedValue, output)
		}
	}
}

func TestJsonWriterPrettyPrint(t *testing.T) {
	expectedValue := "{\n  \"msg\": \"message\"\n}"

	codec := New(true)
	output, err := codec.Encode(common.MapStr{"msg": "message"})

	if err != nil {
		t.Errorf("Error during event write %v", err)
	} else {
		if string(output) != expectedValue {
			t.Errorf("Expected value (%s) does not equal with output (%s)", expectedValue, output)
		}
	}
}
