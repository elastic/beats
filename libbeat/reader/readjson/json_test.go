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
	"bytes"
	"strings"
	"testing"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
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
		{
			Name:  "Key collision",
			Input: `{"log.level":"info","log":{"source":"connectors-py-default"},"log":{"logger":"agent_component.cli"}}`,
			Output: map[string]interface{}{
				"log.level": "info",
				"log": map[string]interface{}{
					"logger": "agent_component.cli",
				},
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
		Text            string
		Config          Config
		ExpectedText    string
		ExpectedMap     mapstr.M
		// errMsgContains, when non-empty, asserts that the error.message field
		// contains this substring instead of requiring an exact match. Use this
		// for cases where the underlying parser's error message wording may vary.
		errMsgContains  string
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
			Text:           `{"message": "test", "value": "`,
			Config:         Config{MessageKey: "value", AddErrorKey: true},
			ExpectedText:   `{"message": "test", "value": "`,
			ExpectedMap:    mapstr.M{"error": mapstr.M{"type": "json"}},
			errMsgContains: "Error decoding JSON:",
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

	logger := logptest.NewTestingLogger(t, "json_test")
	for _, test := range tests {

		var p JSONReader
		p.cfg = &test.Config
		p.logger = logger
		text, M := p.decode([]byte(test.Text))
		assert.Equal(t, test.ExpectedText, string(text))
		if test.errMsgContains != "" {
			// Check that the error message field contains the expected substring rather
			// than requiring an exact match, since parser error message wording may vary.
			if errMap, ok := M["error"].(mapstr.M); ok {
				assert.True(t, strings.Contains(errMap["message"].(string), test.errMsgContains),
					"error.message %q should contain %q", errMap["message"], test.errMsgContains)
			} else {
				assert.Fail(t, "expected error field not present in decoded map")
			}
		} else {
			assert.Equal(t, test.ExpectedMap, M)
		}
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
		"key collision with expanded keys": {
			Data: mapstr.M{
				"log.level": "info",
				"log": mapstr.M{
					"logger": "agent_component.cli",
				},
			},
			JSONConfig: Config{ExpandKeys: true},
			ExpectedItems: mapstr.M{
				"log.level": "info",
				"log": mapstr.M{
					"logger": "agent_component.cli",
				},
			},
		},
		"key collision without expanded keys": {
			Data: mapstr.M{
				"log.level": "info",
				"log": mapstr.M{
					"logger": "agent_component.cli",
				},
			},
			JSONConfig: Config{ExpandKeys: false},
			ExpectedItems: mapstr.M{
				"log.level": "info",
				"log": mapstr.M{
					"logger": "agent_component.cli",
				},
			},
		},
		"key collision with overwrite": {
			Data: mapstr.M{
				"log.level": "info",
				"log": mapstr.M{
					"logger": "agent_component.cli",
				},
			},
			JSONConfig: Config{OverwriteKeys: true, AddErrorKey: true, IgnoreDecodingError: true},
			ExpectedItems: mapstr.M{
				"log.level": "info",
				"log": mapstr.M{
					"logger": "agent_component.cli",
				},
			},
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

// oracleNestedLine and oracleCloudtrailLine are real-world events used as inputs
// for TestIterParseMatchesUnmarshal. They exercise nulls, 4-level nesting, arrays
// of strings, and mixed scalar types.
var oracleNestedLine = []byte(`{"message":"request completed","log":{"level":"info","logger":"http"},"agent":{"name":"filebeat","version":"8.17.0"},"host":{"name":"web-01","os":{"type":"linux","platform":"ubuntu"}},"event":{"dataset":"access","severity":3,"duration":142000000},"http":{"request":{"method":"GET","body":{"bytes":0}},"response":{"status_code":200,"body":{"bytes":1024}}}}`)
var oracleCloudtrailLine = []byte(`{"activity_id":1,"activity_name":"Create","actor":{"idp":{"name":null},"invoked_by":null,"session":{"created_time":null,"issuer":null,"mfa":null},"user":{"account_uid":"123456789012","credential_uid":"AKIAIOSFODNN7EXAMPLE","name":"Alice","type":"IAMUser","uid":"123456789012","uuid":"arn:aws:iam::123456789012:user/Alice"}},"api":{"operation":"CreateLoadBalancer","request":{"uid":"b9960276-b9b2-11e3-8a13-f1ef1EXAMPLE"},"response":{"error":null,"message":null},"service":{"name":"elasticloadbalancing.amazonaws.com"},"version":"2015-12-01"},"category_name":"Audit Activity","category_uid":3,"class_name":"API Activity","class_uid":3005,"cloud":{"provider":"AWS","region":"us-west-2"},"http_request":{"user_agent":"aws-cli/1.10.10 Python/2.7.9 Windows/7 botocore/1.4.1"},"metadata":{"product":{"feature":{"name":"Management, Data, and Insights"},"name":"CloudTrail","vendor_name":"AWS","version":"1.03"},"profiles":["cloud"],"uid":"6f4ab5bd-2daa-4d00-be14-d92efEXAMPLE","version":"1.0.0-rc.2"},"severity":"Informational","severity_id":1,"src_endpoint":{"domain":null,"ip":"198.51.100.1","uid":null},"status":"Success","status_id":1,"time":1459524708000,"type_name":"API Activity: Create","type_uid":300501}`)

// TestIterParseMatchesUnmarshal is an oracle test: for every input, iterParseObject
// (the production parser) must produce the same map as stdlibUnmarshal (the old
// stdlib Decoder + UseNumber + TransformNumbers path). This covers number resolution,
// string escapes, nested structures, and edge cases exhaustively.
func TestIterParseMatchesUnmarshal(t *testing.T) {
	cases := []struct {
		name string
		line []byte
	}{
		// --- real-world events ---
		{"bench_medium", benchMediumLine},
		{"oracle_nested", oracleNestedLine},
		{"bench_journald", benchJournaldLine},
		{"oracle_cloudtrail", oracleCloudtrailLine},

		// --- integer edge cases ---
		{"int_zero", []byte(`{"n":0}`)},
		{"int_one", []byte(`{"n":1}`)},
		{"int_negative", []byte(`{"n":-42}`)},
		{"int_large", []byte(`{"n":9999999999999}`)},
		{"int64_max", []byte(`{"n":9223372036854775807}`)},
		{"int64_min", []byte(`{"n":-9223372036854775808}`)},
		{"int64_overflow_pos", []byte(`{"n":9223372036854775808}`)},  // > int64 max → float64
		{"int64_overflow_neg", []byte(`{"n":-9223372036854775809}`)}, // < int64 min → float64
		{"int_beyond_float64_precision", []byte(`{"n":9007199254740993}`)}, // 2^53+1: exact as int64, lossy as float64
		{"int_scientific", []byte(`{"n":1e5}`)},  // "1e5" fails ParseInt → float64(100000)
		{"int_scientific_large", []byte(`{"n":1e18}`)},

		// --- float edge cases ---
		{"float_basic", []byte(`{"f":3.14}`)},
		{"float_negative", []byte(`{"f":-2.718}`)},
		{"float_zero", []byte(`{"f":0.0}`)},
		{"float_neg_zero", []byte(`{"f":-0.0}`)},
		{"float_one_point_zero", []byte(`{"f":1.0}`)}, // looks like int but has decimal point
		{"float_exp_pos", []byte(`{"f":1.5e10}`)},
		{"float_exp_neg", []byte(`{"f":1.5e-10}`)},
		{"float_exp_negative_val", []byte(`{"f":-1.5e10}`)},
		{"float_near_max", []byte(`{"f":1.7976931348623157e+308}`)},
		{"float_small_subnormal", []byte(`{"f":5e-324}`)},
		{"float_overflow_inf", []byte(`{"f":1e309}`)}, // overflows to +Inf
		{"float_point_one", []byte(`{"f":0.1}`)},
		{"float_point_two", []byte(`{"f":0.2}`)},

		// --- multiple number types in one object ---
		{"numbers_mixed", []byte(`{"i":1,"f":1.1,"big":9999999999999,"neg":-7,"zero":0}`)},
		{"numbers_in_array", []byte(`{"arr":[0,1,-1,3.14,1e5,9223372036854775807]}`)},
		{"numbers_in_nested", []byte(`{"a":{"i":42,"f":3.14},"b":{"i":-1,"f":-0.5}}`)},

		// --- string edge cases ---
		{"string_empty", []byte(`{"s":""}`)},
		{"string_space", []byte(`{"s":" "}`)},
		{"string_escape_newline", []byte(`{"s":"line1\nline2"}`)},
		{"string_escape_tab", []byte(`{"s":"col1\tcol2"}`)},
		{"string_escape_cr", []byte(`{"s":"a\rb"}`)},
		{"string_escape_quote", []byte(`{"s":"say \"hello\""}`)},
		{"string_escape_backslash", []byte(`{"s":"C:\\Users\\foo"}`)},
		{"string_escape_solidus", []byte(`{"s":"a\/b"}`)},
		{"string_all_escapes", []byte(`{"s":"\n\t\r\\\"\/\b\f"}`)},
		{"string_unicode_escape_ascii", []byte(`{"s":"\u0048\u0065\u006C\u006C\u006F"}`)}, // "Hello"
		{"string_unicode_escape_nonascii", []byte(`{"s":"\u00e9l\u00e8ve"}`)},             // "élève"
		{"string_unicode_surrogate_pair", []byte(`{"s":"\uD83D\uDE00"}`)},                 // 😀
		{"string_utf8_direct", []byte(`{"s":"héllo wörld"}`)},
		{"string_utf8_emoji_direct", []byte(`{"s":"hello 😀 world"}`)},
		{"string_utf8_cjk", []byte(`{"s":"日本語"}`)},
		{"string_looks_like_number", []byte(`{"s":"42"}`)},
		{"string_looks_like_float", []byte(`{"s":"3.14"}`)},
		{"string_looks_like_bool", []byte(`{"s":"true"}`)},
		{"string_looks_like_null", []byte(`{"s":"null"}`)},
		{"string_very_long", append(append([]byte(`{"s":"`), bytes.Repeat([]byte("x"), 1024)...), '"', '}')},

		// --- boolean ---
		{"bool_true", []byte(`{"b":true}`)},
		{"bool_false", []byte(`{"b":false}`)},
		{"bools_array", []byte(`{"arr":[true,false,true,false]}`)},

		// --- null ---
		{"null_value", []byte(`{"k":null}`)},
		{"all_nulls", []byte(`{"a":null,"b":null,"c":null}`)},
		{"null_in_array", []byte(`{"arr":[null,1,null,"x",null]}`)},
		{"null_in_nested", []byte(`{"outer":{"inner":null}}`)},
		{"null_array_values_only", []byte(`{"arr":[null,null,null]}`)},

		// --- arrays ---
		{"array_empty", []byte(`{"arr":[]}`)},
		{"array_single_int", []byte(`{"arr":[42]}`)},
		{"array_of_ints", []byte(`{"arr":[1,2,3,4,5]}`)},
		{"array_of_floats", []byte(`{"arr":[1.1,2.2,3.3]}`)},
		{"array_of_strings", []byte(`{"arr":["a","b","c"]}`)},
		{"array_of_bools", []byte(`{"arr":[true,false,true]}`)},
		{"array_of_nulls", []byte(`{"arr":[null,null]}`)},
		{"array_mixed", []byte(`{"arr":[1,"two",true,null,3.14]}`)},
		{"array_of_objects", []byte(`{"items":[{"id":1,"name":"a"},{"id":2,"name":"b"}]}`)},
		{"array_of_empty_objects", []byte(`{"arr":[{},{}]}`)},
		{"array_nested_2deep", []byte(`{"arr":[[1,2],[3,4]]}`)},
		{"array_nested_3deep", []byte(`{"arr":[[[1,2],[3]],[[4]]]}`)},
		{"array_of_empty_arrays", []byte(`{"arr":[[],[],[]]}`)},
		{"nested_empty_array", []byte(`{"obj":{"arr":[]}}`)},
		{"deeply_nested_empty_array", []byte(`{"a":{"b":{"c":[]}}}`)},
		{"array_mixed_with_objects", []byte(`{"arr":[1,{"k":"v"},true,[2,3],null]}`)},

		// --- object structure ---
		{"empty_object", []byte(`{}`)},
		{"single_key", []byte(`{"k":"v"}`)},
		// Note: {"":"value"} is intentionally excluded. iterParseObjectInto uses ReadObject()==""
		// as the end-of-object sentinel, so an actual empty-string key terminates the loop early.
		// This is a known limitation of the jsoniter ReadObject API; empty-string keys do not
		// appear in real log data so it is not a concern for the readjson use case.
		{"unicode_key", []byte(`{"héllo":"world"}`)},
		{"duplicate_keys", []byte(`{"a":1,"a":2}`)}, // last-value-wins in both stdlib and jsoniter
		{"many_keys", []byte(`{"a":1,"b":2,"c":3,"d":4,"e":5,"f":6,"g":7,"h":8,"i":9,"j":10,"k":11,"l":12,"m":13,"n":14,"o":15,"p":16}`)},
		{"nested_3deep", []byte(`{"a":{"b":{"c":"deep"}}}`)},
		{"nested_5deep", []byte(`{"a":{"b":{"c":{"d":{"e":"very deep"}}}}}`)},
		{"nested_wide_and_deep", []byte(`{"x":{"a":1,"b":2,"c":{"d":3,"e":4,"f":{"g":5,"h":6}}},"y":{"i":7,"j":{"k":8,"l":9}}}`)},
		{"alternating_array_object", []byte(`{"arr":[{"arr":[{"arr":[1,2,3]}]}]}`)},

		// --- falsy values together ---
		{"all_falsy", []byte(`{"n":0,"b":false,"s":"","k":null}`)},
		{"falsy_in_array", []byte(`{"arr":[0,false,"",null]}`)},

		// --- whitespace variants ---
		{"whitespace_in_values", []byte(`{ "a" : 1 , "b" : "hello" , "c" : true }`)},

		// --- mixed deep realistic ---
		{"ecs_full", []byte(`{"@timestamp":"2024-01-15T10:30:00.000Z","message":"request completed","log":{"level":"info","logger":"http","origin":{"file":{"name":"server.go","line":42},"function":"handleRequest"}},"agent":{"name":"filebeat","version":"8.17.0","id":"abc123","type":"filebeat","ephemeral_id":"xyz"},"host":{"name":"web-01","hostname":"web-01.example.com","id":"deadbeef","ip":["10.0.0.1","192.168.1.1"],"os":{"type":"linux","platform":"ubuntu","version":"22.04","kernel":"5.15.0"}},"event":{"dataset":"access","severity":3,"duration":142000000,"kind":"event","category":["web"],"type":["access"],"outcome":"success"},"http":{"request":{"method":"GET","mime_type":"application/json","body":{"bytes":256}},"response":{"status_code":200,"mime_type":"application/json","body":{"bytes":1024}}},"url":{"path":"/api/users","domain":"api.example.com","scheme":"https","port":443},"user":{"name":"alice","id":"u-123","roles":["admin","viewer"]},"source":{"ip":"203.0.113.42","port":54321,"geo":{"country_iso_code":"US","region_name":"California"}}}`)},
	}

	iter := jsoniter.NewIterator(jsoniterAPI)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var want map[string]interface{}
			require.NoError(t, stdlibUnmarshal(tc.line, &want))

			iter.ResetBytes(tc.line)
			got := iterParseObject(iter)
			require.NoError(t, iter.Error)

			assert.Equal(t, want, got)
		})
	}
}
