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

	common.MapStr{
		"test_string": "hello",
		"test_int":    int64(42),
		"test_bool":   true,
		"test_float":  42.1,
		"test_obj": common.MapStr{
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

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/schema"
)

// toBool converts value to bool. In case of error, returns false
func toBool(key string, data map[string]interface{}) (interface{}, error) {

	str, err := getString(key, data)
	if err != nil {
		return false, err
	}

	value, err := strconv.ParseBool(str)
	if err != nil {
		return false, fmt.Errorf("Error converting param to bool: %s", key)
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
		return 0.0, fmt.Errorf("Error converting param to float: %s", key)
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
		return 0, fmt.Errorf("Error converting param to int: %s", key)
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
				return 0, fmt.Errorf("Error converting param to time.Time: %s. Original: %s", key, str)
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
		return "", fmt.Errorf("Key `%s` not found", key)
	}

	str, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("Expected value of `%s` to have type string but has %T", key, val)
	}

	return str, nil
}
