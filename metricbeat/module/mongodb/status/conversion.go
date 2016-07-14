package status

import (
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// MSchema describes how a map[string]interface{} object can be parsed and converted into
// an event. The conversions can be described using an (optionally nested) common.MapStr
// that contains Conv objects.
type MSchema struct {
	conversions common.MapStr
}

// NewMSchema creates a new converting schema.
func NewMSchema(conversions common.MapStr) MSchema {
	return MSchema{conversions}
}

// An MConv object represents a conversion mechanism from the data map to the event map.
type MConv struct {
	Func MConvertor // Convertor function
	Key  string     // The key in the data map
}

func Map(key string, conversions common.MapStr) MConvMap {
	return MConvMap{Key: key, Conversions: conversions}
}

// An MConvMap object represents a conversion mechanism from sub-map in the data to
// the event map.
type MConvMap struct {
	Key         string        // The key in the data map
	Conversions common.MapStr // The schema describing how to convert the sub-map
}

type MConvertor func(key string, data map[string]interface{}) (interface{}, error)

func applyMSchemaToEvent(event common.MapStr, data map[string]interface{}, conversions common.MapStr) {
	for key, conversion := range conversions {
		switch conversion.(type) {
		case MConv:
			conv := conversion.(MConv)
			value, err := conv.Func(conv.Key, data)
			if err != nil {
				logp.Err("Error on field '%s': %v", key, err)
			} else {
				event[key] = value
			}
		case MConvMap:
			convMap := conversion.(MConvMap)
			subData, ok := data[convMap.Key].(map[string]interface{})
			if !ok {
				logp.Err("Error accessing sub-dictionary `%s`", convMap.Key)
				continue
			}

			subEvent := common.MapStr{}
			applyMSchemaToEvent(subEvent, subData, convMap.Conversions)
			event[key] = subEvent
		case common.MapStr:
			subEvent := common.MapStr{}
			applyMSchemaToEvent(subEvent, data, conversion.(common.MapStr))
			event[key] = subEvent
		default:
			logp.Err("Unexpected type for '%s' in schema: %T", key, conversion)
		}
	}
}

// ApplyTo adds the fields extracted from data, converted using the schema, to the
// event map.
func (s MSchema) ApplyTo(event common.MapStr, data map[string]interface{}) common.MapStr {
	applyMSchemaToEvent(event, data, s.conversions)
	return event
}

// Apply converts the fields extracted from data, using the schema, into a new map.
func (s MSchema) Apply(data map[string]interface{}) common.MapStr {
	return s.ApplyTo(common.MapStr{}, data)
}

func toString(key string, data map[string]interface{}) (interface{}, error) {
	emptyIface, exists := data[key]
	if !exists {
		return "", fmt.Errorf("Key not found")
	}
	str, ok := emptyIface.(string)
	if !ok {
		return "", fmt.Errorf("Expected string, found %T", emptyIface)
	}
	return str, nil
}

func String(key string) MConv {
	return MConv{Key: key, Func: toString}
}

func toBool(key string, data map[string]interface{}) (interface{}, error) {
	emptyIface, exists := data[key]
	if !exists {
		return false, fmt.Errorf("Key not found")
	}
	boolean, ok := emptyIface.(bool)
	if !ok {
		return false, fmt.Errorf("Expected bool, found %T", emptyIface)
	}
	return boolean, nil
}

func Bool(key string) MConv {
	return MConv{Key: key, Func: toBool}
}

func toInteger(key string, data map[string]interface{}) (interface{}, error) {
	emptyIface, exists := data[key]
	if !exists {
		return 0, fmt.Errorf("Key not found")
	}
	switch emptyIface.(type) {
	case int64:
		return emptyIface.(int64), nil
	case int:
		return int64(emptyIface.(int)), nil
	case float64:
		return int64(emptyIface.(float64)), nil
	default:
		return 0, fmt.Errorf("Expected integer, found %T", emptyIface)
	}
}

func Int(key string) MConv {
	return MConv{Key: key, Func: toInteger}
}

func toTime(key string, data map[string]interface{}) (interface{}, error) {
	emptyIface, exists := data[key]
	if !exists {
		return common.Time(time.Unix(0, 0)), fmt.Errorf("Key not found")
	}
	ts, ok := emptyIface.(time.Time)
	if !ok {
		return common.Time(time.Unix(0, 0)), fmt.Errorf("Expected date, found %T", emptyIface)
	}
	return common.Time(ts), nil
}

func Time(key string) MConv {
	return MConv{Key: key, Func: toTime}
}
