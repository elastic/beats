package reader

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/stretchr/testify/assert"
)

func TestUnmarshal(t *testing.T) {
	tests := []struct {
		Name   string
		Input  string
		Output map[string]interface{}
	}{
		{
			Name:  "Top level int, float, string, bool",
			Input: `{"a": 3, "b": 2.0, "c": "hello", "d": true}`,
			Output: map[string]interface{}{
				"a": int64(3),
				"b": float64(2),
				"c": "hello",
				"d": true,
			},
		},
		{
			Name:  "Nested objects with ints",
			Input: `{"a": 3, "b": {"c": {"d": 5}}}`,
			Output: map[string]interface{}{
				"a": int64(3),
				"b": map[string]interface{}{
					"c": map[string]interface{}{
						"d": int64(5),
					},
				},
			},
		},
		{
			Name:  "Array of floats",
			Input: `{"a": 3, "b": {"c": [4.0, 4.1, 4.2]}}`,
			Output: map[string]interface{}{
				"a": int64(3),
				"b": map[string]interface{}{
					"c": []interface{}{
						float64(4.0), float64(4.1), float64(4.2),
					},
				},
			},
		},
		{
			Name:  "Array of mixed ints and floats",
			Input: `{"a": 3, "b": {"c": [4, 4.1, 4.2]}}`,
			Output: map[string]interface{}{
				"a": int64(3),
				"b": map[string]interface{}{
					"c": []interface{}{
						int64(4), float64(4.1), float64(4.2),
					},
				},
			},
		},
		{
			Name:  "Negative values",
			Input: `{"a": -3, "b": -1.0}`,
			Output: map[string]interface{}{
				"a": int64(-3),
				"b": float64(-1),
			},
		},
	}

	for _, test := range tests {
		t.Logf("Running test %s", test.Name)
		var output map[string]interface{}
		err := unmarshal([]byte(test.Input), &output)
		assert.NoError(t, err)
		assert.Equal(t, test.Output, output)
	}
}

func TestDecodeJSON(t *testing.T) {
	var tests = []struct {
		Text         string
		Config       JSONConfig
		ExpectedText string
		ExpectedMap  common.MapStr
	}{
		{
			Text:         `{"message": "test", "value": 1}`,
			Config:       JSONConfig{MessageKey: "message"},
			ExpectedText: "test",
			ExpectedMap:  common.MapStr{"message": "test", "value": int64(1)},
		},
		{
			Text:         `{"message": "test", "value": 1}`,
			Config:       JSONConfig{MessageKey: "message1"},
			ExpectedText: "",
			ExpectedMap:  common.MapStr{"message": "test", "value": int64(1)},
		},
		{
			Text:         `{"message": "test", "value": 1}`,
			Config:       JSONConfig{MessageKey: "value"},
			ExpectedText: "",
			ExpectedMap:  common.MapStr{"message": "test", "value": int64(1)},
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
			ExpectedMap:  common.MapStr{"json_error": "Error decoding JSON: unexpected EOF"},
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
			ExpectedMap:  common.MapStr{"message": "test", "value": int64(1), "json_error": "Value of key 'value' is not a string"},
		},
		{
			// Without a text key, simple return the json and an empty text
			Text:         `{"message": "test", "value": 1}`,
			Config:       JSONConfig{AddErrorKey: true},
			ExpectedText: ``,
			ExpectedMap:  common.MapStr{"message": "test", "value": int64(1)},
		},
	}

	for _, test := range tests {

		var p JSON
		p.cfg = &test.Config
		text, M := p.decodeJSON([]byte(test.Text))
		assert.Equal(t, test.ExpectedText, string(text))
		assert.Equal(t, test.ExpectedMap, M)
	}
}
