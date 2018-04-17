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

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/schema"
	"github.com/elastic/beats/libbeat/logp"
)

type ConvMap struct {
	Key      string        // The key in the data map
	Schema   schema.Schema // The schema describing how to convert the sub-map
	Optional bool
}

// Map drills down in the data dictionary by using the key
func (convMap ConvMap) Map(key string, event common.MapStr, data map[string]interface{}) *schema.Errors {
	subData, ok := data[convMap.Key].(map[string]interface{})
	if !ok {
		err := schema.NewError(convMap.Key, "Error accessing sub-dictionary")
		if convMap.Optional {
			err.SetType(schema.OptionalType)
		} else {
			logp.Err("Error accessing sub-dictionary `%s`", convMap.Key)
		}

		errors := schema.NewErrors()
		errors.AddError(err)

		return errors
	}

	subEvent := common.MapStr{}
	convMap.Schema.ApplyTo(subEvent, subData)
	event[key] = subEvent
	return nil
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
	emptyIface, exists := data[key]
	if !exists {
		return false, fmt.Errorf("Key %s not found", key)
	}
	switch emptyIface.(type) {
	case int, int32, int64, uint, uint32, uint64, float32, float64:
		return fmt.Sprintf("%v", emptyIface), nil
	case json.Number:
		return string(emptyIface.(json.Number)), nil
	default:
		return "", fmt.Errorf("Expected number, found %T", emptyIface)
	}
}

// StrFromNum creates a schema.Conv object that transforms numbers to strings.
func StrFromNum(key string, opts ...schema.SchemaOption) schema.Conv {
	return schema.SetOptions(schema.Conv{Key: key, Func: toStrFromNum}, opts)
}

func toStr(key string, data map[string]interface{}) (interface{}, error) {
	emptyIface, err := common.MapStr(data).GetValue(key)
	if err != nil {
		return "", fmt.Errorf("Key %s not found: %s", key, err.Error())
	}
	str, ok := emptyIface.(string)
	if !ok {
		return "", fmt.Errorf("Expected string, found %T", emptyIface)
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
		return "", fmt.Errorf("Key %s not found: %s", key, err.Error())
	}
	return intf, nil
}

// Ifc creates a schema.Conv object for converting the given data to interface.
func Ifc(key string, opts ...schema.SchemaOption) schema.Conv {
	return schema.SetOptions(schema.Conv{Key: key, Func: toIfc}, opts)
}

func toBool(key string, data map[string]interface{}) (interface{}, error) {
	emptyIface, exists := data[key]
	if !exists {
		return false, fmt.Errorf("Key %s not found", key)
	}
	boolean, ok := emptyIface.(bool)
	if !ok {
		return false, fmt.Errorf("Expected bool, found %T", emptyIface)
	}
	return boolean, nil
}

// Bool creates a Conv object for converting booleans.
func Bool(key string, opts ...schema.SchemaOption) schema.Conv {
	return schema.SetOptions(schema.Conv{Key: key, Func: toBool}, opts)
}

func toInteger(key string, data map[string]interface{}) (interface{}, error) {
	emptyIface, exists := data[key]
	if !exists {
		return 0, fmt.Errorf("Key %s not found", key)
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
		return 0, fmt.Errorf("expected integer, found json.Number (%v) that cannot be converted", num)
	default:
		return 0, fmt.Errorf("expected integer, found %T", emptyIface)
	}
}

// Float creates a Conv object for converting floats. Acceptable input
// types are int64, int, and float64.
func Float(key string, opts ...schema.SchemaOption) schema.Conv {
	return schema.SetOptions(schema.Conv{Key: key, Func: toFloat}, opts)
}

func toFloat(key string, data map[string]interface{}) (interface{}, error) {
	emptyIface, exists := data[key]
	if !exists {
		return 0, fmt.Errorf("key %s not found", key)
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
		return 0, fmt.Errorf("expected float, found json.Number (%v) that cannot be converted", num)
	default:
		return 0, fmt.Errorf("expected float, found %T", emptyIface)
	}
}

// Int creates a Conv object for converting integers. Acceptable input
// types are int64, int, and float64.
func Int(key string, opts ...schema.SchemaOption) schema.Conv {
	return schema.SetOptions(schema.Conv{Key: key, Func: toInteger}, opts)
}

func toTime(key string, data map[string]interface{}) (interface{}, error) {
	emptyIface, exists := data[key]
	if !exists {
		return common.Time(time.Unix(0, 0)), fmt.Errorf("Key %s not found", key)
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

	return common.Time(time.Unix(0, 0)), fmt.Errorf("Expected date, found %T", emptyIface)
}

// Time creates a Conv object for converting Time objects.
func Time(key string, opts ...schema.SchemaOption) schema.Conv {
	return schema.SetOptions(schema.Conv{Key: key, Func: toTime}, opts)
}

// SchemaOption is for adding optional parameters to the conversion
// functions
type DictSchemaOption func(c ConvMap) ConvMap

// The optional flag suppresses the error message in case the key
// doesn't exist or results in an error.
func DictOptional(c ConvMap) ConvMap {
	c.Optional = true
	return c
}

// setOptions adds the optional flags to the Conv object
func dictSetOptions(c ConvMap, opts []DictSchemaOption) ConvMap {
	for _, opt := range opts {
		c = opt(c)
	}
	return c
}
