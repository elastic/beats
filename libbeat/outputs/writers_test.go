package outputs

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/fmtstr"
	"testing"
)

func event(k, v string) common.MapStr {
	return common.MapStr{k: v}
}

func TestFormatStringWriter(t *testing.T) {
	format := fmtstr.MustCompileEvent("test %{[msg]}")
	expectedValue := "test message"

	formatStringWriter := NewFormatStringWriter(format)
	output, err := formatStringWriter.Write(event("msg", "message"))

	if err != nil {
		t.Errorf("Error during event write %v", err)
	} else {
		if string(output) != expectedValue {
			t.Errorf("Expected value (%s) does not equal with output %s", expectedValue, output)
		}
	}
}

func TestJsonWriter(t *testing.T) {
	expectedValue := "{\"msg\":\"message\"}"

	formatStringWriter := NewJsonWriter(false)
	output, err := formatStringWriter.Write(event("msg", "message"))

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

	formatStringWriter := NewJsonWriter(true)
	output, err := formatStringWriter.Write(event("msg", "message"))

	if err != nil {
		t.Errorf("Error during event write %v", err)
	} else {
		if string(output) != expectedValue {
			t.Errorf("Expected value (%s) does not equal with output (%s)", expectedValue, output)
		}
	}
}

func TestCreatWriterDefaultsToJsonWriterWithoutPrettyprint(t *testing.T) {
	expectedValue := "{\"msg\":\"message\"}"

	formatStringWriter := CreateWriter(WriterConfig{})
	output, err := formatStringWriter.Write(event("msg", "message"))

	if err != nil {
		t.Errorf("Error during event write %v", err)
	} else {
		if string(output) != expectedValue {
			t.Errorf("Expected value (%s) does not equal with output (%s)", expectedValue, output)
		}
	}
}
