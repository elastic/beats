package synthexec

import (
	"reflect"
	"testing"
	"time"
)



func TestJsonToSynthEvent(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		synthEvent *SynthEvent
	}{
		{
			name: "an empty line",
			line: "",
			synthEvent: nil,
		},
		{
			name: "a blank line",
			line: "   ",
			synthEvent: nil,
		},
		{
			name: "a valid line",
			line: `{\"@timestamp\":\"2020-09-19T02:46:15.759Z\",\"type\":\"step/end\",\"journey\":{\"name\":\"inline\",\"id\":\"inline\"},\"step\":{\"name\":\"Go to home page\",\"index\":0},\"payload\":{\"source\":\"async ({page, params}) => {\\n  await page.goto(\\\"http://www.elastic.co\\\");\\n}\",\"duration_ms\":3472,\"url\":\"https://www.elastic.co/\",\"status\":\"succeeded\"},\"url\":\"https://www.elastic.co/\",\"package_version\":\"0.0.1\"}`,
			synthEvent: &SynthEvent{
				Timestamp: time.Now(),
				Type: "step/end",
				Journey: &Journey{
					Name: "inline",
					Id: "inline",
				},
				Step: &Step{
					Name: "Go to homepage",
					Index: 0,
				},
				Payload: map[string]interface{}{
					"source": "async ({page, params}) => {\n  await page.goto(\"http://www.elastic.co\")\n}",
					"duration_ms": 3472,
					"url": "https://www.elastic.co/",
					"status": "succeeded",
				},
				PackageVersion: "0.0.1",
				URL: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotRes := jsonToSynthEvent([]byte(tt.line), tt.line); !reflect.DeepEqual(gotRes, tt.synthEvent) {
				t.Errorf("jsonToSynthEvent() = %v, want %v", gotRes, tt.synthEvent)
			}
		})
	}
}
