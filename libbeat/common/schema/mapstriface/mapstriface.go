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
Package mapstriface contains utilities for transforming map[string]interface{} objects
into metricbeat events. For example, given this input object:

	input := map[string]interface{}{
		"testString":       "hello",
		"testInt":          42,
		"testIntFromFloat": 42.0,
		"testIntFromInt64": int64(42),
		"testBool":         true,
		"testObj": map[string]interface{}{
			"testObjString": "hello, object",
		},
		"testNonNestedObj": "hello from top level",
	}

And the requirement to transform it into this one:

	common.MapStr{
		"test_string":         "hello",
		"test_int":            int64(42),
		"test_int_from_float": int64(42),
		"test_int_from_int64": int64(42),
		"test_bool":           true,
		"test_time":           common.Time(ts),
		"test_obj_1": common.MapStr{
			"test": "hello from top level",
		},
		"test_obj_2": common.MapStr{
			"test": "hello, object",
		},
	}

It can be done with the following code:

	schema := s.Schema{
		"test_string":         Str("testString"),
		"test_int":            Int("testInt"),
		"test_int_from_float": Int("testIntFromFloat"),
		"test_int_from_int64": Int("testIntFromInt64"),
		"test_bool":           Bool("testBool"),
		"test_time":           Time("testTime"),
		"test_obj_1": s.Object{
			"test": Str("testNonNestedObj"),
		},
		"test_obj_2": Dict("testObj", s.Schema{
			"test": Str("testObjString"),
		}),
	}
	output := schema.Apply(input)

Note that this allows for converting, renaming, and restructuring the data.
*/
package mapstriface

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/joeshaw/multierror"

	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/common/schema"
	"github.com/menderesk/beats/v7/libbeat/logp"
)

type ConvMap struct {
	Key      string        // The key in the data map
	Schema   schema.Schema // The schema describing how to convert the sub-map
	Optional bool
	Required bool
}

// Map drills down in the data dictionary by using the key
func (convMap ConvMap) Map(key string, event common.MapStr, data map[string]interface{}) multierror.Errors {
	d, err := common.MapStr(data).GetValue(convMap.Key)
	if err != nil {
		err := schema.NewKeyNotFoundError(convMap.Key)
		err.Optional = convMap.Optional
		err.Required = convMap.Required
		return multierror.Errors{err}
	}
	switch subData := d.(type) {
	case map[string]interface{}, common.MapStr:
		subEvent := common.MapStr{}
		_, errors := convMap.Schema.ApplyTo(subEvent, subData.(map[string]interface{}))
		for _, err := range errors {
			if err, ok := err.(schema.KeyError); ok {
				err.SetKey(convMap.Key + "." + err.Key())
			}
		}
		event[key] = subEvent
		return errors
	default:
		msg := fmt.Sprintf("expected dictionary, found %T", subData)
		err := schema.NewWrongFormatError(convMap.Key, msg)
		logp.Err(err.Error())
		return multierror.Errors{err}
	}
}

func (convMap ConvMap) HasKey(key string) bool {
	if convMap.Key == key {
		return true
	}

	return convMap.Schema.HasKey(key)
}

func Dict(key string, s schema.Schema, opts ...DictSchemaOption) ConvMap {
	return dictSetOptions(ConvMap{Key: key, Schema: s}, opts)
}

func toStrFromNum(key string, data map[string]interface{}) (interface{}, error) {
	emptyIface, err := common.MapStr(data).GetValue(key)
	if err != nil {
		return "", schema.NewKeyNotFoundError(key)
	}
	switch emptyIface.(type) {
	case int, int32, int64, uint, uint32, uint64, float32, float64:
		return fmt.Sprintf("%v", emptyIface), nil
	case json.Number:
		return string(emptyIface.(json.Number)), nil
	default:
		msg := fmt.Sprintf("expected number, found %T", emptyIface)
		return "", schema.NewWrongFormatError(key, msg)
	}
}

// StrFromNum creates a schema.Conv object that transforms numbers to strings.
func StrFromNum(key string, opts ...schema.SchemaOption) schema.Conv {
	return schema.SetOptions(schema.Conv{Key: key, Func: toStrFromNum}, opts)
}

func toStr(key string, data map[string]interface{}) (interface{}, error) {
	emptyIface, err := common.MapStr(data).GetValue(key)
	if err != nil {
		return "", schema.NewKeyNotFoundError(key)
	}
	str, ok := emptyIface.(string)
	if !ok {
		msg := fmt.Sprintf("expected string, found %T", emptyIface)
		return "", schema.NewWrongFormatError(key, msg)
	}
	return str, nil
}

// Str creates a schema.Conv object for converting strings.
func Str(key string, opts ...schema.SchemaOption) schema.Conv {
	return schema.SetOptions(schema.Conv{Key: key, Func: toStr}, opts)
}

func toIfc(key string, data map[string]interface{}) (interface{}, error) {
	intf, err := common.MapStr(data).GetValue(key)
	if err != nil {
		e := schema.NewKeyNotFoundError(key)
		e.Err = err
		return nil, e
	}
	return intf, nil
}

// Ifc creates a schema.Conv object for converting the given data to interface.
func Ifc(key string, opts ...schema.SchemaOption) schema.Conv {
	return schema.SetOptions(schema.Conv{Key: key, Func: toIfc}, opts)
}

func toBool(key string, data map[string]interface{}) (interface{}, error) {
	emptyIface, err := common.MapStr(data).GetValue(key)
	if err != nil {
		return false, schema.NewKeyNotFoundError(key)
	}
	boolean, ok := emptyIface.(bool)
	if !ok {
		msg := fmt.Sprintf("expected bool, found %T", emptyIface)
		return false, schema.NewWrongFormatError(key, msg)
	}
	return boolean, nil
}

// Bool creates a Conv object for converting booleans.
func Bool(key string, opts ...schema.SchemaOption) schema.Conv {
	return schema.SetOptions(schema.Conv{Key: key, Func: toBool}, opts)
}

func toInteger(key string, data map[string]interface{}) (interface{}, error) {
	emptyIface, err := common.MapStr(data).GetValue(key)
	if err != nil {
		return 0, schema.NewKeyNotFoundError(key)
	}
	switch emptyIface.(type) {
	case int64:
		return emptyIface.(int64), nil
	case int:
		return int64(emptyIface.(int)), nil
	case float64:
		return int64(emptyIface.(float64)), nil
	case json.Number:
		num := emptyIface.(json.Number)
		i64, err := num.Int64()
		if err == nil {
			return i64, nil
		}
		f64, err := num.Float64()
		if err == nil {
			return int64(f64), nil
		}
		msg := fmt.Sprintf("expected integer, found json.Number (%v) that cannot be converted", num)
		return 0, schema.NewWrongFormatError(key, msg)
	default:
		msg := fmt.Sprintf("expected integer, found %T", emptyIface)
		return 0, schema.NewWrongFormatError(key, msg)
	}
}

// Float creates a Conv object for converting floats. Acceptable input
// types are int64, int, and float64.
func Float(key string, opts ...schema.SchemaOption) schema.Conv {
	return schema.SetOptions(schema.Conv{Key: key, Func: toFloat}, opts)
}

func toFloat(key string, data map[string]interface{}) (interface{}, error) {
	emptyIface, err := common.MapStr(data).GetValue(key)
	if err != nil {
		return 0.0, schema.NewKeyNotFoundError(key)
	}
	switch emptyIface.(type) {
	case float64:
		return emptyIface.(float64), nil
	case int:
		return float64(emptyIface.(int)), nil
	case int64:
		return float64(emptyIface.(int64)), nil
	case json.Number:
		num := emptyIface.(json.Number)
		i64, err := num.Float64()
		if err == nil {
			return i64, nil
		}
		f64, err := num.Float64()
		if err == nil {
			return f64, nil
		}
		msg := fmt.Sprintf("expected float, found json.Number (%v) that cannot be converted", num)
		return 0.0, schema.NewWrongFormatError(key, msg)
	default:
		msg := fmt.Sprintf("expected float, found %T", emptyIface)
		return 0.0, schema.NewWrongFormatError(key, msg)
	}
}

// Int creates a Conv object for converting integers. Acceptable input
// types are int64, int, and float64.
func Int(key string, opts ...schema.SchemaOption) schema.Conv {
	return schema.SetOptions(schema.Conv{Key: key, Func: toInteger}, opts)
}

func toTime(key string, data map[string]interface{}) (interface{}, error) {
	emptyIface, err := common.MapStr(data).GetValue(key)
	if err != nil {
		return common.Time(time.Unix(0, 0)), schema.NewKeyNotFoundError(key)
	}

	switch emptyIface.(type) {
	case time.Time:
		ts, ok := emptyIface.(time.Time)
		if ok {
			return common.Time(ts), nil
		}
	case common.Time:
		ts, ok := emptyIface.(common.Time)
		if ok {
			return ts, nil
		}
	}

	msg := fmt.Sprintf("expected date, found %T", emptyIface)
	return common.Time(time.Unix(0, 0)), schema.NewWrongFormatError(key, msg)
}

// Time creates a Conv object for converting Time objects.
func Time(key string, opts ...schema.SchemaOption) schema.Conv {
	return schema.SetOptions(schema.Conv{Key: key, Func: toTime}, opts)
}

// SchemaOption is for adding optional parameters to the conversion
// functions
type DictSchemaOption func(c ConvMap) ConvMap

// DictOptional sets the optional flag, which suppresses the error in
// case the key doesn't exist or results in an error.
func DictOptional(c ConvMap) ConvMap {
	c.Optional = true
	return c
}

// DictRequired sets the required flag, which forces an error even if fields
// are optional by default
func DictRequired(c ConvMap) ConvMap {
	c.Required = true
	return c
}

// setOptions adds the optional flags to the Conv object
func dictSetOptions(c ConvMap, opts []DictSchemaOption) ConvMap {
	for _, opt := range opts {
		c = opt(c)
	}
	return c
}
