package decode_duration

import (
	"math"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
)

func testProcessor() *decodeDuration {
	return &decodeDuration{
		config: decodeDurationConfig{
			Field:  "duration",
			Format: "",
		},
	}
}

func TestDecodeDuration(t *testing.T) {
	cases := []struct {
		Duration time.Duration
		Format   string
		Result   float64
	}{
		{time.Second + time.Millisecond, "", 1001},
		{time.Second + time.Millisecond, "milliseconds", 1001},
		{time.Second + time.Millisecond, "seconds", 1.001},
		{3 * time.Second, "minutes", 0.05},
		{3 * time.Minute, "hours", 0.05},
	}
	evt := &beat.Event{Fields: common.MapStr{}}
	c := testProcessor()

	for _, testCase := range cases {
		c.config.Format = testCase.Format
		if _, err := evt.PutValue("duration", testCase.Duration.String()); err != nil {
			t.Fatal(err)
		}
		evt, err := c.Run(evt)
		if err != nil {
			t.Fatal(err)
		}
		d, err := evt.GetValue("duration")
		if err != nil {
			t.Fatal(err)
		}
		floatD, ok := d.(float64)
		if !ok {
			t.Fatal("result value is not duration")
		}
		floatD = math.Round(floatD*math.Pow10(6)) / math.Pow10(6)
		if floatD != testCase.Result {
			t.Fatalf("test case except: %f, actual: %f", testCase.Result, floatD)
		}
	}
}
