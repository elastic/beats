package converter

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/schema"
)

type Schema map[string]Conv

// A Conv object represents a conversion mechanism from the data map to the event map.
type Conv struct {
	Func     Converter // Convertor function
	Key      string    // The key in the data map
	Optional bool      // Whether to ignore errors if the key is not found
	Required bool      // Whether to provoke errors if the key is not found
}

// Converter function type
type Converter func(key string, data map[string]interface{}) (interface{}, error)

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

// StrFromNum creates a Conv object that transforms numbers to strings.
func StrFromNum(key string, opts ...SchemaOption) Conv {
	return SetOptions(Conv{Key: key, Func: toStrFromNum}, opts)
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

// Str creates a Conv object for converting strings.
func Str(key string, opts ...SchemaOption) Conv {
	return SetOptions(Conv{Key: key, Func: toStr}, opts)
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

// Ifc creates a Conv object for converting the given data to interface.
func Ifc(key string, opts ...SchemaOption) Conv {
	return SetOptions(Conv{Key: key, Func: toIfc}, opts)
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
func Bool(key string, opts ...SchemaOption) Conv {
	return SetOptions(Conv{Key: key, Func: toBool}, opts)
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
func Float(key string, opts ...SchemaOption) Conv {
	return SetOptions(Conv{Key: key, Func: toFloat}, opts)
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
func Int(key string, opts ...SchemaOption) Conv {
	return SetOptions(Conv{Key: key, Func: toInteger}, opts)
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
func Time(key string, opts ...SchemaOption) Conv {
	return SetOptions(Conv{Key: key, Func: toTime}, opts)
}

type SchemaOption func(c Conv) Conv

// setOptions adds the optional flags to the Conv object
func SetOptions(c Conv, opts []SchemaOption) Conv {
	for _, opt := range opts {
		c = opt(c)
	}
	return c
}
