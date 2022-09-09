// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package readjson

import (
	"testing"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common"
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
		t.Run(test.Name, func(t *testing.T) {
			var output map[string]interface{}
			err := unmarshal([]byte(test.Input), &output)
			assert.NoError(t, err)
			assert.Equal(t, test.Output, output)
		})

	}
}

func TestDecodeJSON(t *testing.T) {
	var tests = []struct {
		Text         string
		Config       Config
		ExpectedText string
		ExpectedMap  mapstr.M
	}{
		{
			Text:         `{"message": "test", "value": 1}`,
			Config:       Config{MessageKey: "message"},
			ExpectedText: "test",
			ExpectedMap:  mapstr.M{"message": "test", "value": int64(1)},
		},
		{
			Text:         `{"message": "test", "value": 1}`,
			Config:       Config{MessageKey: "message1"},
			ExpectedText: "",
			ExpectedMap:  mapstr.M{"message": "test", "value": int64(1)},
		},
		{
			Text:         `{"message": "test", "value": 1}`,
			Config:       Config{MessageKey: "value"},
			ExpectedText: "",
			ExpectedMap:  mapstr.M{"message": "test", "value": int64(1)},
		},
		{
			Text:         `{"message": "test", "value": "1"}`,
			Config:       Config{MessageKey: "value"},
			ExpectedText: "1",
			ExpectedMap:  mapstr.M{"message": "test", "value": "1"},
		},
		{
			// in case of JSON decoding errors, the text is passed as is
			Text:         `{"message": "test", "value": "`,
			Config:       Config{MessageKey: "value"},
			ExpectedText: `{"message": "test", "value": "`,
			ExpectedMap:  nil,
		},
		{
			// in case the JSON is "null", we should just not panic
			Text:         `null`,
			Config:       Config{MessageKey: "value", AddErrorKey: true},
			ExpectedText: `null`,
			ExpectedMap:  mapstr.M{"error": mapstr.M{"message": "Error decoding JSON: <nil>", "type": "json"}},
		},
		{
			// Add key error helps debugging this
			Text:         `{"message": "test", "value": "`,
			Config:       Config{MessageKey: "value", AddErrorKey: true},
			ExpectedText: `{"message": "test", "value": "`,
			ExpectedMap:  mapstr.M{"error": mapstr.M{"message": "Error decoding JSON: unexpected EOF", "type": "json"}},
		},
		{
			// If the text key is not found, put an error
			Text:         `{"message": "test", "value": "1"}`,
			Config:       Config{MessageKey: "hello", AddErrorKey: true},
			ExpectedText: ``,
			ExpectedMap:  mapstr.M{"message": "test", "value": "1", "error": mapstr.M{"message": "Key 'hello' not found", "type": "json"}},
		},
		{
			// If the text key is found, but not a string, put an error
			Text:         `{"message": "test", "value": 1}`,
			Config:       Config{MessageKey: "value", AddErrorKey: true},
			ExpectedText: ``,
			ExpectedMap:  mapstr.M{"message": "test", "value": int64(1), "error": mapstr.M{"message": "Value of key 'value' is not a string", "type": "json"}},
		},
		{
			// Without a text key, simple return the json and an empty text
			Text:         `{"message": "test", "value": 1}`,
			Config:       Config{AddErrorKey: true},
			ExpectedText: ``,
			ExpectedMap:  mapstr.M{"message": "test", "value": int64(1)},
		},
		{
			// If AddErrorKey set to false, error event should not be set.
			Text:         `{"message": "test", "value": "`,
			Config:       Config{MessageKey: "value", AddErrorKey: false},
			ExpectedText: `{"message": "test", "value": "`,
			ExpectedMap:  nil,
		},
	}

	for _, test := range tests {

		var p JSONReader
		p.cfg = &test.Config
		p.logger = logp.NewLogger("json_test")
		text, M := p.decode([]byte(test.Text))
		assert.Equal(t, test.ExpectedText, string(text))
		assert.Equal(t, test.ExpectedMap, M)
	}
}

func TestMergeJSONFields(t *testing.T) {
	type io struct {
	}

	text := "hello"

	now := time.Now().UTC()

	tests := map[string]struct {
		Data              mapstr.M
		Text              *string
		JSONConfig        Config
		ExpectedItems     mapstr.M
		ExpectedTimestamp time.Time
		ExpectedID        string
	}{
		"default: do not overwrite": {
			Data:       mapstr.M{"type": "test_type", "json": mapstr.M{"type": "test", "text": "hello"}},
			Text:       &text,
			JSONConfig: Config{KeysUnderRoot: true},
			ExpectedItems: mapstr.M{
				"type": "test_type",
				"text": "hello",
			},
			ExpectedTimestamp: time.Time{},
		},
		"overwrite keys if configured": {
			Data:       mapstr.M{"type": "test_type", "json": mapstr.M{"type": "test", "text": "hello"}},
			Text:       &text,
			JSONConfig: Config{KeysUnderRoot: true, OverwriteKeys: true},
			ExpectedItems: mapstr.M{
				"type": "test",
				"text": "hello",
			},
			ExpectedTimestamp: time.Time{},
		},
		"use json namespace w/o keys_under_root": {
			// without keys_under_root, put everything in a json key
			Data:       mapstr.M{"type": "test_type", "json": mapstr.M{"type": "test", "text": "hello"}},
			Text:       &text,
			JSONConfig: Config{},
			ExpectedItems: mapstr.M{
				"json": mapstr.M{"type": "test", "text": "hello"},
			},
			ExpectedTimestamp: time.Time{},
		},

		"write result to message_key field": {
			// when MessageKey is defined, the Text overwrites the value of that key
			Data:       mapstr.M{"type": "test_type", "json": mapstr.M{"type": "test", "text": "hi"}},
			Text:       &text,
			JSONConfig: Config{MessageKey: "text"},
			ExpectedItems: mapstr.M{
				"json": mapstr.M{"type": "test", "text": "hello"},
				"type": "test_type",
			},
			ExpectedTimestamp: time.Time{},
		},
		"parse @timestamp": {
			// when @timestamp is in JSON and overwrite_keys is true, parse it
			// in a common.Time
			Data:       mapstr.M{"@timestamp": now, "type": "test_type", "json": mapstr.M{"type": "test", "@timestamp": "2016-04-05T18:47:18.444Z"}},
			Text:       &text,
			JSONConfig: Config{KeysUnderRoot: true, OverwriteKeys: true},
			ExpectedItems: mapstr.M{
				"type": "test",
			},
			ExpectedTimestamp: time.Time(common.MustParseTime("2016-04-05T18:47:18.444Z")),
		},
		"fail to parse @timestamp": {
			// when the parsing on @timestamp fails, leave the existing value and add an error key
			// in a common.Time
			Data:       mapstr.M{"@timestamp": common.Time(now), "type": "test_type", "json": mapstr.M{"type": "test", "@timestamp": "2016-04-05T18:47:18.44XX4Z"}},
			Text:       &text,
			JSONConfig: Config{KeysUnderRoot: true, OverwriteKeys: true, AddErrorKey: true},
			ExpectedItems: mapstr.M{
				"type":  "test",
				"error": mapstr.M{"type": "json", "message": "@timestamp not overwritten (parse error on 2016-04-05T18:47:18.44XX4Z)"},
			},
			ExpectedTimestamp: time.Time{},
		},

		"wrong @timestamp format": {
			// when the @timestamp has the wrong type, leave the existing value and add an error key
			// in a common.Time
			Data:       mapstr.M{"@timestamp": common.Time(now), "type": "test_type", "json": mapstr.M{"type": "test", "@timestamp": 42}},
			Text:       &text,
			JSONConfig: Config{KeysUnderRoot: true, OverwriteKeys: true, AddErrorKey: true},
			ExpectedItems: mapstr.M{
				"type":  "test",
				"error": mapstr.M{"type": "json", "message": "@timestamp not overwritten (not string)"},
			},
			ExpectedTimestamp: time.Time{},
		},
		"ignore non-string type field": {
			// if overwrite_keys is true, but the `type` key in json is not a string, ignore it
			Data:       mapstr.M{"type": "test_type", "json": mapstr.M{"type": 42}},
			Text:       &text,
			JSONConfig: Config{KeysUnderRoot: true, OverwriteKeys: true, AddErrorKey: true},
			ExpectedItems: mapstr.M{
				"type":  "test_type",
				"error": mapstr.M{"type": "json", "message": "type not overwritten (not string)"},
			},
			ExpectedTimestamp: time.Time{},
		},

		"ignore empty type field": {
			// if overwrite_keys is true, but the `type` key in json is empty, ignore it
			Data:       mapstr.M{"type": "test_type", "json": mapstr.M{"type": ""}},
			Text:       &text,
			JSONConfig: Config{KeysUnderRoot: true, OverwriteKeys: true, AddErrorKey: true},
			ExpectedItems: mapstr.M{
				"type":  "test_type",
				"error": mapstr.M{"type": "json", "message": "type not overwritten (invalid value [])"},
			},
			ExpectedTimestamp: time.Time{},
		},
		"ignore type names starting with underscore": {
			// if overwrite_keys is true, but the `type` key in json starts with _, ignore it
			Data:       mapstr.M{"@timestamp": common.Time(now), "type": "test_type", "json": mapstr.M{"type": "_type"}},
			Text:       &text,
			JSONConfig: Config{KeysUnderRoot: true, OverwriteKeys: true, AddErrorKey: true},
			ExpectedItems: mapstr.M{
				"type":  "test_type",
				"error": mapstr.M{"type": "json", "message": "type not overwritten (invalid value [_type])"},
			},
			ExpectedTimestamp: time.Time{},
		},
		"do not set error if AddErrorKey is false": {
			Data:       mapstr.M{"@timestamp": common.Time(now), "type": "test_type", "json": mapstr.M{"type": "_type"}},
			Text:       &text,
			JSONConfig: Config{KeysUnderRoot: true, OverwriteKeys: true, AddErrorKey: false},
			ExpectedItems: mapstr.M{
				"type":  "test_type",
				"error": nil,
			},
			ExpectedTimestamp: time.Time{},
		},
		"extract event id": {
			// if document_id is set, extract the ID from the event
			Data:       mapstr.M{"@timestamp": common.Time(now), "json": mapstr.M{"id": "test_id"}},
			JSONConfig: Config{DocumentID: "id"},
			ExpectedID: "test_id",
		},
		"extract event id with wrong type": {
			// if document_id is set, extract the ID from the event
			Data:       mapstr.M{"@timestamp": common.Time(now), "json": mapstr.M{"id": 42}},
			JSONConfig: Config{DocumentID: "id"},
			ExpectedID: "",
		},
		"expand dotted fields": {
			Data:          mapstr.M{"json": mapstr.M{"a.b": mapstr.M{"c": "c"}, "a.b.d": "d"}},
			JSONConfig:    Config{ExpandKeys: true, KeysUnderRoot: true},
			ExpectedItems: mapstr.M{"a": mapstr.M{"b": mapstr.M{"c": "c", "d": "d"}}},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var jsonFields mapstr.M
			if fields, ok := test.Data["json"]; ok {
				jsonFields = fields.(mapstr.M)
			}

			id, ts := MergeJSONFields(test.Data, jsonFields, test.Text, test.JSONConfig)

			t.Log("Executing test:", test)
			for k, v := range test.ExpectedItems {
				assert.Equal(t, v, test.Data[k])
			}
			assert.Equal(t, test.ExpectedTimestamp, ts)
			assert.Equal(t, test.ExpectedID, id)
		})
	}
}
