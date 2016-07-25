// +build !integration

package reader

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/stretchr/testify/assert"
)

func TestDecodeJSON(t *testing.T) {
	type io struct {
		Text         string
		Config       JSONConfig
		ExpectedText string
		ExpectedMap  common.MapStr
	}

	var tests = []io{
		{
			Text:         `{"message": "test", "value": 1}`,
			Config:       JSONConfig{MessageKey: "message"},
			ExpectedText: "test",
			ExpectedMap:  common.MapStr{"message": "test", "value": float64(1)},
		},
		{
			Text:         `{"message": "test", "value": 1}`,
			Config:       JSONConfig{MessageKey: "message1"},
			ExpectedText: "",
			ExpectedMap:  common.MapStr{"message": "test", "value": float64(1)},
		},
		{
			Text:         `{"message": "test", "value": 1}`,
			Config:       JSONConfig{MessageKey: "value"},
			ExpectedText: "",
			ExpectedMap:  common.MapStr{"message": "test", "value": float64(1)},
		},
		{
			Text:         `{"message": "test", "value": "1"}`,
			Config:       JSONConfig{MessageKey: "value"},
			ExpectedText: "1",
			ExpectedMap:  common.MapStr{"message": "test", "value": "1"},
		},
		{
			// in case of JSON decoding errors, the text is passed as is
			Text:         `{"message": "test", "value": "`,
			Config:       JSONConfig{MessageKey: "value"},
			ExpectedText: `{"message": "test", "value": "`,
			ExpectedMap:  nil,
		},
		{
			// Add key error helps debugging this
			Text:         `{"message": "test", "value": "`,
			Config:       JSONConfig{MessageKey: "value", AddErrorKey: true},
			ExpectedText: `{"message": "test", "value": "`,
			ExpectedMap:  common.MapStr{"json_error": "Error decoding JSON: unexpected end of JSON input"},
		},
		{
			// If the text key is not found, put an error
			Text:         `{"message": "test", "value": "1"}`,
			Config:       JSONConfig{MessageKey: "hello", AddErrorKey: true},
			ExpectedText: ``,
			ExpectedMap:  common.MapStr{"message": "test", "value": "1", "json_error": "Key 'hello' not found"},
		},
		{
			// If the text key is found, but not a string, put an error
			Text:         `{"message": "test", "value": 1}`,
			Config:       JSONConfig{MessageKey: "value", AddErrorKey: true},
			ExpectedText: ``,
			ExpectedMap:  common.MapStr{"message": "test", "value": float64(1), "json_error": "Value of key 'value' is not a string"},
		},
		{
			// Without a text key, simple return the json and an empty text
			Text:         `{"message": "test", "value": 1}`,
			Config:       JSONConfig{AddErrorKey: true},
			ExpectedText: ``,
			ExpectedMap:  common.MapStr{"message": "test", "value": float64(1)},
		},
	}

	for _, test := range tests {

		var p JSON
		p.cfg = &test.Config
		text, map_ := p.decodeJSON([]byte(test.Text))
		assert.Equal(t, test.ExpectedText, string(text))
		assert.Equal(t, test.ExpectedMap, map_)
	}
}
