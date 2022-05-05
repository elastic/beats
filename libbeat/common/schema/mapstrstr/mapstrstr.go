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

/*
Package mapstrstr contains utilities for transforming map[string]string objects
into metricbeat events. For example, given this input object:

	input := map[string]interface{}{
		"testString":     "hello",
		"testInt":        "42",
		"testBool":       "true",
		"testFloat":      "42.1",
		"testObjString":  "hello, object",
	}

And the requirement to transform it into this one:

	mapstr.M{
		"test_string": "hello",
		"test_int":    int64(42),
		"test_bool":   true,
		"test_float":  42.1,
		"test_obj": mapstr.M{
			"test_obj_string": "hello, object",
		},
	}

It can be done with the following code:

	schema := s.Schema{
		"test_string": Str("testString"),
		"test_int":    Int("testInt"),
		"test_bool":   Bool("testBool"),
		"test_float":  Float("testFloat"),
		"test_obj": s.Object{
			"test_obj_string": Str("testObjString"),
		},
	}
	schema.Apply(input)

Note that this allows parsing, renaming of fields and restructuring the result
object.
*/
package mapstrstr

import (
	"fmt"
	"strconv"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/schema"
)

// toBool converts value to bool. In case of error, returns false
func toBool(key string, data map[string]interface{}) (interface{}, error) {
	str, err := getString(key, data)
	if err != nil {
		return false, err
	}

	value, err := strconv.ParseBool(str)
	if err != nil {
		msg := fmt.Sprintf("error converting param to bool: `%s`", str)
		return false, schema.NewWrongFormatError(key, msg)
	}

	return value, nil
}

// Bool creates a Conv object for parsing booleans
func Bool(key string, opts ...schema.SchemaOption) schema.Conv {
	return schema.SetOptions(schema.Conv{Key: key, Func: toBool}, opts)
}

// toFloat converts value to float64. In case of error, returns 0.0
func toFloat(key string, data map[string]interface{}) (interface{}, error) {
	str, err := getString(key, data)
	if err != nil {
		return false, err
	}

	value, err := strconv.ParseFloat(str, 64)
	if err != nil {
		msg := fmt.Sprintf("error converting param to float: `%s`", str)
		return 0.0, schema.NewWrongFormatError(key, msg)
	}

	return value, nil
}

// Float creates a Conv object for parsing floats
func Float(key string, opts ...schema.SchemaOption) schema.Conv {
	return schema.SetOptions(schema.Conv{Key: key, Func: toFloat}, opts)
}

// toInt converts value to int. In case of error, returns 0
func toInt(key string, data map[string]interface{}) (interface{}, error) {
	str, err := getString(key, data)
	if err != nil {
		return false, err
	}

	value, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		msg := fmt.Sprintf("error converting param to int: `%s`", str)
		return 0, schema.NewWrongFormatError(key, msg)
	}

	return value, nil
}

// Int creates a Conv object for parsing integers
func Int(key string, opts ...schema.SchemaOption) schema.Conv {
	return schema.SetOptions(schema.Conv{Key: key, Func: toInt}, opts)
}

// toStr converts value to str. In case of error, returns ""
func toStr(key string, data map[string]interface{}) (interface{}, error) {
	return getString(key, data)
}

// Time creates a schema.Conv object for parsing timestamps. Unlike the
// other functions, Time receives a `layout` parameter which defines the
// time.Time layout to use for parsing.
func Time(layout, key string, opts ...schema.SchemaOption) schema.Conv {
	return schema.SetOptions(schema.Conv{
		Key: key,
		Func: func(key string, data map[string]interface{}) (interface{}, error) {
			str, err := getString(key, data)
			if err != nil {
				return false, err
			}

			value, err := time.Parse(layout, str)
			if err != nil {
				msg := fmt.Sprintf("error converting param to time.Time: `%s`", str)
				return common.Time{}, schema.NewWrongFormatError(key, msg)
			}

			return common.Time(value), nil
		},
	}, opts)
}

// Str creates a schema.Conv object for parsing strings
func Str(key string, opts ...schema.SchemaOption) schema.Conv {
	return schema.SetOptions(schema.Conv{Key: key, Func: toStr}, opts)
}

// checkExists checks if a key exists in the given data set
func getString(key string, data map[string]interface{}) (string, error) {
	val, exists := data[key]
	if !exists {
		return "", schema.NewKeyNotFoundError(key)
	}

	str, ok := val.(string)
	if !ok {
		return "", schema.NewWrongFormatError(key, fmt.Sprintf("expected type string but has %T", val))
	}

	return str, nil
}
