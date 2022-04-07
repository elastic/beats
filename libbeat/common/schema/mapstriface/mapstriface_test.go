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

package mapstriface

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v8/libbeat/common"
	s "github.com/elastic/beats/v8/libbeat/common/schema"
)

func TestConversions(t *testing.T) {
	ts := time.Now()

	cTs := common.Time{}

	input := map[string]interface{}{
		"testString":          "hello",
		"testInt":             42,
		"testIntFromFloat":    42.2,
		"testFloat":           42.7,
		"testFloatFromInt":    43,
		"testIntFromInt32":    int32(32),
		"testIntFromInt64":    int64(42),
		"testJsonNumber":      json.Number("3910564293633576924"),
		"testJsonNumberFloat": json.Number("43.7"),
		"testBool":            true,
		"testObj": map[string]interface{}{
			"testObjString": "hello, object",
		},
		"rawObject": map[string]interface{}{
			"nest1": map[string]interface{}{
				"nest2": "world",
			},
		},
		"testArray":        []string{"a", "b", "c"},
		"testNonNestedObj": "hello from top level",
		"testTime":         ts,
		"commonTime":       cTs,

		// wrong types
		"testErrorInt":    "42",
		"testErrorTime":   12,
		"testErrorBool":   "false",
		"testErrorString": 32,
	}

	schema := s.Schema{
		"test_string":               Str("testString"),
		"test_int":                  Int("testInt"),
		"test_int_from_float":       Int("testIntFromFloat"),
		"test_int_from_int64":       Int("testIntFromInt64"),
		"test_float":                Float("testFloat"),
		"test_float_from_int":       Float("testFloatFromInt"),
		"test_int_from_json":        Int("testJsonNumber"),
		"test_float_from_json":      Float("testJsonNumberFloat"),
		"test_string_from_num":      StrFromNum("testIntFromInt32"),
		"test_string_from_json_num": StrFromNum("testJsonNumber"),
		"test_bool":                 Bool("testBool"),
		"test_time":                 Time("testTime"),
		"common_time":               Time("commonTime"),
		"test_obj_1": s.Object{
			"test": Str("testNonNestedObj"),
		},
		"test_obj_2": Dict("testObj", s.Schema{
			"test": Str("testObjString"),
		}),
		"test_nested":       Ifc("rawObject"),
		"test_array":        Ifc("testArray"),
		"test_error_int":    Int("testErrorInt", s.Optional),
		"test_error_time":   Time("testErrorTime", s.Optional),
		"test_error_bool":   Bool("testErrorBool", s.Optional),
		"test_error_string": Str("testErrorString", s.Optional),
	}

	expected := common.MapStr{
		"test_string":               "hello",
		"test_int":                  int64(42),
		"test_int_from_float":       int64(42),
		"test_int_from_int64":       int64(42),
		"test_float":                float64(42.7),
		"test_float_from_int":       float64(43),
		"test_int_from_json":        int64(3910564293633576924),
		"test_float_from_json":      float64(43.7),
		"test_string_from_num":      "32",
		"test_string_from_json_num": "3910564293633576924",
		"test_bool":                 true,
		"test_time":                 common.Time(ts),
		"common_time":               cTs,
		"test_obj_1": common.MapStr{
			"test": "hello from top level",
		},
		"test_obj_2": common.MapStr{
			"test": "hello, object",
		},
		"test_nested": map[string]interface{}{
			"nest1": map[string]interface{}{
				"nest2": "world",
			},
		},
		"test_array": []string{"a", "b", "c"},
	}

	event, _ := schema.Apply(input)
	assert.Equal(t, event, expected)
}

func TestOptionalField(t *testing.T) {
	cases := []struct {
		Description string
		Input       map[string]interface{}
		Schema      s.Schema
		Expected    common.MapStr
		ExpectError bool
	}{
		{
			"missing optional field",
			map[string]interface{}{
				"testString": "hello",
				"testInt":    42,
			},
			s.Schema{
				"test_string": Str("testString"),
				"test_int":    Int("testInt"),
				"test_opt":    Bool("testOptionalInt", s.Optional),
			},
			common.MapStr{
				"test_string": "hello",
				"test_int":    int64(42),
			},
			false,
		},
		{
			"wrong format in optional field",
			map[string]interface{}{
				"testInt": "hello",
			},
			s.Schema{
				"test_int": Int("testInt", s.Optional),
			},
			common.MapStr{},
			true,
		},
	}

	for _, c := range cases {
		event, err := c.Schema.Apply(c.Input)
		if c.ExpectError {
			assert.Error(t, err, c.Description)
		} else {
			assert.NoError(t, err, c.Description)
			assert.Equal(t, c.Expected, event, c.Description)
		}
	}
}

func TestFullFieldPathInErrors(t *testing.T) {
	cases := []struct {
		Description string
		Schema      s.Schema
		Input       map[string]interface{}
		Expected    string
	}{
		{
			"missing nested key",
			s.Schema{
				"a": Dict("A", s.Schema{
					"b": Dict("B", s.Schema{
						"c": Bool("C"),
					}),
				}),
			},
			map[string]interface{}{
				"A": map[string]interface{}{
					"B": map[string]interface{}{},
				},
			},
			`A.B.C`,
		},
		{
			"wrong nested format key",
			s.Schema{
				"test_dict": Dict("testDict", s.Schema{
					"test_bool": Bool("testBool"),
				}),
			},
			map[string]interface{}{
				"testDict": map[string]interface{}{
					"testBool": "foo",
				},
			},
			`testDict.testBool`,
		},
		{
			"wrong nested sub-dictionary",
			s.Schema{
				"test_dict": Dict("testDict", s.Schema{
					"test_dict": Dict("testDict", s.Schema{}),
				}),
			},
			map[string]interface{}{
				"testDict": map[string]interface{}{
					"testDict": "foo",
				},
			},
			`testDict.testDict`,
		},
		{
			"empty input",
			s.Schema{
				"test_dict": Dict("rootDict", s.Schema{
					"test_dict": Dict("testDict", s.Schema{}),
				}),
			},
			map[string]interface{}{},
			`rootDict`,
		},
	}

	for _, c := range cases {
		_, err := c.Schema.Apply(c.Input)
		if assert.Error(t, err, c.Description) {
			assert.Contains(t, err.Error(), c.Expected, c.Description)
		}

		_, errs := c.Schema.ApplyTo(common.MapStr{}, c.Input)
		assert.Error(t, errs.Err(), c.Description)
		if assert.Equal(t, 1, len(errs), c.Description) {
			keyErr, ok := errs[0].(s.KeyError)
			if assert.True(t, ok, c.Description) {
				assert.Equal(t, c.Expected, keyErr.Key(), c.Description)
			}
		}
	}
}

func TestNestedFieldPaths(t *testing.T) {
	cases := []struct {
		Description string
		Input       map[string]interface{}
		Schema      s.Schema
		Expected    common.MapStr
		ExpectError bool
	}{
		{
			"nested values",
			map[string]interface{}{
				"root": map[string]interface{}{
					"foo":   "bar",
					"float": 4.5,
					"int":   4,
					"bool":  true,
				},
			},
			s.Schema{
				"foo":   Str("root.foo"),
				"float": Float("root.float"),
				"int":   Int("root.int"),
				"bool":  Bool("root.bool"),
			},
			common.MapStr{
				"foo":   "bar",
				"float": float64(4.5),
				"int":   int64(4),
				"bool":  true,
			},
			false,
		},
		{
			"not really nested values, path contains dots",
			map[string]interface{}{
				"root.foo": "bar",
			},
			s.Schema{
				"foo": Str("root.foo"),
			},
			common.MapStr{
				"foo": "bar",
			},
			false,
		},
		{
			"nested dict",
			map[string]interface{}{
				"root": map[string]interface{}{
					"dict": map[string]interface{}{
						"foo": "bar",
					},
				},
			},
			s.Schema{
				"dict": Dict("root.dict", s.Schema{
					"foo": Str("foo"),
				}),
			},
			common.MapStr{
				"dict": common.MapStr{
					"foo": "bar",
				},
			},
			false,
		},
	}

	for _, c := range cases {
		event, err := c.Schema.Apply(c.Input)
		if c.ExpectError {
			assert.Error(t, err, c.Description)
		} else {
			assert.NoError(t, err, c.Description)
			assert.Equal(t, c.Expected, event, c.Description)
		}
	}
}
