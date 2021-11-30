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

package actions

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/processors"
)

var fields = [1]string{"msg"}
var testConfig, _ = common.NewConfigFrom(map[string]interface{}{
	"fields":       fields,
	"processArray": false,
})

func TestDecodeJSONFieldsCheckConfig(t *testing.T) {
	// All fields defined in config should be allowed.
	cfg := common.MustNewConfigFrom(map[string]interface{}{
		"decode_json_fields": &config{
			// Rely on zero values for all fields that don't have validation.
			MaxDepth: 1,
		},
	})
	_, err := processors.New(processors.PluginConfig([]*common.Config{cfg}))
	assert.NoError(t, err)

	// Unknown fields should not be allowed.
	cfg = common.MustNewConfigFrom(map[string]interface{}{
		"decode_json_fields": map[string]interface{}{
			"fields":     []string{"required"},
			"extraneous": "field",
		},
	})
	_, err = processors.New(processors.PluginConfig([]*common.Config{cfg}))
	assert.Error(t, err)
	assert.EqualError(t, err, "unexpected extraneous option in decode_json_fields")
}

func TestMissingKey(t *testing.T) {
	input := common.MapStr{
		"pipeline": "us1",
	}

	actual := getActualValue(t, testConfig, input)

	expected := common.MapStr{
		"pipeline": "us1",
	}

	assert.Equal(t, expected.String(), actual.String())
}

func TestFieldNotString(t *testing.T) {
	input := common.MapStr{
		"msg":      123,
		"pipeline": "us1",
	}

	actual := getActualValue(t, testConfig, input)

	expected := common.MapStr{
		"msg":      123,
		"pipeline": "us1",
	}

	assert.Equal(t, expected.String(), actual.String())
}

func TestInvalidJSON(t *testing.T) {
	input := common.MapStr{
		"msg":      "{\"log\":\"{\\\"level\\\":\\\"info\\\"}\",\"stream\":\"stderr\",\"count\":3",
		"pipeline": "us1",
	}

	actual := getActualValue(t, testConfig, input)

	expected := common.MapStr{
		"msg":      "{\"log\":\"{\\\"level\\\":\\\"info\\\"}\",\"stream\":\"stderr\",\"count\":3",
		"pipeline": "us1",
	}
	assert.Equal(t, expected.String(), actual.String())
}

func TestInvalidJSONMultiple(t *testing.T) {
	input := common.MapStr{
		"msg":      "11:38:04,323 |-INFO testing",
		"pipeline": "us1",
	}

	actual := getActualValue(t, testConfig, input)

	expected := common.MapStr{
		"msg":      "11:38:04,323 |-INFO testing",
		"pipeline": "us1",
	}
	assert.Equal(t, expected.String(), actual.String())
}

func TestDocumentID(t *testing.T) {
	log := logp.NewLogger("decode_json_fields_test")

	input := common.MapStr{
		"msg": `{"log": "message", "myid": "myDocumentID"}`,
	}

	config := common.MustNewConfigFrom(map[string]interface{}{
		"fields":      []string{"msg"},
		"document_id": "myid",
	})

	p, err := NewDecodeJSONFields(config)
	if err != nil {
		log.Error("Error initializing decode_json_fields")
		t.Fatal(err)
	}

	actual, err := p.Run(&beat.Event{Fields: input})
	require.NoError(t, err)

	wantFields := common.MapStr{
		"msg": map[string]interface{}{"log": "message"},
	}
	wantMeta := common.MapStr{
		"_id": "myDocumentID",
	}

	assert.Equal(t, wantFields, actual.Fields)
	assert.Equal(t, wantMeta, actual.Meta)
}

func TestValidJSONDepthOne(t *testing.T) {
	input := common.MapStr{
		"msg":      "{\"log\":\"{\\\"level\\\":\\\"info\\\"}\",\"stream\":\"stderr\",\"count\":3}",
		"pipeline": "us1",
	}

	actual := getActualValue(t, testConfig, input)

	expected := common.MapStr{
		"msg": map[string]interface{}{
			"log":    "{\"level\":\"info\"}",
			"stream": "stderr",
			"count":  3,
		},
		"pipeline": "us1",
	}

	assert.Equal(t, expected.String(), actual.String())
}

func TestValidJSONDepthTwo(t *testing.T) {
	input := common.MapStr{
		"msg":      "{\"log\":\"{\\\"level\\\":\\\"info\\\"}\",\"stream\":\"stderr\",\"count\":3}",
		"pipeline": "us1",
	}

	testConfig, _ = common.NewConfigFrom(map[string]interface{}{
		"fields":        fields,
		"process_array": false,
		"max_depth":     2,
	})

	actual := getActualValue(t, testConfig, input)

	expected := common.MapStr{
		"msg": map[string]interface{}{
			"log": map[string]interface{}{
				"level": "info",
			},
			"stream": "stderr",
			"count":  3,
		},
		"pipeline": "us1",
	}

	assert.Equal(t, expected.String(), actual.String())
}

func TestTargetOption(t *testing.T) {
	input := common.MapStr{
		"msg":      "{\"log\":\"{\\\"level\\\":\\\"info\\\"}\",\"stream\":\"stderr\",\"count\":3}",
		"pipeline": "us1",
	}

	testConfig, _ = common.NewConfigFrom(map[string]interface{}{
		"fields":        fields,
		"process_array": false,
		"max_depth":     2,
		"target":        "doc",
	})

	actual := getActualValue(t, testConfig, input)

	expected := common.MapStr{
		"doc": map[string]interface{}{
			"log": map[string]interface{}{
				"level": "info",
			},
			"stream": "stderr",
			"count":  3,
		},
		"msg":      "{\"log\":\"{\\\"level\\\":\\\"info\\\"}\",\"stream\":\"stderr\",\"count\":3}",
		"pipeline": "us1",
	}

	assert.Equal(t, expected.String(), actual.String())
}

func TestTargetRootOption(t *testing.T) {
	input := common.MapStr{
		"msg":      "{\"log\":\"{\\\"level\\\":\\\"info\\\"}\",\"stream\":\"stderr\",\"count\":3}",
		"pipeline": "us1",
	}

	testConfig, _ = common.NewConfigFrom(map[string]interface{}{
		"fields":        fields,
		"process_array": false,
		"max_depth":     2,
		"target":        "",
	})

	actual := getActualValue(t, testConfig, input)

	expected := common.MapStr{
		"log": map[string]interface{}{
			"level": "info",
		},
		"stream":   "stderr",
		"count":    3,
		"msg":      "{\"log\":\"{\\\"level\\\":\\\"info\\\"}\",\"stream\":\"stderr\",\"count\":3}",
		"pipeline": "us1",
	}

	assert.Equal(t, expected.String(), actual.String())
}

func TestNotJsonObjectOrArray(t *testing.T) {
	var cases = []struct {
		MaxDepth int
		Expected common.MapStr
	}{
		{
			MaxDepth: 1,
			Expected: common.MapStr{
				"msg": common.MapStr{
					"someDate":           "2016-09-28T01:40:26.760+0000",
					"someNumber":         1475026826760,
					"someNumberAsString": "1475026826760",
					"someString":         "foobar",
					"someString2":        "2017 is awesome",
					"someMap":            "{\"a\":\"b\"}",
					"someArray":          "[1,2,3]",
				},
			},
		},
		{
			MaxDepth: 10,
			Expected: common.MapStr{
				"msg": common.MapStr{
					"someDate":           "2016-09-28T01:40:26.760+0000",
					"someNumber":         1475026826760,
					"someNumberAsString": "1475026826760",
					"someString":         "foobar",
					"someString2":        "2017 is awesome",
					"someMap":            common.MapStr{"a": "b"},
					"someArray":          []int{1, 2, 3},
				},
			},
		},
	}

	for _, testCase := range cases {
		t.Run(fmt.Sprintf("TestNotJsonObjectOrArrayDepth-%v", testCase.MaxDepth), func(t *testing.T) {
			input := common.MapStr{
				"msg": `{
					"someDate": "2016-09-28T01:40:26.760+0000",
					"someNumberAsString": "1475026826760",
					"someNumber": 1475026826760,
					"someString": "foobar",
					"someString2": "2017 is awesome",
					"someMap": "{\"a\":\"b\"}",
					"someArray": "[1,2,3]"
				  }`,
			}

			testConfig, _ = common.NewConfigFrom(map[string]interface{}{
				"fields":        fields,
				"process_array": true,
				"max_depth":     testCase.MaxDepth,
			})

			actual := getActualValue(t, testConfig, input)
			assert.Equal(t, testCase.Expected.String(), actual.String())
		})
	}
}

func TestArrayWithArraysDisabled(t *testing.T) {
	input := common.MapStr{
		"msg": `{
			"arrayOfMap": "[{\"a\":\"b\"}]"
		  }`,
	}

	testConfig, _ = common.NewConfigFrom(map[string]interface{}{
		"fields":        fields,
		"max_depth":     10,
		"process_array": false,
	})

	actual := getActualValue(t, testConfig, input)

	expected := common.MapStr{
		"msg": common.MapStr{
			"arrayOfMap": "[{\"a\":\"b\"}]",
		},
	}

	assert.Equal(t, expected.String(), actual.String())
}

func TestArrayWithArraysEnabled(t *testing.T) {
	input := common.MapStr{
		"msg": `{
			"arrayOfMap": "[{\"a\":\"b\"}]"
		  }`,
	}

	testConfig, _ = common.NewConfigFrom(map[string]interface{}{
		"fields":        fields,
		"max_depth":     10,
		"process_array": true,
	})

	actual := getActualValue(t, testConfig, input)

	expected := common.MapStr{
		"msg": common.MapStr{
			"arrayOfMap": []common.MapStr{common.MapStr{"a": "b"}},
		},
	}

	assert.Equal(t, expected.String(), actual.String())
}

func TestArrayWithInvalidArray(t *testing.T) {
	input := common.MapStr{
		"msg": `{
			"arrayOfMap": "[]]"
		  }`,
	}

	testConfig, _ = common.NewConfigFrom(map[string]interface{}{
		"fields":        fields,
		"max_depth":     10,
		"process_array": true,
	})

	actual := getActualValue(t, testConfig, input)

	expected := common.MapStr{
		"msg": common.MapStr{
			"arrayOfMap": "[]]",
		},
	}

	assert.Equal(t, expected.String(), actual.String())
}

func TestAddErrKeyOption(t *testing.T) {
	tests := []struct {
		name           string
		addErrOption   bool
		expectedOutput common.MapStr
	}{
		{name: "With add_error_key option", addErrOption: true, expectedOutput: common.MapStr{
			"error": common.MapStr{"message": "@timestamp not overwritten (parse error on {})", "type": "json"},
			"msg":   "{\"@timestamp\":\"{}\"}",
		}},
		{name: "Without add_error_key option", addErrOption: false, expectedOutput: common.MapStr{
			"msg": "{\"@timestamp\":\"{}\"}",
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := common.MapStr{
				"msg": "{\"@timestamp\":\"{}\"}",
			}

			testConfig, _ = common.NewConfigFrom(map[string]interface{}{
				"fields":         fields,
				"add_error_key":  test.addErrOption,
				"overwrite_keys": true,
				"target":         "",
			})
			actual := getActualValue(t, testConfig, input)

			assert.Equal(t, test.expectedOutput.String(), actual.String())

		})
	}
}

func TestExpandKeys(t *testing.T) {
	testConfig := common.MustNewConfigFrom(map[string]interface{}{
		"fields":      fields,
		"expand_keys": true,
		"target":      "",
	})
	input := common.MapStr{"msg": `{"a.b": {"c": "c"}, "a.b.d": "d"}`}
	expected := common.MapStr{
		"msg": `{"a.b": {"c": "c"}, "a.b.d": "d"}`,
		"a": common.MapStr{
			"b": map[string]interface{}{
				"c": "c",
				"d": "d",
			},
		},
	}
	actual := getActualValue(t, testConfig, input)
	assert.Equal(t, expected, actual)
}

func TestExpandKeysError(t *testing.T) {
	testConfig := common.MustNewConfigFrom(map[string]interface{}{
		"fields":        fields,
		"expand_keys":   true,
		"add_error_key": true,
		"target":        "",
	})
	input := common.MapStr{"msg": `{"a.b": "c", "a.b.c": "d"}`}
	expected := common.MapStr{
		"msg": `{"a.b": "c", "a.b.c": "d"}`,
		"error": common.MapStr{
			"message": "cannot expand ...",
			"type":    "json",
		},
	}

	actual := getActualValue(t, testConfig, input)
	assert.Contains(t, actual, "error")
	errorField := actual["error"].(common.MapStr)
	assert.Contains(t, errorField, "message")

	// The order in which keys are processed is not defined, so the error
	// message is not defined. Apart from that, the outcome is the same.
	assert.Regexp(t, `cannot expand ".*": .*`, errorField["message"])
	errorField["message"] = "cannot expand ..."
	assert.Equal(t, expected, actual)
}

func TestOverwriteMetadata(t *testing.T) {
	testConfig := common.MustNewConfigFrom(map[string]interface{}{
		"fields":         fields,
		"target":         "",
		"overwrite_keys": true,
	})

	input := common.MapStr{
		"msg": "{\"@metadata\":{\"beat\":\"libbeat\"},\"msg\":\"overwrite metadata test\"}",
	}

	expected := common.MapStr{
		"msg": "overwrite metadata test",
	}
	actual := getActualValue(t, testConfig, input)

	assert.Equal(t, expected, actual)
}

func TestAddErrorToEventOnUnmarshalError(t *testing.T) {
	testConfig := common.MustNewConfigFrom(map[string]interface{}{
		"fields":        "message",
		"add_error_key": true,
	})

	input := common.MapStr{
		"message": "Broken JSON [[",
	}

	actual := getActualValue(t, testConfig, input)

	errObj, ok := actual["error"].(common.MapStr)
	require.True(t, ok, "'error' field not present or of invalid type")
	require.NotNil(t, actual["error"])

	assert.Equal(t, "message", errObj["field"])
	assert.NotNil(t, errObj["data"])
	assert.NotNil(t, errObj["message"])
}

func getActualValue(t *testing.T, config *common.Config, input common.MapStr) common.MapStr {
	log := logp.NewLogger("decode_json_fields_test")

	p, err := NewDecodeJSONFields(config)
	if err != nil {
		log.Error("Error initializing decode_json_fields")
		t.Fatal(err)
	}

	actual, _ := p.Run(&beat.Event{Fields: input})
	return actual.Fields
}
