package common

import (
	"encoding"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/elastic/beats/libbeat/logp"

	"github.com/pkg/errors"
)

const eventDebugSelector = "event"

var eventDebugf = logp.MakeDebug(eventDebugSelector)

var textMarshalerType = reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem()

// ConvertToGenericEvent normalizes the types contained in the given MapStr.
//
// Nil values in maps are dropped during the conversion. Any unsupported types
// that are found in the MapStr are dropped and warnings are logged.
func ConvertToGenericEvent(m MapStr) MapStr {
	event, errs := normalizeMap("", m)
	if len(errs) > 0 {
		logp.Warn("Unsuccessful conversion to generic event: %v errors: %v, "+
			"event=%#v", len(errs), errs, m)
	}
	return event
}

// normalizeMap normalizes each element contained in the given map. If an error
// occurs during normalization, processing of m will continue, and all errors
// are returned at the end.
func normalizeMap(baseKey string, m MapStr) (MapStr, []error) {
	var errs []error

	out := make(MapStr, len(m))
	for key, value := range m {
		fullKey := joinKeys(baseKey, key)
		v, err := normalizeValue(fullKey, value)
		if len(err) > 0 {
			errs = append(errs, err...)
		}

		// Drop nil values from maps.
		if v == nil {
			if logp.IsDebug(eventDebugSelector) {
				eventDebugf("Dropped nil value from event where key=%v", fullKey)
			}
			continue
		}

		out[key] = v
	}

	return out, errs
}

// normalizeMapStrSlice normalizes each individual MapStr.
func normalizeMapStrSlice(baseKey string, maps []MapStr) ([]MapStr, []error) {
	var errs []error

	out := make([]MapStr, 0, len(maps))
	for i, m := range maps {
		normalizedMap, err := normalizeMap(joinKeys(baseKey, strconv.Itoa(i)), m)
		if len(err) > 0 {
			errs = append(errs, err...)
		}
		out = append(out, normalizedMap)
	}

	return out, errs
}

// normalizeMapStringSlice normalizes each individual map[string]interface{} and
// returns a []MapStr.
func normalizeMapStringSlice(baseKey string, maps []map[string]interface{}) ([]MapStr, []error) {
	var errs []error

	out := make([]MapStr, 0, len(maps))
	for i, m := range maps {
		normalizedMap, err := normalizeMap(joinKeys(baseKey, strconv.Itoa(i)), m)
		if len(err) > 0 {
			errs = append(errs, err...)
		}
		out = append(out, normalizedMap)
	}

	return out, errs
}

// normalizeSlice normalizes each element of the slice and returns a []interface{}.
func normalizeSlice(baseKey string, v reflect.Value) (interface{}, []error) {
	var errs []error
	var sliceValues []interface{}

	n := v.Len()
	for i := 0; i < n; i++ {
		sliceValue, err := normalizeValue(joinKeys(baseKey, strconv.Itoa(i)), v.Index(i).Interface())
		if len(err) > 0 {
			errs = append(errs, err...)
		}

		sliceValues = append(sliceValues, sliceValue)
	}

	return sliceValues, errs
}

func normalizeValue(key string, value interface{}) (interface{}, []error) {
	// Dereference pointers.
	value = followPointer(value)

	if value == nil {
		return nil, nil
	}

	switch value.(type) {
	case encoding.TextMarshaler:
		text, err := value.(encoding.TextMarshaler).MarshalText()
		if err != nil {
			return nil, []error{errors.Wrapf(err, "key=%v: error converting %T to string", key, value)}
		}
		return string(text), nil
	case bool, []bool:
	case int, int8, int16, int32, int64:
	case []int, []int8, []int16, []int32, []int64:
	case uint, uint8, uint16, uint32, uint64:
	case []uint, []uint8, []uint16, []uint32, []uint64:
	case float32, float64:
	case []float32, []float64:
	case complex64, complex128:
	case []complex64, []complex128:
	case string, []string:
	case Time, []Time:
	case MapStr:
		return normalizeMap(key, value.(MapStr))
	case []MapStr:
		return normalizeMapStrSlice(key, value.([]MapStr))
	case map[string]interface{}:
		return normalizeMap(key, value.(map[string]interface{}))
	case []map[string]interface{}:
		return normalizeMapStringSlice(key, value.([]map[string]interface{}))
	default:
		v := reflect.ValueOf(value)

		switch v.Type().Kind() {
		case reflect.Bool:
			return v.Bool(), nil
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return v.Int(), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return v.Uint(), nil
		case reflect.Float32, reflect.Float64:
			return v.Float(), nil
		case reflect.Complex64, reflect.Complex128:
			return v.Complex(), nil
		case reflect.String:
			return v.String(), nil
		case reflect.Array, reflect.Slice:
			return normalizeSlice(key, v)
		case reflect.Map, reflect.Struct:
			var m MapStr
			err := marshalUnmarshal(value, &m)
			if err != nil {
				return m, []error{errors.Wrapf(err, "key=%v: error converting %T to MapStr", key, value)}
			}
			return m, nil
		default:
			// Drop Uintptr, UnsafePointer, Chan, Func, Interface, and any other
			// types not specifically handled above.
			return nil, []error{fmt.Errorf("key=%v: error unsupported type=%T value=%#v", key, value, value)}
		}
	}

	return value, nil
}

// marshalUnmarshal converts an interface to a MapStr by marshalling to JSON
// then unmarshalling the JSON object into a MapStr.
func marshalUnmarshal(in interface{}, out interface{}) error {
	// Decode and encode as JSON to normalized the types.
	marshaled, err := json.Marshal(in)
	if err != nil {
		return errors.Wrap(err, "error marshalling to JSON")
	}
	err = json.Unmarshal(marshaled, out)
	if err != nil {
		return errors.Wrap(err, "error unmarshalling from JSON")
	}

	return nil
}

// followPointer accepts an interface{} and if the interface is a pointer then
// the value that v points to is returned. If v is not a pointer then v is
// returned.
func followPointer(v interface{}) interface{} {
	if v == nil || reflect.TypeOf(v).Kind() != reflect.Ptr {
		return v
	}

	val := reflect.ValueOf(v)
	if val.IsNil() {
		return nil
	}

	return val.Elem().Interface()
}

// joinKeys concatenates the keys into a single string with each key separated
// by a dot.
func joinKeys(keys ...string) string {
	// Strip leading empty string.
	if len(keys) > 0 && keys[0] == "" {
		keys = keys[1:]
	}
	return strings.Join(keys, ".")
}
