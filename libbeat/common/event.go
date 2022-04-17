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

package common

import (
	"encoding"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/menderesk/beats/v7/libbeat/logp"
)

// EventConverter is used to convert MapStr objects for publishing
type EventConverter interface {
	Convert(m MapStr) MapStr
}

// GenericEventConverter is used to normalize MapStr objects for publishing
type GenericEventConverter struct {
	log      *logp.Logger
	keepNull bool
}

// NewGenericEventConverter creates an EventConverter with the given configuration options
func NewGenericEventConverter(keepNull bool) *GenericEventConverter {
	return &GenericEventConverter{
		log:      logp.NewLogger("event"),
		keepNull: keepNull,
	}
}

// Convert normalizes the types contained in the given MapStr.
//
// Nil values in maps are dropped during the conversion. Any unsupported types
// that are found in the MapStr are dropped and warnings are logged.
func (e *GenericEventConverter) Convert(m MapStr) MapStr {
	keys := make([]string, 0, 10)
	event, errs := e.normalizeMap(m, keys...)
	if len(errs) > 0 {
		e.log.Warnf("Unsuccessful conversion to generic event: %v errors: %v, "+
			"event=%#v", len(errs), errs, m)
	}
	return event
}

// normalizeMap normalizes each element contained in the given map. If an error
// occurs during normalization, processing of m will continue, and all errors
// are returned at the end.
func (e *GenericEventConverter) normalizeMap(m MapStr, keys ...string) (MapStr, []error) {
	var errs []error

	out := make(MapStr, len(m))
	for key, value := range m {
		v, err := e.normalizeValue(value, append(keys, key)...)
		if len(err) > 0 {
			errs = append(errs, err...)
		}

		// Drop nil values from maps.
		if !e.keepNull && v == nil {
			if e.log.IsDebug() {
				e.log.Debugf("Dropped nil value from event where key=%v", joinKeys(append(keys, key)...))
			}
			continue
		}

		out[key] = v
	}

	return out, errs
}

// normalizeMapStrSlice normalizes each individual MapStr.
func (e *GenericEventConverter) normalizeMapStrSlice(maps []MapStr, keys ...string) ([]MapStr, []error) {
	var errs []error

	out := make([]MapStr, 0, len(maps))
	for i, m := range maps {
		normalizedMap, err := e.normalizeMap(m, append(keys, strconv.Itoa(i))...)
		if len(err) > 0 {
			errs = append(errs, err...)
		}
		out = append(out, normalizedMap)
	}

	return out, errs
}

// normalizeMapStringSlice normalizes each individual map[string]interface{} and
// returns a []MapStr.
func (e *GenericEventConverter) normalizeMapStringSlice(maps []map[string]interface{}, keys ...string) ([]MapStr, []error) {
	var errs []error

	out := make([]MapStr, 0, len(maps))
	for i, m := range maps {
		normalizedMap, err := e.normalizeMap(m, append(keys, strconv.Itoa(i))...)
		if len(err) > 0 {
			errs = append(errs, err...)
		}
		out = append(out, normalizedMap)
	}

	return out, errs
}

// normalizeSlice normalizes each element of the slice and returns a []interface{}.
func (e *GenericEventConverter) normalizeSlice(v reflect.Value, keys ...string) (interface{}, []error) {
	var errs []error
	var sliceValues []interface{}

	n := v.Len()
	for i := 0; i < n; i++ {
		sliceValue, err := e.normalizeValue(v.Index(i).Interface(), append(keys, strconv.Itoa(i))...)
		if len(err) > 0 {
			errs = append(errs, err...)
		}

		sliceValues = append(sliceValues, sliceValue)
	}

	return sliceValues, errs
}

func (e *GenericEventConverter) normalizeValue(value interface{}, keys ...string) (interface{}, []error) {
	if value == nil {
		return nil, nil
	}

	// Normalize time values to a common.Time with UTC time zone.
	switch v := value.(type) {
	case time.Time:
		value = Time(v.UTC())
	case []time.Time:
		times := make([]Time, 0, len(v))
		for _, t := range v {
			times = append(times, Time(t.UTC()))
		}
		value = times
	case Time:
		value = Time(time.Time(v).UTC())
	case []Time:
		times := make([]Time, 0, len(v))
		for _, t := range v {
			times = append(times, Time(time.Time(t).UTC()))
		}
		value = times
	}

	switch value.(type) {
	case encoding.TextMarshaler:
		if reflect.ValueOf(value).Kind() == reflect.Ptr && reflect.ValueOf(value).IsNil() {
			return nil, nil
		}
		text, err := value.(encoding.TextMarshaler).MarshalText()
		if err != nil {
			return nil, []error{errors.Wrapf(err, "key=%v: error converting %T to string", joinKeys(keys...), value)}
		}
		return string(text), nil
	case string, []string:
	case bool, []bool:
	case int, int8, int16, int32, int64:
	case []int, []int8, []int16, []int32, []int64:
	case uint, uint8, uint16, uint32:
	case uint64:
		return value.(uint64) &^ (1 << 63), nil
	case []uint, []uint8, []uint16, []uint32:
	case []uint64:
		arr := value.([]uint64)
		mask := false
		for _, v := range arr {
			if v >= (1 << 63) {
				mask = true
				break
			}
		}
		if !mask {
			return value, nil
		}

		tmp := make([]uint64, len(arr))
		for i, v := range arr {
			tmp[i] = v &^ (1 << 63)
		}
		return tmp, nil

	case float32, float64:
	case []float32, []float64:
	case complex64, complex128:
	case []complex64, []complex128:
	case Time, []Time:
	case MapStr:
		return e.normalizeMap(value.(MapStr), keys...)
	case []MapStr:
		return e.normalizeMapStrSlice(value.([]MapStr), keys...)
	case map[string]interface{}:
		return e.normalizeMap(value.(map[string]interface{}), keys...)
	case []map[string]interface{}:
		return e.normalizeMapStringSlice(value.([]map[string]interface{}), keys...)
	default:
		v := reflect.ValueOf(value)

		switch v.Type().Kind() {
		case reflect.Ptr:
			// Dereference pointers.
			return e.normalizeValue(followPointer(value), keys...)
		case reflect.Bool:
			return v.Bool(), nil
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return v.Int(), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return v.Uint() &^ (1 << 63), nil
		case reflect.Float32, reflect.Float64:
			return v.Float(), nil
		case reflect.Complex64, reflect.Complex128:
			return v.Complex(), nil
		case reflect.String:
			return v.String(), nil
		case reflect.Array, reflect.Slice:
			return e.normalizeSlice(v, keys...)
		case reflect.Map, reflect.Struct:
			var m MapStr
			err := marshalUnmarshal(value, &m)
			if err != nil {
				return m, []error{errors.Wrapf(err, "key=%v: error converting %T to MapStr", joinKeys(keys...), value)}
			}
			return m, nil
		default:
			// Drop Uintptr, UnsafePointer, Chan, Func, Interface, and any other
			// types not specifically handled above.
			return nil, []error{fmt.Errorf("key=%v: error unsupported type=%T value=%#v", joinKeys(keys...), value, value)}
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

// DeDot a string by replacing all . with _
// This helps when sending data to Elasticsearch to prevent object and key collisions.
func DeDot(s string) string {
	return strings.Replace(s, ".", "_", -1)
}

// DeDotJSON replaces in keys all . with _
// This helps when sending data to Elasticsearch to prevent object and key collisions.
func DeDotJSON(json interface{}) interface{} {
	switch json := json.(type) {
	case map[string]interface{}:
		result := map[string]interface{}{}
		for key, value := range json {
			result[DeDot(key)] = DeDotJSON(value)
		}
		return result
	case MapStr:
		result := MapStr{}
		for key, value := range json {
			result[DeDot(key)] = DeDotJSON(value)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(json))
		for i, value := range json {
			result[i] = DeDotJSON(value)
		}
		return result
	default:
		return json
	}
}
